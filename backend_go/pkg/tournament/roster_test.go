package tournament

import (
	"fmt"
	"testing"

	"football_sim/pkg/models"
)

func TestEnforceRosterCapsMovesOverflowWithoutTouchingWonderkids(t *testing.T) {
	inter := &models.Club{ClubID: "SEA-INT", ClubName: "Inter", ShortName: "INT", League: "Serie A"}
	monza := &models.Club{ClubID: "SEA-MON", ClubName: "Monza", ShortName: "MON", League: "Serie A"}
	wk := &models.Player{PlayerID: "WK_Izyan_Levin_Bantol", FullName: "Izyan", UniverseWonderkid: true, ClubID: inter.ClubID, OVR: 60, Age: 20}
	inter.Squad = []*models.Player{wk}
	inter.CaptainID = wk.PlayerID
	for i := 0; i < models.MaxSeniorSquadSize+1; i++ {
		inter.Squad = append(inter.Squad, &models.Player{
			PlayerID: fmt.Sprintf("INT_%02d", i), ClubID: inter.ClubID, OVR: 50 + i%10, Age: 24,
		})
	}
	monza.Squad = []*models.Player{{PlayerID: "MON_01", ClubID: monza.ClubID, OVR: 70, Age: 26}}
	tm := &TournamentManager{
		Clubs:     map[string]*models.Club{inter.ClubID: inter, monza.ClubID: monza},
		ClubsList: []*models.Club{inter, monza},
	}
	tm.EnforceRosterCapsUnlocked()
	if len(inter.Squad) > models.MaxSeniorSquadSize {
		t.Fatalf("Inter still over cap: %d", len(inter.Squad))
	}
	foundWK := false
	for _, p := range inter.Squad {
		if p.PlayerID == wk.PlayerID {
			foundWK = true
		}
	}
	if !foundWK {
		t.Fatal("canonical wonderkid was moved or dropped")
	}
	if len(monza.Squad) <= 1 {
		t.Fatal("expected overflow to land at a club with spare slots")
	}
}
