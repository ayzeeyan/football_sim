package matchengine

import (
	"math/rand"
	"testing"
)

func packet5LiveShot(seed int64, weather string) *LiveMatchEngine {
	home, away := createTestClubs()
	e := NewLiveMatchEngine(home, away, nil, nil, seed)
	e.ResetMatch()
	e.SetFixtureWeather(weather)
	e.RNG = rand.New(rand.NewSource(seed))
	e.ResolveShot(home, away, e.HomeStarters, e.AwayStarters)
	return e
}

func TestPacket5LiveWeatherContextLifecycle(t *testing.T) {
	home, away := createTestClubs()
	e := NewLiveMatchEngine(home, away, nil, nil, 501)
	if e.Weather != "clear" {
		t.Fatalf("new live engine weather = %q; want clear", e.Weather)
	}

	e.SetFixtureContext("super-league", 4)
	e.SetFixtureWeather("rain")
	e.ResetMatch()
	if e.Weather != "rain" {
		t.Fatalf("same-fixture reset cleared weather: %q", e.Weather)
	}

	e.SetFixtureContext("super-league", 5)
	if e.Weather != "clear" {
		t.Fatalf("new fixture context retained stale weather: %q", e.Weather)
	}
	e.SetFixtureWeather("")
	if e.Weather != "clear" {
		t.Fatalf("empty fixture weather = %q; want clear", e.Weather)
	}
	e.SetFixtureWeather("rain")
	e.ClearFixtureContext()
	if e.Weather != "clear" {
		t.Fatalf("cleared fixture context retained weather: %q", e.Weather)
	}
}

func TestPacket5LiveRainAccuracyFactorAppliesOnce(t *testing.T) {
	for seed := int64(1); seed <= 500; seed++ {
		clear := packet5LiveShot(seed, "clear")
		rain := packet5LiveShot(seed, "rain")
		if clear.HomeShots != 1 || rain.HomeShots != 1 || clear.HomeShotsOn != 1 || rain.HomeShotsOn != 0 {
			continue
		}
		if len(clear.LiveShots) != 1 || len(rain.LiveShots) != 1 || clear.LiveShots[0].Outcome == "miss" || rain.LiveShots[0].Outcome != "miss" {
			continue
		}
		return
	}
	t.Fatal("no seeded open-play shot crossed clear .78 versus rain .78*.96 boundary")
}

func TestPacket5ShootingAccuracyWeatherFactors(t *testing.T) {
	for weather, want := range map[string]float64{
		"clear": 1.0, "overcast": 1.0, "rain": 0.96, "wind": 0.92, "snow": 1.0,
	} {
		if got := shootingAccuracyWeatherFactor(weather); got != want {
			t.Errorf("weather %q factor = %v; want %v", weather, got, want)
		}
	}
}

func TestPacket8RainCardClimateFactor(t *testing.T) {
	for weather, want := range map[string]float64{
		"clear": 1.0, "overcast": 1.0, "rain": 1.05, "wind": 1.0, "snow": 1.0, "": 1.0,
	} {
		if got := weatherCardClimateFactor(weather); got != want {
			t.Errorf("weather %q card climate = %v; want %v", weather, got, want)
		}
	}
}
