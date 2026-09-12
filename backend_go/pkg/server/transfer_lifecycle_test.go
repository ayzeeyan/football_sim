package server

import (
	"net/http"
	"testing"

	"football_sim/pkg/persistence"
	"football_sim/pkg/tournament"
)

func TestTransferWindowLifecycleProgression(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	// Put the world into transfer_window phase with completed season results and reputation applied
	srv.worldMu.Lock()
	hg, ag := 1, 0
	for i := range srv.TournamentManager.Fixtures {
		srv.TournamentManager.Fixtures[i].Status = "finished"
		srv.TournamentManager.Fixtures[i].HomeGoals = &hg
		srv.TournamentManager.Fixtures[i].AwayGoals = &ag
	}
	srv.TournamentManager.UCLFixtures = nil
	srv.TournamentManager.SuperCupFixtures = []tournament.Fixture{{
		FixtureID: "SC-TEST",
		Status:    "finished",
		HomeGoals: &hg,
		AwayGoals: &ag,
	}}
	srv.TournamentManager.SuperCupFinal.WinnerID = srv.TournamentManager.ClubsList[0].ClubID
	srv.TournamentManager.SuperCupChampionID = srv.TournamentManager.ClubsList[0].ClubID
	srv.TournamentManager.CurrentMatchweek = 45
	srv.TournamentManager.SeasonPhase = "transfer_window"
	srv.TournamentManager.ReputationAppliedSeason = srv.TournamentManager.SeasonName
	srv.TransferEngine.BeginOffSeasonWindow()
	srv.worldMu.Unlock()

	// 1. Check initial transfer window state (Week 1 of 12)
	board := getTransfersBoard(t, ts.URL)
	if board["is_window_open"] != true {
		t.Fatalf("expected window open, got %v", board["is_window_open"])
	}
	if board["season_phase"] != "transfer_window" {
		t.Fatalf("season_phase=%v", board["season_phase"])
	}

	// 2. Sim Week: Week 1 -> Week 2
	simRes := postMacro(t, ts.URL, "/api/sim/week")
	if simRes.SeasonPhase != "transfer_window" {
		t.Fatalf("sim week changed season_phase unexpectedly: %v", simRes.SeasonPhase)
	}
	srv.worldMu.Lock()
	if srv.TransferEngine.CurrentWeek != 2 {
		t.Fatalf("expected transfer week 2, got %d", srv.TransferEngine.CurrentWeek)
	}
	srv.worldMu.Unlock()

	// 3. Sim Month: Week 2 -> Week 6 (advances 4 weeks)
	simRes = postMacro(t, ts.URL, "/api/sim/month")
	srv.worldMu.Lock()
	if srv.TransferEngine.CurrentWeek != 6 {
		t.Fatalf("expected transfer week 6, got %d", srv.TransferEngine.CurrentWeek)
	}
	srv.worldMu.Unlock()

	// 4. Advance to Week 7, save career, reload, and verify persistence & continuation
	advResp, err := http.Post(ts.URL+"/api/transfers/advance", "application/json", nil)
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	if advResp.StatusCode != http.StatusOK {
		t.Fatalf("advance status=%d", advResp.StatusCode)
	}
	srv.worldMu.Lock()
	if srv.TransferEngine.CurrentWeek != 7 {
		t.Fatalf("expected transfer week 7 after advance, got %d", srv.TransferEngine.CurrentWeek)
	}
	srv.worldMu.Unlock()

	// Save and restore
	srv.worldMu.Lock()
	snap := persistence.BuildSnapshot(srv.TournamentManager, srv.GrowthEngine, srv.TransferEngine)
	srv.worldMu.Unlock()

	srv.worldMu.Lock()
	// Deliberately corrupt IsOffSeason to false to test self-healing
	srv.TransferEngine.IsOffSeason = false
	if err := persistence.RestoreCareer(srv.TournamentManager, srv.GrowthEngine, srv.TransferEngine, snap); err != nil {
		t.Fatalf("RestoreCareer failed: %v", err)
	}
	if !srv.TransferEngine.IsOffSeason {
		t.Fatalf("RestoreCareer failed to heal IsOffSeason=true during transfer_window")
	}
	if srv.TransferEngine.CurrentWeek != 7 {
		t.Fatalf("RestoreCareer corrupted CurrentWeek: got %d want 7", srv.TransferEngine.CurrentWeek)
	}
	srv.worldMu.Unlock()

	// 5. Set to Week 10 and test Sim Month capping at Week 12
	srv.worldMu.Lock()
	srv.TransferEngine.CurrentWeek = 10
	srv.worldMu.Unlock()

	simRes = postMacro(t, ts.URL, "/api/sim/month")
	srv.worldMu.Lock()
	if srv.TransferEngine.CurrentWeek != 12 {
		t.Fatalf("Sim Month from week 10 should cap at week 12, got %d", srv.TransferEngine.CurrentWeek)
	}
	if !srv.TransferEngine.IsWindowOpen() {
		t.Fatalf("Week 12 should still be open for business before final sim")
	}
	srv.worldMu.Unlock()

	// 6. Sim Week at Week 12 processes final week and finalizes season transition to Season 2 (2027-28) Week 1
	simRes = postMacro(t, ts.URL, "/api/sim/week")
	if !simRes.NewSeasonStarted {
		t.Fatalf("expected NewSeasonStarted=true after final transfer week")
	}
	srv.worldMu.Lock()
	if srv.TournamentManager.SeasonPhase != "season" {
		t.Fatalf("Sim Week should have rolled over to season, got %s", srv.TournamentManager.SeasonPhase)
	}
	if srv.TournamentManager.CurrentMatchweek != 1 {
		t.Fatalf("Expected CurrentMatchweek 1 after rollover, got %d", srv.TournamentManager.CurrentMatchweek)
	}
	if srv.TournamentManager.SeasonName != "2027-28" {
		t.Fatalf("Expected Season 2027-28, got %s", srv.TournamentManager.SeasonName)
	}
	srv.worldMu.Unlock()
}
