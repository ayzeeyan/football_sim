package tournament

import (
	"strings"
	"testing"

	"football_sim/pkg/models"
)

func TestLeg2FirstLegLabel(t *testing.T) {
	hg, ag := 2, 0
	tm := &TournamentManager{
		Clubs: map[string]*models.Club{
			"H": {ClubID: "H", ClubName: "Home FC", ShortName: "HOM"},
			"A": {ClubID: "A", ClubName: "Away FC", ShortName: "AWY"},
		},
		UCLFixtures: []Fixture{
			{FixtureID: "L1", Competition: "ucl", Stage: "Quarter-final", Leg: 1, TieID: "T1", HomeID: "H", AwayID: "A", Status: "finished", HomeGoals: &hg, AwayGoals: &ag},
			{FixtureID: "L2", Competition: "ucl", Stage: "Quarter-final", Leg: 2, TieID: "T1", HomeID: "A", AwayID: "H", Status: "scheduled"},
		},
	}
	leg2 := tm.UCLFixtures[1]
	label, _ := tm.leg2FirstLegLabelUnlocked(&leg2).(string)
	if !strings.Contains(label, "1st leg:") || !strings.Contains(label, "2–0") || !strings.Contains(label, "HOM") {
		t.Fatalf("leg-2 card needs the first-leg score, got %q", label)
	}
	// Super Cup legs never get the label.
	sc := Fixture{FixtureID: "SC", Competition: "super-cup", Stage: "Final", Leg: 1, Status: "scheduled"}
	if tm.leg2FirstLegLabelUnlocked(&sc) != nil {
		t.Fatal("super-cup cards must stay unchanged")
	}
	// Unplayed first leg: no label.
	tm.UCLFixtures[0].Status = "scheduled"
	if tm.leg2FirstLegLabelUnlocked(&leg2) != nil {
		t.Fatal("no first-leg label before leg 1 is decided")
	}
}
