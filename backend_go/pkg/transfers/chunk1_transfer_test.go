package transfers

import (
	"testing"

	"football_sim/pkg/models"
)

func testClub(id string, budget int64, players ...*models.Player) *models.Club {
	c := &models.Club{
		ClubID: id, ClubName: id, ShortName: id, Squad: players,
		OverallTeamRating: 80,
		Identity: models.DefaultClubIdentity(id, 80),
		Finances: models.ClubFinances{TransferBudget: budget, Balance: budget},
	}
	for _, p := range players {
		if p != nil {
			p.ClubID = id
			if p.OriginalClubID == "" { p.OriginalClubID = id }
		}
	}
	c.RecalculateRatings()
	return c
}

func TestPermanentTransferUpdatesClubFinancesAndOnlyOnce(t *testing.T) {
	p := &models.Player{PlayerID: "P_NORMAL", FullName: "Normal Player", Position: "CM", Category: "MID", OVR: 80, Age: 22}
	seller := testClub("EPL-ARS", 80_000_000, p)
	buyer := testClub("EPL-LIV", 100_000_000)
	te := NewTransferEngine([]*models.Club{seller, buyer}, nil, 7)
	te.BeginOffSeasonWindow()
	buyer.Finances.TransferBudget, buyer.Finances.Balance = 100_000_000, 100_000_000
	seller.Finances.TransferBudget, seller.Finances.Balance = 80_000_000, 80_000_000

	fee := int64(30_000_000)
	neg := &TransferNegotiation{Player:p, Seller:seller, Buyer:buyer, CurrentBid:fee}
	if !te.executeTransfer(neg) { t.Fatal("first legal transfer rejected") }
	if p.ClubID != buyer.ClubID { t.Fatalf("club=%s want %s", p.ClubID, buyer.ClubID) }
	if buyer.Finances.TransferBudget != 70_000_000 || buyer.Finances.Balance != 70_000_000 {
		t.Fatalf("buyer finances=%+v", buyer.Finances)
	}
	if seller.Finances.Balance != 110_000_000 || seller.Finances.TransferBudget != 110_000_000 {
		t.Fatalf("seller finances=%+v", seller.Finances)
	}
	if !te.TransferredThisWindow[p.PlayerID] { t.Fatal("transfer marker not set") }

	buyer2 := testClub("BUN-BAY", 100_000_000)
	te.Clubs[buyer2.ClubID] = buyer2
	beforeBuyer2 := buyer2.Finances
	beforeCurrent := buyer.Finances
	if te.executeTransfer(&TransferNegotiation{Player:p, Seller:buyer, Buyer:buyer2, CurrentBid:20_000_000}) {
		t.Fatal("second transfer in same window succeeded")
	}
	if buyer2.Finances != beforeBuyer2 || buyer.Finances != beforeCurrent {
		t.Fatal("failed second transfer changed finances")
	}
}

func TestCannotBuyAboveClubBudget(t *testing.T) {
	p := &models.Player{PlayerID:"P_EXPENSIVE", FullName:"Expensive", Position:"ST", Category:"FWD", OVR:84, Age:23}
	seller := testClub("EPL-ARS", 100_000_000, p)
	buyer := testClub("EPL-LIV", 10_000_000)
	te := NewTransferEngine([]*models.Club{seller,buyer}, nil, 8)
	buyer.Finances = models.ClubFinances{TransferBudget:10_000_000, Balance:100_000_000}
	beforeSeller, beforeBuyer := seller.Finances, buyer.Finances
	if te.executeTransfer(&TransferNegotiation{Player:p,Seller:seller,Buyer:buyer,CurrentBid:11_000_000}) {
		t.Fatal("over-budget transfer succeeded")
	}
	if seller.Finances != beforeSeller || buyer.Finances != beforeBuyer || p.ClubID != seller.ClubID {
		t.Fatal("failed over-budget transfer mutated world state")
	}
}

func TestCanonicalWonderkidRestrictedToDesignatedTwelveClubs(t *testing.T) {
	wk := &models.Player{PlayerID:"WK_Izyan_Levin_Bantol", FullName:"Izyan Levin Bantol", UniverseWonderkid:true, Position:"CAM", Category:"MID", OVR:76, Age:14}
	seller := testClub("EPL-ARS", 150_000_000, wk)
	allowed := testClub("LAL-RMA", 150_000_000)
	outside := testClub("EPL-MCI", 150_000_000)
	te := NewTransferEngine([]*models.Club{seller,allowed,outside}, nil, 9)
	allowed.Finances = models.ClubFinances{TransferBudget:150_000_000,Balance:150_000_000}
	outside.Finances = models.ClubFinances{TransferBudget:150_000_000,Balance:150_000_000}
	if te.executeTransfer(&TransferNegotiation{Player:wk,Seller:seller,Buyer:outside,CurrentBid:20_000_000}) {
		t.Fatal("canonical wonderkid transferred outside designated 12-club ecosystem")
	}
	if wk.ClubID != seller.ClubID { t.Fatal("rejected move changed wonderkid club") }
	if !te.executeTransfer(&TransferNegotiation{Player:wk,Seller:seller,Buyer:allowed,CurrentBid:20_000_000}) {
		t.Fatal("canonical wonderkid move to designated club rejected")
	}
}

func TestOrdinaryPlayerNotAccidentallyRestrictedToTwelveClubs(t *testing.T) {
	p := &models.Player{PlayerID:"P_ORDINARY",FullName:"Ordinary",Position:"RW",Category:"FWD",OVR:78,Age:21}
	seller := testClub("EPL-ARS",100_000_000,p)
	outside := testClub("EPL-MCI",100_000_000)
	te := NewTransferEngine([]*models.Club{seller,outside},nil,10)
	outside.Finances=models.ClubFinances{TransferBudget:100_000_000,Balance:100_000_000}
	if !te.executeTransfer(&TransferNegotiation{Player:p,Seller:seller,Buyer:outside,CurrentBid:10_000_000}) {
		t.Fatal("ordinary player was incorrectly restricted by special ecosystem rule")
	}
}

func TestTransferMarkerResetsOnlyAtNewWindow(t *testing.T) {
	p := &models.Player{PlayerID:"P_MARKER",FullName:"Marker",Position:"CM",Category:"MID",OVR:80,Age:22}
	club := testClub("EPL-ARS",100_000_000,p)
	te := NewTransferEngine([]*models.Club{club},nil,11)
	te.TransferredThisWindow[p.PlayerID]=true
	te.IsOffSeason=true
	te.CurrentWeek=6
	te.ResetForNewSeason()
	if !te.TransferredThisWindow[p.PlayerID] { t.Fatal("new-season reset cleared transfer marker too early") }
	te.BeginOffSeasonWindow()
	if te.TransferredThisWindow[p.PlayerID] { t.Fatal("new transfer window did not reset eligibility marker") }
	if te.CurrentWeek != 1 { t.Fatalf("new window week=%d want 1",te.CurrentWeek) }
}
