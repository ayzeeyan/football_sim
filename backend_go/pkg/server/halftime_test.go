package server

import (
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestHalfTimeCommandPublishesXIAndRejectsDuplicates(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()
	srv.worldMu.Lock()
	e := srv.LiveMatchEngine
	e.ResetMatch()
	e.Kickoff()
	e.CurrentMinute = 44.99
	e.Update(0.1)
	out, in := e.AwayStarters[5].PlayerID, e.AwayBench[0].PlayerID
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
	send := func(cmd map[string]interface{}) {
		t.Helper()
		if err := conn.WriteJSON(cmd); err != nil {
			t.Fatal(err)
		}
	}
	// A speed update is an observable barrier after invalid commands.
	send(map[string]interface{}{"action": "halftime_resume", "side": "away", "stance": "OVERLOAD", "out_id": out, "in_id": "invalid"})
	send(map[string]interface{}{"action": "kickoff"})
	send(map[string]interface{}{"action": "pause"})
	send(map[string]interface{}{"action": "set_speed", "speed": 2})
	for {
		tick := readLiveCompletionTick(t, conn)
		if tick["speed"] != float64(2) {
			continue
		}
		if tick["state"] != "HALF_TIME" || tick["minute"] != float64(45) || tick["away_tactical_stance"] != "NORMAL" {
			t.Fatalf("invalid command mutated match: %v", tick["state"])
		}
		break
	}
	cmd := map[string]interface{}{"action": "halftime_resume", "side": "away", "stance": "OVERLOAD", "out_id": out, "in_id": in}
	send(cmd)
	send(cmd)
	for {
		tick := readLiveCompletionTick(t, conn)
		if tick["state"] != "PLAYING" {
			continue
		}
		if tick["tick_type"] != "full" || tick["away_tactical_stance"] != "OVERLOAD" {
			t.Fatal("resume did not publish full stance snapshot")
		}
		found := false
		for _, raw := range tick["away_coords"].([]interface{}) {
			p := raw.(map[string]interface{})["player"].(map[string]interface{})
			if p["player_id"] == out {
				t.Fatal("outgoing player still in XI")
			}
			if p["player_id"] == in {
				found = true
			}
		}
		if !found {
			t.Fatal("substitute not in snapshot")
		}
		break
	}
	send(map[string]interface{}{"action": "set_speed", "speed": 999})
	for {
		tick := readLiveCompletionTick(t, conn)
		if tick["state"] == "FULL_TIME" {
			break
		}
	}
	srv.worldMu.RLock()
	defer srv.worldMu.RUnlock()
	count := 0
	for _, event := range e.Events {
		if event.Type == "sub" && event.PlayerIn != nil && event.PlayerIn.PlayerID == in {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("substitute entered %d times", count)
	}
}
