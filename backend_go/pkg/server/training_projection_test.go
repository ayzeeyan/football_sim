package server

import (
	"net/http"
	"testing"
)

// The training projection endpoint serves any player read-only; prodigies
// are flagged trainable so the planner can offer the interactive path.
func TestTrainingProjectionEndpoint(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	if payload := postNewCareer(t, ts.URL, false, nil); payload["status"] != "success" {
		t.Fatalf("fresh world career payload: %v", payload)
	}

	// Any squad player projects a staff plan.
	resp, err := http.Get(ts.URL + "/api/clubs")
	if err != nil {
		t.Fatalf("GET /api/clubs: %v", err)
	}
	var clubs []map[string]interface{}
	decodeJSONBody(t, resp, &clubs)
	if len(clubs) == 0 {
		t.Fatal("no clubs")
	}

	// Find a real squad player through the squad endpoint, skipping the
	// canonical prodigies (they project as trainable).
	resp, err = http.Get(ts.URL + "/api/prodigies")
	if err != nil {
		t.Fatalf("GET prodigies: %v", err)
	}
	var prodigies []map[string]interface{}
	decodeJSONBody(t, resp, &prodigies)
	prodigyIDs := make(map[string]bool, len(prodigies))
	for _, p := range prodigies {
		if id, _ := p["player_id"].(string); id != "" {
			prodigyIDs[id] = true
		}
	}

	firstClubID, _ := clubs[0]["club_id"].(string)
	resp, err = http.Get(ts.URL + "/api/clubs/" + firstClubID + "/squad")
	if err != nil {
		t.Fatalf("GET squad: %v", err)
	}
	var squad []map[string]interface{}
	decodeJSONBody(t, resp, &squad)
	if len(squad) == 0 {
		t.Fatal("no squad players")
	}
	var playerID string
	for _, row := range squad {
		id, _ := row["player_id"].(string)
		if !prodigyIDs[id] {
			playerID = id
			break
		}
	}
	if playerID == "" {
		t.Fatal("no non-prodigy squad player found")
	}

	resp, err = http.Get(ts.URL + "/api/training/projection/" + playerID)
	if err != nil {
		t.Fatalf("GET projection: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("projection status=%d", resp.StatusCode)
	}
	var proj map[string]interface{}
	decodeJSONBody(t, resp, &proj)
	if proj["player_id"] != playerID {
		t.Fatalf("projection player_id=%v want %v", proj["player_id"], playerID)
	}
	focus, _ := proj["focus"].(string)
	if focus != "hypertrophy" && focus != "technical" && focus != "tactical" {
		t.Fatalf("projection focus %q is not a known regimen", focus)
	}
	if proj["rationale"] == "" {
		t.Fatal("projection must include a rationale")
	}
	if trainable, _ := proj["trainable"].(bool); trainable {
		t.Fatal("a non-prodigy squad player must not be trainable")
	}

	// Unknown players 404.
	resp, err = http.Get(ts.URL + "/api/training/projection/NO_SUCH_PLAYER")
	if err != nil {
		t.Fatalf("GET unknown projection: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown player projection status=%d, want 404", resp.StatusCode)
	}
}
