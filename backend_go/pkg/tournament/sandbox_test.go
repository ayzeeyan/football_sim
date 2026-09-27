package tournament

import (
	"encoding/json"
	"sort"
	"testing"
)

// sandboxSnapshot captures every observable a read-only path must not touch:
// an optional fixture, the full league table, and per-player season stats.
// An empty fixtureID skips the fixture section (for non-fixture paths).
func sandboxSnapshot(t *testing.T, tm *TournamentManager, fixtureID string) []byte {
	t.Helper()
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	snapshot := map[string]interface{}{}
	if fixtureID != "" {
		f := tm.findFixtureUnlocked(fixtureID)
		if f == nil {
			t.Fatalf("fixture %s vanished", fixtureID)
		}
		snapshot["fixture"] = f
	}
	if tm.World != nil {
		tables := map[string]interface{}{}
		for id, comp := range tm.World.Competitions {
			if comp == nil || comp.Kind != CompetitionLeague {
				continue
			}
			rows := []map[string]interface{}{}
			for _, c := range tm.worldLeagueStandingsUnlocked(id) {
				rows = append(rows, map[string]interface{}{
					"club_id": c.ClubID, "p": c.Played, "pts": c.Points,
					"gf": c.GoalsFor, "ga": c.GoalsAgainst, "gd": c.GoalDifference,
				})
			}
			tables[id] = rows
		}
		snapshot["tables"] = tables
	}
	players := []map[string]interface{}{}
	for _, c := range tm.Clubs {
		for _, p := range c.Squad {
			players = append(players, map[string]interface{}{
				"id": p.PlayerID, "apps": p.Appearances, "goals": p.Goals,
				"assists": p.Assists, "fitness": p.Fitness, "morale": p.Morale,
			})
		}
	}
	// tm.Clubs is a map: sort so the snapshot is order-independent.
	sort.Slice(players, func(i, j int) bool {
		return players[i]["id"].(string) < players[j]["id"].(string)
	})
	snapshot["players"] = players
	out, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}
	return out
}

func firstSlateFixtureID(t *testing.T, tm *TournamentManager, domesticOnly bool) (string, *Fixture) {
	t.Helper()
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	ids := make([]string, 0, len(tm.Fixtures))
	for _, f := range tm.Fixtures {
		if f.Status != "scheduled" {
			continue
		}
		if domesticOnly && !tm.isWorldDomesticLeague(f.Competition) {
			continue
		}
		ids = append(ids, f.FixtureID)
	}
	if len(ids) == 0 {
		t.Fatal("no scheduled fixture found")
	}
	// Deterministic pick: first in sorted order.
	for i := 1; i < len(ids); i++ {
		if ids[i] < ids[0] {
			ids[0], ids[i] = ids[i], ids[0]
		}
	}
	return ids[0], tm.findFixtureUnlocked(ids[0])
}

// TestWhatIfSandboxReadOnlyAndDeterministic pins the two sandbox contracts:
// the real universe is byte-identical before and after a what-if, and the
// same scratch seed always yields the same answer.
func TestWhatIfSandboxReadOnlyAndDeterministic(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	fid, f := firstSlateFixtureID(t, tm, true)

	before := sandboxSnapshot(t, tm, fid)
	r1, errMsg := tm.WhatIfSandbox(fid, 777)
	if r1 == nil {
		t.Fatalf("what-if failed: %s", errMsg)
	}
	r2, _ := tm.WhatIfSandbox(fid, 777)
	after := sandboxSnapshot(t, tm, fid)

	if string(before) != string(after) {
		t.Fatal("what-if sandbox mutated the real universe")
	}
	b1, err := json.Marshal(r1)
	if err != nil {
		t.Fatalf("marshal r1: %v", err)
	}
	b2, err := json.Marshal(r2)
	if err != nil {
		t.Fatalf("marshal r2: %v", err)
	}
	if string(b1) != string(b2) {
		t.Fatal("same scratch seed produced different what-if results")
	}

	if r1.Hypothetical.HomeXG < 0 || r1.Hypothetical.AwayXG < 0 {
		t.Fatalf("hypothetical xG negative: %+v", r1.Hypothetical)
	}
	if r1.Actual != nil {
		t.Fatal("scheduled fixture must not report an actual result")
	}
	if r1.Table == nil || !r1.Table.Applicable {
		t.Fatal("domestic league fixture must report table movement")
	}
	if r1.Table.Home == nil || r1.Table.Away == nil {
		t.Fatalf("table deltas missing: %+v", r1.Table)
	}
	// A scheduled fixture's hypothetical adds exactly one played game.
	if r1.Table.Home.AfterPts != r1.Table.Home.BeforePts+pointsFor(r1.Hypothetical.HomeGoals, r1.Hypothetical.AwayGoals) {
		t.Fatalf("home after pts=%d want %d", r1.Table.Home.AfterPts,
			r1.Table.Home.BeforePts+pointsFor(r1.Hypothetical.HomeGoals, r1.Hypothetical.AwayGoals))
	}
	if r1.Table.Away.AfterPts != r1.Table.Away.BeforePts+pointsFor(r1.Hypothetical.AwayGoals, r1.Hypothetical.HomeGoals) {
		t.Fatalf("away after pts=%d want %d", r1.Table.Away.AfterPts,
			r1.Table.Away.BeforePts+pointsFor(r1.Hypothetical.AwayGoals, r1.Hypothetical.HomeGoals))
	}
	if f.Status != "scheduled" {
		t.Fatalf("fixture status changed to %q", f.Status)
	}
}

