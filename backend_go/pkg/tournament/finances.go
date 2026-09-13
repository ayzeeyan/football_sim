package tournament

import (
	"fmt"
	"sort"

	"football_sim/pkg/models"
)

func (tm *TournamentManager) awardSeasonPrizeMoneyUnlocked() {
	if tm == nil || tm.World == nil {
		return
	}
	for _, def := range domesticLeagueDefinitions {
		table := tm.worldLeagueStandingsUnlocked(def.ID)
		n := len(table)
		for i, club := range table {
			if club == nil {
				continue
			}
			place := i + 1
			prize := leagueFinishPrize(place, n, def.Prestige)
			creditClubPrize(club, prize)
			if place == 1 {
				tm.PushInbox("honour", club.ShortName+" collect the league prize",
					fmt.Sprintf("%s receive %s for winning %s.", club.ClubName, models.FormatCurrency(prize), def.Name),
					tm.CurrentMatchweek, []string{club.ClubID}, "", "")
			}
		}
	}
	// Domestic cups: participation + winner + runner-up (simple, deterministic).
	for _, id := range []string{"fa-cup", "efl-cup", "copa-del-rey", "dfb-pokal", "coppa-italia", "coupe-de-france"} {
		comp := tm.worldCompetitionUnlocked(id)
		if comp == nil {
			continue
		}
		for _, pid := range comp.ParticipantIDs {
			if club := tm.Clubs[pid]; club != nil {
				creditClubPrize(club, domesticCupParticipationPrize())
			}
		}
		if comp.ChampionID == "" {
			continue
		}
		if club := tm.Clubs[comp.ChampionID]; club != nil {
			prize := cupPrize(comp.Prestige)
			creditClubPrize(club, prize)
			tm.PushInbox("honour", club.ShortName+" cash the "+comp.Name+" cheque",
				fmt.Sprintf("%s receive %s in prize money for winning the %s.", club.ClubName, models.FormatCurrency(prize), comp.Name),
				tm.CurrentMatchweek, []string{club.ClubID}, "", "")
		}
		if runner := tm.worldCupRunnerUpUnlocked(comp); runner != "" {
			if club := tm.Clubs[runner]; club != nil && runner != comp.ChampionID {
				rPrize := cupPrize(comp.Prestige) * 2 / 5
				creditClubPrize(club, rPrize)
				tm.PushInbox("honour", club.ShortName+" bank the "+comp.Name+" runners-up cheque",
					fmt.Sprintf("%s receive %s as %s runners-up.", club.ClubName, models.FormatCurrency(rPrize), comp.Name),
					tm.CurrentMatchweek, []string{club.ClubID}, "", "")
			}
		}
	}
	// Europe: participation + league-phase rank + knockout-round + runner-up/winner.
	for _, id := range []string{"champions-league", "europa-league", "conference-league"} {
		tm.awardEuropeanPrizeMoneyUnlocked(id)
	}
	for _, club := range tm.ClubsList {
		if club != nil {
			club.RecalculateWageBill()
		}
	}
}

