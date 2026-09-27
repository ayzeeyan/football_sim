package tournament

import (
	"fmt"
	"testing"

	"football_sim/pkg/growth"
	"football_sim/pkg/models"
	"football_sim/pkg/transfers"
)

func TestArrangeAndReturnLoansMovesProspects(t *testing.T) {
	parent := &models.Club{
		ClubID: "BIG", ClubName: "Big Town", ShortName: "BIG", League: "Premier League",
		OverallTeamRating: 88, SquadAvgOVR: 84,
	}
	farm := &models.Club{
		ClubID: "SML", ClubName: "Small Town", ShortName: "SML", League: "Premier League",
		OverallTeamRating: 71, SquadAvgOVR: 70,
	}
	for i := 0; i < 24; i++ {
		parent.Squad = append(parent.Squad, &models.Player{
			PlayerID: fmt.Sprintf("SEN%02d", i), FullName: "Senior",
			OVR: 82, Age: 27, Category: "MID", ClubID: parent.ClubID, SquadRole: models.RoleImportant,
		})
	}
	kid := &models.Player{
		PlayerID: "KID1", FullName: "Loan Kid", OVR: 68, Age: 19, Category: "FWD",
		ClubID: parent.ClubID, SquadRole: models.RoleProspect, UniverseWonderkid: false,
	}
	parent.Squad = append(parent.Squad, kid)
	for i := 0; i < 18; i++ {
		farm.Squad = append(farm.Squad, &models.Player{
			PlayerID: fmt.Sprintf("FARM%02d", i), FullName: "Farm", OVR: 70, Age: 24,
			Category: "MID", ClubID: farm.ClubID, SquadRole: models.RoleSquad,
		})
	}

	tm := &TournamentManager{
		Clubs:      map[string]*models.Club{parent.ClubID: parent, farm.ClubID: farm},
		ClubsList:  []*models.Club{parent, farm},
		SeasonName: "2026-27",
	}
	if n := tm.ArrangeLoansUnlocked(); n < 1 {
		t.Fatalf("expected a loan, moved=%d", n)
	}
	if !kid.OnLoan || kid.ClubID != farm.ClubID || kid.ParentClubID != parent.ClubID {
		t.Fatalf("loan state: on=%v club=%s parent=%s", kid.OnLoan, kid.ClubID, kid.ParentClubID)
	}
	if n := tm.ReturnLoansUnlocked(); n < 1 {
		t.Fatalf("expected a return, got %d", n)
	}
	if kid.OnLoan || kid.ClubID != parent.ClubID {
		t.Fatalf("return state: on=%v club=%s", kid.OnLoan, kid.ClubID)
	}
}

func TestLoanBuyClausesStayWithinValuationClamps(t *testing.T) {
	parent := &models.Club{
		ClubID: "BIG", ClubName: "Big Town", ShortName: "BIG", League: "Premier League",
		OverallTeamRating: 88, SquadAvgOVR: 84,
		Identity: models.ClubIdentity{FinancialPower: 80, Reputation: 80},
	}
	farm := &models.Club{
		ClubID: "SML", ClubName: "Small Town", ShortName: "SML", League: "Premier League",
		OverallTeamRating: 71, SquadAvgOVR: 70,
		Identity: models.ClubIdentity{FinancialPower: 80, Reputation: 60},
	}
	for i := 0; i < 24; i++ {
		parent.Squad = append(parent.Squad, &models.Player{
			PlayerID: fmt.Sprintf("SEN%02d", i), FullName: "Senior",
			OVR: 82, Age: 27, Category: "MID", ClubID: parent.ClubID, SquadRole: models.RoleImportant,
		})
	}
	kid := &models.Player{
		PlayerID: "KID2", FullName: "Clause Kid", OVR: 68, Age: 19, Category: "FWD",
		ClubID: parent.ClubID, SquadRole: models.RoleProspect, UniverseWonderkid: false,
	}
	parent.Squad = append(parent.Squad, kid)
	for i := 0; i < 18; i++ {
		farm.Squad = append(farm.Squad, &models.Player{
			PlayerID: fmt.Sprintf("FARM%02d", i), FullName: "Farm", OVR: 70, Age: 24,
			Category: "MID", ClubID: farm.ClubID, SquadRole: models.RoleSquad,
		})
	}
	tm := &TournamentManager{
		Clubs:      map[string]*models.Club{parent.ClubID: parent, farm.ClubID: farm},
		ClubsList:  []*models.Club{parent, farm},
		SeasonName: "2026-27",
	}
	if n := tm.ArrangeLoansUnlocked(); n < 1 {
		t.Fatalf("expected a loan, moved=%d", n)
	}
	if !kid.OnLoan {
		t.Fatal("expected the prospect to go on loan")
	}
	if kid.LoanBuyClauseEUR < 300_000 || kid.LoanBuyClauseEUR > 500_000_000 {
		t.Fatalf("buy clause %d outside valuation clamps", kid.LoanBuyClauseEUR)
	}
}

