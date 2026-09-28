package persistence

import (
	"path/filepath"
	"testing"

	"football_sim/pkg/tournament"
)

// The current season's promotion/relegation moves survive a save/load
// round trip and malformed records are rejected (SaveVersion 16).
func TestSeasonLeagueMovesSurviveRoundTrip(t *testing.T) {
	ge, tm, te := setupEuropeanWorld(t)
	clubID := tm.ClubsList[0].ClubID
	tm.SeasonLeagueMoves = []tournament.RelegationMove{{
		ClubID: clubID, ClubName: tm.ClubsList[0].ClubName,
		FromLeague: "Premier League", ToLeague: "La Liga", Direction: "relegated",
	}}

	tempDir := t.TempDir()
	savePath := filepath.Join(tempDir, "league_moves_career.json")
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
	if len(snap.SeasonLeagueMoves) != 1 || snap.SeasonLeagueMoves[0].ClubID != clubID {
		t.Fatalf("league moves lost in save: %+v", snap.SeasonLeagueMoves)
	}

	freshGE, freshTM, freshTE := setupEuropeanWorld(t)
	if err := RestoreCareer(freshTM, freshGE, freshTE, snap); err != nil {
		t.Fatalf("RestoreCareer failed: %v", err)
	}
	if len(freshTM.SeasonLeagueMoves) != 1 || freshTM.SeasonLeagueMoves[0].ToLeague != "La Liga" {
		t.Fatalf("league moves lost in restore: %+v", freshTM.SeasonLeagueMoves)
	}
}

func TestValidationRejectsMalformedSeasonLeagueMoves(t *testing.T) {
	ge, tm, te := setupEuropeanWorld(t)
	clubID := tm.ClubsList[0].ClubID

	snap := BuildSnapshot(tm, ge, te)
	snap.SeasonLeagueMoves = []tournament.RelegationMove{{
		ClubID: "NO_SUCH_CLUB", FromLeague: "A", ToLeague: "B", Direction: "relegated",
	}}
	if err := ValidateCareerSnapshot(snap); err == nil {
		t.Fatal("validation must reject a league move referencing an unknown club")
	}

	snap = BuildSnapshot(tm, ge, te)
	snap.SeasonLeagueMoves = []tournament.RelegationMove{{
		ClubID: clubID, FromLeague: "A", ToLeague: "B", Direction: "sideways",
	}}
	if err := ValidateCareerSnapshot(snap); err == nil {
		t.Fatal("validation must reject an unknown move direction")
	}

	snap = BuildSnapshot(tm, ge, te)
	snap.SeasonLeagueMoves = []tournament.RelegationMove{{
		ClubID: clubID, FromLeague: "", ToLeague: "B", Direction: "promoted",
	}}
	if err := ValidateCareerSnapshot(snap); err == nil {
		t.Fatal("validation must reject a move with an empty league")
	}

	snap = BuildSnapshot(tm, ge, te)
	snap.SeasonLeagueMoves = []tournament.RelegationMove{{
		ClubID: clubID, FromLeague: "Premier League", ToLeague: "La Liga", Direction: "relegated",
	}}
	if err := ValidateCareerSnapshot(snap); err != nil {
		t.Fatalf("validation must accept a well-formed move: %v", err)
	}
}
