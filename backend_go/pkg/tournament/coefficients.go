package tournament

import (
	"sort"

	"football_sim/pkg/models"
)

// Club coefficients are UEFA-style points earned from European campaigns and
// used to seed Swiss pots (and displayed in the hub). League-phase results
// pay per result; each knockout tie won pays by stage; the champion takes a
// small bonus. Domestic cups and leagues never pay coefficient points.
// Iteration is over configured competition order and sorted club IDs, so
// accumulation is deterministic for a given universe seed.
func (tm *TournamentManager) applyEuropeanCoefficientPointsUnlocked() {
	if tm == nil || tm.World == nil {
		return
	}
	for _, def := range europeanDefinitions {
		comp := tm.World.Competitions[def.ID]
		if comp == nil {
			continue
		}
		ids := append([]string(nil), comp.ParticipantIDs...)
		sort.Strings(ids)
		for _, pid := range ids {
			club := tm.Clubs[pid]
			if club == nil {
				continue
			}
			pts := 0
			if rec := comp.Records[pid]; rec != nil {
				pts += rec.Won*2 + rec.Drawn
			}
			for _, round := range comp.Rounds {
				for _, wid := range round.WinnerIDs {
					if wid == pid {
						pts += coefficientStageBonus(round.Stage)
					}
				}
			}
			if comp.ChampionID != "" && comp.ChampionID == pid {
				pts += 4
			}
			club.Coefficient += pts
		}
	}
}

func coefficientStageBonus(stage string) int {
	switch stage {
	case "Knockout play-off", "Play-off":
		return 1
	case "Round of 16":
		return 2
	case "Quarter-final":
		return 4
	case "Semi-final":
		return 6
	case "Final":
		return 8
	default:
		return 0
	}
}

// rankClubsForSwiss orders clubs for coefficient pots: earned coefficient
// first, then opening strength (rating, reputation) with ClubID last, so
// fresh worlds (all coefficients zero) match the legacy rating order exactly
// and pots only diverge once European points are earned.
func rankClubsForSwiss(clubs []*models.Club) []*models.Club {
	out := append([]*models.Club(nil), clubs...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Coefficient != out[j].Coefficient {
			return out[i].Coefficient > out[j].Coefficient
		}
		if out[i].OverallTeamRating != out[j].OverallTeamRating {
			return out[i].OverallTeamRating > out[j].OverallTeamRating
		}
		if out[i].Identity.Reputation != out[j].Identity.Reputation {
			return out[i].Identity.Reputation > out[j].Identity.Reputation
		}
		return out[i].ClubID < out[j].ClubID
	})
	return out
}
