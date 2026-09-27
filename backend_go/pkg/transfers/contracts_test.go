package transfers

import (
	"fmt"
	"testing"

	"football_sim/pkg/models"
)

func TestSnakeLeavesOnAFreeWhenContractExpires(t *testing.T) {
	milan := &models.Club{
		ClubID: "SEA-MIL", ClubName: "AC Milan", ShortName: "MIL", League: "Serie A",
		OverallTeamRating: 84,
		Identity:          models.DefaultClubIdentity("SEA-MIL", 84),
	}
	milan.Finances = models.InitialClubFinances(milan.Identity)
	inter := &models.Club{
		ClubID: "SEA-INT", ClubName: "Inter", ShortName: "INT", League: "Serie A",
		OverallTeamRating: 88,
		Identity:          models.DefaultClubIdentity("SEA-INT", 88),
	}
	inter.Finances = models.InitialClubFinances(inter.Identity)
	hernando := &models.Player{
		PlayerID: "WK_Earl_Josh_Hernando", FullName: "Earl Josh Hernando",
		Position: "LW", Category: "FWD", OVR: 78, Age: 20, WageEUR: 40000,
		UniverseWonderkid: true, ClubID: milan.ClubID,
		ContractYears: 0, Loyalty: 20, Personality: "snake", TransferRequested: true,
	}
	milan.Squad = []*models.Player{hernando}
	fillSellerToTransferableSize(milan)
	inter.Squad = []*models.Player{}
	te := NewTransferEngine([]*models.Club{milan, inter}, nil, 1)
	te.ResolveExpiredContracts()
	if hernando.ClubID != inter.ClubID {
		t.Fatalf("expected free move to Inter, club=%s", hernando.ClubID)
	}
	if hernando.ContractYears != 3 {
		t.Fatalf("new deal should be 3 years, got %d", hernando.ContractYears)
	}
	if len(te.CompletedTransfers) != 1 || te.CompletedTransfers[0].FeeEUR != 0 {
		t.Fatalf("expected a free transfer, got %#v", te.CompletedTransfers)
	}
}

func TestLoyalPlayerResignsWhenContractExpires(t *testing.T) {
	club := &models.Club{
		ClubID: "SEA-MIL", ClubName: "AC Milan", ShortName: "MIL",
		OverallTeamRating: 84, Identity: models.DefaultClubIdentity("SEA-MIL", 84),
	}
	club.Finances = models.InitialClubFinances(club.Identity)
	p := &models.Player{
		PlayerID: "P1", FullName: "Loyal Pro", Position: "CM", OVR: 78, Age: 26,
		WageEUR: 50000, ClubID: club.ClubID, ContractYears: 0, Loyalty: 80,
	}
	club.Squad = []*models.Player{p}
	te := NewTransferEngine([]*models.Club{club}, nil, 1)
	te.ResolveExpiredContracts()
	if p.ClubID != club.ClubID {
		t.Fatalf("loyal player moved: %s", p.ClubID)
	}
	if p.ContractYears != 3 {
		t.Fatalf("resign years=%d", p.ContractYears)
	}
}

func TestUnplacedLeaverBecomesFreeAgentWhenNoClubCanSign(t *testing.T) {
	club := &models.Club{
		ClubID: "ONLY", ClubName: "Only FC", ShortName: "ONE", OverallTeamRating: 70,
		Identity: models.DefaultClubIdentity("ONLY", 70),
	}
	club.Finances = models.InitialClubFinances(club.Identity)
	club.Finances.WageCap = 1
	p := &models.Player{
		PlayerID: "P_FREE", FullName: "Unplaced Pro", Position: "ST", Category: "FWD",
		OVR: 70, Age: 27, ClubID: club.ClubID, ContractYears: 0, Loyalty: 20, TransferRequested: true,
		WageEUR: 80_000,
	}
	club.Squad = []*models.Player{p}
	te := NewTransferEngine([]*models.Club{club}, nil, 1)
	club.Finances.WageCap = 1
	te.ResolveExpiredContracts()
	if p.ClubID != "" || !p.IsFreeAgent() || p.ContractYears != 0 {
		t.Fatalf("expected free agent, club=%s years=%d status=%s", p.ClubID, p.ContractYears, p.RegistrationStatus)
	}
	if len(club.Squad) != 0 {
		t.Fatalf("free agent still listed in squad: %d", len(club.Squad))
	}
	if len(te.FreeAgentList()) != 1 || te.FreeAgentList()[0].PlayerID != p.PlayerID {
		t.Fatal("free-agent pool missing the released player")
	}
	te.ResolveExpiredContracts()
	if p.ClubID == club.ClubID && p.ContractYears == 1 {
		t.Fatal("emergency 1-year hold returned the player to the old club")
	}
	if p.ClubID != "" || !p.IsFreeAgent() || p.ContractYears != 0 {
		t.Fatalf("second pass must leave the player a free agent, club=%s years=%d", p.ClubID, p.ContractYears)
	}
}

