package server

import (
	"encoding/json"
	"net/http"
	"testing"
)

// The scouting endpoint serves a deterministic, read-only recruitment
// shortlist per club.
func TestScoutingEndpointServesShortlist(t *testing.T) {
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
	if len(clubs) == 0 {
		t.Fatal("no clubs in the world")
	}
	clubID := clubs[0].ClubID

	fetch := func(url string) (map[string]interface{}, int) {
		t.Helper()
		r, err := http.Get(url)
		if err != nil {
			t.Fatalf("GET scouting: %v", err)
		}
		defer r.Body.Close()
		var body map[string]interface{}
		if r.StatusCode == http.StatusOK {
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode scouting: %v", err)
			}
		}
		return body, r.StatusCode
	}

	first, code := fetch(ts.URL + "/api/clubs/" + clubID + "/scouting")
	if code != http.StatusOK {
		t.Fatalf("scouting status=%d body=%v", code, first)
	}
	if first["club_id"] != clubID {
		t.Fatalf("scouting club_id=%v want %s", first["club_id"], clubID)
	}
	shortlist, ok := first["shortlist"].([]interface{})
	if !ok || len(shortlist) == 0 {
		t.Fatalf("shortlist missing or empty: %v", first["shortlist"])
	}
	entry, ok := shortlist[0].(map[string]interface{})
	if !ok {
		t.Fatalf("shortlist entry malformed: %v", shortlist[0])
	}
	for _, key := range []string{"player_id", "potential_ceiling", "consistency", "form", "value_trend", "risk", "verdict", "scout_score", "region"} {
		if _, present := entry[key]; !present {
			t.Fatalf("shortlist entry missing %q: %v", key, entry)
		}
	}
	if entry["club_id"] == clubID {
		t.Fatal("shortlist must exclude the buying club's own players")
	}

	// Deterministic: same request, same answer.
	second, _ := fetch(ts.URL + "/api/clubs/" + clubID + "/scouting")
	b1, _ := json.Marshal(first)
	b2, _ := json.Marshal(second)
	if string(b1) != string(b2) {
		t.Fatal("scouting shortlist is not deterministic across requests")
	}

	// Limit is respected.
	limited, _ := fetch(ts.URL + "/api/clubs/" + clubID + "/scouting?limit=3")
	if entries, ok := limited["shortlist"].([]interface{}); !ok || len(entries) > 3 {
		t.Fatalf("limit=3 must cap the shortlist, got %v", limited["shortlist"])
	}

	// Unknown club is a clean 404.
	if _, code := fetch(ts.URL + "/api/clubs/NO-SUCH-CLUB/scouting"); code != http.StatusNotFound {
		t.Fatalf("unknown club status=%d want 404", code)
	}
}
