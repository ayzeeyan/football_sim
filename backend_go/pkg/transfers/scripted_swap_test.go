package transfers

import (
	"testing"

	"football_sim/pkg/models"
)

func TestSummerWindowDoesNotForcePlayerTransfers(t *testing.T) {
	libatan := &models.Player{PlayerID: "WK_Reid_Randell_Libatan", FullName: "Reid Randell Libatan", Position: "LW", Category: "FWD", OVR: 76, Age: 17, ContractYears: 3, MarketValueEUR: 40_000_000, ClubID: "BUN-BAY"}
	cliergy := &models.Player{PlayerID: "WK_Cliergy_Jave_Lanticse", FullName: "Cliergy Jave Lanticse", Position: "CM", Category: "MID", OVR: 76, Age: 17, ContractYears: 3, MarketValueEUR: 40_000_000, ClubID: "LAL-BAR"}
	clubs := []*models.Club{
		{ClubID: "BUN-BAY", ClubName: "Bayern", ShortName: "BAY", League: "Bundesliga", Squad: []*models.Player{libatan}},
		{ClubID: "LAL-BAR", ClubName: "Barcelona", ShortName: "BAR", League: "La Liga", Squad: []*models.Player{cliergy}},
	}
	te := NewTransferEngine(clubs, nil, 42)
	te.BeginOffSeasonWindow()

	if libatan.ClubID != "BUN-BAY" || cliergy.ClubID != "LAL-BAR" {
		t.Fatalf("summer window moved players without negotiations: Libatan=%s Cliergy=%s", libatan.ClubID, cliergy.ClubID)
	}
	if len(te.AllTimeTransfers) != 0 {
		t.Fatalf("summer window recorded unnegotiated transfers: %+v", te.AllTimeTransfers)
	}
	if te.ScriptedSwapDone {
		t.Fatal("legacy swap flag must remain inert")
	}
}
