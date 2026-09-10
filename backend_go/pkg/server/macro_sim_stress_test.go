package server

import (
	"encoding/json"
	"net/http"
	"sync"
	"testing"

	"football_sim/pkg/tournament"
)

func TestConcurrentMacroWeekRequestsSerializeWithoutCorruption(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer srv.Stop()
	defer ts.Close()

	const requests = 4
	type response struct {
		status int
		body   tournament.BatchSimResult
		err    error
	}
	results := make(chan response, requests)
	var wg sync.WaitGroup
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Post(ts.URL+"/api/sim/week", "application/json", nil)
			if err != nil {
				results <- response{err: err}
				return
			}
			defer resp.Body.Close()
			var out tournament.BatchSimResult
			err = json.NewDecoder(resp.Body).Decode(&out)
			results <- response{status: resp.StatusCode, body: out, err: err}
		}()
	}
	wg.Wait()
	close(results)

	for result := range results {
		if result.err != nil {
			t.Fatalf("concurrent macro request failed: %v", result.err)
		}
		if result.status != http.StatusOK || result.body.Status != "success" {
			t.Fatalf("unexpected concurrent response: status=%d body=%+v", result.status, result.body)
		}
	}

	srv.worldMu.RLock()
	defer srv.worldMu.RUnlock()
	if got, want := srv.TournamentManager.CurrentMatchweek, requests+1; got != want {
		t.Fatalf("serialized requests advanced to MW%d, want MW%d", got, want)
	}
	if err := srv.TournamentManager.ValidateWorldState(); err != nil {
		t.Fatalf("world corrupted after concurrent macro requests: %v", err)
	}
	for _, club := range srv.TournamentManager.ClubsList {
		if club.Played != requests {
			t.Fatalf("club %s played=%d after %d serialized weeks, want %d", club.ClubID, club.Played, requests, requests)
		}
	}

	seenFinished := make(map[string]struct{})
	for _, fixture := range srv.TournamentManager.Fixtures {
		if fixture.Status != "finished" {
			continue
		}
		if _, exists := seenFinished[fixture.FixtureID]; exists {
			t.Fatalf("fixture %s appears as a duplicate finished commit", fixture.FixtureID)
		}
		seenFinished[fixture.FixtureID] = struct{}{}
	}
	if got, want := len(seenFinished), requests*6; got != want {
		t.Fatalf("finished league fixture count=%d, want %d", got, want)
	}
}

func TestTenSeasonMacroSoakValidatesEveryBoundary(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer srv.Stop()
	defer ts.Close()

	const seasons = 10
	for season := 1; season <= seasons; season++ {
		srv.worldMu.Lock()
		seasonResult, status := srv.runMacroSimulationLocked("season")
		srv.worldMu.Unlock()
		if status != http.StatusOK || seasonResult.Status != "success" || !seasonResult.SeasonFinished {
			t.Fatalf("season %d league simulation failed: status=%d result=%+v", season, status, seasonResult)
		}
		if seasonResult.Champion == "" {
			t.Fatalf("season %d completed without a champion", season)
		}

		srv.worldMu.RLock()
		if err := srv.TournamentManager.ValidateWorldState(); err != nil {
			srv.worldMu.RUnlock()
			t.Fatalf("season %d boundary failed validation: %v", season, err)
		}
		srv.worldMu.RUnlock()

		srv.worldMu.Lock()
		offseasonResult, status := srv.runMacroSimulationLocked("season")
		srv.worldMu.Unlock()
		if status != http.StatusOK || offseasonResult.Status != "success" || !offseasonResult.NewSeasonStarted {
			t.Fatalf("season %d offseason simulation failed: status=%d result=%+v", season, status, offseasonResult)
		}

		srv.worldMu.RLock()
		if srv.TournamentManager.SeasonPhase != "season" || srv.TournamentManager.CurrentMatchweek != 1 {
			srv.worldMu.RUnlock()
			t.Fatalf("season %d rollover left phase=%q MW=%d", season, srv.TournamentManager.SeasonPhase, srv.TournamentManager.CurrentMatchweek)
		}
		if err := srv.TournamentManager.ValidateWorldState(); err != nil {
			srv.worldMu.RUnlock()
			t.Fatalf("season %d new-season state failed validation: %v", season, err)
		}
		srv.worldMu.RUnlock()
	}
}
