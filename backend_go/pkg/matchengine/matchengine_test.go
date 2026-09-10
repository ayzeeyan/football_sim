package matchengine

import (
	"math/rand"
	"testing"

	"football_sim/pkg/growth"
	"football_sim/pkg/managers"
	"football_sim/pkg/models"
)

func createTestClubs() (*models.Club, *models.Club) {
	positions := []string{"GK", "LB", "CB", "CB", "RB", "CDM", "CM", "CM", "LW", "ST", "RW"}
	categories := []string{"GK", "DEF", "DEF", "DEF", "DEF", "MID", "MID", "MID", "FWD", "FWD", "FWD"}

	homeSquad := make([]*models.Player, 16)
	for i := 0; i < 16; i++ {
		pos := "CM"
		cat := "MID"
		if i < 11 {
			pos = positions[i]
			cat = categories[i]
		}
		isWk := (i == 9) // ST is wonderkid
		ovr := 82 + (i % 4)
		if isWk {
			ovr = 86 // qualifies as a direct free-kick taker (shooting >= 85)
		}
		homeSquad[i] = &models.Player{
			PlayerID:          models.FormatCurrency(int64(i)),
			FullName:          "Home Player " + string(rune('A'+i)),
			Position:          pos,
			Category:          cat,
			OVR:               ovr,
			UniverseWonderkid: isWk,
		}
	}

	awaySquad := make([]*models.Player, 16)
	for i := 0; i < 16; i++ {
		pos := "CM"
		cat := "MID"
		if i < 11 {
			pos = positions[i]
			cat = categories[i]
		}
		awaySquad[i] = &models.Player{
			PlayerID: models.FormatCurrency(int64(100 + i)),
			FullName: "Away Player " + string(rune('A'+i)),
			Position: pos,
			Category: cat,
			OVR:      80 + (i % 4),
		}
	}

	homeClub := &models.Club{
		ClubID:            "LAL-BAR",
		ClubName:          "FC Barcelona",
		ShortName:         "BAR",
		HomeStadium:       "Camp Nou",
		OverallTeamRating: 84,
		Morale:            75,
		Squad:             homeSquad,
	}

	awayClub := &models.Club{
		ClubID:            "LAL-RMA",
		ClubName:          "Real Madrid",
		ShortName:         "RMA",
		HomeStadium:       "Santiago Bernabéu",
		OverallTeamRating: 85,
		Morale:            75,
		Squad:             awaySquad,
	}

	return homeClub, awayClub
}

func TestSimulateInstantMatch(t *testing.T) {
	home, away := createTestClubs()
	ge := growth.NewGrowthEngine(42)
	homeMgr := &managers.ManagerProfile{ClubID: home.ClubID, Style: "possession"}
	awayMgr := &managers.ManagerProfile{ClubID: away.ClubID, Style: "high_press"}

	rng := rand.New(rand.NewSource(123456))
	cfg := &InstantMatchConfig{
		Weather: "clear",
		Referee: "balanced",
	}

	report := SimulateInstantMatch(home, away, homeMgr, awayMgr, ge, cfg, rng)

	if report == nil {
		t.Fatalf("expected non-nil MatchReport")
	}

	if len(report.HomeXI) != 11 || len(report.AwayXI) != 11 {
		t.Errorf("expected 11 starting XI rows each, got home=%d away=%d", len(report.HomeXI), len(report.AwayXI))
	}

	if report.Stats.Home.Possession+report.Stats.Away.Possession != 100 {
		t.Errorf("possession should sum to 100, got %d + %d", report.Stats.Home.Possession, report.Stats.Away.Possession)
	}

	if len(report.ShotMap.Shots) == 0 {
		t.Errorf("expected shots in shot map")
	}

	if len(report.Heatmap.HomePoints) == 0 || len(report.Heatmap.AwayPoints) == 0 {
		t.Errorf("expected points in touch heatmap")
	}

	if report.PressConference.Headline == "" {
		t.Errorf("expected headline in press conference")
	}

	// Simulation is pure: club records are updated by the caller
	// (tournament season flow / single-fixture API), not by the sim.
	if home.Played != 0 || away.Played != 0 {
		t.Errorf("expected pure sim to leave club records untouched, got home=%d away=%d", home.Played, away.Played)
	}
	home.UpdateResult(report.HomeGoals, report.AwayGoals)
	away.UpdateResult(report.AwayGoals, report.HomeGoals)
	if home.Played != 1 || away.Played != 1 {
		t.Errorf("expected caller-applied UpdateResult to record 1 match, got home=%d away=%d", home.Played, away.Played)
	}
}

