package persistence

import (
	"path/filepath"
	"testing"

	"football_sim/pkg/datamanager"
	"football_sim/pkg/growth"
	"football_sim/pkg/tournament"
	"football_sim/pkg/transfers"
)

// setupEuropeanWorld builds a full five-league world so the national teams
// competition exists, mirroring the tournament package's test loader.
func setupEuropeanWorld(t *testing.T) (*growth.GrowthEngine, *tournament.TournamentManager, *transfers.TransferEngine) {
	t.Helper()
	ge := growth.NewGrowthEngine(8181)
	dm := datamanager.NewDataManager(filepath.Join("..", "..", "..", "dataset.json"), ge)
	if len(dm.ClubsList) != 96 {
		t.Fatalf("dataset clubs=%d want 96", len(dm.ClubsList))
	}
	tm := tournament.NewEuropeanWorldManager(dm.ClubsList, ge, 8181)
	te := transfers.NewTransferEngine(dm.ClubsList, tm.Managers, 9191)
	tm.TransferEngine = te
	return ge, tm, te
}

// firstNationalTeam returns a deterministic team ID and its AI squad.
func firstNationalTeam(tm *tournament.TournamentManager) (string, []string) {
	competition := tm.World.NationalTeams
	teamID := competition.TeamOrder[0]
	team := competition.Teams[teamID]
	return teamID, append([]string(nil), team.PlayerIDs...)
}

// The viewer's national squad selection survives a save/load round trip
// (SaveVersion 15).
func TestNationalSquadSelectionSurvivesRoundTrip(t *testing.T) {
	ge, tm, te := setupEuropeanWorld(t)
	teamID, squad := firstNationalTeam(tm)
	if msg, _ := tm.SetNationalSquad(teamID, squad); msg != "" {
		t.Fatalf("SetNationalSquad failed: %v", msg)
	}

	tempDir := t.TempDir()
	savePath := filepath.Join(tempDir, "nations_career.json")
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
	if snap.World == nil || snap.World.NationalTeams == nil {
		t.Fatal("national teams competition lost in save")
	}
	if len(snap.World.NationalTeams.ViewerSquads[teamID]) != tournament.NationalSquadSize {
		t.Fatalf("viewer squad lost in save: %+v", snap.World.NationalTeams.ViewerSquads)
	}

	freshGE, freshTM, freshTE := setupEuropeanWorld(t)
	if err := RestoreCareer(freshTM, freshGE, freshTE, snap); err != nil {
		t.Fatalf("RestoreCareer failed: %v", err)
	}
	restored := freshTM.World.NationalTeams
	if len(restored.ViewerSquads[teamID]) != tournament.NationalSquadSize {
		t.Fatal("viewer squad lost in restore")
	}
	got := restored.Teams[teamID].PlayerIDs
	for i := range squad {
		if got[i] != squad[i] {
			t.Fatalf("restored squad mismatch at %d: %q want %q", i, got[i], squad[i])
		}
	}
	// Clearing after restore works.
	if msg, _ := freshTM.ClearNationalSquad(teamID); msg != "" {
		t.Fatalf("post-restore clear failed: %v", msg)
	}
}

func TestValidationRejectsMalformedViewerSquads(t *testing.T) {
	ge, tm, te := setupEuropeanWorld(t)
	teamID, squad := firstNationalTeam(tm)
	otherTeamID := tm.World.NationalTeams.TeamOrder[1]
	otherSquad := append([]string(nil), tm.World.NationalTeams.Teams[otherTeamID].PlayerIDs...)

	mutate := func(fn func(snap *CareerSnapshot)) error {
		snap := BuildSnapshot(tm, ge, te)
		fn(snap)
		return ValidateCareerSnapshot(snap)
	}

	// Wrong size.
	if err := mutate(func(snap *CareerSnapshot) {
		snap.World.NationalTeams.ViewerSquads = map[string][]string{teamID: squad[:20]}
	}); err == nil {
		t.Fatal("validation must reject an undersized viewer squad")
	}
	// Unknown team.
	if err := mutate(func(snap *CareerSnapshot) {
		snap.World.NationalTeams.ViewerSquads = map[string][]string{"no-such-nation": squad}
	}); err == nil {
		t.Fatal("validation must reject an unknown national team")
	}
	// Ineligible player.
	if err := mutate(func(snap *CareerSnapshot) {
		foreign := append([]string(nil), squad...)
		foreign[22] = otherSquad[0]
		snap.World.NationalTeams.ViewerSquads = map[string][]string{teamID: foreign}
	}); err == nil {
		t.Fatal("validation must reject an ineligible player")
	}
	// Duplicate.
	if err := mutate(func(snap *CareerSnapshot) {
		dup := append([]string(nil), squad...)
		dup[22] = dup[0]
		snap.World.NationalTeams.ViewerSquads = map[string][]string{teamID: dup}
	}); err == nil {
		t.Fatal("validation must reject a duplicate player")
	}
	// No goalkeeper: 23 eligible field players.
	if err := mutate(func(snap *CareerSnapshot) {
		noGK := make([]string, 0, tournament.NationalSquadSize)
		payload := tm.NationalSquadSelection(teamID)
		pool, _ := payload["eligible_pool"].([]map[string]interface{})
		for _, row := range pool {
			if row["category"] == "GK" {
				continue
			}
			if len(noGK) < tournament.NationalSquadSize {
				noGK = append(noGK, row["player_id"].(string))
			}
		}
		if len(noGK) == tournament.NationalSquadSize {
			snap.World.NationalTeams.ViewerSquads = map[string][]string{teamID: noGK}
		}
	}); err == nil {
		t.Fatal("validation must reject a goalkeeper-less squad")
	}
	// A well-formed selection passes.
	if err := mutate(func(snap *CareerSnapshot) {
		snap.World.NationalTeams.ViewerSquads = map[string][]string{teamID: squad}
	}); err != nil {
		t.Fatalf("validation must accept a legal viewer squad: %v", err)
	}
}
