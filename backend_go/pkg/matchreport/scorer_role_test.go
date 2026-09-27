package matchreport

import (
	"math/rand"
	"testing"

	"football_sim/pkg/models"
)

func TestScorerRoleAndFormWeightsArePlayerAgnostic(t *testing.T) {
	jhed := &models.Player{
		PlayerID: "WK_Jhed_Anthony_Guinita", FullName: "Jhed Anthony Guinita",
		Position: "CF", Category: "FWD", OVR: 75,
	}
	sameRoleDifferentIdentity := &models.Player{
		PlayerID: "P100", FullName: "Generic Center Forward",
		Position: "CF", Category: "FWD", OVR: 75,
	}
	winger := &models.Player{
		PlayerID: "P101", FullName: "Generic Winger",
		Position: "RW", Category: "FWD", OVR: 75,
	}

	jhedWeight := ScorerSelectionWeight(jhed, false, false)
	if got := ScorerSelectionWeight(sameRoleDifferentIdentity, false, false); got != jhedWeight {
		t.Fatalf("equal-role players received identity-dependent weights: Jhed=%v generic=%v", jhedWeight, got)
	}
	if jhedWeight <= ScorerSelectionWeight(winger, false, false) {
		t.Fatal("natural center-forward role should receive more goal attribution weight than an equal-rated winger")
	}

	inForm := &models.Player{
		PlayerID: "P102", FullName: "In-Form Striker", Position: "ST", Category: "FWD", OVR: 86,
		RecentRatings: []float64{8.4, 8.1, 8.6, 8.0},
	}
	if ScorerSelectionWeight(inForm, false, false) <= ScorerSelectionWeight(&models.Player{
		PlayerID: "P103", FullName: "Same Striker Out of Form", Position: "ST", Category: "FWD", OVR: 86,
		RecentRatings: []float64{5.8, 6.0, 5.9, 6.1},
	}, false, false) {
		t.Fatal("recent form should affect scorer selection for all players")
	}
}

func TestEliteStrikerCanClearFortyGoalAttributionsWithoutNamedBoost(t *testing.T) {
	star := &models.Player{
		PlayerID: "P200", FullName: "Elite Striker", Position: "ST", Category: "FWD", OVR: 99,
		SquadRole: models.RoleCrucial, RecentRatings: []float64{8.5, 8.7, 8.4, 8.6},
	}
	xi := []*models.Player{
		star,
		{PlayerID: "P201", Position: "LW", Category: "FWD", OVR: 88},
		{PlayerID: "P202", Position: "RW", Category: "FWD", OVR: 88},
		{PlayerID: "P203", Position: "CAM", Category: "MID", OVR: 88},
		{PlayerID: "P204", Position: "CM", Category: "MID", OVR: 88},
		{PlayerID: "P205", Position: "CM", Category: "MID", OVR: 88},
	}
	rng := rand.New(rand.NewSource(20260923))
	goals := 0
	const teamGoalAttributions = 120
	for i := 0; i < teamGoalAttributions; i++ {
		if got := PickScorer(xi, 60, false, rng); got == star {
			goals++
		}
	}
	if goals <= 40 {
		t.Fatalf("elite central striker received %d of %d goal attributions; want a plausible 40+ season path", goals, teamGoalAttributions)
	}
}
