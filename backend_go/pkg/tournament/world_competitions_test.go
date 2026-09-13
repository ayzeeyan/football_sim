package tournament

import (
	"path/filepath"
	"testing"

	"football_sim/pkg/datamanager"
	"football_sim/pkg/growth"
	"football_sim/pkg/transfers"
)

func loadEuropeanWorldForTest(t *testing.T) (*TournamentManager, *growth.GrowthEngine, *transfers.TransferEngine) {
	t.Helper()
	ge := growth.NewGrowthEngine(8181)
	dm := datamanager.NewDataManager(filepath.Join("..", "..", "..", "dataset.json"), ge)
	if len(dm.ClubsList) != 96 {
		t.Fatalf("dataset clubs=%d want 96", len(dm.ClubsList))
	}
	tm := NewEuropeanWorldManager(dm.ClubsList, ge, 8181)
	te := transfers.NewTransferEngine(dm.ClubsList, tm.Managers, 9191)
	tm.TransferEngine = te
	return tm, ge, te
}

func TestEuropeanWorldBuildsTopFiveDomesticSchedulesAndEuropeanFields(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	if tm.World == nil || tm.MaxMatchweeks != 38 {
		t.Fatalf("world=%v max weeks=%d", tm.World != nil, tm.MaxMatchweeks)
	}
	if err := tm.ValidateWorldState(); err != nil {
		t.Fatalf("fresh European world invalid: %v", err)
	}
	wants := map[string]struct{ clubs, fixtures, games int }{
		"premier-league": {20, 380, 38}, "la-liga": {20, 380, 38}, "serie-a": {20, 380, 38},
		"bundesliga": {18, 306, 34}, "ligue-1": {18, 306, 34},
	}
	for id, want := range wants {
		comp := tm.World.Competitions[id]
		if comp == nil || len(comp.ParticipantIDs) != want.clubs {
			t.Fatalf("%s participants=%d want %d", id, len(comp.ParticipantIDs), want.clubs)
		}
		fixtures := tm.worldCompetitionFixturesUnlocked(id)
		if len(fixtures) != want.fixtures {
			t.Fatalf("%s fixtures=%d want %d", id, len(fixtures), want.fixtures)
		}
		games := map[string]int{}
		for _, f := range fixtures {
			games[f.HomeID]++
			games[f.AwayID]++
		}
		for _, clubID := range comp.ParticipantIDs {
			if games[clubID] != want.games {
				t.Fatalf("%s %s games=%d want %d", id, clubID, games[clubID], want.games)
			}
		}
	}
	for _, id := range []string{"europa-league", "conference-league"} {
		comp := tm.World.Competitions[id]
		if comp == nil || len(comp.ParticipantIDs) != 20 || len(comp.LeaguePhaseFixtureIDs) != 80 {
			t.Fatalf("%s participants/fixtures=%d/%d want 20/80", id, len(comp.ParticipantIDs), len(comp.LeaguePhaseFixtureIDs))
		}
		for _, clubID := range comp.ParticipantIDs {
			if comp.Records[clubID] == nil || comp.QualificationSources[clubID] == "" {
				t.Fatalf("%s missing record or qualification provenance for %s", id, clubID)
			}
		}
	}
	ucl := tm.World.Competitions["champions-league"]
	if ucl == nil || len(ucl.ParticipantIDs) != 36 || len(ucl.LeaguePhaseFixtureIDs) != 144 {
		t.Fatalf("champions-league participants/fixtures=%d/%d want 36/144", len(ucl.ParticipantIDs), len(ucl.LeaguePhaseFixtureIDs))
	}
	for _, clubID := range ucl.ParticipantIDs {
		if ucl.Records[clubID] == nil || ucl.QualificationSources[clubID] == "" {
			t.Fatalf("champions-league missing record or qualification provenance for %s", clubID)
		}
	}
	if tm.World.Competitions["efl-cup"] == nil {
		t.Fatal("missing EFL Cup")
	}
}

