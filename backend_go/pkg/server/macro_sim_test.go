package server

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"football_sim/pkg/models"
	"football_sim/pkg/tournament"
	"football_sim/pkg/transfers"
)

func postMacro(t *testing.T, baseURL, path string) tournament.BatchSimResult {
	t.Helper()
	resp, err := http.Post(baseURL+path, "application/json", nil)
	if err != nil { t.Fatalf("POST %s failed: %v", path, err) }
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK { t.Fatalf("POST %s status=%d", path, resp.StatusCode) }
	var out tournament.BatchSimResult
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil { t.Fatalf("decode %s: %v", path, err) }
	return out
}

type clubTableState struct { ClubID string; Played, Won, Drawn, Lost, GoalsFor, GoalsAgainst, Points int }
type macroLogicalState struct {
	SeasonName string; SeasonPhase string; CurrentMatchweek int; TransferWeek int; CompletedFixtures int
	LiveFixtureID string; LastCommittedInstance int; ManagerHistoryLen int; SeasonHistoryLen int; Standings []clubTableState
}

func captureMacroLogicalState(srv *Server) macroLogicalState {
	completed := 0
	for _, fixture := range srv.TournamentManager.Fixtures { if fixture.Status == "finished" { completed++ } }
	transferWeek := 0
	if srv.TransferEngine != nil { transferWeek = srv.TransferEngine.CurrentWeek }
	standings := make([]clubTableState, 0, len(srv.TournamentManager.ClubsList))
	for _, club := range srv.TournamentManager.ClubsList {
		if club == nil { continue }
		standings = append(standings, clubTableState{ClubID: club.ClubID, Played: club.Played, Won: club.Won, Drawn: club.Drawn, Lost: club.Lost, GoalsFor: club.GoalsFor, GoalsAgainst: club.GoalsAgainst, Points: club.Points})
	}
	return macroLogicalState{SeasonName: srv.TournamentManager.SeasonName, SeasonPhase: srv.TournamentManager.SeasonPhase, CurrentMatchweek: srv.TournamentManager.CurrentMatchweek, TransferWeek: transferWeek, CompletedFixtures: completed, LiveFixtureID: srv.liveFixtureID, LastCommittedInstance: srv.lastCommittedLiveInstance, ManagerHistoryLen: len(srv.TournamentManager.ManagerHistory), SeasonHistoryLen: len(srv.TournamentManager.SeasonHistory), Standings: standings}
}

func postMacroExpectConflict(t *testing.T, baseURL, path string) string {
	t.Helper()
	resp, err := http.Post(baseURL+path, "application/json", nil)
	if err != nil { t.Fatalf("POST %s failed: %v", path, err) }
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict { t.Fatalf("expected HTTP 409 Conflict for %s, got %d", path, resp.StatusCode) }
	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil { t.Fatalf("decode conflict response for %s: %v", path, err) }
	if len(body) != 1 { t.Fatalf("expected consistent single-field error response for %s, got %#v", path, body) }
	if body["message"] != liveFixtureConflictMessage { t.Fatalf("unexpected conflict message for %s: %q", path, body["message"]) }
	return body["message"]
}

func TestMacroSimWeekEndpoint(t *testing.T) {
	srv, ts := setupTestServer(t); defer srv.Stop(); defer ts.Close()
	out := postMacro(t, ts.URL, "/api/sim/week")
	if out.Status != "success" || out.Mode != "week" { t.Fatalf("unexpected response: %+v", out) }
	if out.WeeksAdvanced != 1 || len(out.Digests) != 1 { t.Fatalf("weeks=%d digests=%d, want 1/1", out.WeeksAdvanced, len(out.Digests)) }
	if out.CurrentMatchweek != 2 || srv.TournamentManager.CurrentMatchweek != 2 { t.Fatalf("current matchweek response=%d manager=%d, want 2", out.CurrentMatchweek, srv.TournamentManager.CurrentMatchweek) }
	if out.Digests[0].CalendarLabel == "" || out.Digests[0].Played == 0 { t.Fatalf("incomplete digest: %+v", out.Digests[0]) }
}

