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
