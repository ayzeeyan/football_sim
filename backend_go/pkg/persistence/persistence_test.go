package persistence

import (
	"os"
	"path/filepath"
	"testing"

	"football_sim/pkg/datamanager"
	"football_sim/pkg/growth"
	"football_sim/pkg/models"
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

func TestRestoreCareerPersistsClubIdentityAcrossReloads(t *testing.T) {
	_, ge, tm, te := setupTestWorld(t)
	want := map[string]models.ClubIdentity{
		"LAL-RMA": {
			Reputation: 1, HistoricalPrestige: 2, FinancialPower: 3,
			BoardPatience: 4, AcademyQuality: 5, RecruitmentAmbition: 6,
			YouthPreference: 7, TransferAggressiveness: 8, SellingTendency: 9,
		},
		"LAL-BAR": {},
		"BUN-BAY": {
			Reputation: 100, HistoricalPrestige: 0, FinancialPower: 100,
			BoardPatience: 0, AcademyQuality: 100, RecruitmentAmbition: 0,
			YouthPreference: 100, TransferAggressiveness: 0, SellingTendency: 100,
		},
	}
	for clubID, identity := range want {
		tm.Clubs[clubID].Identity = identity
	}

	path := filepath.Join(t.TempDir(), "identity.json")
	if _, err := SaveCareer(tm, ge, te, path); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	loaded, err := LoadCareer(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	_, freshGE, freshTM, freshTE := setupTestWorld(t)
	if err := RestoreCareer(freshTM, freshGE, freshTE, loaded); err != nil {
		t.Fatalf("restore failed: %v", err)
	}
	assertClubIdentities(t, freshTM, want)

	secondPath := filepath.Join(t.TempDir(), "identity-second.json")
	if _, err := SaveCareer(freshTM, freshGE, freshTE, secondPath); err != nil {
		t.Fatalf("second save failed: %v", err)
	}
	second, err := LoadCareer(secondPath)
	if err != nil {
		t.Fatalf("second load failed: %v", err)
	}
	_, repeatedGE, repeatedTM, repeatedTE := setupTestWorld(t)
	if err := RestoreCareer(repeatedTM, repeatedGE, repeatedTE, second); err != nil {
		t.Fatalf("repeated restore failed: %v", err)
	}
	assertClubIdentities(t, repeatedTM, want)
}

func TestRestoreCareerPersistsClubFinancesAcrossReloads(t *testing.T) {
	_, ge, tm, te := setupTestWorld(t)
	wantFinances := map[string]models.ClubFinances{
		"LAL-RMA": {TransferBudget: 145_000_000, Balance: 300_000_000},
		"LAL-BAR": {TransferBudget: 25_000_000, Balance: 50_000_000},
		"BUN-BAY": {TransferBudget: 0, Balance: 100_000_000},
	}
	for clubID, fin := range wantFinances {
		tm.Clubs[clubID].Finances = fin
	}

	path := filepath.Join(t.TempDir(), "finances.json")
	if _, err := SaveCareer(tm, ge, te, path); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	loaded, err := LoadCareer(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	_, freshGE, freshTM, freshTE := setupTestWorld(t)
	if err := RestoreCareer(freshTM, freshGE, freshTE, loaded); err != nil {
		t.Fatalf("restore failed: %v", err)
	}

	for clubID, want := range wantFinances {
		got := freshTM.Clubs[clubID].Finances
		if got != want {
			t.Errorf("club %s finances mismatch: got %+v, want %+v", clubID, got, want)
		}
		if freshTE != nil && freshTE.Managers[clubID] != nil {
			if mgrBudget := freshTE.Managers[clubID].BudgetEur; mgrBudget != want.TransferBudget {
				t.Errorf("club %s manager budget mismatch: got %d, want %d", clubID, mgrBudget, want.TransferBudget)
			}
		}
	}
}

func TestRestoreCareerPreservesFreshIdentityForProgrammaticLegacySnapshot(t *testing.T) {
	_, ge, tm, te := setupTestWorld(t)
	clubID := "LAL-RMA"
	want := models.DefaultClubIdentity(clubID, tm.Clubs[clubID].OverallTeamRating)
	snap := &CareerSnapshot{Clubs: map[string]*models.Club{
		clubID: {ClubID: clubID},
	}}

	if err := RestoreCareer(tm, ge, te, snap); err != nil {
		t.Fatalf("restore failed: %v", err)
	}
	assertClubIdentities(t, tm, map[string]models.ClubIdentity{clubID: want})
}

func assertClubIdentities(t *testing.T, tm *tournament.TournamentManager, want map[string]models.ClubIdentity) {
	t.Helper()
	for clubID, expected := range want {
		got := tm.Clubs[clubID].Identity
		if got.Reputation != expected.Reputation || got.HistoricalPrestige != expected.HistoricalPrestige ||
			got.FinancialPower != expected.FinancialPower || got.BoardPatience != expected.BoardPatience ||
			got.AcademyQuality != expected.AcademyQuality || got.RecruitmentAmbition != expected.RecruitmentAmbition ||
			got.YouthPreference != expected.YouthPreference || got.TransferAggressiveness != expected.TransferAggressiveness ||
			got.SellingTendency != expected.SellingTendency {
			t.Errorf("club %s identity got %+v, want %+v", clubID, got, expected)
		}
	}
}

func TestReputationAppliedSeasonSurvivesSaveRestore(t *testing.T) {
	_, ge, tm, te := setupTestWorld(t)
	tm.ReputationAppliedSeason = tm.SeasonName

	snap := BuildSnapshot(tm, ge, te)
	if snap.ReputationAppliedSeason != tm.SeasonName {
		t.Fatalf("snapshot marker=%q want %q", snap.ReputationAppliedSeason, tm.SeasonName)
	}

	_, freshGE, freshTM, freshTE := setupTestWorld(t)
	freshTM.ReputationAppliedSeason = "stale-marker"
	if err := RestoreCareer(freshTM, freshGE, freshTE, snap); err != nil {
		t.Fatalf("RestoreCareer failed: %v", err)
	}
	if freshTM.ReputationAppliedSeason != tm.SeasonName {
		t.Fatalf("restored marker=%q want %q", freshTM.ReputationAppliedSeason, tm.SeasonName)
	}

	// A legacy snapshot omits the additive marker; restore must clear any
	// pre-existing in-memory value rather than treating the old save as current.
	snap.ReputationAppliedSeason = ""
	freshTM.ReputationAppliedSeason = "stale-marker"
	if err := RestoreCareer(freshTM, freshGE, freshTE, snap); err != nil {
		t.Fatalf("legacy RestoreCareer failed: %v", err)
	}
	if freshTM.ReputationAppliedSeason != "" {
		t.Fatalf("legacy snapshot retained marker=%q", freshTM.ReputationAppliedSeason)
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

	if snap.Version <= 0 {
		t.Errorf("expected positive version, got %d", snap.Version)
	}
	if snap.SeasonName == "" {
		t.Errorf("expected non-empty season_name")
	}
	if len(snap.Clubs) == 0 {
		t.Errorf("expected clubs in existing saves/career.json")
	}
}

func TestLoadLegacyCareerFixture(t *testing.T) {
	tempDir := t.TempDir()
	savePath := filepath.Join(tempDir, "legacy_v1.json")
	v1JSON := []byte(`{"version": 1, "season_name": "2024-25", "clubs": {"ARS": {"club_id": "ARS", "club_name": "Arsenal", "short_name": "ARS", "squad": [{"player_id": "P001", "full_name": "Bukayo Saka", "position": "RW", "ovr": 88, "age": 23, "original_club_id": "ARS"}]}}}`)
	if err := os.WriteFile(savePath, v1JSON, 0644); err != nil {
		t.Fatalf("failed to write legacy fixture: %v", err)
	}
	snap, err := LoadCareer(savePath)
	if err != nil {
		t.Fatalf("failed to load legacy fixture: %v", err)
	}
	if snap.Version != 1 {
		t.Errorf("expected version 1, got %d", snap.Version)
	}
	if snap.SeasonName != "2024-25" {
		t.Errorf("expected season_name 2024-25, got %s", snap.SeasonName)
	}
	if len(snap.Clubs) != 1 {
		t.Fatalf("expected 1 club, got %d", len(snap.Clubs))
	}
	arsClub := snap.Clubs["ARS"]
	if arsClub == nil || len(arsClub.Squad) != 1 || arsClub.Squad[0].OriginalClubID != "ARS" {
		t.Errorf("expected original_club_id ARS, got %#v", arsClub)
	}
}

func TestRestoreCareerOriginalClubIDMetadata(t *testing.T) {
	_, ge, tm, te := setupTestWorld(t)
	player := tm.ClubsList[0].Squad[0]
	origClubID := player.ClubID
	player.OriginalClubID = origClubID

	snap := BuildSnapshot(tm, ge, te)
	if snap.Clubs[origClubID].Squad[0].OriginalClubID != origClubID {
		t.Fatalf("snapshot missing original_club_id: %q", snap.Clubs[origClubID].Squad[0].OriginalClubID)
	}

	_, freshGE, freshTM, freshTE := setupTestWorld(t)
	if err := RestoreCareer(freshTM, freshGE, freshTE, snap); err != nil {
		t.Fatalf("RestoreCareer failed: %v", err)
	}

	restored := freshTM.Clubs[origClubID].Squad[0]
	if restored.OriginalClubID != origClubID {
		t.Errorf("restored player missing original_club_id: got %q, want %q", restored.OriginalClubID, origClubID)
	}
}
