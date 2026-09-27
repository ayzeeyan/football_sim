package models

import (
	"reflect"
	"sort"
	"testing"
)

func tacticalTestPlayer(id, position string, ovr int) *Player {
	return &Player{
		PlayerID: id, FullName: id, Position: position,
		Category: GetPositionCategory(position), OVR: ovr,
		Fitness: 80, Sharpness: 70, Morale: 70,
	}
}

func assignmentsBySlot(assignments []StartingSlot) map[string]*Player {
	result := make(map[string]*Player, len(assignments))
	for _, assignment := range assignments {
		result[assignment.Slot] = assignment.Player
	}
	return result
}

func TestFormationDefinitionsOwnUniqueTacticalSlots(t *testing.T) {
	want := map[string][]string{
		Formation433:       {"GK", "LB", "LCB", "RCB", "RB", "LCM", "CM", "RCM", "LW", "ST", "RW"},
		Formation433Attack: {"GK", "LB", "LCB", "RCB", "RB", "LCM", "RCM", "CAM", "LW", "ST", "RW"},
		Formation4231:      {"GK", "LB", "LCB", "RCB", "RB", "LDM", "RDM", "LW", "CAM", "RW", "ST"},
		Formation442:       {"GK", "LB", "LCB", "RCB", "RB", "LM", "LCM", "RCM", "RM", "LST", "RST"},
	}
	for formation, slots := range want {
		got := FormationSlots(formation)
		if !reflect.DeepEqual(got, slots) {
			t.Fatalf("%s slots = %v, want %v", formation, got, slots)
		}
		seen := map[string]bool{}
		for _, slot := range got {
			if seen[slot] {
				t.Fatalf("%s repeats slot %s", formation, slot)
			}
			seen[slot] = true
		}
	}
}

func TestCantalejoCAMAssignmentIn4231(t *testing.T) {
	cantalejo := tacticalTestPlayer("WK_Maverick_Cantalejo", "CAM", 77)
	players := []*Player{
		tacticalTestPlayer("GK", "GK", 80), tacticalTestPlayer("LB", "LB", 80),
		tacticalTestPlayer("CB1", "CB", 80), tacticalTestPlayer("CB2", "CB", 79), tacticalTestPlayer("RB", "RB", 80),
		tacticalTestPlayer("DM1", "CDM", 80), tacticalTestPlayer("DM2", "CDM", 79),
		tacticalTestPlayer("LW", "LW", 80), cantalejo, tacticalTestPlayer("RW", "RW", 80), tacticalTestPlayer("ST", "ST", 80),
	}
	club := &Club{ClubID: "RMA", Squad: players}
	bySlot := assignmentsBySlot(club.GetStartingElevenSlotsForFormation(Formation4231, "free_flowing", "youth", "la-liga:2"))
	if bySlot["CAM"] == nil || bySlot["CAM"].PlayerID != cantalejo.PlayerID {
		t.Fatalf("CAM assignment = %+v, want Cantalejo", bySlot["CAM"])
	}
}

func TestNaturalWingerBeatsSlightlyHigherCAMAtLW(t *testing.T) {
	players := []*Player{
		tacticalTestPlayer("GK", "GK", 80), tacticalTestPlayer("LB", "LB", 80), tacticalTestPlayer("CB1", "CB", 80),
		tacticalTestPlayer("CB2", "CB", 79), tacticalTestPlayer("RB", "RB", 80), tacticalTestPlayer("DM1", "CDM", 80),
		tacticalTestPlayer("DM2", "CDM", 79), tacticalTestPlayer("LW82", "LW", 82), tacticalTestPlayer("CAM84", "CAM", 84),
		tacticalTestPlayer("RW", "RW", 80), tacticalTestPlayer("ST", "ST", 80),
	}
	bySlot := assignmentsBySlot((&Club{Squad: players}).GetStartingElevenSlotsForFormation(Formation4231, "", ""))
	if bySlot["LW"].PlayerID != "LW82" || bySlot["CAM"].PlayerID != "CAM84" {
		t.Fatalf("fit-first attack = LW:%s CAM:%s", bySlot["LW"].PlayerID, bySlot["CAM"].PlayerID)
	}
}

func TestCentreForwardIsNaturalFitForStrikerSlots(t *testing.T) {
	centreForward := tacticalTestPlayer("CF", "CF", 75)
	for _, slot := range []string{"ST", "LST", "RST"} {
		if got := PositionFitForPlayer(centreForward, slot); got != PositionFitNatural {
			t.Errorf("CF in %s fit = %s; want Natural", slot, got)
		}
	}
}

func TestGenericCentreBacksReceiveLCBAndRCB(t *testing.T) {
	left := tacticalTestPlayer("CB-A", "CB", 80)
	right := tacticalTestPlayer("CB-B", "CB", 80)
	assignments := AssignPlayersToFormation([]*Player{tacticalTestPlayer("GK", "GK", 70), right, left}, Formation433)
	bySlot := assignmentsBySlot(assignments)
	if bySlot["LCB"] == nil || bySlot["RCB"] == nil || bySlot["LCB"] == bySlot["RCB"] {
		t.Fatalf("generic CB assignments = %+v", bySlot)
	}
	if PositionFitForPlayer(bySlot["LCB"], "LCB") != PositionFitNatural || PositionFitForPlayer(bySlot["RCB"], "RCB") != PositionFitNatural {
		t.Fatal("generic centre-backs should be natural fits on both sides")
	}
}

