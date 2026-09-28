package tournament

import (
	"testing"
)

// Tier B national squad control (B6): the viewer picks one nation's squad
// from the eligible pool; the AI selection stays the default elsewhere.

// viewerSquadTestPool builds a legal 23-player selection from a nation's
// eligible pool: the AI-selected squad is itself a valid selection.
func viewerSquadTestSelection(tm *TournamentManager, teamID string) []string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	team := tm.World.NationalTeams.Teams[teamID]
	if team == nil {
		return nil
	}
	return append([]string(nil), team.PlayerIDs...)
}

func TestSetNationalSquadAppliesViewerSelection(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	competition := tm.World.NationalTeams
	teamID := competition.TeamOrder[0]
	before := competition.Teams[teamID].Rating

	// The AI squad is a legal selection; swap two eligible players in.
	selection := viewerSquadTestSelection(tm, teamID)
	msg, team := tm.SetNationalSquad(teamID, selection)
	if msg != "" || team == nil {
		t.Fatalf("SetNationalSquad failed: %v", msg)
	}
	if len(competition.Teams[teamID].PlayerIDs) != NationalSquadSize {
		t.Fatalf("squad size=%d want %d", len(competition.Teams[teamID].PlayerIDs), NationalSquadSize)
	}
	if len(competition.ViewerSquads[teamID]) != NationalSquadSize {
		t.Fatal("viewer selection not persisted on the competition")
	}
	if competition.Teams[teamID].Rating != before {
		t.Fatalf("rating changed for an identical selection: %d -> %d", before, competition.Teams[teamID].Rating)
	}

	payload := tm.NationalSquadSelection(teamID)
	if payload == nil {
		t.Fatal("selection payload missing")
	}
	if payload["viewer_selected"] != true {
		t.Fatal("payload must report viewer_selected=true")
	}
	pool, _ := payload["eligible_pool"].([]map[string]interface{})
	if len(pool) < NationalSquadSize {
		t.Fatalf("eligible pool=%d want at least %d", len(pool), NationalSquadSize)
	}
	selected := 0
	for _, row := range pool {
		if row["selected"] == true {
			selected++
		}
	}
	if selected != NationalSquadSize {
		t.Fatalf("selected flags=%d want %d", selected, NationalSquadSize)
	}
}

func TestSetNationalSquadRejectsBrokenSelections(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	competition := tm.World.NationalTeams
	teamID := competition.TeamOrder[0]
	selection := viewerSquadTestSelection(tm, teamID)

	// Wrong size.
	if msg, _ := tm.SetNationalSquad(teamID, selection[:20]); msg == "" {
		t.Fatal("short selection must be rejected")
	}
	// Duplicate.
	dup := append([]string(nil), selection...)
	dup[22] = dup[0]
	if msg, _ := tm.SetNationalSquad(teamID, dup); msg == "" {
		t.Fatal("duplicate selection must be rejected")
	}
	// No goalkeeper: strip every GK by swapping in field players from the pool.
	payload := tm.NationalSquadSelection(teamID)
	pool, _ := payload["eligible_pool"].([]map[string]interface{})
	noGK := make([]string, 0, NationalSquadSize)
	for _, row := range pool {
		if row["category"] == "GK" {
			continue
		}
		if len(noGK) < NationalSquadSize {
			noGK = append(noGK, row["player_id"].(string))
		}
	}
	if len(noGK) == NationalSquadSize {
		if msg, _ := tm.SetNationalSquad(teamID, noGK); msg == "" {
			t.Fatal("goalkeeper-less selection must be rejected")
		}
	}
	// Ineligible player: a player from another nation's squad.
	other := competition.Teams[competition.TeamOrder[1]]
	foreign := append([]string(nil), selection...)
	foreign[22] = other.PlayerIDs[0]
	if msg, _ := tm.SetNationalSquad(teamID, foreign); msg == "" {
		t.Fatal("ineligible player must be rejected")
	}
	// Unknown team.
	if msg, _ := tm.SetNationalSquad("no-such-nation", selection); msg == "" {
		t.Fatal("unknown team must be rejected")
	}
	// Nothing was applied.
	if len(competition.ViewerSquads) != 0 {
		t.Fatalf("rejected selections must not persist: %+v", competition.ViewerSquads)
	}
}

