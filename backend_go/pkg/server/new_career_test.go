package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"football_sim/pkg/datamanager"
	"football_sim/pkg/persistence"
)

func decodeJSONBody(t *testing.T, resp *http.Response, dest interface{}) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

func postNewCareer(t *testing.T, tsURL string, shuffle bool, homes map[string]string) map[string]interface{} {
	t.Helper()
	body, err := json.Marshal(map[string]interface{}{"shuffle": shuffle, "homes": homes})
	if err != nil {
		t.Fatalf("marshal new career request: %v", err)
	}
	resp, err := http.Post(tsURL+"/api/career/new", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /api/career/new: %v", err)
	}
	var payload map[string]interface{}
	decodeJSONBody(t, resp, &payload)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /api/career/new status=%d payload=%v", resp.StatusCode, payload)
	}
	return payload
}

func prodigyClubIDs(t *testing.T, srv *Server) map[string]string {
	t.Helper()
	got := map[string]string{}
	for _, cfg := range datamanager.EliteProdigyConfigs {
		for _, club := range srv.TournamentManager.ClubsList {
			for _, p := range club.Squad {
				if p != nil && p.FullName == cfg.FullName && p.UniverseWonderkid {
					got[cfg.FullName] = club.ClubID
				}
			}
		}
		if got[cfg.FullName] == "" {
			t.Fatalf("prodigy %s missing from squads", cfg.FullName)
		}
	}
	return got
}

func TestDefaultHomesAndPreviewShuffleMatchReactContract(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	resp, err := http.Get(ts.URL + "/api/career/default-homes")
	if err != nil {
		t.Fatalf("GET default-homes: %v", err)
	}
	var defaults map[string]interface{}
	decodeJSONBody(t, resp, &defaults)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("default-homes status=%d", resp.StatusCode)
	}
	homes, _ := defaults["homes"].(map[string]interface{})
	draw, _ := defaults["draw"].([]interface{})
	if len(homes) != 12 || len(draw) != 12 {
		t.Fatalf("default-homes expected 12 homes and draw rows, got homes=%d draw=%d", len(homes), len(draw))
	}
	for _, cfg := range datamanager.EliteProdigyConfigs {
		if homes[cfg.FullName] != cfg.ClubID {
			t.Fatalf("default home for %s = %v, want %s", cfg.FullName, homes[cfg.FullName], cfg.ClubID)
		}
	}

	resp, err = http.Get(ts.URL + "/api/career/preview-shuffle")
	if err != nil {
		t.Fatalf("GET preview-shuffle: %v", err)
	}
	var preview map[string]interface{}
	decodeJSONBody(t, resp, &preview)
	previewHomes, _ := preview["homes"].(map[string]interface{})
	previewDraw, _ := preview["draw"].([]interface{})
	if len(previewHomes) != 12 || len(previewDraw) != 12 {
		t.Fatalf("preview-shuffle expected 12 homes and draw rows, got homes=%d draw=%d", len(previewHomes), len(previewDraw))
	}
	seen := map[string]bool{}
	for _, cid := range previewHomes {
		id, _ := cid.(string)
		if seen[id] {
			t.Fatalf("preview shuffle dealt %s twice", id)
		}
		seen[id] = true
	}
	if len(seen) != 12 {
		t.Fatalf("preview shuffle unique clubs=%d, want 12", len(seen))
	}
	srv.worldMu.RLock()
	liveHomes := prodigyClubIDs(t, srv)
	srv.worldMu.RUnlock()
	for name, cid := range datamanager.DefaultProdigyHomes() {
		if liveHomes[name] != cid {
			t.Fatalf("preview shuffle mutated the live world for %s: got %s", name, liveHomes[name])
		}
	}
}

