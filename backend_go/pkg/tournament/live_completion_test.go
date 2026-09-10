package tournament

import (
	"encoding/json"
	"reflect"
	"testing"

	"football_sim/pkg/matchengine"
	"football_sim/pkg/matchreport"
	"football_sim/pkg/models"
)

func liveCompletionFixtureWithProdigy(t *testing.T, tm *TournamentManager) (Fixture, *models.Player) {
	t.Helper()
	for _, f := range tm.GetSlate(tm.CurrentMatchweek) {
		if f.Competition != "super-league" || f.Status != "scheduled" {
			continue
		}
		for _, clubID := range []string{f.HomeID, f.AwayID} {
			club := tm.Clubs[clubID]
			if club == nil {
				continue
			}
			for _, p := range club.GetStartingEleven(models.FixtureContext(f.Competition, f.Matchweek)) {
				if p != nil && p.UniverseWonderkid && tm.GrowthEngine != nil && tm.GrowthEngine.Biometrics[p.PlayerID] != nil {
					return f, p
				}
			}
		}
	}
	t.Fatal("no scheduled Super League fixture with a registered starting prodigy")
	return Fixture{}, nil
}

func liveCompletionInboxCount(tm *TournamentManager, fixtureID string) int {
	count := 0
	for _, item := range tm.Inbox {
		if item.FixtureID == fixtureID {
			count++
		}
	}
	return count
}

func driveLiveEngineToFullTime(t *testing.T, engine *matchengine.LiveMatchEngine) {
	t.Helper()
	engine.SetSpeed(1)
	engine.StartKickoff()
	for ticks := 0; ticks < 200 && engine.State != "FULL_TIME"; ticks++ {
		engine.Tick(0.5)
		if engine.State == "HALF_TIME" {
			if !engine.ResumeHalfTime("home", "", "", "") {
				t.Fatal("could not resume half time")
			}
		}
	}
	if engine.State != "FULL_TIME" || engine.CurrentMinute != 90 {
		t.Fatalf("live engine did not reach full time after normal progression: state=%s minute=%v", engine.State, engine.CurrentMinute)
	}
	touches := len(engine.LiveTouches["home"]) + len(engine.LiveTouches["away"])
	if engine.CurrentMinute <= 0 || touches == 0 {
		t.Fatalf("live phase engine did not progress: minute=%v touches=%d", engine.CurrentMinute, touches)
	}
}

