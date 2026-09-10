package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"football_sim/pkg/datamanager"
)

func TestClubXIAndFixturePreviewSitExamProdigies(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	kidID := datamanager.ProdigyStableID("Venjamin Valerio")
	homeID := "LAL-BAR"

	srv.worldMu.Lock()
	srv.TournamentManager.CurrentMatchweek = 12
	var fixtureID string
	for _, f := range srv.TournamentManager.GetSlate(12) {
		if f.Competition == "super-league" && (f.HomeID == homeID || f.AwayID == homeID) {
			fixtureID = f.FixtureID
			break
		}
	}
	srv.worldMu.Unlock()
	if fixtureID == "" {
		t.Fatal("expected a Super League fixture involving Barcelona in week 12")
	}

	resp, err := http.Get(ts.URL + "/api/clubs/" + homeID + "/xi")
	if err != nil {
		t.Fatalf("GET xi: %v", err)
	}
	var xi []map[string]interface{}
	decodeJSONBody(t, resp, &xi)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("xi status=%d", resp.StatusCode)
	}
	for _, row := range xi {
		if row["player_id"] == kidID {
			t.Fatalf("exam-week club XI still listed %s", kidID)
		}
	}

	resp, err = http.Get(ts.URL + "/api/fixtures/" + fixtureID)
	if err != nil {
		t.Fatalf("GET fixture: %v", err)
	}
	var payload map[string]interface{}
	decodeJSONBody(t, resp, &payload)
	preview, _ := payload["preview"].(map[string]interface{})
	if preview == nil {
		t.Fatal("fixture omitted preview")
	}
	note, _ := preview["kickoff_note"].(string)
	if note == "" || !strings.Contains(note, "Exam week") {
		t.Fatalf("kickoff note should mention exam week: %q", note)
	}
	side := "home_missing"
	xiKey := "home_xi"
	if payload["away_id"] == homeID {
		side = "away_missing"
		xiKey = "away_xi"
	}
	missing, _ := preview[side].([]interface{})
	found := false
	for _, raw := range missing {
		row, _ := raw.(map[string]interface{})
		if row["player_id"] == kidID {
			found = true
			if row["availability"] != "Exams" {
				t.Fatalf("missing prodigy availability=%v, want Exams", row["availability"])
			}
		}
	}
	if !found {
		t.Fatalf("exam prodigy not in %s", side)
	}
	for _, raw := range preview[xiKey].([]interface{}) {
		row, _ := raw.(map[string]interface{})
		if row["player_id"] == kidID {
			t.Fatalf("exam prodigy still in probable XI")
		}
	}
}

func TestGetClubXIReturnsArrayWhenEmpty(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	resp, err := http.Get(ts.URL + "/api/clubs/LAL-BAR/xi")
	if err != nil {
		t.Fatalf("GET xi: %v", err)
	}
	defer resp.Body.Close()
	var raw json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(raw) == 0 || raw[0] != '[' {
		t.Fatalf("XI payload should be a JSON array, got %s", raw)
	}
}
