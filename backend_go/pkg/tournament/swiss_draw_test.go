package tournament

import (
	"path/filepath"
	"testing"

	"football_sim/pkg/datamanager"
	"football_sim/pkg/growth"
	"football_sim/pkg/models"
)

// checkSwissWorld asserts the exact Champions League league-phase contract:
// 36 clubs, 144 fixtures, 8 games each, 4 home / 4 away each, exactly 2
// opponents from each persisted pot, 18 fixtures per phase week with no club
// doubling up, and no repeated pairing.
func checkSwissWorld(t *testing.T, seed int64) {
	t.Helper()
	ge := growth.NewGrowthEngine(seed)
	dm := datamanager.NewDataManager(filepath.Join("..", "..", "..", "dataset.json"), ge)
	tm := NewEuropeanWorldManager(dm.ClubsList, ge, seed)
	comp := tm.World.Competitions["champions-league"]
	if len(comp.ParticipantIDs) != 36 || len(comp.LeaguePhaseFixtureIDs) != 144 {
		t.Fatalf("seed %d: participants=%d fixtures=%d", seed, len(comp.ParticipantIDs), len(comp.LeaguePhaseFixtureIDs))
	}
	// Pots are snapshotted at draw time (ratings drift later via loans) and
	// must partition all 36 participants into 4x9.
	if len(comp.Pots) != 4 {
		t.Fatalf("seed %d: pots not persisted (%d)", seed, len(comp.Pots))
	}
	potOf := map[string]int{}
	covered := map[string]bool{}
	for p, ids := range comp.Pots {
		if len(ids) != 9 {
			t.Fatalf("seed %d: pot %d has %d clubs", seed, p, len(ids))
		}
		for _, id := range ids {
			if covered[id] {
				t.Fatalf("seed %d: %s in two pots", seed, id)
			}
			covered[id] = true
			potOf[id] = p
		}
	}
	for _, id := range comp.ParticipantIDs {
		if !covered[id] {
			t.Fatalf("seed %d: participant %s missing from pots", seed, id)
		}
	}
	games := map[string]int{}
	homes := map[string]int{}
	aways := map[string]int{}
	oppPots := map[string]map[int]int{}
	weekUse := map[int]map[string]bool{}
	weekCount := map[int]int{}
	seenPair := map[string]bool{}
	for _, fid := range comp.LeaguePhaseFixtureIDs {
		f := tm.worldFixtureUnlocked(fid)
		if f == nil {
			t.Fatalf("seed %d: missing fixture %s", seed, fid)
		}
		if f.Stage != "League Phase" {
			t.Fatalf("seed %d: %s has stage %q", seed, fid, f.Stage)
		}
		games[f.HomeID]++
		games[f.AwayID]++
		homes[f.HomeID]++
		aways[f.AwayID]++
		if oppPots[f.HomeID] == nil {
			oppPots[f.HomeID] = map[int]int{}
		}
		if oppPots[f.AwayID] == nil {
			oppPots[f.AwayID] = map[int]int{}
		}
		oppPots[f.HomeID][potOf[f.AwayID]]++
		oppPots[f.AwayID][potOf[f.HomeID]]++
		if weekUse[f.Matchweek] == nil {
			weekUse[f.Matchweek] = map[string]bool{}
		}
		if weekUse[f.Matchweek][f.HomeID] || weekUse[f.Matchweek][f.AwayID] {
			t.Fatalf("seed %d: club doubles up in week %d (%s)", seed, f.Matchweek, fid)
		}
		weekUse[f.Matchweek][f.HomeID] = true
		weekUse[f.Matchweek][f.AwayID] = true
		weekCount[f.Matchweek]++
		a, b := f.HomeID, f.AwayID
		if a > b {
			a, b = b, a
		}
		k := a + "\x00" + b
		if seenPair[k] {
			t.Fatalf("seed %d: duplicate pairing %s vs %s", seed, a, b)
		}
		seenPair[k] = true
	}
	for _, id := range comp.ParticipantIDs {
		if games[id] != 8 {
			t.Fatalf("seed %d: %s games=%d", seed, id, games[id])
		}
		if homes[id] != 4 || aways[id] != 4 {
			t.Fatalf("seed %d: %s H=%d A=%d", seed, id, homes[id], aways[id])
		}
		for p := 0; p < 4; p++ {
			if oppPots[id][p] != 2 {
				t.Fatalf("seed %d: %s pots=%v", seed, id, oppPots[id])
			}
		}
	}
	for _, mw := range europeanPhaseWeeks() {
		if weekCount[mw] != 18 {
			t.Fatalf("seed %d: week %d has %d fixtures, want 18", seed, mw, weekCount[mw])
		}
	}
}

func TestChampionsLeagueSwissDrawIsExact(t *testing.T) {
	for _, seed := range []int64{8181, 1, 2, 3, 42, 20260907, 999983} {
		checkSwissWorld(t, seed)
	}
}

func TestChampionsLeagueSwissDrawIsDeterministic(t *testing.T) {
	build := func() []string {
		ge := growth.NewGrowthEngine(8181)
		dm := datamanager.NewDataManager(filepath.Join("..", "..", "..", "dataset.json"), ge)
		tm := NewEuropeanWorldManager(dm.ClubsList, ge, 8181)
		return append([]string(nil), tm.World.Competitions["champions-league"].LeaguePhaseFixtureIDs...)
	}
	a, b := build(), build()
	if len(a) != len(b) {
		t.Fatalf("fixture count mismatch %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("nondeterministic draw at %d: %s vs %s", i, a[i], b[i])
		}
	}
}

// TestSwissSplitWeeksSucceedsAcrossSeeds pins the weekly-decomposition step
// directly on block-built draws: 8 perfect matchings of 18, every club once
// per week. A failure here would force the circle fallback in scheduling.
func TestSwissSplitWeeksSucceedsAcrossSeeds(t *testing.T) {
	ge := growth.NewGrowthEngine(8181)
	dm := datamanager.NewDataManager(filepath.Join("..", "..", "..", "dataset.json"), ge)
	tm := NewEuropeanWorldManager(dm.ClubsList, ge, 8181)
	comp := tm.World.Competitions["champions-league"]
	byID := map[string]*models.Club{}
	for _, c := range clubsForIDs(tm.Clubs, comp.ParticipantIDs) {
		byID[c.ClubID] = c
	}
	pots := make([][]*models.Club, 4)
	leagueOf := map[string]string{}
	for p, ids := range comp.Pots {
		for _, id := range ids {
			pots[p] = append(pots[p], byID[id])
			leagueOf[id] = byID[id].League
		}
	}
	pairs := swissBlockPairs(8181, comp.ID, pots, leagueOf)
	if len(pairs) != 144 {
		t.Fatalf("block pairs=%d want 144", len(pairs))
	}
	for _, seed := range []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30} {
		wks, ok := swissSplitWeeks(seed, comp.ID, pairs)
		if !ok {
			t.Fatalf("seed %d: split failed", seed)
		}
		if len(wks) != 8 {
			t.Fatalf("seed %d: weeks=%d", seed, len(wks))
		}
		for w, wk := range wks {
			if len(wk) != 18 {
				t.Fatalf("seed %d week %d: pairs=%d", seed, w, len(wk))
			}
			used := map[string]bool{}
			for _, pr := range wk {
				if used[pr[0]] || used[pr[1]] {
					t.Fatalf("seed %d week %d: club doubles up", seed, w)
				}
				used[pr[0]], used[pr[1]] = true, true
			}
			if len(used) != 36 {
				t.Fatalf("seed %d week %d: covers %d clubs", seed, w, len(used))
			}
		}
	}
}
