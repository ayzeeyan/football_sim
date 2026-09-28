package tournament

import (
	"testing"

	"football_sim/pkg/transfers"
)

// Multi-season soak: the closed country-pure pyramid never moves a club
// between leagues. Every transition must conserve league membership, keep
// the world valid, publish the survival-battle news, and stay deterministic
// across identical seeds.
func TestRelegationStakesMultiSeasonSoak(t *testing.T) {
	tm, _, te := loadEuropeanWorldForTest(t)

	startLeague := map[string]string{}
	leagueSizes := func() map[string]int {
		sizes := map[string]int{}
		for _, club := range tm.ClubsList {
			sizes[club.League]++
		}
		return sizes
	}
	for _, club := range tm.ClubsList {
		startLeague[club.ClubID] = club.League
	}
	startSizes := leagueSizes()

	runSeasonAndTransition := func(season int) {
		batch := tm.SimulateBatchWeeks(tm.MaxMatchweeks)
		if batch.Status != "success" || !batch.SeasonFinished {
			t.Fatalf("season %d did not finish: %+v", season, batch)
		}
		te.BeginOffSeasonWindow()
		for i := 0; i < transfers.TransferWindowWeeks; i++ {
			te.AdvanceOpenWindow()
		}
		if transition := tm.FinalizeSeasonTransition(); transition["status"] != "success" {
			t.Fatalf("season %d transition failed: %v", season, transition)
		}
	}

	runSeasonAndTransition(1)

	// No club changed league across the transition.
	for _, club := range tm.ClubsList {
		if club.League != startLeague[club.ClubID] {
			t.Fatalf("%s changed league: %q -> %q (cross-country moves are forbidden)", club.ClubID, startLeague[club.ClubID], club.League)
		}
	}
	// League membership is conserved.
	afterSizes := leagueSizes()
	for league, size := range startSizes {
		if afterSizes[league] != size {
			t.Fatalf("league %s size drifted: %d -> %d", league, size, afterSizes[league])
		}
	}
	// The world stays valid after the stakes (schedules, participants, cups).
	if err := tm.ValidateWorldState(); err != nil {
		t.Fatalf("world invalid after transition: %v", err)
	}
	// The survival-battle news item was published.
	foundNews := false
	for _, item := range tm.Inbox {
		if item.Headline == "Survival battle settled: relegation stakes paid" {
			foundNews = true
		}
	}
	if !foundNews {
		t.Fatal("no survival-battle news item was pushed")
	}
	// The persisted move ledger stays an empty array, never null.
	if tm.SeasonLeagueMoves == nil || len(tm.SeasonLeagueMoves) != 0 {
		t.Fatalf("SeasonLeagueMoves must be empty, got %+v", tm.SeasonLeagueMoves)
	}

	// Second season: the invariants hold across another transition — a soak
	// against drift.
	runSeasonAndTransition(2)
	for _, club := range tm.ClubsList {
		if club.League != startLeague[club.ClubID] {
			t.Fatalf("%s changed league in season 2: %q -> %q", club.ClubID, startLeague[club.ClubID], club.League)
		}
	}
	after2 := leagueSizes()
	for league, size := range startSizes {
		if after2[league] != size {
			t.Fatalf("league %s size drifted after season 2: %d -> %d", league, size, after2[league])
		}
	}
	if err := tm.ValidateWorldState(); err != nil {
		t.Fatalf("world invalid after second transition: %v", err)
	}
}

// Identical seeds produce identical stakes plans.
func TestRelegationStakesPlanIsDeterministicAcrossSeeds(t *testing.T) {
	run := func() []RelegationStake {
		tm, _, _ := loadEuropeanWorldForTest(t)
		batch := tm.SimulateBatchWeeks(tm.MaxMatchweeks)
		if batch.Status != "success" {
			t.Fatalf("season did not finish: %+v", batch)
		}
		return tm.PlanDomesticRelegationStakes()
	}
	a := run()
	b := run()
	if len(a) != len(b) {
		t.Fatalf("plans differ in length: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("stake %d differs between identical seeds: %+v vs %+v", i, a[i], b[i])
		}
	}
}
