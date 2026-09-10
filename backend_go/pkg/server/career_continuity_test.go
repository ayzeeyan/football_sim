package server

import (
	"net/http"
	"testing"

	"football_sim/pkg/datamanager"
	"football_sim/pkg/persistence"
)

func TestRestoreCareerKeepsFinishedFixturesAndHomes(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	srv.worldMu.Lock()
	homes := datamanager.ShuffleProdigyHomes(nil)
	srv.TournamentManager.ProdigyHomes = homes
	res := srv.TournamentManager.SimulateMatchweek(1)
	if res["status"] != "success" {
		srv.worldMu.Unlock()
		t.Fatalf("simulate mw1: %v", res)
	}
	weekAfter := srv.TournamentManager.CurrentMatchweek
	var finishedID string
	var finishedHome, finishedAway int
	for _, f := range srv.TournamentManager.Fixtures {
		if f.Status == "finished" && f.HomeGoals != nil {
			finishedID = f.FixtureID
			finishedHome, finishedAway = *f.HomeGoals, *f.AwayGoals
			break
		}
	}
	if finishedID == "" {
		srv.worldMu.Unlock()
		t.Fatal("expected a finished league fixture after MW1")
	}
	snap := persistence.BuildSnapshot(srv.TournamentManager, srv.GrowthEngine, srv.TransferEngine)
	srv.worldMu.Unlock()

	fresh, _ := setupTestServer(t)
	defer fresh.Stop()
	if err := persistence.RestoreCareer(fresh.TournamentManager, fresh.GrowthEngine, fresh.TransferEngine, snap); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if fresh.TournamentManager.CurrentMatchweek != weekAfter {
		t.Fatalf("restored matchweek=%d want %d", fresh.TournamentManager.CurrentMatchweek, weekAfter)
	}
	got := fresh.TournamentManager.FindFixture(finishedID)
	if got == nil || got.Status != "finished" || got.HomeGoals == nil || *got.HomeGoals != finishedHome || *got.AwayGoals != finishedAway {
		t.Fatalf("finished fixture did not survive restore: %+v", got)
	}
	for name, cid := range homes {
		if fresh.TournamentManager.ProdigyHomes[name] != cid {
			t.Fatalf("homes for %s = %s want %s", name, fresh.TournamentManager.ProdigyHomes[name], cid)
		}
	}
}

