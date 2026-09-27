package tournament

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"football_sim/pkg/models"
)

func TestNationalTeamsCupBuildsStableRostersAndSeparateSchedule(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	competition := tm.World.NationalTeams
	if competition == nil {
		t.Fatal("fresh world has no national teams competition")
	}
	if competition.ID != nationalTeamsCompetitionID || competition.Stage != "Scheduled" {
		t.Fatalf("competition identity/stage=%q/%q", competition.ID, competition.Stage)
	}
	if len(competition.TeamOrder) != 5 || len(competition.Fixtures) != 10 {
		t.Fatalf("national sides/fixtures=%d/%d, want 5/10", len(competition.TeamOrder), len(competition.Fixtures))
	}
	for _, teamID := range competition.TeamOrder {
		team := competition.Teams[teamID]
		if team == nil || len(team.PlayerIDs) != 23 || team.Rating == 0 {
			t.Fatalf("incomplete national roster %q: %#v", teamID, team)
		}
		for _, playerID := range team.PlayerIDs {
			player, _ := tm.nationalPlayerUnlocked(playerID)
			if player == nil {
				t.Fatalf("national roster %s references missing player %s", teamID, playerID)
			}
			originID := player.OriginalClubID
			if originID == "" {
				originID = player.ClubID
			}
			if origin := tm.Clubs[originID]; origin == nil || origin.Country != team.Country {
				t.Fatalf("player %s assigned to %s from origin %q", playerID, team.Country, originID)
			}
		}
	}
	for _, fixture := range competition.Fixtures {
		if fixture.Matchweek < 1 || fixture.Matchweek > tm.MaxMatchweeks || fixture.Status != "scheduled" {
			t.Fatalf("unexpected international fixture state: %+v", fixture)
		}
	}
	for _, fixture := range tm.Fixtures {
		if fixture.Matchweek > tm.MaxMatchweeks {
			t.Fatalf("national fixture contaminated club calendar: %+v", fixture)
		}
	}
	for _, fixture := range tm.World.Fixtures {
		if fixture.Matchweek > tm.MaxMatchweeks {
			t.Fatalf("national fixture contaminated shared club calendar: %+v", fixture)
		}
	}

	encoded, err := json.Marshal(tm.World)
	if err != nil {
		t.Fatalf("marshal world with national state: %v", err)
	}
	var restored EuropeanWorld
	if err := json.Unmarshal(encoded, &restored); err != nil {
		t.Fatalf("restore world with national state: %v", err)
	}
	if restored.NationalTeams == nil || !reflect.DeepEqual(competition, restored.NationalTeams) {
		t.Fatal("national teams competition did not survive world JSON persistence")
	}
}

func TestEnsureNationalTeamsMigratesOnlyMissingWorldState(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	tm.mu.Lock()
	tm.World.NationalTeams = nil
	tm.mu.Unlock()
	if !tm.EnsureNationalTeams() {
		t.Fatal("missing national competition was not initialized")
	}
	if tm.World.NationalTeams == nil || tm.World.NationalTeams.Season != tm.SeasonName {
		t.Fatalf("migrated national competition missing current season: %+v", tm.World.NationalTeams)
	}
	if tm.EnsureNationalTeams() {
		t.Fatal("existing national competition was replaced")
	}
}

func TestNationalTeamRatingUsesBalancedStartingEleven(t *testing.T) {
	positions := []struct {
		category string
		position string
		values   []int
	}{
		{"GK", "GK", []int{90, 80, 75}},
		{"DEF", "CB", []int{100, 90, 80, 70, 65, 60, 55, 50}},
		{"MID", "CM", []int{100, 90, 80, 70, 65, 60, 55, 50}},
		{"FWD", "ST", []int{100, 90, 80, 70}},
	}
	var ids []string
	var candidates []nationalPlayerCandidate
	for _, row := range positions {
		for i, rating := range row.values {
			id := fmt.Sprintf("%s-%d", row.category, i)
			ids = append(ids, id)
			candidates = append(candidates, nationalPlayerCandidate{player: &models.Player{PlayerID: id, Position: row.position, OVR: rating}})
		}
	}
	if got, want := nationalSquadRating(ids, candidates), 88; got != want {
		t.Fatalf("balanced national XI rating=%d want %d", got, want)
	}
}

