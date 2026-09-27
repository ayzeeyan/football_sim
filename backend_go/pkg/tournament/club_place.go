package tournament

import "football_sim/pkg/models"

func (tm *TournamentManager) leaguePlaceUnlocked(club *models.Club) int {
	if tm == nil || club == nil {
		return 0
	}
	for i, ranked := range tm.clubLeagueTableUnlocked(club) {
		if ranked != nil && ranked.ClubID == club.ClubID {
			return i + 1
		}
	}
	return 0
}
