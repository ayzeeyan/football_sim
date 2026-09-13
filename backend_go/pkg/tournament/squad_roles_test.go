package tournament

import (
	"testing"

	"football_sim/pkg/matchreport"
	"football_sim/pkg/models"
)

func TestAssignClubSquadRolesRanksByOVR(t *testing.T) {
	club := &models.Club{
		ClubID: "TEST",
		Squad: []*models.Player{
			{PlayerID: "A", FullName: "Star", OVR: 90, Age: 28},
			{PlayerID: "B", FullName: "Kid", OVR: 68, Age: 17, UniverseWonderkid: true},
			{PlayerID: "C", FullName: "Vet", OVR: 82, Age: 31},
			{PlayerID: "D", FullName: "Depth", OVR: 72, Age: 24},
		},
	}
	assignClubSquadRoles(club)
	byID := map[string]string{}
	for _, p := range club.Squad {
		byID[p.PlayerID] = p.SquadRole
		if p.Morale < 1 || p.Fitness < 1 || p.Sharpness < 1 {
			t.Fatalf("%s missing dynamics", p.PlayerID)
		}
	}
	if byID["A"] != models.RoleCrucial {
		t.Fatalf("star role=%s want Crucial", byID["A"])
	}
	if byID["B"] != models.RoleProspect {
		t.Fatalf("wonderkid role=%s want Prospect", byID["B"])
	}
}

func TestApplyPlayerMatchStatsRecordsCompetitionTotals(t *testing.T) {
	home := &models.Club{ClubID: "H", Squad: []*models.Player{{PlayerID: "P1", FullName: "Scorer", OVR: 80, Morale: 70, Fitness: 80, Sharpness: 60}}}
	away := &models.Club{ClubID: "A", Squad: []*models.Player{{PlayerID: "P2", FullName: "Other", OVR: 78, Morale: 70, Fitness: 80, Sharpness: 60}}}
	rating := 7.2
	report := &matchreport.MatchReport{
		HomeGoals: 1,
		AwayGoals: 0,
		HomeXI: []matchreport.MatchPlayerRow{
			{PlayerID: home.Squad[0].PlayerID, Played: true, Minutes: 90, Rating: &rating, MatchGoals: 1},
		},
		AwayXI: []matchreport.MatchPlayerRow{
			{PlayerID: away.Squad[0].PlayerID, Played: true, Minutes: 90, Rating: &rating},
		},
	}
	ApplyPlayerMatchStatsInCompetition(home, away, report, "premier-league")
	stats := home.Squad[0].CompetitionStats["premier-league"]
	if stats == nil || stats.Appearances != 1 {
		t.Fatalf("competition stats missing: %+v", stats)
	}
	if home.Squad[0].Appearances != 1 {
		t.Fatalf("season appearances=%d", home.Squad[0].Appearances)
	}
}