func TestCommitLiveFixtureFromFullTimeIsIdempotentAndSkipsFinishedSlate(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	fixture, prodigy := liveCompletionFixtureWithProdigy(t, tm)
	home := tm.Clubs[fixture.HomeID]
	away := tm.Clubs[fixture.AwayID]
	if home == nil || away == nil {
		t.Fatalf("fixture clubs missing: %+v", fixture)
	}

	engine := matchengine.NewLiveMatchEngine(
		home,
		away,
		tm.Managers[fixture.HomeID],
		tm.Managers[fixture.AwayID],
		7717,
	)
	engine.SetFixtureContext(fixture.Competition, fixture.Matchweek)
	driveLiveEngineToFullTime(t, engine)

	homePlayedBefore := home.Played
	awayPlayedBefore := away.Played
	prodigyAppsBefore := prodigy.Appearances
	bioBefore := tm.GrowthEngine.Biometrics[prodigy.PlayerID]
	if bioBefore == nil {
		t.Fatalf("missing prodigy biometric profile for %s", prodigy.PlayerID)
	}
	xpBefore, targetBefore := bioBefore.AccumulatedXP, bioBefore.LevelXPTarget
	inboxBefore := len(tm.Inbox)

	result := tm.CommitLiveFixture(fixture.HomeID, fixture.AwayID, engine)
	if result["status"] != "success" || result["recorded"] != true {
		t.Fatalf("live commit failed: %v", result)
	}
	finished := tm.FindFixture(fixture.FixtureID)
	if finished == nil {
		t.Fatalf("committed fixture %s disappeared", fixture.FixtureID)
	}
	if finished.Status != "finished" || finished.Method != "live" || finished.Report == nil {
		t.Fatalf("fixture was not fully committed: status=%s method=%s report=%v", finished.Status, finished.Method, finished.Report != nil)
	}
	if finished.HomeGoals == nil || finished.AwayGoals == nil || *finished.HomeGoals != engine.HomeScore || *finished.AwayGoals != engine.AwayScore {
		t.Fatalf("fixture score %v-%v does not match engine %d-%d", finished.HomeGoals, finished.AwayGoals, engine.HomeScore, engine.AwayScore)
	}
	if len(finished.Report.Events) != len(engine.Events) || len(finished.Report.HomeXI) == 0 || len(finished.Report.AwayXI) == 0 || len(finished.Report.HomeBench) == 0 || len(finished.Report.AwayBench) == 0 {
		t.Fatalf("live report omitted engine data: events=%d/%d homeXI=%d awayXI=%d benches=%d/%d", len(finished.Report.Events), len(engine.Events), len(finished.Report.HomeXI), len(finished.Report.AwayXI), len(finished.Report.HomeBench), len(finished.Report.AwayBench))
	}
	if home.Played != homePlayedBefore+1 || away.Played != awayPlayedBefore+1 {
		t.Fatalf("standings updated incorrectly: home played %d (from %d), away played %d (from %d)", home.Played, homePlayedBefore, away.Played, awayPlayedBefore)
	}
	prodigyRow := false
	for _, row := range append(append([]matchreport.MatchPlayerRow{}, finished.Report.HomeXI...), finished.Report.AwayXI...) {
		if row.PlayerID == prodigy.PlayerID {
			prodigyRow = row.Played && row.Minutes > 0
			break
		}
	}
	if !prodigyRow || prodigy.Appearances != prodigyAppsBefore+1 {
		t.Fatalf("participating prodigy application missing: played=%v appearances=%d from %d", prodigyRow, prodigy.Appearances, prodigyAppsBefore)
	}
	bioAfter := tm.GrowthEngine.Biometrics[prodigy.PlayerID]
	if bioAfter.AccumulatedXP == xpBefore && bioAfter.LevelXPTarget == targetBefore {
		t.Fatalf("participating prodigy received no match XP")
	}
	if len(tm.Inbox) <= inboxBefore || liveCompletionInboxCount(tm, fixture.FixtureID) == 0 {
		t.Fatalf("live commit did not add fixture inbox items: total=%d from %d", len(tm.Inbox), inboxBefore)
	}

	homePlayedAfterCommit := home.Played
	awayPlayedAfterCommit := away.Played
	prodigyAppsAfterCommit := prodigy.Appearances
	prodigyXPAfterCommit := bioAfter.AccumulatedXP
	prodigyTargetAfterCommit := bioAfter.LevelXPTarget
	inboxAfterCommit := len(tm.Inbox)
	homeGoalsForAfterCommit, awayGoalsForAfterCommit := home.GoalsFor, away.GoalsFor
	reportJSON, err := json.Marshal(finished.Report)
	if err != nil {
		t.Fatalf("failed to snapshot live report: %v", err)
	}
	fixtureHomeGoalsAfterCommit, fixtureAwayGoalsAfterCommit := *finished.HomeGoals, *finished.AwayGoals
	second := tm.CommitLiveFixture(fixture.HomeID, fixture.AwayID, engine)
	if second["recorded"] == true {
		t.Fatalf("same live engine was recorded twice: %v", second)
	}
	secondFixture := tm.FindFixture(fixture.FixtureID)
	secondReportJSON, err := json.Marshal(secondFixture.Report)
	if err != nil {
		t.Fatalf("failed to snapshot duplicate live report: %v", err)
	}
	if home.Played != homePlayedAfterCommit || away.Played != awayPlayedAfterCommit || home.GoalsFor != homeGoalsForAfterCommit || away.GoalsFor != awayGoalsForAfterCommit || prodigy.Appearances != prodigyAppsAfterCommit || bioAfter.AccumulatedXP != prodigyXPAfterCommit || bioAfter.LevelXPTarget != prodigyTargetAfterCommit || len(tm.Inbox) != inboxAfterCommit || secondFixture.Status != "finished" || secondFixture.Method != "live" || secondFixture.HomeGoals == nil || secondFixture.AwayGoals == nil || *secondFixture.HomeGoals != fixtureHomeGoalsAfterCommit || *secondFixture.AwayGoals != fixtureAwayGoalsAfterCommit || string(secondReportJSON) != string(reportJSON) {
		t.Fatalf("duplicate commit changed career state: result=%v", second)
	}

	slate := tm.GetSlate(tm.CurrentMatchweek)
	remainingExpected := 0
	for _, f := range slate {
		if f.Status == "scheduled" {
			remainingExpected++
		}
	}
	remaining := tm.SimulateRemaining()
	played, ok := remaining["played"].(int)
	if !ok || played != remainingExpected {
		t.Fatalf("remaining slate played=%v, want %d", remaining["played"], remainingExpected)
	}
	if tm.CurrentMatchweek != fixture.Matchweek+1 {
		t.Fatalf("remaining slate did not roll over once: current matchweek=%d", tm.CurrentMatchweek)
	}
	if replayed := tm.FindFixture(fixture.FixtureID); replayed == nil || replayed.Status != "finished" || replayed.Method != "live" || replayed.HomeGoals == nil || *replayed.HomeGoals != engine.HomeScore {
		t.Fatalf("live fixture changed during remaining simulation: %+v", replayed)
	}
	for _, f := range slate {
		got := tm.FindFixture(f.FixtureID)
		if got == nil {
			t.Fatalf("slate fixture %s disappeared", f.FixtureID)
		}
		if f.Status == "scheduled" && got.Status != "finished" {
			t.Fatalf("scheduled slate fixture %s was not simulated: %s", f.FixtureID, got.Status)
		}
	}
}

