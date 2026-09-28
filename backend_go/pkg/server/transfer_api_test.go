package server

import (
	"net/http"
	"testing"
)

func getTransfersBoard(t *testing.T, tsURL string) map[string]interface{} {
	t.Helper()
	resp, err := http.Get(tsURL + "/api/transfers")
	if err != nil {
		t.Fatalf("GET /api/transfers: %v", err)
	}
	var payload map[string]interface{}
	decodeJSONBody(t, resp, &payload)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/transfers status=%d payload=%v", resp.StatusCode, payload)
	}
	return payload
}

func TestTransfersBoardMatchesReactContractWhileWindowClosed(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()
	srv.worldMu.Lock()
	for _, club := range srv.TournamentManager.ClubsList {
		if len(club.Squad) == 0 {
			continue
		}
		club.Squad[0].ContractYears = 1
		club.Squad[0].OVR = 99
		break
	}
	srv.worldMu.Unlock()
	payload := getTransfersBoard(t, ts.URL)
	for _, key := range []string{"window_name", "is_window_open", "season_phase", "window_day", "active_negotiations", "transfer_feed", "completed_transfers", "expiring_contracts", "warchests"} {
		if _, ok := payload[key]; !ok {
			t.Fatalf("missing %s in transfers payload", key)
		}
	}
	if _, ok := payload["feed"]; ok {
		t.Fatal("legacy feed key should not be present")
	}
	if payload["is_window_open"] != false {
		t.Fatalf("window should be closed in season, got %v", payload["is_window_open"])
	}
	if payload["season_phase"] != "season" {
		t.Fatalf("season_phase=%v", payload["season_phase"])
	}
	name, _ := payload["window_name"].(string)
	if name != "Window Closed (Opens at season end)" {
		t.Fatalf("window_name=%q", name)
	}
	expiring, _ := payload["expiring_contracts"].([]interface{})
	if len(expiring) == 0 {
		t.Fatal("expected expiring contracts while the window is shut")
	}
	row, _ := expiring[0].(map[string]interface{})
	if row["player_id"] == nil || row["formatted_wage"] == nil || row["club_short"] == nil {
		t.Fatalf("expiring row incomplete: %v", row)
	}

}