func TestLoanBuyClauseTriggersPermanentMove(t *testing.T) {
	parent := &models.Club{
		ClubID: "BIG", ClubName: "Big Town", ShortName: "BIG", League: "Premier League",
		OverallTeamRating: 88, SquadAvgOVR: 84,
		Identity: models.ClubIdentity{FinancialPower: 80, Reputation: 80},
		Finances: models.ClubFinances{Balance: 200_000_000, TransferBudget: 100_000_000, WageCap: 400_000_000},
	}
	farm := &models.Club{
		ClubID: "SML", ClubName: "Small Town", ShortName: "SML", League: "Premier League",
		OverallTeamRating: 71, SquadAvgOVR: 70,
		Identity: models.ClubIdentity{FinancialPower: 80, Reputation: 60},
		Finances: models.ClubFinances{Balance: 500_000_000, TransferBudget: 200_000_000, WageCap: 400_000_000},
	}
	for i := 0; i < 24; i++ {
		parent.Squad = append(parent.Squad, &models.Player{
			PlayerID: fmt.Sprintf("SEN%02d", i), FullName: "Senior",
			OVR: 82, Age: 27, Category: "MID", ClubID: parent.ClubID, SquadRole: models.RoleImportant,
			WageEUR: 50_000,
		})
	}
	kid := &models.Player{
		PlayerID: "KID3", FullName: "Bought Kid", OVR: 68, Age: 19, Category: "FWD",
		ClubID: parent.ClubID, SquadRole: models.RoleProspect, UniverseWonderkid: false,
		WageEUR: 10_000, TransferRequested: true,
	}
	parent.Squad = append(parent.Squad, kid)
	for i := 0; i < 18; i++ {
		farm.Squad = append(farm.Squad, &models.Player{
			PlayerID: fmt.Sprintf("FARM%02d", i), FullName: "Farm", OVR: 70, Age: 24,
			Category: "MID", ClubID: farm.ClubID, SquadRole: models.RoleSquad, WageEUR: 20_000,
		})
	}
	tm := &TournamentManager{
		Clubs:      map[string]*models.Club{parent.ClubID: parent, farm.ClubID: farm},
		ClubsList:  []*models.Club{parent, farm},
		SeasonName: "2026-27",
	}
	if n := tm.ArrangeLoansUnlocked(); n < 1 {
		t.Fatalf("expected a loan, moved=%d", n)
	}
	fee := kid.LoanBuyClauseEUR
	if fee <= 0 {
		t.Fatal("expected a buy clause on this loan")
	}
	parentBalance, farmBalance := parent.Finances.Balance, farm.Finances.Balance
	if n := tm.ReturnLoansUnlocked(); n != 0 {
		t.Fatalf("buyout should replace the return, returned=%d", n)
	}
	if kid.OnLoan || kid.ParentClubID != "" || kid.LoanBuyClauseEUR != 0 {
		t.Fatalf("buyout flags wrong: on=%v parent=%q clause=%d", kid.OnLoan, kid.ParentClubID, kid.LoanBuyClauseEUR)
	}
	if kid.ClubID != farm.ClubID {
		t.Fatalf("bought player should stay at %s, got %s", farm.ClubID, kid.ClubID)
	}
	if farm.Finances.Balance != farmBalance-fee || parent.Finances.Balance != parentBalance+fee {
		t.Fatalf("fee did not move: farm %d->%d parent %d->%d fee %d",
			farmBalance, farm.Finances.Balance, parentBalance, parent.Finances.Balance, fee)
	}
	// Exactly one squad holds the player: no duplicates.
	found := 0
	for _, club := range tm.ClubsList {
		for _, p := range club.Squad {
			if p.PlayerID == kid.PlayerID {
				found++
			}
		}
	}
	if found != 1 {
		t.Fatalf("player in %d squads, want 1", found)
	}
	if kid.ContractYears != 3 {
		t.Fatalf("bought player contract years=%d want 3", kid.ContractYears)
	}
}

