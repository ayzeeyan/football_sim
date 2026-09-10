package transfers

import (
	"math/rand"
	"testing"

	"football_sim/pkg/managers"
	"football_sim/pkg/models"
)

func createTransferTestUniverse() (*TransferEngine, *models.Club, *models.Club) {
	barca := &models.Club{
		ClubID:            "LAL-BAR",
		ClubName:          "FC Barcelona",
		ShortName:         "BAR",
		OverallTeamRating: 85,
		Squad: []*models.Player{
			{PlayerID: "P1", FullName: "Barca Star", OVR: 84, Age: 24, ClubID: "LAL-BAR"},
			{PlayerID: "WK1", FullName: "Venjamin Valerio", OVR: 78, Age: 14, UniverseWonderkid: true, ClubID: "LAL-BAR"},
		},
	}

	madrid := &models.Club{
		ClubID:            "LAL-RMA",
		ClubName:          "Real Madrid",
		ShortName:         "RMA",
		OverallTeamRating: 86,
		Squad: []*models.Player{
			{PlayerID: "P2", FullName: "Madrid Veteran", OVR: 86, Age: 29, ClubID: "LAL-RMA"},
		},
	}

	clubs := []*models.Club{barca, madrid}
	mgrs := managers.BuildManagers(clubs)
	te := NewTransferEngine(clubs, mgrs, 42)
	return te, barca, madrid
}

func TestTransferEngine_TriggerSpecificBidIsIdempotent(t *testing.T) {
	te, barca, madrid := createTransferTestUniverse()
	p := barca.Squad[0]
	p.MarketValueEUR = 40_000_000

	first := te.TriggerSpecificBid(madrid.ClubID, barca.ClubID, p.PlayerID)
	if first == nil || first.Buyer.ClubID != madrid.ClubID || first.Seller.ClubID != barca.ClubID {
		t.Fatalf("expected a Super League bid, got %+v", first)
	}
	if first.StageName != "INQUIRY" || first.CurrentBid <= 0 {
		t.Fatalf("bid not initialized: %+v", first)
	}
	second := te.TriggerSpecificBid(madrid.ClubID, barca.ClubID, p.PlayerID)
	if second != first {
		t.Fatal("second bid should return the live negotiation")
	}
	if len(te.ActiveNegotiations) != 1 {
		t.Fatalf("duplicate talks: %d", len(te.ActiveNegotiations))
	}
}

func TestTransferEngine_WindowStatus(t *testing.T) {
	te, _, _ := createTransferTestUniverse()

	te.CurrentMatchweek = 1
	if !te.IsWindowOpen() {
		t.Errorf("MW 1 should be open window")
	}

	te.CurrentMatchweek = 5
	if te.IsWindowOpen() {
		t.Errorf("MW 5 should be closed window")
	}

	te.CurrentMatchweek = 22
	if !te.IsWindowOpen() {
		t.Errorf("MW 22 (winter window) should be open window")
	}
}

func TestTransferEngine_NegotiationAndExecution(t *testing.T) {
	te, barca, madrid := createTransferTestUniverse()
	te.RNG = rand.New(rand.NewSource(12345))

	p := barca.Squad[0]
	neg := &TransferNegotiation{
		NegotiationID:    "TEST_NEG_1",
		Player:           p,
		Buyer:            madrid,
		Seller:           barca,
		CurrentBid:       50_000_000,
		AskingPrice:      50_000_000,
		CreatedMatchweek: 1,
		StageIndex:       1,
		StageName:        "INQUIRY",
		ProgressPct:      20,
	}
	te.ActiveNegotiations = append(te.ActiveNegotiations, neg)

	// Advance through stages 2, 3, 4, 5
	for step := 2; step <= 5; step++ {
		te.UpdateDailyMarket()
	}

	// Deal should be completed
	if len(te.CompletedTransfers) == 0 {
		t.Fatalf("expected completed transfer after 4 updates")
	}

	done := te.CompletedTransfers[0]
	if done.PlayerID != p.PlayerID {
		t.Errorf("expected player %s to be transferred, got %s", p.PlayerID, done.PlayerID)
	}

	// Check squad mutation
	foundInBuyer := false
	for _, pl := range madrid.Squad {
		if pl.PlayerID == p.PlayerID {
			foundInBuyer = true
			break
		}
	}
	if !foundInBuyer {
		t.Errorf("expected player to be present in buyer squad")
	}

	foundInSeller := false
	for _, pl := range barca.Squad {
		if pl.PlayerID == p.PlayerID {
			foundInSeller = true
			break
		}
	}
	if foundInSeller {
		t.Errorf("expected player to be removed from seller squad")
	}

	// Check records
	records := te.GetTransferRecords()
	if len(records.TopSignings) != 1 {
		t.Errorf("expected 1 top signing, got %d", len(records.TopSignings))
	}
	if records.NetSpend["LAL-RMA"].Spent != done.FeeEUR {
		t.Errorf("expected Madrid net spend %d, got %d", done.FeeEUR, records.NetSpend["LAL-RMA"].Spent)
	}
}
