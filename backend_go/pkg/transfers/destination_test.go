package transfers

import (
	"testing"

	"football_sim/pkg/models"
)

func TestStarRejectsWeakerDestination(t *testing.T) {
	star := &models.Player{PlayerID: "ST1", FullName: "Star", OVR: 88, SquadRole: models.RoleCrucial, Morale: 80}
	milan := &models.Club{ClubID: "SEA-MIL", ClubName: "Milan", ShortName: "MIL", League: "Serie A", OverallTeamRating: 86, Identity: models.ClubIdentity{Reputation: 88}}
	mid := &models.Club{ClubID: "FL1-REI", ClubName: "Reims", ShortName: "REI", League: "Ligue 1", OverallTeamRating: 72, Identity: models.ClubIdentity{Reputation: 55}}
	if PlayerAcceptsDestination(star, milan, mid) {
		t.Fatal("star should reject a clear step down")
	}
	united := &models.Club{ClubID: "EPL-MUN", ClubName: "Manchester United", ShortName: "MUN", League: "Premier League", OverallTeamRating: 87, Identity: models.ClubIdentity{Reputation: 90}}
	if !PlayerAcceptsDestination(star, milan, united) {
		t.Fatal("star should consider a comparable Premier League move")
	}
	star.TransferRequested = true
	if !PlayerAcceptsDestination(star, milan, mid) {
		t.Fatal("a transfer request should make the player willing to leave")
	}
}
