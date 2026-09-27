package tournament

import (
	"testing"

	"football_sim/pkg/transfers"
)

// Multi-season soak: promotion and relegation must move clubs across league
// boundaries at every transition, conserve league sizes, keep the world
// valid, and stay deterministic across identical seeds.
func TestPromotionRelegationMultiSeasonSoak(t *testing.T) {
	tm, _, te := loadEuropeanWorldForTest(t)

	leagueSizes := func() map[string]int {
		sizes := map[string]int{}
		for _, club := range tm.ClubsList {
			sizes[club.League]++
		}
		return sizes
	}
	startSizes := leagueSizes()

	runSeasonAndTransition := func(season int) []RelegationMove {
		batch := tm.SimulateBatchWeeks(tm.MaxMatchweeks)
		if batch.Status != "success" || !batch.SeasonFinished {
			t.Fatalf("season %d did not finish: %+v", season, batch)
		}
		// Capture the final tables before the transition mutates anything.
		moves := tm.PlanDomesticPromotionRelegation()
		if len(moves) == 0 {
			t.Fatalf("season %d planned no promotion/relegation moves", season)
		}
		te.BeginOffSeasonWindow()
		for i := 0; i < transfers.TransferWindowWeeks; i++ {
			te.AdvanceOpenWindow()
		}
		if transition := tm.FinalizeSeasonTransition(); transition["status"] != "success" {
			t.Fatalf("season %d transition failed: %v", season, transition)
		}
		return moves
	}

	moves1 := runSeasonAndTransition(1)

	// Every planned move was applied to the club's League field.
	for _, move := range moves1 {
		club := tm.Clubs[move.ClubID]
		if club == nil {
			t.Fatalf("move references unknown club %s", move.ClubID)
		}
		if club.League != move.ToLeague {
			t.Fatalf("%s league=%q want %q", move.ClubID, club.League, move.ToLeague)
		}
	}
	// League sizes are conserved.
	afterSizes := leagueSizes()
	for league, size := range startSizes {
		if afterSizes[league] != size {
			t.Fatalf("league %s size drifted: %d -> %d", league, size, afterSizes[league])
		}
	}
	// The world stays valid after the swap (schedules, participants, cups).
	if err := tm.ValidateWorldState(); err != nil {
		t.Fatalf("world invalid after transition: %v", err)
	}
	// The relegation news item was published.
	foundNews := false
	for _, item := range tm.Inbox {
		if item.Headline == "Promotion and relegation confirmed" {
			foundNews = true
		}
	}
	if !foundNews {
		t.Fatal("no promotion/relegation news item was pushed")
	}

	// Second season: the swap repeats, sizes stay conserved, and the world
	// remains valid — a soak against drift across transitions.
	moves2 := runSeasonAndTransition(2)
	if len(moves2) == 0 {
		t.Fatal("season 2 planned no moves")
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
	// A club that moved in season 1 can move again: the boundary exchange is
	// unbounded across seasons (worst-to-champion journeys are possible).
	t.Logf("season 1 moves=%d season 2 moves=%d", len(moves1), len(moves2))
}

// Identical seeds produce identical first-season swap plans.
func TestPromotionRelegationPlanIsDeterministicAcrossSeeds(t *testing.T) {
	run := func() []RelegationMove {
		tm, _, te := loadEuropeanWorldForTest(t)
		batch := tm.SimulateBatchWeeks(tm.MaxMatchweeks)
		if batch.Status != "success" {
			t.Fatalf("season did not finish: %+v", batch)
		}
		_ = te
		return tm.PlanDomesticPromotionRelegation()
	}
	a := run()
	b := run()
	if len(a) != len(b) {
		t.Fatalf("plans differ in length: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("move %d differs between identical seeds: %+v vs %+v", i, a[i], b[i])
		}
	}
}
