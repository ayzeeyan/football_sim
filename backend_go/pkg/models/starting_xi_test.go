package models

import (
	"testing"
)

func TestGetStartingElevenSlotsAllocationAndUniqueness(t *testing.T) {
	// 1. Full squad with natural positions including 2 RBs and 1 LB (reproducing Barcelona scenario)
	club := &Club{
		ClubID:   "TEST-FC",
		ClubName: "Test FC",
		Squad: []*Player{
			{PlayerID: "GK1", FullName: "Goalie One", Position: "GK", Category: "GK", OVR: 85},
			{PlayerID: "LB1", FullName: "Alejandro Balde", Position: "LB", Category: "DEF", OVR: 81},
			{PlayerID: "CB1", FullName: "Ronald Araujo", Position: "CB", Category: "DEF", OVR: 86},
			{PlayerID: "CB2", FullName: "Andreas Christensen", Position: "CB", Category: "DEF", OVR: 83},
			{PlayerID: "RB1", FullName: "Joao Cancelo", Position: "RB", Category: "DEF", OVR: 85},
			{PlayerID: "RB2", FullName: "Jules Kounde", Position: "RB", Category: "DEF", OVR: 84},
			{PlayerID: "M1", FullName: "Mid One", Position: "LM", Category: "MID", OVR: 84},
			{PlayerID: "M2", FullName: "Mid Two", Position: "CDM", Category: "MID", OVR: 85},
			{PlayerID: "M3", FullName: "Mid Three", Position: "RM", Category: "MID", OVR: 83},
			{PlayerID: "F1", FullName: "Winger Left", Position: "LW", Category: "FWD", OVR: 86},
			{PlayerID: "F2", FullName: "Striker", Position: "ST", Category: "FWD", OVR: 88},
			{PlayerID: "F3", FullName: "Winger Right", Position: "RW", Category: "FWD", OVR: 85},
		},
	}

	slots := club.GetStartingElevenSlots()
	if len(slots) != 11 {
		t.Fatalf("expected 11 starting slots, got %d", len(slots))
	}

	seenPIDs := make(map[string]bool)
	slotMap := make(map[string]*Player)
	for _, s := range slots {
		if s.Player == nil {
			t.Fatalf("slot %s has nil player", s.Slot)
		}
		if seenPIDs[s.Player.PlayerID] {
			t.Fatalf("duplicate player %s in starting XI", s.Player.PlayerID)
		}
		seenPIDs[s.Player.PlayerID] = true
		slotMap[s.Slot] = s.Player
	}

	// Natural LB Balde should be in LB slot
	if slotMap["LB"] == nil || slotMap["LB"].PlayerID != "LB1" {
		t.Errorf("expected LB1 in LB slot, got %v", slotMap["LB"])
	}
	// Best natural RB Cancelo should be in RB slot
	if slotMap["RB"] == nil || slotMap["RB"].PlayerID != "RB1" {
		t.Errorf("expected RB1 in RB slot, got %v", slotMap["RB"])
	}
	// CBs
	if slotMap["LCB"] == nil || slotMap["RCB"] == nil {
		t.Errorf("missing LCB or RCB: %v, %v", slotMap["LCB"], slotMap["RCB"])
	}
	// Kounde (RB2) was lower priority than Cancelo for RB and CBs were filled by Araujo/Christensen
	if seenPIDs["RB2"] {
		// If Kounde did start, his slot must NOT be RB!
		for _, s := range slots {
			if s.Player.PlayerID == "RB2" && s.Slot == "RB" {
				t.Errorf("RB2 should not have collided into RB slot")
			}
		}
	}
	// Midfielder slot "CDM" should match M2's position
	if slotMap["CDM"] == nil || slotMap["CDM"].PlayerID != "M2" {
		t.Errorf("expected CDM slot to be filled by M2, got %v", slotMap["CDM"])
	}
}

