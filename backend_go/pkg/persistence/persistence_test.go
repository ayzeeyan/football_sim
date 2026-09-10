package persistence

import (
	"os"
	"path/filepath"
	"testing"

	"football_sim/pkg/datamanager"
	"football_sim/pkg/growth"
	"football_sim/pkg/tournament"
	"football_sim/pkg/transfers"
)

func setupTestWorld(t *testing.T) (*datamanager.DataManager, *growth.GrowthEngine, *tournament.TournamentManager, *transfers.TransferEngine) {
	t.Helper()
	ge := growth.NewGrowthEngine(42)
	datasetPath := filepath.Join("..", "..", "..", "dataset.json")
	dm := datamanager.NewDataManager(datasetPath, ge)
	if dm == nil || len(dm.Clubs) == 0 {
		t.Fatalf("failed to load dataset from %s", datasetPath)
	}

	eliteClubs := dm.GetEliteClubs()
	if len(eliteClubs) != 12 {
		t.Fatalf("expected 12 elite clubs, got %d", len(eliteClubs))
	}

	tm := tournament.NewTournamentManager(eliteClubs, ge, 42)
	te := transfers.NewTransferEngine(eliteClubs, tm.Managers, 42)
	return dm, ge, tm, te
}

func TestSaveAndLoadCareer(t *testing.T) {
	_, ge, tm, te := setupTestWorld(t)

	// Simulate MW 1 to generate realistic match activity
	res := tm.SimulateMatchweek(1)
	if res["status"] != "success" {
		t.Fatalf("expected success, got %v", res)
	}

	tempDir := t.TempDir()
	savePath := filepath.Join(tempDir, "test_career.json")

	// 1. Save career
	savedPath, err := SaveCareer(tm, ge, te, savePath)
	if err != nil {
		t.Fatalf("SaveCareer failed: %v", err)
	}
	if savedPath != savePath {
		t.Fatalf("expected path %s, got %s", savePath, savedPath)
	}

	// 2. Verify file exists and has content
	info, err := os.Stat(savePath)
	if err != nil {
		t.Fatalf("save file does not exist: %v", err)
	}
	if info.Size() == 0 {
		t.Fatalf("save file is empty")
	}

	// 3. Load career
	snap, err := LoadCareer(savePath)
	if err != nil {
		t.Fatalf("LoadCareer failed: %v", err)
	}

	if snap.Version != SaveVersion {
		t.Errorf("expected version %d, got %d", SaveVersion, snap.Version)
	}
	if snap.CurrentMatchweek != 2 {
		t.Errorf("expected matchweek 2 after MW1 simulation, got %d", snap.CurrentMatchweek)
	}
	if len(snap.Clubs) != 12 {
		t.Errorf("expected 12 clubs in snapshot, got %d", len(snap.Clubs))
	}
	if len(snap.Fixtures) != 264 {
		t.Errorf("expected 264 fixtures in snapshot, got %d", len(snap.Fixtures))
	}
	if len(snap.Growth.Biometrics) != 12 {
		t.Errorf("expected 12 wonderkid biometrics, got %d", len(snap.Growth.Biometrics))
	}
}

func TestWriteSnapshotRoundTrip(t *testing.T) {
	_, ge, tm, te := setupTestWorld(t)
	built := BuildSnapshot(tm, ge, te)
	tempDir := t.TempDir()
	savePath := filepath.Join(tempDir, "snapshot_career.json")
	if _, err := WriteSnapshot(built, savePath); err != nil {
		t.Fatalf("WriteSnapshot failed: %v", err)
	}
	loaded, err := LoadCareer(savePath)
	if err != nil {
		t.Fatalf("LoadCareer failed: %v", err)
	}
	if loaded.SeasonName != built.SeasonName || loaded.CurrentMatchweek != built.CurrentMatchweek {
		t.Fatalf("snapshot round-trip mismatch: %+v vs %+v", loaded, built)
	}
	if _, err := WriteSnapshot(nil, savePath); err == nil {
		t.Fatal("WriteSnapshot(nil) succeeded")
	}
}

