package persistence

import (
	"path/filepath"
	"testing"
)

// A swap-era save (domestic registry disagreeing with the dataset-derived
// club leagues) must realign to the country-pure pyramid on restore, pass
// validation, and write back as the current SaveVersion.
func TestRestoreRealignsSwapEraSave(t *testing.T) {
	ge, tm, te := setupEuropeanWorld(t)
	snap := BuildSnapshot(tm, ge, te)
	if snap.World == nil || snap.World.Competitions["premier-league"] == nil {
		t.Fatal("snapshot has no domestic league registry")
	}

	// Corrupt the save exactly like the reported failing one: Atletico
	// Madrid registered to the Premier League while its league is La Liga.
	epl := snap.World.Competitions["premier-league"]
	epl.ParticipantIDs = append(epl.ParticipantIDs, "LAL-ATM")
	// The failing save predates the current format.
	snap.Version = SaveVersion - 1

	// The corrupted snapshot restores into a realigned, valid world.
	freshGE, freshTM, freshTE := setupEuropeanWorld(t)
	if err := RestoreCareer(freshTM, freshGE, freshTE, snap); err != nil {
		t.Fatalf("RestoreCareer failed: %v", err)
	}
	if err := freshTM.ValidateWorldState(); err != nil {
		t.Fatalf("restored world invalid after realignment: %v", err)
	}
	if freshTM.CurrentMatchweek != 1 {
		t.Fatalf("realigned season must restart at matchweek 1, got %d", freshTM.CurrentMatchweek)
	}

	// The realigned world passes the migration write-back end to end.
	tempDir := t.TempDir()
	savePath := filepath.Join(tempDir, "realign_career.json")
	wrote, err := MaybeWriteMigratedCareer(freshTM, freshGE, freshTE, savePath, snap.Version)
	if err != nil {
		t.Fatalf("MaybeWriteMigratedCareer failed: %v", err)
	}
	if !wrote {
		t.Fatal("migration write-back must run for an older snapshot")
	}
	loaded, err := LoadCareer(savePath)
	if err != nil {
		t.Fatalf("LoadCareer failed: %v", err)
	}
	if loaded.Version != SaveVersion {
		t.Fatalf("written version %d want %d", loaded.Version, SaveVersion)
	}
	// The written save is country-pure and needs no second realignment.
	thirdGE, thirdTM, thirdTE := setupEuropeanWorld(t)
	if err := RestoreCareer(thirdTM, thirdGE, thirdTE, loaded); err != nil {
		t.Fatalf("second RestoreCareer failed: %v", err)
	}
	if thirdTM.RealignCountryPureWorld() {
		t.Fatal("a realigned save must not realign again")
	}
	if err := thirdTM.ValidateWorldState(); err != nil {
		t.Fatalf("reloaded world invalid: %v", err)
	}
}