func TestGetStartingElevenSlotsMissingPositionsAndThinSquad(t *testing.T) {
	// Squad without natural LB (two RBs only)
	club := &Club{
		ClubID:   "TEST-2",
		ClubName: "Test 2",
		Squad: []*Player{
			{PlayerID: "GK1", FullName: "Goalie", Position: "GK", Category: "GK", OVR: 80},
			{PlayerID: "CB1", FullName: "CB One", Position: "CB", Category: "DEF", OVR: 82},
			{PlayerID: "CB2", FullName: "CB Two", Position: "CB", Category: "DEF", OVR: 81},
			{PlayerID: "RB1", FullName: "RB One", Position: "RB", Category: "DEF", OVR: 85},
			{PlayerID: "RB2", FullName: "RB Two", Position: "RB", Category: "DEF", OVR: 83},
			{PlayerID: "M1", FullName: "Mid One", Position: "CM", Category: "MID", OVR: 80},
			{PlayerID: "M2", FullName: "Mid Two", Position: "CM", Category: "MID", OVR: 80},
			{PlayerID: "M3", FullName: "Mid Three", Position: "CM", Category: "MID", OVR: 80},
			{PlayerID: "F1", FullName: "Fwd One", Position: "ST", Category: "FWD", OVR: 80},
			{PlayerID: "F2", FullName: "Fwd Two", Position: "ST", Category: "FWD", OVR: 80},
			{PlayerID: "F3", FullName: "Fwd Three", Position: "ST", Category: "FWD", OVR: 80},
		},
	}

	slots := club.GetStartingElevenSlots()
	if len(slots) != 11 {
		t.Fatalf("expected 11 slots, got %d", len(slots))
	}

	seenSlots := make(map[string]int)
	seenPlayers := make(map[string]int)
	for _, s := range slots {
		seenSlots[s.Slot]++
		seenPlayers[s.Player.PlayerID]++
	}

	for sl, count := range seenSlots {
		if count > 1 {
			t.Errorf("slot %s assigned multiple times: %d", sl, count)
		}
	}
	for pid, count := range seenPlayers {
		if count > 1 {
			t.Errorf("player %s assigned multiple times: %d", pid, count)
		}
	}

	// Verify both LB and RB slots are occupied without collision
	if seenSlots["LB"] != 1 || seenSlots["RB"] != 1 {
		t.Errorf("expected both LB and RB slots to be occupied: LB=%d RB=%d", seenSlots["LB"], seenSlots["RB"])
	}

	// Thin squad: only 5 available players
	thinClub := &Club{
		ClubID: "THIN",
		Squad:  club.Squad[:5],
	}
	thinSlots := thinClub.GetStartingElevenSlots()
	if len(thinSlots) != 5 {
		t.Fatalf("expected 5 slots for 5-player squad, got %d", len(thinSlots))
	}
}

func TestGetStartingElevenSlotsExcludesInjuredAndUnavailable(t *testing.T) {
	club := &Club{
		ClubID: "AVAIL",
		Squad: []*Player{
			{PlayerID: "GK1", Position: "GK", Category: "GK", OVR: 85},
			{PlayerID: "GK2", Position: "GK", Category: "GK", OVR: 75},
			{PlayerID: "DEF1", Position: "LB", Category: "DEF", OVR: 90, InjuredMatches: 2}, // injured!
			{PlayerID: "DEF2", Position: "LB", Category: "DEF", OVR: 78},
			{PlayerID: "DEF3", Position: "CB", Category: "DEF", OVR: 82},
			{PlayerID: "DEF4", Position: "CB", Category: "DEF", OVR: 81},
			{PlayerID: "DEF5", Position: "RB", Category: "DEF", OVR: 83},
			{PlayerID: "MID1", Position: "CM", Category: "MID", OVR: 82},
			{PlayerID: "MID2", Position: "CM", Category: "MID", OVR: 81},
			{PlayerID: "MID3", Position: "CM", Category: "MID", OVR: 80},
			{PlayerID: "FWD1", Position: "LW", Category: "FWD", OVR: 85},
			{PlayerID: "FWD2", Position: "ST", Category: "FWD", OVR: 84},
			{PlayerID: "FWD3", Position: "RW", Category: "FWD", OVR: 83},
		},
	}

	slots := club.GetStartingElevenSlots()
	if len(slots) != 11 {
		t.Fatalf("expected 11 slots, got %d", len(slots))
	}
	for _, s := range slots {
		if s.Player.PlayerID == "DEF1" {
			t.Fatalf("injured player DEF1 should not be in starting slots")
		}
	}
	// DEF2 should have taken the LB slot
	foundLB := false
	for _, s := range slots {
		if s.Slot == "LB" && s.Player.PlayerID == "DEF2" {
			foundLB = true
		}
	}
	if !foundLB {
		t.Errorf("DEF2 should have filled LB slot in place of injured DEF1")
	}
}

