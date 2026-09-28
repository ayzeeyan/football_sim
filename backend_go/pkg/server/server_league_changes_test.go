package server

import (
	"encoding/json"
	"net/http"
	"testing"

	"football_sim/pkg/tournament"
)

// Wire-contract regression: array-typed fields must never marshal as null.
// A fresh career has no league moves, but the frontend types `moves` as an
// array and calls .map on it — a nil slice would crash the Tables tab.
func TestLeagueChangesEndpointEmitsArrayOnFreshCareer(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	if payload := postNewCareer(t, ts.URL, false, nil); payload["status"] != "success" {
		t.Fatalf("fresh world career payload: %v", payload)
	}

	resp, err := http.Get(ts.URL + "/api/season/league-changes")
	if err != nil {
		t.Fatalf("GET league-changes: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("league-changes status=%d", resp.StatusCode)
	}
	var payload struct {
		Season string                      `json:"season"`
		Moves  []tournament.RelegationMove `json:"moves"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode league-changes: %v", err)
	}
	if payload.Moves == nil {
		t.Fatal("moves must be an empty array on a fresh career, not null")
	}
	if len(payload.Moves) != 0 {
		t.Fatalf("fresh career must have no moves, got %d", len(payload.Moves))
	}
	if payload.Season == "" {
		t.Fatal("season must be present")
	}
}
