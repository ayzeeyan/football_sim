package tournament

import (
	"math"
	"testing"
)

func TestValidateWorldStateAcceptsFreshUniverse(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	if err := tm.ValidateWorldState(); err != nil {
		t.Fatalf("fresh universe failed validation: %v", err)
	}
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
