package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"football_sim/pkg/datamanager"
	"football_sim/pkg/growth"
	"football_sim/pkg/tournament"
	"football_sim/pkg/transfers"

	"github.com/gorilla/websocket"
)

func setupTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	ge := growth.NewGrowthEngine(42)
	datasetPath := filepath.Join("..", "..", "..", "dataset.json")
	dm := datamanager.NewDataManager(datasetPath, ge)
	if dm == nil || len(dm.Clubs) == 0 {
		t.Fatalf("failed to load dataset from %s", datasetPath)
	}

	eliteClubs := dm.GetEliteClubs()
	tm := tournament.NewTournamentManager(eliteClubs, ge, 42)
	te := transfers.NewTransferEngine(eliteClubs, tm.Managers, 42)

	tempDir := t.TempDir()
	savePath := filepath.Join(tempDir, "server_test_career.json")

	srv := NewServer(dm, ge, tm, te, savePath, "")
	ts := httptest.NewServer(srv)
	return srv, ts
}

func TestServer_HealthAndStats(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer srv.Stop()
	defer ts.Close()

	// 1. Health
	resp, err := http.Get(ts.URL + "/api/health")
	if err != nil {
		t.Fatalf("GET /api/health failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var health map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&health)
	if health["status"] != "ok" || health["backend"] != "go" {
		t.Errorf("unexpected health payload: %v", health)
	}

	// 2. Stats
	respStats, err := http.Get(ts.URL + "/api/stats")
	if err != nil {
		t.Fatalf("GET /api/stats failed: %v", err)
	}
	defer respStats.Body.Close()
	var stats map[string]interface{}
	_ = json.NewDecoder(respStats.Body).Decode(&stats)
	if stats["clubs_count"].(float64) != 12 {
		t.Errorf("expected 12 clubs in stats, got %v", stats["clubs_count"])
	}
}

func TestServer_ClubsAndRosters(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer srv.Stop()
	defer ts.Close()

	// 1. Clubs list
	resp, err := http.Get(ts.URL + "/api/clubs")
	if err != nil {
		t.Fatalf("GET /api/clubs failed: %v", err)
	}
	defer resp.Body.Close()
	var clubs []map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&clubs)
	if len(clubs) != 12 {
		t.Fatalf("expected 12 clubs, got %d", len(clubs))
	}

	// 2. Club Squad
	respSquad, err := http.Get(ts.URL + "/api/clubs/LAL-BAR/squad")
	if err != nil {
		t.Fatalf("GET /api/clubs/LAL-BAR/squad failed: %v", err)
	}
	defer respSquad.Body.Close()
	var squad []interface{}
	_ = json.NewDecoder(respSquad.Body).Decode(&squad)
	if len(squad) < 18 {
		t.Errorf("expected at least 18 squad members, got %d", len(squad))
	}

	// 3. Starting XI & Bench
	respXI, err := http.Get(ts.URL + "/api/clubs/LAL-BAR/xi")
	if err != nil {
		t.Fatalf("GET /api/clubs/LAL-BAR/xi failed: %v", err)
	}
	defer respXI.Body.Close()
	var starters []interface{}
	_ = json.NewDecoder(respXI.Body).Decode(&starters)
	if len(starters) != 11 {
		t.Errorf("expected 11 starters, got %d", len(starters))
	}
}

func TestServer_EmptyClubXIIsJSONArray(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer srv.Stop()
	defer ts.Close()

	srv.worldMu.Lock()
	club := srv.TournamentManager.Clubs["LAL-BAR"]
	if club == nil {
		srv.worldMu.Unlock()
		t.Fatal("expected test club")
	}
	for _, p := range club.Squad {
		p.InjuredMatches = 1
	}
	srv.worldMu.Unlock()

	resp, err := http.Get(ts.URL + "/api/clubs/LAL-BAR/xi")
	if err != nil {
		t.Fatalf("GET empty /api/clubs/LAL-BAR/xi failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var starters []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&starters); err != nil {
		t.Fatalf("decode empty XI response: %v", err)
	}
	if starters == nil {
		t.Fatalf("empty XI response must be [] rather than null")
	}
	if len(starters) != 0 {
		t.Fatalf("expected empty XI, got %d players", len(starters))
	}
}

func TestServer_ContextualLiveCoordsExcludeExamPlayer(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer srv.Stop()
	defer ts.Close()

	srv.worldMu.Lock()
	home := srv.TournamentManager.Clubs["LAL-BAR"]
	away := srv.TournamentManager.Clubs["LAL-RMA"]
	if home == nil || away == nil || len(home.Squad) == 0 {
		srv.worldMu.Unlock()
		t.Fatal("expected test clubs with a wonderkid")
	}
	var examKidID string
	for _, p := range home.Squad {
		if p != nil && p.UniverseWonderkid {
			examKidID = p.PlayerID
			p.Age = 16
			p.Education = "high_school"
			break
		}
	}
	if examKidID == "" {
		srv.worldMu.Unlock()
		t.Fatal("expected a wonderkid in the test club")
	}
	srv.LiveMatchEngine.SetClubs(home, away, srv.TournamentManager.Managers[home.ClubID], srv.TournamentManager.Managers[away.ClubID])
	srv.LiveMatchEngine.SetFixtureContext("super-league", 12)
	payload := srv.buildMatchTickPayload()
	selectedHome := len(srv.LiveMatchEngine.HomeStarters)
	selectedAway := len(srv.LiveMatchEngine.AwayStarters)
	srv.worldMu.Unlock()

	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal live tick payload: %v", err)
	}
	var wire map[string]interface{}
	if err := json.Unmarshal(encoded, &wire); err != nil {
		t.Fatalf("decode live tick payload: %v", err)
	}
	for _, check := range []struct {
		key  string
		want int
	}{
		{key: "home_coords", want: selectedHome},
		{key: "away_coords", want: selectedAway},
	} {
		coords, ok := wire[check.key].([]interface{})
		if !ok {
			t.Fatalf("%s missing from serialized live tick: %v", check.key, wire[check.key])
		}
		if len(coords) != check.want {
			t.Fatalf("%s count = %d; want %d selected actors", check.key, len(coords), check.want)
		}
		for _, raw := range coords {
			coord, ok := raw.(map[string]interface{})
			if !ok {
				t.Fatalf("%s actor has unexpected shape: %T", check.key, raw)
			}
			player, ok := coord["player"].(map[string]interface{})
			if !ok || player["player_id"] == "" {
				t.Fatalf("%s contains blank actor: %v", check.key, coord)
			}
			if player["player_id"] == examKidID {
				t.Fatalf("%s contains exam player %s", check.key, examKidID)
			}
		}
	}
}

func TestServer_LiveTickSerializesDismissalAndSubstituteIdentity(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer srv.Stop()
	defer ts.Close()

	srv.worldMu.Lock()
	e := srv.LiveMatchEngine
	e.ResetMatch()
	if len(e.HomeStarters) < 3 || len(e.HomeBench) == 0 {
		srv.worldMu.Unlock()
		t.Fatal("expected live starters and bench")
	}
	startingRed := e.HomeStarters[1]
	e.Bookings[startingRed.PlayerID] = 2
	firstPayload := srv.buildMatchTickPayload()

	// A replacement must update the radar actor identity, then inherit the
	// dismissal flag from the current bookings map.
	out := e.HomeStarters[2]
	in := e.HomeBench[0]
	e.ExecuteDirectSub("home", out, in, 70, "test")
	e.Bookings[in.PlayerID] = 2
	secondPayload := srv.buildMatchTickPayload()
	srv.worldMu.Unlock()

	type wireActor struct {
		Player  map[string]interface{} `json:"player"`
		SentOff bool                   `json:"sent_off"`
	}
	type wireTick struct {
		HomeCoords []wireActor `json:"home_coords"`
	}
	decode := func(payload map[string]interface{}) wireTick {
		t.Helper()
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal live tick payload: %v", err)
		}
		var wire wireTick
		if err := json.Unmarshal(encoded, &wire); err != nil {
			t.Fatalf("decode live tick payload: %v", err)
		}
		return wire
	}
	first := decode(firstPayload)
	foundStartingRed := false
	for _, actor := range first.HomeCoords {
		if actor.Player["player_id"] == startingRed.PlayerID {
			foundStartingRed = actor.SentOff
		}
	}
	if !foundStartingRed {
		t.Fatalf("starting dismissed actor missing sent_off=true: %+v", first.HomeCoords)
	}
	second := decode(secondPayload)
	foundSubRed := false
	for _, actor := range second.HomeCoords {
		if actor.Player["player_id"] == in.PlayerID {
			foundSubRed = actor.SentOff
		}
		if actor.Player["player_id"] == out.PlayerID {
			t.Fatalf("subbed-out actor still represented in radar: %+v", actor)
		}
	}
	if !foundSubRed {
		t.Fatalf("dismissed substitute missing current identity/sent_off=true: %+v", second.HomeCoords)
	}
}

