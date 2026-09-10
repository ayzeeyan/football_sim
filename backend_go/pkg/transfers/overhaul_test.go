package transfers

import (
	"strings"
	"testing"

	"football_sim/pkg/managers"
	"football_sim/pkg/models"
)

func createOverhaulTestUniverse() (*TransferEngine, *models.Club, *models.Club, *models.Club) {
	barca := &models.Club{
		ClubID:            "LAL-BAR",
		ClubName:          "FC Barcelona",
		ShortName:         "BAR",
		OverallTeamRating: 88,
		Squad: []*models.Player{
			{PlayerID: "P1", FullName: "Pedri", OVR: 86, Age: 23, MarketValueEUR: 80_000_000, ClubID: "LAL-BAR", OriginalClubID: "LAL-BAR"},
			{PlayerID: "WK_BAR_1", FullName: "Venjamin Valerio", OVR: 75, Age: 14, UniverseWonderkid: true, MarketValueEUR: 30_000_000, ClubID: "LAL-BAR", OriginalClubID: "LAL-BAR"},
		},
	}

	madrid := &models.Club{
		ClubID:            "LAL-RMA",
		ClubName:          "Real Madrid",
		ShortName:         "RMA",
		OverallTeamRating: 90,
		Squad: []*models.Player{
			{PlayerID: "P2", FullName: "Bellingham", OVR: 90, Age: 23, MarketValueEUR: 120_000_000, ClubID: "LAL-RMA", OriginalClubID: "LAL-RMA"},
		},
	}

	nonLeague := &models.Club{
		ClubID:            "LOW-TIER",
		ClubName:          "Low Tier FC",
		ShortName:         "LOW",
		OverallTeamRating: 65,
		Squad: []*models.Player{
			{PlayerID: "P3", FullName: "Local Guy", OVR: 68, Age: 25, MarketValueEUR: 2_000_000, ClubID: "LOW-TIER", OriginalClubID: "LOW-TIER"},
		},
	}

	clubs := []*models.Club{barca, madrid, nonLeague}
	mgrs := managers.BuildManagers(clubs)
	te := NewTransferEngine(clubs, mgrs, 42)
	return te, barca, madrid, nonLeague
}

func TestWarchestInitialization(t *testing.T) {
	te, barca, madrid, nonLeague := createOverhaulTestUniverse()

	bMgr := te.Managers[barca.ClubID]
	mMgr := te.Managers[madrid.ClubID]
	lMgr := te.Managers[nonLeague.ClubID]

	// All budgets must fall between €50M and €250M
	if bMgr.BudgetEur < 50_000_000 || bMgr.BudgetEur > 250_000_000 {
		t.Errorf("Barca budget out of [50M, 250M] bounds: %d", bMgr.BudgetEur)
	}
	if mMgr.BudgetEur < 50_000_000 || mMgr.BudgetEur > 250_000_000 {
		t.Errorf("Madrid budget out of [50M, 250M] bounds: %d", mMgr.BudgetEur)
	}
	if lMgr.BudgetEur < 50_000_000 || lMgr.BudgetEur > 250_000_000 {
		t.Errorf("Low Tier budget out of [50M, 250M] bounds: %d", lMgr.BudgetEur)
	}

	// Madrid (rating 90) should have higher or equal warchest than Barca (rating 88)
	if mMgr.BudgetEur < bMgr.BudgetEur {
		t.Errorf("Expected Madrid warchest (%d) >= Barca (%d)", mMgr.BudgetEur, bMgr.BudgetEur)
	}
}

func TestSingleTransferLockPerWindow(t *testing.T) {
	te, barca, madrid, _ := createOverhaulTestUniverse()

	p := barca.Squad[0] // Pedri
	neg := &TransferNegotiation{
		NegotiationID: "TEST_LOCK_1",
		Player:        p,
		Buyer:         madrid,
		Seller:        barca,
		CurrentBid:    60_000_000,
	}

	// Execute transfer
	te.executeTransfer(neg)

	if !te.TransferredThisWindow[p.PlayerID] {
		t.Fatalf("Player %s should be flagged as transferred this window", p.PlayerID)
	}

	// Attempt second transfer of the same player in the same window
	neg2 := te.InitiateBid(p.PlayerID, barca.ClubID, 70_000_000)
	if neg2 != nil {
		t.Errorf("Expected InitiateBid to be blocked for already transferred player, got %+v", neg2)
	}

	neg3 := te.TriggerSpecificBid(barca.ClubID, madrid.ClubID, p.PlayerID)
	if neg3 != nil {
		t.Errorf("Expected TriggerSpecificBid to be blocked for already transferred player, got %+v", neg3)
	}
}

