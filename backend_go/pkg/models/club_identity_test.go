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

func TestExplicitZeroIdentityJSONIsPreserved(t *testing.T) {
	const payload = `{"club_id":"EPL-ARS","overall_team_rating":87,"identity":{"reputation":0,"historical_prestige":0,"financial_power":0,"board_patience":0,"academy_quality":0,"recruitment_ambition":0,"youth_preference":0,"transfer_aggressiveness":0,"selling_tendency":0},"squad":[]}`

	var club Club
	if err := json.Unmarshal([]byte(payload), &club); err != nil {
		t.Fatal(err)
	}
	if club.Identity.IsZero() {
		t.Fatal("explicit zero identity was treated as missing")
	}
	if club.Identity.Reputation != 0 || club.Identity.HistoricalPrestige != 0 ||
		club.Identity.FinancialPower != 0 || club.Identity.BoardPatience != 0 ||
		club.Identity.AcademyQuality != 0 || club.Identity.RecruitmentAmbition != 0 ||
		club.Identity.YouthPreference != 0 || club.Identity.TransferAggressiveness != 0 ||
		club.Identity.SellingTendency != 0 {
		t.Fatalf("explicit zero identity changed: %+v", club.Identity)
	}

	encoded, err := json.Marshal(club)
	if err != nil {
		t.Fatal(err)
	}
	var roundTrip Club
	if err := json.Unmarshal(encoded, &roundTrip); err != nil {
		t.Fatal(err)
	}
	if roundTrip.Identity.IsZero() || roundTrip.Identity.Reputation != 0 || roundTrip.Identity.SellingTendency != 0 {
		t.Fatalf("explicit zero identity did not round-trip: %+v", roundTrip.Identity)
	}
}

func TestNullIdentityUsesLegacyDefaults(t *testing.T) {
	var club Club
	if err := json.Unmarshal([]byte(`{"club_id":"EPL-ARS","overall_team_rating":87,"identity":null,"squad":[]}`), &club); err != nil {
		t.Fatal(err)
	}
	want := DefaultClubIdentity("EPL-ARS", 87)
	if club.Identity != want {
		t.Fatalf("null identity got %+v, want default %+v", club.Identity, want)
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
