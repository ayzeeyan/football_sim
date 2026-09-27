package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

// Tier B training control (B2): any squad player can be trained. The first
// session registers a growth profile on demand, the wonderkid-only 99
// potential marker is preserved, and the shared weekly energy still binds.
func TestTrainAnyPlayerEndpoint(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	if payload := postNewCareer(t, ts.URL, false, nil); payload["status"] != "success" {
		t.Fatalf("fresh world career payload: %v", payload)
	}

	// Find a non-wonderkid squad player.
	clubs := []struct {
		ClubID string `json:"club_id"`
	}{}
	resp, err := http.Get(ts.URL + "/api/clubs")
	if err != nil {
		t.Fatalf("GET /api/clubs: %v", err)
	}
	decodeJSONBody(t, resp, &clubs)
	var playerID string
	for _, c := range clubs {
		resp, err := http.Get(ts.URL + "/api/clubs/" + c.ClubID + "/squad")
		if err != nil {
			t.Fatalf("GET squad: %v", err)
		}
		var squad []struct {
			PlayerID          string `json:"player_id"`
			UniverseWonderkid bool   `json:"universe_wonderkid"`
		}
		decodeJSONBody(t, resp, &squad)
		for _, p := range squad {
			if !p.UniverseWonderkid {
				playerID = p.PlayerID
				break
			}
		}
		if playerID != "" {
			break
		}
	}
	if playerID == "" {
		t.Fatal("no non-wonderkid player found")
	}

	train := func(focus string) (map[string]interface{}, int) {
		t.Helper()
		body, _ := json.Marshal(map[string]string{"focus": focus})
		resp, err := http.Post(ts.URL+"/api/players/"+playerID+"/train", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("POST train: %v", err)
		}
		defer resp.Body.Close()
		var out map[string]interface{}
		if resp.StatusCode == http.StatusOK {
			if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
				t.Fatalf("decode train: %v", err)
			}
		}
		return out, resp.StatusCode
	}

	// First session registers the profile on demand and succeeds.
	out, code := train("technical")
	if code != http.StatusOK || out["status"] == "error" {
		t.Fatalf("train non-prodigy failed: %d %v", code, out)
	}
	if _, ok := out["ovr"]; !ok {
		t.Fatalf("train response must carry the OVR: %v", out)
	}

	// The on-demand profile never carries the wonderkid-only 99 marker.
	if pot, tracked := srv.GrowthEngine.PotentialFor(playerID); !tracked {
		t.Fatal("player must be registered after training")
	} else if pot > 98 {
		t.Fatalf("non-wonderkid potential %d must stay below 99", pot)
	}

	// Unknown focus is rejected.
	body, _ := json.Marshal(map[string]string{"focus": "yoga"})
	resp, err = http.Post(ts.URL+"/api/players/"+playerID+"/train", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST bad train: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown focus status=%d want 400", resp.StatusCode)
	}

	// Unknown player is a clean 404.
	body, _ = json.Marshal(map[string]string{"focus": "technical"})
	resp, err = http.Post(ts.URL+"/api/players/NO-SUCH-PLAYER/train", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST unknown train: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown player status=%d want 404", resp.StatusCode)
	}
}
