package server

import (
	"strings"
	"testing"
	"time"

	"football_sim/pkg/tournament"

	"github.com/gorilla/websocket"
)

func TestPacket5ScheduledRainSelectionSetsLiveWeatherBeforeKickoff(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer srv.Stop()
	defer ts.Close()

	srv.worldMu.Lock()
	var homeID, awayID string
	for i := range srv.TournamentManager.Fixtures {
		fixture := &srv.TournamentManager.Fixtures[i]
		if fixture.Status != "scheduled" {
			continue
		}
		fixture.Weather = "rain"
		srv.TournamentManager.CurrentMatchweek = fixture.Matchweek
		homeID, awayID = fixture.HomeID, fixture.AwayID
		break
	}
	srv.worldMu.Unlock()
	if homeID == "" || awayID == "" {
		t.Fatal("expected a scheduled fixture to select")
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
		t.Fatalf("failed to select scheduled rain fixture: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		srv.worldMu.RLock()
		weather := srv.LiveMatchEngine.Weather
		selected := srv.liveFixtureID != ""
		state := srv.LiveMatchEngine.State
		srv.worldMu.RUnlock()
		if selected && weather == "rain" && state == "NOT_STARTED" {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("scheduled rain selection did not set live weather before kickoff")
}

func TestPacket8EmptyScheduledWeatherFilledFromMatchweekOnLiveSelect(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer srv.Stop()
	defer ts.Close()

	srv.worldMu.Lock()
	var homeID, awayID, fixtureID string
	var matchweek int
	for i := range srv.TournamentManager.Fixtures {
		fixture := &srv.TournamentManager.Fixtures[i]
		if fixture.Status != "scheduled" {
			continue
		}
		fixture.Weather = ""
		matchweek = fixture.Matchweek
		fixtureID = fixture.FixtureID
		srv.TournamentManager.CurrentMatchweek = fixture.Matchweek
		homeID, awayID = fixture.HomeID, fixture.AwayID
		break
	}
	if srv.TournamentManager.MatchweekWeather == nil {
		srv.TournamentManager.MatchweekWeather = map[int]string{}
	}
	srv.TournamentManager.MatchweekWeather[matchweek] = "rain"
	srv.worldMu.Unlock()
	if homeID == "" || awayID == "" {
		t.Fatal("expected a scheduled fixture to select")
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
		t.Fatalf("failed to select blank-weather fixture: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		srv.worldMu.RLock()
		weather := srv.LiveMatchEngine.Weather
		selected := srv.liveFixtureID == fixtureID
		stored := ""
		if f := srv.TournamentManager.FindFixture(fixtureID); f != nil {
			stored = f.Weather
		}
		srv.worldMu.RUnlock()
		if selected && weather == "rain" && stored == "rain" {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("blank scheduled weather was not filled from the matchweek table before kickoff")
}

func TestPacket8SerializeFixtureFillsWeatherFromMatchweek(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer srv.Stop()
	defer ts.Close()

	srv.worldMu.Lock()
	var fixture *tournament.Fixture
	for i := range srv.TournamentManager.Fixtures {
		candidate := &srv.TournamentManager.Fixtures[i]
		if candidate.Status != "scheduled" {
			continue
		}
		fixture = candidate
		break
	}
	if fixture == nil {
		srv.worldMu.Unlock()
		t.Fatal("expected a scheduled fixture")
	}
	fixture.Weather = ""
	if srv.TournamentManager.MatchweekWeather == nil {
		srv.TournamentManager.MatchweekWeather = map[int]string{}
	}
	srv.TournamentManager.MatchweekWeather[fixture.Matchweek] = "wind"
	payload := srv.serializeFixture(fixture)
	blank := fixture.Weather
	srv.worldMu.Unlock()

	weather, _ := payload["weather"].(string)
	if weather != "wind" {
		t.Fatalf("serialized weather = %q; want wind", weather)
	}
	if blank != "" {
		t.Fatalf("GET serialize mutated stored weather: %q", blank)
	}
}

func TestPacket8LiveTickIncludesWeather(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer srv.Stop()
	defer ts.Close()

	srv.worldMu.Lock()
	srv.LiveMatchEngine.SetFixtureWeather("rain")
	payload := srv.buildMatchTickPayload()
	srv.worldMu.Unlock()

	weather, _ := payload["weather"].(string)
	if weather != "rain" {
		t.Fatalf("live tick weather = %q; want rain", weather)
	}
}
