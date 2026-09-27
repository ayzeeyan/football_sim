package transfers

import (
	"testing"

	"football_sim/pkg/models"
)

func TestSellerAskingPriceUsesContractAndWillingness(t *testing.T) {
	player := &models.Player{OVR: 82, Age: 24, ContractYears: 4, MarketValueEUR: 50_000_000}
	reluctant := &models.Club{Identity: models.ClubIdentity{SellingTendency: 10}}
	willing := &models.Club{Identity: models.ClubIdentity{SellingTendency: 90}}
	longDealAsk := sellerAskingPrice(player, reluctant)

	player.ContractYears = 1
	shortDealAsk := sellerAskingPrice(player, reluctant)
	if shortDealAsk >= longDealAsk {
		t.Fatalf("short contract ask %d should be below long contract ask %d", shortDealAsk, longDealAsk)
	}
	player.ContractYears = 4
	player.TransferRequested = true
	requestedAsk := sellerAskingPrice(player, reluctant)
	if requestedAsk >= longDealAsk {
		t.Fatalf("transfer request ask %d should be below settled long-deal ask %d", requestedAsk, longDealAsk)
	}
	player.TransferRequested = false
	if willingAsk := sellerAskingPrice(player, willing); willingAsk >= longDealAsk {
		t.Fatalf("willing seller ask %d should be below reluctant ask %d", willingAsk, longDealAsk)
	}
}

func TestNegotiationRejectsOfferBelowSellerFloor(t *testing.T) {
	player := &models.Player{PlayerID: "P_NEGOTIATE", FullName: "Negotiation Target", OVR: 80, Age: 25, ContractYears: 3, WageEUR: 10_000, MarketValueEUR: 30_000_000, ClubID: "SELL"}
	seller := &models.Club{ClubID: "SELL", ClubName: "Seller", ShortName: "SEL", Squad: []*models.Player{player}}
	fillSellerToTransferableSize(seller)
	buyer := &models.Club{ClubID: "BUY", ClubName: "Buyer", ShortName: "BUY"}
	te := NewTransferEngine([]*models.Club{seller, buyer}, nil, 9)
	te.BeginOffSeasonWindow()
	buyer.Finances.TransferBudget = 100_000_000
	buyer.Finances.Balance = 150_000_000
	ask := sellerAskingPrice(player, seller)
	neg := &TransferNegotiation{
		Player: player, Seller: seller, Buyer: buyer,
		CurrentBid: 2_000_000, AskingPrice: ask, StageIndex: 3,
		StageName: "HIJACK_CHECK", IsWonderkid: false,
	}
	if !te.progressNegotiation(neg) {
		t.Fatal("below-floor offer should terminate negotiations")
	}
	if neg.StageName != "COLLAPSED" {
		t.Fatalf("stage=%q, want COLLAPSED", neg.StageName)
	}
}

func TestTransferTargetScoreAccountsForFeeBudget(t *testing.T) {
	player := &models.Player{PlayerID: "P_BUDGET", Position: "ST", Category: "FWD", OVR: 80, Age: 24, ContractYears: 2, MarketValueEUR: 50_000_000}
	seller := &models.Club{ClubID: "SELL", OverallTeamRating: 80, League: "Premier League"}
	constrained := &models.Club{ClubID: "LOW", OverallTeamRating: 80, League: "Premier League", Finances: models.ClubFinances{TransferBudget: 60_000_000, Balance: 60_000_000}}
	wealthy := &models.Club{ClubID: "RICH", OverallTeamRating: 80, League: "Premier League", Finances: models.ClubFinances{TransferBudget: 200_000_000, Balance: 200_000_000}}
	te := &TransferEngine{}
	lowScore := te.scoreTransferTarget(constrained, seller, player)
	highScore := te.scoreTransferTarget(wealthy, seller, player)
	if highScore <= lowScore {
		t.Fatalf("well-funded club should value the same target more: wealthy=%d constrained=%d", highScore, lowScore)
	}
}
