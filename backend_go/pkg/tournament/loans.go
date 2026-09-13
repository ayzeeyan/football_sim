package tournament

import (
	"fmt"
	"sort"

	"football_sim/pkg/models"
	"football_sim/pkg/transfers"
)

const (
	maxOutgoingLoans = 3
	maxIncomingLoans = 2
)

// loanBuyClauseFor agrees an optional permanent-transfer fee when a loan
// starts. Clauses go to financially solid destinations (power 60+) for
// established prospects (66+ OVR) at a 25% premium over the valuation anchor,
// always inside valuation clamps (floor €300k, ceiling €500M). Canonical
// wonderkids never carry clauses (they are never loaned). Deterministic.
func loanBuyClauseFor(p *models.Player, dest *models.Club) int64 {
	if p == nil || dest == nil || p.UniverseWonderkid {
		return 0
	}
	if dest.Identity.FinancialPower < 60 || p.OVR < 66 {
		return 0
	}
	anchor := models.BaselineValue(p.OVR, p.Age, p.UniverseWonderkid)
	return models.ClampValue(anchor+anchor/4, p.OVR, p.Age, p.UniverseWonderkid)
}

// executeLoanBuyClausesUnlocked converts loans with affordable, accepted buy
// clauses into permanent transfers instead of returns. Runs at the season
// boundary (window closed), so the one-move-per-window invariant is preserved
// by the next BeginOffSeasonWindow reset. Squad membership does not change
// (the player is already at the buyer), only flags and finances move, with
// the seller reinvesting 100% capped by balance. PlayerID order keeps it
// deterministic. Returns the number of triggered clauses.
func (tm *TournamentManager) executeLoanBuyClausesUnlocked() int {
	if tm == nil {
		return 0
	}
	type loaned struct {
		player *models.Player
		dest   *models.Club
	}
	var cands []loaned
	for _, club := range tm.ClubsList {
		if club == nil {
			continue
		}
		for _, p := range club.Squad {
			if p != nil && p.OnLoan && p.ParentClubID != "" && p.ParentClubID != club.ClubID && p.LoanBuyClauseEUR > 0 {
				cands = append(cands, loaned{player: p, dest: club})
			}
		}
	}
	sort.SliceStable(cands, func(i, j int) bool {
		return cands[i].player.PlayerID < cands[j].player.PlayerID
	})
	bought := 0
	for _, c := range cands {
		p, dest := c.player, c.dest
		parent := tm.Clubs[p.ParentClubID]
		if parent == nil || parent.ClubID == dest.ClubID {
			continue
		}
		if p.UniverseWonderkid {
			continue
		}
		fee := p.LoanBuyClauseEUR
		// Re-validate bounds at execution, mirroring snapshot validation:
		// creation clamps, but OVR/age drift moves the dynamic corridor, so
		// execution checks the absolute floor/ceiling and never trusts a fee
		// with money attached.
		if fee < 300_000 || fee > 500_000_000 {
			continue
		}
		// Sporting filter evaluated as a permanent move (the loan flag
		// itself always rejects, so probe with it cleared).
		probe := *p
		probe.OnLoan = false
		if !transfers.PlayerAcceptsDestination(&probe, parent, dest) {
			continue
		}
		if dest.Finances.TransferBudget < fee || dest.Finances.Balance < fee {
			continue
		}
		// No squad change, so no new wage burden — but never let a purchase
		// land while the buyer already breaches its cap.
		if dest.WageBill() > dest.WageCap() {
			continue
		}
		dest.Finances.TransferBudget -= fee
		dest.Finances.Balance -= fee
		parent.Finances.Balance += fee
		parent.Finances.TransferBudget += fee
		if parent.Finances.TransferBudget > parent.Finances.Balance {
			parent.Finances.TransferBudget = parent.Finances.Balance
		}
		p.OnLoan = false
		p.ParentClubID = ""
		p.LoanBuyClauseEUR = 0
		p.TransferRequested = false
		p.AdjustMorale(4)
		dest.RecalculateRatings()
		parent.RecalculateRatings()
		bought++
		tm.PushInbox("transfer", dest.ShortName+" trigger "+p.FullName+" buy clause",
			fmt.Sprintf("%s pay %s to %s to make %s's loan permanent.", dest.ClubName, models.FormatCurrency(fee), parent.ClubName, p.FullName),
			tm.CurrentMatchweek, []string{parent.ClubID, dest.ClubID}, p.PlayerID, "")
	}
	if bought > 0 && tm.TransferEngine != nil {
		tm.TransferEngine.SyncAllManagerBudgets()
	}
	return bought
}

