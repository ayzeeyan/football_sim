package persistence

import (
	"path/filepath"
	"testing"

	"football_sim/pkg/tournament"
)

// The watchlist persists across a save/load round trip and unknown
// references are rejected by validation (SaveVersion 10).
func TestWatchlistSurvivesRoundTrip(t *testing.T) {
	_, ge, tm, te := setupTestWorld(t)

	state, watched, ok := tm.ToggleWatchlist(tournament.WatchClub, "LAL-BAR")
	if !ok || !watched {
		t.Fatalf("club watch toggle failed: %+v", state)
	}

	tempDir := t.TempDir()
	savePath := filepath.Join(tempDir, "watchlist_career.json")
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
	if len(snap.Watchlist.Clubs) != 1 || snap.Watchlist.Clubs[0] != "LAL-BAR" {
		t.Fatalf("watchlist lost in save: %+v", snap.Watchlist)
	}

	_, freshGE, freshTM, freshTE := setupTestWorld(t)
	if err := RestoreCareer(freshTM, freshGE, freshTE, snap); err != nil {
		t.Fatalf("RestoreCareer failed: %v", err)
	}
	if !freshTM.IsWatched(tournament.WatchClub, "LAL-BAR") {
		t.Fatal("watchlist lost in restore")
	}
	// Toggling off after restore works (no deadlock, state consistent).
	if _, watched, _ := freshTM.ToggleWatchlist(tournament.WatchClub, "LAL-BAR"); watched {
		t.Fatal("post-restore toggle should remove the club")
	}
}

func TestValidationRejectsUnknownWatchlistEntries(t *testing.T) {
	_, ge, tm, te := setupTestWorld(t)
	snap := BuildSnapshot(tm, ge, te)
	snap.Watchlist.Clubs = []string{"NO_SUCH_CLUB"}
	if err := ValidateCareerSnapshot(snap); err == nil {
		t.Fatal("validation must reject a watchlist entry referencing an unknown club")
	}
	snap = BuildSnapshot(tm, ge, te)
	snap.Watchlist.Players = []string{"NO_SUCH_PLAYER"}
	if err := ValidateCareerSnapshot(snap); err == nil {
		t.Fatal("validation must reject a watchlist entry referencing an unknown player")
	}
}