func TestCommitLiveFixtureAcceptsScheduledCupWithoutLeagueTableChange(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	var fixture Fixture
	for _, f := range tm.GetSlate(SuperCupWeeks["play_in"]) {
		if f.Competition == "super-cup" && f.Status == "scheduled" {
			fixture = f
			break
		}
	}
	if fixture.FixtureID == "" {
		t.Fatal("no scheduled Super Cup play-in in the current slate")
	}
	tm.CurrentMatchweek = fixture.Matchweek
	home := tm.Clubs[fixture.HomeID]
	away := tm.Clubs[fixture.AwayID]
	engine := matchengine.NewLiveMatchEngine(home, away, tm.Managers[fixture.HomeID], tm.Managers[fixture.AwayID], 8821)
	engine.SetFixtureContext(fixture.Competition, fixture.Matchweek)
	driveLiveEngineToFullTime(t, engine)

	homePlayedBefore, awayPlayedBefore := home.Played, away.Played
	result := tm.CommitLiveFixture(fixture.HomeID, fixture.AwayID, engine)
	if result["status"] != "success" || result["recorded"] != true {
		t.Fatalf("cup live commit failed: %v", result)
	}
	finished := tm.FindFixture(fixture.FixtureID)
	if finished == nil || finished.Status != "finished" || finished.Method != "live" || finished.Report == nil {
		t.Fatalf("cup fixture was not fully committed: %+v", finished)
	}
	if home.Played != homePlayedBefore || away.Played != awayPlayedBefore {
		t.Fatalf("cup result changed Super League table: home=%d/%d away=%d/%d", home.Played, homePlayedBefore, away.Played, awayPlayedBefore)
	}
	tie, ok := tm.SuperCupPlayIn[fixture.TieID]
	if !ok || tie.WinnerID == "" || len(tie.Leg1) != 2 || finished.HomeGoals == nil || finished.AwayGoals == nil || tie.Leg1[0] != *finished.HomeGoals || tie.Leg1[1] != *finished.AwayGoals {
		t.Fatalf("live Super Cup result did not resolve its play-in tie: tie=%+v fixture=%+v", tie, finished)
	}
	second := tm.CommitLiveFixture(fixture.HomeID, fixture.AwayID, engine)
	if second["recorded"] == true {
		t.Fatalf("same cup live engine was recorded twice: %v", second)
	}
	secondTie := tm.SuperCupPlayIn[fixture.TieID]
	if !reflect.DeepEqual(secondTie, tie) {
		t.Fatalf("duplicate cup commit changed bracket state: before=%+v after=%+v", tie, secondTie)
	}
}