func TestNationalTeamsCupCompletesDeterministicallyAndExposesAPI(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	second, _, _ := loadEuropeanWorldForTest(t)
	club := tm.ClubsList[0]
	player := club.Squad[0]
	clubPlayed, playerGoals, playerApps := club.Played, player.Goals, player.Appearances
	tm.mu.Lock()
	for week := 1; week <= tm.MaxMatchweeks; week++ {
		tm.advanceNationalTeamsMatchweekUnlocked(week)
	}
	tm.mu.Unlock()
	second.mu.Lock()
	for week := 1; week <= second.MaxMatchweeks; week++ {
		second.advanceNationalTeamsMatchweekUnlocked(week)
	}
	second.mu.Unlock()
	if !reflect.DeepEqual(tm.World.NationalTeams.Fixtures, second.World.NationalTeams.Fixtures) {
		t.Fatal("national cup results changed between same-seed worlds")
	}
	if club.Played != clubPlayed || player.Goals != playerGoals || player.Appearances != playerApps {
		t.Fatalf("national results changed club state: club=%d/%d player=%d/%d apps=%d/%d", club.Played, clubPlayed, player.Goals, playerGoals, player.Appearances, playerApps)
	}
	competition := tm.World.NationalTeams
	if competition.Stage != "Complete" || competition.ChampionID == "" {
		t.Fatalf("national cup did not complete: stage=%q champion=%q", competition.Stage, competition.ChampionID)
	}
	if len(competition.Fixtures) != 11 {
		t.Fatalf("national cup fixtures after final=%d want 11", len(competition.Fixtures))
	}
	final := competition.Fixtures[len(competition.Fixtures)-1]
	if final.Stage != "Final" || final.Matchweek != tm.MaxMatchweeks || final.Status != "finished" || final.HomeID == final.AwayID {
		t.Fatalf("national cup final was not resolved: %+v", final)
	}
	finalWinner := final.AwayID
	if *final.HomeGoals > *final.AwayGoals || (*final.HomeGoals == *final.AwayGoals && final.HomePens > final.AwayPens) {
		finalWinner = final.HomeID
	}
	if finalWinner != competition.ChampionID {
		t.Fatalf("national champion %s does not match final winner %s", competition.ChampionID, finalWinner)
	}
	for _, teamID := range competition.TeamOrder {
		if got := competition.Teams[teamID].Record.Played; got != 4 {
			t.Fatalf("%s played %d national fixtures, want 4", teamID, got)
		}
	}
	championID := competition.ChampionID
	tm.mu.Lock()
	tm.completeNationalTeamsUnlocked()
	tm.mu.Unlock()
	if competition.ChampionID != championID || !reflect.DeepEqual(competition.Fixtures, second.World.NationalTeams.Fixtures) {
		t.Fatal("repeated completion changed national cup results")
	}

	list := tm.GetCompetitions()
	if len(list) != 15 || list[len(list)-1]["id"] != nationalTeamsCompetitionID {
		t.Fatalf("competition discovery missing Nations Cup: count=%d last=%v", len(list), list[len(list)-1])
	}
	detail := tm.GetCompetition(nationalTeamsCompetitionID)
	if detail == nil || detail["stage"] != "Complete" || detail["champion_id"] != championID {
		t.Fatalf("national cup detail API incomplete: %#v", detail)
	}
	if rows, ok := detail["table"].([]map[string]interface{}); !ok || len(rows) != 5 {
		t.Fatalf("national cup table type/size=%T/%d", detail["table"], len(rows))
	}
	if fixtures, ok := detail["fixtures"].([]map[string]interface{}); !ok || len(fixtures) != 11 {
		t.Fatalf("national cup fixtures type/size=%T/%d", detail["fixtures"], len(fixtures))
	}
	tm.mu.Lock()
	tm.archiveNationalTeamsSeasonUnlocked()
	tm.SeasonName = "2027-28"
	tm.initializeNationalTeamsUnlocked()
	tm.mu.Unlock()
	if got := len(tm.World.NationalTeams.History); got != 1 || tm.World.NationalTeams.History[0].Season != "2026-27" {
		t.Fatalf("national history not archived across seasons: %#v", tm.World.NationalTeams.History)
	}
	if tm.World.NationalTeams.Stage != "Scheduled" || tm.World.NationalTeams.Season != "2027-28" {
		t.Fatalf("next national season not initialized: %+v", tm.World.NationalTeams)
	}
}