func TestMacroOffSeasonWeekRollsIntoNewSeason(t *testing.T) {
	srv, ts := setupTestServer(t); defer srv.Stop(); defer ts.Close()
	prepareCompletedSeasonForMacro(srv.TournamentManager)
	srv.TransferEngine.IsOffSeason = true
	srv.TransferEngine.CurrentWeek = transfers.TransferWindowWeeks
	oldSeason := srv.TournamentManager.SeasonName
	out := postMacro(t, ts.URL, "/api/sim/week")
	if !out.NewSeasonStarted { t.Fatalf("expected new season start: %+v", out) }
	if out.SeasonName == oldSeason || out.SeasonPhase != "season" || out.CurrentMatchweek != 1 { t.Fatalf("bad rollover: old=%s out=%+v", oldSeason, out) }
	// The completed window remains represented as week 13 while the market is
	// closed. Its transfer locks reset only when the next offseason begins.
	if srv.TransferEngine.CurrentWeek != transfers.TransferWindowWeeks+1 || srv.TransferEngine.IsOffSeason {
		t.Fatalf("completed transfer window state wrong: week=%d offseason=%v", srv.TransferEngine.CurrentWeek, srv.TransferEngine.IsOffSeason)
	}
}

func prepareCompletedSeasonForMacro(tm *tournament.TournamentManager) {
	for i := range tm.Fixtures {
		hg, ag := 0, 0
		tm.Fixtures[i].Status = "finished"
		tm.Fixtures[i].HomeGoals = &hg
		tm.Fixtures[i].AwayGoals = &ag
	}
	for i, club := range tm.ClubsList {
		club.Played = tournament.LeagueRounds
		club.Points = 0
		club.GoalDifference = 0
		if i == 0 { club.Points = 100 }
	}
	tm.UCLFixtures = nil
	tm.SuperCupFixtures = nil
	tm.CurrentMatchweek = tm.MaxMatchweeks + 1
	tm.SeasonPhase = "transfer_window"
}

func TestMacroSimRejectedWithoutMutationForEveryEndpoint(t *testing.T) {
	endpoints := []string{"/api/sim/week", "/api/sim/month", "/api/sim/season"}
	for _, endpoint := range endpoints {
		endpoint := endpoint
		t.Run(endpoint+"_active", func(t *testing.T) {
			srv, ts := setupTestServer(t); defer srv.Stop(); defer ts.Close()
			home := srv.TournamentManager.ClubsList[0]; away := srv.TournamentManager.ClubsList[1]
			srv.LiveMatchEngine.SetClubs(home, away, srv.TournamentManager.Managers[home.ClubID], srv.TournamentManager.Managers[away.ClubID])
			srv.liveFixtureID = srv.TournamentManager.Fixtures[0].FixtureID; srv.LiveMatchEngine.State = "PLAYING"
			before := captureMacroLogicalState(srv); postMacroExpectConflict(t, ts.URL, endpoint); after := captureMacroLogicalState(srv)
			if !reflect.DeepEqual(before, after) { t.Fatalf("state mutated on rejected active-live request\nbefore=%+v\nafter=%+v", before, after) }
			srv.LiveMatchEngine.State = "NOT_STARTED"; srv.clearLiveFixtureSelection()
			out := postMacro(t, ts.URL, endpoint); if out.Status != "success" { t.Fatalf("expected %s to succeed after clearing live fixture, got %+v", endpoint, out) }
		})
		t.Run(endpoint+"_full_time_uncommitted", func(t *testing.T) {
			srv, ts := setupTestServer(t); defer srv.Stop(); defer ts.Close()
			home := srv.TournamentManager.ClubsList[0]; away := srv.TournamentManager.ClubsList[1]
			srv.LiveMatchEngine.SetClubs(home, away, srv.TournamentManager.Managers[home.ClubID], srv.TournamentManager.Managers[away.ClubID])
			srv.liveFixtureID = srv.TournamentManager.Fixtures[0].FixtureID; srv.LiveMatchEngine.State = "FULL_TIME"; srv.LiveMatchEngine.InstanceID = 99; srv.lastCommittedLiveInstance = 0
			before := captureMacroLogicalState(srv); postMacroExpectConflict(t, ts.URL, endpoint); after := captureMacroLogicalState(srv)
			if !reflect.DeepEqual(before, after) { t.Fatalf("state mutated on rejected uncommitted-FULL_TIME request\nbefore=%+v\nafter=%+v", before, after) }
			srv.lastCommittedLiveInstance = 99; srv.LiveMatchEngine.State = "NOT_STARTED"
			out := postMacro(t, ts.URL, endpoint); if out.Status != "success" { t.Fatalf("expected %s to succeed after committing live instance, got %+v", endpoint, out) }
		})
	}
}

func TestMacroSimUnknownPhaseGuard(t *testing.T) {
	srv, ts := setupTestServer(t); defer srv.Stop(); defer ts.Close()
	srv.TournamentManager.SeasonPhase = "INVALID_UNKNOWN_PHASE"
	before := captureMacroLogicalState(srv)
	resp, err := http.Post(ts.URL+"/api/sim/week", "application/json", nil)
	if err != nil { t.Fatalf("POST /api/sim/week failed: %v", err) }
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError { t.Fatalf("expected HTTP 500 for unknown phase, got %d", resp.StatusCode) }
	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil { t.Fatalf("decode unknown-phase response: %v", err) }
	if body["message"] == "" { t.Fatalf("expected descriptive unknown-phase error, got %#v", body) }
	after := captureMacroLogicalState(srv)
	if !reflect.DeepEqual(before, after) { t.Fatalf("unknown phase request mutated state\nbefore=%+v\nafter=%+v", before, after) }
}

