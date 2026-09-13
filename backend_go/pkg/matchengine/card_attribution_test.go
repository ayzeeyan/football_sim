package matchengine

import (
	"math/rand"
	"testing"

	"football_sim/pkg/models"
)

func TestMaybeBookPlayerHonoursExplicitDefendingSide(t *testing.T) {
	defender := &models.Player{PlayerID: "DEF", FullName: "Defender", ClubID: "HOME", Category: "DEF", Position: "CB", OVR: 80}
	home := &models.Club{ClubID: "HOME", ClubName: "Home FC"}
	away := &models.Club{ClubID: "AWAY", ClubName: "Away FC"}
	for seed := int64(0); seed < 600; seed++ {
		e := &LiveMatchEngine{
			RNG: rand.New(rand.NewSource(seed)), Weather: "clear", CurrentMinute: 37,
			HomeClub: home, AwayClub: away, PossessionTeam: "away", Bookings: map[string]int{},
		}
		e.MaybeBookPlayer([]*models.Player{defender}, "home")
		if len(e.Events) == 0 {
			continue
		}
		event := e.Events[0]
		if event.Side != "home" || event.ClubID != "HOME" || event.ClubName != "Home FC" {
			t.Fatalf("booking attributed to %q/%q (%q), want home/HOME/Home FC", event.Side, event.ClubID, event.ClubName)
		}
		if event.PlayerID != defender.PlayerID || event.PlayerName != defender.FullName {
			t.Fatalf("booking player metadata = %q/%q, want %q/%q", event.PlayerID, event.PlayerName, defender.PlayerID, defender.FullName)
		}
		return
	}
	t.Fatal("no booking was generated in deterministic seed range")
}