func Test433AttackAssignsTwoCMsAndCAM(t *testing.T) {
	players := []*Player{
		tacticalTestPlayer("GK", "GK", 70),
		tacticalTestPlayer("CM-B", "CM", 80),
		tacticalTestPlayer("CAM", "CAM", 80),
		tacticalTestPlayer("CM-A", "CM", 80),
	}
	bySlot := assignmentsBySlot(AssignPlayersToFormation(players, Formation433Attack))
	if bySlot["CAM"] == nil || bySlot["CAM"].PlayerID != "CAM" || bySlot["LCM"] == nil || bySlot["RCM"] == nil {
		t.Fatalf("4-3-3 Attack midfield = %+v", bySlot)
	}
}

func TestProperCentreBacksAreNotDisplacedBySecondRightBack(t *testing.T) {
	players := []*Player{
		tacticalTestPlayer("GK", "GK", 80), tacticalTestPlayer("LB", "LB", 80),
		tacticalTestPlayer("CB1", "CB", 75), tacticalTestPlayer("CB2", "CB", 74),
		tacticalTestPlayer("RB90", "RB", 90), tacticalTestPlayer("RB89", "RB", 89),
		tacticalTestPlayer("CM1", "CM", 80), tacticalTestPlayer("CM2", "CM", 79), tacticalTestPlayer("CM3", "CM", 78),
		tacticalTestPlayer("LW", "LW", 80), tacticalTestPlayer("ST", "ST", 80), tacticalTestPlayer("RW", "RW", 80),
	}
	bySlot := assignmentsBySlot((&Club{Squad: players}).GetStartingElevenSlotsForFormation(Formation433, "", ""))
	if bySlot["LCB"].Position != "CB" || bySlot["RCB"].Position != "CB" {
		t.Fatalf("centre-back slots displaced by fullback: LCB=%s RCB=%s", bySlot["LCB"].Position, bySlot["RCB"].Position)
	}
}

func TestMissingRWUsesBestCompatibleFallbackDeterministically(t *testing.T) {
	players := []*Player{
		tacticalTestPlayer("GK", "GK", 80), tacticalTestPlayer("LB", "LB", 80), tacticalTestPlayer("CB1", "CB", 80),
		tacticalTestPlayer("CB2", "CB", 79), tacticalTestPlayer("RB", "RB", 80), tacticalTestPlayer("CM1", "CM", 80),
		tacticalTestPlayer("CM2", "CM", 79), tacticalTestPlayer("CM3", "CM", 78), tacticalTestPlayer("LW", "LW", 85),
		tacticalTestPlayer("RM", "RM", 75), tacticalTestPlayer("ST", "ST", 80),
	}
	want := assignmentsBySlot((&Club{Squad: players}).GetStartingElevenSlotsForFormation(Formation433, "", ""))["RW"].PlayerID
	if want != "RM" {
		t.Fatalf("missing-RW fallback = %s, want good-fit RM", want)
	}
	sort.Slice(players, func(i, j int) bool { return players[i].PlayerID > players[j].PlayerID })
	got := assignmentsBySlot((&Club{Squad: players}).GetStartingElevenSlotsForFormation(Formation433, "", ""))["RW"].PlayerID
	if got != want {
		t.Fatalf("squad order changed RW fallback: %s -> %s", want, got)
	}
}

func TestSameSquadAndFormationAlwaysProduceSameSlots(t *testing.T) {
	players := []*Player{
		tacticalTestPlayer("GK", "GK", 80), tacticalTestPlayer("LB", "LB", 80), tacticalTestPlayer("CB1", "CB", 80),
		tacticalTestPlayer("CB2", "CB", 80), tacticalTestPlayer("RB", "RB", 80), tacticalTestPlayer("CM1", "CM", 80),
		tacticalTestPlayer("CM2", "CM", 80), tacticalTestPlayer("CAM", "CAM", 80), tacticalTestPlayer("LW", "LW", 80),
		tacticalTestPlayer("ST", "ST", 80), tacticalTestPlayer("RW", "RW", 80),
	}
	encode := func(assignments []StartingSlot) []string {
		result := make([]string, 0, len(assignments))
		for _, assignment := range assignments {
			result = append(result, assignment.Slot+"="+assignment.PlayerID)
		}
		return result
	}
	first := encode(AssignPlayersToFormation(players, Formation433Attack))
	reversed := append([]*Player(nil), players...)
	for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
		reversed[i], reversed[j] = reversed[j], reversed[i]
	}
	second := encode(AssignPlayersToFormation(reversed, Formation433Attack))
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("assignment is not deterministic: %v vs %v", first, second)
	}
}
