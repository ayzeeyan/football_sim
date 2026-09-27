package transfers

import (
	"fmt"
	"sort"

	"football_sim/pkg/models"
)

// SignFreeAgents runs a deterministic registration pass over the free-agent pool.
func (te *TransferEngine) SignFreeAgents() {
	if te == nil {
		return
	}
	te.mu.Lock()
	defer te.mu.Unlock()
	te.signFreeAgentsLocked()
}

func (te *TransferEngine) signFreeAgentsLocked() {
	if len(te.FreeAgents) == 0 {
		return
	}
	clubs := te.sortedClubsLocked()
	sort.SliceStable(clubs, func(i, j int) bool {
		needI := models.MinSeniorSquadSize - len(clubs[i].Squad)
		needJ := models.MinSeniorSquadSize - len(clubs[j].Squad)
		if needI != needJ {
			return needI > needJ
		}
		if clubs[i].OverallTeamRating != clubs[j].OverallTeamRating {
			return clubs[i].OverallTeamRating > clubs[j].OverallTeamRating
		}
		return clubs[i].ClubID < clubs[j].ClubID
	})
	agents := append([]*models.Player(nil), te.FreeAgents...)
	sort.SliceStable(agents, func(i, j int) bool {
		if agents[i] == nil {
			return false
		}
		if agents[j] == nil {
			return true
		}
		if agents[i].OVR != agents[j].OVR {
			return agents[i].OVR > agents[j].OVR
		}
		return agents[i].PlayerID < agents[j].PlayerID
	})

	for _, club := range clubs {
		if club == nil || len(club.Squad) >= models.MaxSeniorSquadSize {
			continue
		}
		needFloor := len(club.Squad) < models.MinSeniorSquadSize
		if !needFloor && !te.clubHasPositionNeedLocked(club) {
			continue
		}
		for _, p := range agents {
			if p == nil || !p.IsFreeAgent() {
				continue
			}
			if te.playerInPoolLocked(p.PlayerID) == nil {
				continue
			}
			if !te.canRegisterFreeAgentLocked(club, p, nil) {
				continue
			}
			if !needFloor && !te.freeAgentAcceptsLocked(p, club) {
				continue
			}
			if te.signFreeAgentLocked(p, club) && !needFloor {
				break
			}
			if needFloor && len(club.Squad) >= models.MinSeniorSquadSize {
				break
			}
		}
	}
}

func (te *TransferEngine) clubHasPositionNeedLocked(club *models.Club) bool {
	if club == nil {
		return false
	}
	counts := map[string]int{}
	for _, p := range club.Squad {
		if p != nil {
			counts[p.Category]++
		}
	}
	return counts["GK"] < 2 || counts["DEF"] < 6 || counts["MID"] < 6 || counts["FWD"] < 4
}

func (te *TransferEngine) canRegisterFreeAgentLocked(buyer *models.Club, p *models.Player, seller *models.Club) bool {
	if buyer == nil || p == nil {
		return false
	}
	if seller != nil && buyer.ClubID == seller.ClubID {
		return false
	}
	if te.committedSquadSize(buyer.ClubID) >= models.MaxSeniorSquadSize {
		return false
	}
	if isCanonicalWonderkid(p) && !isSuperLeagueClub(buyer.ClubID) {
		return false
	}
	if !te.canAffordWithWage(buyer, 0, annualWageFor(p)) {
		return false
	}
	if seller != nil && !PlayerAcceptsDestination(p, seller, buyer) {
		return false
	}
	for _, bp := range buyer.Squad {
		if bp != nil && bp.PlayerID == p.PlayerID {
			return false
		}
	}
	return true
}

func (te *TransferEngine) signFreeAgentLocked(p *models.Player, buyer *models.Club) bool {
	if p == nil || buyer == nil || !p.IsFreeAgent() {
		return false
	}
	if !te.canRegisterFreeAgentLocked(buyer, p, nil) {
		return false
	}
	if te.removeFreeAgentLocked(p.PlayerID) == nil {
		return false
	}
	fromID := p.PreviousClubID
	p.MarkRegistered(buyer.ClubID)
	p.ContractYears = p.ResignContractYears()
	p.TransferRequested = false
	if p.WageEUR <= 0 {
		p.WageEUR = models.WageForOVR(p.OVR)
	}
	buyer.Squad = append(buyer.Squad, p)
	buyer.RecalculateRatings()
	te.syncManagerBudget(buyer.ClubID)
	te.TransferredThisWindow[p.PlayerID] = true
	p.RecordMove("", fromID, buyer.ClubID, models.MoveFree, 0, te.CurrentMatchweek)
	p.RecordContractEvent("", buyer.ClubID, models.ContractEventSigned, p.ContractYears)
	completed := CompletedTransfer{
		PlayerID: p.PlayerID, PlayerName: p.FullName, PlayerPos: p.Position, PlayerOVR: p.OVR,
		IsWonderkid: p.UniverseWonderkid, SellerID: fromID, BuyerID: buyer.ClubID,
		BuyerName: buyer.ClubName, BuyerShort: buyer.ShortName, FeeEUR: 0,
		FormattedFee: "a free", Matchweek: te.CurrentMatchweek,
	}
	if from := te.Clubs[fromID]; from != nil {
		completed.SellerName = from.ClubName
		completed.SellerShort = from.ShortName
	}
	te.CompletedTransfers = append(te.CompletedTransfers, completed)
	te.AllTimeTransfers = append(te.AllTimeTransfers, completed)
	te.prependFeed(TransferFeedItem{
		Headline:    fmt.Sprintf("HERE WE GO: %s joins %s as a free agent", p.FullName, buyer.ShortName),
		Category:    "HERE_WE_GO",
		IsWonderkid: p.UniverseWonderkid,
		Matchweek:   te.CurrentMatchweek,
		Timestamp:   fmt.Sprintf("Week %d", te.CurrentWeek),
	})
	return true
}

func (te *TransferEngine) freeAgentAcceptsLocked(p *models.Player, buyer *models.Club) bool {
	if p == nil || buyer == nil {
		return false
	}
	seller := te.Clubs[p.PreviousClubID]
	if seller == nil {
		return true
	}
	return PlayerAcceptsDestination(p, seller, buyer)
}

func (te *TransferEngine) playerInPoolLocked(playerID string) *models.Player {
	for _, p := range te.FreeAgents {
		if p != nil && p.PlayerID == playerID {
			return p
		}
	}
	return nil
}

// FreeAgentList returns a copy of the unattached pool in PlayerID order.
func (te *TransferEngine) FreeAgentList() []*models.Player {
	if te == nil {
		return nil
	}
	te.mu.RLock()
	defer te.mu.RUnlock()
	out := append([]*models.Player(nil), te.FreeAgents...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i] == nil {
			return false
		}
		if out[j] == nil {
			return true
		}
		return out[i].PlayerID < out[j].PlayerID
	})
	return out
}
