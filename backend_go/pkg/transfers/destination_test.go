package transfers

import (
	"math/rand"
	"testing"

	"football_sim/pkg/models"
)

func TestStarRejectsWeakerDestination(t *testing.T) {
	star := &models.Player{PlayerID: "ST1", FullName: "Star", OVR: 88, SquadRole: models.RoleCrucial, Morale: 80}
	milan := &models.Club{ClubID: "SEA-MIL", ClubName: "Milan", ShortName: "MIL", League: "Serie A", OverallTeamRating: 86, Identity: models.ClubIdentity{Reputation: 88}}
	mid := &models.Club{ClubID: "FL1-REI", ClubName: "Reims", ShortName: "REI", League: "Ligue 1", OverallTeamRating: 72, Identity: models.ClubIdentity{Reputation: 55}}
	if PlayerAcceptsDestination(star, milan, mid) {
		t.Fatal("star should reject a clear step down")
	}
	united := &models.Club{ClubID: "EPL-MUN", ClubName: "Manchester United", ShortName: "MUN", League: "Premier League", OverallTeamRating: 87, Identity: models.ClubIdentity{Reputation: 90}}
	if !PlayerAcceptsDestination(star, milan, united) {
		t.Fatal("star should consider a comparable Premier League move")
	}
	star.TransferRequested = true
	if PlayerAcceptsDestination(star, milan, mid) {
		t.Fatal("a transfer request must not make an elite crucial player accept an extreme downgrade")
	}
}

func TestClubIdentityShapesMarketActivity(t *testing.T) {
	quiet := &models.Club{ClubID: "QUIET", Identity: models.ClubIdentity{TransferAggressiveness: 0, SellingTendency: 0}}
	active := &models.Club{ClubID: "ACTIVE", Identity: models.ClubIdentity{TransferAggressiveness: 100, SellingTendency: 100}}
	te := &TransferEngine{RNG: rand.New(rand.NewSource(7711))}
	buyerCounts := map[string]int{}
	sellerCounts := map[string]int{}
	clubs := []*models.Club{quiet, active}
	for i := 0; i < 1000; i++ {
		buyerCounts[te.weightedClubChoice(clubs, buyerMarketWeight).ClubID]++
		sellerCounts[te.weightedClubChoice(clubs, sellerMarketWeight).ClubID]++
	}
	if buyerCounts[active.ClubID] <= buyerCounts[quiet.ClubID]*3 {
		t.Fatalf("aggressiveness did not shape buying activity: %#v", buyerCounts)
	}
	if sellerCounts[active.ClubID] <= sellerCounts[quiet.ClubID]*3 {
		t.Fatalf("selling tendency did not shape availability: %#v", sellerCounts)
	}
	if reluctant, willing := sellerCounterPremium(quiet), sellerCounterPremium(active); reluctant <= willing {
		t.Fatalf("reluctant seller premium %.2f should exceed willing premium %.2f", reluctant, willing)
	}
}

func TestRecruitmentAmbitionPrefersFirstTeamQuality(t *testing.T) {
	seller := &models.Club{ClubID: "SELL", OverallTeamRating: 70, Identity: models.ClubIdentity{Reputation: 55}}
	lowAmbition := &models.Club{ClubID: "LOW", OverallTeamRating: 80, Identity: models.ClubIdentity{Reputation: 75, RecruitmentAmbition: 10}}
	highAmbition := &models.Club{ClubID: "HIGH", OverallTeamRating: 80, Identity: models.ClubIdentity{Reputation: 75, RecruitmentAmbition: 90}}
	te := &TransferEngine{}
	backup := &models.Player{PlayerID: "BACKUP", Position: "ST", Category: "FWD", OVR: 72}
	starter := &models.Player{PlayerID: "STARTER", Position: "ST", Category: "FWD", OVR: 84}
	if low, high := te.scoreTransferTarget(lowAmbition, seller, backup), te.scoreTransferTarget(highAmbition, seller, backup); high >= low {
		t.Fatalf("ambitious club valued below-level backup too highly: low=%d high=%d", low, high)
	}
	if low, high := te.scoreTransferTarget(lowAmbition, seller, starter), te.scoreTransferTarget(highAmbition, seller, starter); high <= low {
		t.Fatalf("ambitious club did not value first-team upgrade: low=%d high=%d", low, high)
	}
}

func TestSnakePersonalityAcceptsReasonableStepDown(t *testing.T) {
	hernando := &models.Player{
		PlayerID: "WK_Earl_Josh_Hernando", FullName: "Earl Josh Hernando",
		OVR: 78, SquadRole: models.RoleImportant, Morale: 55, Personality: "snake",
	}
	milan := &models.Club{ClubID: "SEA-MIL", ClubName: "Milan", ShortName: "MIL", League: "Serie A", OverallTeamRating: 86, Identity: models.ClubIdentity{Reputation: 88}}
	mid := &models.Club{ClubID: "FL1-REI", ClubName: "Reims", ShortName: "REI", League: "Ligue 1", OverallTeamRating: 72, Identity: models.ClubIdentity{Reputation: 55}}
	if !PlayerAcceptsDestination(hernando, milan, mid) {
		t.Fatal("an unsettled non-elite player should accept a plausible step down")
	}
}

func TestTransferNeedDistinguishesFullBackFromStriker(t *testing.T) {
	club := &models.Club{Squad: []*models.Player{
		{Position: "RB", Category: "DEF"}, {Position: "RB", Category: "DEF"},
		{Position: "LB", Category: "DEF"}, {Position: "LB", Category: "DEF"},
		{Position: "RWB", Category: "DEF"},
	}}
	te := &TransferEngine{}
	fullBack := &models.Player{Position: "RB", Category: "DEF"}
	striker := &models.Player{Position: "ST", Category: "FWD"}
	if fbNeed, stNeed := te.positionNeedScore(club, fullBack), te.positionNeedScore(club, striker); stNeed <= fbNeed {
		t.Fatalf("striker need=%d should exceed overstocked full-back need=%d", stNeed, fbNeed)
	}
}
