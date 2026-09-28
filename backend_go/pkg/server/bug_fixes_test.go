package server

import (
	"encoding/json"
	"testing"

	"football_sim/pkg/datamanager"
	"football_sim/pkg/matchengine"
	"football_sim/pkg/models"
)

func playerStateJSON(t *testing.T, srv *Server, playerID string) []byte {
	t.Helper()
	srv.worldMu.Lock()
	defer srv.worldMu.Unlock()
	player, _ := srv.findPlayer(playerID)
	if player == nil {
		t.Fatalf("player %s not found", playerID)
	}
	bio := srv.GrowthEngine.Biometrics[playerID]
	attrs := srv.GrowthEngine.Attributes[playerID]
	state, err := json.Marshal(struct {
		Player *models.Player
		Bio    interface{}
		Attrs  interface{}
	}{player, bio, attrs})
	if err != nil {
		t.Fatalf("marshal player state: %v", err)
	}
	return state
}

func TestResetLiveMatchDefaultForcesFullSnapshotAfterNewCareer(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	srv.worldMu.Lock()
	srv.resetLiveMatchDefault()
	first := srv.buildTickPayloadLocked()
	second := srv.buildTickPayloadLocked()
	if first["tick_type"] != "full" || second["tick_type"] != "delta" {
		srv.worldMu.Unlock()
		t.Fatalf("initial tick sequence = %v, %v", first["tick_type"], second["tick_type"])
	}
	oldSeq := second["seq"].(uint64)
	radarIDs := func(payload map[string]interface{}, key string) map[string]bool {
		ids := map[string]bool{}
		for _, raw := range payload[key].([]map[string]interface{}) {
			player, ok := raw["player"].(matchengine.LivePlayerRadar)
			if !ok {
				srv.worldMu.Unlock()
				t.Fatalf("%s has unexpected player type %T", key, raw["player"])
			}
			ids[player.PlayerID] = true
		}
		return ids
	}
	oldHomeIDs := radarIDs(first, "home_coords")
	oldAwayIDs := radarIDs(first, "away_coords")
	homes := datamanager.DefaultProdigyHomes()
	homes["Maverick Cantalejo"], homes["Izyan Levin Bantol"] = homes["Izyan Levin Bantol"], homes["Maverick Cantalejo"]
	if err := srv.bootFreshCareer(homes, true, 424242); err != nil {
		srv.worldMu.Unlock()
		t.Fatalf("boot fresh career: %v", err)
	}
	full := srv.buildTickPayloadLocked()
	if full["tick_type"] != "full" {
		srv.worldMu.Unlock()
		t.Fatalf("first tick after new career = %v, want full", full["tick_type"])
	}
	if full["seq"].(uint64) <= oldSeq {
		srv.worldMu.Unlock()
		t.Fatalf("new career sequence %v did not exceed %v", full["seq"], oldSeq)
	}
	newHomeIDs := radarIDs(full, "home_coords")
	newAwayIDs := radarIDs(full, "away_coords")
	// A fresh neutral career may feature any scheduled fixture. Its radar must
	// reference the newly-created engine clubs rather than a hard-coded pair.
	currentHome, currentAway := map[string]bool{}, map[string]bool{}
	for _, player := range srv.LiveMatchEngine.HomeClub.Squad {
		currentHome[player.PlayerID] = true
	}
	for _, player := range srv.LiveMatchEngine.AwayClub.Squad {
		currentAway[player.PlayerID] = true
	}
	for id := range newHomeIDs {
		if !currentHome[id] {
			srv.worldMu.Unlock()
			t.Fatalf("home radar retained stale player %q (old home=%v)", id, oldHomeIDs)
		}
	}
	for id := range newAwayIDs {
		if !currentAway[id] {
			srv.worldMu.Unlock()
			t.Fatalf("away radar retained stale player %q (old away=%v)", id, oldAwayIDs)
		}
	}
	for _, key := range []string{"home_coords", "away_coords"} {
		for _, raw := range full[key].([]map[string]interface{}) {
			if _, ok := raw["player"].(matchengine.LivePlayerRadar); !ok {
				srv.worldMu.Unlock()
				t.Fatalf("new career full %s missing current player blob: %T", key, raw["player"])
			}
		}
	}
	delta := srv.buildTickPayloadLocked()
	srv.worldMu.Unlock()
	if delta["tick_type"] != "delta" {
		t.Fatalf("tick after new career full = %v, want delta", delta["tick_type"])
	}
	if delta["seq"].(uint64) <= full["seq"].(uint64) {
		t.Fatalf("delta sequence %v did not exceed full %v", delta["seq"], full["seq"])
	}
}