func TestEuropeanWorldSharedCalendarAdvancesLeagueAndCupNights(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	first := tm.SimulateRemaining()
	if first["status"] != "success" || first["played"] != 48 || tm.CurrentMatchweek != 2 {
		t.Fatalf("week one result=%v current=%d", first, tm.CurrentMatchweek)
	}
	second := tm.SimulateRemaining()
	if second["status"] != "success" || second["played"].(int) <= 48 || tm.CurrentMatchweek != 3 {
		t.Fatalf("week two did not include domestic cups: result=%v current=%d", second, tm.CurrentMatchweek)
	}
	third := tm.SimulateRemaining()
	if third["status"] != "success" || third["played"].(int) <= 48 || tm.CurrentMatchweek != 4 {
		t.Fatalf("week three did not include European competition: result=%v current=%d", third, tm.CurrentMatchweek)
	}
	wantPlayed := map[string]int{"champions-league": 36, "europa-league": 20, "conference-league": 20}
	for id, want := range wantPlayed {
		comp := tm.World.Competitions[id]
		played := 0
		for _, record := range comp.Records {
			played += record.Played
		}
		if played != want {
			t.Fatalf("%s records show %d club-games after first league phase night, want %d", id, played, want)
		}
	}
	if err := tm.ValidateWorldState(); err != nil {
		t.Fatalf("world invalid after shared calendar: %v", err)
	}
}

func TestEuropeanWorldCompletesCompetitionsAndCarriesQualificationIntoNextSeason(t *testing.T) {
	tm, _, te := loadEuropeanWorldForTest(t)
	batch := tm.SimulateBatchWeeks(38)
	if batch.Status != "success" || !batch.SeasonFinished || tm.SeasonPhase != "transfer_window" {
		t.Fatalf("world season did not finish: batch=%+v phase=%s", batch, tm.SeasonPhase)
	}
	for _, id := range []string{"fa-cup", "efl-cup", "copa-del-rey", "dfb-pokal", "coppa-italia", "coupe-de-france", "champions-league", "europa-league", "conference-league"} {
		comp := tm.World.Competitions[id]
		if comp == nil || comp.ChampionID == "" || tm.Clubs[comp.ChampionID] == nil {
			t.Fatalf("%s did not produce a champion: %#v", id, comp)
		}
	}
	if err := tm.ValidateWorldState(); err != nil {
		t.Fatalf("completed world invalid: %v", err)
	}
	te.BeginOffSeasonWindow()
	for i := 0; i < transfers.TransferWindowWeeks; i++ {
		te.AdvanceOpenWindow()
	}
	if transition := tm.FinalizeSeasonTransition(); transition["status"] != "success" {
		t.Fatalf("world season transition failed: %v", transition)
	}
	if tm.SeasonName != "2027-28" || tm.SeasonPhase != "season" || tm.CurrentMatchweek != 1 {
		t.Fatalf("new world season state=%s/%s/MW%d", tm.SeasonName, tm.SeasonPhase, tm.CurrentMatchweek)
	}
	wants := map[string]int{"champions-league": 36, "europa-league": 20, "conference-league": 20}
	for id, n := range wants {
		comp := tm.World.Competitions[id]
		if len(comp.ParticipantIDs) != n || len(comp.QualificationSources) != n {
			t.Fatalf("next-season %s qualification=%d/%d want %d/%d", id, len(comp.ParticipantIDs), len(comp.QualificationSources), n, n)
		}
	}
	if err := tm.ValidateWorldState(); err != nil {
		t.Fatalf("next world season invalid: %v", err)
	}
}

func TestWorldAwardsIncludeDomesticGoldenBoots(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	ceremony := tm.GetAwardsCeremony()
	cats, _ := ceremony["categories"].([]map[string]interface{})
	found := false
	for _, cat := range cats {
		if cat["key"] == "premier-league-golden-boot" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("world ceremony missing Premier League Golden Boot: %#v", cats)
	}
	leagues, _ := ceremony["league_teams_of_the_season"].(map[string]interface{})
	if leagues["premier-league"] == nil {
		t.Fatalf("missing Premier League team of the season: %#v", leagues)
	}
}

func TestGetCompetitionsListsConfiguredWorldOrder(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	list := tm.GetCompetitions()
	if len(list) != 14 {
		t.Fatalf("competitions=%d want 14", len(list))
	}
	if list[0]["id"] != "premier-league" || list[len(list)-1]["id"] != "conference-league" {
		t.Fatalf("unexpected competition order: first=%v last=%v", list[0]["id"], list[len(list)-1]["id"])
	}
	detail := tm.GetCompetition("champions-league")
	if detail == nil {
		t.Fatal("missing champions-league detail")
	}
	sources, _ := detail["qualification_sources"].(map[string]string)
	if len(sources) != 36 {
		t.Fatalf("opening UCL qualification sources=%d want 36", len(sources))
	}
}