func TestServer_ProdigiesAndTraining(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer srv.Stop()
	defer ts.Close()

	// 1. Prodigies list
	resp, err := http.Get(ts.URL + "/api/prodigies")
	if err != nil {
		t.Fatalf("GET /api/prodigies failed: %v", err)
	}
	defer resp.Body.Close()
	var prodigies []map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&prodigies)
	if len(prodigies) != 12 {
		t.Fatalf("expected 12 wonderkids, got %d", len(prodigies))
	}

	wkID, _ := prodigies[0]["player_id"].(string)
	if wkID == "" {
		t.Fatalf("expected prodigy player_id in payload: %v", prodigies[0])
	}

	// 2. Train Prodigy
	reqBody := strings.NewReader(`{"focus": "hypertrophy"}`)
	respTrain, err := http.Post(ts.URL+"/api/prodigies/"+wkID+"/train", "application/json", reqBody)
	if err != nil {
		t.Fatalf("POST /api/prodigies/%s/train failed: %v", wkID, err)
	}
	defer respTrain.Body.Close()
	var trainResult map[string]interface{}
	_ = json.NewDecoder(respTrain.Body).Decode(&trainResult)
	if trainResult["status"] != "success" {
		t.Errorf("expected success training result, got %v", trainResult)
	}

	// 3. Timeline
	respTimeline, err := http.Get(ts.URL + "/api/prodigies/" + wkID + "/timeline")
	if err != nil {
		t.Fatalf("GET /api/prodigies/%s/timeline failed: %v", wkID, err)
	}
	defer respTimeline.Body.Close()
	if respTimeline.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for timeline, got %d", respTimeline.StatusCode)
	}
}

