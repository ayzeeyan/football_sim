package matchengine

import (
	"testing"

	"football_sim/pkg/managers"
	"football_sim/pkg/models"
)

func seekClubs() (*models.Club, *models.Club) {
	mk := func(id, short string) *models.Club {
		squad := []*models.Player{
			{PlayerID: id + "-GK", FullName: short + " Keeper", Category: "GK", OVR: 72},
		}
		for i := 0; i < 4; i++ {
			squad = append(squad, &models.Player{PlayerID: id + "-D", FullName: short + " Defender", Category: "DEF", OVR: 70})
		}
		for i := 0; i < 3; i++ {
			squad = append(squad, &models.Player{PlayerID: id + "-M", FullName: short + " Midfielder", Category: "MID", OVR: 71})
		}
		for i := 0; i < 3; i++ {
			squad = append(squad, &models.Player{PlayerID: id + "-F", FullName: short + " Forward", Category: "FWD", OVR: 72})
		}
		return &models.Club{ClubID: id, ClubName: short + " FC", ShortName: short, HomeStadium: short + " Park", Squad: squad}
	}
	return mk("H", "HOM"), mk("A", "AWY")
}

func seekEngine() *LiveMatchEngine {
	home, away := seekClubs()
	e := NewLiveMatchEngine(home, away, &managers.ManagerProfile{}, &managers.ManagerProfile{}, 9)
	e.SetClubs(home, away, &managers.ManagerProfile{}, &managers.ManagerProfile{})
	e.Kickoff()
	return e
}

func TestSeekTo70StopsAtDugoutFirstHalf(t *testing.T) {
	e := seekEngine()
	before := e.InstanceID
	if !e.SeekTo70() {
		t.Fatal("seek from kickoff should move the clock")
	}
	if e.State != "HALF_TIME" {
		t.Fatalf("first-half seek must stop at the dugout, got %s @ %.1f", e.State, e.CurrentMinute)
	}
	if e.CurrentMinute != 45 {
		t.Fatalf("dugout at 45', got %.1f", e.CurrentMinute)
	}
	if e.InstanceID != before {
		t.Fatal("seek must never rotate the commit instance")
	}
}

func TestSeekTo70LandsNear70(t *testing.T) {
	e := seekEngine()
	e.CurrentMinute = 50
	e.State = "PLAYING"
	before := e.InstanceID
	if !e.SeekTo70() {
		t.Fatal("second-half seek should move the clock")
	}
	if e.CurrentMinute < 69.5 || e.CurrentMinute > 71 {
		t.Fatalf("seek should land near 70', got %.2f", e.CurrentMinute)
	}
	if e.State != "PLAYING" {
		t.Fatalf("70' is mid-half, got %s", e.State)
	}
	if e.InstanceID != before {
		t.Fatal("seek must never rotate the commit instance")
	}
}

func TestSeekNextChanceFindsShot(t *testing.T) {
	e := seekEngine()
	before := e.InstanceID
	if !e.SeekNextChance() {
		if e.State != "FULL_TIME" && e.State != "HALF_TIME" {
			t.Fatalf("next-chance should find a shot, state=%s min=%.1f shots=%d", e.State, e.CurrentMinute, len(e.LiveShots))
		}
	}
	if e.CurrentMinute <= 0 {
		t.Fatal("seek should advance the clock")
	}
	if e.InstanceID != before {
		t.Fatal("seek must never rotate the commit instance")
	}
}

func TestSeekIdleStatesDoNothing(t *testing.T) {
	e := seekEngine()
	e.State = "NOT_STARTED"
	if e.SeekTo70() || e.SeekNextChance() {
		t.Fatal("seeks before kickoff must be no-ops")
	}
	e.State = "FULL_TIME"
	if e.SeekTo70() || e.SeekNextChance() {
		t.Fatal("seeks after full time must be no-ops")
	}
}
