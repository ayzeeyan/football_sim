package models

import (
	"strings"
	"testing"
)

func lineupSquad() []*Player {
	// A minimal 4-3-3-capable squad with distinct IDs.
	mk := func(id, pos string) *Player {
		return &Player{PlayerID: id, FullName: id, Position: pos, Category: GetPositionCategory(pos), OVR: 75}
	}
	return []*Player{
		mk("GK1", "GK"),
		mk("LB1", "LB"), mk("LCB1", "LCB"), mk("RCB1", "RCB"), mk("RB1", "RB"),
		mk("LCM1", "LCM"), mk("CM1", "CM"), mk("RCM1", "RCM"),
		mk("LW1", "LW"), mk("ST1", "ST"), mk("RW1", "RW"),
		mk("SUB1", "CM"), mk("SUB2", "ST"),
	}
}

func fullLineup() map[string]string {
	return map[string]string{
		"GK": "GK1", "LB": "LB1", "LCB": "LCB1", "RCB": "RCB1", "RB": "RB1",
		"LCM": "LCM1", "CM": "CM1", "RCM": "RCM1",
		"LW": "LW1", "ST": "ST1", "RW": "RW1",
	}
}

func TestValidateLineupOverrideEnforcesRigidSlots(t *testing.T) {
	squad := lineupSquad()

	// A complete, valid 4-3-3 assignment passes.
	valid := &LineupOverride{Formation: "4-3-3", Players: fullLineup()}
	if err := ValidateLineupOverride(squad, valid); err != nil {
		t.Fatalf("valid lineup rejected: %v", err)
	}

	// Missing a slot.
	missing := &LineupOverride{Formation: "4-3-3", Players: fullLineup()}
	delete(missing.Players, "ST")
	if err := ValidateLineupOverride(squad, missing); err == nil || !strings.Contains(err.Error(), "exactly 11 slots") {
		t.Fatalf("missing slot must be rejected: %v", err)
	}

	// A player in two slots.
	dup := &LineupOverride{Formation: "4-3-3", Players: fullLineup()}
	dup.Players["RW"] = "ST1"
	if err := ValidateLineupOverride(squad, dup); err == nil || !strings.Contains(err.Error(), "more than one slot") {
		t.Fatalf("duplicate player must be rejected: %v", err)
	}

	// A player from another squad.
	outsider := &LineupOverride{Formation: "4-3-3", Players: fullLineup()}
	outsider.Players["CM"] = "NOT-IN-SQUAD"
	if err := ValidateLineupOverride(squad, outsider); err == nil || !strings.Contains(err.Error(), "not part of this squad") {
		t.Fatalf("outsider must be rejected: %v", err)
	}

	// A slot that is not part of the formation.
	bogus := &LineupOverride{Formation: "4-3-3", Players: fullLineup()}
	bogus.Players["CAM"] = "SUB1"
	if err := ValidateLineupOverride(squad, bogus); err == nil {
		t.Fatal("bogus slot must be rejected")
	}

	// A 4-3-3 assignment does not fit a 4-2-3-1 shape.
	short := &LineupOverride{Formation: "4-2-3-1", Players: fullLineup()}
	if err := ValidateLineupOverride(squad, short); err == nil || !strings.Contains(err.Error(), "LDM") {
		t.Fatalf("wrong formation shape must be rejected: %v", err)
	}

	// Nil override is rejected by the validator (used only for presence checks).
	if err := ValidateLineupOverride(squad, nil); err == nil {
		t.Fatal("nil override must be rejected")
	}
}

func TestOverrideWinsAndFallsBack(t *testing.T) {
	club := &Club{ClubID: "C1", Squad: lineupSquad()}
	auto := club.GetStartingElevenSlots()
	if len(auto) != 11 {
		t.Fatalf("auto XI must have 11 slots, got %d", len(auto))
	}

	// The override decides the XI, in formation order.
	club.LineupOverride = &LineupOverride{Formation: "4-3-3", Players: fullLineup()}
	slots := club.GetStartingElevenSlots()
	if len(slots) != 11 {
		t.Fatalf("override XI must have 11 slots, got %d", len(slots))
	}
	for _, slot := range slots {
		want, ok := club.LineupOverride.Players[slot.Slot]
		if !ok || slot.Player == nil || slot.Player.PlayerID != want {
			t.Fatalf("slot %s must hold %s, got %+v", slot.Slot, want, slot.Player)
		}
	}

	// An unavailable slotted player invalidates the whole override: the
	// club falls back to the AI XI rather than fielding ten men.
	club.Squad[9].InjuredMatches = 5 // ST1 out
	fallback := club.GetStartingElevenSlots()
	if len(fallback) != 11 {
		t.Fatalf("fallback XI must have 11 slots, got %d", len(fallback))
	}
	for _, slot := range fallback {
		if slot.Player != nil && slot.Player.PlayerID == "ST1" {
			t.Fatal("injured player must not start via the override or the fallback")
		}
	}

	// Clearing the override restores pure AI selection.
	club.Squad[9].InjuredMatches = 0
	club.LineupOverride = nil
	again := club.GetStartingElevenSlots()
	if len(again) != 11 {
		t.Fatal("cleared override must keep a full AI XI")
	}
}
