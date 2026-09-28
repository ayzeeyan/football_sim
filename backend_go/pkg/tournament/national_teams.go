package tournament

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"

	"football_sim/pkg/matchengine"
	"football_sim/pkg/matchreport"
	"football_sim/pkg/models"
)

const nationalTeamsCompetitionID = "nations-cup"

var nationalGroupMatchweeks = [...]int{8, 15, 22, 29, 36}

// NationalTeam is a season snapshot of one national side. Since the player
// dataset does not contain nationality, eligibility is inferred once from the
// country of the player's immutable original club and the selected player IDs
// are persisted with the competition.
type NationalTeam struct {
	ID        string             `json:"id"`
	Name      string             `json:"name"`
	Country   string             `json:"country"`
	PlayerIDs []string           `json:"player_ids"`
	Rating    int                `json:"rating"`
	Record    NationalTeamRecord `json:"record"`
}

// NationalTeamRecord stores standings for the separate national schedule.
type NationalTeamRecord struct {
	Played         int `json:"played"`
	Won            int `json:"won"`
	Drawn          int `json:"drawn"`
	Lost           int `json:"lost"`
	GoalsFor       int `json:"goals_for"`
	GoalsAgainst   int `json:"goals_against"`
	GoalDifference int `json:"goal_difference"`
	Points         int `json:"points"`
}

// NationalTeamFixture is deliberately separate from Fixture: international
// results must never enter club standings, fatigue, player statistics, or
// club fixture lookup.
type NationalTeamFixture struct {
	FixtureID string `json:"fixture_id"`
	Matchweek int    `json:"matchweek"`
	Stage     string `json:"stage"`
	HomeID    string `json:"home_id"`
	AwayID    string `json:"away_id"`
	Status    string `json:"status"`
	HomeGoals *int   `json:"home_goals,omitempty"`
	AwayGoals *int   `json:"away_goals,omitempty"`
	DecidedBy string `json:"decided_by,omitempty"`
	HomePens  int    `json:"home_penalties,omitempty"`
	AwayPens  int    `json:"away_penalties,omitempty"`
	// Report is presentation data only: international results never enter
	// club standings, player season statistics, fatigue, or growth.
	Report *matchreport.MatchReport `json:"report,omitempty"`
}

type NationalTeamTableRow struct {
	TeamID         string `json:"team_id"`
	Name           string `json:"name"`
	Country        string `json:"country"`
	Rating         int    `json:"rating"`
	Played         int    `json:"played"`
	Won            int    `json:"won"`
	Drawn          int    `json:"drawn"`
	Lost           int    `json:"lost"`
	GoalsFor       int    `json:"goals_for"`
	GoalsAgainst   int    `json:"goals_against"`
	GoalDifference int    `json:"goal_difference"`
	Points         int    `json:"points"`
}

type NationalTeamSeason struct {
	Season     string                 `json:"season"`
	ChampionID string                 `json:"champion_id"`
	Table      []NationalTeamTableRow `json:"table"`
	Fixtures   []NationalTeamFixture  `json:"fixtures"`
}

// NationalTeamsCompetition is persisted under EuropeanWorld. Its fixture
// stream has its own post-league matchweeks (39-43), so it shares the career
// season identity and history without mutating the 38-week club calendar.
type NationalTeamsCompetition struct {
	ID              string                   `json:"id"`
	Name            string                   `json:"name"`
	Country         string                   `json:"country"`
	Kind            string                   `json:"kind"`
	Prestige        int                      `json:"prestige"`
	Season          string                   `json:"season"`
	Stage           string                   `json:"stage"`
	EligibilityRule string                   `json:"eligibility_rule"`
	TeamOrder       []string                 `json:"team_order"`
	Teams           map[string]*NationalTeam `json:"teams"`
	Fixtures        []NationalTeamFixture    `json:"fixtures"`
	ChampionID      string                   `json:"champion_id,omitempty"`
	History         []NationalTeamSeason     `json:"history,omitempty"`
}

type nationalPlayerCandidate struct {
	player *models.Player
}