func TestGetStartingElevenSlotsReservesSpecialistsBeforeFallback(t *testing.T) {
	club := &Club{
		ClubID: "ROLE-RESERVATION",
		Squad: []*Player{
			{PlayerID: "GK", Position: "GK", Category: "GK", OVR: 80},
			{PlayerID: "CB1", Position: "CB", Category: "DEF", OVR: 86},
			{PlayerID: "CB2", Position: "CB", Category: "DEF", OVR: 85},
			{PlayerID: "CB3", Position: "CB", Category: "DEF", OVR: 84},
			{PlayerID: "RB1", Position: "RB", Category: "DEF", OVR: 90},
			{PlayerID: "RB2", Position: "RB", Category: "DEF", OVR: 89},
			{PlayerID: "M1", Position: "CM", Category: "MID", OVR: 84},
			{PlayerID: "M2", Position: "CM", Category: "MID", OVR: 83},
			{PlayerID: "M3", Position: "CM", Category: "MID", OVR: 82},
			{PlayerID: "LW", Position: "LW", Category: "FWD", OVR: 85},
			{PlayerID: "ST", Position: "ST", Category: "FWD", OVR: 86},
			{PlayerID: "RW", Position: "RW", Category: "FWD", OVR: 84},
		},
	}

	slots := club.GetStartingElevenSlots()
	bySlot := make(map[string]*Player, len(slots))
	for _, slot := range slots {
		bySlot[slot.Slot] = slot.Player
	}
	if bySlot["RB"] == nil || bySlot["RB"].PlayerID != "RB1" {
		t.Fatalf("natural RB must be reserved for RB, got %#v", bySlot["RB"])
	}
	if bySlot["LB"] == nil || bySlot["LB"].Position != "CB" {
		t.Fatalf("no-LB fallback should prefer an available CB, got %#v", bySlot["LB"])
	}
	for _, slot := range slots {
		if slot.Player.PlayerID == "RB2" {
			t.Fatalf("second RB should stay out when a third CB can cover LB: assigned to %s", slot.Slot)
		}
	}
}

func TestGetStartingElevenSlotsKeepsCAMCentralAndCreatesDeterministicWideFallback(t *testing.T) {
	club := &Club{
		ClubID: "CAM-WIDE-FALLBACK",
		Squad: []*Player{
			{PlayerID: "GK", Position: "GK", Category: "GK", OVR: 80},
			{PlayerID: "LB", Position: "LB", Category: "DEF", OVR: 80},
			{PlayerID: "CB1", Position: "CB", Category: "DEF", OVR: 80},
			{PlayerID: "CB2", Position: "CB", Category: "DEF", OVR: 80},
			{PlayerID: "RB", Position: "RB", Category: "DEF", OVR: 80},
			{PlayerID: "LM", Position: "LM", Category: "MID", OVR: 80},
			{PlayerID: "CAM", Position: "CAM", Category: "MID", OVR: 85},
			{PlayerID: "RM", Position: "RM", Category: "MID", OVR: 80},
			{PlayerID: "LW", Position: "LW", Category: "FWD", OVR: 80},
			{PlayerID: "ST1", Position: "ST", Category: "FWD", OVR: 86},
			{PlayerID: "ST2", Position: "ST", Category: "FWD", OVR: 81},
		},
	}

	first, second := club.GetStartingElevenSlots(), club.GetStartingElevenSlots()
	if len(first) != 11 || len(second) != 11 {
		t.Fatalf("expected deterministic 11-player lineups, got %d and %d", len(first), len(second))
	}
	seen := make(map[string]bool, len(first))
	for i, slot := range first {
		if seen[slot.Player.PlayerID] {
			t.Fatalf("duplicate player %s", slot.Player.PlayerID)
		}
		seen[slot.Player.PlayerID] = true
		if slot.Slot != second[i].Slot || slot.Player.PlayerID != second[i].Player.PlayerID {
			t.Fatalf("slot allocation is not deterministic: %#v then %#v", first, second)
		}
	}
	bySlot := make(map[string]*Player, len(first))
	for _, slot := range first {
		bySlot[slot.Slot] = slot.Player
	}
	if bySlot["CAM"] == nil || bySlot["CAM"].PlayerID != "CAM" {
		t.Fatalf("CAM must remain central and advanced, got %#v", bySlot["CAM"])
	}
	if bySlot["RW"] == nil || bySlot["RW"].PlayerID != "ST2" {
		t.Fatalf("missing-RW fallback should use the spare striker on the right, got %#v", bySlot["RW"])
	}
}
