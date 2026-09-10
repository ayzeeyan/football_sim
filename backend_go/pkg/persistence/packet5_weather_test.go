package persistence

import (
	"path/filepath"
	"testing"

	"football_sim/pkg/matchengine"
)

func TestPacket5LiveRainCommitReportAndSave(t *testing.T) {
	_, ge, tm, te := setupTestWorld(t)
	var targetID, homeID, awayID string
	for i := range tm.Fixtures {
		fixture := &tm.Fixtures[i]
		if fixture.Status != "scheduled" {
			continue
		}
		fixture.Weather = "rain"
		targetID, homeID, awayID = fixture.FixtureID, fixture.HomeID, fixture.AwayID
		tm.CurrentMatchweek = fixture.Matchweek
		break
	}
	if targetID == "" {
		t.Fatal("expected a scheduled fixture")
	}

	home, away := tm.Clubs[homeID], tm.Clubs[awayID]
	engine := matchengine.NewLiveMatchEngine(home, away, tm.Managers[homeID], tm.Managers[awayID], 5515)
	engine.SetFixtureContext("super-league", tm.CurrentMatchweek)
	engine.SetFixtureWeather("rain")
	engine.SetSpeed(999)
	engine.StartKickoff()
	engine.Tick(0.1)
	if engine.State != "FULL_TIME" {
		t.Fatalf("live engine did not complete: state=%s minute=%v", engine.State, engine.CurrentMinute)
	}

	result := tm.CommitLiveFixtureByID(targetID, engine)
	if result["status"] != "success" || result["recorded"] != true {
		t.Fatalf("rain live commit failed: %v", result)
	}
	finished := tm.FindFixture(targetID)
	if finished == nil || finished.Status != "finished" || finished.Report == nil {
		t.Fatalf("rain fixture was not fully committed: %+v", finished)
	}
	if finished.Weather != "rain" || finished.Report.Weather != "rain" || finished.Report.Stats.Home.Shots != engine.HomeShots || finished.Report.Stats.Away.Shots != engine.AwayShots {
		t.Fatalf("rain weather/stats did not reach live report: fixture=%q report=%q stats=%+v engine shots=%d/%d", finished.Weather, finished.Report.Weather, finished.Report.Stats, engine.HomeShots, engine.AwayShots)
	}

	savePath := filepath.Join(t.TempDir(), "packet5_rain.json")
	if _, err := SaveCareer(tm, ge, te, savePath); err != nil {
		t.Fatalf("SaveCareer failed: %v", err)
	}
	snapshot, err := LoadCareer(savePath)
	if err != nil {
		t.Fatalf("LoadCareer failed: %v", err)
	}
	for i := range snapshot.Fixtures {
		fixture := &snapshot.Fixtures[i]
		if fixture.FixtureID == targetID {
			if fixture.Weather != "rain" || fixture.Report == nil || fixture.Report.Weather != "rain" {
				t.Fatalf("saved rain fixture lost weather/report: %+v", fixture)
			}
			return
		}
	}
	t.Fatalf("saved fixture %s not found", targetID)
}
