package transfers

import (
	"strings"
	"testing"

	"football_sim/pkg/managers"
	"football_sim/pkg/models"
)

func createOverhaulTestUniverse() (*TransferEngine, *models.Club, *models.Club, *models.Club) {
	barca := &models.Club{
		ClubID: "LAL-BAR", ClubName: "FC Barcelona", ShortName: "BAR", OverallTeamRating: 88,
		Squad: []*models.Player{
			{PlayerID: "P1", FullName: "Pedri", OVR: 86, Age: 23, MarketValueEUR: 80_000_000, ClubID: "LAL-BAR", OriginalClubID: "LAL-BAR"},
			{PlayerID: "WK_Venjamin_Valerio", FullName: "Venjamin Valerio", OVR: 75, Age: 14, UniverseWonderkid: true, MarketValueEUR: 30_000_000, ClubID: "LAL-BAR", OriginalClubID: "LAL-BAR"},
		},
	}
	madrid := &models.Club{
		ClubID: "LAL-RMA", ClubName: "Real Madrid", ShortName: "RMA", OverallTeamRating: 90,
		Squad: []*models.Player{{PlayerID: "P2", FullName: "Bellingham", OVR: 90, Age: 23, MarketValueEUR: 120_000_000, ClubID: "LAL-RMA", OriginalClubID: "LAL-RMA"}},
	}
	nonLeague := &models.Club{
		ClubID: "LOW-TIER", ClubName: "Low Tier FC", ShortName: "LOW", OverallTeamRating: 65,
		Squad: []*models.Player{{PlayerID: "P3", FullName: "Local Guy", OVR: 68, Age: 25, MarketValueEUR: 2_000_000, ClubID: "LOW-TIER", OriginalClubID: "LOW-TIER"}},
	}
	clubs := []*models.Club{barca, madrid, nonLeague}
	te := NewTransferEngine(clubs, managers.BuildManagers(clubs), 42)
	te.BeginOffSeasonWindow()
	return te, barca, madrid, nonLeague
}

func TestWarchestInitialization(t *testing.T) {
	te, barca, madrid, nonLeague := createOverhaulTestUniverse()
	for _, c := range []*models.Club{barca, madrid, nonLeague} {
		if c.Finances.TransferBudget < 0 || c.Finances.TransferBudget > c.Finances.Balance {
			t.Fatalf("%s has invalid club finances budget=%d balance=%d", c.ShortName, c.Finances.TransferBudget, c.Finances.Balance)
		}
		if te.Managers[c.ClubID].BudgetEur != c.Finances.TransferBudget {
			t.Fatalf("%s manager compatibility budget is not mirroring club warchest", c.ShortName)
		}
	}
	if madrid.Finances.TransferBudget < barca.Finances.TransferBudget {
		t.Errorf("expected Madrid warchest (%d) >= Barca (%d)", madrid.Finances.TransferBudget, barca.Finances.TransferBudget)
	}
}

func TestSingleTransferLockPerWindow(t *testing.T) {
	te, barca, madrid, _ := createOverhaulTestUniverse()
	p := barca.Squad[0]
	neg := &TransferNegotiation{NegotiationID: "TEST_LOCK_1", Player: p, Buyer: madrid, Seller: barca, CurrentBid: 60_000_000}
	if !te.executeTransfer(neg) { t.Fatal("first transfer should commit") }
	if !te.TransferredThisWindow[p.PlayerID] { t.Fatalf("player %s should be flagged as transferred this window", p.PlayerID) }
	if neg2 := te.InitiateBid(p.PlayerID, barca.ClubID, 70_000_000); neg2 != nil { t.Errorf("second transfer should be blocked, got %+v", neg2) }
	if neg3 := te.TriggerSpecificBid(barca.ClubID, madrid.ClubID, p.PlayerID); neg3 != nil { t.Errorf("second targeted transfer should be blocked, got %+v", neg3) }
}

func TestWarchestDeductionAndNoNegativeBalance(t *testing.T) {
	te, barca, madrid, _ := createOverhaulTestUniverse()
	buyerBudget, buyerBalance := madrid.Finances.TransferBudget, madrid.Finances.Balance
	sellerBudget, sellerBalance := barca.Finances.TransferBudget, barca.Finances.Balance
	fee := int64(50_000_000)
	p := barca.Squad[0]
	if !te.executeTransfer(&TransferNegotiation{NegotiationID: "TEST_FINANCE_1", Player: p, Buyer: madrid, Seller: barca, CurrentBid: fee}) {
		t.Fatal("funded transfer should commit")
	}
	if madrid.Finances.TransferBudget != buyerBudget-fee || madrid.Finances.Balance != buyerBalance-fee {
		t.Fatalf("buyer accounting mismatch: budget=%d balance=%d", madrid.Finances.TransferBudget, madrid.Finances.Balance)
	}
	if barca.Finances.Balance != sellerBalance+fee || barca.Finances.TransferBudget < sellerBudget {
		t.Fatalf("seller accounting mismatch: budget=%d balance=%d", barca.Finances.TransferBudget, barca.Finances.Balance)
	}
	// Club finances, not the legacy manager mirror, are authoritative.
	madrid.Finances.TransferBudget = 10_000_000
	te.syncManagerBudget(madrid.ClubID)
	if huge := te.InitiateBid(barca.Squad[0].PlayerID, madrid.ClubID, 50_000_000); huge != nil {
		t.Errorf("club should not be able to bid above its warchest")
	}
}

func TestWonderkidTransferRestrictions(t *testing.T) {
	te, barca, madrid, nonLeague := createOverhaulTestUniverse()
	wk := barca.Squad[1]
	if neg := te.InitiateBid(wk.PlayerID, nonLeague.ClubID, 40_000_000); neg != nil { t.Errorf("canonical wonderkid should not transfer outside designated 12") }
	if neg := te.TriggerSpecificBid(nonLeague.ClubID, barca.ClubID, wk.PlayerID); neg != nil { t.Errorf("outside club should not target canonical wonderkid") }
	neg := te.TriggerSpecificBid(madrid.ClubID, barca.ClubID, wk.PlayerID)
	if neg == nil { t.Fatal("canonical wonderkid transfer between designated clubs should be allowed") }
	if !te.executeTransfer(neg) { t.Fatal("legal wonderkid transfer should commit") }
	if wk.ClubID != madrid.ClubID { t.Errorf("wonderkid should be at Madrid, got %s", wk.ClubID) }
	if wk.OriginalClubID != "LAL-BAR" { t.Errorf("historical original club should remain LAL-BAR, got %s", wk.OriginalClubID) }
}

func TestTwelveWeekWindowProgression(t *testing.T) {
	te, _, _, _ := createOverhaulTestUniverse()
	if te.CurrentWeek != 1 { t.Fatalf("expected starting week 1, got %d", te.CurrentWeek) }
	for week := 1; week <= TransferWindowWeeks; week++ {
		if !te.IsWindowOpen() { t.Errorf("window should be open in offseason week %d", week) }
		if !strings.Contains(te.GetWindowName(), "Week") { t.Errorf("window name should contain Week, got %s", te.GetWindowName()) }
		te.AdvanceOpenWindow()
	}
	if te.CurrentWeek != TransferWindowWeeks+1 { t.Fatalf("expected week=%d, got %d", TransferWindowWeeks+1, te.CurrentWeek) }
	if te.IsWindowOpen() { t.Error("window should close after exactly 12 weeks") }
	prevDay := te.CurrentDay
	te.AdvanceOpenWindow()
	if te.CurrentDay != prevDay { t.Errorf("advance after window close should be a no-op") }
}
