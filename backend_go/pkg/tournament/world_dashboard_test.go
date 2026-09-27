package tournament

import (
	"testing"

	"football_sim/pkg/models"
	"football_sim/pkg/transfers"
)

func TestWorldDashboardExposesFiveLeagueLeadersAndEurope(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	dash := tm.WorldDashboard()
	if dash["world"] != true {
		t.Fatalf("dashboard world=%v", dash["world"])
	}
	leaders, _ := dash["league_leaders"].([]map[string]interface{})
	if len(leaders) != 5 {
		t.Fatalf("league leaders=%d want 5", len(leaders))
	}
	seen := map[string]bool{}
	for _, row := range leaders {
		id, _ := row["competition_id"].(string)
		seen[id] = true
		if row["club_id"] == nil || row["club_name"] == nil {
			t.Fatalf("leader missing club: %v", row)
		}
	}
	for _, id := range []string{"premier-league", "la-liga", "bundesliga", "serie-a", "ligue-1"} {
		if !seen[id] {
			t.Fatalf("missing leader for %s", id)
		}
	}
	europe, _ := dash["europe"].(map[string]interface{})
	if europe["competition_id"] != "champions-league" || europe["stage"] == "" {
		t.Fatalf("europe payload=%v", europe)
	}
	win, _ := dash["transfer_window"].(map[string]interface{})
	if win["open"] != false {
		t.Fatalf("fresh world should start with a closed window: %v", win)
	}
}

func TestWorldDashboardRanksTransfersInjuriesAndUpcomingDeterministically(t *testing.T) {
	tm, _, te := loadEuropeanWorldForTest(t)
	if len(tm.ClubsList) < 4 {
		t.Fatal("need clubs")
	}
	a, b := tm.ClubsList[0], tm.ClubsList[1]
	if len(a.Squad) == 0 || len(b.Squad) == 0 {
		t.Fatal("need squads")
	}
	injured := a.Squad[0]
	injured.InjuredMatches = 3
	injured.Injury = "hamstring"
	injured.OVR = 88
	te.CompletedTransfers = []transfers.CompletedTransfer{
		{PlayerID: "P-LO", PlayerName: "Low Fee", FeeEUR: 1_000_000, FormattedFee: "€1.0M", BuyerID: a.ClubID, SellerID: b.ClubID, Matchweek: 1},
		{PlayerID: "P-HI", PlayerName: "High Fee", FeeEUR: 80_000_000, FormattedFee: "€80.0M", BuyerID: b.ClubID, SellerID: a.ClubID, Matchweek: 1},
	}
	tm.ManagerHistory = []ManagerHistoryEntry{
		{SeasonName: tm.SeasonName, Matchweek: 4, ClubID: a.ClubID, ClubName: a.ClubName, Action: "sacked", OldManager: "Old", NewManager: "New", Reason: "results"},
	}
	dash := tm.WorldDashboard()
	trs, _ := dash["biggest_transfers"].([]map[string]interface{})
	if len(trs) < 2 {
		t.Fatalf("transfers=%d", len(trs))
	}
	if trs[0]["player_id"] != "P-HI" {
		t.Fatalf("transfers should sort by fee desc, got %v", trs[0]["player_id"])
	}
	inj, _ := dash["injuries"].([]map[string]interface{})
	if len(inj) == 0 || inj[0]["player_id"] != injured.PlayerID {
		t.Fatalf("injuries=%v", inj)
	}
	sacks, _ := dash["sackings"].([]map[string]interface{})
	if len(sacks) == 0 || sacks[0]["old_manager"] != "Old" {
		t.Fatalf("sackings=%v", sacks)
	}
	upcoming, _ := dash["upcoming_fixtures"].([]map[string]interface{})
	if len(upcoming) == 0 {
		t.Fatal("expected upcoming fixtures on a fresh calendar")
	}
}

func TestWorldDashboardStorylinesIgnoreFavouriteClub(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	before := tm.WorldDashboard()
	storiesBefore, _ := before["storylines"].([]string)
	featuredBefore, _ := before["next_fixture"].(map[string]interface{})
	if featuredBefore == nil {
		t.Fatal("expected featured fixture")
	}
	if !tm.SetFavouriteClubID("LAL-RMA") {
		t.Fatal("could not select Madrid")
	}
	afterMadrid := tm.WorldDashboard()
	if !tm.SetFavouriteClubID("EPL-ARS") {
		t.Fatal("could not select Arsenal")
	}
	afterArsenal := tm.WorldDashboard()
	if !tm.SetFavouriteClubID("BUN-BAY") {
		t.Fatal("could not select Bayern")
	}
	afterBayern := tm.WorldDashboard()
	for _, dash := range []map[string]interface{}{afterMadrid, afterArsenal, afterBayern} {
		featured, _ := dash["next_fixture"].(map[string]interface{})
		if featured["id"] != featuredBefore["id"] {
			t.Fatalf("featured fixture followed club selection: want %v got %v", featuredBefore["id"], featured["id"])
		}
		stories, _ := dash["storylines"].([]string)
		if len(stories) != len(storiesBefore) {
			t.Fatalf("storyline count followed club selection: %d vs %d", len(stories), len(storiesBefore))
		}
		for i := range stories {
			if stories[i] != storiesBefore[i] {
				t.Fatalf("storyline %d followed club selection: %q vs %q", i, stories[i], storiesBefore[i])
			}
		}
	}
}

func TestCompetitionLeadersUseCompetitionStatsNotGlobalTotals(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	club := tm.ClubsList[0]
	if club == nil || len(club.Squad) < 2 {
		t.Fatal("need squad")
	}
	p := club.Squad[0]
	p.Goals = 40
	p.CompetitionStats = map[string]*models.CompetitionSeasonStats{
		"premier-league":   {CompetitionID: "premier-league", Appearances: 10, Starts: 10, Minutes: 900, Goals: 2, Assists: 1},
		"champions-league": {CompetitionID: "champions-league", Appearances: 4, Starts: 4, Minutes: 360, Goals: 7, Assists: 3},
	}
	scorers, _ := tm.competitionLeadersUnlocked("champions-league")
	if len(scorers) == 0 || scorers[0]["player_id"] != p.PlayerID || scorers[0]["goals"] != 7 {
		t.Fatalf("ucl scorers=%v", scorers)
	}
	leagueScorers, _ := tm.competitionLeadersUnlocked("premier-league")
	if len(leagueScorers) == 0 {
		t.Fatal("expected league scorers")
	}
	if leagueScorers[0]["player_id"] == p.PlayerID && leagueScorers[0]["goals"] != 2 {
		t.Fatalf("league goals should be 2, got %v", leagueScorers[0]["goals"])
	}
}
