package server

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// F3: full snapshot on kickoff/goal/join, deltas otherwise, smaller ordinary
// ticks, monotonically increasing seq.
func TestDeltaTickProtocol(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	srv.worldMu.Lock()
	srv.LiveMatchEngine.ResetMatch()
	srv.LiveMatchEngine.Kickoff()
	// Advance so momentum/trail/ball fields are live, without guaranteeing
	// any random events.
	for i := 0; i < 120; i++ {
		srv.LiveMatchEngine.Tick(0.016)
	}

	first := srv.buildTickPayloadLocked()
	if first["tick_type"] != "full" {
		t.Fatalf("first tick after kickoff should be full, got %v", first["tick_type"])
	}
	second := srv.buildTickPayloadLocked()
	if second["tick_type"] != "delta" {
		t.Fatalf("ordinary tick should be delta, got %v", second["tick_type"])
	}

	// A goal forces the next tick back to a full snapshot.
	srv.LiveMatchEngine.HomeScore++
	third := srv.buildTickPayloadLocked()
	if third["tick_type"] != "full" {
		t.Fatalf("tick after a goal should be full, got %v", third["tick_type"])
	}
	if third["home_score"] != second["home_score"].(int)+1 {
		t.Fatalf("full tick after goal has stale score: %v", third["home_score"])
	}
	fourth := srv.buildTickPayloadLocked()
	if fourth["tick_type"] != "delta" {
		t.Fatalf("tick after the goal full should be delta, got %v", fourth["tick_type"])
	}

	// Seq must increase on every broadcast.
	seqs := []uint64{
		first["seq"].(uint64), second["seq"].(uint64),
		third["seq"].(uint64), fourth["seq"].(uint64),
	}
	for i := 1; i < len(seqs); i++ {
		if seqs[i] <= seqs[i-1] {
			t.Fatalf("seq not increasing: %v", seqs)
		}
	}

	// Delta carries coords + score + last event + ball/momentum/trail...
	for _, key := range []string{
		"home_coords", "away_coords", "home_score", "away_score",
		"ball", "possession_momentum", "pass_trail", "last_event", "event_count",
	} {
		if _, ok := fourth[key]; !ok {
			t.Fatalf("delta tick missing %q", key)
		}
	}
	// ...but no full player blobs and no full commentary/events arrays.
	for _, coordsKey := range []string{"home_coords", "away_coords"} {
		for _, raw := range fourth[coordsKey].([]map[string]interface{}) {
			if _, ok := raw["player"]; ok {
				t.Fatalf("delta %s ships full player blobs", coordsKey)
			}
			if _, ok := raw["player_id"]; !ok {
				t.Fatalf("delta %s missing player_id", coordsKey)
			}
		}
	}
	if _, ok := fourth["commentary"]; ok {
		t.Fatal("delta tick must not ship full commentary")
	}
	if _, ok := fourth["match_events"]; ok {
		t.Fatal("delta tick must not ship full match_events")
	}

	// Ordinary ticks must be substantially smaller than full snapshots.
	fullBytes, _ := json.Marshal(first)
	deltaBytes, _ := json.Marshal(second)
	srv.worldMu.Unlock()
	t.Logf("full=%d bytes delta=%d bytes", len(fullBytes), len(deltaBytes))
	if len(deltaBytes)*2 >= len(fullBytes) {
		t.Fatalf("delta (%dB) not under half of full (%dB)", len(deltaBytes), len(fullBytes))
	}
}

// F3: joining mid-match must yield a full snapshot before painting.
func TestLateJoinReceivesFullSnapshot(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	srv.worldMu.Lock()
	srv.LiveMatchEngine.ResetMatch()
	srv.LiveMatchEngine.Kickoff()
	srv.worldMu.Unlock()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/match"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect websocket: %v", err)
	}
	defer conn.Close()

	deadline := time.Now().Add(5 * time.Second)
	for {
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		var msg map[string]interface{}
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("no tick received after join: %v", err)
		}
		if msg["tick_type"] == "delta" {
			if time.Now().After(deadline) {
				t.Fatal("joining mid-match never yielded a full snapshot")
			}
			continue
		}
		if msg["tick_type"] != "full" {
			t.Fatalf("unexpected tick_type %v", msg["tick_type"])
		}
		home, _ := msg["home_coords"].([]interface{})
		if len(home) == 0 {
			t.Fatal("full snapshot has empty players")
		}
		first, _ := home[0].(map[string]interface{})
		player, _ := first["player"].(map[string]interface{})
		if player == nil || player["player_id"] == "" {
			t.Fatal("full snapshot coords lack player identities")
		}
		return
	}
}
