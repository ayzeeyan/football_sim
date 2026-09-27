package persistence

import (
	"path/filepath"
	"testing"

	"football_sim/pkg/tournament"
)

// The achievement ledger (SaveVersion 11) survives a save/load round trip
// and malformed ledgers are rejected by validation.
func TestAchievementLedgerSurvivesRoundTrip(t *testing.T) {
	_, ge, tm, te := setupTestWorld(t)

	tm.Achievements = append(tm.Achievements, tournament.Achievement{
		ID: "unbeaten_20", Kind: "club", Title: "The Unbeaten March",
		Description: "A club strung together 20 league matches without defeat.",
		SubjectID:   "LAL-RMA", SubjectName: "Real Madrid",
		Season: "2026-27", Matchweek: 20,
		UnlockKey: "unbeaten_20:LAL-RMA:20",
	})
	tm.AchievementsFired["unbeaten_20:LAL-RMA:20"] = true
	tm.ClubUnbeatenRuns["LAL-RMA"] = 20
	tm.YoungestScorer = &tournament.YoungestScorerRecord{
		PlayerID: "P00001", PlayerName: "Test Prodigy", ClubID: "LAL-RMA",
		Age: 16, Season: "2026-27", Matchweek: 7,
	}

	tempDir := t.TempDir()
	savePath := filepath.Join(tempDir, "achievements_career.json")
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
	if len(snap.Achievements) != 1 || snap.Achievements[0].UnlockKey != "unbeaten_20:LAL-RMA:20" {
		t.Fatalf("achievement lost in save: %+v", snap.Achievements)
	}
	if snap.ClubUnbeatenRuns["LAL-RMA"] != 20 {
		t.Fatalf("unbeaten run lost in save: %+v", snap.ClubUnbeatenRuns)
	}
	if snap.YoungestScorer == nil || snap.YoungestScorer.Age != 16 {
		t.Fatalf("youngest scorer lost in save: %+v", snap.YoungestScorer)
	}

	_, freshGE, freshTM, freshTE := setupTestWorld(t)
	if err := RestoreCareer(freshTM, freshGE, freshTE, snap); err != nil {
		t.Fatalf("RestoreCareer failed: %v", err)
	}
	ledger := freshTM.GetAchievements()
	if len(ledger) != 1 || ledger[0].ID != "unbeaten_20" {
		t.Fatalf("achievement lost in restore: %+v", ledger)
	}
	if freshTM.ClubUnbeatenRuns["LAL-RMA"] != 20 {
		t.Fatalf("unbeaten run lost in restore: %+v", freshTM.ClubUnbeatenRuns)
	}
	if rec := freshTM.GetYoungestScorerRecord(); rec == nil || rec.Age != 16 {
		t.Fatalf("youngest scorer lost in restore: %+v", rec)
	}
}

func TestValidationRejectsMalformedAchievements(t *testing.T) {
	_, ge, tm, te := setupTestWorld(t)

	withAchievement := func(a tournament.Achievement, fired bool) *CareerSnapshot {
		snap := BuildSnapshot(tm, ge, te)
		snap.Achievements = []tournament.Achievement{a}
		if fired {
			snap.AchievementsFired = map[string]bool{a.UnlockKey: true}
		}
		return snap
	}

	unknown := tournament.Achievement{ID: "no_such_achievement", UnlockKey: "k", SubjectID: "LAL-RMA"}
	if err := ValidateCareerSnapshot(withAchievement(unknown, true)); err == nil {
		t.Fatal("validation must reject unknown achievement ids")
	}
	unfired := tournament.Achievement{ID: "unbeaten_20", UnlockKey: "k", SubjectID: "LAL-RMA"}
	if err := ValidateCareerSnapshot(withAchievement(unfired, false)); err == nil {
		t.Fatal("validation must reject achievements that are not marked fired")
	}
	badClub := tournament.Achievement{ID: "unbeaten_20", UnlockKey: "k", Kind: "club", SubjectID: "NO_SUCH_CLUB"}
	if err := ValidateCareerSnapshot(withAchievement(badClub, true)); err == nil {
		t.Fatal("validation must reject achievements referencing unknown clubs")
	}
	badPlayer := tournament.Achievement{ID: "goals_30_season", UnlockKey: "k", Kind: "player", SubjectID: "NO_SUCH_PLAYER"}
	if err := ValidateCareerSnapshot(withAchievement(badPlayer, true)); err == nil {
		t.Fatal("validation must reject achievements referencing unknown players")
	}

	snap := BuildSnapshot(tm, ge, te)
	snap.ClubUnbeatenRuns = map[string]int{"LAL-RMA": -3}
	if err := ValidateCareerSnapshot(snap); err == nil {
		t.Fatal("validation must reject negative unbeaten runs")
	}

	snap = BuildSnapshot(tm, ge, te)
	snap.YoungestScorer = &tournament.YoungestScorerRecord{PlayerID: "NO_SUCH_PLAYER", Age: 15}
	if err := ValidateCareerSnapshot(snap); err == nil {
		t.Fatal("validation must reject a youngest-scorer record referencing an unknown player")
	}
}