func TestWarchestDeductionAndNoNegativeBalance(t *testing.T) {
	te, barca, madrid, _ := createOverhaulTestUniverse()

	bMgr := te.Managers[barca.ClubID]
	mMgr := te.Managers[madrid.ClubID]

	initialBuyerBudget := mMgr.BudgetEur
	initialSellerBudget := bMgr.BudgetEur
	fee := int64(50_000_000)

	p := barca.Squad[0]
	neg := &TransferNegotiation{
		NegotiationID: "TEST_FINANCE_1",
		Player:        p,
		Buyer:         madrid,
		Seller:        barca,
		CurrentBid:    fee,
	}

	te.executeTransfer(neg)

	if mMgr.BudgetEur != initialBuyerBudget-fee {
		t.Errorf("Buyer budget mismatch: want %d, got %d", initialBuyerBudget-fee, mMgr.BudgetEur)
	}
	if bMgr.BudgetEur != initialSellerBudget+fee {
		t.Errorf("Seller budget mismatch: want %d, got %d", initialSellerBudget+fee, bMgr.BudgetEur)
	}

	// Ensure clubs cannot bid if they lack budget
	mMgr.BudgetEur = 10_000_000
	hugeBid := te.InitiateBid(barca.Squad[0].PlayerID, madrid.ClubID, 50_000_000)
	if hugeBid != nil {
		t.Errorf("Club should not be able to bid %d with warchest of %d", 50_000_000, mMgr.BudgetEur)
	}
}

func TestWonderkidTransferRestrictions(t *testing.T) {
	te, barca, madrid, nonLeague := createOverhaulTestUniverse()

	wk := barca.Squad[1] // Venjamin Valerio (UniverseWonderkid)

	// Wonderkid transfer to non-Super League club MUST be rejected
	negIllegal := te.InitiateBid(wk.PlayerID, nonLeague.ClubID, 40_000_000)
	if negIllegal != nil {
		t.Errorf("Wonderkid should not be able to transfer to non-Super League club, got: %+v", negIllegal)
	}

	negIllegal2 := te.TriggerSpecificBid(nonLeague.ClubID, barca.ClubID, wk.PlayerID)
	if negIllegal2 != nil {
		t.Errorf("Wonderkid should not be able to be targeted by non-Super League club, got: %+v", negIllegal2)
	}

	// Transfer within Super League clubs (Barca -> Madrid) is allowed
	negLegal := te.TriggerSpecificBid(madrid.ClubID, barca.ClubID, wk.PlayerID)
	if negLegal == nil {
		t.Fatal("Wonderkid transfer between Super League clubs should be allowed")
	}

	// Verify OriginalClubID is preserved during wonderkid transfer
	te.executeTransfer(negLegal)
	if wk.ClubID != madrid.ClubID {
		t.Errorf("Wonderkid should be at Madrid, got %s", wk.ClubID)
	}
	if wk.OriginalClubID != "LAL-BAR" {
		t.Errorf("Wonderkid OriginalClubID should remain LAL-BAR, got %s", wk.OriginalClubID)
	}
}

func TestTwelveWeekWindowProgression(t *testing.T) {
	te, _, _, _ := createOverhaulTestUniverse()
	te.IsOffSeason = true

	if te.CurrentWeek != 1 {
		t.Fatalf("Expected starting week 1, got %d", te.CurrentWeek)
	}

	for week := 1; week <= 12; week++ {
		if !te.IsWindowOpen() {
			t.Errorf("Window should be open in off-season week %d", week)
		}
		if !strings.Contains(te.GetWindowName(), "Week") {
			t.Errorf("Window name should contain 'Week', got %s", te.GetWindowName())
		}
		te.AdvanceOpenWindow()
	}

	if te.CurrentWeek != 13 {
		t.Fatalf("Expected CurrentWeek=13 after 12 advances, got %d", te.CurrentWeek)
	}

	// Window should now be closed
	if te.IsWindowOpen() {
		t.Errorf("Window should be closed after 12 weeks, got IsWindowOpen=true")
	}

	// Advancing beyond week 12 should be a no-op
	prevDay := te.CurrentDay
	te.AdvanceOpenWindow()
	if te.CurrentDay != prevDay {
		t.Errorf("Advancing beyond week 12 should be no-op, day moved from %d to %d", prevDay, te.CurrentDay)
	}
}