func TestLoanBuyClauseRecordsPermanentTransfer(t *testing.T) {
	parent := &models.Club{
		ClubID: "BIG", ClubName: "Big Town", ShortName: "BIG", League: "Premier League",
		OverallTeamRating: 88, SquadAvgOVR: 84,
		Identity: models.ClubIdentity{FinancialPower: 80, Reputation: 80},
		Finances: models.ClubFinances{Balance: 200_000_000, TransferBudget: 100_000_000, WageCap: 400_000_000},
	}
	farm := &models.Club{
		ClubID: "SML", ClubName: "Small Town", ShortName: "SML", League: "Premier League",
		OverallTeamRating: 71, SquadAvgOVR: 70,
		Identity: models.ClubIdentity{FinancialPower: 80, Reputation: 60},
		Finances: models.ClubFinances{Balance: 500_000_000, TransferBudget: 200_000_000, WageCap: 400_000_000},
	}
	for i := 0; i < 24; i++ {
		parent.Squad = append(parent.Squad, &models.Player{
			PlayerID: fmt.Sprintf("SENB%02d", i), FullName: "Senior",
			OVR: 82, Age: 27, Category: "MID", ClubID: parent.ClubID, SquadRole: models.RoleImportant,
			WageEUR: 50_000,
		})
	}
	kid := &models.Player{
		PlayerID: "KIDBUY", FullName: "Bought Kid", OVR: 68, Age: 19, Category: "FWD",
		ClubID: parent.ClubID, SquadRole: models.RoleProspect, UniverseWonderkid: false,
		WageEUR: 10_000, TransferRequested: true, ContractYears: 2,
	}
	parent.Squad = append(parent.Squad, kid)
	for i := 0; i < 18; i++ {
		farm.Squad = append(farm.Squad, &models.Player{
			PlayerID: fmt.Sprintf("FARMB%02d", i), FullName: "Farm", OVR: 70, Age: 24,
			Category: "MID", ClubID: farm.ClubID, SquadRole: models.RoleSquad, WageEUR: 20_000,
		})
	}
	te := transfers.NewTransferEngine([]*models.Club{parent, farm}, nil, 1)
	tm := &TournamentManager{
		Clubs:          map[string]*models.Club{parent.ClubID: parent, farm.ClubID: farm},
		ClubsList:      []*models.Club{parent, farm},
		SeasonName:     "2026-27",
		TransferEngine: te,
	}
	if n := tm.ArrangeLoansUnlocked(); n < 1 {
		t.Fatalf("expected a loan, moved=%d", n)
	}
	if kid.LoanBuyClauseEUR <= 0 {
		t.Fatal("expected a buy clause on this loan")
	}
	if n := tm.ReturnLoansUnlocked(); n != 0 {
		t.Fatalf("buyout should replace the return, returned=%d", n)
	}
	if len(te.AllTimeTransfers) != 1 {
		t.Fatalf("buy clause should record one permanent transfer, got %d", len(te.AllTimeTransfers))
	}
	done := te.AllTimeTransfers[0]
	if done.PlayerID != kid.PlayerID || done.BuyerID != farm.ClubID || done.SellerID != parent.ClubID || done.FeeEUR <= 0 {
		t.Fatalf("recorded transfer wrong: %+v", done)
	}
	if kid.ClubID != farm.ClubID {
		t.Fatalf("player club=%s want %s", kid.ClubID, farm.ClubID)
	}
}

