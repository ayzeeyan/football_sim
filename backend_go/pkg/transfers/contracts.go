package transfers

import (
	"fmt"
	"sort"

	"football_sim/pkg/models"
)

// ResolveExpiredContracts re-signs players who want to stay and releases the
// rest. Destination choice is deterministic: highest squad rating, then ClubID,
// subject to wage cap and wonderkid club rules. Players with no legal buyer
// become free agents rather than receiving an artificial extra year.
func (te *TransferEngine) ResolveExpiredContracts() {
	if te == nil {
		return
	}
	te.mu.Lock()
	defer te.mu.Unlock()
	te.resolveExpiredContractsLocked()
	te.signFreeAgentsLocked()
}

func (te *TransferEngine) resolveExpiredContractsLocked() {
	clubs := te.sortedClubsLocked()

	type row struct {
		player *models.Player
		club   *models.Club
	}
	var leavers []row
	for _, club := range clubs {
		for _, p := range club.Squad {
			if p == nil || !p.OutOfContract() {
				continue
			}
			if !p.WantsToLeaveOnFree() {
				years := p.ResignContractYears()
				p.ContractYears = years
				p.RecordContractEvent("", club.ClubID, models.ContractEventRenewed, years)
				continue
			}
			leavers = append(leavers, row{player: p, club: club})
		}
	}
	sort.SliceStable(leavers, func(i, j int) bool {
		return leavers[i].player.PlayerID < leavers[j].player.PlayerID
	})
	for _, item := range leavers {
		dest := te.pickFreeDestination(item.player, item.club, clubs)
		if dest != nil {
			neg := &TransferNegotiation{
				Player:      item.player,
				Buyer:       dest,
				Seller:      item.club,
				CurrentBid:  0,
				IsWonderkid: item.player.UniverseWonderkid,
			}
			if te.executeTransfer(neg) {
				continue
			}
		}
		te.releaseToFreeAgencyLocked(item.player, item.club)
	}
}

func (te *TransferEngine) pickFreeDestination(p *models.Player, seller *models.Club, clubs []*models.Club) *models.Club {
	if p == nil || seller == nil {
		return nil
	}
	ranked := append([]*models.Club(nil), clubs...)
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].OverallTeamRating != ranked[j].OverallTeamRating {
			return ranked[i].OverallTeamRating > ranked[j].OverallTeamRating
		}
		return ranked[i].ClubID < ranked[j].ClubID
	})
	for _, buyer := range ranked {
		if buyer == nil || buyer.ClubID == seller.ClubID {
			continue
		}
		if !te.canRegisterFreeAgentLocked(buyer, p, seller) {
			continue
		}
		return buyer
	}
	return nil
}

func (te *TransferEngine) releaseToFreeAgencyLocked(p *models.Player, club *models.Club) {
	if p == nil {
		return
	}
	fromID := ""
	if club != nil {
		fromID = club.ClubID
		kept := club.Squad[:0]
		for _, sp := range club.Squad {
			if sp != nil && sp.PlayerID != p.PlayerID {
				kept = append(kept, sp)
			}
		}
		club.Squad = kept
		club.RecalculateRatings()
	}
	p.RecordContractEvent("", fromID, models.ContractEventExpired, 0)
	p.RecordContractEvent("", fromID, models.ContractEventReleased, 0)
	p.RecordMove("", fromID, "", models.MoveFree, 0, te.CurrentMatchweek)
	p.MarkFreeAgent(fromID)
	te.addFreeAgentLocked(p)
	name := fromID
	if club != nil {
		name = club.ShortName
	}
	te.prependFeed(TransferFeedItem{
		Headline:    fmt.Sprintf("%s is a free agent after leaving %s", p.FullName, name),
		Category:    "RUMOR",
		IsWonderkid: p.UniverseWonderkid,
		Matchweek:   te.CurrentMatchweek,
		Timestamp:   fmt.Sprintf("Week %d", te.CurrentWeek),
	})
}

func (te *TransferEngine) addFreeAgentLocked(p *models.Player) {
	if p == nil {
		return
	}
	for _, existing := range te.FreeAgents {
		if existing != nil && existing.PlayerID == p.PlayerID {
			return
		}
	}
	te.FreeAgents = append(te.FreeAgents, p)
	sort.SliceStable(te.FreeAgents, func(i, j int) bool {
		a, b := te.FreeAgents[i], te.FreeAgents[j]
		if a == nil {
			return false
		}
		if b == nil {
			return true
		}
		return a.PlayerID < b.PlayerID
	})
}

func (te *TransferEngine) removeFreeAgentLocked(playerID string) *models.Player {
	if playerID == "" {
		return nil
	}
	kept := te.FreeAgents[:0]
	var found *models.Player
	for _, p := range te.FreeAgents {
		if p != nil && p.PlayerID == playerID && found == nil {
			found = p
			continue
		}
		kept = append(kept, p)
	}
	te.FreeAgents = kept
	return found
}

func (te *TransferEngine) sortedClubsLocked() []*models.Club {
	clubs := make([]*models.Club, 0, len(te.Clubs))
	for _, club := range te.Clubs {
		if club != nil {
			clubs = append(clubs, club)
		}
	}
	sort.Slice(clubs, func(i, j int) bool { return clubs[i].ClubID < clubs[j].ClubID })
	return clubs
}