// ReturnLoansUnlocked sends every loanee back to their parent club,
// except loans whose buy clauses trigger into permanent transfers first.
func (tm *TournamentManager) ReturnLoansUnlocked() int {
	if tm == nil {
		return 0
	}
	tm.executeLoanBuyClausesUnlocked()
	returned := 0
	returnedTo := map[string]*models.Club{}
	for _, club := range tm.ClubsList {
		if club == nil {
			continue
		}
		kept := make([]*models.Player, 0, len(club.Squad))
		for _, p := range club.Squad {
			if p == nil {
				continue
			}
			if !p.OnLoan || p.ParentClubID == "" || p.ParentClubID == club.ClubID {
				kept = append(kept, p)
				continue
			}
			parent := tm.Clubs[p.ParentClubID]
			if parent == nil {
				p.OnLoan = false
				p.ParentClubID = ""
				p.LoanBuyClauseEUR = 0
				kept = append(kept, p)
				continue
			}
		p.OnLoan = false
		p.ParentClubID = ""
		p.LoanBuyClauseEUR = 0
		p.ClubID = parent.ClubID
		if !inSquad(parent, p) {
			parent.Squad = append(parent.Squad, p)
		}
		returned++
		returnedTo[parent.ClubID] = parent
		tm.PushInbox("transfer", p.FullName+" returns from loan",
			fmt.Sprintf("%s is back at %s after a season away.", p.FullName, parent.ClubName),
			tm.CurrentMatchweek, []string{parent.ClubID, club.ClubID}, p.PlayerID, "")
		}
		club.Squad = kept
	}
	// Returns are contractual and always land, but a parent that filled its
	// cap while the loanee was away is now squeezed: one aggregated notice
	// instead of a silent breach (signings stay blocked until compliant).
	squeezed := make([]string, 0)
	for id, parent := range returnedTo {
		if parent.WageBill() > parent.WageCap() {
			squeezed = append(squeezed, id)
		}
	}
	sort.Strings(squeezed)
	if len(squeezed) > 0 {
		tm.PushInbox("transfer", "Wage squeeze on returning loanees",
			fmt.Sprintf("%d club(s) sit above the wage cap after loan returns and must sell before signing again.", len(squeezed)),
			tm.CurrentMatchweek, squeezed, "", "")
	}
	for _, club := range tm.ClubsList {
		if club != nil {
			club.RecalculateRatings()
		}
	}
	tm.RefreshClubCultureUnlocked()
	return returned
}

// ArrangeLoansUnlocked sends spare young players to weaker same-league clubs.
// Canonical wonderkids stay put. Deterministic given current squads.
func (tm *TournamentManager) ArrangeLoansUnlocked() int {
	return tm.arrangeLoansUnlocked(maxOutgoingLoans, maxIncomingLoans, 99)
}

// ArrangeWinterLoansUnlocked is a shorter January wave for unused prospects.
func (tm *TournamentManager) ArrangeWinterLoansUnlocked() int {
	return tm.arrangeLoansUnlocked(maxOutgoingLoans+1, maxIncomingLoans, 6)
}

