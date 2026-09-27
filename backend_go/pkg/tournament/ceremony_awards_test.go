package tournament

import (
	"testing"

	"football_sim/pkg/models"
)

func TestAwardsUseCompetitionStatsAndAddRoleCategories(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	clubA, clubB := tm.ClubsList[0], tm.ClubsList[1]
	clubA.League = "Premier League"
	clubB.League = "Premier League"

	youngScorer := &models.Player{
		PlayerID: "AWARD_YOUNG", FullName: "Young Scorer", Position: "ST", Category: "FWD",
		OVR: 82, Age: 20, UniverseWonderkid: true, ClubID: clubA.ClubID, Goals: 14, Assists: 3, Appearances: 24,
		CompetitionStats: map[string]*models.CompetitionSeasonStats{
			"premier-league": {CompetitionID: "premier-league", Appearances: 24, Goals: 9, Assists: 3},
			"fa-cup":         {CompetitionID: "fa-cup", Appearances: 4, Goals: 1, Assists: 0},
		},
	}
	keeper := &models.Player{
		PlayerID: "AWARD_KEEPER", FullName: "Clean Sheet Keeper", Position: "GK", Category: "GK",
		OVR: 84, Age: 27, ClubID: clubA.ClubID, Appearances: 20, CleanSheets: 12,
	}
	cupScorer := &models.Player{
		PlayerID: "AWARD_CUP", FullName: "Cup Scorer", Position: "ST", Category: "FWD",
		OVR: 80, Age: 29, ClubID: clubB.ClubID, Goals: 20, Appearances: 25,
		CompetitionStats: map[string]*models.CompetitionSeasonStats{
			"premier-league": {CompetitionID: "premier-league", Appearances: 25, Goals: 5},
			"fa-cup":         {CompetitionID: "fa-cup", Appearances: 7, Goals: 6, Assists: 1},
		},
	}
	clubA.Squad = []*models.Player{youngScorer, keeper}
	clubB.Squad = []*models.Player{cupScorer}
	tm.ClubsList = []*models.Club{clubA, clubB}
	tm.Clubs = map[string]*models.Club{clubA.ClubID: clubA, clubB.ClubID: clubB}
	tm.World = &EuropeanWorld{
		Competitions: map[string]*Competition{
			"premier-league": {ID: "premier-league", Name: "Premier League", Kind: CompetitionLeague, ParticipantIDs: []string{clubA.ClubID, clubB.ClubID}},
			"fa-cup":         {ID: "fa-cup", Name: "FA Cup", Kind: CompetitionDomestic, ParticipantIDs: []string{clubA.ClubID, clubB.ClubID}},
		},
		CompetitionOrder: []string{"premier-league", "fa-cup"},
	}

	ceremony := tm.awardsCeremonyUnlocked()
	categories, ok := ceremony["categories"].([]map[string]interface{})
	if !ok {
		t.Fatalf("categories have unexpected type %T", ceremony["categories"])
	}
	byKey := make(map[string]map[string]interface{}, len(categories))
	for _, category := range categories {
		key, _ := category["key"].(string)
		byKey[key] = category
	}
	if got := byKey["young_player"]["winner_id"]; got != youngScorer.PlayerID {
		t.Fatalf("young player winner=%v, want %s", got, youngScorer.PlayerID)
	}
	if got := byKey["golden_glove"]["winner_id"]; got != keeper.PlayerID {
		t.Fatalf("Golden Glove winner=%v, want %s", got, keeper.PlayerID)
	}
	if got := byKey["premier-league-golden-boot"]["winner_id"]; got != youngScorer.PlayerID {
		t.Fatalf("league scorer winner=%v, want %s from league-only totals", got, youngScorer.PlayerID)
	}
	if got := byKey["fa-cup-top-scorer"]["winner_id"]; got != cupScorer.PlayerID {
		t.Fatalf("cup scorer winner=%v, want %s from cup-only totals", got, cupScorer.PlayerID)
	}
	if byKey["fa-cup-top-scorer"]["competition_id"] != "fa-cup" {
		t.Fatalf("cup award missing competition_id: %+v", byKey["fa-cup-top-scorer"])
	}
}

func TestWorldBallonDorKeepsTenPointDomesticChampionBonus(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	championClub, runnerUpClub := tm.ClubsList[0], tm.ClubsList[1]
	championClub.League = "Premier League"
	runnerUpClub.League = "Premier League"
	championClub.Points = 90
	runnerUpClub.Points = 88
	champion := &models.Player{PlayerID: "BONUS_CHAMP", OVR: 80, Age: 27, ClubID: championClub.ClubID, Goals: 10, Assists: 5, Appearances: 30}
	runnerUp := &models.Player{PlayerID: "BONUS_RUNNER", OVR: 80, Age: 27, ClubID: runnerUpClub.ClubID, Goals: 10, Assists: 5, Appearances: 30}
	championClub.Squad = []*models.Player{champion}
	runnerUpClub.Squad = []*models.Player{runnerUp}
	tm.ClubsList = []*models.Club{championClub, runnerUpClub}
	tm.Clubs = map[string]*models.Club{championClub.ClubID: championClub, runnerUpClub.ClubID: runnerUpClub}
	tm.World = &EuropeanWorld{Competitions: map[string]*Competition{
		"premier-league": {ID: "premier-league", Name: "Premier League", Kind: CompetitionLeague, ParticipantIDs: []string{championClub.ClubID, runnerUpClub.ClubID}},
	}}

	difference := tm.ballonScoreUnlocked(champion) - tm.ballonScoreUnlocked(runnerUp)
	if difference < 10 {
		t.Fatalf("domestic champion should receive explicit +10 Ballon d'Or bonus; score difference=%v", difference)
	}
}
