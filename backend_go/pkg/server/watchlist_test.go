package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

// The watchlist endpoints toggle entities and persist the change.
func TestWatchlistEndpoints(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	if payload := postNewCareer(t, ts.URL, false, nil); payload["status"] != "success" {
		t.Fatalf("fresh world career payload: %v", payload)
	}

	resp, err := http.Get(ts.URL + "/api/clubs")
	if err != nil {
		t.Fatalf("GET /api/clubs: %v", err)
	}
	var clubs []map[string]interface{}
	decodeJSONBody(t, resp, &clubs)
	clubID, _ := clubs[0]["club_id"].(string)

	// Empty watchlist.
	resp, err = http.Get(ts.URL + "/api/watchlist")
	if err != nil {
		t.Fatalf("GET /api/watchlist: %v", err)
	}
	var list map[string]interface{}
	decodeJSONBody(t, resp, &list)
	if len(list["clubs"].([]interface{})) != 0 {
		t.Fatalf("fresh watchlist should be empty: %v", list)
	}

	// Toggle a club on.
	body, _ := json.Marshal(map[string]string{"entity": "club", "id": clubID})
	resp, err = http.Post(ts.URL+"/api/watchlist", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /api/watchlist: %v", err)
	}
	var toggle map[string]interface{}
	decodeJSONBody(t, resp, &toggle)
	if resp.StatusCode != http.StatusOK || toggle["watched"] != true {
		t.Fatalf("toggle-on failed: status=%d payload=%v", resp.StatusCode, toggle)
	}

	// The list now shows the club with display names.
	resp, err = http.Get(ts.URL + "/api/watchlist")
	if err != nil {
		t.Fatalf("GET /api/watchlist: %v", err)
	}
	decodeJSONBody(t, resp, &list)
	entries := list["clubs"].([]interface{})
	if len(entries) != 1 {
		t.Fatalf("watchlist should hold one club: %v", list)
	}
	entry := entries[0].(map[string]interface{})
	if entry["id"] != clubID || entry["name"] == "" {
		t.Fatalf("watchlist entry missing display data: %v", entry)
	}

	// Toggle off.
	resp, err = http.Post(ts.URL+"/api/watchlist", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /api/watchlist (off): %v", err)
	}
	decodeJSONBody(t, resp, &toggle)
	if toggle["watched"] != false {
		t.Fatalf("toggle-off failed: %v", toggle)
	}

	// Unknown entities are rejected.
	bad, _ := json.Marshal(map[string]string{"entity": "club", "id": "NO_SUCH_CLUB"})
	resp, err = http.Post(ts.URL+"/api/watchlist", "application/json", bytes.NewReader(bad))
	if err != nil {
		t.Fatalf("POST bad watchlist: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown entity status=%d, want 400", resp.StatusCode)
	}
}
