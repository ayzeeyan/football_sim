package persistence

import (
	"path/filepath"
	"testing"

	"football_sim/pkg/medical"
)

// Injury history (SaveVersion 12) survives a save/load round trip and
// malformed ledgers are rejected by validation.
func TestInjuryHistorySurvivesRoundTrip(t *testing.T) {
	_, ge, tm, te := setupTestWorld(t)

	club := tm.ClubsList[0]
	var player = club.Squad[0]
	player.InjuryHistory = medical.AppendHistory(nil, medical.Record{
		Season: "2026-27", Matchweek: 5,
		Kind: "hamstring tear", Severity: medical.SeverityModerate, MatchesOut: 7,
		FixtureID: "FX-1",
	})

	tempDir := t.TempDir()
	savePath := filepath.Join(tempDir, "injury_career.json")
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
	restored := snap.Clubs[club.ClubID].Squad[0]
	if len(restored.InjuryHistory) != 1 || restored.InjuryHistory[0].Kind != "hamstring tear" {
		t.Fatalf("injury history lost in save: %+v", restored.InjuryHistory)
	}

	_, freshGE, freshTM, freshTE := setupTestWorld(t)
	if err := RestoreCareer(freshTM, freshGE, freshTE, snap); err != nil {
		t.Fatalf("RestoreCareer failed: %v", err)
	}
	if got := freshTM.Clubs[club.ClubID].Squad[0].InjuryHistory; len(got) != 1 || got[0].MatchesOut != 7 {
		t.Fatalf("injury history lost in restore: %+v", got)
	}
}

func TestValidationRejectsMalformedInjuryHistory(t *testing.T) {
	_, ge, tm, te := setupTestWorld(t)

	withRecord := func(rec medical.Record) *CareerSnapshot {
		snap := BuildSnapshot(tm, ge, te)
		club := snap.Clubs[tm.ClubsList[0].ClubID]
		club.Squad[0].InjuryHistory = []medical.Record{rec}
		return snap
	}

	bad := medical.Record{Season: "2026-27", Matchweek: 5, Kind: "mystery", Severity: "catastrophic", MatchesOut: 3}
	if err := ValidateCareerSnapshot(withRecord(bad)); err == nil {
		t.Fatal("validation must reject unknown injury severities")
	}
	negative := medical.Record{Season: "2026-27", Matchweek: 5, Kind: "knock", Severity: medical.SeverityMinor, MatchesOut: 0}
	if err := ValidateCareerSnapshot(withRecord(negative)); err == nil {
		t.Fatal("validation must reject zero-match injuries")
	}
	huge := medical.Record{Season: "2026-27", Matchweek: 5, Kind: "knock", Severity: medical.SeverityMinor, MatchesOut: 99}
	if err := ValidateCareerSnapshot(withRecord(huge)); err == nil {
		t.Fatal("validation must reject out-of-bounds layoffs")
	}
	empty := medical.Record{Season: "2026-27", Matchweek: 5, Kind: "", Severity: medical.SeverityMinor, MatchesOut: 2}
	if err := ValidateCareerSnapshot(withRecord(empty)); err == nil {
		t.Fatal("validation must reject injuries without a kind")
	}

	// Over-cap history is rejected.
	snap := BuildSnapshot(tm, ge, te)
	club := snap.Clubs[tm.ClubsList[0].ClubID]
	history := make([]medical.Record, medical.MaxHistoryPerPlayer+1)
	for i := range history {
		history[i] = medical.Record{Season: "2026-27", Matchweek: i + 1, Kind: "knock", Severity: medical.SeverityMinor, MatchesOut: 1}
	}
	club.Squad[0].InjuryHistory = history
	if err := ValidateCareerSnapshot(snap); err == nil {
		t.Fatal("validation must reject injury history above the cap")
	}
}
