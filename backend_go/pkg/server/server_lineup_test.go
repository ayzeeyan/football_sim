package server

import (
	"bytes"
	"encoding/json"
	"football_sim/pkg/models"
	"net/http"
	"testing"
)

// Tier B lineup control: the viewer sets and clears a club's starting XI
// through the rigid-slot contract, and the effective XI reflects it.
// Save/load persistence of the override is covered by the persistence
// package's round-trip test.
func TestLineupOverrideEndpoints(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	if payload := postNewCareer(t, ts.URL, false, nil); payload["status"] != "success" {
		t.Fatalf("fresh world career payload: %v", payload)
	}

	clubs := []struct {
		ClubID string `json:"club_id"`
	}{}
	resp, err := http.Get(ts.URL + "/api/clubs")
	if err != nil {
		t.Fatalf("GET /api/clubs: %v", err)
	}
	decodeJSONBody(t, resp, &clubs)
	clubID := clubs[0].ClubID

	// Read the current XI to build a valid override.
	resp, err = http.Get(ts.URL + "/api/clubs/" + clubID + "/xi")
	if err != nil {
		t.Fatalf("GET xi: %v", err)
	}
	var xi []struct {
		PlayerID     string `json:"player_id"`
		StartingSlot string `json:"starting_slot"`
	}
	decodeJSONBody(t, resp, &xi)
	if len(xi) != 11 {
		t.Fatalf("auto XI must have 11 starters, got %d", len(xi))
	}
	players := map[string]string{}
	for _, row := range xi {
		players[row.StartingSlot] = row.PlayerID
	}

	// The override must use the formation the XI was picked for: match
	// the observed slot set against the known formations.
	observed := map[string]bool{}
	for _, row := range xi {
		observed[row.StartingSlot] = true
	}
	formation := ""
	for _, candidate := range []string{models.Formation433, models.Formation433Attack, models.Formation4231, models.Formation442} {
		slots := models.FormationSlots(candidate)
		if len(slots) != len(observed) {
			continue
		}
		match := true
		for _, slot := range slots {
			if !observed[slot] {
				match = false
				break
			}
		}
		if match {
			formation = candidate
			break
		}
	}
	if formation == "" {
		t.Fatalf("could not derive the XI formation from slots %v", observed)
	}

	// Set the override.
	body, _ := json.Marshal(map[string]interface{}{"formation": formation, "players": players})
	resp, err = http.Post(ts.URL+"/api/clubs/"+clubID+"/lineup", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST lineup: %v", err)
	}
	var set map[string]interface{}
	decodeJSONBody(t, resp, &set)
	if resp.StatusCode != http.StatusOK || set["status"] != "success" {
		t.Fatalf("set lineup failed: %d %v", resp.StatusCode, set)
	}

	// The XI endpoint now reflects the override.
	resp, err = http.Get(ts.URL + "/api/clubs/" + clubID + "/xi")
	if err != nil {
		t.Fatalf("GET xi after set: %v", err)
	}
	var after []struct {
		PlayerID     string `json:"player_id"`
		StartingSlot string `json:"starting_slot"`
	}
	decodeJSONBody(t, resp, &after)
	if len(after) != 11 {
		t.Fatalf("override XI must have 11 starters, got %d", len(after))
	}
	for _, row := range after {
		if players[row.StartingSlot] != row.PlayerID {
			t.Fatalf("override not honoured at %s: want %s got %s", row.StartingSlot, players[row.StartingSlot], row.PlayerID)
		}
	}

	// An invalid override is rejected without touching the stored one.
	bad, _ := json.Marshal(map[string]interface{}{"formation": formation, "players": map[string]string{"GK": "NO-SUCH-PLAYER"}})
	resp, err = http.Post(ts.URL+"/api/clubs/"+clubID+"/lineup", "application/json", bytes.NewReader(bad))
	if err != nil {
		t.Fatalf("POST bad lineup: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid lineup status=%d want 400", resp.StatusCode)
	}

	// The stored override is intact after the rejected request.
	resp, err = http.Get(ts.URL + "/api/clubs/" + clubID + "/xi")
	if err != nil {
		t.Fatalf("GET xi after bad set: %v", err)
	}
	var intact []struct {
		PlayerID     string `json:"player_id"`
		StartingSlot string `json:"starting_slot"`
	}
	decodeJSONBody(t, resp, &intact)
	for _, row := range intact {
		if players[row.StartingSlot] != row.PlayerID {
			t.Fatalf("rejected request must not change the stored override at %s", row.StartingSlot)
		}
	}

	// Clear returns the club to AI selection.
	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/clubs/"+clubID+"/lineup", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE lineup: %v", err)
	}
	var cleared map[string]interface{}
	decodeJSONBody(t, resp, &cleared)
	if resp.StatusCode != http.StatusOK || cleared["status"] != "success" {
		t.Fatalf("clear lineup failed: %d %v", resp.StatusCode, cleared)
	}
	if cleared["override"] != nil {
		t.Fatal("cleared override must be nil")
	}

	// Unknown club is a clean 404.
	req, _ = http.NewRequest(http.MethodDelete, ts.URL+"/api/clubs/NO-SUCH-CLUB/lineup", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE unknown club lineup: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown club status=%d want 404", resp.StatusCode)
	}
}
