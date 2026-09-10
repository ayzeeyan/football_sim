package server

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestPacket10FixtureNightAndHeadToHeadOnWire(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer srv.Stop()
	defer ts.Close()

	srv.worldMu.Lock()
	var leagueID, uclID, homeID, awayID string
	for i := range srv.TournamentManager.UCLFixtures {
		f := &srv.TournamentManager.UCLFixtures[i]
		if f.Status == "scheduled" {
			uclID = f.FixtureID
			break
		}
	}
	for i := range srv.TournamentManager.Fixtures {
		f := &srv.TournamentManager.Fixtures[i]
		if f.Status != "scheduled" || f.Matchweek != 1 {
			continue
		}
		leagueID, homeID, awayID = f.FixtureID, f.HomeID, f.AwayID
		break
	}
	srv.worldMu.Unlock()
	if uclID == "" || leagueID == "" {
		t.Fatal("expected scheduled league and Champions Cup fixtures")
	}

	resp, err := http.Get(ts.URL + "/api/fixtures/" + uclID)
	if err != nil {
		t.Fatalf("GET ucl fixture: %v", err)
	}
	var ucl map[string]interface{}
	decodeJSONBody(t, resp, &ucl)
	night, _ := ucl["night"].(map[string]interface{})
	if night == nil || night["kind"] != "ucl" || night["badge"] != "Champions Cup" {
		t.Fatalf("ucl fixture night = %#v", ucl["night"])
	}
	if _, ok := ucl["head_to_head"].([]interface{}); !ok {
		t.Fatalf("ucl fixture omitted head_to_head: %#v", ucl["head_to_head"])
	}

	sim, err := http.Post(ts.URL+"/api/fixtures/"+leagueID+"/simulate", "application/json", nil)
	if err != nil {
		t.Fatalf("simulate opener: %v", err)
	}
	var simBody map[string]interface{}
	decodeJSONBody(t, sim, &simBody)
	if simBody["status"] != "success" {
		t.Fatalf("simulate opener failed: %#v", simBody)
	}

	srv.worldMu.Lock()
	var reverseID string
	for i := range srv.TournamentManager.Fixtures {
		f := &srv.TournamentManager.Fixtures[i]
		if f.Status == "scheduled" && f.HomeID == awayID && f.AwayID == homeID {
			reverseID = f.FixtureID
			break
		}
	}
	srv.worldMu.Unlock()
	if reverseID == "" {
		t.Fatal("expected a return fixture")
	}
	rev, err := http.Get(ts.URL + "/api/fixtures/" + reverseID)
	if err != nil {
		t.Fatalf("GET return fixture: %v", err)
	}
	var payload map[string]interface{}
	decodeJSONBody(t, rev, &payload)
	h2h, _ := payload["head_to_head"].([]interface{})
	if len(h2h) == 0 {
		t.Fatal("return fixture head_to_head should include the finished opener")
	}
	row, _ := h2h[0].(map[string]interface{})
	if row["id"] != leagueID && row["id"] != reverseID {
		t.Fatalf("head_to_head row = %#v", row)
	}
}