// Legacy 12-club careers run winter loans mid-season, so the legacy season
// reset must bring loanees home and wipe per-season tracking exactly like
// the world path (minutes, ratings, requests, dynamics).
func TestLegacyResetReturnsLoansAndWipesSeasonTracking(t *testing.T) {
	parent := &models.Club{
		ClubID: "BIG", ClubName: "Big Town", ShortName: "BIG", League: "Premier League",
		OverallTeamRating: 82, SquadAvgOVR: 80,
	}
	farm := &models.Club{
		ClubID: "SML", ClubName: "Small Town", ShortName: "SML", League: "Premier League",
		OverallTeamRating: 71, SquadAvgOVR: 70,
	}
	kid := &models.Player{
		PlayerID: "KIDR", FullName: "Return Kid", OVR: 68, Age: 19, Category: "FWD",
		ClubID: farm.ClubID, ParentClubID: parent.ClubID, SquadRole: models.RoleProspect,
		OnLoan: true, TransferRequested: true, Fitness: 60, Sharpness: 50,
		Appearances: 9, Goals: 2,
		CompetitionStats: map[string]*models.CompetitionSeasonStats{
			"super-league": {CompetitionID: "super-league", Appearances: 9, Minutes: 500},
		},
		RecentRatings: []float64{7.5, 8.0},
	}
	farm.Squad = []*models.Player{kid}
	for i := 0; i < 10; i++ {
		parent.Squad = append(parent.Squad, &models.Player{
			PlayerID: fmt.Sprintf("SENR%02d", i), FullName: "Senior", OVR: 80, Age: 27,
			Category: "MID", ClubID: parent.ClubID, SquadRole: models.RoleImportant,
		})
		farm.Squad = append(farm.Squad, &models.Player{
			PlayerID: fmt.Sprintf("FRMR%02d", i), FullName: "Farm", OVR: 70, Age: 24,
			Category: "MID", ClubID: farm.ClubID, SquadRole: models.RoleSquad,
		})
	}
	tm := &TournamentManager{
		Clubs:         map[string]*models.Club{parent.ClubID: parent, farm.ClubID: farm},
		ClubsList:     []*models.Club{parent, farm},
		SeasonName:    "2026-27",
		SeasonPhase:   "transfer_window",
		GrowthEngine:  growth.NewGrowthEngine(99),
		MaxMatchweeks: LeagueRounds,
	}
	out := tm.ResetNewSeason()
	if out["status"] != "success" {
		t.Fatalf("legacy reset failed: %v", out)
	}
	if kid.OnLoan || kid.ParentClubID != "" || kid.ClubID != parent.ClubID {
		t.Fatalf("loanee not home: on=%v parent=%q club=%q", kid.OnLoan, kid.ParentClubID, kid.ClubID)
	}
	if kid.CompetitionStats != nil || kid.RecentRatings != nil {
		t.Fatal("per-season tracking should reset with appearances")
	}
	if kid.TransferRequested || kid.Fitness != 80 || kid.Sharpness != 65 {
		t.Fatalf("dynamics/requests should reset: requested=%v fitness=%d sharpness=%d",
			kid.TransferRequested, kid.Fitness, kid.Sharpness)
	}
	found := false
	for _, p := range parent.Squad {
		if p.PlayerID == kid.PlayerID {
			found = true
		}
	}
	if !found {
		t.Fatal("returned loanee missing from parent squad")
	}
}

