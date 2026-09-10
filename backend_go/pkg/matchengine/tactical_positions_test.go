package matchengine

import "testing"

func TestTacticalPositionAliasesUseDistinctSlots(t *testing.T) {
	for _, pos := range []string{"GK","LB","LWB","LCB","CB","RCB","RB","RWB","CDM","LDM","RDM","LM","LCM","CM","RCM","RM","LAM","CAM","RAM","LW","LF","ST","CF","RF","RW"} {
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
	if positionHomeCoords["RW"][1] <= 0.5 {
		t.Fatal("RW is not on the right attacking channel")
	}
	if positionHomeCoords["LW"][1] >= 0.5 {
		t.Fatal("LW is not on the left attacking channel")
	}
}

func TestDuplicateGenericCentralPositionsDoNotOverlap(t *testing.T) {
	players := []LivePlayerRadar{
		{PlayerID:"CB1",Position:"CB",X:positionHomeCoords["CB"][0],Y:positionHomeCoords["CB"][1]},
		{PlayerID:"CB2",Position:"CB",X:positionHomeCoords["CB"][0],Y:positionHomeCoords["CB"][1]},
		{PlayerID:"CM1",Position:"CM",X:positionHomeCoords["CM"][0],Y:positionHomeCoords["CM"][1]},
		{PlayerID:"CM2",Position:"CM",X:positionHomeCoords["CM"][0],Y:positionHomeCoords["CM"][1]},
	}
	staggerDuplicates(players)
	if players[0].X != players[1].X || players[0].Y == players[1].Y {
		t.Fatal("two generic CBs were not separated into central slots")
	}
	if players[2].X != players[3].X || players[2].Y == players[3].Y {
		t.Fatal("two generic CMs were not separated into central slots")
	}
}