func TestCommitLiveFixtureRollsOverExactlyOnceWhenItIsLastLeagueMatch(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	fixture, _ := liveCompletionFixtureWithProdigy(t, tm)
	for _, other := range tm.GetSlate(tm.CurrentMatchweek) {
		if other.Competition != "super-league" || other.Status != "scheduled" || other.FixtureID == fixture.FixtureID {
			continue
		}
		result := tm.SimulateFixture(other.FixtureID)
		if result["status"] != "success" {
			t.Fatalf("failed to simulate preceding league fixture %s: %v", other.FixtureID, result)
		}
	}
	if tm.CurrentMatchweek != fixture.Matchweek {
		t.Fatalf("other league fixtures rolled over before the last match: current=%d target=%d", tm.CurrentMatchweek, fixture.Matchweek)
	}
	home := tm.Clubs[fixture.HomeID]
	away := tm.Clubs[fixture.AwayID]
	homePlayedBefore, awayPlayedBefore := home.Played, away.Played
	engine := matchengine.NewLiveMatchEngine(home, away, tm.Managers[fixture.HomeID], tm.Managers[fixture.AwayID], 9931)
	engine.SetFixtureContext(fixture.Competition, fixture.Matchweek)
	driveLiveEngineToFullTime(t, engine)

	result := tm.CommitLiveFixture(fixture.HomeID, fixture.AwayID, engine)
	if result["recorded"] != true || result["rolled_over"] != true || tm.CurrentMatchweek != fixture.Matchweek+1 {
		t.Fatalf("last live league fixture did not roll over once: result=%v current=%d", result, tm.CurrentMatchweek)
	}
	if home.Played != homePlayedBefore+1 || away.Played != awayPlayedBefore+1 {
		t.Fatalf("last live league result did not update table exactly once: home=%d/%d away=%d/%d", home.Played, homePlayedBefore, away.Played, awayPlayedBefore)
	}
	playedAfter, homeGoalsAfter, awayGoalsAfter := tm.CurrentMatchweek, home.GoalsFor, away.GoalsFor
	inboxAfter := len(tm.Inbox)
	second := tm.CommitLiveFixture(fixture.HomeID, fixture.AwayID, engine)
	if second["recorded"] == true || tm.CurrentMatchweek != playedAfter || home.GoalsFor != homeGoalsAfter || away.GoalsFor != awayGoalsAfter || len(tm.Inbox) != inboxAfter {
		t.Fatalf("repeated last-fixture commit replayed rollover/result: result=%v current=%d", second, tm.CurrentMatchweek)
	}
}

func TestCommitLiveFixtureByIDDoesNotFollowSamePairAfterMatchweekAdvance(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	target, _ := liveCompletionFixtureWithProdigy(t, tm)
	later := target
	later.FixtureID = "TEST-LATER-" + target.FixtureID
	later.Matchweek = target.Matchweek + 1
	tm.Fixtures = append(tm.Fixtures, later)

	home := tm.Clubs[target.HomeID]
	away := tm.Clubs[target.AwayID]
	engine := matchengine.NewLiveMatchEngine(home, away, tm.Managers[target.HomeID], tm.Managers[target.AwayID], 10401)
	engine.SetFixtureContext(target.Competition, target.Matchweek)
	driveLiveEngineToFullTime(t, engine)

	tm.CurrentMatchweek = later.Matchweek
	homePlayedBefore := home.Played
	result := tm.CommitLiveFixtureByID(target.FixtureID, engine)
	if result["status"] != "success" || result["recorded"] != true || result["fixture_id"] != target.FixtureID {
		t.Fatalf("exact live commit failed after matchweek advance: %v", result)
	}
	committed := tm.FindFixture(target.FixtureID)
	if committed == nil || committed.Status != "finished" || committed.Method != "live" {
		t.Fatalf("selected fixture was not committed: %+v", committed)
	}
	laterAfter := tm.FindFixture(later.FixtureID)
	if laterAfter == nil || laterAfter.Status != "scheduled" || laterAfter.HomeGoals != nil || laterAfter.AwayGoals != nil {
		t.Fatalf("exact live commit followed the pair into a later fixture: %+v", laterAfter)
	}
	if home.Played != homePlayedBefore+1 {
		t.Fatalf("exact live commit updated the league table incorrectly: played=%d from %d", home.Played, homePlayedBefore)
	}

	repeat := tm.CommitLiveFixtureByID(target.FixtureID, engine)
	if repeat["recorded"] == true || repeat["terminal"] != true || tm.CurrentMatchweek != later.Matchweek {
		t.Fatalf("stale exact live identity was not terminally ignored: %v current=%d", repeat, tm.CurrentMatchweek)
	}
	if laterAfter = tm.FindFixture(later.FixtureID); laterAfter == nil || laterAfter.Status != "scheduled" {
		t.Fatalf("stale exact identity changed the later fixture: %+v", laterAfter)
	}
}

