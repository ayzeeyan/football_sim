package tournament

import (
	"fmt"
	"strings"

	"football_sim/pkg/models"
)

// Domestic relegation stakes for the closed 96-club world.
//
// The dataset contains only the five top flights — there is no second
// division — and every league is national: a German club belongs in the
// Bundesliga for the life of the world. No club ever changes leagues. The
// relegation places are instead a survival battle with real stakes: at the
// season transition the bottom three of every league lose reputation and
// pay a financial penalty. The stakes are a pure function of the final
// tables: no randomness, no clock, deterministic ordering. League sizes
// never change and the 96 dataset clubs (and their crest mappings) are
// preserved exactly.

// relegationStakesSize is how many clubs per league pay the survival
// stakes. It matches the frontend qualification band ("last three go down").
const relegationStakesSize = 3

// relegationReputationPenalty is the reputation hit each relegated club
// takes on top of the annual table-driven reputation update.
const relegationReputationPenalty = 3

// relegationFinancePenaltyPercent of the available balance is lost to the
// relegation stakes; the deduction can never push a balance negative and
// the transfer budget is clamped back under the reduced balance.
const relegationFinancePenaltyPercent = 10

// RelegationMove records one club crossing a league boundary at the
// season transition. The closed country-pure pyramid never produces moves;
// the type and the persisted ledger are retained so older saves load and
// the wire contract stays stable if second divisions are ever added.
type RelegationMove struct {
	ClubID     string `json:"club_id"`
	ClubName   string `json:"club_name"`
	FromLeague string `json:"from_league"`
	ToLeague   string `json:"to_league"`
	Direction  string `json:"direction"` // "relegated" | "promoted"
}

// RelegationStake records one club paying the survival-battle price.
type RelegationStake struct {
	ClubID   string `json:"club_id"`
	ClubName string `json:"club_name"`
	League   string `json:"league"`
}

// PlanDomesticRelegationStakes computes the survival-battle stakes from the
// current (final) league tables without mutating anything. Exposed for
// tests and for the season-preview surface.
func (tm *TournamentManager) PlanDomesticRelegationStakes() []RelegationStake {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.planDomesticRelegationStakesUnlocked()
}

func (tm *TournamentManager) planDomesticRelegationStakesUnlocked() []RelegationStake {
	if tm.World == nil {
		return nil
	}
	var stakes []RelegationStake
	for _, def := range domesticLeagueDefinitions {
		table := tm.worldLeagueStandingsUnlocked(def.ID)
		if len(table) < relegationStakesSize {
			// Degenerate league sizes never pay stakes.
			continue
		}
		for _, club := range table[len(table)-relegationStakesSize:] {
			stakes = append(stakes, RelegationStake{
				ClubID: club.ClubID, ClubName: club.ClubName, League: def.League,
			})
		}
	}
	return stakes
}

// applyRelegationStakesUnlocked charges the survival-battle price: a
// reputation hit and a balance deduction for each staked club. Penalties
// are computed from live finances at application time, so the deduction
// can never push a balance negative and the budget invariant
// (budget <= balance) is preserved by clamping. The caller must hold tm.mu.
func (tm *TournamentManager) applyRelegationStakesUnlocked(stakes []RelegationStake) {
	for _, stake := range stakes {
		club := tm.Clubs[stake.ClubID]
		if club == nil {
			continue
		}
		club.Identity.Reputation = models.ClampClubRating(club.Identity.Reputation - relegationReputationPenalty)
		penalty := club.Finances.Balance / 100 * relegationFinancePenaltyPercent
		if penalty <= 0 {
			continue
		}
		if penalty > club.Finances.Balance {
			penalty = club.Finances.Balance
		}
		club.Finances.Balance -= penalty
		if club.Finances.TransferBudget > club.Finances.Balance {
			club.Finances.TransferBudget = club.Finances.Balance
		}
	}
}

// pushRelegationStakesNewsUnlocked publishes the survival-battle outcome to
// the inbox. The copy states explicitly that no club changes league: the
// closed pyramid has no second division and every league is national.
func (tm *TournamentManager) pushRelegationStakesNewsUnlocked(stakes []RelegationStake, matchweek int) {
	if len(stakes) == 0 {
		return
	}
	byLeague := map[string][]string{}
	var leagues []string
	for _, stake := range stakes {
		if _, ok := byLeague[stake.League]; !ok {
			leagues = append(leagues, stake.League)
		}
		byLeague[stake.League] = append(byLeague[stake.League], stake.ClubName)
	}
	var lines []string
	for _, league := range leagues {
		lines = append(lines, fmt.Sprintf("%s: %s pay the survival price.", league, strings.Join(byLeague[league], ", ")))
	}
	body := strings.Join(lines, " ") + " No club changes league: the closed pyramid has no second division, so the bottom three lose reputation and finances instead of their place."
	clubIDs := make([]string, 0, len(stakes))
	for _, stake := range stakes {
		clubIDs = append(clubIDs, stake.ClubID)
	}
	tm.PushInbox(MsgCategorySystem, "Survival battle settled: relegation stakes paid", body, matchweek, clubIDs, "", "")
}
