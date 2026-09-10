package models

import (
	"testing"
)

func TestCompetitionRecord(t *testing.T) {
	rec := &CompetitionRecord{}

	// Win 3-1
	rec.UpdateResult(3, 1)
	if rec.Played != 1 || rec.Won != 1 || rec.Points != 3 || rec.GoalsFor != 3 || rec.GoalsAgainst != 1 || rec.GoalDifference != 2 {
		t.Errorf("CompetitionRecord win incorrect: %+v", rec)
	}
	if len(rec.Form) != 1 || rec.Form[0] != "W" {
		t.Errorf("CompetitionRecord form: %v", rec.Form)
	}

	// Draw 2-2
	rec.UpdateResult(2, 2)
	if rec.Played != 2 || rec.Drawn != 1 || rec.Points != 4 || rec.GoalDifference != 2 {
		t.Errorf("CompetitionRecord draw incorrect: %+v", rec)
	}

	// Loss 0-1
	rec.UpdateResult(0, 1)
	if rec.Played != 3 || rec.Lost != 1 || rec.Points != 4 || rec.GoalDifference != 1 {
		t.Errorf("CompetitionRecord loss incorrect: %+v", rec)
	}

	// Reset
	rec.Reset()
	if rec.Played != 0 || rec.Points != 0 || len(rec.Form) != 0 {
		t.Errorf("CompetitionRecord reset failed: %+v", rec)
	}
}

func TestStandingsSortingTiebreakers(t *testing.T) {
	rows := StandingsTable{
		// Identical stats to test name sort
		{ClubID: "B_TEAM", ClubName: "Beta FC", Points: 10, GoalDifference: 5, GoalsFor: 12, TeamRating: 80},
		{ClubID: "A_TEAM", ClubName: "Alpha FC", Points: 10, GoalDifference: 5, GoalsFor: 12, TeamRating: 80},

		// Test TeamRating tiebreaker
		{ClubID: "RATING_HIGH", ClubName: "Rating High", Points: 12, GoalDifference: 3, GoalsFor: 10, TeamRating: 85},
		{ClubID: "RATING_LOW", ClubName: "Rating Low", Points: 12, GoalDifference: 3, GoalsFor: 10, TeamRating: 80},

		// Test GF tiebreaker
		{ClubID: "GF_HIGH", ClubName: "GF High", Points: 15, GoalDifference: 4, GoalsFor: 15, TeamRating: 75},
		{ClubID: "GF_LOW", ClubName: "GF Low", Points: 15, GoalDifference: 4, GoalsFor: 10, TeamRating: 75},

		// Test GD tiebreaker
		{ClubID: "GD_HIGH", ClubName: "GD High", Points: 18, GoalDifference: 10, GoalsFor: 15, TeamRating: 75},
		{ClubID: "GD_LOW", ClubName: "GD Low", Points: 18, GoalDifference: 5, GoalsFor: 20, TeamRating: 75},

		// Clear leader
		{ClubID: "LEADER", ClubName: "Leader FC", Points: 25, GoalDifference: 15, GoalsFor: 30, TeamRating: 88},
	}

	SortStandings(rows)

	// Verify order:
	// 1: Leader FC (25 pts)
	// 2: GD High (18 pts, GD 10)
	// 3: GD Low (18 pts, GD 5)
	// 4: GF High (15 pts, GD 4, GF 15)
	// 5: GF Low (15 pts, GD 4, GF 10)
	// 6: Rating High (12 pts, GD 3, GF 10, Rating 85)
	// 7: Rating Low (12 pts, GD 3, GF 10, Rating 80)
	// 8: Alpha FC (10 pts, name Alpha)
	// 9: Beta FC (10 pts, name Beta)

	expectedOrder := []string{
		"LEADER",
		"GD_HIGH",
		"GD_LOW",
		"GF_HIGH",
		"GF_LOW",
		"RATING_HIGH",
		"RATING_LOW",
		"A_TEAM",
		"B_TEAM",
	}

	for i, id := range expectedOrder {
		if rows[i].ClubID != id {
			t.Errorf("Position %d: got %s, want %s", i+1, rows[i].ClubID, id)
		}
		if rows[i].Position != i+1 {
			t.Errorf("Row %d position field not numbered correctly: %d", i+1, rows[i].Position)
		}
	}
}

func TestSortClubs(t *testing.T) {
	c1 := &Club{ClubID: "C1", ClubName: "Club 1", Points: 10, GoalDifference: 2, GoalsFor: 5, OverallTeamRating: 75}
	c2 := &Club{ClubID: "C2", ClubName: "Club 2", Points: 15, GoalDifference: 5, GoalsFor: 10, OverallTeamRating: 80}
	c3 := &Club{ClubID: "C3", ClubName: "Club 3", Points: 10, GoalDifference: 4, GoalsFor: 5, OverallTeamRating: 75}

	clubs := []*Club{c1, c2, c3}
	SortClubs(clubs)

	if clubs[0].ClubID != "C2" || clubs[1].ClubID != "C3" || clubs[2].ClubID != "C1" {
		t.Errorf("SortClubs produced unexpected order: %s, %s, %s", clubs[0].ClubID, clubs[1].ClubID, clubs[2].ClubID)
	}
}
