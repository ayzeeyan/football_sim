package server

import "testing"

// F1: the WS tick must surface momentum / trail / ball flight, not stubs.
func TestMatchTickPayloadSurfacesLiveFields(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	srv.worldMu.Lock()
	// Mirror the real set_clubs flow: rosters snapshot on reset, then kickoff.
	srv.LiveMatchEngine.ResetMatch()
	srv.LiveMatchEngine.Kickoff()
	for i := 0; i < 600 && srv.LiveMatchEngine.PassTrail == nil; i++ {
		srv.LiveMatchEngine.Tick(0.016)
	}
	srv.LiveMatchEngine.ResolveShot(
		srv.LiveMatchEngine.HomeClub, srv.LiveMatchEngine.AwayClub,
		srv.LiveMatchEngine.HomeStarters, srv.LiveMatchEngine.AwayStarters,
	)
	payload := srv.buildMatchTickPayload()
	srv.worldMu.Unlock()

	if payload["possession_momentum"] == 0.0 {
		t.Fatal("possession_momentum stuck at 0 in tick payload")
	}
	if payload["pass_trail"] == nil {
		t.Fatal("pass_trail is nil in tick payload during live play")
	}
	ball, _ := payload["ball"].(map[string]interface{})
	if ball == nil {
		t.Fatal("ball missing from tick payload")
	}
	if h, _ := ball["height"].(float64); h <= 0 {
		t.Fatalf("ball height not elevated on a shot tick: %v", ball["height"])
	}
	if shot, _ := ball["is_shot"].(bool); !shot {
		t.Fatal("ball is_shot false on a shot tick")
	}
}
