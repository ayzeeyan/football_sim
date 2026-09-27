package persistence

import (
	"path/filepath"
	"sort"
	"testing"

	"football_sim/pkg/tournament"
)

// firstClubID returns a deterministic club ID from the test world.
func firstClubID(tm *tournament.TournamentManager) string {
	ids := make([]string, 0, len(tm.Clubs))
	for id := range tm.Clubs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids[0]
}

// The viewer manager career survives a save/load round trip and malformed
// ledgers are rejected by validation (SaveVersion 14).
func TestViewerManagerSurvivesRoundTrip(t *testing.T) {
	_, ge, tm, te := setupTestWorld(t)
	clubID := firstClubID(tm)

	vm, msg := tm.AcceptViewerJob(clubID, "Test Gaffer")
	if msg != "" || vm == nil {
		t.Fatalf("AcceptViewerJob failed: %v", msg)
	}
	if vm.ClubID != clubID || len(vm.History) != 1 || vm.History[0].Outcome != "active" {
		t.Fatalf("unexpected career state after accepting: %+v", vm)
	}
	tm.RecordViewerTrophies(tm.Clubs[clubID], []string{"Super League"})

	tempDir := t.TempDir()
	savePath := filepath.Join(tempDir, "viewer_career.json")
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
	if snap.ViewerManager == nil || snap.ViewerManager.ClubID != clubID {
		t.Fatalf("viewer manager lost in save: %+v", snap.ViewerManager)
	}
	if len(snap.ViewerManager.Trophies) != 1 {
		t.Fatalf("trophies lost in save: %+v", snap.ViewerManager.Trophies)
	}

	_, freshGE, freshTM, freshTE := setupTestWorld(t)
	if err := RestoreCareer(freshTM, freshGE, freshTE, snap); err != nil {
		t.Fatalf("RestoreCareer failed: %v", err)
	}
	restored := freshTM.GetViewerManager()
	if restored == nil || restored.ClubID != clubID || restored.Name != "Test Gaffer" {
		t.Fatalf("viewer manager lost in restore: %+v", restored)
	}
	if len(restored.Trophies) != 1 || restored.Sackings != 0 {
		t.Fatalf("career ledger lost in restore: %+v", restored)
	}
	if mgr := freshTM.Managers[clubID]; mgr == nil || mgr.Name != "Test Gaffer" {
		t.Fatalf("viewer not installed as club manager after restore: %+v", mgr)
	}
	// Resigning after restore works and appoints a successor.
	if _, msg := freshTM.ResignViewerJob(); msg != "" {
		t.Fatalf("post-restore resign failed: %v", msg)
	}
	if m := freshTM.GetViewerManager(); m == nil || m.ClubID != "" {
		t.Fatalf("resign did not clear the active job: %+v", m)
	}
	if mgr := freshTM.Managers[clubID]; mgr == nil || mgr.Name == "Test Gaffer" {
		t.Fatalf("resign did not appoint a successor: %+v", mgr)
	}
}

func TestValidationRejectsMalformedViewerManager(t *testing.T) {
	_, ge, tm, te := setupTestWorld(t)
	clubID := firstClubID(tm)
	if _, msg := tm.AcceptViewerJob(clubID, ""); msg != "" {
		t.Fatalf("AcceptViewerJob failed: %v", msg)
	}

	snap := BuildSnapshot(tm, ge, te)
	snap.ViewerManager.Name = ""
	if err := ValidateCareerSnapshot(snap); err == nil {
		t.Fatal("validation must reject an empty viewer manager name")
	}

	snap = BuildSnapshot(tm, ge, te)
	snap.ViewerManager.ClubID = "NO_SUCH_CLUB"
	if err := ValidateCareerSnapshot(snap); err == nil {
		t.Fatal("validation must reject a viewer manager referencing an unknown club")
	}

	snap = BuildSnapshot(tm, ge, te)
	snap.ViewerManager.History[0].Outcome = "vanished"
	if err := ValidateCareerSnapshot(snap); err == nil {
		t.Fatal("validation must reject an unknown job outcome")
	}

	snap = BuildSnapshot(tm, ge, te)
	for i := 0; i < tournament.MaxViewerJobRecords+1; i++ {
		snap.ViewerManager.History = append(snap.ViewerManager.History, tournament.ViewerJobRecord{
			ClubID: clubID, Season: snap.SeasonName, Outcome: "resigned",
		})
	}
	if err := ValidateCareerSnapshot(snap); err == nil {
		t.Fatal("validation must reject a job ledger above the cap")
	}
}