func TestClearNationalSquadRestoresAISelection(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	competition := tm.World.NationalTeams
	teamID := competition.TeamOrder[0]
	aiSquad := viewerSquadTestSelection(tm, teamID)

	// Reorder the selection so the viewer squad differs from the AI order.
	reordered := append([]string(nil), aiSquad...)
	reordered[0], reordered[1] = reordered[1], reordered[0]
	if msg, _ := tm.SetNationalSquad(teamID, reordered); msg != "" {
		t.Fatalf("SetNationalSquad failed: %v", msg)
	}
	if msg, _ := tm.ClearNationalSquad(teamID); msg != "" {
		t.Fatalf("ClearNationalSquad failed: %v", msg)
	}
	if len(competition.ViewerSquads) != 0 {
		t.Fatal("clear must remove the viewer selection")
	}
	restored := competition.Teams[teamID].PlayerIDs
	if len(restored) != NationalSquadSize {
		t.Fatalf("restored squad size=%d", len(restored))
	}
	for i := range aiSquad {
		if restored[i] != aiSquad[i] {
			t.Fatalf("AI selection not restored at %d: %q want %q", i, restored[i], aiSquad[i])
		}
	}
	// Clearing again is a clean refusal.
	if msg, _ := tm.ClearNationalSquad(teamID); msg == "" {
		t.Fatal("clearing an AI-selected squad must fail")
	}
}

func TestViewerSquadSurvivesSeasonRebuild(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	competition := tm.World.NationalTeams
	teamID := competition.TeamOrder[0]
	selection := viewerSquadTestSelection(tm, teamID)
	// Reverse the order so the re-applied squad is distinguishable.
	reversed := make([]string, 0, len(selection))
	for i := len(selection) - 1; i >= 0; i-- {
		reversed = append(reversed, selection[i])
	}
	if msg, _ := tm.SetNationalSquad(teamID, reversed); msg != "" {
		t.Fatalf("SetNationalSquad failed: %v", msg)
	}

	// A season transition rebuilds the competition through the same path.
	tm.mu.Lock()
	tm.initializeNationalTeamsUnlocked()
	tm.mu.Unlock()

	competition = tm.World.NationalTeams
	if len(competition.ViewerSquads[teamID]) != NationalSquadSize {
		t.Fatal("viewer squad lost across the season rebuild")
	}
	got := competition.Teams[teamID].PlayerIDs
	for i := range reversed {
		if got[i] != reversed[i] {
			t.Fatalf("viewer squad not re-applied at %d: %q want %q", i, got[i], reversed[i])
		}
	}
}

func TestViewerSquadFallsBackToAIWhenBrokenAtRebuild(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	competition := tm.World.NationalTeams
	teamID := competition.TeamOrder[0]
	aiSquad := viewerSquadTestSelection(tm, teamID)

	// Simulate a stale selection: a player who has left the world.
	tm.mu.Lock()
	competition.ViewerSquads = map[string][]string{teamID: append(append([]string(nil), aiSquad[:22]...), "RETIRED-PLAYER")}
	tm.mu.Unlock()

	tm.mu.Lock()
	tm.initializeNationalTeamsUnlocked()
	tm.mu.Unlock()

	competition = tm.World.NationalTeams
	if len(competition.ViewerSquads) != 0 {
		t.Fatal("stale viewer squad must be discarded at the rebuild")
	}
	restored := competition.Teams[teamID].PlayerIDs
	for i := range aiSquad {
		if restored[i] != aiSquad[i] {
			t.Fatalf("AI fallback not applied at %d: %q want %q", i, restored[i], aiSquad[i])
		}
	}
}