func TestPaidSummerTransferStartsNextSeasonWithThreeYears(t *testing.T) {
	seller := &models.Club{ClubID: "SELL", ClubName: "Seller", ShortName: "SEL", OverallTeamRating: 75, Identity: models.DefaultClubIdentity("SELL", 75)}
	buyer := &models.Club{ClubID: "BUY", ClubName: "Buyer", ShortName: "BUY", OverallTeamRating: 80, Identity: models.DefaultClubIdentity("BUY", 80)}
	seller.Finances = models.InitialClubFinances(seller.Identity)
	buyer.Finances = models.InitialClubFinances(buyer.Identity)
	p := &models.Player{PlayerID: "P_DEAL", FullName: "New Signing", Position: "CM", Category: "MID", OVR: 76, Age: 24, ClubID: seller.ClubID, ContractYears: 1, WageEUR: 20_000}
	seller.Squad = []*models.Player{p}
	fillSellerToTransferableSize(seller)
	te := NewTransferEngine([]*models.Club{seller, buyer}, nil, 1)
	te.BeginOffSeasonWindow()
	fee := int64(5_000_000)
	if !te.executeTransfer(&TransferNegotiation{Player: p, Seller: seller, Buyer: buyer, CurrentBid: fee}) {
		t.Fatal("paid summer transfer failed")
	}
	if p.ContractYears != 3 {
		t.Fatalf("summer contract after pre-window tick=%d want 3", p.ContractYears)
	}
}

func TestPermanentTransferCannotOverflowBuyerSquad(t *testing.T) {
	seller := &models.Club{ClubID: "SELL", Identity: models.DefaultClubIdentity("SELL", 75)}
	buyer := &models.Club{ClubID: "BUY", Identity: models.DefaultClubIdentity("BUY", 80)}
	seller.Finances = models.InitialClubFinances(seller.Identity)
	buyer.Finances = models.InitialClubFinances(buyer.Identity)
	target := &models.Player{PlayerID: "TARGET", ClubID: seller.ClubID, OVR: 76, ContractYears: 2}
	seller.Squad = []*models.Player{target}
	for i := 0; i < models.MaxSeniorSquadSize; i++ {
		buyer.Squad = append(buyer.Squad, &models.Player{PlayerID: string(rune('A' + i)), ClubID: buyer.ClubID})
	}
	te := NewTransferEngine([]*models.Club{seller, buyer}, nil, 1)
	te.BeginOffSeasonWindow()
	if te.executeTransfer(&TransferNegotiation{Player: target, Seller: seller, Buyer: buyer, CurrentBid: 1_000_000}) {
		t.Fatal("transfer overflowed a full 34-player squad")
	}
	if target.ClubID != seller.ClubID || len(seller.Squad) != 1 || len(buyer.Squad) != models.MaxSeniorSquadSize {
		t.Fatal("rejected overflow mutated ownership")
	}
}

func TestOutboundLoaneeStillConsumesParentTransferCapacity(t *testing.T) {
	seller := &models.Club{ClubID: "SELL", Identity: models.DefaultClubIdentity("SELL", 75)}
	buyer := &models.Club{ClubID: "BUY", Identity: models.DefaultClubIdentity("BUY", 80)}
	loanClub := &models.Club{ClubID: "LOAN", Identity: models.DefaultClubIdentity("LOAN", 70)}
	for _, club := range []*models.Club{seller, buyer, loanClub} {
		club.Finances = models.InitialClubFinances(club.Identity)
	}
	target := &models.Player{PlayerID: "TARGET", ClubID: seller.ClubID, OVR: 76, ContractYears: 2}
	seller.Squad = []*models.Player{target}
	for i := 0; i < models.MaxSeniorSquadSize-1; i++ {
		buyer.Squad = append(buyer.Squad, &models.Player{PlayerID: "B" + string(rune('A'+i)), ClubID: buyer.ClubID})
	}
	loanClub.Squad = []*models.Player{{PlayerID: "OUT", ClubID: loanClub.ClubID, ParentClubID: buyer.ClubID, OnLoan: true}}
	te := NewTransferEngine([]*models.Club{seller, buyer, loanClub}, nil, 1)
	te.BeginOffSeasonWindow()
	if got := te.committedSquadSize(buyer.ClubID); got != models.MaxSeniorSquadSize {
		t.Fatalf("committed squad=%d want %d", got, models.MaxSeniorSquadSize)
	}
	if te.executeTransfer(&TransferNegotiation{Player: target, Seller: seller, Buyer: buyer, CurrentBid: 1_000_000}) {
		t.Fatal("outbound loan created a false transfer slot")
	}
}

func TestPermanentTransferCannotLeaveSellerWithoutPlayableSquad(t *testing.T) {
	seller := &models.Club{ClubID: "SELL", Identity: models.DefaultClubIdentity("SELL", 75)}
	buyer := &models.Club{ClubID: "BUY", Identity: models.DefaultClubIdentity("BUY", 80)}
	for _, club := range []*models.Club{seller, buyer} {
		club.Finances = models.InitialClubFinances(club.Identity)
	}
	for i := 0; i < models.MinSeniorSquadSize; i++ {
		seller.Squad = append(seller.Squad, &models.Player{PlayerID: "S" + string(rune('A'+i)), ClubID: seller.ClubID, ContractYears: 2})
	}
	target := seller.Squad[0]
	te := NewTransferEngine([]*models.Club{seller, buyer}, nil, 1)
	te.BeginOffSeasonWindow()
	if te.executeTransfer(&TransferNegotiation{Player: target, Seller: seller, Buyer: buyer, CurrentBid: 1_000_000}) {
		t.Fatal("transfer left seller without a playable eleven")
	}
	if len(seller.Squad) != models.MinSeniorSquadSize || target.ClubID != seller.ClubID {
		t.Fatal("rejected seller-floor move mutated ownership")
	}
}

func fillSellerToTransferableSize(club *models.Club) {
	for len(club.Squad) <= models.MinSeniorSquadSize {
		i := len(club.Squad)
		club.Squad = append(club.Squad, &models.Player{
			PlayerID:      fmt.Sprintf("%s_DEPTH_%02d", club.ClubID, i),
			FullName:      fmt.Sprintf("Depth Player %02d", i),
			ClubID:        club.ClubID,
			ContractYears: 2,
		})
	}
}
