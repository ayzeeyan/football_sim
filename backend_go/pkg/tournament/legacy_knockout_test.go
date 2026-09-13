package tournament

import (
	"testing"

	"football_sim/pkg/models"
)

// Legacy two-legged ties swap venues on the return leg, so leg-2 penalties
// belong to the tie's away side first: Penalties[0] is the leg-2 home score.
// Awarding [0] > [1] to tie.HomeID flips shootout winners.
func TestResolveTwoLeggedShootoutGoesToLegTwoWinner(t *testing.T) {
	tm := &TournamentManager{
		UCLFixtures:      []Fixture{},
		UCLQuarterFinals: map[string]CupTie{},
		UCLSemiFinals:    map[string]CupTie{},
		MaxMatchweeks:    44,
	}
	a := &models.Club{ClubID: "AAA", ClubName: "Alpha", ShortName: "ALP"}
	b := &models.Club{ClubID: "BBB", ClubName: "Beta", ShortName: "BET"}
	tm.UCLQuarterFinals["qf_1"] = CupTie{HomeID: "AAA", AwayID: "BBB"}
	tm.makeUCLLeg("qf_1", "Quarter-final", 1, 35, a, b)
	tm.makeUCLLeg("qf_1", "Quarter-final", 2, 36, b, a)
	l1h, l1a := 1, 0
	l2h, l2a := 1, 0
	for i := range tm.UCLFixtures {
		f := &tm.UCLFixtures[i]
		f.Status = "finished"
		if f.Leg == 1 {
			f.HomeGoals, f.AwayGoals = &l1h, &l1a
		} else {
			f.HomeGoals, f.AwayGoals = &l2h, &l2a
			f.DecidedBy = "penalties"
			f.Penalties = []int{4, 2} // leg-2 home side (BBB) wins the shootout
		}
	}
	tm.syncUCLBracket()
	if got := tm.UCLQuarterFinals["qf_1"].WinnerID; got != "BBB" {
		t.Fatalf("shootout winner=%q want BBB (leg-2 home side)", got)
	}
	// Aggregate still decides when it is not level: AAA 2-0, 0-1 -> AAA.
	l1h, l1a, l2h, l2a = 2, 0, 0, 1
	for i := range tm.UCLFixtures {
		f := &tm.UCLFixtures[i]
		if f.Leg == 1 {
			f.HomeGoals, f.AwayGoals = &l1h, &l1a
		} else {
			f.HomeGoals, f.AwayGoals = &l2h, &l2a
			f.DecidedBy = ""
			f.Penalties = nil
		}
	}
	tie := tm.UCLQuarterFinals["qf_1"]
	tie.WinnerID = ""
	tm.UCLQuarterFinals["qf_1"] = tie
	tm.syncUCLBracket()
	if got := tm.UCLQuarterFinals["qf_1"].WinnerID; got != "AAA" {
		t.Fatalf("aggregate winner=%q want AAA", got)
	}
}