func TestServer_SuperLeagueSimulation(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer srv.Stop()
	defer ts.Close()

	// 1. Super League Standings
	resp, err := http.Get(ts.URL + "/api/super-league")
	if err != nil {
		t.Fatalf("GET /api/super-league failed: %v", err)
	}
	defer resp.Body.Close()
	var sl map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&sl)
	if sl["current_matchweek"].(float64) != 1 {
		t.Errorf("expected matchweek 1, got %v", sl["current_matchweek"])
	}

	// 2. Simulate Matchweek
	respSim, err := http.Post(ts.URL+"/api/fixtures/simulate-remaining", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /api/fixtures/simulate-remaining failed: %v", err)
	}
	defer respSim.Body.Close()
	var simResult map[string]interface{}
	_ = json.NewDecoder(respSim.Body).Decode(&simResult)
	if simResult["status"] != "success" {
		t.Errorf("expected success simulation, got %v", simResult)
	}
	if simResult["simulated_count"].(float64) != 6 {
		t.Errorf("expected 6 matches simulated, got %v", simResult["simulated_count"])
	}

	// 3. Scoring Race
	respRace, err := http.Get(ts.URL + "/api/scoring-race")
	if err != nil {
		t.Fatalf("GET /api/scoring-race failed: %v", err)
	}
	defer respRace.Body.Close()
	var race []map[string]interface{}
	_ = json.NewDecoder(respRace.Body).Decode(&race)
	if len(race) == 0 {
		t.Errorf("expected non-empty scoring race")
	}

	// 4. NXGN 50
	respNXGN, err := http.Get(ts.URL + "/api/nxgn50")
	if err != nil {
		t.Fatalf("GET /api/nxgn50 failed: %v", err)
	}
	defer respNXGN.Body.Close()
	var nxgn map[string]interface{}
	_ = json.NewDecoder(respNXGN.Body).Decode(&nxgn)
	rankings := nxgn["rankings"].([]interface{})
	if len(rankings) == 0 {
		t.Errorf("expected non-empty NXGN rankings")
	}
}

func TestServer_TransfersAndInbox(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer srv.Stop()
	defer ts.Close()

	// 1. Transfers board (React contract)
	resp, err := http.Get(ts.URL + "/api/transfers")
	if err != nil {
		t.Fatalf("GET /api/transfers failed: %v", err)
	}
	defer resp.Body.Close()
	var transfersPayload map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&transfersPayload)
	feed, _ := transfersPayload["transfer_feed"].([]interface{})
	if len(feed) == 0 {
		t.Errorf("expected transfer feed items")
	}
	if _, ok := transfersPayload["is_window_open"]; !ok {
		t.Errorf("transfers payload missing is_window_open")
	}
	if _, ok := transfersPayload["warchests"]; !ok {
		t.Errorf("transfers payload missing warchests")
	}

	// 2. Transfer records
	respRec, err := http.Get(ts.URL + "/api/transfers/records")
	if err != nil {
		t.Fatalf("GET /api/transfers/records failed: %v", err)
	}
	defer respRec.Body.Close()

	// 5. Inbox
	respInbox, err := http.Get(ts.URL + "/api/inbox")
	if err != nil {
		t.Fatalf("GET /api/inbox failed: %v", err)
	}
	defer respInbox.Body.Close()
	var inbox map[string]interface{}
	_ = json.NewDecoder(respInbox.Body).Decode(&inbox)
	items := inbox["items"].([]interface{})
	if len(items) == 0 {
		t.Errorf("expected inbox news items")
	}
}

