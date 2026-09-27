package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

// Tier B board & finance control (B4): the viewer sets a club's budget,
// wage cap, and board objective. The domain invariants still bind.
func TestBoardControlEndpoint(t *testing.T) {
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

	// Read the profile for the current balance.
	resp, err = http.Get(ts.URL + "/api/clubs/" + clubID + "/profile")
	if err != nil {
		t.Fatalf("GET profile: %v", err)
	}
	var profile struct {
		Club struct {
			Finances struct {
				Balance        int64 `json:"balance"`
				TransferBudget int64 `json:"transfer_budget"`
				WageCap        int64 `json:"wage_cap"`
			} `json:"finances"`
			BoardObjective string `json:"board_objective"`
		} `json:"club"`
	}
	decodeJSONBody(t, resp, &profile)
	balance := profile.Club.Finances.Balance
	if balance <= 0 {
		t.Fatalf("club balance must be positive, got %d", balance)
	}

	post := func(body interface{}) (map[string]interface{}, int) {
		t.Helper()
		raw, _ := json.Marshal(body)
		resp, err := http.Post(ts.URL+"/api/clubs/"+clubID+"/board", "application/json", bytes.NewReader(raw))
		if err != nil {
			t.Fatalf("POST board: %v", err)
		}
		defer resp.Body.Close()
		var out map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			t.Fatalf("decode board: %v", err)
		}
		return out, resp.StatusCode
	}

	// A legal budget and objective are accepted.
	out, code := post(map[string]interface{}{
		"transfer_budget": balance / 2,
		"board_objective": "Title challenge",
	})
	if code != http.StatusOK || out["status"] != "success" {
		t.Fatalf("board set failed: %d %v", code, out)
	}

	// The change is visible on the profile.
	resp, err = http.Get(ts.URL + "/api/clubs/" + clubID + "/profile")
	if err != nil {
		t.Fatalf("GET profile after set: %v", err)
	}
	var after struct {
		Club struct {
			Finances struct {
				TransferBudget int64 `json:"transfer_budget"`
			} `json:"finances"`
			BoardObjective string `json:"board_objective"`
		} `json:"club"`
	}
	decodeJSONBody(t, resp, &after)
	if after.Club.Finances.TransferBudget != balance/2 {
		t.Fatalf("budget not applied: %d", after.Club.Finances.TransferBudget)
	}
	if after.Club.BoardObjective != "Title challenge" {
		t.Fatalf("objective not applied: %q", after.Club.BoardObjective)
	}

	// A budget above the balance is refused.
	_, code = post(map[string]interface{}{"transfer_budget": balance * 2})
	if code != http.StatusBadRequest {
		t.Fatalf("over-balance budget status=%d want 400", code)
	}

	// A negative budget is refused.
	_, code = post(map[string]interface{}{"transfer_budget": -1})
	if code != http.StatusBadRequest {
		t.Fatalf("negative budget status=%d want 400", code)
	}

	// A wage cap below the committed wage bill is refused.
	srv.worldMu.RLock()
	bill := srv.TournamentManager.Clubs[clubID].WageBill()
	srv.worldMu.RUnlock()
	if bill > 0 {
		_, code = post(map[string]interface{}{"wage_cap": bill - 1})
		if code != http.StatusBadRequest {
			t.Fatalf("below-bill wage cap status=%d want 400", code)
		}
		// A cap at the bill is accepted.
		_, code = post(map[string]interface{}{"wage_cap": bill})
		if code != http.StatusOK {
			t.Fatalf("at-bill wage cap status=%d want 200", code)
		}
	}

	// An unknown objective is refused.
	_, code = post(map[string]interface{}{"board_objective": "Win everything forever"})
	if code != http.StatusBadRequest {
		t.Fatalf("unknown objective status=%d want 400", code)
	}

	// An empty request is refused.
	_, code = post(map[string]interface{}{})
	if code != http.StatusBadRequest {
		t.Fatalf("empty board request status=%d want 400", code)
	}

	// Unknown club is a clean 404.
	raw, _ := json.Marshal(map[string]interface{}{"transfer_budget": 1})
	resp, err = http.Post(ts.URL+"/api/clubs/NO-SUCH-CLUB/board", "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("POST unknown club board: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown club status=%d want 404", resp.StatusCode)
	}
}
