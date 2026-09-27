package matchengine

import (
	"testing"

	"football_sim/pkg/models"
)

func TestTacticalPositionAliasesUseDistinctSlots(t *testing.T) {
	for _, pos := range []string{"GK", "LB", "LWB", "LCB", "CB", "RCB", "RB", "RWB", "CDM", "LDM", "RDM", "LM", "LCM", "CM", "RCM", "RM", "LAM", "CAM", "RAM", "LW", "LF", "LST", "ST", "RST", "CF", "RF", "RW"} {
		if _, ok := positionHomeCoords[pos]; !ok {
			t.Errorf("home tactical slot missing %s", pos)
		}
		if _, ok := positionAwayCoords[pos]; !ok {
			t.Errorf("away tactical slot missing %s", pos)
		}
	}
	if positionHomeCoords["LCB"][1] >= positionHomeCoords["RCB"][1] {
		t.Fatal("LCB/RCB are not separated left-to-right")
	}
	if positionHomeCoords["LCM"][1] >= positionHomeCoords["RCM"][1] {
		t.Fatal("LCM/RCM are not separated left-to-right")
	}
	if positionHomeCoords["CAM"][0] <= positionHomeCoords["CM"][0] {
		t.Fatal("CAM is not advanced of CM")
	}
	if positionHomeCoords["CAM"] == positionAwayCoords["CAM"] {
		t.Fatal("opposing CAMs overlap on the halfway line")
	}
	if positionHomeCoords["RW"][1] <= 0.5 {
		t.Fatal("RW is not on the right attacking channel")
	}
	if positionHomeCoords["LW"][1] >= 0.5 {
		t.Fatal("LW is not on the left attacking channel")
	}
}

func TestRadarPlayersUseAssignedSlotWithoutOverwritingNaturalPosition(t *testing.T) {
	cantalejo := &models.Player{
		PlayerID: "WK_Maverick_Cantalejo", FullName: "Maverick Cantalejo",
		Position: "CAM", Category: "FWD", OVR: 77,
	}
	assignments := []models.StartingSlot{{
		Slot: "CAM", NaturalPosition: "CAM", PositionFit: models.PositionFitNatural, Player: cantalejo,
	}}
	radar := radarPlayersSlotted(assignments, positionHomeCoords)
	if len(radar) != 1 {
		t.Fatalf("radar players = %d, want 1", len(radar))
	}
	got := radar[0]
	want := positionHomeCoords["CAM"]
	if got.Position != "CAM" || got.NaturalPosition != "CAM" || got.TacticalSlot != "CAM" || got.PositionFit != string(models.PositionFitNatural) {
		t.Fatalf("Cantalejo tactical metadata = %+v", got)
	}
	if got.X != want[0] || got.Y != want[1] {
		t.Fatalf("Cantalejo coordinates = (%.2f, %.2f), want CAM (%.2f, %.2f)", got.X, got.Y, want[0], want[1])
	}
}

func TestHomeAndAwayRadarPreserveTeamRelativeLeftRightSemantics(t *testing.T) {
	lw := &models.Player{PlayerID: "LW", FullName: "Left Winger", Position: "LW", Category: "FWD", OVR: 80}
	rw := &models.Player{PlayerID: "RW", FullName: "Right Winger", Position: "RW", Category: "FWD", OVR: 80}
	assignments := []models.StartingSlot{{Slot: "LW", Player: lw}, {Slot: "RW", Player: rw}}

	home := radarPlayersSlotted(assignments, positionHomeCoords)
	away := radarPlayersSlotted(assignments, positionAwayCoords)
	if home[0].TacticalSlot != "LW" || home[1].TacticalSlot != "RW" || away[0].TacticalSlot != "LW" || away[1].TacticalSlot != "RW" {
		t.Fatalf("slot semantics changed across orientation: home=%+v away=%+v", home, away)
	}
	if !(home[0].Y < home[1].Y) {
		t.Fatalf("home LW/RW were not left/right: LW %.2f, RW %.2f", home[0].Y, home[1].Y)
	}
	if !(away[0].Y > away[1].Y) {
		t.Fatalf("away 180-degree view did not preserve team-relative left/right: LW %.2f, RW %.2f", away[0].Y, away[1].Y)
	}
	if away[0].X != positionAwayCoords["LW"][0] || away[0].Y != positionAwayCoords["LW"][1] ||
		away[1].X != positionAwayCoords["RW"][0] || away[1].Y != positionAwayCoords["RW"][1] {
		t.Fatal("away coordinates were not derived from their explicit slots")
	}
}

func TestSubstitutionInheritsExistingTacticalSlot(t *testing.T) {
	starter := &models.Player{PlayerID: "LW", FullName: "Left Winger", Position: "LW", Category: "FWD", OVR: 80}
	replacement := &models.Player{PlayerID: "CAM", FullName: "Emergency Ten", Position: "CAM", Category: "FWD", OVR: 84}
	engine := &LiveMatchEngine{
		HomeSlots: []models.StartingSlot{{Slot: "LW", Player: starter}},
		HomePlayers: radarPlayersSlotted(
			[]models.StartingSlot{{Slot: "LW", Player: starter}},
			positionHomeCoords,
		),
	}
	want := positionHomeCoords["LW"]
	engine.syncRadarActor("home", starter.PlayerID, replacement)
	if got := engine.HomePlayers[0]; got.PlayerID != replacement.PlayerID || got.TacticalSlot != "LW" || got.NaturalPosition != "CAM" ||
		got.PositionFit != string(models.PositionFitEmergency) || got.X != want[0] || got.Y != want[1] {
		t.Fatalf("substitution did not inherit the LW role: %+v", got)
	}
	if engine.HomeSlots[0].PlayerID != replacement.PlayerID || engine.HomeSlots[0].Slot != "LW" {
		t.Fatalf("current slot assignment was not updated: %+v", engine.HomeSlots[0])
	}
}

func TestDuplicateGenericCentralPositionsDoNotOverlap(t *testing.T) {
	players := []LivePlayerRadar{
		{PlayerID: "CB1", Position: "CB", X: positionHomeCoords["CB"][0], Y: positionHomeCoords["CB"][1]},
		{PlayerID: "CB2", Position: "CB", X: positionHomeCoords["CB"][0], Y: positionHomeCoords["CB"][1]},
		{PlayerID: "CM1", Position: "CM", X: positionHomeCoords["CM"][0], Y: positionHomeCoords["CM"][1]},
		{PlayerID: "CM2", Position: "CM", X: positionHomeCoords["CM"][0], Y: positionHomeCoords["CM"][1]},
	}
	staggerDuplicates(players)
	if players[0].X != players[1].X || players[0].Y == players[1].Y {
		t.Fatal("two generic CBs were not separated into central slots")
	}
	if players[2].X != players[3].X || players[2].Y == players[3].Y {
		t.Fatal("two generic CMs were not separated into central slots")
	}
}
