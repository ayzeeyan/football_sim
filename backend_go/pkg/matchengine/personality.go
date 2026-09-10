package matchengine

import (
	"fmt"
	"strings"

	"football_sim/pkg/models"
)

// Personality off the ball (Feature 5).
//
// Three situations, each producing one commentary line tied to the prodigy
// plus a tiny nudge to the existing composure hook (which already feeds the
// live conversion math). No new archetype table, no xG rewrite.

// mustWinMoment reports a late trailing state for one side: the plain
// must-score situation, resolved from the live scoreboard.
func (e *LiveMatchEngine) mustWinMoment(side string, minute int) bool {
	if minute < 70 {
		return false
	}
	if side == "home" {
		return e.HomeScore < e.AwayScore
	}
	return e.AwayScore < e.HomeScore
}

// isSuperCupFinal reports the one-night Super Cup final for bottle lines.
func (e *LiveMatchEngine) isSuperCupFinal() bool {
	return strings.EqualFold(strings.TrimSpace(e.Competition), "super-cup") &&
		strings.EqualFold(strings.TrimSpace(e.Stage), "final")
}

// personalityMissLine resolves an off-target miss by a prodigy into a
// personality line, or "" when no situation applies. The tiny hook dips the
// shooter's existing composure by 1 (floored), which the live conversion
// math already reads.
func (e *LiveMatchEngine) personalityMissLine(shooter *models.Player, side string, minute int) string {
	if shooter == nil || !shooter.UniverseWonderkid {
		return ""
	}
	if e.isSuperCupFinal() {
		nudgeComposure(shooter, -1)
		return fmt.Sprintf("%s snatches at it on final night and drags it wide — the occasion gets to the kid.", shooter.FullName)
	}
	if shooter.Personality == "flamboyant_star" && e.mustWinMoment(side, minute) {
		nudgeComposure(shooter, -1)
		return fmt.Sprintf("%s tries the audacious flick with his side chasing the game — over the bar! Too casual when it matters.", shooter.FullName)
	}
	return ""
}

// dedicatedTenManLine rallies a dedicated prodigy still on the pitch after
// his side goes down to ten. Returns "" when no dedicated kid is out there.
// The tiny hook steadies the kid's existing composure by 1 (capped).
func (e *LiveMatchEngine) dedicatedTenManLine(sentOffSide string, minute int) string {
	_ = minute
	starters := e.HomeStarters
	if sentOffSide != "home" {
		starters = e.AwayStarters
	}
	for _, p := range e.OnPitch(starters) {
		if p != nil && p.UniverseWonderkid && p.Personality == "dedicated_pro" {
			nudgeComposure(p, 1)
			return fmt.Sprintf("Down to ten, %s grabs the game by the scruff — the dedicated kid demands one more run from everyone.", p.FullName)
		}
	}
	return ""
}

// nudgeComposure moves the existing composure hook by delta, bounded.
func nudgeComposure(p *models.Player, delta int) {
	if p == nil {
		return
	}
	v := p.Composure + delta
	if v < 40 {
		v = 40
	}
	if v > 99 {
		v = 99
	}
	p.Composure = v
}
