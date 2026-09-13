package tournament

import (
	"testing"

	"football_sim/pkg/models"
)

// TestCoefficientPointsAccumulate checks the exact points rule on a synthetic
// European campaign: 2 per league-phase win, 1 per draw, stage bonuses per
// knockout tie won, plus 4 for the champion. Domestic results never pay.
func TestCoefficientPointsAccumulate(t *testing.T) {
	a := &models.Club{ClubID: "A", ClubName: "Alpha", ShortName: "ALP", League: "Premier League"}
	b := &models.Club{ClubID: "B", ClubName: "Beta", ShortName: "BET", League: "La Liga"}
	c := &models.Club{ClubID: "C", ClubName: "Gamma", ShortName: "GAM", League: "Serie A"}
	d := &models.Club{ClubID: "D", ClubName: "Delta", ShortName: "DEL", League: "La Liga"}
	tm := &TournamentManager{
		Clubs:     map[string]*models.Club{"A": a, "B": b, "C": c, "D": d},
		ClubsList: []*models.Club{a, b, c, d},
		World: &EuropeanWorld{Competitions: map[string]*Competition{
			"champions-league": {
				ID: "champions-league", Name: "UEFA Champions League", Kind: CompetitionEuropean,
				ParticipantIDs: []string{"A", "B", "C", "D"},
				Records: map[string]*models.CompetitionRecord{
					"A": {Won: 5, Drawn: 1},
					"B": {Won: 2, Drawn: 2},
					"C": {Won: 0, Drawn: 0},
					"D": {Won: 0, Drawn: 0},
				},
				Rounds: []KnockoutRound{
					{Stage: "Semi-final", EntrantIDs: []string{"A", "B", "C", "D"}, FixtureIDs: []string{"SF1", "SF2"}, TieIDs: []string{"T1", "T2"}, WinnerIDs: []string{"A", "C"}},
					{Stage: "Final", EntrantIDs: []string{"A", "C"}, FixtureIDs: []string{"F"}, TieIDs: []string{"TF"}, WinnerIDs: []string{"A"}},
				},
				ChampionID: "A",
			},
		}, CompetitionOrder: []string{"champions-league"}},
	}
	tm.applyEuropeanCoefficientPointsUnlocked()
	// A: 5*2+1 + 6 (SF) + 8 (F) + 4 (champion) = 29.
	if a.Coefficient != 29 {
		t.Fatalf("A coefficient=%d want 29", a.Coefficient)
	}
	// B: 2*2+2 = 6, no knockout wins.
	if b.Coefficient != 6 {
		t.Fatalf("B coefficient=%d want 6", b.Coefficient)
	}
	// C: 0 + 6 (SF) = 6.
	if c.Coefficient != 6 {
		t.Fatalf("C coefficient=%d want 6", c.Coefficient)
	}
	if d.Coefficient != 0 {
		t.Fatalf("D coefficient=%d want 0", d.Coefficient)
	}
}

func TestRankClubsForSwissPrefersCoefficient(t *testing.T) {
	low := &models.Club{ClubID: "LOW", OverallTeamRating: 90, Identity: models.ClubIdentity{Reputation: 90}, Coefficient: 0}
	mid := &models.Club{ClubID: "MID", OverallTeamRating: 80, Identity: models.ClubIdentity{Reputation: 80}, Coefficient: 25}
	ordered := rankClubsForSwiss([]*models.Club{low, mid})
	if ordered[0].ClubID != "MID" {
		t.Fatalf("earned coefficient should outrank rating: %s first", ordered[0].ClubID)
	}
	// Fresh worlds (all zero) match the legacy rating order exactly.
	low.Coefficient, mid.Coefficient = 0, 0
	legacy := rankClubsForOpening([]*models.Club{low, mid})
	fresh := rankClubsForSwiss([]*models.Club{low, mid})
	for i := range legacy {
		if legacy[i].ClubID != fresh[i].ClubID {
			t.Fatalf("fresh Swiss order diverges from rating order at %d", i)
		}
	}
}
