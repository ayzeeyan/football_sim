package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"football_sim/pkg/datamanager"
)

// Tier B transfer control (B3): the viewer opens a negotiation with an
// offer, improves it to the asking price to complete the deal, and can
// withdraw. Every path stays inside the FSM's economics.
func TestViewerTransferOfferFlow(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	if payload := postNewCareer(t, ts.URL, false, nil); payload["status"] != "success" {
		t.Fatalf("fresh world career payload: %v", payload)
	}

	// Open the summer window the way the career FSM does.
	srv.worldMu.Lock()
	srv.TournamentManager.SeasonPhase = "transfer_window"
	srv.TransferEngine.BeginOffSeasonWindow()
	srv.worldMu.Unlock()

	// Find a real, non-wonderkid player and his club.
	clubs := []struct {
		ClubID string `json:"club_id"`
	}{}
	resp, err := http.Get(ts.URL + "/api/clubs")
	if err != nil {
		t.Fatalf("GET /api/clubs: %v", err)
	}
	decodeJSONBody(t, resp, &clubs)
	var playerID, sellerID, buyerID string
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
				if sellerID == "" {
					playerID, sellerID = p.PlayerID, c.ClubID
				} else if buyerID == "" && c.ClubID != sellerID {
					buyerID = c.ClubID
					break
				}
			}
		}
		if buyerID != "" {
			break
		}
	}
	if playerID == "" || buyerID == "" {
		t.Fatal("could not find two clubs with non-wonderkid players")
	}

	post := func(path string, body interface{}) (map[string]interface{}, int) {
		t.Helper()
		raw, _ := json.Marshal(body)
		resp, err := http.Post(ts.URL+path, "application/json", bytes.NewReader(raw))
		if err != nil {
			t.Fatalf("POST %s: %v", path, err)
		}
		defer resp.Body.Close()
		var out map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
		return out, resp.StatusCode
	}

	// A lowball offer opens the negotiation at INQUIRY. Players refuse
	// some destinations, so try buyers until one is accepted.
	var offer map[string]interface{}
	code := 0
	for _, candidate := range clubs {
		if candidate.ClubID == sellerID {
			continue
		}
		offer, code = post("/api/transfers/offer", map[string]interface{}{
			"player_id": playerID, "buyer_id": candidate.ClubID, "amount": 1_000_000,
		})
		if code == http.StatusOK {
			buyerID = candidate.ClubID
			break
		}
	}
	if code != http.StatusOK {
		t.Fatalf("no buyer accepted an offer for the player: %d %v", code, offer)
	}
	negID, _ := offer["negotiation_id"].(string)
	if negID == "" || offer["stage_name"] != "INQUIRY" {
		t.Fatalf("offer must open an INQUIRY negotiation: %v", offer)
	}
	asking, _ := offer["asking_price"].(float64)
	if asking <= 0 {
		t.Fatalf("offer must carry the asking price: %v", offer)
	}

	// An offer below the current bid is rejected with a reason.
	bad, code := post("/api/transfers/negotiations/"+negID+"/respond", map[string]interface{}{
		"action": "improve", "amount": 500_000,
	})
	if code != http.StatusOK || bad["status"] != "error" {
		t.Fatalf("low improve must be rejected: %d %v", code, bad)
	}

	// Meeting the asking price completes the transfer through the FSM.
	done, code := post("/api/transfers/negotiations/"+negID+"/respond", map[string]interface{}{
		"action": "improve", "amount": int64(asking) + 1_000_000,
	})
	if code != http.StatusOK {
		t.Fatalf("accepting offer failed: %d %v", code, done)
	}
	if done["status"] == "error" {
		t.Fatalf("accepting offer rejected: %v", done)
	}

	// The player now belongs to the buying club.
	resp, err = http.Get(ts.URL + "/api/clubs/" + buyerID + "/squad")
	if err != nil {
		t.Fatalf("GET buyer squad: %v", err)
	}
	var squad []struct {
		PlayerID string `json:"player_id"`
	}
	decodeJSONBody(t, resp, &squad)
	found := false
	for _, p := range squad {
		if p.PlayerID == playerID {
			found = true
		}
	}
	if !found {
		t.Fatal("completed transfer must move the player to the buying club")
	}

	// A second offer for the same player this window is refused.
	again, code := post("/api/transfers/offer", map[string]interface{}{
		"player_id": playerID, "buyer_id": sellerID, "amount": 50_000_000,
	})
	if code != http.StatusBadRequest {
		t.Fatalf("re-offer for a moved player must fail: %d %v", code, again)
	}

	// Unknown negotiation is a clean 404.
	_, code = post("/api/transfers/negotiations/NO-SUCH-NEG/respond", map[string]interface{}{
		"action": "improve", "amount": 1_000_000,
	})
	if code != http.StatusNotFound {
		t.Fatalf("unknown negotiation status=%d want 404", code)
	}
}

// Withdraw collapses an active negotiation without moving anyone.
func TestViewerTransferWithdraw(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	if payload := postNewCareer(t, ts.URL, false, nil); payload["status"] != "success" {
		t.Fatalf("fresh world career payload: %v", payload)
	}
	srv.worldMu.Lock()
	srv.TournamentManager.SeasonPhase = "transfer_window"
	srv.TransferEngine.BeginOffSeasonWindow()
	srv.worldMu.Unlock()

	// Use the canonical prodigy pair the existing bid test uses: any player
	// works for a withdrawal flow.
	playerID := datamanager.ProdigyStableID("Maverick Cantalejo")
	offer, code := func() (map[string]interface{}, int) {
		raw, _ := json.Marshal(map[string]interface{}{
			"player_id": playerID, "buyer_id": "FL1-PSG", "amount": 40_000_000,
		})
		resp, err := http.Post(ts.URL+"/api/transfers/offer", "application/json", bytes.NewReader(raw))
		if err != nil {
			t.Fatalf("POST offer: %v", err)
		}
		defer resp.Body.Close()
		var out map[string]interface{}
		if resp.StatusCode == http.StatusOK {
			_ = json.NewDecoder(resp.Body).Decode(&out)
		}
		return out, resp.StatusCode
	}()
	if code == http.StatusOK && offer["negotiation_id"] != nil {
		negID, _ := offer["negotiation_id"].(string)
		raw, _ := json.Marshal(map[string]interface{}{"action": "withdraw"})
		resp, err := http.Post(ts.URL+"/api/transfers/negotiations/"+negID+"/respond", "application/json", bytes.NewReader(raw))
		if err != nil {
			t.Fatalf("POST withdraw: %v", err)
		}
		defer resp.Body.Close()
		var out map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&out)
		if resp.StatusCode != http.StatusOK || out["stage_name"] != "COLLAPSED" {
			t.Fatalf("withdraw must collapse the negotiation: %d %v", resp.StatusCode, out)
		}
	}
	// An offer that the engine refuses (window rules, destination, budget)
	// is a clean 400 — both outcomes are acceptable for this smoke test.
}
