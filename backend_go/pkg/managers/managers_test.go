package managers

import (
	"math/rand"
	"testing"

	"football_sim/pkg/models"
)

func TestTacticalArchetypesMap(t *testing.T) {
	expected := []string{"high_press", "possession", "low_block", "free_flowing"}
	for _, key := range expected {
		arch, ok := TacticalArchetypes[key]
		if !ok {
			t.Fatalf("expected archetype %s to exist in TacticalArchetypes", key)
		}
		if arch.Title == "" || arch.Badge == "" || arch.Beats == "" {
			t.Errorf("archetype %s has missing metadata: %+v", key, arch)
		}
	}
}

func TestTacticEdge(t *testing.T) {
	tests := []struct {
		home     string
		away     string
		expected float64
	}{
		{"high_press", "possession", 0.16},
		{"possession", "high_press", -0.12},
		{"possession", "low_block", 0.16},
		{"low_block", "possession", -0.12},
		{"low_block", "free_flowing", 0.16},
		{"free_flowing", "low_block", -0.12},
		{"free_flowing", "high_press", 0.16},
		{"high_press", "free_flowing", -0.12},
		{"high_press", "high_press", 0.0},
		{"", "possession", 0.0},
		{"press", "possession", 0.16}, // canonical mapping
	}

	for _, tc := range tests {
		got := TacticEdge(tc.home, tc.away)
		if got != tc.expected {
			t.Errorf("TacticEdge(%q, %q) = %f; want %f", tc.home, tc.away, got, tc.expected)
		}
	}
}

func TestBuildManagers(t *testing.T) {
	clubs := []*models.Club{
		{ClubID: "LAL-RMA", OverallTeamRating: 86},
		{ClubID: "LAL-BAR", OverallTeamRating: 84},
		{ClubID: "BUN-BAY", OverallTeamRating: 85},
		{ClubID: "EPL-ARS", OverallTeamRating: 83},
	}

	mgrs := BuildManagers(clubs)
	if len(mgrs) != len(clubs) {
		t.Fatalf("expected %d managers, got %d", len(clubs), len(mgrs))
	}

	rma := mgrs["LAL-RMA"]
	if rma.Style != "free_flowing" || rma.Focus != "stars" {
		t.Errorf("expected LAL-RMA to have free_flowing/stars, got %s/%s", rma.Style, rma.Focus)
	}
	if rma.BudgetEur <= 0 {
		t.Errorf("expected positive budget, got %d", rma.BudgetEur)
	}
	if rma.DogmaTitle() != "The Free-Flowing Attacker" {
		t.Errorf("unexpected dogma title: %s", rma.DogmaTitle())
	}
}

func TestAppointManager(t *testing.T) {
	clubs := []*models.Club{
		{ClubID: "LAL-RMA", OverallTeamRating: 86},
	}
	mgrs := BuildManagers(clubs)
	oldMgr := mgrs["LAL-RMA"]

	rng := rand.New(rand.NewSource(12345))
	oldReplaced, newMgr := AppointManager(mgrs, clubs[0], rng)

	if oldReplaced.Name != oldMgr.Name {
		t.Errorf("expected replaced manager to be %s, got %s", oldMgr.Name, oldReplaced.Name)
	}
	if newMgr.Name == oldMgr.Name {
		t.Errorf("expected new manager to have different name, got same: %s", newMgr.Name)
	}
	if newMgr.BudgetEur != oldMgr.BudgetEur {
		t.Errorf("expected budget to carry over (%d), got %d", oldMgr.BudgetEur, newMgr.BudgetEur)
	}
	if mgrs["LAL-RMA"].Name != newMgr.Name {
		t.Errorf("expected map to be updated with new manager %s", newMgr.Name)
	}
}

func TestManagerAffordabilityAndWeakestLine(t *testing.T) {
	club := &models.Club{
		ClubID:            "TEST-FC",
		OverallTeamRating: 80,
		Squad: []*models.Player{
			{PlayerID: "D1", Category: "DEF", OVR: 82, WageEUR: 50000},
			{PlayerID: "D2", Category: "DEF", OVR: 84, WageEUR: 60000},
			{PlayerID: "M1", Category: "MID", OVR: 70, WageEUR: 20000},
			{PlayerID: "M2", Category: "MID", OVR: 72, WageEUR: 25000},
			{PlayerID: "F1", Category: "FWD", OVR: 85, WageEUR: 70000},
			{PlayerID: "F2", Category: "FWD", OVR: 88, WageEUR: 80000},
		},
	}
	mgr := &ManagerProfile{
		ClubID:    "TEST-FC",
		Name:      "Test Manager",
		BudgetEur: 50_000_000,
	}

	weak := mgr.WeakestLine(club)
	if weak != "MID" {
		t.Errorf("expected weakest line to be MID, got %s", weak)
	}

	// Wage cap = 150M + (80-78)*25M = 200M
	// Current wage bill = (50+60+20+25+70+80)k * 52 = 305k * 52 = 15.86M
	if !mgr.CanAfford(club, 20_000_000, 100_000) {
		t.Errorf("manager should be able to afford 20M fee and 100k wage")
	}
	if mgr.CanAfford(club, 60_000_000, 100_000) {
		t.Errorf("manager should NOT afford fee higher than 50M budget")
	}
}
