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
	// Wanting out reduces a player's resistance; it does not erase sporting
	// ambition. Elite crucial players still reject a severe downgrade, while
	// unsettled squad players can accept a modest step down for a fresh start.
	if player.TransferRequested {
		score += 5
	}
	if player.Personality == "snake" {
		score += 7
	}
	return score >= -1
}

func tacticalPositionGroup(position, category string) string {
	pos := strings.ToUpper(strings.TrimSpace(position))
	switch pos {
	case "GK":
		return "GK"
	case "CB", "LCB", "RCB":
		return "CB"
	case "LB", "RB", "LWB", "RWB":
		return "FB"
	case "LW", "RW", "LM", "RM":
		return "WIDE"
	case "ST", "CF":
		return "ST"
	case "CM", "CDM", "CAM", "LCM", "RCM":
		return "MID"
	}
	switch strings.ToUpper(strings.TrimSpace(category)) {
	case "GK":
		return "GK"
	case "DEF":
		return "CB"
	case "FWD":
		return "ST"
	default:
		return "MID"
	}
}

func (te *TransferEngine) positionGroupCount(club *models.Club, group string) int {
	if club == nil {
		return 0
	}
	n := 0
	for _, p := range club.Squad {
		if p != nil && tacticalPositionGroup(p.Position, p.Category) == group {
			n++
		}
	}
	return n
}

func (te *TransferEngine) positionNeedScore(buyer *models.Club, p *models.Player) int {
	group := tacticalPositionGroup(p.Position, p.Category)
	have := te.positionGroupCount(buyer, group)
	want := map[string]int{"GK": 2, "CB": 4, "FB": 4, "MID": 7, "WIDE": 4, "ST": 3}[group]
	need := want - have
	if need < -2 {
		return -2
	}
	return need
}

func (te *TransferEngine) scoreTransferTarget(buyer, seller *models.Club, p *models.Player) int {
	if p == nil || buyer == nil || seller == nil {
		return -1000
	}
	score := p.OVR + te.positionNeedScore(buyer, p)*6
	qualityGap := p.OVR - buyer.OverallTeamRating
	ambition := buyer.Identity.Clamp().RecruitmentAmbition
	if qualityGap < 0 {
		// Ambitious clubs are more reluctant to spend scarce budget and roster
		// space on players well below their current first-team level.
		score += qualityGap * (2 + ambition/25)
	} else {
		score += qualityGap * (1 + ambition/50)
	}
	if p.TransferRequested {
		score += 8
	}
	if p.OutOfContract() {
		score += 20
	} else if p.ContractYears <= 1 {
		// A short deal is a cheaper opportunity, but an established long-term
		// contract should cost a little more and make the target less attractive.
		score += 8
	} else if p.ContractYears >= 4 {
		score -= 4
	}
	if p.Age <= 23 {
		score += buyer.Identity.YouthPreference / 20
	}
	fee := sellerAcceptanceFloor(sellerAskingPrice(p, seller), p, seller)
	budget := buyer.Finances.TransferBudget
	if buyer.Finances.Balance < budget {
		budget = buyer.Finances.Balance
	}
	if budget > 0 && fee > 0 {
		spendShare := float64(fee) / float64(budget)
		if spendShare > 0.5 {
			score -= int((spendShare - 0.5) * 40)
		}
	}
	if !PlayerAcceptsDestination(p, seller, buyer) {
		return -1000
	}
	return score
}
