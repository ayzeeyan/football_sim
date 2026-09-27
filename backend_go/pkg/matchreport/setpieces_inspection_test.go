package matchreport

import (
	"testing"

	"football_sim/pkg/growth"
	"football_sim/pkg/models"
)

func xiPlayer(id, pos, category string, ovr int) *models.Player {
	return &models.Player{PlayerID: id, FullName: id, Position: pos, Category: category, OVR: ovr}
}

// The inspection must agree with the engine's deterministic picks and stay
// stable across calls (no RNG consumption).
func TestInspectSetPiecesMatchesEnginePicks(t *testing.T) {
	ge := growth.NewGrowthEngine(7)
	xi := []*models.Player{
		xiPlayer("P-GK", "GK", "GK", 80),
		xiPlayer("P-DEF-TALL", "CB", "DEF", 78),
		xiPlayer("P-DEF", "LB", "DEF", 74),
		xiPlayer("P-MID-PLAY", "CM", "MID", 88),
		xiPlayer("P-MID", "CM", "MID", 76),
		xiPlayer("P-FWD-STAR", "ST", "FWD", 90),
		xiPlayer("P-FWD", "RW", "FWD", 79),
	}

	first := InspectSetPieces(xi, ge)
	second := InspectSetPieces(xi, ge)
	if first == nil || second == nil {
		t.Fatal("inspection must return a briefing")
	}

	// Deterministic picks must match the engine exactly.
	if want := DesignatedPenaltyTaker(xi, ge); first.PenaltyTaker == nil || want == nil || first.PenaltyTaker.PlayerID != want.PlayerID {
		t.Fatalf("penalty taker inspection %v must match engine pick %v", first.PenaltyTaker, want)
	}
	if want := PickFreeKickTaker(xi, ge); (want == nil) != (first.FreeKickTaker == nil) || (want != nil && first.FreeKickTaker.PlayerID != want.PlayerID) {
		t.Fatalf("free-kick taker inspection %v must match engine pick %v", first.FreeKickTaker, want)
	}

	// Probabilistic picks surface the most likely candidate with a share.
	if first.AerialTarget == nil || first.AerialTarget.Confidence <= 0 || first.AerialTarget.Confidence > 1 {
		t.Fatalf("aerial target must carry a weight share: %+v", first.AerialTarget)
	}
	if first.CornerTaker == nil || first.CornerTaker.Confidence <= 0 || first.CornerTaker.Confidence > 1 {
		t.Fatalf("corner taker must carry a weight share: %+v", first.CornerTaker)
	}
	// The corner taker is never the aerial target.
	if first.CornerTaker.PlayerID == first.AerialTarget.PlayerID {
		t.Fatal("corner taker must differ from the aerial target")
	}
	// Goalkeepers never take outfield set pieces.
	for _, pick := range []*SetPiecePick{first.AerialTarget, first.CornerTaker, first.PenaltyTaker, first.FreeKickTaker} {
		if pick != nil && pick.PlayerID == "P-GK" {
			t.Fatalf("goalkeeper picked for an outfield set piece: %+v", pick)
		}
	}
	// Every pick carries a reason.
	for _, pick := range []*SetPiecePick{first.PenaltyTaker, first.FreeKickTaker, first.AerialTarget, first.CornerTaker} {
		if pick != nil && pick.Reason == "" {
			t.Fatalf("pick %s missing reason", pick.PlayerID)
		}
	}

	// Stability: identical inputs, identical briefing.
	if first.AerialTarget.PlayerID != second.AerialTarget.PlayerID ||
		first.CornerTaker.PlayerID != second.CornerTaker.PlayerID ||
		first.PenaltyTaker.PlayerID != second.PenaltyTaker.PlayerID {
		t.Fatal("inspection must be stable across calls (no RNG consumption)")
	}
}

func TestInspectSetPiecesEmptyXI(t *testing.T) {
	ins := InspectSetPieces(nil, nil)
	if ins == nil || ins.PenaltyTaker != nil || ins.AerialTarget != nil {
		t.Fatalf("empty XI must yield an empty briefing: %+v", ins)
	}
}