func TestFirstTransferWindowBudgetUsesCompletedSeasonReputation(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer srv.Stop(); defer ts.Close()
	srv.worldMu.Lock()
	defer srv.worldMu.Unlock()
	tm := srv.TournamentManager
	prepareCompletedSeasonForMacro(tm)
	for _, club := range tm.ClubsList {
		for _, player := range club.Squad { player.OVR = 50 }
	}
	beforeRep := tm.ClubsList[0].Identity.Reputation
	out, status := srv.runMacroSimulationLocked("week")
	if status != http.StatusOK || out.Status != "success" { t.Fatalf("first offseason week failed: status=%d out=%+v", status, out) }
	if tm.ReputationAppliedSeason != tm.SeasonName { t.Fatalf("reputation marker=%q want %q", tm.ReputationAppliedSeason, tm.SeasonName) }
	if tm.ClubsList[0].Identity.Reputation == beforeRep { t.Fatal("completed season did not change champion reputation") }
	identity := tm.ClubsList[0].Identity.Clamp()
	millions := int64(20 + (identity.FinancialPower*13)/10 + (identity.Reputation*3)/10)
	if millions < 25 { millions = 25 }
	if millions > 180 { millions = 180 }
	expected := millions * models.EuroMillion
	if tm.ClubsList[0].Finances.Balance > 0 && expected > tm.ClubsList[0].Finances.Balance { expected = tm.ClubsList[0].Finances.Balance }
	if got := tm.ClubsList[0].Finances.TransferBudget; got != expected { t.Fatalf("week 1 budget=%d want %d from updated reputation=%d", got, expected, identity.Reputation) }
}

func TestMacroTransferWindowWaitsForIncompleteCup(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer srv.Stop(); defer ts.Close()
	srv.worldMu.Lock()
	tm := srv.TournamentManager
	prepareCompletedSeasonForMacro(tm)
	tm.UCLFixtures = []tournament.Fixture{{FixtureID: "MACRO-UCL-FINAL", Competition: "ucl", Stage: "Final", HomeID: tm.ClubsList[0].ClubID, AwayID: tm.ClubsList[1].ClubID, Status: "scheduled"}}
	beforeRep := tm.ClubsList[0].Identity.Reputation
	beforeWindow, beforeWeek := srv.TransferEngine.IsOffSeason, srv.TransferEngine.CurrentWeek
	_, status := srv.runMacroSimulationLocked("week")
	if status != http.StatusConflict { srv.worldMu.Unlock(); t.Fatalf("incomplete cup should block window start with HTTP 409, got %d", status) }
	if tm.ClubsList[0].Identity.Reputation != beforeRep || srv.TransferEngine.IsOffSeason != beforeWindow || srv.TransferEngine.CurrentWeek != beforeWeek {
		srv.worldMu.Unlock()
		t.Fatalf("blocked window changed state: reputation=%d/%d offseason=%v/%v week=%d/%d", tm.ClubsList[0].Identity.Reputation, beforeRep, srv.TransferEngine.IsOffSeason, beforeWindow, srv.TransferEngine.CurrentWeek, beforeWeek)
	}
	srv.worldMu.Unlock()
}

func TestSeasonResetRouteRequiresTwelveTransferWeeks(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer srv.Stop(); defer ts.Close()
	srv.worldMu.Lock()
	srv.TournamentManager.SeasonPhase = "transfer_window"
	season := srv.TournamentManager.SeasonName
	age := srv.TournamentManager.ClubsList[0].Squad[0].Age
	srv.TransferEngine.CurrentWeek = 1
	srv.worldMu.Unlock()
	resp, err := http.Post(ts.URL+"/api/season/reset", "application/json", nil)
	if err != nil { t.Fatalf("reset request failed: %v", err) }
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest { t.Fatalf("expected guarded reset status 400, got %d", resp.StatusCode) }
	srv.worldMu.RLock()
	defer srv.worldMu.RUnlock()
	if srv.TournamentManager.SeasonName != season || srv.TournamentManager.ClubsList[0].Squad[0].Age != age {
		t.Fatalf("guarded reset mutated season=%q/%q age=%d/%d", srv.TournamentManager.SeasonName, season, srv.TournamentManager.ClubsList[0].Squad[0].Age, age)
	}
}
