package server

import (
	"encoding/json"
	"fmt"
	"testing"
)

func assertWireXIHasExplicitSlots(t *testing.T, raw interface{}, label string) {
	t.Helper()
	rows, ok := raw.([]map[string]interface{})
	if !ok {
		t.Fatalf("%s has type %T, want []map[string]interface{}", label, raw)
	}
	if len(rows) != 11 {
		t.Fatalf("%s has %d players, want 11", label, len(rows))
	}
	seen := make(map[string]bool, len(rows))
	for _, row := range rows {
		slot := fmt.Sprint(row["tactical_slot"])
		natural := fmt.Sprint(row["natural_position"])
		fit := fmt.Sprint(row["position_fit"])
		if slot == "" || natural == "" || fit == "" {
			t.Fatalf("%s row lacks tactical metadata: %#v", label, row)
		}
		if seen[slot] {
			t.Fatalf("%s repeats tactical slot %s", label, slot)
		}
		seen[slot] = true
	}
}

func TestPreviewAndLiveTickExposeExplicitTacticalSlotContract(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	srv.worldMu.Lock()
	fixture := &srv.TournamentManager.Fixtures[0]
	home := srv.TournamentManager.Clubs[fixture.HomeID]
	away := srv.TournamentManager.Clubs[fixture.AwayID]
	preview := srv.fixturePreview(fixture, home, away)
	srv.worldMu.Unlock()
	if preview["home_formation"] == "" || preview["away_formation"] == "" {
		t.Fatalf("preview omitted formation: %#v", preview)
	}
	assertWireXIHasExplicitSlots(t, preview["home_xi"], "preview home XI")
	assertWireXIHasExplicitSlots(t, preview["away_xi"], "preview away XI")

	srv.worldMu.Lock()
	srv.LiveMatchEngine.SetClubs(home, away, srv.TournamentManager.Managers[home.ClubID], srv.TournamentManager.Managers[away.ClubID])
	payload := srv.buildMatchTickPayload()
	srv.worldMu.Unlock()

	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal live tick: %v", err)
	}
	var tick struct {
		HomeFormation string `json:"home_formation"`
		AwayFormation string `json:"away_formation"`
		HomeCoords    []struct {
			Player struct {
				PlayerID        string `json:"player_id"`
				Position        string `json:"position"`
				NaturalPosition string `json:"natural_position"`
				TacticalSlot    string `json:"tactical_slot"`
				PositionFit     string `json:"position_fit"`
			} `json:"player"`
		} `json:"home_coords"`
		AwayCoords []struct {
			Player struct {
				PlayerID        string `json:"player_id"`
				Position        string `json:"position"`
				NaturalPosition string `json:"natural_position"`
				TacticalSlot    string `json:"tactical_slot"`
				PositionFit     string `json:"position_fit"`
			} `json:"player"`
		} `json:"away_coords"`
	}
	if err := json.Unmarshal(encoded, &tick); err != nil {
		t.Fatalf("decode live tick: %v", err)
	}
	if tick.HomeFormation == "" || tick.AwayFormation == "" {
		t.Fatalf("live tick omitted formation: %q/%q", tick.HomeFormation, tick.AwayFormation)
	}
	assertActors := func(label string, actors []struct {
		Player struct {
			PlayerID        string `json:"player_id"`
			Position        string `json:"position"`
			NaturalPosition string `json:"natural_position"`
			TacticalSlot    string `json:"tactical_slot"`
			PositionFit     string `json:"position_fit"`
		} `json:"player"`
	}) {
		t.Helper()
		if len(actors) != 11 {
			t.Fatalf("%s has %d actors, want 11", label, len(actors))
		}
		seen := make(map[string]bool, len(actors))
		for _, actor := range actors {
			p := actor.Player
			if p.PlayerID == "" || p.Position == "" || p.NaturalPosition != p.Position || p.TacticalSlot == "" || p.PositionFit == "" {
				t.Fatalf("%s actor lacks separated natural/tactical metadata: %+v", label, p)
			}
			if seen[p.TacticalSlot] {
				t.Fatalf("%s repeats slot %s", label, p.TacticalSlot)
			}
			seen[p.TacticalSlot] = true
		}
	}
	assertActors("home", tick.HomeCoords)
	assertActors("away", tick.AwayCoords)
}
