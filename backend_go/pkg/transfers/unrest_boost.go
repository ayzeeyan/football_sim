package transfers

import (
	"football_sim/pkg/models"
)

// unrestTargetBoost raises a transfer target's score when the player is
// starved of minutes: unhappy players are more willing to move, so the AI
// market chases them harder. Derived from the same pure assessment the
// weekly tick uses (models.AssessUnrest) — no state, no persistence.
func unrestTargetBoost(seller *models.Club, p *models.Player, ranks map[string]int) int {
	if seller == nil || p == nil {
		return 0
	}
	assessment := models.AssessUnrest(seller, p, ranks[p.PlayerID])
	if assessment == nil {
		return 0
	}
	switch assessment.Level {
	case models.UnrestWantsToLeave:
		return 30
	case models.UnrestUnsettled:
		return 10
	default:
		return 0
	}
}
