package server

import (
	"encoding/json"
	"net/http"
	"testing"
)

// The medical endpoint serves the club medical view: injuries with rehab
// roadmaps, risk assessments, and the season history summary.
func TestMedicalEndpointServesClubView(t *testing.T) {
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

	resp, err = http.Get(ts.URL + "/api/clubs/" + clubID + "/medical")
	if err != nil {
		t.Fatalf("GET medical: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("medical status=%d", resp.StatusCode)
	}
	var body struct {
		ClubID  string `json:"club_id"`
		Season  string `json:"season"`
		Injured []struct {
			PlayerID   string `json:"player_id"`
			Kind       string `json:"kind"`
			MatchesOut int    `json:"matches_out"`
			Rehab      struct {
				Stages []struct {
					Phase string `json:"phase"`
				} `json:"stages"`
			} `json:"rehab"`
		} `json:"injured"`
		TopRisks []struct {
			PlayerID   string `json:"player_id"`
			Fitness    int    `json:"fitness"`
			Assessment struct {
				RiskScore int `json:"risk_score"`
				Factors   []struct {
					Label string  `json:"label"`
					Mult  float64 `json:"mult"`
				} `json:"factors"`
			} `json:"assessment"`
		} `json:"top_risks"`
		History struct {
			TotalInjuries int `json:"total_injuries"`
		} `json:"history"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode medical: %v", err)
	}
	if body.ClubID != clubID || body.Season == "" {
		t.Fatalf("medical club/season missing: %+v", body)
	}
	if len(body.TopRisks) == 0 || len(body.TopRisks) > 8 {
		t.Fatalf("top risks must be a bounded non-empty list: %d", len(body.TopRisks))
	}
	for _, risk := range body.TopRisks {
		if risk.Assessment.RiskScore < 1 || risk.Assessment.RiskScore > 100 {
			t.Fatalf("risk score out of bounds for %s: %d", risk.PlayerID, risk.Assessment.RiskScore)
		}
		if len(risk.Assessment.Factors) == 0 {
			t.Fatalf("risk assessment must itemise factors for %s", risk.PlayerID)
		}
	}
	// A fresh career has no injuries yet; the list is present but empty.
	if body.Injured == nil {
		t.Fatal("injured list must be present (empty is fine)")
	}
	if body.History.TotalInjuries != 0 {
		t.Fatalf("fresh career must have no injury history, got %d", body.History.TotalInjuries)
	}

	// Unknown club is a clean 404.
	resp, err = http.Get(ts.URL + "/api/clubs/NO-SUCH-CLUB/medical")
	if err != nil {
		t.Fatalf("GET unknown club medical: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown club status=%d want 404", resp.StatusCode)
	}
}

// The player payload carries the injury history for the player sheet.
func TestPlayerPayloadCarriesInjuryHistory(t *testing.T) {
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

	resp, err = http.Get(ts.URL + "/api/clubs/" + clubID + "/squad")
	if err != nil {
		t.Fatalf("GET squad: %v", err)
	}
	defer resp.Body.Close()
	var squad []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&squad); err != nil {
		t.Fatalf("decode squad: %v", err)
	}
	if len(squad) == 0 {
		t.Fatal("empty squad")
	}
	if _, present := squad[0]["injury_history"]; !present {
		t.Fatal("serialized player must carry injury_history")
	}
}
