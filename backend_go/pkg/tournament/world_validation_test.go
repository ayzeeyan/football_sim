package tournament

import (
	"math"
	"strings"
	"testing"

	"football_sim/pkg/models"
)

func TestValidateWorldStateAcceptsFreshUniverse(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	if err := tm.ValidateWorldState(); err != nil {
		t.Fatalf("fresh universe failed validation: %v", err)
	}
}

func TestValidateEuropeanWorldRejectsCareerHealthCorruption(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	club := tm.ClubsList[0]
	player := club.Squad[0]
	assertRejected := func(fragment string) {
		t.Helper()
		err := tm.ValidateWorldState()
		if err == nil || !strings.Contains(err.Error(), fragment) {
			t.Fatalf("expected validation error containing %q, got %v", fragment, err)
		}
	}

	oldContract := player.ContractYears
	player.ContractYears = 0
	if err := tm.ValidateWorldState(); err != nil {
		t.Fatalf("expired contract length 0 must round-trip, got %v", err)
	}
	player.ContractYears = -1
	assertRejected("contract length")
	player.ContractYears = oldContract

	oldValue := player.MarketValueEUR
	player.MarketValueEUR = models.MinPlayerValueEUR - 1
	assertRejected("valuation")
	player.MarketValueEUR = oldValue

	oldMorale := player.Morale
	player.Morale = 101
	assertRejected("dynamics")
	player.Morale = oldMorale

	oldSquad := club.Squad
	club.Squad = append([]*models.Player(nil), oldSquad...)
	for len(club.Squad) <= models.MaxSeniorSquadSize {
		club.Squad = append(club.Squad, &models.Player{PlayerID: "OVER_CAP", ClubID: club.ClubID})
	}
	assertRejected("above the")
	club.Squad = oldSquad

	manager := tm.Managers[club.ClubID]
	delete(tm.Managers, club.ClubID)
	assertRejected("active managers")
	tm.Managers[club.ClubID] = manager

	player.OnLoan, player.ParentClubID = true, "UNKNOWN"
	assertRejected("unknown parent club")
	player.OnLoan, player.ParentClubID = false, ""
}

func TestValidateWorldStateRejectsUnknownPhase(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	tm.SeasonPhase = "corrupted_phase"
	if err := tm.ValidateWorldState(); err == nil {
		t.Fatal("expected unknown phase validation error")
	}
}

func TestValidateWorldStateRejectsDuplicatePlayerID(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	if len(tm.ClubsList[0].Squad) == 0 || len(tm.ClubsList[1].Squad) == 0 {
		t.Fatal("test universe missing squad players")
	}
	tm.ClubsList[1].Squad[0].PlayerID = tm.ClubsList[0].Squad[0].PlayerID
	if err := tm.ValidateWorldState(); err == nil {
		t.Fatal("expected duplicate player id validation error")
	}
}

func TestValidateWorldStateRejectsImpossibleFixture(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	if len(tm.Fixtures) == 0 {
		t.Fatal("test universe missing fixtures")
	}
	tm.Fixtures[0].AwayID = tm.Fixtures[0].HomeID
	if err := tm.ValidateWorldState(); err == nil {
		t.Fatal("expected identical home/away validation error")
	}
}

func TestValidateWorldStateRejectsNegativeStats(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	if len(tm.ClubsList[0].Squad) == 0 {
		t.Fatal("test universe missing squad players")
	}
	tm.ClubsList[0].Squad[0].Goals = -1
	if err := tm.ValidateWorldState(); err == nil {
		t.Fatal("expected negative player statistics validation error")
	}
}

func TestValidateWorldStateRejectsNonFiniteBiometrics(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	if tm.GrowthEngine == nil || len(tm.GrowthEngine.Biometrics) == 0 {
		t.Skip("test universe has no biometric profiles")
	}
	for _, bio := range tm.GrowthEngine.Biometrics {
		bio.CurrentHeightCM = math.NaN()
		break
	}
	if err := tm.ValidateWorldState(); err == nil {
		t.Fatal("expected NaN biometric validation error")
	}
}
