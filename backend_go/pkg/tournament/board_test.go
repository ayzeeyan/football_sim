package tournament

import (
	"testing"

	"football_sim/pkg/models"
)

func TestAssignBoardExpectationsSetsLeagueRelativeTargets(t *testing.T) {
	elite := &models.Club{ClubID: "A", League: "Premier League", OverallTeamRating: 90, Identity: models.ClubIdentity{Reputation: 95, FinancialPower: 90}}
	mid := &models.Club{ClubID: "B", League: "Premier League", OverallTeamRating: 78, Identity: models.ClubIdentity{Reputation: 60, FinancialPower: 55}}
	scrap := &models.Club{ClubID: "C", League: "Premier League", OverallTeamRating: 70, Identity: models.ClubIdentity{Reputation: 40, FinancialPower: 35}}
	tm := &TournamentManager{ClubsList: []*models.Club{elite, mid, scrap}}
	tm.AssignBoardExpectationsUnlocked()
	if elite.ExpectedFinish != 1 || elite.BoardObjective != "Title challenge" {
		t.Fatalf("elite board=%s finish=%d", elite.BoardObjective, elite.ExpectedFinish)
	}
	if scrap.ExpectedFinish != 3 || scrap.BoardObjective != "Survival" {
		t.Fatalf("scrap board=%s finish=%d", scrap.BoardObjective, scrap.ExpectedFinish)
	}
	if mid.ExpectedFinish != 2 {
		t.Fatalf("mid finish=%d", mid.ExpectedFinish)
	}
}
