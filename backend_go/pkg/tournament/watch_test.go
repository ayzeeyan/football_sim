package tournament

import (
	"testing"

	"football_sim/pkg/datamanager"
	"football_sim/pkg/growth"
)

func testManagerForWatch(t *testing.T) *TournamentManager {
	t.Helper()
	ge := growth.NewGrowthEngine(7)
	dm := datamanager.NewDataManager("../../dataset.json", ge)
	if dm == nil {
		t.Skip("dataset unavailable")
	}
	elite := dm.GetEliteClubs()
	if len(elite) != 12 {
		t.Skip("elite clubs unavailable")
	}
	return NewTournamentManager(elite, ge, 7)
}

func TestFavouritePersistsAndWeekWatch(t *testing.T) {
	tm := testManagerForWatch(t)
	if tm.FavouriteClub() != "" {
		t.Fatal("favourite should start empty")
	}
	fav := tm.ClubsList[0].ClubID
	if !tm.SetFavouriteClubID(fav) {
		t.Fatal("could not set favourite")
	}
	if tm.FavouriteClub() != fav {
		t.Fatal("favourite not returned")
	}
	if tm.SetFavouriteClubID("NOPE") {
		t.Fatal("unknown club accepted")
	}
	favID, fixture, cups, mw := tm.WeekWatch()
	if favID != fav {
		t.Fatalf("watch fav %q want %q", favID, fav)
	}
	if mw != tm.CurrentMatchweek {
		t.Fatalf("watch mw %d want %d", mw, tm.CurrentMatchweek)
	}
	if fixture == nil {
		t.Fatal("expected favourite fixture on MW1 slate")
	}
	if fixture.HomeID != fav && fixture.AwayID != fav {
		t.Fatalf("fixture %s does not involve favourite %s", fixture.FixtureID, fav)
	}
	_ = cups
}

func TestSimulateRemainingExcludingSkipsLive(t *testing.T) {
	tm := testManagerForWatch(t)
	mw, ids := tm.slateBatchUnlocked()
	if len(ids) < 2 {
		t.Skip("slate too small")
	}
	liveID := ids[0]
	res := tm.SimulateRemainingExcluding(liveID)
	if res["status"] != "success" {
		t.Fatalf("sim failed: %v", res)
	}
	if got, _ := res["excluded_fixture_id"].(string); got != liveID {
		t.Fatalf("excluded %q want %q", got, liveID)
	}
	if f := tm.findFixtureUnlocked(liveID); f == nil || f.Status != "scheduled" {
		t.Fatalf("live fixture should stay scheduled, got %+v", f)
	}
	played, _ := res["played"].(int)
	if played != len(ids)-1 {
		// Blocked cup legs may skip; at minimum nothing replayed and live kept.
		if played > len(ids)-1 {
			t.Fatalf("played %d exceeds slate %d", played, len(ids))
		}
	}
	// Simulate the live fixture afterwards (as the live commit would), then a
	// second sweep must not replay it.
	single := tm.SimulateFixture(liveID)
	if single["status"] != "success" {
		t.Fatalf("live sim failed: %v", single)
	}
	before := tm.findFixtureUnlocked(liveID)
	hg, ag := *before.HomeGoals, *before.AwayGoals
	res2 := tm.SimulateRemainingExcluding("something-else")
	_ = res2
	after := tm.findFixtureUnlocked(liveID)
	if *after.HomeGoals != hg || *after.AwayGoals != ag {
		t.Fatal("finished live result was replayed by sim-remaining")
	}
	_ = mw
}
