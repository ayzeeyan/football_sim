package server

import (
	"net/http"
	"sync"
	"testing"

	"football_sim/pkg/models"
	"football_sim/pkg/persistence"
	"football_sim/pkg/tournament"
)

func TestHardeningWorldValidation(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer srv.Stop()
	defer ts.Close()

	if err := srv.TournamentManager.ValidateWorldState(); err != nil {
		t.Fatalf("fresh test world failed validation: %v", err)
	}

	// Corrupt a club ID and verify validation catches it
	origID := srv.TournamentManager.ClubsList[0].ClubID
	srv.TournamentManager.ClubsList[0].ClubID = ""
	if err := srv.TournamentManager.ValidateWorldState(); err == nil {
		t.Fatalf("expected validation error for empty ClubID, got nil")
	}
	srv.TournamentManager.ClubsList[0].ClubID = origID
}

func TestHardeningSaveRoundTripRestoration(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer srv.Stop()
	defer ts.Close()

	// Advance 2 weeks
	postMacro(t, ts.URL, "/api/sim/week")
	postMacro(t, ts.URL, "/api/sim/week")

	tmBefore := srv.TournamentManager
	mwBefore := tmBefore.CurrentMatchweek
	seasonBefore := tmBefore.SeasonName
	seedBefore := tmBefore.Seed

	// Save
	snap := persistence.BuildSnapshot(tmBefore, srv.GrowthEngine, srv.TransferEngine)
	if snap.Seed != seedBefore {
		t.Fatalf("snapshot seed %d != manager seed %d", snap.Seed, seedBefore)
	}

	// Restore onto fresh instance
	srv2, ts2 := setupTestServer(t)
	defer srv2.Stop()
	defer ts2.Close()

	if err := persistence.RestoreCareer(srv2.TournamentManager, srv2.GrowthEngine, srv2.TransferEngine, snap); err != nil {
		t.Fatalf("RestoreCareer failed: %v", err)
	}

	if srv2.TournamentManager.CurrentMatchweek != mwBefore {
		t.Fatalf("restored matchweek = %d, want %d", srv2.TournamentManager.CurrentMatchweek, mwBefore)
	}
	if srv2.TournamentManager.SeasonName != seasonBefore {
		t.Fatalf("restored season = %s, want %s", srv2.TournamentManager.SeasonName, seasonBefore)
	}
	if srv2.TournamentManager.Seed != seedBefore {
		t.Fatalf("restored seed = %d, want %d", srv2.TournamentManager.Seed, seedBefore)
	}
	if err := srv2.TournamentManager.ValidateWorldState(); err != nil {
		t.Fatalf("restored world failed validation: %v", err)
	}
}

func TestHardeningDeterministicSeedSimulation(t *testing.T) {
	// Create universe 1
	elite1 := loadEliteClubsForTest(t)
	tm1 := tournament.NewTournamentManager(elite1, nil, 424242)
	res1 := tm1.SimulateBatchWeeks(5)

	// Create universe 2 with same seed
	elite2 := loadEliteClubsForTest(t)
	tm2 := tournament.NewTournamentManager(elite2, nil, 424242)
	res2 := tm2.SimulateBatchWeeks(5)

	if res1.CurrentMatchweek != res2.CurrentMatchweek {
		t.Fatalf("matchweek mismatch: %d vs %d", res1.CurrentMatchweek, res2.CurrentMatchweek)
	}
	if len(res1.Digests) != len(res2.Digests) {
		t.Fatalf("digests length mismatch: %d vs %d", len(res1.Digests), len(res2.Digests))
	}
	for i := range res1.Digests {
		if len(res1.Digests[i].Results) != len(res2.Digests[i].Results) {
			t.Fatalf("results count mismatch in week %d", i+1)
		}
		for j := range res1.Digests[i].Results {
			r1, r2 := res1.Digests[i].Results[j], res2.Digests[i].Results[j]
			if r1.HomeGoals != r2.HomeGoals || r1.AwayGoals != r2.AwayGoals {
				t.Fatalf("deterministic result mismatch week %d fixture %s vs %s: %d-%d vs %d-%d",
					i+1, r1.FixtureID, r2.FixtureID, r1.HomeGoals, r1.AwayGoals, r2.HomeGoals, r2.AwayGoals)
			}
		}
	}
}

func TestHardeningConcurrentMacroRequests(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer srv.Stop()
	defer ts.Close()

	var wg sync.WaitGroup
	workers := 5
	errChan := make(chan error, workers)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Post(ts.URL+"/api/sim/week", "application/json", nil)
			if err != nil {
				errChan <- err
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusConflict {
				t.Errorf("unexpected status code: %d", resp.StatusCode)
			}
		}()
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			t.Fatalf("concurrent macro request error: %v", err)
		}
	}

	if err := srv.TournamentManager.ValidateWorldState(); err != nil {
		t.Fatalf("world state corrupted after concurrent macro requests: %v", err)
	}
}

func TestHardeningSoakTenSeasons(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping soak test in short mode")
	}

	elite := loadEliteClubsForTest(t)
	tm := tournament.NewTournamentManager(elite, nil, 99999)

	for season := 1; season <= 10; season++ {
		if err := tm.ValidateWorldState(); err != nil {
			t.Fatalf("soak test season %d initial validation failed: %v", season, err)
		}

		// Simulate entire league season
		batch := tm.SimulateBatchWeeks(44)
		if batch.Status != "success" {
			t.Fatalf("soak test season %d batch sim failed: %+v", season, batch)
		}

		if err := tm.ValidateWorldState(); err != nil {
			t.Fatalf("soak test season %d post-league validation failed: %v", season, err)
		}

		// Reset for new season
		resetRes := tm.ResetNewSeason()
		if resetRes["status"] != "success" {
			t.Fatalf("soak test season %d ResetNewSeason failed: %+v", season, resetRes)
		}
	}

	if err := tm.ValidateWorldState(); err != nil {
		t.Fatalf("soak test final validation failed: %v", err)
	}
}

func loadEliteClubsForTest(t *testing.T) []*models.Club {
	t.Helper()
	srv, ts := setupTestServer(t)
	defer srv.Stop()
	defer ts.Close()
	clubs := make([]*models.Club, len(srv.TournamentManager.ClubsList))
	for i, c := range srv.TournamentManager.ClubsList {
		// Deep clone basic properties
		cloned := *c
		cloned.Squad = nil
		for _, p := range c.Squad {
			pClone := *p
			cloned.Squad = append(cloned.Squad, &pClone)
		}
		clubs[i] = &cloned
	}
	return clubs
}
