package matchreport

import (
	"testing"

	"football_sim/pkg/models"
)

func TestPickPenaltyTakerSkipsRedCardedAndSubbedPlayers(t *testing.T) {
	primary := &models.Player{PlayerID: "P_PEN", FullName: "Primary", Position: "ST", Category: "FWD", OVR: 92, Composure: 88}
	second := &models.Player{PlayerID: "P_SEC", FullName: "Second", Position: "CAM", Category: "MID", OVR: 84, Composure: 80}
	third := &models.Player{PlayerID: "P_THD", FullName: "Third", Position: "CM", Category: "MID", OVR: 78, Composure: 70}
	gk := &models.Player{PlayerID: "P_GK", FullName: "Keeper", Position: "GK", Category: "GK", OVR: 86, Composure: 90}

	designated := DesignatedPenaltyTaker([]*models.Player{primary, second, third, gk}, nil)
	if designated == nil || designated.PlayerID != primary.PlayerID {
		t.Fatalf("designated=%v want primary", designated)
	}

	got := PickPenaltyTaker([]*models.Player{second, third, gk}, nil, primary.PlayerID)
	if got == nil || got.PlayerID != second.PlayerID {
		t.Fatalf("after red/sub got=%v want second", got)
	}

	ten := PickPenaltyTaker([]*models.Player{third, gk}, nil, primary.PlayerID)
	if ten == nil || ten.PlayerID != third.PlayerID {
		t.Fatalf("10-player side got=%v want third", ten)
	}

	nine := PickPenaltyTaker([]*models.Player{gk}, nil, primary.PlayerID)
	if nine == nil || nine.PlayerID != gk.PlayerID {
		t.Fatalf("9-player side must still pick the remaining keeper, got=%v", nine)
	}
}

func TestPickPenaltyTakerTieBreaksOnPlayerID(t *testing.T) {
	a := &models.Player{PlayerID: "B_ID", FullName: "B", Position: "ST", Category: "FWD", OVR: 80, Composure: 80}
	b := &models.Player{PlayerID: "A_ID", FullName: "A", Position: "ST", Category: "FWD", OVR: 80, Composure: 80}
	got := PickPenaltyTaker([]*models.Player{a, b}, nil, "")
	if got == nil || got.PlayerID != "A_ID" {
		t.Fatalf("tie-break got=%v want A_ID", got)
	}
}