func TestRestoreCareer(t *testing.T) {
	_, ge, tm, te := setupTestWorld(t)

	// Simulate MW 1 & MW 2
	tm.SimulateMatchweek(1)
	tm.SimulateMatchweek(2)

	// Ingest a simulated transfer
	p := tm.Clubs["LAL-BAR"].Squad[0]
	te.CompletedTransfers = append(te.CompletedTransfers, transfers.CompletedTransfer{
		PlayerID:   p.PlayerID,
		PlayerName: p.FullName,
		SellerID:   "LAL-BAR",
		SellerName: "Barcelona",
		BuyerID:    "FL1-PSG",
		BuyerName:  "Paris Saint-Germain",
		FeeEUR:     75000000,
		Matchweek:  2,
	})

	tempDir := t.TempDir()
	savePath := filepath.Join(tempDir, "restore_career.json")

	_, err := SaveCareer(tm, ge, te, savePath)
	if err != nil {
		t.Fatalf("SaveCareer failed: %v", err)
	}

	snap, err := LoadCareer(savePath)
	if err != nil {
		t.Fatalf("LoadCareer failed: %v", err)
	}

	// Create fresh world
	_, freshGE, freshTM, freshTE := setupTestWorld(t)
	if freshTM.CurrentMatchweek != 1 {
		t.Fatalf("expected fresh world MW 1, got %d", freshTM.CurrentMatchweek)
	}

	// Restore snapshot
	if err := RestoreCareer(freshTM, freshGE, freshTE, snap); err != nil {
		t.Fatalf("RestoreCareer failed: %v", err)
	}

	if freshTM.CurrentMatchweek != 3 {
		t.Errorf("expected restored matchweek 3, got %d", freshTM.CurrentMatchweek)
	}
	if len(freshTM.Inbox) == 0 {
		t.Errorf("expected restored inbox items")
	}

	// Verify wonderkids integrity
	for _, cfg := range datamanager.EliteProdigyConfigs {
		id := datamanager.ProdigyStableID(cfg.FullName)
		bio, ok := freshGE.Biometrics[id]
		if !ok {
			t.Errorf("missing biometric for wonderkid %s", id)
			continue
		}
		if bio.Potential < 93 || bio.Potential > 96 {
			t.Errorf("wonderkid %s has invalid potential %d (expected 93-96)", id, bio.Potential)
		}
	}

	// Verify strict 0-duplicate invariant across all restored clubs
	seenGlobal := make(map[string]string)
	for _, club := range freshTM.ClubsList {
		seenInClub := make(map[string]bool)
		for _, pl := range club.Squad {
			if seenInClub[pl.PlayerID] {
				t.Errorf("duplicate player %s (%s) within club %s", pl.FullName, pl.PlayerID, club.ClubID)
			}
			seenInClub[pl.PlayerID] = true
			if otherClub, exists := seenGlobal[pl.PlayerID]; exists {
				t.Errorf("player %s (%s) exists in multiple clubs: %s and %s", pl.FullName, pl.PlayerID, otherClub, club.ClubID)
			}
			seenGlobal[pl.PlayerID] = club.ClubID
		}
	}

	// Verify completed transfer restored
	if len(freshTE.CompletedTransfers) != 1 {
		t.Errorf("expected 1 restored completed transfer, got %d", len(freshTE.CompletedTransfers))
	}
}

func TestDeleteCareer(t *testing.T) {
	tempDir := t.TempDir()
	savePath := filepath.Join(tempDir, "delete_test.json")

	if err := os.WriteFile(savePath, []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to create dummy save: %v", err)
	}

	if err := DeleteCareer(savePath); err != nil {
		t.Fatalf("DeleteCareer failed: %v", err)
	}

	if _, err := os.Stat(savePath); !os.IsNotExist(err) {
		t.Errorf("expected file to be deleted")
	}

	// Deleting non-existent file should be a no-op / nil error
	if err := DeleteCareer(savePath); err != nil {
		t.Errorf("deleting non-existent file returned error: %v", err)
	}
}

func TestLoadExistingCareerJSON(t *testing.T) {
	existingPath := filepath.Join("..", "..", "..", "saves", "career.json")
	if _, err := os.Stat(existingPath); os.IsNotExist(err) {
		t.Skip("saves/career.json not present in workspace")
	}

	snap, err := LoadCareer(existingPath)
	if err != nil {
		t.Fatalf("failed to load existing saves/career.json: %v", err)
	}

	if snap.Version != 1 {
		t.Errorf("expected version 1, got %d", snap.Version)
	}
	if snap.SeasonName == "" {
		t.Errorf("expected non-empty season_name")
	}
	if len(snap.Clubs) == 0 {
		t.Errorf("expected clubs in existing saves/career.json")
	}
}
