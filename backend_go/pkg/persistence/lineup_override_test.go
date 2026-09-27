package persistence

import (
	"path/filepath"
	"testing"

	"football_sim/pkg/models"
)

// The viewer lineup override (SaveVersion 13) survives a save/load round
// trip, restores onto the live club, and malformed overrides are rejected.
func TestLineupOverrideSurvivesRoundTrip(t *testing.T) {
	_, ge, tm, te := setupTestWorld(t)

	club := tm.ClubsList[0]
	// Build a valid override from the club's own auto XI so the rigid-slot
	// contract holds regardless of dataset composition.
	slots := club.GetStartingElevenSlots()
	if len(slots) != 11 {
		t.Fatalf("auto XI must have 11 slots, got %d", len(slots))
	}
	players := map[string]string{}
	for _, slot := range slots {
		if slot.Player != nil {
			players[slot.Slot] = slot.Player.PlayerID
		}
	}
	override := &models.LineupOverride{Formation: models.Formation433, Players: players}
	if err := models.ValidateLineupOverride(club.Squad, override); err != nil {
		t.Fatalf("fixture override invalid: %v", err)
	}
	club.LineupOverride = override

	tempDir := t.TempDir()
	savePath := filepath.Join(tempDir, "lineup_career.json")
	if _, err := SaveCareer(tm, ge, te, savePath); err != nil {
		t.Fatalf("SaveCareer failed: %v", err)
	}
	snap, err := LoadCareer(savePath)
	if err != nil {
		t.Fatalf("LoadCareer failed: %v", err)
	}
	if snap.Version != SaveVersion {
		t.Fatalf("snapshot version %d, want %d", snap.Version, SaveVersion)
	}
	saved := snap.Clubs[club.ClubID]
	if saved.LineupOverride == nil || saved.LineupOverride.Formation != models.Formation433 {
		t.Fatalf("lineup override lost in save: %+v", saved.LineupOverride)
	}

	_, freshGE, freshTM, freshTE := setupTestWorld(t)
	if err := RestoreCareer(freshTM, freshGE, freshTE, snap); err != nil {
		t.Fatalf("RestoreCareer failed: %v", err)
	}
	restored := freshTM.Clubs[club.ClubID]
	if restored.LineupOverride == nil || len(restored.LineupOverride.Players) != 11 {
		t.Fatalf("lineup override lost in restore: %+v", restored.LineupOverride)
	}
	// The restored override still decides the XI.
	effective := restored.GetStartingElevenSlots()
	for _, slot := range effective {
		want, ok := restored.LineupOverride.Players[slot.Slot]
		if !ok || slot.Player == nil || slot.Player.PlayerID != want {
			t.Fatalf("restored override not honoured at slot %s", slot.Slot)
		}
	}
}

func TestValidationRejectsMalformedLineupOverride(t *testing.T) {
	_, ge, tm, te := setupTestWorld(t)

	snap := BuildSnapshot(tm, ge, te)
	club := snap.Clubs[tm.ClubsList[0].ClubID]
	club.LineupOverride = &models.LineupOverride{
		Formation: "4-3-3",
		Players:   map[string]string{"GK": "NOT-A-PLAYER"},
	}
	if err := ValidateCareerSnapshot(snap); err == nil {
		t.Fatal("validation must reject a lineup override referencing a non-squad player")
	}

	snap = BuildSnapshot(tm, ge, te)
	club = snap.Clubs[tm.ClubsList[0].ClubID]
	club.LineupOverride = &models.LineupOverride{Formation: "4-3-3", Players: map[string]string{"GK": ""}}
	if err := ValidateCareerSnapshot(snap); err == nil {
		t.Fatal("validation must reject an empty lineup override")
	}

	// A well-formed override passes validation.
	snap = BuildSnapshot(tm, ge, te)
	club = snap.Clubs[tm.ClubsList[0].ClubID]
	slots := club.GetStartingElevenSlots()
	players := map[string]string{}
	for _, slot := range slots {
		if slot.Player != nil {
			players[slot.Slot] = slot.Player.PlayerID
		}
	}
	club.LineupOverride = &models.LineupOverride{Formation: models.Formation433, Players: players}
	if err := ValidateCareerSnapshot(snap); err != nil {
		t.Fatalf("well-formed override must validate: %v", err)
	}
}