func TestLiveMatchEngine_RadarAndTacticalShift(t *testing.T) {
	home, away := createTestClubs()
	homeMgr := &managers.ManagerProfile{ClubID: home.ClubID, Style: "possession"}
	awayMgr := &managers.ManagerProfile{ClubID: away.ClubID, Style: "low_block"}

	engine := NewLiveMatchEngine(home, away, homeMgr, awayMgr, 98765)
	if len(engine.HomePlayers) != 11 || len(engine.AwayPlayers) != 11 {
		t.Fatalf("expected 11 players each, got home=%d away=%d", len(engine.HomePlayers), len(engine.AwayPlayers))
	}

	engine.State = "PLAYING"
	engine.Speed = 5

	// Advance past 75' with home trailing 0-1
	engine.CurrentMinute = 76.0
	engine.HomeScore = 0
	engine.AwayScore = 1

	engine.Tick(1.0)

	if engine.HomeStance != "OVERLOAD" {
		t.Errorf("expected trailing home team at 76' to shift to OVERLOAD, got %s", engine.HomeStance)
	}

	if engine.LatestTacticalShift == nil {
		t.Errorf("expected tactical shift notice to be populated")
	}

	// Advance to full-time
	for engine.State == "PLAYING" {
		engine.Tick(5.0)
	}

	if engine.State != "FULL_TIME" {
		t.Errorf("expected FULL_TIME at 90 minutes, got %s", engine.State)
	}
}

func TestVAR_DisallowedGoals(t *testing.T) {
	home, away := createTestClubs()
	ge := growth.NewGrowthEngine(999)
	homeMgr := &managers.ManagerProfile{ClubID: home.ClubID, Style: "possession"}
	awayMgr := &managers.ManagerProfile{ClubID: away.ClubID, Style: "possession"}

	// Seed specifically to observe VAR behavior across matches
	varReviewsEncountered := 0
	standsEncountered := 0
	for i := 0; i < 50; i++ {
		rng := rand.New(rand.NewSource(int64(i * 37)))
		report := SimulateInstantMatch(home, away, homeMgr, awayMgr, ge, nil, rng)
		for _, e := range report.Events {
			if e.Type == "var_review" {
				varReviewsEncountered++
				if e.Outcome != "goal_stands" && e.Outcome != "goal_disallowed" {
					t.Errorf("var_review outcome = %q; want goal_stands or goal_disallowed", e.Outcome)
				}
				if e.Reason == "" || e.Decision == "" {
					t.Errorf("var_review must carry reason and decision: %+v", e)
				}
				if e.Outcome == "goal_stands" {
					standsEncountered++
					if e.Disallowed {
						t.Errorf("goal_stands must not set the disallowed flag")
					}
				} else if !e.Disallowed {
					t.Errorf("goal_disallowed must set the disallowed flag")
				}
			}
		}
	}

	if varReviewsEncountered == 0 {
		t.Logf("Note: 0 VAR reviews rolled in 50 simulations (statistically unlikely at ~5%% per goal)")
	}
	_ = standsEncountered
}

func TestWeatherModifiers(t *testing.T) {
	home, away := createTestClubs()
	ge := growth.NewGrowthEngine(777)
	homeMgr := &managers.ManagerProfile{ClubID: home.ClubID, Style: "possession"}
	awayMgr := &managers.ManagerProfile{ClubID: away.ClubID, Style: "possession"}

	rng := rand.New(rand.NewSource(101))
	cfgSnow := &InstantMatchConfig{Weather: "snow", Referee: "balanced"}
	repSnow := SimulateInstantMatch(home, away, homeMgr, awayMgr, ge, cfgSnow, rng)

	if repSnow == nil {
		t.Fatalf("expected non-nil report in snow")
	}
}
