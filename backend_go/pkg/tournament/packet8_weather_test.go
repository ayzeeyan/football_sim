package tournament

import (
	"testing"

	"football_sim/pkg/matchengine"
)

func TestPacket8ResolveWeatherPrefersFixtureThenMatchweek(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	fixture := &tm.Fixtures[0]
	original := fixture.Weather
	fixture.Weather = ""
	if tm.MatchweekWeather == nil {
		tm.MatchweekWeather = map[int]string{}
	}
	tm.MatchweekWeather[fixture.Matchweek] = "rain"

	if got := tm.ResolveWeather(fixture); got != "rain" {
		t.Fatalf("blank fixture resolve = %q; want rain from matchweek", got)
	}
	if fixture.Weather != "" {
		t.Fatalf("ResolveWeather mutated blank fixture: %q", fixture.Weather)
	}

	fixture.Weather = "wind"
	if got := tm.ResolveWeather(fixture); got != "wind" {
		t.Fatalf("recorded fixture weather = %q; want wind", got)
	}
	fixture.Weather = original
}

func TestPacket8EnsureFixtureWeatherWritesMatchweekClimate(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	fixture := &tm.Fixtures[0]
	fixture.Weather = ""
	if tm.MatchweekWeather == nil {
		tm.MatchweekWeather = map[int]string{}
	}
	tm.MatchweekWeather[fixture.Matchweek] = "snow"

	if got := tm.EnsureFixtureWeather(fixture); got != "snow" {
		t.Fatalf("ensure weather = %q; want snow", got)
	}
	if fixture.Weather != "snow" {
		t.Fatalf("blank fixture was not filled: %q", fixture.Weather)
	}
}

func TestPacket8LiveCommitKeepsEngineRainWhenFixtureWeatherBlank(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	var fixture *Fixture
	for i := range tm.Fixtures {
		if tm.Fixtures[i].Status != "scheduled" {
			continue
		}
		fixture = &tm.Fixtures[i]
		break
	}
	if fixture == nil {
		t.Fatal("expected a scheduled league fixture")
	}
	fixture.Weather = ""
	tm.CurrentMatchweek = fixture.Matchweek

	home, away := tm.Clubs[fixture.HomeID], tm.Clubs[fixture.AwayID]
	engine := matchengine.NewLiveMatchEngine(home, away, tm.Managers[fixture.HomeID], tm.Managers[fixture.AwayID], 808)
	engine.SetFixtureContext(fixture.Competition, fixture.Matchweek)
	engine.SetFixtureWeather("rain")
	engine.SetSpeed(999)
	engine.StartKickoff()
	engine.Tick(0.1)
	if engine.State != "FULL_TIME" {
		t.Fatalf("live engine did not complete: state=%s minute=%v", engine.State, engine.CurrentMinute)
	}

	result := tm.CommitLiveFixtureByID(fixture.FixtureID, engine)
	if result["status"] != "success" || result["recorded"] != true {
		t.Fatalf("live rain commit failed: %v", result)
	}
	finished := tm.FindFixture(fixture.FixtureID)
	if finished == nil || finished.Report == nil {
		t.Fatalf("committed fixture missing report: %+v", finished)
	}
	if finished.Weather != "rain" || finished.Report.Weather != "rain" {
		t.Fatalf("engine rain did not survive blank-fixture commit: fixture=%q report=%q", finished.Weather, finished.Report.Weather)
	}
}