func (tm *TournamentManager) arrangeLoansUnlocked(maxOut, maxIn, maxApps int) int {
	if tm == nil {
		return 0
	}
	outgoing := map[string]int{}
	incoming := map[string]int{}
	for _, club := range tm.ClubsList {
		if club == nil {
			continue
		}
		for _, p := range club.Squad {
			if p != nil && p.OnLoan {
				outgoing[p.ParentClubID]++
				incoming[club.ClubID]++
			}
		}
	}
	moved := 0

	type candidate struct {
		player *models.Player
		parent *models.Club
	}
	cands := make([]candidate, 0)
	for _, club := range tm.ClubsList {
		if club == nil || club.OverallTeamRating < 80 || len(club.Squad) < 24 {
			continue
		}
		avg := club.SquadAvgOVR
		if avg == 0 {
			club.RecalculateRatings()
			avg = club.SquadAvgOVR
		}
		for _, p := range club.Squad {
			if p == nil || p.UniverseWonderkid || p.OnLoan || p.Age > 21 || p.IsCaptain {
				continue
			}
			if p.OriginalClubID != "" && p.OriginalClubID != club.ClubID {
				continue
			}
		// Playing time is minutes-weighted like development: a stack of
		// ten-minute cameos must not read as a busy season.
		if effectiveAppearances(p) > maxApps {
			continue
		}
			if p.SquadRole != models.RoleProspect && p.SquadRole != models.RoleSquad {
				continue
			}
			if float64(p.OVR) > avg-2 {
				continue
			}
			cands = append(cands, candidate{player: p, parent: club})
		}
	}
	sort.SliceStable(cands, func(i, j int) bool {
		if cands[i].player.OVR != cands[j].player.OVR {
			return cands[i].player.OVR < cands[j].player.OVR
		}
		return cands[i].player.PlayerID < cands[j].player.PlayerID
	})

	dests := append([]*models.Club(nil), tm.ClubsList...)
	sort.SliceStable(dests, func(i, j int) bool {
		if dests[i].OverallTeamRating != dests[j].OverallTeamRating {
			return dests[i].OverallTeamRating < dests[j].OverallTeamRating
		}
		return dests[i].ClubID < dests[j].ClubID
	})

	for _, cand := range cands {
		if outgoing[cand.parent.ClubID] >= maxOut {
			continue
		}
		// Loans move a real wage burden: skip destinations that cannot fit
		// it under their cap (unsigned players fall back to the OVR anchor).
		annual := cand.player.WageEUR * 52
		if annual <= 0 {
			annual = models.WageForOVR(cand.player.OVR) * 52
		}
		var dest *models.Club
		for _, club := range dests {
			if club == nil || club.ClubID == cand.parent.ClubID {
				continue
			}
			if club.League != cand.parent.League {
				continue
			}
			if club.OverallTeamRating >= cand.parent.OverallTeamRating {
				continue
			}
			if len(club.Squad) >= 32 || incoming[club.ClubID] >= maxIn {
				continue
			}
			if club.WageBill()+annual > club.WageCap() {
				continue
			}
			dest = club
			break
		}
		if dest == nil {
			continue
		}
		kept := make([]*models.Player, 0, len(cand.parent.Squad))
		for _, p := range cand.parent.Squad {
			if p == nil || p.PlayerID != cand.player.PlayerID {
				kept = append(kept, p)
			}
		}
		cand.parent.Squad = kept
		cand.player.OnLoan = true
		cand.player.ParentClubID = cand.parent.ClubID
		cand.player.ClubID = dest.ClubID
		cand.player.LoanBuyClauseEUR = loanBuyClauseFor(cand.player, dest)
		dest.Squad = append(dest.Squad, cand.player)
		outgoing[cand.parent.ClubID]++
		incoming[dest.ClubID]++
		moved++
		body := fmt.Sprintf("%s has been loaned from %s to %s for the season.", cand.player.FullName, cand.parent.ClubName, dest.ClubName)
		if cand.player.LoanBuyClauseEUR > 0 {
			body += fmt.Sprintf(" %s hold a %s buy clause.", dest.ShortName, models.FormatCurrency(cand.player.LoanBuyClauseEUR))
		}
		tm.PushInbox("transfer", cand.player.FullName+" joins "+dest.ShortName+" on loan",
			body,
			tm.CurrentMatchweek, []string{cand.parent.ClubID, dest.ClubID}, cand.player.PlayerID, "")
	}
	for _, club := range tm.ClubsList {
		if club != nil {
			club.RecalculateRatings()
		}
	}
	tm.RefreshClubCultureUnlocked()
	return moved
}
