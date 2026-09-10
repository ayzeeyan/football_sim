package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"football_sim/pkg/datamanager"
	"football_sim/pkg/matchengine"
	"football_sim/pkg/models"
)

func postTraining(t *testing.T, baseURL, playerID string) *http.Response {
	t.Helper()
	resp, err := http.Post(
		baseURL+"/api/prodigies/"+playerID+"/train",
		"application/json",
		strings.NewReader(`{"focus":"technical"}`),
	)
	if err != nil {
		t.Fatalf("POST training for %s: %v", playerID, err)
	}
	return resp
}

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

func TestTrainingErrorsReturnBadRequestWithoutMutationOrSave(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	playerID := datamanager.ProdigyStableID("Venjamin Valerio")
	valid := postTraining(t, ts.URL, playerID)
	if valid.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(valid.Body)
		valid.Body.Close()
		t.Fatalf("valid training status=%d body=%s", valid.StatusCode, body)
	}
	valid.Body.Close()

	beforeState := playerStateJSON(t, srv, playerID)
	beforeGen := srv.saveGen.Load()
	beforeSave, err := os.ReadFile(srv.savePath)
	if err != nil {
		t.Fatalf("read save after valid training: %v", err)
	}

	srv.worldMu.Lock()
	srv.GrowthEngine.TrainingEnergy = 0
	srv.worldMu.Unlock()

	exhausted := postTraining(t, ts.URL, playerID)
	exhaustedBody, _ := io.ReadAll(exhausted.Body)
	exhausted.Body.Close()
	if exhausted.StatusCode != http.StatusBadRequest {
		t.Fatalf("exhausted training status=%d body=%s", exhausted.StatusCode, exhaustedBody)
	}
	if got := playerStateJSON(t, srv, playerID); !bytes.Equal(got, beforeState) {
		t.Fatalf("exhausted training mutated player state: before=%s after=%s", beforeState, got)
	}
	srv.worldMu.RLock()
	energyAfterExhausted := srv.GrowthEngine.TrainingEnergy
	srv.worldMu.RUnlock()
	if energyAfterExhausted != 0 {
		t.Fatalf("exhausted training changed energy to %d", energyAfterExhausted)
	}
	if got := srv.saveGen.Load(); got != beforeGen {
		t.Fatalf("exhausted training advanced save generation from %d to %d", beforeGen, got)
	}
	afterSave, err := os.ReadFile(srv.savePath)
	if err != nil {
		t.Fatalf("read save after exhausted training: %v", err)
	}
	if !bytes.Equal(afterSave, beforeSave) {
		t.Fatal("exhausted training changed the persisted save")
	}

	srv.worldMu.Lock()
	srv.GrowthEngine.TrainingEnergy = srv.GrowthEngine.MaxTrainingEnergy
	srv.worldMu.Unlock()
	unknown := postTraining(t, ts.URL, "missing-prodigy")
	unknownBody, _ := io.ReadAll(unknown.Body)
	unknown.Body.Close()
	if unknown.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown training status=%d body=%s", unknown.StatusCode, unknownBody)
	}
	srv.worldMu.RLock()
	energyAfterUnknown := srv.GrowthEngine.TrainingEnergy
	maxEnergy := srv.GrowthEngine.MaxTrainingEnergy
	srv.worldMu.RUnlock()
	if energyAfterUnknown != maxEnergy {
		t.Fatalf("unknown training changed energy to %d", energyAfterUnknown)
	}
	if got := playerStateJSON(t, srv, playerID); !bytes.Equal(got, beforeState) {
		t.Fatalf("unknown training changed player state: before=%s after=%s", beforeState, got)
	}
	if got := srv.saveGen.Load(); got != beforeGen {
		t.Fatalf("unknown training advanced save generation from %d to %d", beforeGen, got)
	}
	if got, err := os.ReadFile(srv.savePath); err != nil || !bytes.Equal(got, beforeSave) {
		t.Fatalf("unknown training changed persisted save: err=%v", err)
	}
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
	if err := srv.bootFreshCareer(homes, true); err != nil {
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
	newHomeProdigy := datamanager.ProdigyStableID("Izyan Levin Bantol")
	newAwayProdigy := datamanager.ProdigyStableID("Maverick Cantalejo")
	if !newHomeIDs[newHomeProdigy] || oldHomeIDs[newHomeProdigy] {
		srv.worldMu.Unlock()
		t.Fatalf("new home identities did not replace the previous placement: old=%v new=%v", oldHomeIDs, newHomeIDs)
	}
	if !newAwayIDs[newAwayProdigy] || oldAwayIDs[newAwayProdigy] {
		srv.worldMu.Unlock()
		t.Fatalf("new away identities did not replace the previous placement: old=%v new=%v", oldAwayIDs, newAwayIDs)
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
