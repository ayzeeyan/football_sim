package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"football_sim/pkg/datamanager"
	"football_sim/pkg/persistence"
)

func getTransfersBoard(t *testing.T, tsURL string) map[string]interface{} {
	t.Helper()
	resp, err := http.Get(tsURL + "/api/transfers")
	if err != nil { t.Fatalf("GET /api/transfers: %v", err) }
	var payload map[string]interface{}
	decodeJSONBody(t, resp, &payload)
	if resp.StatusCode != http.StatusOK { t.Fatalf("GET /api/transfers status=%d payload=%v", resp.StatusCode, payload) }
	return payload
}

func TestTransfersBoardMatchesReactContractWhileWindowClosed(t *testing.T) {
	srv, ts := setupTestServer(t); defer ts.Close(); defer srv.Stop()
	srv.worldMu.Lock()
	for _, club := range srv.TournamentManager.ClubsList {
		if len(club.Squad) == 0 { continue }
		club.Squad[0].ContractYears = 1; club.Squad[0].OVR = 99; break
	}
	srv.worldMu.Unlock()
	payload := getTransfersBoard(t, ts.URL)
	for _, key := range []string{"window_name", "is_window_open", "season_phase", "window_day", "active_negotiations", "transfer_feed", "completed_transfers", "expiring_contracts", "warchests"} {
		if _, ok := payload[key]; !ok { t.Fatalf("missing %s in transfers payload", key) }
	}
	if _, ok := payload["feed"]; ok { t.Fatal("legacy feed key should not be present") }
	if payload["is_window_open"] != false { t.Fatalf("window should be closed in season, got %v", payload["is_window_open"]) }
	if payload["season_phase"] != "season" { t.Fatalf("season_phase=%v", payload["season_phase"]) }
	name, _ := payload["window_name"].(string)
	if name != "Window Closed (Opens at season end)" { t.Fatalf("window_name=%q", name) }
	expiring, _ := payload["expiring_contracts"].([]interface{})
	if len(expiring) == 0 { t.Fatal("expected expiring contracts while the window is shut") }
	row, _ := expiring[0].(map[string]interface{})
	if row["player_id"] == nil || row["formatted_wage"] == nil || row["club_short"] == nil { t.Fatalf("expiring row incomplete: %v", row) }
	resp, err := http.Post(ts.URL+"/api/transfers/advance", "application/json", nil)
	if err != nil { t.Fatalf("advance while closed: %v", err) }
	var closed map[string]interface{}
	decodeJSONBody(t, resp, &closed)
	if resp.StatusCode != http.StatusBadRequest { t.Fatalf("closed advance status=%d", resp.StatusCode) }
	if closed["detail"] != "The window opens when the season ends." { t.Fatalf("closed advance detail=%v", closed["detail"]) }
}

func TestTransferBidAndAdvanceDuringCareerWindow(t *testing.T) {
	srv, ts := setupTestServer(t); defer ts.Close(); defer srv.Stop()
	playerID := datamanager.ProdigyStableID("Venjamin Valerio")
	sellerID := "LAL-BAR"; buyerID := "FL1-PSG"

	srv.worldMu.Lock()
	srv.TournamentManager.SeasonPhase = "transfer_window"
	// SeasonPhase alone must not implicitly open/reset the market. The actual
	// state-machine boundary initializes budgets and Week 1 exactly once.
	srv.TransferEngine.BeginOffSeasonWindow()
	dayBefore := srv.TransferEngine.CurrentDay
	srv.worldMu.Unlock()

	body, _ := json.Marshal(map[string]string{"buyer_id": buyerID, "seller_id": sellerID, "player_id": playerID})
	resp, err := http.Post(ts.URL+"/api/transfers/bid", "application/json", bytes.NewReader(body))
	if err != nil { t.Fatalf("bid: %v", err) }
	var bid map[string]interface{}
	decodeJSONBody(t, resp, &bid)
	if resp.StatusCode != http.StatusOK { t.Fatalf("bid status=%d payload=%v", resp.StatusCode, bid) }
	if bid["formatted_bid"] == nil || bid["stage_name"] != "INQUIRY" { t.Fatalf("bid payload missing React fields: %v", bid) }
	player, _ := bid["player"].(map[string]interface{}); buyer, _ := bid["buyer"].(map[string]interface{}); seller, _ := bid["seller"].(map[string]interface{})
	if player["full_name"] != "Venjamin Valerio" || buyer["club_id"] != buyerID || seller["club_id"] != sellerID { t.Fatalf("bid clubs/player: %v", bid) }

	resp2, err := http.Post(ts.URL+"/api/transfers/bid", "application/json", bytes.NewReader(body))
	if err != nil { t.Fatalf("repeat bid: %v", err) }
	var bid2 map[string]interface{}
	decodeJSONBody(t, resp2, &bid2)
	if bid2["negotiation_id"] != bid["negotiation_id"] { t.Fatalf("repeat bid opened a second talk: %v vs %v", bid["negotiation_id"], bid2["negotiation_id"]) }

	saved, err := persistence.LoadCareer(srv.savePath)
	if err != nil { t.Fatalf("bid did not persist: %v", err) }
	if saved.Transfers.CurrentDay != dayBefore { t.Fatalf("bid should not advance the window day, got %d from %d", saved.Transfers.CurrentDay, dayBefore) }

	respAdv, err := http.Post(ts.URL+"/api/transfers/advance", "application/json", nil)
	if err != nil { t.Fatalf("advance: %v", err) }
	var board map[string]interface{}
	decodeJSONBody(t, respAdv, &board)
	if respAdv.StatusCode != http.StatusOK { t.Fatalf("advance status=%d payload=%v", respAdv.StatusCode, board) }
	if board["is_window_open"] != true { t.Fatalf("advance should keep the career window open: %v", board["is_window_open"]) }
	if board["window_day"].(float64) != float64(dayBefore+1) { t.Fatalf("window_day=%v want %d", board["window_day"], dayBefore+1) }
	negs, _ := board["active_negotiations"].([]interface{})
	if len(negs) == 0 { t.Fatal("expected negotiations after a bid and a market day") }
	first, _ := negs[0].(map[string]interface{})
	if first["formatted_bid"] == nil || first["player"] == nil { t.Fatalf("negotiation not serialized for React: %v", first) }
	warchests, _ := board["warchests"].([]interface{})
	if len(warchests) != 12 { t.Fatalf("warchests=%d, want 12", len(warchests)) }
	saved, err = persistence.LoadCareer(srv.savePath)
	if err != nil { t.Fatalf("advance persist: %v", err) }
	if saved.Transfers.CurrentDay != dayBefore+1 { t.Fatalf("saved window day=%d want %d", saved.Transfers.CurrentDay, dayBefore+1) }
}
