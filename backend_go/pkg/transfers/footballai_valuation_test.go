package transfers

import (
	"testing"

	"football_sim/pkg/footballai"
	"football_sim/pkg/models"
)

func valuationTestModel(t *testing.T) *footballai.Model {
	t.Helper()
	net, err := footballai.NewNetwork(footballai.DefaultConfig(), 13)
	if err != nil {
		t.Fatal(err)
	}
	return footballai.BuildModel(net, "test-hash")
}

// TestValuationPremiumGating proves the market stays deterministic without an
// enabled flag (premium 1.0) and bounded inside [0.9, 1.15] when enabled.
func TestValuationPremiumGating(t *testing.T) {
	player := &models.Player{
		PlayerID: "P1", OVR: 82, Age: 24, Position: "CM",
		ContractYears: 3, Fitness: 88, Sharpness: 78, Morale: 75,
	}
	buyer := &models.Club{ClubID: "BUY", OverallTeamRating: 80}
	buyer.Finances.TransferBudget = 150_000_000

	te := &TransferEngine{}
	if got := te.valuationPremium(player, buyer); got != 1 {
		t.Fatalf("nil brain premium %g, want 1", got)
	}

	te = &TransferEngine{AIBrain: footballai.NewBrainWithModel(valuationTestModel(t), footballai.AIConfig{}, nil)}
	if got := te.valuationPremium(player, buyer); got != 1 {
		t.Fatalf("disabled brain premium %g, want 1", got)
	}

	te = &TransferEngine{AIBrain: footballai.NewBrainWithModel(valuationTestModel(t), footballai.AIConfig{UseValuationModel: true}, nil)}
	got := te.valuationPremium(player, buyer)
	if got < 0.9-1e-6 || got > 1.15+1e-6 {
		t.Fatalf("premium %g outside corridor [0.9, 1.15]", got)
	}
	// Deterministic across calls.
	for i := 0; i < 10; i++ {
		if again := te.valuationPremium(player, buyer); again != got {
			t.Fatalf("premium not deterministic: %g vs %g", again, got)
		}
	}
	// Nil player fails open to a neutral premium.
	if got := te.valuationPremium(nil, buyer); got != 1 {
		t.Fatalf("nil player premium %g, want 1", got)
	}
}
