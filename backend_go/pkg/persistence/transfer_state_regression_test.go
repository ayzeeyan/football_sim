package persistence

import (
	"path/filepath"
	"testing"
)

func TestTransferWindowStateSurvivesSaveRestore(t *testing.T) {
	_, ge, tm, te := setupTestWorld(t)

	playerID := tm.ClubsList[0].Squad[0].PlayerID
	te.CurrentDay = 7
	te.CurrentMatchweek = 33
	te.CurrentWeek = 6
	te.IsOffSeason = true
	te.TransferredThisWindow[playerID] = true

	savePath := filepath.Join(t.TempDir(), "career.json")
	if _, err := SaveCareer(tm, ge, te, savePath); err != nil {
		t.Fatalf("SaveCareer failed: %v", err)
	}

	snap, err := LoadCareer(savePath)
	if err != nil {
		t.Fatalf("LoadCareer failed: %v", err)
	}
	if snap.Transfers.CurrentWeek != 6 {
		t.Fatalf("saved transfer week = %d, want 6", snap.Transfers.CurrentWeek)
	}
	if !snap.Transfers.IsOffSeason {
		t.Fatal("saved transfer state lost off-season flag")
	}
	if !snap.Transfers.TransferredThisWindow[playerID] {
		t.Fatal("saved transfer state lost single-window transfer guard")
	}

	_, freshGE, freshTM, freshTE := setupTestWorld(t)
	if err := RestoreCareer(freshTM, freshGE, freshTE, snap); err != nil {
		t.Fatalf("RestoreCareer failed: %v", err)
	}

	if freshTE.CurrentDay != 7 || freshTE.CurrentWeek != 6 || freshTE.CurrentMatchweek != 33 {
		t.Fatalf("restored transfer clock = day %d, week %d, MW %d; want day 7, week 6, MW 33",
			freshTE.CurrentDay, freshTE.CurrentWeek, freshTE.CurrentMatchweek)
	}
	if !freshTE.IsOffSeason {
		t.Fatal("restored transfer state lost off-season flag")
	}
	if !freshTE.TransferredThisWindow[playerID] {
		t.Fatal("restored transfer state lost single-window transfer guard")
	}
}