// awardEuropeanPrizeMoneyUnlocked pays a finer European table while keeping it
// simple: participation for every entrant, rank-scaled league-phase money,
// a per-round knockout payment for every tie reached, plus distinct
// runner-up and winner bonuses. Success moves Balance and half into
// TransferBudget via creditClubPrize. Deterministic: sorted iteration.
func (tm *TournamentManager) awardEuropeanPrizeMoneyUnlocked(compID string) {
	comp := tm.worldCompetitionUnlocked(compID)
	if comp == nil || len(comp.ParticipantIDs) == 0 {
		return
	}
	for _, pid := range comp.ParticipantIDs {
		if club := tm.Clubs[pid]; club != nil {
			creditEuropeanPrize(club, europeanParticipationPrize(compID))
		}
	}
	ranked := tm.worldEuropeanStandingsUnlocked(comp)
	for i, club := range ranked {
		if club == nil {
			continue
		}
		creditEuropeanPrize(club, europeanLeaguePhaseRankPrize(compID, i+1, len(ranked)))
	}
	// Knockout-round payments: every entrant of every knockout round banks the
	// stage prize (cumulative for deep runs). Finalists are covered by the
	// Final stage prize; winner/runner-up get extra bonuses below.
	for _, round := range comp.Rounds {
		prize := europeanKnockoutRoundPrize(compID, round.Stage)
		if prize <= 0 {
			continue
		}
		// EntrantIDs is authoritative; fall back to TieIDs-derived winners path
		// for legacy shapes. Sorted for determinism.
		seen := map[string]bool{}
		ordered := append([]string(nil), round.EntrantIDs...)
		if len(ordered) == 0 && len(round.TieIDs) > 0 {
			// TieIDs alone cannot recover entrants; skip to avoid double-pay.
			continue
		}
		// Sort a copy for payment order; amounts are identical per entrant.
		sorted := append([]string(nil), ordered...)
		sort.Strings(sorted)
		for _, pid := range sorted {
			if seen[pid] {
				continue
			}
			seen[pid] = true
			if club := tm.Clubs[pid]; club != nil {
				creditEuropeanPrize(club, prize)
			}
		}
	}
	if comp.ChampionID == "" {
		return
	}
	if club := tm.Clubs[comp.ChampionID]; club != nil {
		bonus := europeanWinnerBonus(compID)
		creditEuropeanPrize(club, bonus)
		// Report only the plaqued amounts (this bonus plus the Final stage
		// prize already banked above), never a hinted estimate as received.
		finalPrize := europeanKnockoutRoundPrize(compID, "Final")
		tm.PushInbox("honour", club.ShortName+" cash the "+comp.Name+" cheque",
			fmt.Sprintf("%s collect %s final prize plus a %s winner bonus for winning the %s (on top of participation, rank and round money).", club.ClubName, models.FormatCurrency(finalPrize), models.FormatCurrency(bonus), comp.Name),
			tm.CurrentMatchweek, []string{club.ClubID}, "", "")
	}
	if runner := tm.worldCupRunnerUpUnlocked(comp); runner != "" && runner != comp.ChampionID {
		if club := tm.Clubs[runner]; club != nil {
			rPrize := europeanRunnerUpBonus(compID)
			creditEuropeanPrize(club, rPrize)
			tm.PushInbox("honour", club.ShortName+" bank the "+comp.Name+" runners-up cheque",
				fmt.Sprintf("%s receive %s as %s runners-up.", club.ClubName, models.FormatCurrency(rPrize), comp.Name),
				tm.CurrentMatchweek, []string{club.ClubID}, "", "")
		}
	}
}

// worldCupRunnerUpUnlocked finds the losing finalist without manufacturing
// results: the final round's entrants minus the champion, or the final
// fixtures' non-champion side when entrants are missing.
func (tm *TournamentManager) worldCupRunnerUpUnlocked(comp *Competition) string {
	if comp == nil || comp.ChampionID == "" || len(comp.Rounds) == 0 {
		return ""
	}
	last := comp.Rounds[len(comp.Rounds)-1]
	// The final round has exactly 2 entrants; the runner-up is the other one.
	if len(last.EntrantIDs) == 2 {
		for _, eid := range last.EntrantIDs {
			if eid != "" && eid != comp.ChampionID {
				return eid
			}
		}
	}
	// Fallback: inspect final fixtures' clubs, but only for an actual final.
	// Gating on the stage (not just entrant count) keeps a semi-final loser
	// from ever being paid as runner-up when entrant lists are thin.
	if last.Stage != "Final" {
		return ""
	}
	for _, fid := range last.FixtureIDs {
		f := tm.worldFixtureUnlocked(fid)
		if f == nil {
			continue
		}
		if f.HomeID != "" && f.HomeID != comp.ChampionID {
			return f.HomeID
		}
		if f.AwayID != "" && f.AwayID != comp.ChampionID {
			return f.AwayID
		}
	}
	return ""
}

func domesticCupParticipationPrize() int64 {
	return 250_000
}

