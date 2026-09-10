package matchreport

import (
	"math/rand"
	"testing"

	"football_sim/pkg/models"
)

func TestRateXI(t *testing.T) {
	xi := []*models.Player{
		{PlayerID: "P1", FullName: "Top Scorer", Position: "ST", OVR: 85, Category: "FWD"},
		{PlayerID: "P2", FullName: "Midfield Maestro", Position: "CM", OVR: 84, Category: "MID"},
		{PlayerID: "P3", FullName: "Solid Defender", Position: "CB", OVR: 82, Category: "DEF"},
		{PlayerID: "P4", FullName: "Shot Stopper", Position: "GK", OVR: 86, Category: "GK"},
	}

	scorerMini := ToMiniPlayer(xi[0])
	assistMini := ToMiniPlayer(xi[1])
	bookedMini := ToMiniPlayer(xi[2])

	events := []MatchEventItem{
		{
			Minute:   24,
			Type:     "goal",
			Side:     "home",
			Scorer:   &scorerMini,
			Assister: &assistMini,
			Display:  "Top Scorer 24' (Assist: Midfield Maestro)",
		},
		{
			Minute:  65,
			Type:    "yellow",
			Side:    "home",
			Player:  &bookedMini,
			Display: "Yellow Card: Solid Defender 65'",
		},
	}

	rng := rand.New(rand.NewSource(42))
	rows := RateXI(xi, events, "home", 0, true, true, rng)

	if len(rows) != len(xi) {
		t.Fatalf("expected %d rows, got %d", len(xi), len(rows))
	}

	// Scorer rating should be high (>= 7.5)
	if rows[0].Rating == nil || *rows[0].Rating < 7.5 {
		t.Errorf("scorer rating expected >= 7.5, got %v", rows[0].Rating)
	}
	if rows[0].MatchGoals != 1 {
		t.Errorf("expected 1 match goal, got %d", rows[0].MatchGoals)
	}

	// Defender with yellow and clean sheet
	if rows[2].Card == nil || *rows[2].Card != "yellow" {
		t.Errorf("expected yellow card on defender, got %v", rows[2].Card)
	}

	// Clean sheet bonus for GK & DEF with 0 conceded
	if rows[3].Rating == nil || *rows[3].Rating < 6.5 {
		t.Errorf("clean sheet GK rating expected >= 6.5, got %v", rows[3].Rating)
	}
}

func TestGenerateShotMap(t *testing.T) {
	homeClub := &models.Club{ClubID: "BAR", ShortName: "BAR", OverallTeamRating: 85}
	awayClub := &models.Club{ClubID: "RMA", ShortName: "RMA", OverallTeamRating: 86}

	p := &models.Player{PlayerID: "WK1", FullName: "Venjamin Valerio", Position: "ST", OVR: 80, UniverseWonderkid: true}
	mini := ToMiniPlayer(p)

	events := []MatchEventItem{
		{
			Minute:  33,
			Type:    "goal",
			Side:    "home",
			Scorer:  &mini,
			Display: "Venjamin Valerio 33'",
		},
	}

	rng := rand.New(rand.NewSource(100))
	shotMap := GenerateShotMap(homeClub, awayClub, events, 12, 10, 5, 4, rng, nil)

	if len(shotMap.Shots) != 22 {
		t.Fatalf("expected 22 total shots (12 + 10), got %d", len(shotMap.Shots))
	}

	if shotMap.TotalHomeXG <= 0.0 || shotMap.TotalAwayXG <= 0.0 {
		t.Errorf("expected positive xG totals, got home=%f away=%f", shotMap.TotalHomeXG, shotMap.TotalAwayXG)
	}

	// Check wonderkid goal tag
	foundWkGoal := false
	for _, s := range shotMap.Shots {
		if s.Outcome == "goal" && s.IsWonderkid && s.Shooter.FullName == "Venjamin Valerio" {
			foundWkGoal = true
			break
		}
	}
	if !foundWkGoal {
		t.Errorf("expected to find wonderkid goal in shot map")
	}
}

func TestGenerateTouchHeatmap(t *testing.T) {
	homeClub := &models.Club{ClubID: "BAR", ShortName: "BAR"}
	awayClub := &models.Club{ClubID: "RMA", ShortName: "RMA"}

	rng := rand.New(rand.NewSource(200))
	hm := GenerateTouchHeatmap(homeClub, awayClub, 60, rng, nil)

	if len(hm.HomePoints) == 0 || len(hm.AwayPoints) == 0 {
		t.Fatalf("expected points in heatmap, got home=%d away=%d", len(hm.HomePoints), len(hm.AwayPoints))
	}

	totalHomeZone := hm.HomeZones.Defensive + hm.HomeZones.Midfield + hm.HomeZones.Attacking
	if totalHomeZone < 95 || totalHomeZone > 105 {
		t.Errorf("expected home zones sum to ~100, got %d", totalHomeZone)
	}
}

func TestGeneratePressConference(t *testing.T) {
	homeClub := &models.Club{ClubID: "BAR", ClubName: "FC Barcelona", ShortName: "BAR", HomeStadium: "Camp Nou"}
	awayClub := &models.Club{ClubID: "RMA", ClubName: "Real Madrid", ShortName: "RMA", HomeStadium: "Santiago Bernabéu"}

	pcWin := GeneratePressConference(homeClub, awayClub, 2, 0, nil, "Hansi Flick", "Carlo Ancelotti")
	if pcWin.Headline == "" || pcWin.HomeQuote == "" || pcWin.AwayQuote == "" {
		t.Errorf("empty press conference fields for home win: %+v", pcWin)
	}
	if pcWin.HomeManager != "Hansi Flick" || pcWin.AwayManager != "Carlo Ancelotti" {
		t.Errorf("manager names not carried: %+v", pcWin)
	}

	pcDraw := GeneratePressConference(homeClub, awayClub, 1, 1, nil, "", "")
	if pcDraw.Headline == "" || pcDraw.HomeQuote == "" {
		t.Errorf("empty press conference fields for draw: %+v", pcDraw)
	}
}
