package matchreport

import (
	"math/rand"
	"testing"

	"football_sim/pkg/models"
)

func TestPacket4PickScorerRequiresExplicitBigGameContext(t *testing.T) {
	wonderkid := &models.Player{
		PlayerID:          "WK",
		FullName:          "Big Game Kid",
		Category:          "FWD",
		OVR:               76,
		UniverseWonderkid: true,
		Personality:       "big_game_performer",
	}
	plain := &models.Player{PlayerID: "PLAIN", FullName: "Plain Kid", Category: "FWD", OVR: 76}
	xi := []*models.Player{wonderkid, plain}

	count := func(explicit ...bool) int {
		rng := rand.New(rand.NewSource(4127))
		picked := 0
		for i := 0; i < 10000; i++ {
			if PickScorer(xi, 80, true, rng, explicit...) == wonderkid {
				picked++
			}
		}
		return picked
	}

	neutral := count()
	neutralExplicit := count(false)
	bigGame := count(true)
	if neutral != neutralExplicit {
		t.Fatalf("omitted and explicit-neutral contexts diverged: %d != %d", neutral, neutralExplicit)
	}
	if bigGame <= neutral+200 {
		t.Fatalf("explicit big-game context did not apply scorer weight: neutral=%d big_game=%d", neutral, bigGame)
	}
}
