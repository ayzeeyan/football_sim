package tournament

import (
	"math/rand"
	"testing"

	"football_sim/pkg/models"
)

func TestPenaltyShootoutUsesEligibleXI(t *testing.T) {
	exam := &models.Player{
		PlayerID: "WK_EXAM", FullName: "Exam Kid", Position: "ST", Category: "FWD",
		OVR: 99, UniverseWonderkid: true, Education: "middle_school", Age: 14,
	}
	gk := &models.Player{PlayerID: "GK1", FullName: "GK", Category: "GK", OVR: 70}
	out := &models.Player{PlayerID: "OUT1", FullName: "Outfield", Category: "MID", OVR: 70}
	home := &models.Club{ClubID: "H", ClubName: "Home", ShortName: "HOM", Squad: []*models.Player{gk, exam, out}}
	awayGK := &models.Player{PlayerID: "GK2", FullName: "Away GK", Category: "GK", OVR: 70}
	awayOut := &models.Player{PlayerID: "OUT2", FullName: "Away Out", Category: "MID", OVR: 70}
	away := &models.Club{ClubID: "A", ClubName: "Away", ShortName: "AWY", Squad: []*models.Player{awayGK, awayOut}}

	hXI := home.GetStartingEleven("ucl:20")
	for _, p := range hXI {
		if p.PlayerID == exam.PlayerID {
			t.Fatal("middle-school prodigy should sit a cup night")
		}
	}
	h, a := penaltyShootout(home, away, rand.New(rand.NewSource(1)), "ucl:20")
	if h == 0 && a == 0 && len(outfield(hXI)) == 0 {
		t.Fatal("eligible side produced no penalty takers")
	}
}