func TestNewCareerKeepHomesWipesTableAndGrowth(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	srv.worldMu.Lock()
	before := srv.TournamentManager.SimulateRemaining()
	srv.worldMu.Unlock()
	if before["status"] != "success" {
		t.Fatalf("setup simulate remaining: %v", before)
	}
	if srv.TournamentManager.ClubsList[0].Played == 0 {
		t.Fatal("expected standings to move before the wipe")
	}

	payload := postNewCareer(t, ts.URL, false, nil)
	if payload["status"] != "success" || payload["season_name"] != "2026-27" {
		t.Fatalf("new career payload: %v", payload)
	}
	if payload["current_matchweek"].(float64) != 1 || payload["max_matchweeks"].(float64) != 44 {
		t.Fatalf("calendar after new career: %v", payload)
	}

	srv.worldMu.RLock()
	defer srv.worldMu.RUnlock()
	if srv.TournamentManager.CurrentMatchweek != 1 || srv.TournamentManager.SeasonName != "2026-27" {
		t.Fatalf("live calendar: week=%d season=%s", srv.TournamentManager.CurrentMatchweek, srv.TournamentManager.SeasonName)
	}
	for _, club := range srv.TournamentManager.ClubsList {
		if club.Played != 0 || club.Points != 0 {
			t.Fatalf("%s still has table stats after new career: played=%d points=%d", club.ClubName, club.Played, club.Points)
		}
	}
	got := prodigyClubIDs(t, srv)
	for name, cid := range datamanager.DefaultProdigyHomes() {
		if got[name] != cid {
			t.Fatalf("keep-homes placed %s at %s, want %s", name, got[name], cid)
		}
		club := srv.TournamentManager.Clubs[cid]
		for _, p := range club.Squad {
			if p.FullName == name {
				if p.Appearances != 0 || p.Goals != 0 {
					t.Fatalf("%s kept season stats appearances=%d goals=%d", name, p.Appearances, p.Goals)
				}
				bio := srv.GrowthEngine.Biometrics[p.PlayerID]
				if bio == nil || bio.AccumulatedXP != 0 {
					t.Fatalf("%s growth was not reset: %+v", name, bio)
				}
				break
			}
		}
	}
	if srv.liveFixtureID != "" || srv.LiveMatchEngine.State != "NOT_STARTED" {
		t.Fatalf("live session was not cleared: fixture=%s state=%s", srv.liveFixtureID, srv.LiveMatchEngine.State)
	}
}

func TestNewCareerHonorsPreviewedHomesAndPersistsThem(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	resp, err := http.Get(ts.URL + "/api/career/preview-shuffle")
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	var preview struct {
		Homes map[string]string            `json:"homes"`
		Draw  []datamanager.ProdigyDrawRow `json:"draw"`
	}
	decodeJSONBody(t, resp, &preview)
	if len(preview.Homes) != 12 {
		t.Fatalf("preview homes=%d", len(preview.Homes))
	}

	payload := postNewCareer(t, ts.URL, true, preview.Homes)
	returned, _ := payload["homes"].(map[string]interface{})
	for name, cid := range preview.Homes {
		if returned[name] != cid {
			t.Fatalf("new career reshuffled %s: got %v want %s", name, returned[name], cid)
		}
	}

	srv.worldMu.RLock()
	got := prodigyClubIDs(t, srv)
	srv.worldMu.RUnlock()
	for name, cid := range preview.Homes {
		if got[name] != cid {
			t.Fatalf("squad placement for %s = %s, want previewed %s", name, got[name], cid)
		}
	}

	saved, err := persistence.LoadCareer(srv.savePath)
	if err != nil {
		t.Fatalf("load career: %v", err)
	}
	if saved.CurrentMatchweek != 1 {
		t.Fatalf("saved matchweek=%d", saved.CurrentMatchweek)
	}
	for name, cid := range preview.Homes {
		if saved.ProdigyHomes[name] != cid {
			t.Fatalf("saved homes for %s = %s, want %s", name, saved.ProdigyHomes[name], cid)
		}
	}
}

func TestNewCareerInvalidHomesFallBackToDefault(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	payload := postNewCareer(t, ts.URL, true, map[string]string{"Venjamin Valerio": "NOPE"})
	returned, _ := payload["homes"].(map[string]interface{})
	for name, cid := range datamanager.DefaultProdigyHomes() {
		if returned[name] != cid {
			t.Fatalf("invalid homes kept %s at %v, want default %s", name, returned[name], cid)
		}
	}
	srv.worldMu.RLock()
	defer srv.worldMu.RUnlock()
	got := prodigyClubIDs(t, srv)
	for name, cid := range datamanager.DefaultProdigyHomes() {
		if got[name] != cid {
			t.Fatalf("invalid homes placed %s at %s", name, got[name])
		}
	}
}