func europeanParticipationPrize(compID string) int64 {
	switch compID {
	case "champions-league":
		return 12 * models.EuroMillion
	case "europa-league":
		return 6 * models.EuroMillion
	case "conference-league":
		return 4 * models.EuroMillion
	default:
		return 2 * models.EuroMillion
	}
}

func europeanLeaguePhaseRankPrize(compID string, rank, total int) int64 {
	if rank < 1 || total < 1 {
		return 0
	}
	switch compID {
	case "champions-league":
		base := int64(500_000)
		step := int64(150_000)
		return base + int64(total-rank)*step
	case "europa-league":
		base := int64(250_000)
		step := int64(100_000)
		return base + int64(total-rank)*step
	case "conference-league":
		base := int64(150_000)
		step := int64(60_000)
		return base + int64(total-rank)*step
	default:
		return int64(100_000)
	}
}

func europeanKnockoutRoundPrize(compID, stage string) int64 {
	m := models.EuroMillion
	switch stage {
	case "Knockout play-off", "Play-off":
		switch compID {
		case "champions-league":
			return 800_000
		case "europa-league":
			return 400_000
		default:
			return 200_000
		}
	case "Round of 16":
		switch compID {
		case "champions-league":
			return 15 * m / 10
		case "europa-league":
			return 800_000
		default:
			return 400_000
		}
	case "Quarter-final":
		switch compID {
		case "champions-league":
			return 25 * m / 10
		case "europa-league":
			return 12 * m / 10
		default:
			return 600_000
		}
	case "Semi-final":
		switch compID {
		case "champions-league":
			return 4 * m
		case "europa-league":
			return 2 * m
		default:
			return 1 * m
		}
	case "Final":
		switch compID {
		case "champions-league":
			return 5 * m
		case "europa-league":
			return 25 * m / 10
		default:
			return 12 * m / 10
		}
	default:
		return 0
	}
}

func europeanWinnerBonus(compID string) int64 {
	switch compID {
	case "champions-league":
		return 5 * models.EuroMillion
	case "europa-league":
		return 25 * models.EuroMillion / 10
	default:
		return 12 * models.EuroMillion / 10
	}
}

func europeanRunnerUpBonus(compID string) int64 {
	switch compID {
	case "champions-league":
		return 3 * models.EuroMillion
	case "europa-league":
		return 15 * models.EuroMillion / 10
	default:
		return 700_000
	}
}

func leagueFinishPrize(place, n, prestige int) int64 {
	if place < 1 || n < 1 {
		return 0
	}
	top := int64(8+prestige/4) * models.EuroMillion
	share := top * int64(n-place+1) / int64(n)
	if share < models.EuroMillion {
		share = models.EuroMillion
	}
	return share
}

func cupPrize(prestige int) int64 {
	return int64(5+prestige/5) * models.EuroMillion
}

func creditClubPrize(club *models.Club, prize int64) {
	if club == nil || prize <= 0 {
		return
	}
	club.Finances.Balance += prize
	reinvest := prize / 2
	club.Finances.TransferBudget += reinvest
	if club.Finances.TransferBudget > club.Finances.Balance {
		club.Finances.TransferBudget = club.Finances.Balance
	}
}

// creditEuropeanPrize banks European prize money exactly like generic prize
// money and additionally tracks it in the separate European revenue ledger.
// The ledger is zeroed before each season's banking, so it always holds the
// last completed season's intake (replace, never accumulate).
func creditEuropeanPrize(club *models.Club, prize int64) {
	if club == nil || prize <= 0 {
		return
	}
	creditClubPrize(club, prize)
	club.Finances.EuropeanRevenue += prize
}

// resetEuropeanRevenueLedgerUnlocked opens a fresh European revenue ledger
// for the season ahead. Called before prizes are banked at rollover.
func (tm *TournamentManager) resetEuropeanRevenueLedgerUnlocked() {
	if tm == nil {
		return
	}
	for _, club := range tm.ClubsList {
		if club != nil {
			club.Finances.EuropeanRevenue = 0
		}
	}
}
