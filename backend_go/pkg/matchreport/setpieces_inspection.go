package matchreport

import (
	"fmt"

	"football_sim/pkg/growth"
	"football_sim/pkg/models"
)

// SetPiecePick is one inspected set-piece decision: who the engine would
// pick from a kickoff XI and why. Confidence is the candidate's weight share
// for probabilistic picks (aerial target, corner taker) and 1.0 for
// deterministic picks (penalties, free kicks).
type SetPiecePick struct {
	PlayerID   string  `json:"player_id"`
	FullName   string  `json:"full_name"`
	Position   string  `json:"position"`
	OVR        int     `json:"ovr"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason"`
}

// SetPieceInspection is the full read-only set-piece briefing for one XI.
type SetPieceInspection struct {
	PenaltyTaker  *SetPiecePick `json:"penalty_taker"`
	FreeKickTaker *SetPiecePick `json:"free_kick_taker"`
	AerialTarget  *SetPiecePick `json:"aerial_target"`
	CornerTaker   *SetPiecePick `json:"corner_taker"`
}

// InspectSetPieces reports the engine's set-piece choices for a kickoff XI
// without consuming randomness: probabilistic picks (aerial target, corner
// taker) surface the most likely candidate with its weight share instead of
// drawing from the stream. This is inspection data only — the match engine
// keeps its own RNG-driven picks during play.
func InspectSetPieces(xi []*models.Player, ge *growth.GrowthEngine) *SetPieceInspection {
	out := &SetPieceInspection{}
	if len(xi) == 0 {
		return out
	}

	// Penalties: deterministic strongest composite score.
	if p := DesignatedPenaltyTaker(xi, ge); p != nil {
		out.PenaltyTaker = &SetPiecePick{
			PlayerID: p.PlayerID, FullName: p.FullName, Position: p.Position, OVR: p.OVR,
			Confidence: 1.0,
			Reason:     fmt.Sprintf("highest penalty score (%d): shooting weighted by category, composure %d", penaltyScore(p, ge), p.Composure),
		}
	}

	// Free kicks: deterministic best dead-ball specialist (shooting >= 85).
	if p := PickFreeKickTaker(xi, ge); p != nil {
		sh := freeKickShooting(p, ge)
		out.FreeKickTaker = &SetPiecePick{
			PlayerID: p.PlayerID, FullName: p.FullName, Position: p.Position, OVR: p.OVR,
			Confidence: 1.0,
			Reason:     fmt.Sprintf("best dead-ball specialist (shooting %d, threshold 85)", sh),
		}
	}

	// Aerial target: most likely corner header target by squared aerial score.
	target, targetConfidence := mostLikelyAerialTarget(xi, ge)
	if target != nil {
		out.AerialTarget = &SetPiecePick{
			PlayerID: target.PlayerID, FullName: target.FullName, Position: target.Position, OVR: target.OVR,
			Confidence: targetConfidence,
			Reason:     fmt.Sprintf("highest squared aerial score (%.0f) among DEF/FWD/MID candidates", aerialScore(target, ge)),
		}
	}

	// Corner taker: most likely delivery given the most likely target.
	if target != nil {
		if taker, confidence := mostLikelyCornerTaker(xi, target); taker != nil {
			out.CornerTaker = &SetPiecePick{
				PlayerID: taker.PlayerID, FullName: taker.FullName, Position: taker.Position, OVR: taker.OVR,
				Confidence: confidence,
				Reason:     "highest delivery weighting (midfielders favoured, never the aerial target)",
			}
		}
	}
	return out
}

// freeKickShooting mirrors PickFreeKickTaker's rating source for the reason
// string: tracked attributes when present, category-based OVR fallback
// otherwise.
func freeKickShooting(p *models.Player, ge *growth.GrowthEngine) int {
	if ge != nil {
		if attrs := ge.Attributes[p.PlayerID]; attrs != nil {
			return attrs.Shooting
		}
	}
	switch p.Category {
	case "FWD":
		return p.OVR
	case "MID":
		return p.OVR - 5
	default:
		return p.OVR - 20
	}
}

// mostLikelyAerialTarget returns the mode of PickAerialTarget's weighted
// distribution plus its weight share, without consuming randomness.
func mostLikelyAerialTarget(xi []*models.Player, ge *growth.GrowthEngine) (*models.Player, float64) {
	cands := make([]*models.Player, 0, len(xi))
	for _, p := range xi {
		if p.Category == "DEF" || p.Category == "FWD" || p.Category == "MID" {
			cands = append(cands, p)
		}
	}
	if len(cands) == 0 {
		cands = xi
	}
	if len(cands) == 0 {
		return nil, 0
	}
	best, bestWeight, total := cands[0], 0.0, 0.0
	for _, p := range cands {
		w := maxFloat(1.0, aerialScore(p, ge)*aerialScore(p, ge))
		total += w
		if w > bestWeight {
			best, bestWeight = p, w
		}
	}
	if total <= 0 {
		return best, 0
	}
	return best, bestWeight / total
}

// mostLikelyCornerTaker returns the mode of PickCornerTaker's weighted
// distribution given a fixed aerial target, plus its weight share.
func mostLikelyCornerTaker(xi []*models.Player, target *models.Player) (*models.Player, float64) {
	cands := make([]*models.Player, 0, len(xi))
	for _, p := range xi {
		if p != target && (p.Category == "MID" || p.Category == "FWD" || p.Category == "DEF") {
			cands = append(cands, p)
		}
	}
	if len(cands) == 0 {
		return nil, 0
	}
	best, bestWeight, total := cands[0], 0.0, 0.0
	for _, p := range cands {
		mult := 1.2
		if p.Category == "MID" {
			mult = 2.0
		}
		w := float64(p.EffectiveOVR()+p.FormModifier()) / 75.0 * mult
		total += w
		if w > bestWeight {
			best, bestWeight = p, w
		}
	}
	if total <= 0 {
		return best, 0
	}
	return best, bestWeight / total
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
