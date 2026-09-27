package tournament

import (
	"fmt"
	"testing"

	"football_sim/pkg/models"
	"football_sim/pkg/transfers"
)

func TestSeasonWrapTicksContractsAndLoyalty(t *testing.T) {
	club := &models.Club{
		ClubID: "SEA-MIL", ClubName: "AC Milan", ShortName: "MIL", League: "Serie A",
		Played: 38, Won: 22, Points: 70, ExpectedFinish: 2,
	}
	p := &models.Player{
		PlayerID: "P1", FullName: "Pro", ClubID: club.ClubID,
		ContractYears: 3, Loyalty: 60, Appearances: 20, OVR: 78, Age: 24,
	}
	club.Squad = []*models.Player{p}
	tm := &TournamentManager{
		Clubs:      map[string]*models.Club{club.ClubID: club},
		ClubsList:  []*models.Club{club},
		SeasonName: "2026-27",
	}
	tm.tickContractsUnlocked()
	if p.ContractYears != 2 {
		t.Fatalf("contract tick = %d want 2", p.ContractYears)
	}
	applySeasonEndLoyalty(p, 1)
	if p.Loyalty != 65 {
		t.Fatalf("top-four loyalty = %d want 65", p.Loyalty)
	}
}

func TestExpiredSnakeResolvesThroughTransferEngine(t *testing.T) {
	milan := &models.Club{ClubID: "SEA-MIL", ClubName: "Milan", ShortName: "MIL", OverallTeamRating: 80, Identity: models.DefaultClubIdentity("SEA-MIL", 80)}
	milan.Finances = models.InitialClubFinances(milan.Identity)
	inter := &models.Club{ClubID: "SEA-INT", ClubName: "Inter", ShortName: "INT", OverallTeamRating: 90, Identity: models.DefaultClubIdentity("SEA-INT", 90)}
	inter.Finances = models.InitialClubFinances(inter.Identity)
	p := &models.Player{
		PlayerID: "WK_Earl_Josh_Hernando", FullName: "Earl Josh Hernando",
		UniverseWonderkid: true, ClubID: milan.ClubID, ContractYears: 0,
		Loyalty: 18, Personality: "snake", TransferRequested: true, OVR: 78, WageEUR: 30000, Position: "LW",
	}
	milan.Squad = []*models.Player{p}
	for len(milan.Squad) <= models.MinSeniorSquadSize {
		i := len(milan.Squad)
		milan.Squad = append(milan.Squad, &models.Player{PlayerID: fmt.Sprintf("DEPTH_%02d", i), ClubID: milan.ClubID, ContractYears: 2})
	}
	tm := &TournamentManager{
		Clubs:          map[string]*models.Club{milan.ClubID: milan, inter.ClubID: inter},
		ClubsList:      []*models.Club{milan, inter},
		TransferEngine: transfers.NewTransferEngine([]*models.Club{milan, inter}, nil, 2),
		SeasonName:     "2026-27",
	}
	tm.resolveExpiredContractsUnlocked()
	if p.ClubID != inter.ClubID {
		t.Fatalf("snake should leave on a free, club=%s", p.ClubID)
	}
}

func TestWonderkidContractBackfillFromAge(t *testing.T) {
	p := &models.Player{UniverseWonderkid: true, Age: 20, ContractYears: 3}
	p.BackfillElapsedWonderkidContract()
	if p.ContractYears != 0 {
		t.Fatalf("age 20 should have used a 3-year deal, remaining=%d", p.ContractYears)
	}
	renewed := &models.Player{UniverseWonderkid: true, Age: 21, ContractYears: 2}
	renewed.BackfillElapsedWonderkidContract()
	if renewed.ContractYears != 2 {
		t.Fatalf("renewed wonderkid deal must not be clamped on load, remaining=%d", renewed.ContractYears)
	}
}

func TestSeasonContractsTickOnceBeforeSummerWindow(t *testing.T) {
	p := &models.Player{PlayerID: "P1", FullName: "One Year Left", Position: "ST", Category: "FWD", OVR: 78, Age: 26, ClubID: "C1", ContractYears: 1, Loyalty: 80}
	club := &models.Club{ClubID: "C1", ClubName: "Club", ShortName: "CLB", Squad: []*models.Player{p}}
	tm := &TournamentManager{
		Clubs:      map[string]*models.Club{club.ClubID: club},
		ClubsList:  []*models.Club{club},
		SeasonName: "2026-27",
	}
	tm.resolveSeasonContractsOnceUnlocked()
	if p.ContractYears != 3 {
		t.Fatalf("loyal expiry should re-sign after the season-end tick, years=%d", p.ContractYears)
	}
	tm.resolveSeasonContractsOnceUnlocked()
	if p.ContractYears != 3 {
		t.Fatalf("second season-end pass must not tick again, years=%d", p.ContractYears)
	}
}

func TestExpiredContractIsValidWorldState(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	player := tm.ClubsList[0].Squad[0]
	player.ContractYears = 0
	if err := tm.ValidateWorldState(); err != nil {
		t.Fatalf("expired contract 0 should be legal world state: %v", err)
	}
}
