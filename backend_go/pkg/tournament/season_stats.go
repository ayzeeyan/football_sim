package tournament

import (
	"sort"

	"football_sim/pkg/models"
)

// GetSeasonStats compiles the standings-tab stat blocks: scoring races,
// player of the week, monthly awards, and archived seasons. Slices are never
// nil so the React contract (`.map`/`.length`) holds from matchweek 1.
func (tm *TournamentManager) GetSeasonStats() map[string]interface{} {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var all []*models.Player
	for _, c := range tm.ClubsList {
		all = append(all, c.Squad...)
	}
	row := func(p *models.Player) map[string]interface{} {
		club := tm.Clubs[p.ClubID]
		cName, cShort := p.ClubID, ""
		if club != nil {
			cName, cShort = club.ClubName, club.ShortName
		}
		return map[string]interface{}{
			"player_id":    p.PlayerID,
			"full_name":    p.FullName,
			"position":     p.Position,
			"ovr":          p.OVR,
			"goals":        p.Goals,
			"assists":      p.Assists,
			"appearances":  p.Appearances,
			"club_name":    cName,
			"club_short":   cShort,
			"is_wonderkid": p.UniverseWonderkid,
		}
	}
	byGoals := append([]*models.Player{}, all...)
	sort.Slice(byGoals, func(i, j int) bool {
		if byGoals[i].Goals != byGoals[j].Goals {
			return byGoals[i].Goals > byGoals[j].Goals
		}
		if byGoals[i].Assists != byGoals[j].Assists {
			return byGoals[i].Assists > byGoals[j].Assists
		}
		return byGoals[i].OVR > byGoals[j].OVR
	})
	byAssists := append([]*models.Player{}, all...)
	sort.Slice(byAssists, func(i, j int) bool {
		if byAssists[i].Assists != byAssists[j].Assists {
			return byAssists[i].Assists > byAssists[j].Assists
		}
		if byAssists[i].Goals != byAssists[j].Goals {
			return byAssists[i].Goals > byAssists[j].Goals
		}
		return byAssists[i].OVR > byAssists[j].OVR
	})
	scorers := []map[string]interface{}{}
	for i := 0; i < 10 && i < len(byGoals); i++ {
		scorers = append(scorers, row(byGoals[i]))
	}
	assisters := []map[string]interface{}{}
	for i := 0; i < 10 && i < len(byAssists); i++ {
		assisters = append(assisters, row(byAssists[i]))
	}
	monthly := []map[string]interface{}{}
	monthly = append(monthly, tm.MonthlyAwards...)
	history := []map[string]interface{}{}
	history = append(history, tm.SeasonHistory...)
	return map[string]interface{}{
		"scorers":            scorers,
		"assisters":          assisters,
		"player_of_the_week": tm.PlayerOfTheWeek,
		"monthly_awards":     monthly,
		"history":            history,
	}
}
