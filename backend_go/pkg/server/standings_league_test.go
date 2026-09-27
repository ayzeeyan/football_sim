package server

import (
	"net/http"
	"testing"
)

// The ?league= selector on the standings endpoint picks one domestic league
// table; unknown selectors fall back to the default compatibility view.
func TestStandingsLeagueSelector(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	if payload := postNewCareer(t, ts.URL, false, nil); payload["status"] != "success" {
		t.Fatalf("fresh world career payload: %v", payload)
	}

	getTable := func(query string) []map[string]interface{} {
		t.Helper()
		resp, err := http.Get(ts.URL + "/api/super-league" + query)
		if err != nil {
			t.Fatalf("GET /api/super-league%s: %v", query, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("GET /api/super-league%s status=%d", query, resp.StatusCode)
		}
		var payload struct {
			World bool                     `json:"world"`
			Clubs []map[string]interface{} `json:"clubs"`
		}
		decodeJSONBody(t, resp, &payload)
		if !payload.World {
			t.Fatalf("expected world career standings, world=%v", payload.World)
		}
		return payload.Clubs
	}

	// Default view remains the first domestic league (Premier League).
	if got := getTable(""); len(got) != 20 {
		t.Fatalf("default standings clubs=%d want 20", len(got))
	}

	// Selector by competition ID.
	byID := getTable("?league=la-liga")
	if len(byID) != 20 {
		t.Fatalf("la-liga standings clubs=%d want 20", len(byID))
	}
	for _, club := range byID {
		if club["league"] != "La Liga" {
			t.Fatalf("la-liga table contains %v league=%v", club["club_name"], club["league"])
		}
	}

	// Selector by display name.
	byName := getTable("?league=Bundesliga")
	if len(byName) != 18 {
		t.Fatalf("Bundesliga standings clubs=%d want 18", len(byName))
	}
	for _, club := range byName {
		if club["league"] != "Bundesliga" {
			t.Fatalf("Bundesliga table contains %v league=%v", club["club_name"], club["league"])
		}
	}

	// Unknown selector falls back to the default view without error.
	if got := getTable("?league=nonexistent-league"); len(got) != 20 {
		t.Fatalf("unknown selector fallback clubs=%d want 20", len(got))
	}
}