func TestPacket10LiveTickIncludesBenches(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer srv.Stop()
	defer ts.Close()

	srv.worldMu.Lock()
	var homeID, awayID string
	for i := range srv.TournamentManager.Fixtures {
		f := &srv.TournamentManager.Fixtures[i]
		if f.Status == "scheduled" && f.Matchweek == srv.TournamentManager.CurrentMatchweek {
			homeID, awayID = f.HomeID, f.AwayID
			break
		}
	}
	srv.worldMu.Unlock()
	if homeID == "" {
		t.Fatal("expected a current-week league fixture")
	}

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/match"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("ws connect: %v", err)
	}
	defer conn.Close()
	if err := conn.WriteJSON(map[string]interface{}{
		"action": "set_clubs", "home_id": homeID, "away_id": awayID,
	}); err != nil {
		t.Fatalf("set_clubs: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		srv.worldMu.RLock()
		payload := srv.buildMatchTickPayload()
		srv.worldMu.RUnlock()
		home, _ := payload["home_bench"].([]map[string]interface{})
		away, _ := payload["away_bench"].([]map[string]interface{})
		if len(home) > 0 && len(away) > 0 {
			if _, ok := home[0]["player"].(map[string]interface{}); !ok {
				t.Fatalf("bench row missing player: %#v", home[0])
			}
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("live tick never included both benches")
}

func TestPacket10PlayerProfileMatchLogAndManagerLabel(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer srv.Stop()
	defer ts.Close()

	srv.worldMu.Lock()
	var fixtureID string
	for i := range srv.TournamentManager.Fixtures {
		f := &srv.TournamentManager.Fixtures[i]
		if f.Status == "scheduled" && f.Matchweek == 1 {
			fixtureID = f.FixtureID
			break
		}
	}
	srv.worldMu.Unlock()
	if fixtureID == "" {
		t.Fatal("expected a week-1 fixture")
	}
	sim, err := http.Post(ts.URL+"/api/fixtures/"+fixtureID+"/simulate", "application/json", nil)
	if err != nil {
		t.Fatalf("simulate: %v", err)
	}
	var simBody map[string]interface{}
	decodeJSONBody(t, sim, &simBody)
	if simBody["status"] != "success" {
		t.Fatalf("simulate failed: %#v", simBody)
	}

	fx, err := http.Get(ts.URL + "/api/fixtures/" + fixtureID)
	if err != nil {
		t.Fatalf("GET fixture: %v", err)
	}
	var payload map[string]interface{}
	decodeJSONBody(t, fx, &payload)
	xi, _ := payload["home_xi"].([]interface{})
	if len(xi) == 0 {
		t.Fatal("finished fixture missing home_xi")
	}
	row, _ := xi[0].(map[string]interface{})
	playerID, _ := row["player_id"].(string)
	if playerID == "" {
		t.Fatalf("home_xi row = %#v", row)
	}

	prof, err := http.Get(ts.URL + "/api/players/" + playerID)
	if err != nil {
		t.Fatalf("GET player: %v", err)
	}
	var profile map[string]interface{}
	decodeJSONBody(t, prof, &profile)
	log, _ := profile["last_matches"].([]interface{})
	if len(log) == 0 {
		t.Fatalf("player profile omitted last_matches: %#v", profile["last_matches"])
	}
	if _, ok := profile["apps_rated"]; !ok {
		t.Fatal("player profile omitted apps_rated")
	}

	clubs, err := http.Get(ts.URL + "/api/clubs")
	if err != nil {
		t.Fatalf("GET clubs: %v", err)
	}
	var list []map[string]interface{}
	decodeJSONBody(t, clubs, &list)
	if len(list) == 0 {
		t.Fatal("expected clubs")
	}
	mgr, _ := list[0]["manager"].(map[string]interface{})
	if mgr["archetype_label"] == "" || mgr["archetype"] == "" {
		t.Fatalf("manager labels missing: %#v", mgr)
	}
}

func TestPacket10ClubHistoryTrophySummary(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer srv.Stop()
	defer ts.Close()

	srv.worldMu.Lock()
	srv.TournamentManager.ClubSeasonHistory["LAL-RMA"] = []map[string]interface{}{
		{"season_name": "2024-25", "trophies": []string{"Super League Champion", "Champions Cup"}},
		{"season_name": "2025-26", "trophies": []interface{}{"Super Cup"}},
	}
	srv.worldMu.Unlock()

	resp, err := http.Get(ts.URL + "/api/clubs/LAL-RMA/history")
	if err != nil {
		t.Fatalf("GET history: %v", err)
	}
	var payload map[string]interface{}
	decodeJSONBody(t, resp, &payload)
	summary, _ := payload["trophies_summary"].(map[string]interface{})
	if summary["super_league"] != 1.0 || summary["ucl"] != 1.0 || summary["super_cup"] != 1.0 {
		t.Fatalf("trophy summary = %#v", summary)
	}
}