// EnsureNationalTeams upgrades an older EuropeanWorld save that predates the
// international competition. It is safe to call after restoration and does
// not replace an existing competition or its history.
func (tm *TournamentManager) EnsureNationalTeams() bool {
	if tm == nil {
		return false
	}
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if tm.World == nil || tm.World.NationalTeams != nil {
		return false
	}
	tm.initializeNationalTeamsUnlocked()
	return tm.World.NationalTeams != nil
}

func nationalTeamID(country string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(country), " ", "-"))
}

func (tm *TournamentManager) initializeNationalTeamsUnlocked() {
	if tm == nil || tm.World == nil {
		return
	}
	var history []NationalTeamSeason
	if tm.World.NationalTeams != nil {
		history = append(history, tm.World.NationalTeams.History...)
	}
	competition := &NationalTeamsCompetition{
		ID:              nationalTeamsCompetitionID,
		Name:            "European Nations Cup",
		Country:         "Europe",
		Kind:            "INTERNATIONAL",
		Prestige:        82,
		Season:          tm.SeasonName,
		Stage:           "Scheduled",
		EligibilityRule: "Inferred from each player's original club country; player nationality is not present in the source dataset.",
		Teams:           map[string]*NationalTeam{},
		History:         history,
	}

	playersByCountry := make(map[string][]nationalPlayerCandidate)
	seen := make(map[string]bool)
	for _, club := range tm.ClubsList {
		if club == nil {
			continue
		}
		for _, player := range club.Squad {
			if player == nil || player.PlayerID == "" || seen[player.PlayerID] {
				continue
			}
			seen[player.PlayerID] = true
			originID := player.OriginalClubID
			if originID == "" {
				originID = player.ClubID
			}
			origin := tm.Clubs[originID]
			if origin == nil {
				origin = club
			}
			if origin.Country == "" {
				continue
			}
			playersByCountry[origin.Country] = append(playersByCountry[origin.Country], nationalPlayerCandidate{player: player})
		}
	}
	if tm.TransferEngine != nil {
		for _, player := range tm.TransferEngine.FreeAgents {
			if player == nil || player.PlayerID == "" || seen[player.PlayerID] {
				continue
			}
			seen[player.PlayerID] = true
			originID := player.OriginalClubID
			if originID == "" {
				originID = player.ClubID
			}
			origin := tm.Clubs[originID]
			if origin != nil && origin.Country != "" {
				playersByCountry[origin.Country] = append(playersByCountry[origin.Country], nationalPlayerCandidate{player: player})
			}
		}
	}

	for _, def := range domesticLeagueDefinitions {
		candidates := playersByCountry[def.Country]
		teamID := nationalTeamID(def.Country)
		team := &NationalTeam{ID: teamID, Name: def.Country, Country: def.Country, Record: NationalTeamRecord{}}
		team.PlayerIDs = selectNationalSquad(candidates)
		team.Rating = nationalSquadRating(team.PlayerIDs, candidates)
		if len(team.PlayerIDs) == 0 {
			continue
		}
		competition.TeamOrder = append(competition.TeamOrder, teamID)
		competition.Teams[teamID] = team
	}
	competition.Fixtures = generateNationalRoundRobin(competition.Season, competition.TeamOrder)
	tm.World.NationalTeams = competition
}

func selectNationalSquad(candidates []nationalPlayerCandidate) []string {
	byCategory := map[string][]nationalPlayerCandidate{"GK": {}, "DEF": {}, "MID": {}, "FWD": {}}
	for _, candidate := range candidates {
		if candidate.player == nil {
			continue
		}
		category := models.GetPositionCategory(candidate.player.Position)
		if _, ok := byCategory[category]; ok {
			byCategory[category] = append(byCategory[category], candidate)
		}
	}
	less := func(rows []nationalPlayerCandidate) {
		sort.SliceStable(rows, func(i, j int) bool {
			if rows[i].player.OVR != rows[j].player.OVR {
				return rows[i].player.OVR > rows[j].player.OVR
			}
			if rows[i].player.Age != rows[j].player.Age {
				return rows[i].player.Age < rows[j].player.Age
			}
			return rows[i].player.PlayerID < rows[j].player.PlayerID
		})
	}
	for category := range byCategory {
		less(byCategory[category])
	}

	targets := []struct {
		category string
		count    int
	}{{"GK", 3}, {"DEF", 8}, {"MID", 8}, {"FWD", 4}}
	selected := make([]string, 0, 23)
	selectedIDs := make(map[string]bool)
	for _, target := range targets {
		for i, candidate := range byCategory[target.category] {
			if i >= target.count {
				break
			}
			id := candidate.player.PlayerID
			selected = append(selected, id)
			selectedIDs[id] = true
		}
	}
	if len(selected) < 23 {
		all := append([]nationalPlayerCandidate(nil), candidates...)
		less(all)
		for _, candidate := range all {
			if len(selected) >= 23 {
				break
			}
			if candidate.player == nil || selectedIDs[candidate.player.PlayerID] {
				continue
			}
			selected = append(selected, candidate.player.PlayerID)
			selectedIDs[candidate.player.PlayerID] = true
		}
	}
	return selected
}

