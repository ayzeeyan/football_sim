package tournament

import (
	"testing"
)

func TestMatchImportanceAndStorylines(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)

	if len(tm.ClubsList) == 0 {
		t.Fatal("no clubs loaded")
	}

	favID := tm.ClubsList[0].ClubID

	// 1. Test Club Pulse
	pulse := tm.GenerateClubPulse(favID)
	if pulse == nil {
		t.Fatal("expected non-nil club pulse")
	}
	if pulse["club_id"] != favID {
		t.Errorf("expected club_id %s, got %v", favID, pulse["club_id"])
	}
	if _, ok := pulse["board_confidence"]; !ok {
		t.Error("expected board_confidence in pulse")
	}
	if _, ok := pulse["financial_health"]; !ok {
		t.Error("expected financial_health in pulse")
	}

	// 2. Test Season Storylines
	storylines := tm.GenerateSeasonStorylines(favID)
	if len(storylines) == 0 {
		t.Error("expected at least 1 storyline")
	}

	worldA := tm.GenerateWorldStorylines()
	if len(worldA) == 0 {
		t.Fatal("expected world storylines")
	}
	if !tm.SetFavouriteClubID(tm.ClubsList[len(tm.ClubsList)-1].ClubID) {
		t.Fatal("could not flip viewing preference")
	}
	worldB := tm.GenerateWorldStorylines()
	if len(worldA) != len(worldB) {
		t.Fatalf("world storylines changed with favourite club: %v vs %v", worldA, worldB)
	}
	for i := range worldA {
		if worldA[i] != worldB[i] {
			t.Fatalf("world storyline %d changed with favourite club: %q vs %q", i, worldA[i], worldB[i])
		}
	}

	// 3. Test Match Importance
	fixture := &Fixture{
		FixtureID:   "test-final",
		Competition: "fa-cup",
		Stage:       "final",
		HomeID:      favID,
		AwayID:      tm.ClubsList[1].ClubID,
	}
	imp := tm.EvaluateMatchImportance(fixture)
	if imp != ImportanceFinal {
		t.Errorf("expected Final importance, got %s", imp)
	}

	derbyFixture := &Fixture{
		FixtureID:   "test-derby",
		Competition: "premier-league",
		Stage:       "league",
		HomeID:      "EPL-ARS",
		AwayID:      "EPL-TOT",
		DerbyName:   "North London Derby",
		DerbyHeat:   85,
	}
	impDerby := tm.EvaluateMatchImportance(derbyFixture)
	if impDerby != ImportanceDerby {
		t.Errorf("expected Derby importance, got %s", impDerby)
	}
}