func TestLoanReturnToFullParentPreservesEveryPlayer(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		for _, rating := range []int{50, 90} {
			for _, size := range []int{20, models.MaxSeniorSquadSize} {
				t.Run(fmt.Sprintf("reverse_%v_rating_%d_size_%d", reverse, rating, size), func(t *testing.T) {
					parent := &models.Club{ClubID: "PARENT"}
					borrower := &models.Club{ClubID: "BORROWER"}
					for _, club := range []*models.Club{parent, borrower} {
						n := models.MaxSeniorSquadSize
						if club == borrower {
							n = size - 1
						}
						for i := 0; i < n; i++ {
							club.Squad = append(club.Squad, &models.Player{PlayerID: fmt.Sprintf("%s_%d", club.ClubID, i), ClubID: club.ClubID, OVR: 70})
						}
					}
					kid := &models.Player{PlayerID: "LOAN", ClubID: borrower.ClubID, ParentClubID: parent.ClubID, OnLoan: true, OVR: rating}
					borrower.Squad = append(borrower.Squad, kid)
					clubs := []*models.Club{parent, borrower}
					if reverse {
						clubs[0], clubs[1] = clubs[1], clubs[0]
					}
					tm := &TournamentManager{ClubsList: clubs, Clubs: map[string]*models.Club{parent.ClubID: parent, borrower.ClubID: borrower}}
					if got := tm.ReturnLoansUnlocked(); got != 0 {
						t.Fatalf("full parent reported %d returns", got)
					}
					seen := map[string]bool{}
					for _, club := range clubs {
						for _, p := range club.Squad {
							if seen[p.PlayerID] || p.ClubID != club.ClubID {
								t.Fatalf("invalid ownership for %s", p.PlayerID)
							}
							seen[p.PlayerID] = true
						}
					}
					if len(seen) != models.MaxSeniorSquadSize+size || !kid.OnLoan || kid.ParentClubID != parent.ClubID {
						t.Fatal("blocked return lost a player or erased the parent contract")
					}
					parent.Squad = parent.Squad[1:]
					if got := tm.ReturnLoansUnlocked(); got != 1 || kid.OnLoan || kid.ClubID != parent.ClubID {
						t.Fatalf("return did not resume after capacity became available: %d", got)
					}
				})
			}
		}
	}
}

func TestCanonicalWonderkidsAreNotLoaned(t *testing.T) {
	parent := &models.Club{
		ClubID: "BIG", ClubName: "Big Town", ShortName: "BIG", League: "Premier League",
		OverallTeamRating: 90, SquadAvgOVR: 86,
	}
	farm := &models.Club{
		ClubID: "SML", ClubName: "Small Town", ShortName: "SML", League: "Premier League",
		OverallTeamRating: 70, SquadAvgOVR: 68,
	}
	wk := &models.Player{
		PlayerID: "WK_Loan_Test", FullName: "Wonder Kid", OVR: 75, Age: 17, Category: "FWD",
		ClubID: parent.ClubID, SquadRole: models.RoleProspect, UniverseWonderkid: true,
	}
	parent.Squad = append(parent.Squad, wk)
	for i := 0; i < 24; i++ {
		parent.Squad = append(parent.Squad, &models.Player{PlayerID: fmt.Sprintf("S%02d", i), OVR: 84, Age: 26, Category: "MID", ClubID: parent.ClubID, SquadRole: models.RoleImportant})
	}
	tm := &TournamentManager{
		Clubs:     map[string]*models.Club{parent.ClubID: parent, farm.ClubID: farm},
		ClubsList: []*models.Club{parent, farm},
	}
	if n := tm.ArrangeLoansUnlocked(); n != 0 {
		t.Fatalf("wonderkid was loaned, moved=%d", n)
	}
	if wk.OnLoan {
		t.Fatal("canonical wonderkid must stay at the parent club")
	}
}