func TestNationalFixturesCarryFullReportsWithoutTouchingClubStats(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	competition := tm.World.NationalTeams
	if competition == nil {
		t.Fatal("fresh world has no national teams competition")
	}

	// Snapshot club season statistics for every national squad member before
	// the international matchweek resolves.
	type statLine struct{ goals, assists, apps, fatigue int }
	before := make(map[string]statLine)
	for _, teamID := range competition.TeamOrder {
		for _, playerID := range competition.Teams[teamID].PlayerIDs {
			player, _ := tm.nationalPlayerUnlocked(playerID)
			if player != nil {
				before[playerID] = statLine{player.Goals, player.Assists, player.Appearances, player.ConsecutiveStarts}
			}
		}
	}

	tm.advanceNationalTeamsMatchweekUnlocked(nationalGroupMatchweeks[0])

	played := 0
	for i := range competition.Fixtures {
		fixture := &competition.Fixtures[i]
		if fixture.Status != "finished" {
			continue
		}
		played++
		if fixture.Report == nil {
			t.Fatalf("finished international fixture %s has no match report", fixture.FixtureID)
		}
		report := fixture.Report
		if report.HomeGoals != *fixture.HomeGoals || report.AwayGoals != *fixture.AwayGoals {
			t.Fatalf("report scoreline %d-%d does not match fixture %d-%d",
				report.HomeGoals, report.AwayGoals, *fixture.HomeGoals, *fixture.AwayGoals)
		}
		if len(report.HomeXI) != 11 || len(report.AwayXI) != 11 {
			t.Fatalf("report lineups %d/%d, want 11/11", len(report.HomeXI), len(report.AwayXI))
		}
		if len(report.Events) == 0 {
			t.Fatalf("report for %s has no events", fixture.FixtureID)
		}
	}
	if played == 0 {
		t.Fatal("no international fixtures were played on the first group matchweek")
	}

	// International results must never mutate club season statistics.
	for _, teamID := range competition.TeamOrder {
		for _, playerID := range competition.Teams[teamID].PlayerIDs {
			player, _ := tm.nationalPlayerUnlocked(playerID)
			if player == nil {
				continue
			}
			want := before[playerID]
			if player.Goals != want.goals || player.Assists != want.assists || player.Appearances != want.apps || player.ConsecutiveStarts != want.fatigue {
				t.Fatalf("international match mutated club stats for %s: goals %d->%d assists %d->%d apps %d->%d starts %d->%d",
					playerID, want.goals, player.Goals, want.assists, player.Assists, want.apps, player.Appearances, want.fatigue, player.ConsecutiveStarts)
			}
		}
	}

	// The wire payload must mirror the club fixture shape the report
	// components already consume.
	payload := tm.NationsFixturePayload(competition.Fixtures[0].FixtureID)
	if payload == nil {
		t.Fatal("nations fixture payload missing")
	}
	if payload["has_report"] != true {
		t.Fatalf("payload has_report=%v, want true", payload["has_report"])
	}
	if _, ok := payload["stats"]; !ok {
		t.Fatal("payload is missing the stats block")
	}

	// On-demand simulation resolves a scheduled fixture and is idempotent.
	var scheduledID string
	for i := range competition.Fixtures {
		if competition.Fixtures[i].Status == "scheduled" {
			scheduledID = competition.Fixtures[i].FixtureID
			break
		}
	}
	if scheduledID != "" {
		first := tm.SimulateNationsFixture(scheduledID)
		if first == nil || first["status"] != "finished" {
			t.Fatalf("on-demand simulate did not finish %s: %#v", scheduledID, first)
		}
		again := tm.SimulateNationsFixture(scheduledID)
		if again == nil || again["status"] != "finished" {
			t.Fatalf("re-simulate changed a finished fixture: %#v", again)
		}
		if first["home_goals"] != again["home_goals"] || first["away_goals"] != again["away_goals"] {
			t.Fatal("re-simulating a finished international fixture changed the scoreline")
		}
	}
}
