package matchengine

import (
	"encoding/json"
	"math/rand"
	"testing"

	"football_sim/pkg/matchreport"
	"football_sim/pkg/models"
)

func assertEventIdentityJSON(t *testing.T, eventType, side string, eventPlayer *models.Player, event interface{ }) {
	t.Helper()

	encoded, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal %s event: %v", eventType, err)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatalf("decode %s event: %v", eventType, err)
	}
	if payload["player_id"] != eventPlayer.PlayerID || payload["player_name"] != eventPlayer.FullName {
		t.Fatalf("%s identity payload=%v, want %s/%s", eventType, payload, eventPlayer.PlayerID, eventPlayer.FullName)
	}
	if payload["club_id"] != eventPlayer.ClubID {
		t.Fatalf("%s club id payload=%v, want %s", eventType, payload, eventPlayer.ClubID)
	}
	if payload["club_name"] == "" {
		t.Fatalf("%s omitted club name: %v", eventType, payload)
	}
	if payload["side"] != side {
		t.Fatalf("%s side payload=%v, want %s", eventType, payload, side)
	}
}

func TestMaybeBookPlayerSideAttribution(t *testing.T) {
	homeClub := &models.Club{ClubID: "BAR", ClubName: "Barcelona"}
	awayClub := &models.Club{ClubID: "RMA", ClubName: "Real Madrid"}

	homeStarters := []*models.Player{
		{PlayerID: "H1", FullName: "Home Defender", Category: "DEF", ClubID: "BAR", OVR: 80},
	}
	awayStarters := []*models.Player{
		{PlayerID: "A1", FullName: "Away Defender", Category: "DEF", ClubID: "RMA", OVR: 82},
	}

	e := &LiveMatchEngine{
		HomeClub:       homeClub,
		AwayClub:       awayClub,
		HomeStarters:   homeStarters,
		AwayStarters:   awayStarters,
		Bookings:       make(map[string]int),
		PossessionTeam: "away", // Turnover already happened: home took shot, now possession is away
		CurrentMinute:  35,
		RNG:            rand.New(rand.NewSource(1)), // deterministic seed
	}

	// Book away defenders with explicit defendingSide "away"
	// Force roll < 0.085 by ensuring bookable
	booked := false
	for i := 0; i < 50; i++ {
		e.Bookings = make(map[string]int)
		e.Events = nil
		e.MaybeBookPlayer(awayStarters, "away")
		if len(e.Events) > 0 {
			booked = true
			ev := e.Events[0]
			if ev.Side != "away" {
				t.Fatalf("expected card side 'away', got %q", ev.Side)
			}
			if ev.PlayerID != "A1" {
				t.Fatalf("expected card PlayerID 'A1', got %q", ev.PlayerID)
			}
			if ev.PlayerName != "Away Defender" {
				t.Fatalf("expected card PlayerName 'Away Defender', got %q", ev.PlayerName)
			}
			if ev.ClubID != "RMA" {
				t.Fatalf("expected card ClubID 'RMA', got %q", ev.ClubID)
			}
			assertEventIdentityJSON(t, "yellow", "away", awayStarters[0], ev)
			break
		}
	}
	if !booked {
		t.Fatal("expected at least one booking in 50 attempts")
	}

	// Test second yellow dismissal
	e.Bookings["A1"] = 1
	dismissed := false
	for i := 0; i < 50; i++ {
		e.Events = nil
		e.MaybeBookPlayer(awayStarters, "away")
		if len(e.Events) > 0 {
			dismissed = true
			ev := e.Events[0]
			if ev.Type != "red" {
				t.Fatalf("second yellow should produce red card, got %q", ev.Type)
			}
			if !ev.SentOff {
				t.Fatalf("second yellow should have SentOff=true")
			}
			if ev.Detail != "second_yellow" {
				t.Fatalf("expected detail 'second_yellow', got %q", ev.Detail)
			}
			assertEventIdentityJSON(t, "red", "away", awayStarters[0], ev)
			break
		}
	}
	if !dismissed {
		t.Fatal("expected dismissal on second yellow")
	}
}

func TestPlayerLedEventsShareSerializedIdentityContract(t *testing.T) {
	homeClub := &models.Club{ClubID: "HOME", ClubName: "Home FC"}
	awayClub := &models.Club{ClubID: "AWAY", ClubName: "Away FC"}
	homePlayer := &models.Player{PlayerID: "H1", FullName: "Home Player", ClubID: "HOME"}
	awayPlayer := &models.Player{PlayerID: "A1", FullName: "Away Player", ClubID: "AWAY"}
	engine := &LiveMatchEngine{HomeClub: homeClub, AwayClub: awayClub}

	for _, tc := range []struct {
		typeName string
		side     string
		player   *models.Player
	}{
		{typeName: "goal", side: "home", player: homePlayer},
		{typeName: "penalty", side: "away", player: awayPlayer},
		{typeName: "own_goal", side: "away", player: awayPlayer},
		{typeName: "sub", side: "home", player: homePlayer},
	} {
		event := engine.eventForPlayer(matchreport.MatchEventItem{Type: tc.typeName, Side: tc.side}, tc.player)
		assertEventIdentityJSON(t, tc.typeName, tc.side, tc.player, event)
	}
}