func TestSeasonResetAgesWorldAndRestartDoesNot(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	srv.worldMu.Lock()
	prodigyID := datamanager.ProdigyStableID("Venjamin Valerio")
	var prodigyClub string
	var beforeAge, beforeCareer int
	for _, club := range srv.TournamentManager.ClubsList {
		for _, p := range club.Squad {
			if p.PlayerID == prodigyID {
				p.Goals = 3
				p.Appearances = 5
				p.CareerGoals = 10
				beforeAge = p.Age
				beforeCareer = p.CareerGoals
				prodigyClub = club.ClubID
			}
		}
		club.Played = 8
		club.Points = 15
	}
	srv.worldMu.Unlock()

	resp, err := http.Post(ts.URL+"/api/season/restart", "application/json", nil)
	if err != nil {
		t.Fatalf("restart: %v", err)
	}
	var restart map[string]interface{}
	decodeJSONBody(t, resp, &restart)
	if resp.StatusCode != http.StatusOK || restart["status"] != "success" {
		t.Fatalf("restart payload: %v", restart)
	}
	if restart["season_name"] != "2026-27" {
		t.Fatalf("restart should keep the campaign year: %v", restart["season_name"])
	}

	srv.worldMu.RLock()
	if srv.TournamentManager.CurrentMatchweek != 1 || srv.TournamentManager.SeasonPhase != "season" {
		srv.worldMu.RUnlock()
		t.Fatalf("restart calendar mw=%d phase=%s", srv.TournamentManager.CurrentMatchweek, srv.TournamentManager.SeasonPhase)
	}
	for _, club := range srv.TournamentManager.ClubsList {
		if club.Played != 0 || club.Points != 0 {
			srv.worldMu.RUnlock()
			t.Fatalf("%s table survived restart", club.ClubName)
		}
	}
	club := srv.TournamentManager.Clubs[prodigyClub]
	found := false
	for _, p := range club.Squad {
		if p.PlayerID == prodigyID {
			found = true
			if p.Age != beforeAge {
				srv.worldMu.RUnlock()
				t.Fatalf("restart aged the prodigy: %d from %d", p.Age, beforeAge)
			}
			if p.CareerGoals != beforeCareer {
				srv.worldMu.RUnlock()
				t.Fatalf("restart wiped career goals: %d from %d", p.CareerGoals, beforeCareer)
			}
			if p.Goals != 0 || p.Appearances != 0 {
				srv.worldMu.RUnlock()
				t.Fatalf("restart left season stats goals=%d apps=%d", p.Goals, p.Appearances)
			}
		}
	}
	srv.worldMu.RUnlock()
	if !found {
		t.Fatal("prodigy missing after restart")
	}

	srv.worldMu.Lock()
	for _, club := range srv.TournamentManager.ClubsList {
		for _, p := range club.Squad {
			if p.PlayerID == prodigyID {
				p.Goals = 2
				p.Appearances = 4
			}
		}
	}
	srv.worldMu.Unlock()

	resp, err = http.Post(ts.URL+"/api/season/reset", "application/json", nil)
	if err != nil {
		t.Fatalf("reset: %v", err)
	}
	var reset map[string]interface{}
	decodeJSONBody(t, resp, &reset)
	if resp.StatusCode != http.StatusOK || reset["status"] != "success" {
		t.Fatalf("reset payload: %v", reset)
	}

	srv.worldMu.RLock()
	defer srv.worldMu.RUnlock()
	if srv.TournamentManager.SeasonName == "2026-27" {
		t.Fatal("reset should roll the season year")
	}
	if srv.TournamentManager.SeasonPhase != "season" || srv.TournamentManager.CurrentMatchweek != 1 {
		t.Fatalf("reset calendar mw=%d phase=%s", srv.TournamentManager.CurrentMatchweek, srv.TournamentManager.SeasonPhase)
	}
	if srv.TransferEngine.CurrentDay != 1 {
		t.Fatalf("new season should reset the market day, got %d", srv.TransferEngine.CurrentDay)
	}
	for _, club := range srv.TournamentManager.ClubsList {
		for _, p := range club.Squad {
			if p.PlayerID == prodigyID {
				if p.Age != beforeAge+1 {
					t.Fatalf("reset should age the world: age=%d want %d", p.Age, beforeAge+1)
				}
				if p.CareerGoals != beforeCareer+2 {
					t.Fatalf("reset should bank season goals: career=%d want %d", p.CareerGoals, beforeCareer+2)
				}
				if p.Goals != 0 {
					t.Fatalf("reset left this year's goals at %d", p.Goals)
				}
				bio := srv.GrowthEngine.Biometrics[p.PlayerID]
				if bio == nil || bio.Age != p.Age {
					t.Fatalf("growth age not rolled with the player: %+v", bio)
				}
			}
		}
	}

	saved, err := persistence.LoadCareer(srv.savePath)
	if err != nil {
		t.Fatalf("reset persist: %v", err)
	}
	if saved.SeasonName != srv.TournamentManager.SeasonName {
		t.Fatalf("saved season=%s live=%s", saved.SeasonName, srv.TournamentManager.SeasonName)
	}
}

func TestAdoptLongSeasonKeepsPlayedResults(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	srv.worldMu.Lock()
	short := srv.TournamentManager.Fixtures[:11*6]
	for i := range short {
		if short[i].Matchweek == 1 {
			g := 1
			short[i].Status = "finished"
			short[i].HomeGoals = &g
			short[i].AwayGoals = new(int)
		}
	}
	srv.TournamentManager.Fixtures = short
	srv.TournamentManager.MaxMatchweeks = 11
	srv.TournamentManager.SuperCupFixtures = nil
	changed := srv.TournamentManager.AdoptLongSeason()
	srv.worldMu.Unlock()
	if !changed {
		t.Fatal("expected a short calendar to stretch to 44 weeks")
	}
	if srv.TournamentManager.MaxMatchweeks != 44 {
		t.Fatalf("max weeks=%d", srv.TournamentManager.MaxMatchweeks)
	}
	finished := 0
	for _, f := range srv.TournamentManager.Fixtures {
		if f.Matchweek == 1 && f.Status == "finished" {
			finished++
		}
	}
	if finished == 0 {
		t.Fatal("adopt long season dropped already-played results")
	}
	if len(srv.TournamentManager.SuperCupFixtures) == 0 {
		t.Fatal("adopt long season should draw Super Cup")
	}
}