func pointsFor(forGoals, againstGoals int) int {
	switch {
	case forGoals > againstGoals:
		return 3
	case forGoals == againstGoals:
		return 1
	default:
		return 0
	}
}

// TestWhatIfSandboxFinishedFixtureSwapsResult verifies the finished-fixture
// semantics: the hypothetical table replaces the recorded result instead of
// stacking on top of it.
func TestWhatIfSandboxFinishedFixtureSwapsResult(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	fid, _ := firstSlateFixtureID(t, tm, true)
	if out := tm.SimulateRemaining(); out["status"] != "success" {
		t.Fatalf("slate simulation failed: %v", out)
	}
	tm.mu.RLock()
	f := tm.findFixtureUnlocked(fid)
	finished := f != nil && f.Status == "finished"
	tm.mu.RUnlock()
	if !finished {
		t.Skip("first fixture was skipped by the slate (cup ordering); nothing to swap")
	}

	r, errMsg := tm.WhatIfSandbox(fid, 4242)
	if r == nil {
		t.Fatalf("what-if on finished fixture failed: %s", errMsg)
	}
	if r.Actual == nil {
		t.Fatal("finished fixture must report the recorded result")
	}
	if r.Table == nil || !r.Table.Applicable || r.Table.Home == nil || r.Table.Away == nil {
		t.Fatalf("table movement missing: %+v", r.Table)
	}
	wantHome := r.Table.Home.BeforePts - pointsFor(r.Actual.HomeGoals, r.Actual.AwayGoals) + pointsFor(r.Hypothetical.HomeGoals, r.Hypothetical.AwayGoals)
	if r.Table.Home.AfterPts != wantHome {
		t.Fatalf("home after pts=%d want %d (swap semantics)", r.Table.Home.AfterPts, wantHome)
	}
	wantAway := r.Table.Away.BeforePts - pointsFor(r.Actual.AwayGoals, r.Actual.HomeGoals) + pointsFor(r.Hypothetical.AwayGoals, r.Hypothetical.HomeGoals)
	if r.Table.Away.AfterPts != wantAway {
		t.Fatalf("away after pts=%d want %d (swap semantics)", r.Table.Away.AfterPts, wantAway)
	}
}

// TestWhatIfSandboxCupFixtureHasNoTableMovement checks the cup branch: the
// sandbox still resolves the match but reports no league-table movement.
func TestWhatIfSandboxCupFixtureHasNoTableMovement(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	// Cup competitions enter the calendar after the opening league weeks;
	// advance until one is scheduled (or the season runs out).
	cupID := ""
	scanCups := func() {
		tm.mu.RLock()
		defer tm.mu.RUnlock()
		sources := [][]Fixture{tm.Fixtures}
		if tm.World != nil {
			sources = append(sources, tm.World.Fixtures)
		}
		for _, fixtures := range sources {
			for _, f := range fixtures {
				if f.Status == "scheduled" && !tm.isWorldDomesticLeague(f.Competition) {
					if cupID == "" || f.FixtureID < cupID {
						cupID = f.FixtureID
					}
				}
			}
		}
	}
	for week := 0; week < tm.MaxMatchweeks+2 && cupID == ""; week++ {
		scanCups()
		if cupID == "" {
			if out := tm.SimulateRemaining(); out["status"] != "success" {
				t.Fatalf("slate simulation failed: %v", out)
			}
		}
	}
	if cupID == "" {
		t.Fatal("no scheduled cup fixture appeared all season")
	}
	r, errMsg := tm.WhatIfSandbox(cupID, 99)
	if r == nil {
		t.Fatalf("cup what-if failed: %s", errMsg)
	}
	if r.Table == nil || r.Table.Applicable {
		t.Fatalf("cup fixture must not report table movement: %+v", r.Table)
	}
}
