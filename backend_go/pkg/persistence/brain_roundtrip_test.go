package persistence

import (
	"path/filepath"
	"testing"

	"football_sim/pkg/brain"
)

// The world brain survives a save/load round trip (SaveVersion 19) and
// keeps post-training from where it left off.
func TestBrainSurvivesRoundTrip(t *testing.T) {
	ge, tm, te := setupEuropeanWorld(t)
	batch := tm.SimulateBatchWeeks(8)
	if batch.Status != "success" {
		t.Fatalf("simulation failed: %+v", batch)
	}
	before := tm.BrainState()
	if before == nil || before.Samples == 0 {
		t.Fatal("brain never trained")
	}

	tempDir := t.TempDir()
	savePath := filepath.Join(tempDir, "brain_career.json")
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
	if snap.Brain == nil || !snap.Brain.Valid() {
		t.Fatal("brain lost in save")
	}
	if snap.Brain.Samples != before.Samples {
		t.Fatalf("brain samples lost: %d want %d", snap.Brain.Samples, before.Samples)
	}

	freshGE, freshTM, freshTE := setupEuropeanWorld(t)
	if err := RestoreCareer(freshTM, freshGE, freshTE, snap); err != nil {
		t.Fatalf("RestoreCareer failed: %v", err)
	}
	after := freshTM.BrainState()
	if after == nil || *after != *before {
		t.Fatal("brain did not survive the round trip")
	}
}

// Validation rejects a corrupted brain.
func TestValidationRejectsCorruptedBrain(t *testing.T) {
	ge, tm, te := setupEuropeanWorld(t)
	snap := BuildSnapshot(tm, ge, te)
	snap.Brain = brain.New()
	if err := ValidateCareerSnapshot(snap); err != nil {
		t.Fatalf("a valid brain must pass validation: %v", err)
	}
	snap.Brain.Weights[0] = 1e6
	if err := ValidateCareerSnapshot(snap); err == nil {
		t.Fatal("validation must reject an out-of-range brain weight")
	}
}
