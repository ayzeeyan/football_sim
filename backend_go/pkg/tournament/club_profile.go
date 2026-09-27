package tournament

import (
	"sort"

	"football_sim/pkg/models"
	"football_sim/pkg/transfers"
)

func (tm *TournamentManager) ClubScheduleUnlocked(clubID string) []Fixture {
	if tm == nil || clubID == "" {
		return nil
	}
	src := tm.Fixtures
	if tm.World != nil {
		src = tm.World.Fixtures
	}
	out := make([]Fixture, 0, 48)
	for i := range src {
		f := src[i]
		if f.HomeID == clubID || f.AwayID == clubID {
			out = append(out, f)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Matchweek != out[j].Matchweek {
			return out[i].Matchweek < out[j].Matchweek
		}
		return out[i].FixtureID < out[j].FixtureID
	})
	return out
}

func (tm *TournamentManager) ClubCompetitionsUnlocked(clubID string) []map[string]interface{} {
	if tm == nil || tm.World == nil || clubID == "" {
		return nil
	}
	var out []map[string]interface{}
	for _, id := range tm.World.CompetitionOrder {
		comp := tm.World.Competitions[id]
		if comp == nil {
			continue
		}
		for _, pid := range comp.ParticipantIDs {
			if pid != clubID {
				continue
			}
			out = append(out, map[string]interface{}{
				"id":     comp.ID,
				"name":   comp.Name,
				"kind":   comp.Kind,
				"stage":  comp.Stage,
				"source": comp.QualificationSources[clubID],
			})
			break
		}
	}
	return out
}

func (tm *TournamentManager) ClubLeaguePlaceUnlocked(clubID string) (int, int, *models.Club) {
	club := tm.Clubs[clubID]
	if club == nil {
		return 0, 0, nil
	}
	table := tm.worldLeagueStandingsUnlocked(leagueIDForName(club.League))
	if len(table) == 0 {
		table = tm.standingsUnlocked()
	}
	for i, row := range table {
		if row != nil && row.ClubID == clubID {
			return i + 1, len(table), club
		}
	}
	return 0, len(table), club
}

func (tm *TournamentManager) ClubTransferActivityUnlocked(clubID string) map[string]interface{} {
	out := map[string]interface{}{
		"arrivals":   []transfers.CompletedTransfer{},
		"departures": []transfers.CompletedTransfer{},
		"loans_in":   []map[string]interface{}{},
		"loans_out":  []map[string]interface{}{},
		"spent":      int64(0),
		"received":   int64(0),
		"net_spend":  int64(0),
	}
	if tm == nil || tm.TransferEngine == nil || clubID == "" {
		return out
	}
	var arrivals, departures []transfers.CompletedTransfer
	var spent, received int64
	seen := map[string]bool{}
	appendDeal := func(deal transfers.CompletedTransfer) {
		key := deal.PlayerID + ":" + deal.BuyerID + ":" + deal.SellerID + ":" + deal.FormattedFee
		if seen[key] {
			return
		}
		seen[key] = true
		if deal.BuyerID == clubID {
			arrivals = append(arrivals, deal)
			spent += deal.FeeEUR
		}
		if deal.SellerID == clubID {
			departures = append(departures, deal)
			received += deal.FeeEUR
		}
	}
	for _, deal := range tm.TransferEngine.CompletedTransfers {
		appendDeal(deal)
	}
	if len(arrivals)+len(departures) == 0 {
		for _, deal := range tm.TransferEngine.AllTimeTransfers {
			appendDeal(deal)
		}
	}
	var loansIn, loansOut []map[string]interface{}
	if club := tm.Clubs[clubID]; club != nil {
		for _, p := range club.Squad {
			if p == nil || !p.OnLoan {
				continue
			}
			loansIn = append(loansIn, map[string]interface{}{
				"player_id": p.PlayerID, "full_name": p.FullName, "from_club_id": p.ParentClubID,
			})
		}
	}
	for _, other := range tm.ClubsList {
		if other == nil || other.ClubID == clubID {
			continue
		}
		for _, p := range other.Squad {
			if p != nil && p.OnLoan && p.ParentClubID == clubID {
				loansOut = append(loansOut, map[string]interface{}{
					"player_id": p.PlayerID, "full_name": p.FullName, "to_club_id": other.ClubID, "to_club_name": other.ClubName,
				})
			}
		}
	}
	sort.SliceStable(arrivals, func(i, j int) bool { return arrivals[i].PlayerID < arrivals[j].PlayerID })
	sort.SliceStable(departures, func(i, j int) bool { return departures[i].PlayerID < departures[j].PlayerID })
	out["arrivals"] = arrivals
	out["departures"] = departures
	out["loans_in"] = loansIn
	out["loans_out"] = loansOut
	out["spent"] = spent
	out["received"] = received
	out["net_spend"] = spent - received
	return out
}
