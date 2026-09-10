package tournament

import (
	"path/filepath"
	"testing"

	"football_sim/pkg/datamanager"
	"football_sim/pkg/growth"
)

func loadTestUniverse(t *testing.T) (*TournamentManager, *growth.GrowthEngine) {
	ge := growth.NewGrowthEngine(42)
	datasetPath := filepath.Join("..", "..", "..", "dataset.json")
	dm := datamanager.NewDataManager(datasetPath, ge)
	if dm == nil || len(dm.Clubs) == 0 {
		t.Fatalf("failed to load dataset from %s", datasetPath)
	}

	eliteClubs := dm.GetEliteClubs()
	if len(eliteClubs) != 12 {
		t.Fatalf("expected 12 elite clubs, got %d", len(eliteClubs))
	}

	tm := NewTournamentManager(eliteClubs, ge, 12345)
	return tm, ge
}

func TestTournamentManager_Initialization(t *testing.T) {
	tm, _ := loadTestUniverse(t)

	if len(tm.ClubsList) != 12 {
		t.Fatalf("expected 12 clubs, got %d", len(tm.ClubsList))
	}

	// 12 clubs * 11 opponents = 66 matches * 4 cycles = 264 total fixtures across 44 matchweeks (6 per week)
	expectedFixtures := 44 * 6
	if len(tm.Fixtures) != expectedFixtures {
		t.Errorf("expected %d league fixtures, got %d", expectedFixtures, len(tm.Fixtures))
	}

	if tm.CurrentMatchweek != 1 {
		t.Errorf("expected current matchweek 1, got %d", tm.CurrentMatchweek)
	}

	if len(tm.Inbox) == 0 {
		t.Errorf("expected opening inbox news item")
	}

	standings := tm.GetStandings()
	if len(standings) != 12 {
		t.Errorf("expected 12 teams in standings, got %d", len(standings))
	}
}

func TestTournamentManager_SeniorMentorshipPairing(t *testing.T) {
	tm, _ := loadTestUniverse(t)

	// Check wonderkid mentorship pairing
	foundProdigy := false
	for _, club := range tm.ClubsList {
		for _, p := range club.Squad {
			if p.UniverseWonderkid {
				foundProdigy = true
				if p.MentorID == "" || p.MentorName == "" || p.MentorOVR <= 0 {
					t.Errorf("wonderkid %s at %s has incomplete mentor: ID=%q Name=%q OVR=%d",
						p.FullName, club.ShortName, p.MentorID, p.MentorName, p.MentorOVR)
				}
			}
		}
	}

	if !foundProdigy {
		t.Errorf("expected to find wonderkids across elite clubs")
	}
}

func TestTournamentManager_CupsInitialized(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	if tm.UCLStage != "GROUP_STAGE" {
		t.Errorf("expected GROUP_STAGE, got %s", tm.UCLStage)
	}
	if len(tm.UCLGroupA) != 6 || len(tm.UCLGroupB) != 6 {
		t.Errorf("expected 6+6 UCL groups, got %d+%d", len(tm.UCLGroupA), len(tm.UCLGroupB))
	}
	if len(tm.UCLFixtures) == 0 {
		t.Errorf("expected UCL group fixtures")
	}
	if tm.SuperCupStage != "PLAY_IN" {
		t.Errorf("expected Super Cup PLAY_IN, got %s", tm.SuperCupStage)
	}
	if len(tm.SuperCupFixtures) != 4 {
		t.Errorf("expected 4 Super Cup play-in fixtures, got %d", len(tm.SuperCupFixtures))
	}
	slate := tm.GetSlate(1)
	if len(slate) != 6 {
		t.Errorf("week 1 should be 6 league games, got %d", len(slate))
	}
	slate5 := tm.GetSlate(5)
	cups := 0
	for _, f := range slate5 {
		if f.Competition == "super-cup" {
			cups++
		}
	}
	if cups != 4 {
		t.Errorf("week 5 should include 4 Super Cup play-ins, got %d cups of %d total", cups, len(slate5))
	}
}

func TestTournamentManager_SimulateMatchweek(t *testing.T) {
	tm, _ := loadTestUniverse(t)

	res := tm.SimulateMatchweek(1)
	if res["status"] != "success" {
		t.Fatalf("failed to simulate matchweek 1: %v", res)
	}

	if tm.CurrentMatchweek != 2 {
		t.Errorf("expected current matchweek to advance to 2, got %d", tm.CurrentMatchweek)
	}

	// Verify all 6 fixtures for MW 1 are finished
	mw1Fixtures := tm.GetMatchweekFixtures(1)
	if len(mw1Fixtures) != 6 {
		t.Fatalf("expected 6 fixtures in MW 1, got %d", len(mw1Fixtures))
	}

	for _, f := range mw1Fixtures {
		if f.Status != "finished" {
			t.Errorf("fixture %s status expected finished, got %s", f.FixtureID, f.Status)
		}
		if f.HomeGoals == nil || f.AwayGoals == nil {
			t.Errorf("fixture %s missing score", f.FixtureID)
		}
	}

	// Standings updated
	standings := tm.GetStandings()
	var totalPlayed int
	for _, c := range standings {
		totalPlayed += c.Played
	}
	if totalPlayed != 12 {
		t.Errorf("expected 12 total team appearances (6 matches * 2), got %d", totalPlayed)
	}
}

func TestTournamentManager_NXGN50Rankings(t *testing.T) {
	tm, ge := loadTestUniverse(t)

	rankings := GenerateNXGN50(tm.ClubsList, ge)
	if len(rankings) == 0 {
		t.Fatalf("expected non-empty NXGN rankings")
	}

	top := rankings[0]
	if top.Potential < 90 || top.Potential > 99 {
		t.Errorf("unexpected top wonderkid potential: %d", top.Potential)
	}

	// Ensure no wonderkids report fake 99 potential
	for _, r := range rankings {
		if r.IsWonderkid && r.Potential == 99 {
			t.Errorf("wonderkid %s has potential 99; expected authentic biometrics potential (93-96)", r.FullName)
		}
	}
}

func TestTournamentManager_HeadToHead(t *testing.T) {
	tm, _ := loadTestUniverse(t)

	// Simulate first 5 matchweeks
	for mw := 1; mw <= 5; mw++ {
		tm.SimulateMatchweek(mw)
	}

	h2h := tm.GetHeadToHead("LAL-BAR", "LAL-RMA")
	if h2h == nil {
		t.Fatalf("expected non-nil H2H result for El Clasico")
	}

	if h2h["derby_name"] != "El Clasico" {
		t.Errorf("expected derby_name El Clasico, got %v", h2h["derby_name"])
	}
}

func TestTournamentManager_TrophyCabinetAndAwards(t *testing.T) {
	tm, _ := loadTestUniverse(t)

	cabinet := tm.GetTrophyCabinet()
	if len(cabinet) != 12 {
		t.Fatalf("expected 12 clubs in trophy cabinet, got %d", len(cabinet))
	}

	awards := tm.GetSeasonAwards()
	if awards["ballon_dor"] == nil {
		t.Errorf("expected ballon_dor rankings in awards")
	}
}
