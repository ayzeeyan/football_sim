package tournament

import (
	"sort"

	"football_sim/pkg/models"
)

// AssignBoardExpectationsUnlocked sets a league-relative finish target and a
// short board brief. Reputation and squad rating matter more than one cup run.
func (tm *TournamentManager) AssignBoardExpectationsUnlocked() {
	if tm == nil {
		return
	}
	byLeague := map[string][]*models.Club{}
	for _, club := range tm.ClubsList {
		if club == nil {
			continue
		}
		league := club.League
		if league == "" {
			league = "Super League"
		}
		byLeague[league] = append(byLeague[league], club)
	}
	for _, clubs := range byLeague {
		ranked := append([]*models.Club(nil), clubs...)
		sort.SliceStable(ranked, func(i, j int) bool {
			ri, rj := ranked[i].Identity.Reputation, ranked[j].Identity.Reputation
			if ri != rj {
				return ri > rj
			}
			if ranked[i].OverallTeamRating != ranked[j].OverallTeamRating {
				return ranked[i].OverallTeamRating > ranked[j].OverallTeamRating
			}
			if ranked[i].Identity.FinancialPower != ranked[j].Identity.FinancialPower {
				return ranked[i].Identity.FinancialPower > ranked[j].Identity.FinancialPower
			}
			return ranked[i].ClubID < ranked[j].ClubID
		})
		n := len(ranked)
		for i, club := range ranked {
			club.ExpectedFinish = i + 1
			club.BoardObjective = boardObjectiveFor(i+1, n)
		}
	}
}

func boardObjectiveFor(place, n int) string {
	if n <= 0 {
		return "Competitive finish"
	}
	switch {
	case place <= 1:
		return "Title challenge"
	case n >= 16 && place <= 4:
		return "Champions League qualification"
	case n >= 16 && place <= 8:
		return "European qualification"
	case place <= (n+1)/2:
		return "Top-half finish"
	default:
		return "Survival"
	}
}

func (tm *TournamentManager) clubLeagueTableUnlocked(club *models.Club) []*models.Club {
	if club != nil && tm.World != nil {
		for _, def := range domesticLeagueDefinitions {
			if def.League == club.League {
				if table := tm.worldLeagueStandingsUnlocked(def.ID); len(table) > 0 {
					return table
				}
			}
		}
	}
	return tm.standingsUnlocked()
}
