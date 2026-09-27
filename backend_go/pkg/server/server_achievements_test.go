package server

import (
	"encoding/json"
	"net/http"
	"testing"
)

// The achievements endpoint serves the catalogue, the unlocked ledger, and
// the youngest-scorer record after real matchweeks.
func TestAchievementsEndpointServesLedger(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	if payload := postNewCareer(t, ts.URL, false, nil); payload["status"] != "success" {
		t.Fatalf("fresh world career payload: %v", payload)
	}

	// Play one real matchweek so the weekly evaluation runs.
	resp, err := http.Post(ts.URL+"/api/fixtures/simulate-remaining", "application/json", nil)
	if err != nil {
		t.Fatalf("POST simulate-remaining: %v", err)
	}
	var sim map[string]interface{}
	decodeJSONBody(t, resp, &sim)
	if sim["status"] != "success" {
		t.Fatalf("slate simulation failed: %v", sim)
	}

	resp, err = http.Get(ts.URL + "/api/achievements")
	if err != nil {
		t.Fatalf("GET /api/achievements: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("achievements status=%d", resp.StatusCode)
	}
	var body struct {
		Definitions []struct {
			ID       string `json:"id"`
			Kind     string `json:"kind"`
			Unlocked bool   `json:"unlocked"`
		} `json:"definitions"`
		Achievements []struct {
			ID        string `json:"id"`
			UnlockKey string `json:"unlock_key"`
		} `json:"achievements"`
		YoungestScorerRecord *struct {
			PlayerID string `json:"player_id"`
			Age      int    `json:"age"`
		} `json:"youngest_scorer_record"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode achievements: %v", err)
	}

	if len(body.Definitions) != 7 {
		t.Fatalf("catalogue must list 7 milestones, got %d", len(body.Definitions))
	}
	seen := map[string]bool{}
	for _, def := range body.Definitions {
		seen[def.ID] = true
		if def.Kind == "" {
			t.Fatalf("definition %s missing kind", def.ID)
		}
	}
	for _, id := range []string{"unbeaten_20", "unbeaten_100", "treble", "worst_to_champion", "youngest_scorer", "goals_30_season", "sacked_before_season_end"} {
		if !seen[id] {
			t.Fatalf("catalogue missing %q: %+v", id, body.Definitions)
		}
	}
	// After a scored matchweek the youngest-scorer record must exist and the
	// matching achievement must be unlocked.
	if body.YoungestScorerRecord == nil || body.YoungestScorerRecord.Age <= 0 {
		t.Fatalf("youngest scorer record missing after a matchweek: %+v", body.YoungestScorerRecord)
	}
	unlockedYoungest := false
	for _, a := range body.Achievements {
		if a.UnlockKey == "" {
			t.Fatalf("ledger entry %s missing unlock key", a.ID)
		}
		if a.ID == "youngest_scorer" {
			unlockedYoungest = true
		}
	}
	if !unlockedYoungest {
		t.Fatalf("youngest scorer achievement must be in the ledger: %+v", body.Achievements)
	}
	for _, def := range body.Definitions {
		if def.ID == "youngest_scorer" && !def.Unlocked {
			t.Fatal("youngest scorer must be flagged unlocked in the catalogue")
		}
	}
}