func TestServer_WebSocketLiveMatch(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer srv.Stop()
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/match"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()

	// 1. Configure matchup
	setClubsMsg := map[string]interface{}{
		"action":  "set_clubs",
		"home_id": "LAL-BAR",
		"away_id": "LAL-RMA",
	}
	if err := conn.WriteJSON(setClubsMsg); err != nil {
		t.Fatalf("failed to write set_clubs: %v", err)
	}

	// 2. Start kickoff
	kickoffMsg := map[string]interface{}{
		"action": "kickoff",
	}
	if err := conn.WriteJSON(kickoffMsg); err != nil {
		t.Fatalf("failed to write kickoff: %v", err)
	}

	// 3. Read broadcasted ticks
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var receivedTick map[string]interface{}
	for i := 0; i < 5; i++ {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("failed to read tick from WebSocket: %v", err)
		}
		if err := json.Unmarshal(msg, &receivedTick); err == nil {
			break
		}
	}

	if receivedTick == nil {
		t.Fatalf("expected to receive tick payload from WebSocket")
	}
	if receivedTick["home_score"] == nil || receivedTick["ball"] == nil {
		t.Errorf("tick missing home_score or ball coordinates: %v", receivedTick)
	}
	homeCoords := receivedTick["home_coords"].([]interface{})
	if len(homeCoords) != 11 {
		t.Errorf("expected 11 on-pitch home coordinates, got %d", len(homeCoords))
	}

	// 4. Pause and reset
	_ = conn.WriteJSON(map[string]interface{}{"action": "pause"})
	_ = conn.WriteJSON(map[string]interface{}{"action": "reset"})
}

func TestServer_ClubSerializationAndWarchestFinances(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer srv.Stop()
	defer ts.Close()

	// 1. Verify /api/clubs serialization includes identity, finances, and warchest
	resp, err := http.Get(ts.URL + "/api/clubs")
	if err != nil {
		t.Fatalf("GET /api/clubs failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var clubs []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&clubs); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(clubs) == 0 {
		t.Fatalf("expected clubs in /api/clubs response")
	}
	clubData := clubs[0]

	identity, ok := clubData["identity"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected identity object in club payload: %v", clubData["identity"])
	}
	for _, field := range []string{
		"reputation", "historical_prestige", "financial_power", "board_patience",
		"academy_quality", "recruitment_ambition", "youth_preference",
		"transfer_aggressiveness", "selling_tendency",
	} {
		if _, exists := identity[field]; !exists {
			t.Errorf("identity missing field %s", field)
		}
	}

	finances, ok := clubData["finances"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected finances object in club payload: %v", clubData["finances"])
	}
	for _, field := range []string{"transfer_budget", "balance"} {
		if _, exists := finances[field]; !exists {
			t.Errorf("finances missing field %s", field)
		}
	}

	for _, key := range []string{"reputation", "budget_eur", "transfer_warchest_eur", "formatted_transfer_warchest"} {
		if _, exists := clubData[key]; !exists {
			t.Errorf("club payload missing %s", key)
		}
	}

	// 2. Verify /api/transfers warchests reads club finances
	respTransfers, err := http.Get(ts.URL + "/api/transfers")
	if err != nil {
		t.Fatalf("GET /api/transfers failed: %v", err)
	}
	defer respTransfers.Body.Close()
	var transfersPayload map[string]interface{}
	if err := json.NewDecoder(respTransfers.Body).Decode(&transfersPayload); err != nil {
		t.Fatalf("decode transfers: %v", err)
	}
	warchests, ok := transfersPayload["warchests"].([]interface{})
	if !ok || len(warchests) == 0 {
		t.Fatalf("expected non-empty warchests in transfers payload")
	}
	firstWarchest := warchests[0].(map[string]interface{})
	if _, ok := firstWarchest["budget_eur"]; !ok {
		t.Errorf("warchest entry missing budget_eur")
	}
	if _, ok := firstWarchest["formatted_budget"]; !ok {
		t.Errorf("warchest entry missing formatted_budget")
	}
}
