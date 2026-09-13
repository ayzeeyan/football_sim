package tournament

import (
	"testing"

	"football_sim/pkg/matchreport"
	"football_sim/pkg/models"
)

func TestWorldFixturesFeedHeadToHeadAndPlayerHistory(t *testing.T) {
	a := &models.Club{ClubID: "A", ClubName: "Alpha", ShortName: "ALP"}
	b := &models.Club{ClubID: "B", ClubName: "Beta", ShortName: "BET"}
	hg, ag := 2, 1
	rating := 8.1
	worldFixture := Fixture{
		FixtureID: "WORLD-1", Matchweek: 7, Competition: "champions-league", Stage: "League phase",
		HomeID: "A", AwayID: "B", Status: "finished", HomeGoals: &hg, AwayGoals: &ag,
		Report: &matchreport.MatchReport{HomeXI: []matchreport.MatchPlayerRow{{PlayerID: "P", FullName: "Player", Played: true, Minutes: 90, Rating: &rating}}},
	}
	tm := &TournamentManager{
		Clubs: map[string]*models.Club{"A": a, "B": b}, ClubsList: []*models.Club{a, b},
		World: &EuropeanWorld{Fixtures: []Fixture{worldFixture}},
	}

	h2h := tm.GetHeadToHead("A", "B")
	if h2h == nil || h2h["matches_played"] != 1 || h2h["goals_a"] != 2 {
		t.Fatalf("world head-to-head omitted finished fixture: %#v", h2h)
	}
	current := &Fixture{FixtureID: "NEXT", HomeID: "B", AwayID: "A"}
	if got := tm.FixtureHeadToHead(current); len(got) != 1 || got[0]["id"] != "WORLD-1" {
		t.Fatalf("fixture history omitted world fixture: %#v", got)
	}
	if got := tm.PlayerMatchLog("P", 5); len(got) != 1 || got[0]["fixture_id"] != "WORLD-1" {
		t.Fatalf("player history omitted world fixture: %#v", got)
	}
}
