package tournament

import (
	"fmt"
	"sort"
	"strings"
)

// Domestic promotion and relegation for the closed 96-club world.
//
// The dataset contains only the five top flights — there is no second
// division to promote from — so the pyramid is closed: the five leagues
// form a prestige ladder and each adjacent pair exchanges its boundary
// clubs. The bottom three of the stronger league are relegated to the
// weaker league; the top three of the weaker league are promoted into
// the stronger one. The base of the ladder (the weakest league) has no
// lower tier: its relegation places are a survival battle whose stakes
// are reputation and finances rather than demotion.
//
// The exchange is a pure function of the final tables: no randomness,
// no clock, deterministic ordering. League sizes never change and the
// 96 dataset clubs (and their crest mappings) are preserved exactly.

// relegationSwapSize is how many clubs cross each boundary. It matches
// the frontend qualification band ("last three go down").
const relegationSwapSize = 3

// RelegationMove records one club crossing a league boundary at the
// season transition.
type RelegationMove struct {
	ClubID     string `json:"club_id"`
	ClubName   string `json:"club_name"`
	FromLeague string `json:"from_league"`
	ToLeague   string `json:"to_league"`
	Direction  string `json:"direction"` // "relegated" | "promoted"
}

// domesticRelegationChain returns the five league definitions ordered
// strongest-first by prestige; ties keep definition order so the chain
// is stable across runs.
func domesticRelegationChain() []CompetitionDefinition {
	chain := append([]CompetitionDefinition(nil), domesticLeagueDefinitions...)
	sort.SliceStable(chain, func(i, j int) bool { return chain[i].Prestige > chain[j].Prestige })
	return chain
}

// PlanDomesticPromotionRelegation computes the boundary swaps from the
// current (final) league tables without mutating anything. Exposed for
// tests and for the season-preview surface.
func (tm *TournamentManager) PlanDomesticPromotionRelegation() []RelegationMove {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.planDomesticPromotionRelegationUnlocked()
}

func (tm *TournamentManager) planDomesticPromotionRelegationUnlocked() []RelegationMove {
	if tm.World == nil {
		return nil
	}
	chain := domesticRelegationChain()
	var moves []RelegationMove
	for i := 0; i+1 < len(chain); i++ {
		stronger, weaker := chain[i], chain[i+1]
		down := tm.worldLeagueStandingsUnlocked(stronger.ID)
		up := tm.worldLeagueStandingsUnlocked(weaker.ID)
		if len(down) < 2*relegationSwapSize || len(up) < 2*relegationSwapSize {
			// Degenerate league sizes never exchange boundary clubs.
			continue
		}
		for _, club := range down[len(down)-relegationSwapSize:] {
			moves = append(moves, RelegationMove{
				ClubID: club.ClubID, ClubName: club.ClubName,
				FromLeague: stronger.League, ToLeague: weaker.League,
				Direction: "relegated",
			})
		}
		for _, club := range up[:relegationSwapSize] {
			moves = append(moves, RelegationMove{
				ClubID: club.ClubID, ClubName: club.ClubName,
				FromLeague: weaker.League, ToLeague: stronger.League,
				Direction: "promoted",
			})
		}
	}
	sort.SliceStable(moves, func(i, j int) bool {
		if moves[i].Direction != moves[j].Direction {
			return moves[i].Direction == "relegated"
		}
		if moves[i].FromLeague != moves[j].FromLeague {
			return moves[i].FromLeague < moves[j].FromLeague
		}
		return moves[i].ClubID < moves[j].ClubID
	})
	return moves
}

// applyPlannedRelegationUnlocked executes pre-computed swaps by rewriting
// each moving club's League field. Planning happens while the final tables
// are intact; application happens after every table read in the transition
// and before the calendar rebuild re-seeds league and cup participants.
// The caller must hold tm.mu.
func (tm *TournamentManager) applyPlannedRelegationUnlocked(moves []RelegationMove) {
	for _, move := range moves {
		if club := tm.Clubs[move.ClubID]; club != nil {
			club.League = move.ToLeague
		}
	}
}

// pushRelegationNewsUnlocked publishes the boundary swaps to the inbox.
// The base league's survival battle is stated explicitly so the viewer
// never infers a demotion that cannot happen.
func (tm *TournamentManager) pushRelegationNewsUnlocked(moves []RelegationMove, matchweek int) {
	if len(moves) == 0 {
		return
	}
	var lines []string
	for _, move := range moves {
		verb := "promoted"
		if move.Direction == "relegated" {
			verb = "relegated"
		}
		lines = append(lines, fmt.Sprintf("%s %s from %s to %s.", move.ClubName, verb, move.FromLeague, move.ToLeague))
	}
	chain := domesticRelegationChain()
	base := chain[len(chain)-1]
	body := strings.Join(lines, " ") + fmt.Sprintf(" %s's bottom three survive: the closed pyramid has no lower tier.", base.Name)
	clubIDs := make([]string, 0, len(moves))
	for _, move := range moves {
		clubIDs = append(clubIDs, move.ClubID)
	}
	tm.PushInbox(MsgCategorySystem, "Promotion and relegation confirmed", body, matchweek, clubIDs, "", "")
}
