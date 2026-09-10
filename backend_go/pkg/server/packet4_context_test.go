package server

import (
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestPacket4ServerFixtureContextDoesNotLeakToExhibition(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer srv.Stop()
	defer ts.Close()

	srv.worldMu.Lock()
	tm := srv.TournamentManager
	var derbyHome, derbyAway string
	for _, f := range tm.Fixtures {
		if f.Status == "scheduled" && f.DerbyName != "" && !f.IsHighHeatDerby && f.DerbyHeat <= 70 {
			tm.CurrentMatchweek = f.Matchweek
			derbyHome, derbyAway = f.HomeID, f.AwayID
			break
		}
	}
	if derbyHome == "" {
		srv.worldMu.Unlock()
		t.Fatal("expected a scheduled recognized derby")
	}
	currentSlate := tm.GetSlate(tm.CurrentMatchweek)
	exhibitionHome, exhibitionAway := "", ""
	for _, home := range tm.ClubsList {
		for _, away := range tm.ClubsList {
			if home == away {
				continue
			}
			found := false
			for _, f := range currentSlate {
				if f.HomeID == home.ClubID && f.AwayID == away.ClubID {
					found = true
					break
				}
			}
			if !found {
				exhibitionHome, exhibitionAway = home.ClubID, away.ClubID
				break
			}
		}
		if exhibitionHome != "" {
			break
		}
	}
	srv.worldMu.Unlock()
	if exhibitionHome == "" {
		t.Fatal("expected an exhibition pair absent from the selected slate")
	}

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/match"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect to live websocket: %v", err)
	}
	defer conn.Close()

	waitFor := func(check func() bool) {
		t.Helper()
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			srv.worldMu.RLock()
			ok := check()
			srv.worldMu.RUnlock()
			if ok {
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
		t.Fatal("websocket fixture command was not applied")
	}

	if err := conn.WriteJSON(map[string]interface{}{
		"action": "set_clubs", "home_id": derbyHome, "away_id": derbyAway,
	}); err != nil {
		t.Fatalf("failed to select derby fixture: %v", err)
	}
	waitFor(func() bool {
		return srv.LiveMatchEngine.IsRecognizedDerby && !srv.LiveMatchEngine.IsHighHeatDerby && srv.LiveMatchEngine.Competition != ""
	})

	// Promote the same fixture to the existing high-heat threshold and ensure
	// the card-climate flag is carried independently of recognized rivalry.
	srv.worldMu.Lock()
	for i := range tm.Fixtures {
		if tm.Fixtures[i].HomeID == derbyHome && tm.Fixtures[i].AwayID == derbyAway && tm.Fixtures[i].Status == "scheduled" {
			tm.Fixtures[i].DerbyHeat = 80
			tm.Fixtures[i].IsHighHeatDerby = true
		}
	}
	srv.worldMu.Unlock()
	if err := conn.WriteJSON(map[string]interface{}{
		"action": "set_clubs", "home_id": derbyHome, "away_id": derbyAway,
	}); err != nil {
		t.Fatalf("failed to reselect high-heat derby: %v", err)
	}
	waitFor(func() bool {
		return srv.LiveMatchEngine.IsRecognizedDerby && srv.LiveMatchEngine.IsHighHeatDerby
	})

	if err := conn.WriteJSON(map[string]interface{}{
		"action": "set_clubs", "home_id": exhibitionHome, "away_id": exhibitionAway,
	}); err != nil {
		t.Fatalf("failed to select exhibition: %v", err)
	}
	waitFor(func() bool {
		return !srv.LiveMatchEngine.IsRecognizedDerby && !srv.LiveMatchEngine.IsHighHeatDerby && srv.LiveMatchEngine.Competition == ""
	})

	// A full-time rematch arrives through kickoff alone. The prior fixture must
	// not be reattached or retain its big-game context.
	if err := conn.WriteJSON(map[string]interface{}{
		"action": "set_clubs", "home_id": derbyHome, "away_id": derbyAway,
	}); err != nil {
		t.Fatalf("failed to reselect derby before rematch: %v", err)
	}
	waitFor(func() bool {
		return srv.LiveMatchEngine.IsRecognizedDerby && srv.LiveMatchEngine.IsHighHeatDerby
	})
	srv.wsMu.Lock()
	delete(srv.wsClients, conn)
	srv.wsMu.Unlock()
	srv.worldMu.Lock()
	srv.LiveMatchEngine.State = "FULL_TIME"
	srv.worldMu.Unlock()
	if err := conn.WriteJSON(map[string]interface{}{"action": "kickoff"}); err != nil {
		t.Fatalf("failed to send full-time rematch kickoff: %v", err)
	}
	waitFor(func() bool {
		return srv.LiveMatchEngine.State == "PLAYING" &&
			!srv.LiveMatchEngine.IsRecognizedDerby &&
			!srv.LiveMatchEngine.IsHighHeatDerby &&
			srv.LiveMatchEngine.Competition == "" && srv.liveFixtureID == ""
	})
}

func TestPacket4ServerUCLSelectionCarriesCompetitionContext(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer srv.Stop()
	defer ts.Close()

	srv.worldMu.Lock()
	tm := srv.TournamentManager
	var homeID, awayID string
	for _, f := range tm.UCLFixtures {
		if f.Status == "scheduled" {
			tm.CurrentMatchweek = f.Matchweek
			homeID, awayID = f.HomeID, f.AwayID
			break
		}
	}
	srv.worldMu.Unlock()
	if homeID == "" {
		t.Fatal("expected a scheduled UCL fixture")
	}

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/match"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect to live websocket: %v", err)
	}
	defer conn.Close()
	if err := conn.WriteJSON(map[string]interface{}{
		"action": "set_clubs", "home_id": homeID, "away_id": awayID,
	}); err != nil {
		t.Fatalf("failed to select UCL fixture: %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		srv.worldMu.RLock()
		applied := srv.LiveMatchEngine.Competition == "ucl" && srv.liveFixtureID != ""
		srv.worldMu.RUnlock()
		if applied {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("UCL fixture context was not applied through websocket selection")
}
