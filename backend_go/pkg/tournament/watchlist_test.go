package tournament

import (
	"strings"
	"testing"

	"football_sim/pkg/matchreport"
	"football_sim/pkg/models"
)

func watchTestTM() *TournamentManager {
	home := &models.Club{ClubID: "WH", ClubName: "Watched FC", ShortName: "WFC"}
	away := &models.Club{ClubID: "WA", ClubName: "Other FC", ShortName: "OFC"}
	scorer := &models.Player{PlayerID: "P1", FullName: "Watched Player", Position: "ST", Category: "FWD"}
	home.Squad = []*models.Player{scorer}
	return &TournamentManager{
		Clubs:     map[string]*models.Club{"WH": home, "WA": away},
		ClubsList: []*models.Club{home, away},
		Watch:     WatchlistState{},
		Inbox:     nil,
	}
}

func TestToggleWatchlistAddRemoveAndValidation(t *testing.T) {
	tm := watchTestTM()

	// Unknown entities are rejected.
	if _, _, ok := tm.ToggleWatchlist(WatchClub, "NOPE"); ok {
		t.Fatal("unknown club must be rejected")
	}
	if _, _, ok := tm.ToggleWatchlist(WatchPlayer, "NOPE"); ok {
		t.Fatal("unknown player must be rejected")
	}
	if _, _, ok := tm.ToggleWatchlist(WatchCompetition, "NOPE"); ok {
		t.Fatal("unknown competition must be rejected")
	}
	if _, _, ok := tm.ToggleWatchlist("bogus", "WH"); ok {
		t.Fatal("unknown entity kind must be rejected")
	}

	// Add a club and a player.
	state, watched, ok := tm.ToggleWatchlist(WatchClub, "WH")
	if !ok || !watched || len(state.Clubs) != 1 || state.Clubs[0] != "WH" {
		t.Fatalf("club add wrong: %+v watched=%v ok=%v", state, watched, ok)
	}
	if !tm.IsWatched(WatchClub, "WH") {
		t.Fatal("club should be watched")
	}
	if _, watched, _ := tm.ToggleWatchlist(WatchPlayer, "P1"); !watched {
		t.Fatal("player add failed")
	}

	// Toggle again removes.
	if _, watched, _ := tm.ToggleWatchlist(WatchClub, "WH"); watched {
		t.Fatal("second toggle should remove")
	}
	if tm.IsWatched(WatchClub, "WH") {
		t.Fatal("club should no longer be watched")
	}
	// The player is still watched.
	if !tm.IsWatched(WatchPlayer, "P1") {
		t.Fatal("player should remain watched")
	}
}

func TestWatchlistDigestReportsWatchedActivity(t *testing.T) {
	tm := watchTestTM()
	tm.ToggleWatchlist(WatchClub, "WH")
	tm.ToggleWatchlist(WatchPlayer, "P1")

	hg, ag := 2, 1
	tm.Fixtures = []Fixture{{
		FixtureID: "WX1", Matchweek: 5, Competition: "premier-league",
		HomeID: "WH", AwayID: "WA", Status: "finished", HomeGoals: &hg, AwayGoals: &ag,
		Report: &matchreport.MatchReport{
			HomeGoals: 2, AwayGoals: 1,
			Events: []matchreport.MatchEventItem{
				{Minute: 10, Type: "goal", Side: "home", Scorer: &matchreport.MiniPlayer{PlayerID: "P1", FullName: "Watched Player"}},
				{Minute: 30, Type: "goal", Side: "home", Disallowed: true, Scorer: &matchreport.MiniPlayer{PlayerID: "P1", FullName: "Watched Player"}},
			},
		},
	}}

	tm.generateWatchlistDigestUnlocked(5)
	found := false
	for _, item := range tm.Inbox {
		if item.Category != "watch" {
			continue
		}
		found = true
		if !strings.Contains(item.Body, "WFC 2–1 OFC") {
			t.Fatalf("digest missing watched club result: %q", item.Body)
		}
		if !strings.Contains(item.Body, "Watched Player scored for WFC in the 10'") {
			t.Fatalf("digest missing watched player goal: %q", item.Body)
		}
		if strings.Contains(item.Body, "30'") {
			t.Fatalf("disallowed goals must not be reported: %q", item.Body)
		}
	}
	if !found {
		t.Fatal("no watch digest was pushed")
	}

	// Empty watchlist produces nothing; other matchweeks produce nothing.
	tm2 := watchTestTM()
	tm2.generateWatchlistDigestUnlocked(5)
	if len(tm2.Inbox) != 0 {
		t.Fatal("empty watchlist must not produce a digest")
	}
	tm.generateWatchlistDigestUnlocked(6)
	before := len(tm.Inbox)
	tm.generateWatchlistDigestUnlocked(6)
	if len(tm.Inbox) != before {
		t.Fatal("a matchweek with no watched activity must not push a digest")
	}
}