func TestCommitLiveFixtureByIDUsesCompetitionIdentityForSamePair(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	league, _ := liveCompletionFixtureWithProdigy(t, tm)
	cup := league
	cup.FixtureID = "TEST-UCL-" + league.FixtureID
	cup.Competition = "ucl"
	cup.Stage = "Group"
	cup.Report = nil
	cup.HomeGoals = nil
	cup.AwayGoals = nil
	tm.UCLFixtures = append(tm.UCLFixtures, cup)

	home := tm.Clubs[cup.HomeID]
	away := tm.Clubs[cup.AwayID]
	homePlayedBefore, awayPlayedBefore := home.Played, away.Played
	engine := matchengine.NewLiveMatchEngine(home, away, tm.Managers[cup.HomeID], tm.Managers[cup.AwayID], 10402)
	engine.SetFixtureContext(cup.Competition, cup.Matchweek)
	driveLiveEngineToFullTime(t, engine)

	result := tm.CommitLiveFixtureByID(cup.FixtureID, engine)
	if result["status"] != "success" || result["recorded"] != true || result["fixture_id"] != cup.FixtureID {
		t.Fatalf("exact cup live commit failed: %v", result)
	}
	finishedCup := tm.FindFixture(cup.FixtureID)
	if finishedCup == nil || finishedCup.Status != "finished" || finishedCup.Method != "live" {
		t.Fatalf("selected cup fixture was not committed: %+v", finishedCup)
	}
	leagueAfter := tm.FindFixture(league.FixtureID)
	if leagueAfter == nil || leagueAfter.Status != "scheduled" {
		t.Fatalf("same-pair league fixture was selected instead of the requested cup: %+v", leagueAfter)
	}
	if home.Played != homePlayedBefore || away.Played != awayPlayedBefore {
		t.Fatalf("cup live result changed league standings: home=%d/%d away=%d/%d", home.Played, homePlayedBefore, away.Played, awayPlayedBefore)
	}
}

func TestResetNewSeasonDoesNotDuplicateHistory(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	// Fast forward matchweeks to transfer_window phase
	tm.SeasonPhase = "transfer_window"
	tm.CurrentMatchweek = tm.MaxMatchweeks + 1

	// Simulate at least one played match so history gets archived
	f := &tm.Fixtures[0]
	g1, g0 := 2, 1
	f.Status = "finished"
	f.HomeGoals = &g1
	f.AwayGoals = &g0
	tm.Clubs[f.HomeID].Played = 1
	tm.Clubs[f.HomeID].Points = 3

	res1 := tm.ResetNewSeason()
	if res1["status"] != "success" {
		t.Fatalf("ResetNewSeason failed: %v", res1)
	}
	historyLen1 := len(tm.SeasonHistory)
	if historyLen1 == 0 {
		t.Fatalf("expected season history to be archived")
	}

	// Immediate second call when 0 games played in new season
	res2 := tm.ResetNewSeason()
	if res2["status"] != "success" {
		t.Fatalf("ResetNewSeason failed: %v", res2)
	}
	if len(tm.SeasonHistory) != historyLen1 {
		t.Fatalf("season history duplicated on unplayed season reset: got %d, want %d", len(tm.SeasonHistory), historyLen1)
	}
}
