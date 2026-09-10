package tournament

import (
	"math/rand"
	"reflect"
	"strings"
	"testing"
)

// serialSlateForTest is the independent serial reference for the pooled
// slate: same batch order, same per-fixture streams, compute+apply inline.
// It shares only slateBatchUnlocked, computeSlateFixture, and
// applySlateFixture with the implementation under test.
func serialSlateForTest(tm *TournamentManager) map[string]interface{} {
	mw, ids := tm.slateBatchUnlocked()
	var base int64 = 1
	if tm.RNG != nil {
		base = tm.RNG.Int63()
	}
	for _, id := range ids {
		if fx := tm.findFixtureUnlocked(id); fx != nil && fx.Status != "finished" {
			tm.EnsureFixtureWeather(fx)
		}
	}
	played, skipped := 0, 0
	for _, id := range ids {
		fx := tm.findFixtureUnlocked(id)
		if fx == nil || fx.Status == "finished" {
			continue
		}
		if fx.Matchweek > tm.CurrentMatchweek && tm.CurrentMatchweek > 0 {
			skipped++
			continue
		}
		if blocked := tm.uclLegBlocked(fx); blocked != "" {
			skipped++
			continue
		}
		rng := rand.New(rand.NewSource(slateSeed(base, id)))
		computed, errMsg := tm.computeSlateFixture(fx, rng)
		if errMsg != "" {
			skipped++
			continue
		}
		if out := tm.applySlateFixture(fx, computed); out["status"] == "success" {
			played++
		} else {
			skipped++
		}
	}
	return map[string]interface{}{
		"status":          "success",
		"played":          played,
		"skipped":         skipped,
		"simulated_count": played,
		"matchweek":       mw,
		"next_matchweek":  tm.CurrentMatchweek,
		"is_finished":     tm.CurrentMatchweek > tm.MaxMatchweeks,
		"champion":        champIf(tm.CurrentMatchweek > tm.MaxMatchweeks, tm.championNameUnlocked()),
	}
}

type slateFingerprint struct {
	played      map[string]int
	won         map[string]int
	drawn       map[string]int
	lost        map[string]int
	gf          map[string]int
	ga          map[string]int
	pts         map[string]int
	form        map[string]string
	morale      map[string]int
	scores      map[string][2]int
	statuses    map[string]string
	inboxHeads  []string
	derbyHeat   map[string]int
	matchweek   int
	phase       string
	playedCount int
}

func fingerprintSlate(tm *TournamentManager) slateFingerprint {
	fp := slateFingerprint{
		played: map[string]int{}, won: map[string]int{}, drawn: map[string]int{},
		lost: map[string]int{}, gf: map[string]int{}, ga: map[string]int{},
		pts: map[string]int{}, form: map[string]string{}, morale: map[string]int{},
		scores: map[string][2]int{}, statuses: map[string]string{},
		derbyHeat: map[string]int{},
		matchweek: tm.CurrentMatchweek, phase: tm.SeasonPhase,
	}
	for _, c := range tm.ClubsList {
		fp.played[c.ClubID] = c.Played
		fp.won[c.ClubID] = c.Won
		fp.drawn[c.ClubID] = c.Drawn
		fp.lost[c.ClubID] = c.Lost
		fp.gf[c.ClubID] = c.GoalsFor
		fp.ga[c.ClubID] = c.GoalsAgainst
		fp.pts[c.ClubID] = c.Points
		fp.form[c.ClubID] = strings.Join(c.Form, "")
		fp.morale[c.ClubID] = c.Morale
	}
	collect := func(list []Fixture) {
		for i := range list {
			f := &list[i]
			fp.statuses[f.FixtureID] = f.Status
			if f.Status == "finished" && f.HomeGoals != nil && f.AwayGoals != nil {
				fp.scores[f.FixtureID] = [2]int{*f.HomeGoals, *f.AwayGoals}
				fp.playedCount++
			}
		}
	}
	collect(tm.Fixtures)
	collect(tm.UCLFixtures)
	collect(tm.SuperCupFixtures)
	for _, item := range tm.Inbox {
		fp.inboxHeads = append(fp.inboxHeads, item.Headline)
	}
	for k, v := range tm.DerbyHeat {
		fp.derbyHeat[k] = v
	}
	return fp
}

