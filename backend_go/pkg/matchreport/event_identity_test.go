package matchreport

import (
	"math/rand"
	"testing"

	"football_sim/pkg/models"
)

func TestAssembleReportEnrichesInstantEventIdentityAndClubMetadata(t *testing.T) {
	home := &models.Player{PlayerID: "HOME-9", FullName: "Home Striker", ClubID: "HOME", Position: "ST", Category: "FWD", OVR: 85}
	away := &models.Player{PlayerID: "AWAY-8", FullName: "Away Midfielder", ClubID: "AWAY", Position: "CM", Category: "MID", OVR: 84}

	report := AssembleReport(InstantPayload{
		HomeGoals:    1,
		HomeClubName: "Home FC",
		AwayClubName: "Away FC",
		HomeXI:       []*models.Player{home},
		AwayXI:       []*models.Player{away},
		Events: []MatchEventItem{
			{Minute: 10, Seq: 1, Type: "goal", Side: "home", Scorer: &MiniPlayer{PlayerID: home.PlayerID, FullName: home.FullName}},
			{Minute: 70, Seq: 2, Type: "yellow", Side: "away", Player: &MiniPlayer{PlayerID: away.PlayerID, FullName: away.FullName}},
		},
	}, "instant", rand.New(rand.NewSource(1)))

	if len(report.Events) != 2 {
		t.Fatalf("events=%d, want 2", len(report.Events))
	}
	for _, event := range report.Events {
		wantPlayer, wantClubID, wantClubName := home, "HOME", "Home FC"
		if event.Side == "away" {
			wantPlayer, wantClubID, wantClubName = away, "AWAY", "Away FC"
		}
		if event.PlayerID != wantPlayer.PlayerID || event.PlayerName != wantPlayer.FullName {
			t.Fatalf("%s identity=%q/%q, want %q/%q", event.Type, event.PlayerID, event.PlayerName, wantPlayer.PlayerID, wantPlayer.FullName)
		}
		if event.ClubID != wantClubID || event.ClubName != wantClubName {
			t.Fatalf("%s club=%q/%q, want %q/%q", event.Type, event.ClubID, event.ClubName, wantClubID, wantClubName)
		}
	}
}
