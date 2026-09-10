package tournament

import "testing"

func TestCalendarLabelIncludesSeasonYear(t *testing.T) {
	if got := CalendarLabel("2026-27", 1); got != "MW 1/44 · August 2026" {
		t.Fatalf("opening calendar label = %q", got)
	}
	if got := CalendarLabel("2026-27", 21); got != "MW 21/44 · January 2027" {
		t.Fatalf("new-year calendar label = %q", got)
	}
}

func TestMacroMonthProducesDigestAndAdvancesFourWeeks(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	result := tm.SimulateBatchWeeks(4)
	if result.Status != "success" {
		t.Fatalf("macro simulation failed: %+v", result)
	}
	if result.WeeksAdvanced != 4 || len(result.Digests) != 4 {
		t.Fatalf("advanced=%d digests=%d, want 4/4", result.WeeksAdvanced, len(result.Digests))
	}
	if tm.CurrentMatchweek != 5 || result.CurrentMatchweek != 5 {
		t.Fatalf("current matchweek tm=%d result=%d, want 5", tm.CurrentMatchweek, result.CurrentMatchweek)
	}
	if result.Played == 0 || len(result.Digests[0].Results) == 0 {
		t.Fatal("expected simulated fixtures in matchweek digest")
	}
	if result.Digests[0].CalendarLabel == "" || result.Digests[0].Year != 2026 {
		t.Fatalf("digest calendar metadata missing: %+v", result.Digests[0])
	}
}

func TestMacroSeasonStopsAtAwardsBoundary(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	result := tm.SimulateBatchWeeks(100)
	if result.Status != "success" {
		t.Fatalf("macro season failed: %+v", result)
	}
	if !result.SeasonFinished || !result.AwardsReady {
		t.Fatalf("season boundary not exposed: finished=%v awards=%v", result.SeasonFinished, result.AwardsReady)
	}
	if result.WeeksAdvanced != 44 || len(result.Digests) != 44 {
		t.Fatalf("season advanced=%d digests=%d, want 44/44", result.WeeksAdvanced, len(result.Digests))
	}
	if tm.SeasonPhase != "transfer_window" || tm.CurrentMatchweek != 45 {
		t.Fatalf("phase=%q mw=%d, want transfer_window/45", tm.SeasonPhase, tm.CurrentMatchweek)
	}
	if result.Champion == "" {
		t.Fatal("season batch should return a champion")
	}
}

func TestProdigyWatchContainsCanonicalTwelve(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	rows := tm.GetProdigyWatch()
	if len(rows) != 12 {
		t.Fatalf("prodigy watch rows=%d, want 12", len(rows))
	}
	last := 1e9
	for i, row := range rows {
		score, ok := row["golden_boy_score"].(float64)
		if !ok {
			t.Fatalf("row %d missing golden_boy_score: %#v", i, row)
		}
		if score > last {
			t.Fatalf("prodigy watch not sorted descending at row %d: %.1f > %.1f", i, score, last)
		}
		last = score
	}
}

func TestAutonomousManagerSackingPersistsSpellHistory(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	target := tm.ClubsList[0]
	oldName := tm.Managers[target.ClubID].Name

	for _, club := range tm.ClubsList {
		club.Played = 8
		club.Won = 4
		club.Drawn = 0
		club.Lost = 4
		club.Points = 12
	}
	target.Won = 0
	target.Lost = 8
	target.Points = 0

	losses := 0
	for i := range tm.Fixtures {
		f := &tm.Fixtures[i]
		if f.Matchweek > 8 || (f.HomeID != target.ClubID && f.AwayID != target.ClubID) {
			continue
		}
		home, away := 1, 0
		if f.HomeID == target.ClubID {
			home, away = 0, 1
		}
		f.Status = "finished"
		f.HomeGoals = &home
		f.AwayGoals = &away
		losses++
	}
	if losses < 6 {
		t.Fatalf("prepared only %d losses", losses)
	}

	tm.evaluateManagerTenure(8)
	if tm.Managers[target.ClubID].Name != oldName {
		t.Fatal("manager should survive the first hot-seat assessment")
	}
	tm.evaluateManagerTenure(9)

	current := tm.Managers[target.ClubID]
	if current == nil || current.Name == oldName {
		t.Fatalf("manager was not replaced: old=%q current=%+v", oldName, current)
	}
	if current.JobSecurity != "Safe" || current.AppointedSeason != tm.SeasonName || current.AppointedMatchweek != 10 {
		t.Fatalf("replacement metadata incomplete: %+v", current)
	}
	if len(current.History) == 0 || current.History[len(current.History)-1].ManagerName != oldName {
		t.Fatalf("completed spell not carried into manager history: %+v", current.History)
	}
	if len(tm.ManagerHistory) == 0 || tm.ManagerHistory[len(tm.ManagerHistory)-1].Action != "sacked" {
		t.Fatalf("universe manager event missing: %+v", tm.ManagerHistory)
	}
}
