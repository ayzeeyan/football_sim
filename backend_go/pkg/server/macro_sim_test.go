package server

import (
	"encoding/json"
	"net/http"
	"testing"

	"football_sim/pkg/tournament"
)

func postMacro(t *testing.T, baseURL, path string) tournament.BatchSimResult {
	t.Helper()
	resp, err := http.Post(baseURL+path, "application/json", nil)
	if err != nil {
		t.Fatalf("POST %s failed: %v", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST %s status=%d", path, resp.StatusCode)
	}
	var out tournament.BatchSimResult
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return out
}

func TestMacroSimWeekEndpoint(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer srv.Stop()
	defer ts.Close()

	out := postMacro(t, ts.URL, "/api/sim/week")
	if out.Status != "success" || out.Mode != "week" {
		t.Fatalf("unexpected response: %+v", out)
	}
	if out.WeeksAdvanced != 1 || len(out.Digests) != 1 {
		t.Fatalf("weeks=%d digests=%d, want 1/1", out.WeeksAdvanced, len(out.Digests))
	}
	if out.CurrentMatchweek != 2 || srv.TournamentManager.CurrentMatchweek != 2 {
		t.Fatalf("current matchweek response=%d manager=%d, want 2", out.CurrentMatchweek, srv.TournamentManager.CurrentMatchweek)
	}
	if out.Digests[0].CalendarLabel == "" || out.Digests[0].Played == 0 {
		t.Fatalf("incomplete digest: %+v", out.Digests[0])
	}
}

func TestMacroOffSeasonWeekRollsIntoNewSeason(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer srv.Stop()
	defer ts.Close()

	srv.TournamentManager.SeasonPhase = "transfer_window"
	srv.TransferEngine.IsOffSeason = true
	srv.TransferEngine.CurrentWeek = 12
	oldSeason := srv.TournamentManager.SeasonName

	out := postMacro(t, ts.URL, "/api/sim/week")
	if !out.NewSeasonStarted {
		t.Fatalf("expected new season start: %+v", out)
	}
	if out.SeasonName == oldSeason || out.SeasonPhase != "season" || out.CurrentMatchweek != 1 {
		t.Fatalf("bad rollover: old=%s out=%+v", oldSeason, out)
	}
	if srv.TransferEngine.CurrentWeek != 1 || srv.TransferEngine.IsOffSeason {
		t.Fatalf("transfer engine not reset: week=%d offseason=%v", srv.TransferEngine.CurrentWeek, srv.TransferEngine.IsOffSeason)
	}
}

func TestMacroSimRejectedWhenLiveFixtureActiveOrUncommitted(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer srv.Stop()
	defer ts.Close()

	// Select a live fixture and set engine to active state PLAYING
	home := srv.TournamentManager.ClubsList[0]
	away := srv.TournamentManager.ClubsList[1]
	srv.LiveMatchEngine.SetClubs(home, away, srv.TournamentManager.Managers[home.ClubID], srv.TournamentManager.Managers[away.ClubID])
	srv.liveFixtureID = srv.TournamentManager.Fixtures[0].FixtureID
	srv.LiveMatchEngine.State = "PLAYING"

	endpoints := []string{"/api/sim/week", "/api/sim/month", "/api/sim/season"}

	// 1. Sim Week / Month / Season rejected with 409 while PLAYING
	for _, ep := range endpoints {
		resp, err := http.Post(ts.URL+ep, "application/json", nil)
		if err != nil {
			t.Fatalf("POST %s failed: %v", ep, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusConflict {
			t.Fatalf("expected HTTP 409 Conflict for %s while PLAYING, got %d", ep, resp.StatusCode)
		}
	}

	// Verify state remained unchanged (CurrentMatchweek still 1)
	if srv.TournamentManager.CurrentMatchweek != 1 {
		t.Fatalf("CurrentMatchweek changed during rejection: got %d, want 1", srv.TournamentManager.CurrentMatchweek)
	}

	// 2. Sim Week rejected with 409 at FULL_TIME when uncommitted
	srv.LiveMatchEngine.State = "FULL_TIME"
	srv.LiveMatchEngine.InstanceID = 99
	srv.lastCommittedLiveInstance = 0 // instance 99 not committed yet

	resp, err := http.Post(ts.URL+"/api/sim/week", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /api/sim/week failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected HTTP 409 Conflict at FULL_TIME uncommitted, got %d", resp.StatusCode)
	}

	// 3. Sim Week succeeds once committed or selection cleared
	srv.lastCommittedLiveInstance = 99 // mark as committed
	srv.LiveMatchEngine.State = "NOT_STARTED"

	out := postMacro(t, ts.URL, "/api/sim/week")
	if out.Status != "success" || out.CurrentMatchweek != 2 {
		t.Fatalf("expected macro sim success after commit, got status=%s mw=%d", out.Status, out.CurrentMatchweek)
	}
}

func TestMacroSimUnknownPhaseGuard(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer srv.Stop()
	defer ts.Close()

	srv.TournamentManager.SeasonPhase = "INVALID_UNKNOWN_PHASE"

	resp, err := http.Post(ts.URL+"/api/sim/week", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /api/sim/week failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected HTTP 500 for unknown phase, got %d", resp.StatusCode)
	}
}
