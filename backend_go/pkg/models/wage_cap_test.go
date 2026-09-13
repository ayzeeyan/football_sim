package models

import (
	"encoding/json"
	"testing"
)

// The wage cap is structural: once set above the bill, later squad growth
// must not ratchet it upward. Otherwise every signing would raise its own
// ceiling and the constraint would be meaningless.
func TestRecalculateWageBillDoesNotRatchetCap(t *testing.T) {
	club := &Club{
		ClubID:   "CAP",
		ClubName: "Cap FC",
		Identity: ClubIdentity{Reputation: 70, FinancialPower: 70, BoardPatience: 60, AcademyQuality: 60, RecruitmentAmbition: 60, YouthPreference: 60, TransferAggressiveness: 60, SellingTendency: 40},
	}
	club.Identity = club.Identity.Clamp()
	for i := 0; i < 20; i++ {
		club.Squad = append(club.Squad, &Player{PlayerID: string(rune('A'+i)) + "1", OVR: 75, WageEUR: 50_000, ClubID: "CAP"})
	}
	club.RecalculateWageBill()
	formula := WageCapForIdentity(club.Identity)
	bill := club.WageBill()
	if club.Finances.WageCap < bill || club.Finances.WageCap < formula {
		t.Fatalf("initial cap=%d bill=%d formula=%d", club.Finances.WageCap, bill, formula)
	}
	frozen := club.Finances.WageCap
	// Simulate a signing that fits under the cap (bypassing transfer checks).
	club.Squad = append(club.Squad, &Player{PlayerID: "NEW1", OVR: 80, WageEUR: 80_000, ClubID: "CAP"})
	club.RecalculateWageBill()
	if club.Finances.WageCap != frozen {
		t.Fatalf("cap ratcheted %d -> %d as the bill grew", frozen, club.Finances.WageCap)
	}
	if !club.CanAffordWage(0) {
		t.Fatal("club should still fit its own bill under a frozen cap")
	}
}

// Legacy saves persisted before cap headroom can carry a cap below the
// committed bill. Unmarshalling must migrate it above the bill exactly once.
func TestUnmarshalMigratesLowWageCap(t *testing.T) {
	raw := `{"club_id":"MIG","club_name":"Mig FC","short_name":"MIG","league":"Premier League",
		"overall_team_rating":80,
		"identity":{"reputation":70,"historical_prestige":70,"financial_power":70,"board_patience":60,"academy_quality":60,"recruitment_ambition":60,"youth_preference":60,"transfer_aggressiveness":60,"selling_tendency":40},
		"finances":{"transfer_budget":10000000,"balance":20000000,"wage_budget":0,"wage_cap":1},
		"squad":[{"player_id":"P1","full_name":"Pro","position":"CM","ovr":78,"age":25,"wage_eur":100000,"club_id":"MIG","contract_years":3,"loyalty":60,"morale":70,"fitness":80,"sharpness":65,"composure":75}]}`
	var club Club
	if err := json.Unmarshal([]byte(raw), &club); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if club.Finances.WageCap < club.WageBill() {
		t.Fatalf("migrated cap=%d below bill=%d", club.Finances.WageCap, club.WageBill())
	}
	if !club.Finances.Valid() {
		t.Fatal("migrated finances should be valid")
	}
}