func nationalSquadRating(ids []string, candidates []nationalPlayerCandidate) int {
	byID := make(map[string]*models.Player, len(candidates))
	for _, row := range candidates {
		if row.player != nil {
			byID[row.player.PlayerID] = row.player
		}
	}
	byCategory := map[string][]*models.Player{"GK": {}, "DEF": {}, "MID": {}, "FWD": {}}
	for _, id := range ids {
		player := byID[id]
		if player == nil {
			continue
		}
		category := models.GetPositionCategory(player.Position)
		if _, ok := byCategory[category]; ok {
			byCategory[category] = append(byCategory[category], player)
		}
	}
	targets := []struct {
		category string
		count    int
	}{{"GK", 1}, {"DEF", 4}, {"MID", 3}, {"FWD", 3}}
	total, count := 0, 0
	for _, target := range targets {
		players := byCategory[target.category]
		sort.SliceStable(players, func(i, j int) bool {
			if players[i].OVR != players[j].OVR {
				return players[i].OVR > players[j].OVR
			}
			return players[i].PlayerID < players[j].PlayerID
		})
		for i := 0; i < len(players) && i < target.count; i++ {
			total += players[i].OVR
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return total / count
}

func generateNationalRoundRobin(season string, ids []string) []NationalTeamFixture {
	if len(ids) < 2 {
		return nil
	}
	rotation := append([]string(nil), ids...)
	if len(rotation)%2 != 0 {
		rotation = append(rotation, "")
	}
	slots := len(rotation)
	fixtures := make([]NationalTeamFixture, 0, len(ids)*(len(ids)-1)/2)
	for round := 0; round < slots-1; round++ {
		for i := 0; i < slots/2; i++ {
			a, b := rotation[i], rotation[slots-1-i]
			if a == "" || b == "" {
				continue
			}
			home, away := a, b
			if (round+i)%2 != 0 {
				home, away = away, home
			}
			fixtureID := fmt.Sprintf("NATIONS-%s-R%02d-%s-%s", strings.ReplaceAll(season, "-", ""), round+1, home, away)
			fixtures = append(fixtures, NationalTeamFixture{
				FixtureID: fixtureID,
				Matchweek: nationalGroupMatchweeks[round],
				Stage:     "League",
				HomeID:    home,
				AwayID:    away,
				Status:    "scheduled",
			})
		}
		last := rotation[slots-1]
		copy(rotation[2:], rotation[1:slots-1])
		rotation[1] = last
	}
	return fixtures
}

func (tm *TournamentManager) completeNationalTeamsUnlocked() {
	if tm == nil || tm.World == nil || tm.World.NationalTeams == nil {
		return
	}
	competition := tm.World.NationalTeams
	if competition.Stage == "Complete" {
		return
	}
	for _, matchweek := range nationalGroupMatchweeks {
		tm.advanceNationalTeamsMatchweekUnlocked(matchweek)
	}
	tm.advanceNationalTeamsMatchweekUnlocked(38)
}

// advanceNationalTeamsMatchweekUnlocked resolves only the independent
// international fixtures scheduled for this domestic matchweek. No club
// fixture, player season stat, or fatigue value is touched.
func (tm *TournamentManager) advanceNationalTeamsMatchweekUnlocked(matchweek int) {
	if tm == nil || tm.World == nil || tm.World.NationalTeams == nil {
		return
	}
	competition := tm.World.NationalTeams
	if competition.Stage == "Complete" {
		return
	}
	for i := range competition.Fixtures {
		fixture := &competition.Fixtures[i]
		if fixture.Matchweek == matchweek && fixture.Status == "scheduled" {
			tm.playNationalFixtureUnlocked(competition, fixture)
		}
	}
	if competition.Stage == "Scheduled" {
		for i := range competition.Fixtures {
			if competition.Fixtures[i].Stage == "League" && competition.Fixtures[i].Status == "finished" {
				competition.Stage = "League Phase"
				break
			}
		}
	}
	leagueComplete := true
	for i := range competition.Fixtures {
		fixture := &competition.Fixtures[i]
		if fixture.Stage == "League" && fixture.Status != "finished" {
			leagueComplete = false
			break
		}
	}
	finalScheduled := false
	finalFinished := false
	for i := range competition.Fixtures {
		fixture := &competition.Fixtures[i]
		if fixture.Stage == "Final" {
			finalScheduled = true
			finalFinished = fixture.Status == "finished"
			break
		}
	}
	if leagueComplete && !finalScheduled {
		table := nationalStandings(competition)
		if len(table) >= 2 {
			competition.Fixtures = append(competition.Fixtures, NationalTeamFixture{
				FixtureID: fmt.Sprintf("NATIONS-%s-FINAL-%s-%s", strings.ReplaceAll(competition.Season, "-", ""), table[0].TeamID, table[1].TeamID),
				Matchweek: 38, Stage: "Final", HomeID: table[0].TeamID, AwayID: table[1].TeamID, Status: "scheduled",
			})
			competition.Stage = "Final"
			finalScheduled = true
		}
	}
	if matchweek == 38 && finalScheduled && !finalFinished {
		for i := range competition.Fixtures {
			fixture := &competition.Fixtures[i]
			if fixture.Stage == "Final" && fixture.Status == "scheduled" {
				tm.playNationalFixtureUnlocked(competition, fixture)
				finalFinished = fixture.Status == "finished"
				break
			}
		}
	}
	if leagueComplete && finalFinished {
		competition.Stage = "Complete"
	}
}

func (tm *TournamentManager) playNationalFixtureUnlocked(competition *NationalTeamsCompetition, fixture *NationalTeamFixture) {
	if tm == nil || tm.World == nil || competition == nil || fixture == nil || fixture.Status != "scheduled" {
		return
	}
	home := competition.Teams[fixture.HomeID]
	away := competition.Teams[fixture.AwayID]
	if home == nil || away == nil {
		return
	}
	seed := worldSeedFor(tm.World.Seed, competition.Season+":"+fixture.FixtureID)
	rng := rand.New(rand.NewSource(seed))
	homeGoals, awayGoals := tm.simulateNationalScoreUnlocked(competition, fixture, home, away, rng)
	if fixture.Stage == "Final" && homeGoals == awayGoals {
		fixture.DecidedBy = "penalties"
		fixture.HomePens, fixture.AwayPens = nationalPenaltyShootout(rng, home.Rating, away.Rating)
		if fixture.Report != nil {
			decided := "penalties"
			fixture.Report.DecidedBy = &decided
			fixture.Report.Penalties = []int{fixture.HomePens, fixture.AwayPens}
		}
	}
	fixture.HomeGoals, fixture.AwayGoals = nationalIntPointer(homeGoals), nationalIntPointer(awayGoals)
	fixture.Status = "finished"
	if fixture.Stage == "League" {
		updateNationalRecord(&home.Record, homeGoals, awayGoals)
		updateNationalRecord(&away.Record, awayGoals, homeGoals)
	} else if homeGoals > awayGoals || (homeGoals == awayGoals && fixture.HomePens > fixture.AwayPens) {
		competition.ChampionID = fixture.HomeID
	} else {
		competition.ChampionID = fixture.AwayID
	}
}

// nationalPseudoClub builds a display-only club view of a national squad so
// the instant engine can produce a full match report. The squad references
// real players, but nothing is ever written back to club state.
func (tm *TournamentManager) nationalPseudoClubUnlocked(team *NationalTeam) *models.Club {
	if team == nil {
		return nil
	}
	squad := make([]*models.Player, 0, len(team.PlayerIDs))
	for _, playerID := range team.PlayerIDs {
		if player, _ := tm.nationalPlayerUnlocked(playerID); player != nil {
			squad = append(squad, player)
		}
	}
	if len(squad) < 11 {
		return nil
	}
	return &models.Club{
		ClubID:            team.ID,
		ClubName:          team.Name,
		ShortName:         team.Country,
		League:            "International",
		Country:           team.Country,
		HomeStadium:       team.Name + " National Stadium",
		OverallTeamRating: team.Rating,
		Squad:             squad,
		SquadSize:         len(squad),
	}
}

// simulateNationalScoreUnlocked resolves the fixture scoreline. When both
// squads resolve, the full instant engine runs and the fixture keeps its
// report (events, ratings, stats, shot map); otherwise the lightweight
// Poisson model is the fallback. Either way the result is deterministic
// for the pinned universe seed.
func (tm *TournamentManager) simulateNationalScoreUnlocked(
	competition *NationalTeamsCompetition,
	fixture *NationalTeamFixture,
	home, away *NationalTeam,
	rng *rand.Rand,
) (int, int) {
	homeClub := tm.nationalPseudoClubUnlocked(home)
	awayClub := tm.nationalPseudoClubUnlocked(away)
	if homeClub != nil && awayClub != nil {
		report := matchengine.SimulateInstantMatch(homeClub, awayClub, nil, nil, nil,
			&matchengine.InstantMatchConfig{
				Competition: competition.ID,
				Matchweek:   fixture.Matchweek,
				Weather:     "clear",
				Referee:     "balanced",
			}, rng)
		fixture.Report = report
		return report.HomeGoals, report.AwayGoals
	}
	gap := float64(home.Rating-away.Rating) / 40
	homeRate, awayRate := 1.25+gap, 1.05-gap
	if fixture.Stage == "Final" {
		homeRate, awayRate = 1.35+gap, 1.1-gap
	}
	return poissonScore(rng, homeRate, true), poissonScore(rng, awayRate, false)
}

func nationalPenaltyShootout(rng *rand.Rand, homeRating, awayRating int) (int, int) {
	chance := func(rating int) float64 {
		return math.Max(.65, math.Min(.9, .77+float64(rating-75)*.003))
	}
	shoot := func(probability float64) int {
		if rng.Float64() < probability {
			return 1
		}
		return 0
	}
	homePens, awayPens := 0, 0
	for i := 0; i < 5; i++ {
		homePens += shoot(chance(homeRating))
		awayPens += shoot(chance(awayRating))
	}
	for kicks := 0; homePens == awayPens && kicks < 10; kicks++ {
		homePens += shoot(chance(homeRating))
		awayPens += shoot(chance(awayRating))
	}
	if homePens == awayPens {
		// A bounded deterministic fallback keeps pathological test RNGs finite.
		if homeRating >= awayRating {
			homePens++
		} else {
			awayPens++
		}
	}
	return homePens, awayPens
}

func poissonScore(rng *rand.Rand, lambda float64, home bool) int {
	if home {
		lambda += .15
	}
	if lambda < .3 {
		lambda = .3
	}
	if lambda > 3.2 {
		lambda = 3.2
	}
	limit := math.Exp(-lambda)
	product, goals := 1.0, 0
	for product > limit && goals < 10 {
		goals++
		product *= rng.Float64()
	}
	return goals - 1
}

func updateNationalRecord(record *NationalTeamRecord, goalsFor, goalsAgainst int) {
	if record == nil {
		return
	}
	record.Played++
	record.GoalsFor += goalsFor
	record.GoalsAgainst += goalsAgainst
	record.GoalDifference = record.GoalsFor - record.GoalsAgainst
	if goalsFor > goalsAgainst {
		record.Won++
		record.Points += 3
	} else if goalsFor == goalsAgainst {
		record.Drawn++
		record.Points++
	} else {
		record.Lost++
	}
}

func nationalIntPointer(value int) *int { return &value }

func nationalStandings(competition *NationalTeamsCompetition) []NationalTeamTableRow {
	if competition == nil {
		return nil
	}
	rows := make([]NationalTeamTableRow, 0, len(competition.TeamOrder))
	for _, id := range competition.TeamOrder {
		team := competition.Teams[id]
		if team == nil {
			continue
		}
		record := team.Record
		rows = append(rows, NationalTeamTableRow{
			TeamID: id, Name: team.Name, Country: team.Country, Rating: team.Rating,
			Played: record.Played, Won: record.Won, Drawn: record.Drawn, Lost: record.Lost,
			GoalsFor: record.GoalsFor, GoalsAgainst: record.GoalsAgainst,
			GoalDifference: record.GoalDifference, Points: record.Points,
		})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Points != rows[j].Points {
			return rows[i].Points > rows[j].Points
		}
		if rows[i].GoalDifference != rows[j].GoalDifference {
			return rows[i].GoalDifference > rows[j].GoalDifference
		}
		if rows[i].GoalsFor != rows[j].GoalsFor {
			return rows[i].GoalsFor > rows[j].GoalsFor
		}
		return rows[i].TeamID < rows[j].TeamID
	})
	return rows
}

func (tm *TournamentManager) archiveNationalTeamsSeasonUnlocked() {
	if tm == nil || tm.World == nil || tm.World.NationalTeams == nil {
		return
	}
	competition := tm.World.NationalTeams
	if competition.Stage != "Complete" || competition.ChampionID == "" {
		return
	}
	for _, season := range competition.History {
		if season.Season == competition.Season {
			return
		}
	}
	fixtures := append([]NationalTeamFixture(nil), competition.Fixtures...)
	competition.History = append(competition.History, NationalTeamSeason{
		Season: competition.Season, ChampionID: competition.ChampionID,
		Table: nationalStandings(competition), Fixtures: fixtures,
	})
}

func (tm *TournamentManager) getNationalCompetitionUnlocked() map[string]interface{} {
	if tm == nil || tm.World == nil || tm.World.NationalTeams == nil {
		return nil
	}
	competition := tm.World.NationalTeams
	participants := make([]map[string]interface{}, 0, len(competition.TeamOrder))
	for _, id := range competition.TeamOrder {
		if team := competition.Teams[id]; team != nil {
			participants = append(participants, tm.compactNationalTeamUnlocked(team, true))
		}
	}
	table := nationalStandings(competition)
	rows := make([]map[string]interface{}, 0, len(table))
	for _, row := range table {
		rows = append(rows, map[string]interface{}{
			"team_id": row.TeamID, "name": row.Name, "country": row.Country, "rating": row.Rating,
			"played": row.Played, "won": row.Won, "drawn": row.Drawn, "lost": row.Lost,
			"goals_for": row.GoalsFor, "goals_against": row.GoalsAgainst,
			"goal_difference": row.GoalDifference, "points": row.Points,
			"team": tm.compactNationalTeamUnlocked(competition.Teams[row.TeamID], false),
		})
	}
	fixtures := make([]map[string]interface{}, 0, len(competition.Fixtures))
	for _, fixture := range competition.Fixtures {
		fixtures = append(fixtures, tm.compactNationalFixtureUnlocked(competition, fixture))
	}
	history := make([]map[string]interface{}, 0, len(competition.History))
	for _, season := range competition.History {
		rows := make([]map[string]interface{}, 0, len(season.Table))
		for _, row := range season.Table {
			rows = append(rows, nationalTableRowMap(row))
		}
		pastFixtures := make([]map[string]interface{}, 0, len(season.Fixtures))
		for _, fixture := range season.Fixtures {
			pastFixtures = append(pastFixtures, map[string]interface{}{
				"id": fixture.FixtureID, "fixture_id": fixture.FixtureID, "matchweek": fixture.Matchweek,
				"stage": fixture.Stage, "status": fixture.Status, "home_id": fixture.HomeID,
				"away_id": fixture.AwayID, "home_goals": fixture.HomeGoals, "away_goals": fixture.AwayGoals,
				"decided_by": fixture.DecidedBy, "home_penalties": fixture.HomePens, "away_penalties": fixture.AwayPens,
			})
		}
		history = append(history, map[string]interface{}{
			"season": season.Season, "champion_id": season.ChampionID,
			"champion": nationalTeamName(competition, season.ChampionID),
			"table":    rows, "fixtures": pastFixtures,
		})
	}
	return map[string]interface{}{
		"id": competition.ID, "name": competition.Name, "country": competition.Country,
		"kind": competition.Kind, "prestige": competition.Prestige, "season": competition.Season,
		"stage": competition.Stage, "eligibility_rule": competition.EligibilityRule,
		"participants": participants, "table": rows, "fixtures": fixtures,
		"champion_id": competition.ChampionID, "champion": tm.compactNationalTeamUnlocked(competition.Teams[competition.ChampionID], false),
		"history": history,
	}
}

func nationalTableRowMap(row NationalTeamTableRow) map[string]interface{} {
	return map[string]interface{}{
		"team_id": row.TeamID, "name": row.Name, "country": row.Country, "rating": row.Rating,
		"played": row.Played, "won": row.Won, "drawn": row.Drawn, "lost": row.Lost,
		"goals_for": row.GoalsFor, "goals_against": row.GoalsAgainst,
		"goal_difference": row.GoalDifference, "points": row.Points,
	}
}

func (tm *TournamentManager) compactNationalTeamUnlocked(team *NationalTeam, withPlayers bool) map[string]interface{} {
	if tm == nil || team == nil {
		return nil
	}
	result := map[string]interface{}{
		"id": team.ID, "name": team.Name, "country": team.Country,
		"rating": team.Rating, "player_count": len(team.PlayerIDs),
		"played": team.Record.Played, "points": team.Record.Points,
	}
	if !withPlayers {
		return result
	}
	players := make([]map[string]interface{}, 0, len(team.PlayerIDs))
	for _, playerID := range team.PlayerIDs {
		if player, club := tm.nationalPlayerUnlocked(playerID); player != nil {
			clubID, clubName := player.ClubID, ""
			if club != nil {
				clubID, clubName = club.ClubID, club.ClubName
			}
			players = append(players, map[string]interface{}{
				"player_id": player.PlayerID, "full_name": player.FullName,
				"position": player.Position, "category": player.Category,
				"ovr": player.OVR, "age": player.Age,
				"club_id": clubID, "club_name": clubName,
			})
		}
	}
	result["players"] = players
	return result
}

func (tm *TournamentManager) nationalPlayerUnlocked(playerID string) (*models.Player, *models.Club) {
	if tm == nil || playerID == "" {
		return nil, nil
	}
	for _, club := range tm.ClubsList {
		if club == nil {
			continue
		}
		for _, player := range club.Squad {
			if player != nil && player.PlayerID == playerID {
				return player, club
			}
		}
	}
	if tm.TransferEngine != nil {
		for _, player := range tm.TransferEngine.FreeAgents {
			if player != nil && player.PlayerID == playerID {
				return player, nil
			}
		}
	}
	return nil, nil
}

func (tm *TournamentManager) compactNationalFixtureUnlocked(competition *NationalTeamsCompetition, fixture NationalTeamFixture) map[string]interface{} {
	return map[string]interface{}{
		"id": fixture.FixtureID, "fixture_id": fixture.FixtureID,
		"matchweek": fixture.Matchweek, "competition": competition.ID,
		"stage": fixture.Stage, "status": fixture.Status,
		"home_id": fixture.HomeID, "away_id": fixture.AwayID,
		"home":       tm.compactNationalTeamUnlocked(competition.Teams[fixture.HomeID], false),
		"away":       tm.compactNationalTeamUnlocked(competition.Teams[fixture.AwayID], false),
		"home_goals": fixture.HomeGoals, "away_goals": fixture.AwayGoals,
		"decided_by": fixture.DecidedBy, "home_penalties": fixture.HomePens, "away_penalties": fixture.AwayPens,
		"has_report": fixture.Report != nil,
	}
}

// nationalTeamClubJSON is the Club-shaped wire view of a national side. The
// frontend report components read club_id/club_name/short_name/country and
// fall back to initials when no crest mapping exists.
func nationalTeamClubJSON(team *NationalTeam) map[string]interface{} {
	if team == nil {
		return nil
	}
	return map[string]interface{}{
		"club_id":             team.ID,
		"club_name":           team.Name,
		"short_name":          team.Country,
		"league":              "International",
		"country":             team.Country,
		"overall_team_rating": team.Rating,
		"home_stadium":        team.Name + " National Stadium",
	}
}

// nationsFixturePayloadUnlocked mirrors the club fixture wire shape so the
// pre-match and post-match components render international matches unchanged.
func (tm *TournamentManager) nationsFixturePayloadUnlocked(competition *NationalTeamsCompetition, fixture *NationalTeamFixture) map[string]interface{} {
	out := map[string]interface{}{
		"id":             fixture.FixtureID,
		"fixture_id":     fixture.FixtureID,
		"matchweek":      fixture.Matchweek,
		"competition":    competition.ID,
		"stage":          fixture.Stage,
		"status":         fixture.Status,
		"home_id":        fixture.HomeID,
		"away_id":        fixture.AwayID,
		"home":           nationalTeamClubJSON(competition.Teams[fixture.HomeID]),
		"away":           nationalTeamClubJSON(competition.Teams[fixture.AwayID]),
		"home_goals":     fixture.HomeGoals,
		"away_goals":     fixture.AwayGoals,
		"decided_by":     fixture.DecidedBy,
		"home_penalties": fixture.HomePens,
		"away_penalties": fixture.AwayPens,
		"is_derby":       false,
		"derby_name":     "",
		"night":          false,
		"events":         []interface{}{},
		"home_xi":        []interface{}{},
		"away_xi":        []interface{}{},
		"home_bench":     []interface{}{},
		"away_bench":     []interface{}{},
		"has_report":     fixture.Report != nil,
	}
	if fixture.Report != nil {
		report := fixture.Report
		out["report"] = report
		out["events"] = report.Events
		out["home_xi"] = report.HomeXI
		out["away_xi"] = report.AwayXI
		out["home_bench"] = report.HomeBench
		out["away_bench"] = report.AwayBench
		out["stats"] = report.Stats
		out["team_stats"] = map[string]interface{}{"home": report.Stats.Home, "away": report.Stats.Away}
		out["motm"] = report.MOTM
		out["ht_home"] = report.HTHome
		out["ht_away"] = report.HTAway
		out["attendance"] = report.Attendance
		out["shot_map"] = report.ShotMap
		out["touch_heatmap"] = report.Heatmap
		out["press_conference"] = report.PressConference
		out["home_formation"] = report.HomeFormation
		out["away_formation"] = report.AwayFormation
		if report.Referee != "" {
			out["referee"] = report.Referee
		}
		if report.Weather != "" {
			out["weather"] = report.Weather
		}
		if report.DecidedBy != nil {
			out["decided_by"] = *report.DecidedBy
		}
		if report.Penalties != nil {
			out["penalties"] = report.Penalties
		}
	}
	return out
}

// NationsFixturePayload returns one international fixture in the club
// fixture wire shape, including its match report when played.
func (tm *TournamentManager) NationsFixturePayload(fixtureID string) map[string]interface{} {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	if tm.World == nil || tm.World.NationalTeams == nil {
		return nil
	}
	competition := tm.World.NationalTeams
	for i := range competition.Fixtures {
		if competition.Fixtures[i].FixtureID == fixtureID {
			return tm.nationsFixturePayloadUnlocked(competition, &competition.Fixtures[i])
		}
	}
	return nil
}

// SimulateNationsFixture plays one scheduled international fixture on
// demand and returns the updated payload. Already-finished fixtures are
// returned unchanged so the report can be reopened safely.
func (tm *TournamentManager) SimulateNationsFixture(fixtureID string) map[string]interface{} {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if tm.World == nil || tm.World.NationalTeams == nil {
		return nil
	}
	competition := tm.World.NationalTeams
	for i := range competition.Fixtures {
		fixture := &competition.Fixtures[i]
		if fixture.FixtureID != fixtureID {
			continue
		}
		if fixture.Status == "scheduled" {
			tm.playNationalFixtureUnlocked(competition, fixture)
		}
		return tm.nationsFixturePayloadUnlocked(competition, fixture)
	}
	return nil
}

func nationalTeamName(competition *NationalTeamsCompetition, id string) string {
	if competition != nil && competition.Teams[id] != nil {
		return competition.Teams[id].Name
	}
	return ""
}
