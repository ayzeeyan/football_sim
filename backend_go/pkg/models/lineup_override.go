package models

import (
	"fmt"
	"strings"
)

// LineupOverride is a viewer-set starting XI (Tier B): an explicit formation
// plus one squad player per rigid tactical slot. When present and still
// valid it takes precedence over the AI's automatic selection; the moment
// any slotted player is unavailable the club falls back to the AI XI rather
// than fielding an illegal lineup.
type LineupOverride struct {
	Formation string            `json:"formation"`
	Players   map[string]string `json:"players"` // tactical slot -> player id
}

// ValidateLineupOverride enforces the rigid-slot contract on a submitted
// lineup: the formation must be known, its slots covered exactly once each,
// every player a member of the squad, and no player in two slots.
func ValidateLineupOverride(squad []*Player, override *LineupOverride) error {
	if override == nil {
		return fmt.Errorf("lineup override is nil")
	}
	formation := NormalizeFormation(override.Formation)
	slots := FormationSlots(formation)
	if len(slots) == 0 {
		return fmt.Errorf("unknown formation %q", override.Formation)
	}
	if len(override.Players) != len(slots) {
		return fmt.Errorf("lineup must assign exactly %d slots for %s, got %d", len(slots), formation, len(override.Players))
	}
	squadIDs := make(map[string]bool, len(squad))
	for _, p := range squad {
		if p != nil {
			squadIDs[p.PlayerID] = true
		}
	}
	seen := make(map[string]bool, len(override.Players))
	for _, slot := range slots {
		playerID, ok := override.Players[slot]
		if !ok {
			return fmt.Errorf("lineup is missing an assignment for slot %q", slot)
		}
		playerID = strings.TrimSpace(playerID)
		if playerID == "" {
			return fmt.Errorf("slot %q has an empty player assignment", slot)
		}
		if seen[playerID] {
			return fmt.Errorf("player %q is assigned to more than one slot", playerID)
		}
		seen[playerID] = true
		if !squadIDs[playerID] {
			return fmt.Errorf("player %q is not part of this squad", playerID)
		}
	}
	for slot := range override.Players {
		valid := false
		for _, s := range slots {
			if s == slot {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("slot %q is not part of the %s formation", slot, formation)
		}
	}
	return nil
}

// startingSlotsFromOverride materialises the override into tactical slots.
// It returns nil when the override can no longer be honoured — a slotted
// player left the squad or is unavailable for this fixture — so callers fall
// back to the AI selection instead of fielding an illegal XI.
func startingSlotsFromOverride(override *LineupOverride, available []*Player) []StartingSlot {
	if override == nil || len(override.Players) == 0 {
		return nil
	}
	formation := NormalizeFormation(override.Formation)
	byID := make(map[string]*Player, len(available))
	for _, p := range available {
		if p != nil {
			byID[p.PlayerID] = p
		}
	}
	slots := FormationSlots(formation)
	out := make([]StartingSlot, 0, len(slots))
	for _, slot := range slots {
		playerID, ok := override.Players[slot]
		if !ok {
			return nil
		}
		player := byID[playerID]
		if player == nil {
			return nil
		}
		out = append(out, StartingSlot{
			Slot:            slot,
			NaturalPosition: player.Position,
			PositionFit:     PositionFitForPlayer(player, slot),
			Player:          player,
		})
	}
	return out
}
