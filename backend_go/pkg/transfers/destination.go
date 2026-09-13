package transfers

import (
	"strings"

	"football_sim/pkg/models"
)

func leaguePrestige(league string) int {
	switch strings.TrimSpace(league) {
	case "Premier League":
		return 5
	case "La Liga":
		return 4
	case "Bundesliga", "Serie A":
		return 4
	case "Ligue 1":
		return 3
	default:
		return 2
	}
}

// PlayerAcceptsDestination is the sporting filter for a move. Fee is not enough.
func PlayerAcceptsDestination(player *models.Player, seller, buyer *models.Club) bool {
	if player == nil || seller == nil || buyer == nil {
		return false
	}
	if player.OnLoan {
		return false
	}
	if seller.ClubID == buyer.ClubID {
		return false
	}
	if player.TransferRequested {
		return true
	}
	score := 0
	score += (buyer.Identity.Reputation - seller.Identity.Reputation) / 8
	if buyer.OverallTeamRating > 0 && seller.OverallTeamRating > 0 {
		score += (buyer.OverallTeamRating - seller.OverallTeamRating) / 4
		if player.OVR >= 84 && buyer.OverallTeamRating+4 < seller.OverallTeamRating {
			score -= 8
		}
		if player.SquadRole == models.RoleCrucial && buyer.OverallTeamRating < seller.OverallTeamRating {
			score -= 5
		}
	}
	if buyer.League != "" && seller.League != "" {
		score += leaguePrestige(buyer.League) - leaguePrestige(seller.League)
	}
	if player.Morale > 0 && player.Morale < 45 {
		score += 4
	}
	if player.Age <= 21 && buyer.Identity.YouthPreference >= 70 {
		score += 2
	}
	return score >= -1
}

func (te *TransferEngine) categoryCount(club *models.Club, category string) int {
	if club == nil {
		return 0
	}
	n := 0
	for _, p := range club.Squad {
		if p != nil && p.Category == category {
			n++
		}
	}
	return n
}

func (te *TransferEngine) positionNeedScore(buyer *models.Club, category string) int {
	have := te.categoryCount(buyer, category)
	want := map[string]int{"GK": 2, "DEF": 7, "MID": 7, "FWD": 5}[category]
	if want == 0 {
		want = 5
	}
	need := want - have
	if need < 0 {
		return 0
	}
	return need
}

func (te *TransferEngine) scoreTransferTarget(buyer, seller *models.Club, p *models.Player) int {
	if p == nil || buyer == nil || seller == nil {
		return -1000
	}
	score := p.OVR + te.positionNeedScore(buyer, p.Category)*6
	if p.TransferRequested {
		score += 8
	}
	if p.Age <= 23 {
		score += buyer.Identity.YouthPreference / 20
	}
	if !PlayerAcceptsDestination(p, seller, buyer) {
		return -1000
	}
	return score
}
