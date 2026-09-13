package tournament

import (
	"sort"

	"football_sim/pkg/models"
)

func (tm *TournamentManager) worldLeagueStandingsUnlocked(competitionID string) []*models.Club {
	comp := tm.worldCompetitionUnlocked(competitionID)
	if comp == nil || comp.Kind != CompetitionLeague {
		return nil
	}
	clubs := clubsForIDs(tm.Clubs, comp.ParticipantIDs)
	models.SortClubs(clubs)
	return clubs
}

// GetCompetitions powers the world-facing Competition Hub. Entries are sorted
// by stable configured order instead of Go map iteration, preserving universe
// determinism and a consistent UI.
func (tm *TournamentManager) GetCompetitions() []map[string]interface{} {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	if tm.World == nil {
		return []map[string]interface{}{}
	}
	out := make([]map[string]interface{}, 0, len(tm.World.CompetitionOrder))
	for _, id := range tm.World.CompetitionOrder {
		comp := tm.World.Competitions[id]
		if comp == nil {
			continue
		}
		entry := map[string]interface{}{
			"id": comp.ID, "name": comp.Name, "country": comp.Country,
			"kind": comp.Kind, "stage": comp.Stage, "prestige": comp.Prestige,
			"participants": len(comp.ParticipantIDs), "champion": compactClub(tm.Clubs[comp.ChampionID]),
			"champion_id": comp.ChampionID,
		}
		out = append(out, entry)
	}
	return out
}

// GetCompetition returns a single source of truth for league tables,
// European league-phase records, knockout rounds, qualification provenance,
// and fixtures on the shared calendar.
func (tm *TournamentManager) GetCompetition(id string) map[string]interface{} {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	comp := tm.worldCompetitionUnlocked(id)
	if comp == nil {
		return nil
	}
	fixtures := tm.worldCompetitionFixturesUnlocked(id)
	compactFixtures := make([]map[string]interface{}, 0, len(fixtures))
	byID := make(map[string]map[string]interface{}, len(fixtures))
	for i := range fixtures {
		row := tm.compactCompetitionFixtureUnlocked(&fixtures[i])
		compactFixtures = append(compactFixtures, row)
		byID[fixtures[i].FixtureID] = row
	}
	rows := make([]map[string]interface{}, 0, len(comp.ParticipantIDs))
	if comp.Kind == CompetitionLeague {
		for _, club := range tm.worldLeagueStandingsUnlocked(id) {
			rows = append(rows, worldClubTableRow(club, nil))
		}
	} else if comp.Kind == CompetitionEuropean {
		for _, club := range tm.worldEuropeanStandingsUnlocked(comp) {
			rows = append(rows, worldClubTableRow(club, comp.Records[club.ClubID]))
		}
	}
	rounds := make([]map[string]interface{}, 0, len(comp.Rounds))
	for _, round := range comp.Rounds {
		ties := make([]map[string]interface{}, 0, len(round.FixtureIDs))
		for _, fid := range round.FixtureIDs {
			if row := byID[fid]; row != nil {
				ties = append(ties, row)
			}
		}
		rounds = append(rounds, map[string]interface{}{
			"stage": round.Stage, "fixture_ids": round.FixtureIDs, "entrant_ids": round.EntrantIDs,
			"bye_ids": round.ByeIDs, "winner_ids": round.WinnerIDs, "tie_ids": round.TieIDs, "ties": ties,
		})
	}
	participants := make([]map[string]interface{}, 0, len(comp.ParticipantIDs))
	for _, club := range clubsForIDs(tm.Clubs, comp.ParticipantIDs) {
		participants = append(participants, compactClub(club))
	}
	return map[string]interface{}{
		"id": comp.ID, "name": comp.Name, "country": comp.Country, "kind": comp.Kind,
		"prestige": comp.Prestige, "stage": comp.Stage, "participants": participants,
		"qualification_sources": comp.QualificationSources, "pots": comp.Pots,
		"table": rows, "rounds": rounds,
		"fixtures": compactFixtures, "champion": compactClub(tm.Clubs[comp.ChampionID]),
		"champion_id": comp.ChampionID,
	}
}

func compactClub(club *models.Club) map[string]interface{} {
	if club == nil {
		return nil
	}
	return map[string]interface{}{
		"club_id": club.ClubID, "club_name": club.ClubName, "short_name": club.ShortName,
		"league": club.League, "country": club.Country,
		"primary_color": club.PrimaryColor, "secondary_color": club.SecondaryColor,
		"form": club.Form, "p": club.Played, "w": club.Won, "d": club.Drawn, "l": club.Lost,
		"gf": club.GoalsFor, "ga": club.GoalsAgainst, "gd": club.GoalDifference, "pts": club.Points,
		"coefficient": club.Coefficient,
	}
}

func (tm *TournamentManager) compactCompetitionFixtureUnlocked(f *Fixture) map[string]interface{} {
	if f == nil {
		return nil
	}
	home, away := f.Home, f.Away
	if home == nil {
		home = tm.Clubs[f.HomeID]
	}
	if away == nil {
		away = tm.Clubs[f.AwayID]
	}
	return map[string]interface{}{
		"id": f.FixtureID, "fixture_id": f.FixtureID, "matchweek": f.Matchweek,
		"competition": f.Competition, "stage": f.Stage, "status": f.Status,
		"home_id": f.HomeID, "away_id": f.AwayID,
		"home": compactClub(home), "away": compactClub(away),
		"home_goals": f.HomeGoals, "away_goals": f.AwayGoals,
		"decided_by": f.DecidedBy, "penalties": f.Penalties,
		"leg": f.Leg, "tie_id": f.TieID,
	}
}

func worldClubTableRow(club *models.Club, record *models.CompetitionRecord) map[string]interface{} {
	if club == nil {
		return map[string]interface{}{}
	}
	if record == nil {
		return map[string]interface{}{
			"club": compactClub(club), "club_id": club.ClubID, "played": club.Played, "won": club.Won,
			"drawn": club.Drawn, "lost": club.Lost, "goals_for": club.GoalsFor, "goals_against": club.GoalsAgainst,
			"goal_difference": club.GoalDifference, "points": club.Points, "form": club.Form,
		}
	}
	return map[string]interface{}{
		"club": compactClub(club), "club_id": club.ClubID, "played": record.Played, "won": record.Won,
		"drawn": record.Drawn, "lost": record.Lost, "goals_for": record.GoalsFor, "goals_against": record.GoalsAgainst,
		"goal_difference": record.GoalDifference, "points": record.Points, "form": record.Form,
	}
}

func (tm *TournamentManager) worldCompetitionFixturesUnlocked(id string) []Fixture {
	fixtures := []Fixture{}
	if tm.isWorldDomesticLeague(id) {
		for _, f := range tm.Fixtures {
			if f.Competition == id {
				fixtures = append(fixtures, f)
			}
		}
	} else if tm.World != nil {
		for _, f := range tm.World.Fixtures {
			if f.Competition == id {
				fixtures = append(fixtures, f)
			}
		}
	}
	sort.SliceStable(fixtures, func(i, j int) bool {
		if fixtures[i].Matchweek != fixtures[j].Matchweek {
			return fixtures[i].Matchweek < fixtures[j].Matchweek
		}
		return fixtures[i].FixtureID < fixtures[j].FixtureID
	})
	return fixtures
}

func (tm *TournamentManager) WorldFixturesCopy() []Fixture {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	if tm.World == nil {
		return []Fixture{}
	}
	out := append([]Fixture(nil), tm.World.Fixtures...)
	return out
}
