package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

// Tier B national squad control (B6): the viewer picks one nation's squad
// from the eligible pool; eligibility and the 23-player contract are
// enforced server-side.
func TestNationsSquadEndpoints(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	if payload := postNewCareer(t, ts.URL, false, nil); payload["status"] != "success" {
		t.Fatalf("fresh world career payload: %v", payload)
	}

	// Discover a team and its AI squad from the competition payload.
	resp, err := http.Get(ts.URL + "/api/competitions/nations-cup")
	if err != nil {
		t.Fatalf("GET nations-cup: %v", err)
	}
	var nations struct {
		Participants []struct {
			ID      string `json:"id"`
			Players []struct {
				PlayerID string `json:"player_id"`
			} `json:"players"`
		} `json:"participants"`
	}
	decodeJSONBody(t, resp, &nations)
	if len(nations.Participants) == 0 {
		t.Fatal("no national teams in the fresh world")
	}
	teamID := nations.Participants[0].ID
	squad := make([]string, 0, len(nations.Participants[0].Players))
	for _, player := range nations.Participants[0].Players {
		squad = append(squad, player.PlayerID)
	}

	// Unknown team is a clean 404.
	resp, err = http.Get(ts.URL + "/api/competitions/nations-cup/teams/no-such-nation/squad")
	if err != nil {
		t.Fatalf("GET unknown squad: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown team status=%d want 404", resp.StatusCode)
	}

	// The squad payload lists the eligible pool with selection flags.
	resp, err = http.Get(ts.URL + "/api/competitions/nations-cup/teams/" + teamID + "/squad")
	if err != nil {
		t.Fatalf("GET squad: %v", err)
	}
	var selection struct {
		ViewerSelected bool `json:"viewer_selected"`
		SquadSize      int  `json:"squad_size"`
		EligiblePool   []struct {
			PlayerID string `json:"player_id"`
			Selected bool   `json:"selected"`
		} `json:"eligible_pool"`
	}
	decodeJSONBody(t, resp, &selection)
	if selection.ViewerSelected || selection.SquadSize != 23 {
		t.Fatalf("fresh squad must be AI-selected: %+v", selection)
	}
	if len(selection.EligiblePool) < 23 {
		t.Fatalf("eligible pool=%d want at least 23", len(selection.EligiblePool))
	}

	post := func(body interface{}) (map[string]interface{}, int) {
		t.Helper()
		raw, _ := json.Marshal(body)
		resp, err := http.Post(ts.URL+"/api/competitions/nations-cup/teams/"+teamID+"/squad", "application/json", bytes.NewReader(raw))
		if err != nil {
			t.Fatalf("POST squad: %v", err)
		}
		defer resp.Body.Close()
		var out map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			t.Fatalf("decode squad response: %v", err)
		}
		return out, resp.StatusCode
	}

	// An undersized selection is refused.
	if _, code := post(map[string]interface{}{"player_ids": squad[:20]}); code != http.StatusBadRequest {
		t.Fatalf("undersized selection status=%d want 400", code)
	}

	// A legal selection is accepted.
	out, code := post(map[string]interface{}{"player_ids": squad})
	if code != http.StatusOK || out["status"] != "success" {
		t.Fatalf("legal selection failed: %d %v", code, out)
	}

	// The squad payload now reports the viewer selection.
	resp, err = http.Get(ts.URL + "/api/competitions/nations-cup/teams/" + teamID + "/squad")
	if err != nil {
		t.Fatalf("GET squad after set: %v", err)
	}
	decodeJSONBody(t, resp, &selection)
	if !selection.ViewerSelected {
		t.Fatal("squad must report viewer_selected=true after setting")
	}
	selected := 0
	for _, row := range selection.EligiblePool {
		if row.Selected {
			selected++
		}
	}
	if selected != 23 {
		t.Fatalf("selected flags=%d want 23", selected)
	}

	// Clearing returns the nation to the AI selection.
	req, err := http.NewRequest(http.MethodDelete, ts.URL+"/api/competitions/nations-cup/teams/"+teamID+"/squad", nil)
	if err != nil {
		t.Fatalf("build DELETE: %v", err)
	}
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE squad: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("DELETE squad status=%d want 200", resp.StatusCode)
	}

	resp, err = http.Get(ts.URL + "/api/competitions/nations-cup/teams/" + teamID + "/squad")
	if err != nil {
		t.Fatalf("GET squad after clear: %v", err)
	}
	decodeJSONBody(t, resp, &selection)
	if selection.ViewerSelected {
		t.Fatal("squad must be AI-selected after clearing")
	}
}
