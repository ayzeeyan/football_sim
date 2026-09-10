package tournament

import (
	"fmt"
	"testing"
)

func TestCalendar44_ScheduleStructure(t *testing.T) {
	tm, _ := loadTestUniverse(t)

	// 1. Total fixtures = 44 * 6 = 264
	expectedTotal := 44 * 6
	if len(tm.Fixtures) != expectedTotal {
		t.Fatalf("total league fixtures = %d; want %d", len(tm.Fixtures), expectedTotal)
	}

	// 2. Exactly 44 matchweeks, each with exactly 6 fixtures
	perWeek := make(map[int][]Fixture)
	homeGames := make(map[string]int)
	awayGames := make(map[string]int)
	headToHead := make(map[string]int)

	for _, f := range tm.Fixtures {
		if f.Competition != "super-league" {
			t.Errorf("fixture %s has competition %q; want super-league", f.FixtureID, f.Competition)
		}
		perWeek[f.Matchweek] = append(perWeek[f.Matchweek], f)
		homeGames[f.HomeID]++
		awayGames[f.AwayID]++
		headToHead[fmt.Sprintf("%s->%s", f.HomeID, f.AwayID)]++
	}

	if len(perWeek) != 44 {
		t.Fatalf("number of matchweeks = %d; want 44", len(perWeek))
	}

	for mw := 1; mw <= 44; mw++ {
		fixtures := perWeek[mw]
		if len(fixtures) != 6 {
			t.Errorf("matchweek %d has %d fixtures; want 6", mw, len(fixtures))
		}

		// Ensure all 12 clubs play exactly once in this matchweek
		seenClubs := make(map[string]bool)
		for _, f := range fixtures {
			if seenClubs[f.HomeID] {
				t.Errorf("MW %d: club %s appears multiple times as home", mw, f.HomeID)
			}
			if seenClubs[f.AwayID] {
				t.Errorf("MW %d: club %s appears multiple times as away", mw, f.AwayID)
			}
			seenClubs[f.HomeID] = true
			seenClubs[f.AwayID] = true
		}
		if len(seenClubs) != 12 {
			t.Errorf("MW %d: %d unique clubs playing; want 12", mw, len(seenClubs))
		}
	}

	// 3. Each club plays exactly 44 games: 22 home, 22 away
	for _, club := range tm.ClubsList {
		h := homeGames[club.ClubID]
		a := awayGames[club.ClubID]
		if h != 22 {
			t.Errorf("club %s has %d home games; want 22", club.ShortName, h)
		}
		if a != 22 {
			t.Errorf("club %s has %d away games; want 22", club.ShortName, a)
		}
		if h+a != 44 {
			t.Errorf("club %s has %d total games; want 44", club.ShortName, h+a)
		}
	}

	// 4. Balanced matchups: for every pair (A, B), A hosts B twice and B hosts A twice
	for i := 0; i < len(tm.ClubsList); i++ {
		for j := i + 1; j < len(tm.ClubsList); j++ {
			c1 := tm.ClubsList[i].ClubID
			c2 := tm.ClubsList[j].ClubID
			c1Hosts := headToHead[fmt.Sprintf("%s->%s", c1, c2)]
			c2Hosts := headToHead[fmt.Sprintf("%s->%s", c2, c1)]
			if c1Hosts != 2 {
				t.Errorf("pair (%s, %s): %s hosted %d times; want 2", c1, c2, c1, c1Hosts)
			}
			if c2Hosts != 2 {
				t.Errorf("pair (%s, %s): %s hosted %d times; want 2", c1, c2, c2, c2Hosts)
			}
		}
	}
}

