package server

import (
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestRepeatedLiveSelectionDoesNotResetMatch(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	fixture, _ := serverLiveCompletionFixtureWithProdigy(t, srv)
	home := srv.TournamentManager.Clubs[fixture.HomeID]
	away := srv.TournamentManager.Clubs[fixture.AwayID]

	srv.worldMu.Lock()
	srv.LiveMatchEngine.SetClubs(home, away, srv.TournamentManager.Managers[fixture.HomeID], srv.TournamentManager.Managers[fixture.AwayID])
	srv.liveFixtureID = fixture.FixtureID
	srv.LiveMatchEngine.State = "PLAYING"
	srv.LiveMatchEngine.CurrentMinute = 23
	srv.worldMu.Unlock()

	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(ts.URL, "http")+"/ws/match", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	readLiveCompletionTick(t, conn)

	if err := conn.WriteJSON(map[string]interface{}{
		"action": "set_clubs", "home_id": fixture.HomeID, "away_id": fixture.AwayID, "fixture_id": fixture.FixtureID,
	}); err != nil {
		t.Fatal(err)
	}
	// This creates an observable barrier after set_clubs has been handled.
	if err := conn.WriteJSON(map[string]interface{}{"action": "set_speed", "speed": 2}); err != nil {
		t.Fatal(err)
	}

	for {
		tick := readLiveCompletionTick(t, conn)
		if tick["speed"] != float64(2) {
			continue
		}
		if tick["state"] != "PLAYING" {
			t.Fatalf("repeated selection reset live state: %v", tick["state"])
		}
		if tick["minute"].(float64) < 23 {
			t.Fatalf("repeated selection reset live minute: %v", tick["minute"])
		}
		break
	}
}