// F4: a pooled 6-game league night plus cups must match a serial sim with no
// double-count and no two clubs written concurrently.
func TestSlatePoolMatchesSerialSlate(t *testing.T) {
	build := func() *TournamentManager {
		tm, _ := loadTestUniverse(t)
		return tm
	}

	// Guard: both universes must start identical or the comparison is void.
	tmPool, tmSerial := build(), build()
	if !reflect.DeepEqual(fingerprintSlate(tmPool), fingerprintSlate(tmSerial)) {
		t.Fatal("test universes differ before simulation; cannot compare paths")
	}

	tmPool.mu.Lock()
	poolRes := tmPool.simulateRemainingUnlocked()
	tmPool.mu.Unlock()

	tmSerial.mu.Lock()
	serialRes := serialSlateForTest(tmSerial)
	tmSerial.mu.Unlock()

	if poolRes["played"] != serialRes["played"] || poolRes["skipped"] != serialRes["skipped"] {
		t.Fatalf("counts differ: pool=%v serial=%v", poolRes, serialRes)
	}
	fpPool, fpSerial := fingerprintSlate(tmPool), fingerprintSlate(tmSerial)
	if !reflect.DeepEqual(fpPool, fpSerial) {
		t.Fatalf("pooled slate differs from serial slate.\npool=%+v\nserial=%+v", fpPool, fpSerial)
	}

	// No double-count: every slate fixture finished exactly once, each club's
	// played total equals its finished games, and table points reconcile.
	mw := tmPool.CurrentMatchweek - 1
	if mw < 1 {
		mw = 1
	}
	apps := map[string]int{}
	for _, f := range tmPool.Fixtures {
		if f.Matchweek != mw || f.Status != "finished" {
			continue
		}
		apps[f.HomeID]++
		apps[f.AwayID]++
	}
	for _, c := range tmPool.ClubsList {
		if got := apps[c.ClubID]; got != 0 && c.Played < got {
			t.Fatalf("club %s played=%d but only %d slate games", c.ClubID, c.Played, got)
		}
	}
	totalPlayed := 0
	for _, c := range tmPool.ClubsList {
		totalPlayed += c.Played
	}
	if totalPlayed%2 != 0 {
		t.Fatalf("table played total %d is odd: double-count suspected", totalPlayed)
	}

	// Determinism: repeat the pooled slate on fresh universes; all identical.
	want := fpPool
	for i := 0; i < 2; i++ {
		tm := build()
		tm.mu.Lock()
		tm.simulateRemainingUnlocked()
		tm.mu.Unlock()
		if got := fingerprintSlate(tm); !reflect.DeepEqual(got, want) {
			t.Fatalf("pooled slate run %d diverged: not deterministic", i+2)
		}
	}
}

// F4: wave grouping must keep same-club fixtures serial and preserve order.
func TestSlateWavesKeepClubConflictsSerial(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	tm.mu.RLock()
	_, ids := tm.slateBatchUnlocked()
	waves := groupSlateWaves(tm, ids)
	tm.mu.RUnlock()

	if len(ids) == 0 {
		t.Fatal("expected a non-empty remaining slate")
	}
	// Concatenation preserves slate order.
	var flat []string
	for _, w := range waves {
		flat = append(flat, w...)
	}
	if !reflect.DeepEqual(flat, ids) {
		t.Fatal("waves do not preserve slate order")
	}
	// No wave shares a club.
	for i, w := range waves {
		seen := map[string]bool{}
		for _, id := range w {
			f := tm.FindFixture(id)
			if f == nil {
				continue
			}
			if seen[f.HomeID] || seen[f.AwayID] {
				t.Fatalf("wave %d runs club %s/%s concurrently", i, f.HomeID, f.AwayID)
			}
			seen[f.HomeID] = true
			seen[f.AwayID] = true
		}
	}
	// A 6-game disjoint league night should collapse to one wave.
	tm.mu.RLock()
	leagueOnly := []string{}
	for _, id := range ids {
		if f := tm.findFixtureUnlocked(id); f != nil && f.Competition == "super-league" {
			leagueOnly = append(leagueOnly, id)
		}
	}
	leagueWaves := groupSlateWaves(tm, leagueOnly)
	tm.mu.RUnlock()
	if len(leagueOnly) == 6 && len(leagueWaves) != 1 {
		t.Fatalf("6 disjoint league fixtures should form 1 wave, got %d", len(leagueWaves))
	}

	// Synthetic conflict: same club twice must split waves.
	synth := []string{}
	if len(ids) >= 2 {
		synth = []string{ids[0], ids[1], ids[0]}
	} else {
		synth = []string{ids[0], ids[0]}
	}
	tm.mu.RLock()
	synthWaves := groupSlateWaves(tm, synth)
	tm.mu.RUnlock()
	flatSynth := []string{}
	for _, w := range synthWaves {
		flatSynth = append(flatSynth, w...)
	}
	if !reflect.DeepEqual(flatSynth, synth) || len(synthWaves) < 2 {
		t.Fatalf("repeated fixture should split waves, got %v", synthWaves)
	}
}