func TestCalendar44_MonthBandsAndChapters(t *testing.T) {
	if len(MonthBands) != 10 {
		t.Fatalf("MonthBands count = %d; want 10", len(MonthBands))
	}
	expectedBands := []struct {
		lo, hi int
		name   string
	}{
		{1, 4, "August"},
		{5, 8, "September"},
		{9, 12, "October"},
		{13, 16, "November"},
		{17, 20, "December"},
		{21, 24, "January"},
		{25, 28, "February"},
		{29, 33, "March"},
		{34, 38, "April"},
		{39, 44, "May"},
	}

	for i, exp := range expectedBands {
		b := MonthBands[i]
		if b.Lo != exp.lo || b.Hi != exp.hi || b.Name != exp.name {
			t.Errorf("band %d = {%d, %d, %s}; want {%d, %d, %s}", i, b.Lo, b.Hi, b.Name, exp.lo, exp.hi, exp.name)
		}
	}

	// Verify UCL knockout weeks
	if len(UCLQFWeeks) != 2 || UCLQFWeeks[0] != 35 || UCLQFWeeks[1] != 36 {
		t.Errorf("UCLQFWeeks = %v; want [35 36]", UCLQFWeeks)
	}
	if len(UCLSFWeeks) != 2 || UCLSFWeeks[0] != 39 || UCLSFWeeks[1] != 40 {
		t.Errorf("UCLSFWeeks = %v; want [39 40]", UCLSFWeeks)
	}
	if UCLFinalWeek != 44 {
		t.Errorf("UCLFinalWeek = %d; want 44", UCLFinalWeek)
	}

	// Verify WeekChapter strings
	if ch := WeekChapter(35); ch != "April: Champions Cup quarters" {
		t.Errorf("MW 35 chapter = %q; want 'April: Champions Cup quarters'", ch)
	}
	if ch := WeekChapter(36); ch != "April: Champions Cup quarters" {
		t.Errorf("MW 36 chapter = %q; want 'April: Champions Cup quarters'", ch)
	}
	if ch := WeekChapter(39); ch != "May: Champions Cup semis" {
		t.Errorf("MW 39 chapter = %q; want 'May: Champions Cup semis'", ch)
	}
	if ch := WeekChapter(40); ch != "May: Champions Cup semis" {
		t.Errorf("MW 40 chapter = %q; want 'May: Champions Cup semis'", ch)
	}
	if ch := WeekChapter(44); ch != "May: Champions Cup final night" {
		t.Errorf("MW 44 chapter = %q; want 'May: Champions Cup final night'", ch)
	}
}

func TestCalendar44_GetCalendarAPI(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	cal := tm.GetCalendar()

	maxMW, ok := cal["max_matchweeks"].(int)
	if !ok || maxMW != 44 {
		t.Errorf("cal.max_matchweeks = %v; want 44", cal["max_matchweeks"])
	}

	weeks, ok := cal["weeks"].([]map[string]interface{})
	if !ok || len(weeks) != 44 {
		t.Fatalf("cal.weeks count = %d; want 44", len(weeks))
	}

	for mw := 1; mw <= 44; mw++ {
		w := weeks[mw-1]
		if w["matchweek"] != mw {
			t.Errorf("week %d has matchweek %v", mw, w["matchweek"])
		}
		if w["league"] != 6 {
			t.Errorf("week %d has %v league matches; want 6", mw, w["league"])
		}
	}
}

func TestCalendar44_SeasonalMatchVolume_DeepRun(t *testing.T) {
	// Calculate mathematical season game volume for an elite club reaching both finals:
	leagueGames := LeagueRounds // 44
	uclGroupGames := len(UCLGroupWeeks) // 5
	uclKnockoutGames := len(UCLQFWeeks) + len(UCLSFWeeks) + 1 // 2 + 2 + 1 = 5
	uclTotal := uclGroupGames + uclKnockoutGames // 10

	// Super cup finalist plays 3 (with top 4 bye) or 4 (via play-in)
	superCupWithBye := 3
	superCupWithPlayIn := 4

	totalWithBye := leagueGames + uclTotal + superCupWithBye
	totalWithPlayIn := leagueGames + uclTotal + superCupWithPlayIn

	if totalWithBye != 57 {
		t.Errorf("total games with bye = %d; want 57", totalWithBye)
	}
	if totalWithPlayIn != 58 {
		t.Errorf("total games with play-in = %d; want 58", totalWithPlayIn)
	}

	// Strictly verify ~55-60 target
	if totalWithBye < 55 || totalWithBye > 60 {
		t.Errorf("total games with bye %d outside ~55-60 range", totalWithBye)
	}
	if totalWithPlayIn < 55 || totalWithPlayIn > 60 {
		t.Errorf("total games with play-in %d outside ~55-60 range", totalWithPlayIn)
	}
}

