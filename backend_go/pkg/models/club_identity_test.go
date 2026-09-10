package models

import (
	"encoding/json"
	"testing"
)

func TestClubIdentityPresetsAreDistinctAndBounded(t *testing.T) {
	real := DefaultClubIdentity("LAL-RMA", 90)
	dortmund := DefaultClubIdentity("BUN-DOR", 84)
	if real == dortmund {
		t.Fatal("expected distinct club identity presets")
	}
	for name, id := range map[string]ClubIdentity{"real": real, "dortmund": dortmund} {
		vals := []int{id.Reputation, id.HistoricalPrestige, id.FinancialPower, id.BoardPatience, id.AcademyQuality, id.RecruitmentAmbition, id.YouthPreference, id.TransferAggressiveness, id.SellingTendency}
		for _, v := range vals {
			if v < ClubRatingMin || v > ClubRatingMax {
				t.Fatalf("%s identity contains out-of-range value %d", name, v)
			}
		}
	}
}

func TestClubIdentityClamp(t *testing.T) {
	got := (ClubIdentity{Reputation: 150, HistoricalPrestige: -5, FinancialPower: 101, BoardPatience: -1}).Clamp()
	if got.Reputation != 100 || got.HistoricalPrestige != 0 || got.FinancialPower != 100 || got.BoardPatience != 0 {
		t.Fatalf("unexpected clamp result: %+v", got)
	}
}

func TestLegacyClubJSONGetsIdentityAndFinances(t *testing.T) {
	var club Club
	if err := json.Unmarshal([]byte(`{"club_id":"EPL-ARS","overall_team_rating":87,"squad":[]}`), &club); err != nil {
		t.Fatal(err)
	}
	if club.Identity.IsZero() {
		t.Fatal("legacy club did not receive identity defaults")
	}
	if club.Finances.TransferBudget <= 0 || club.Finances.Balance <= 0 {
		t.Fatalf("legacy club did not receive sane finances: %+v", club.Finances)
	}
}

func TestPersistedZeroBudgetIsNotRefilled(t *testing.T) {
	payload := `{"club_id":"EPL-ARS","identity":{"reputation":89,"historical_prestige":91,"financial_power":91,"board_patience":62,"academy_quality":92,"recruitment_ambition":93,"youth_preference":91,"transfer_aggressiveness":80,"selling_tendency":25},"finances":{"transfer_budget":0,"balance":10000000},"squad":[]}`
	var club Club
	if err := json.Unmarshal([]byte(payload), &club); err != nil {
		t.Fatal(err)
	}
	if club.Finances.TransferBudget != 0 {
		t.Fatalf("persisted zero warchest was refilled to %d", club.Finances.TransferBudget)
	}
}