func TestCalendar44_AdoptLongSeasonFromLegacy33Weeks(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	// Simulate legacy 33-week save with 198 fixtures
	legacy := tm.Fixtures[:33*6]
	// Mark MW 1 as finished
	g1, g2 := 3, 1
	legacy[0].Status = "finished"
	legacy[0].HomeGoals = &g1
	legacy[0].AwayGoals = &g2

	tm.Fixtures = legacy
	tm.MaxMatchweeks = 33

	adopted := tm.AdoptLongSeason()
	if !adopted {
		t.Fatalf("expected AdoptLongSeason to return true for 33-week save")
	}

	if tm.MaxMatchweeks != 44 {
		t.Errorf("MaxMatchweeks after adoption = %d; want 44", tm.MaxMatchweeks)
	}
	if len(tm.Fixtures) != 264 {
		t.Errorf("total fixtures after adoption = %d; want 264", len(tm.Fixtures))
	}

	// Check that finished match is preserved
	foundFinished := false
	for _, f := range tm.Fixtures {
		if f.FixtureID == legacy[0].FixtureID {
			if f.Status == "finished" && *f.HomeGoals == 3 && *f.AwayGoals == 1 {
				foundFinished = true
			}
		}
	}
	if !foundFinished {
		t.Errorf("finished match from MW 1 was not preserved after adoption")
	}

	// Verify all 44 matchweeks exist with 6 fixtures each
	counts := make(map[int]int)
	for _, f := range tm.Fixtures {
		counts[f.Matchweek]++
	}
	for mw := 1; mw <= 44; mw++ {
		if counts[mw] != 6 {
			t.Errorf("MW %d has %d fixtures; want 6", mw, counts[mw])
		}
	}
}

func TestCalendar44_FullSeasonSimulationAndTotalVolume(t *testing.T) {
	tm, _ := loadTestUniverse(t)

	// Simulate all 44 matchweeks
	for mw := 1; mw <= 44; mw++ {
		res := tm.SimulateMatchweek(mw)
		if res["status"] != "success" {
			t.Fatalf("failed to simulate matchweek %d: %v", mw, res)
		}
	}

	// Verify season finished and rolled to transfer window
	if tm.SeasonPhase != "transfer_window" {
		t.Errorf("season phase after MW 44 = %q; want 'transfer_window'", tm.SeasonPhase)
	}
	if tm.CurrentMatchweek != 45 {
		t.Errorf("CurrentMatchweek after MW 44 = %d; want 45", tm.CurrentMatchweek)
	}

	// Verify all 264 league fixtures are finished
	for _, f := range tm.Fixtures {
		if f.Status != "finished" {
			t.Errorf("fixture %s status = %q; want 'finished'", f.FixtureID, f.Status)
		}
		if f.HomeGoals == nil || f.AwayGoals == nil {
			t.Errorf("fixture %s missing score", f.FixtureID)
		}
	}

	// Verify each club has exactly 44 league games played
	standings := tm.GetStandings()
	if len(standings) != 12 {
		t.Fatalf("standings count = %d; want 12", len(standings))
	}
	for _, c := range standings {
		if c.Played != 44 {
			t.Errorf("club %s league played = %d; want 44", c.ShortName, c.Played)
		}
	}

	// Count total appearances across all competitions for all clubs
	clubTotalMatches := make(map[string]int)
	for _, f := range tm.Fixtures {
		if f.Status == "finished" {
			clubTotalMatches[f.HomeID]++
			clubTotalMatches[f.AwayID]++
		}
	}
	for _, f := range tm.UCLFixtures {
		if f.Status == "finished" {
			clubTotalMatches[f.HomeID]++
			clubTotalMatches[f.AwayID]++
		}
	}
	for _, f := range tm.SuperCupFixtures {
		if f.Status == "finished" {
			clubTotalMatches[f.HomeID]++
			clubTotalMatches[f.AwayID]++
		}
	}

	// UCL champion must have played 10 UCL matches + 44 league + Super Cup matches
	if tm.UCLChampionID != "" {
		total := clubTotalMatches[tm.UCLChampionID]
		if total < 55 || total > 60 {
			t.Errorf("UCL champion %s total games = %d; want between 55 and 60", tm.UCLChampionID, total)
		}
	}

	// Find the club with maximum matches played (deep-run club)
	maxMatches := 0
	maxClub := ""
	for cid, count := range clubTotalMatches {
		if count > maxMatches {
			maxMatches = count
			maxClub = cid
		}
	}
	// Deep-run club should have played ~57-58 matches (strictly 55-60)
	if maxMatches < 55 || maxMatches > 60 {
		t.Errorf("deep run club %s played %d matches; want between 55 and 60", maxClub, maxMatches)
	}
}

