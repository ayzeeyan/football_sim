package matchengine

import (
	"strings"
	"testing"

	"football_sim/pkg/managers"
	"football_sim/pkg/models"
)

func personalityKid(id, name, personality string) *models.Player {
	return &models.Player{
		PlayerID: id, FullName: name, Position: "ST", Category: "FWD",
		OVR: 70, Age: 14, UniverseWonderkid: true, Personality: personality,
		Composure: 75, Education: "middle_school",
	}
}

func personalityClubs() (*models.Club, *models.Club) {
	mk := func(id, short string, kids ...*models.Player) *models.Club {
		squad := []*models.Player{
			{PlayerID: id + "-GK", FullName: short + " Keeper", Category: "GK", OVR: 70},
		}
		for i := 0; i < 4; i++ {
			squad = append(squad, &models.Player{PlayerID: id + "-D", FullName: short + " Defender", Category: "DEF", OVR: 68})
		}
		for i := 0; i < 3; i++ {
			squad = append(squad, &models.Player{PlayerID: id + "-M", FullName: short + " Midfielder", Category: "MID", OVR: 69})
		}
		squad = append(squad, kids...)
		for len(squad) < 11 {
			squad = append(squad, &models.Player{PlayerID: id + "-F", FullName: short + " Forward", Category: "FWD", OVR: 69})
		}
		return &models.Club{ClubID: id, ClubName: short + " FC", ShortName: short, HomeStadium: short + " Park", Squad: squad}
	}
	home := mk("H", "HOM", personalityKid("WK-FLAM", "Flashy Finn", "flamboyant_star"))
	away := mk("A", "AWY", personalityKid("WK-DED", "Steady Sam", "dedicated_pro"))
	return home, away
}

func TestFlamboyantMissInMustWin(t *testing.T) {
	home, away := personalityClubs()
	e := NewLiveMatchEngine(home, away, &managers.ManagerProfile{}, &managers.ManagerProfile{}, 5)
	e.HomeScore, e.AwayScore = 0, 1
	kid := personalityKid("WK-FLAM", "Flashy Finn", "flamboyant_star")
	line := e.personalityMissLine(kid, "home", 78)
	if line == "" || !strings.Contains(line, "Flashy Finn") {
		t.Fatalf("must-win flamboyant miss needs a kid-tied line, got %q", line)
	}
	if kid.Composure != 74 {
		t.Fatalf("composure hook should dip to 74, got %d", kid.Composure)
	}
	// Not trailing: no line, no hook.
	e.HomeScore, e.AwayScore = 1, 1
	kid.Composure = 75
	if line := e.personalityMissLine(kid, "home", 78); line != "" {
		t.Fatalf("level score is not must-win, got %q", line)
	}
	if kid.Composure != 75 {
		t.Fatal("no hook outside the situation")
	}
	// Too early: no line.
	e.HomeScore, e.AwayScore = 0, 1
	if line := e.personalityMissLine(kid, "home", 40); line != "" {
		t.Fatalf("40' is not must-win, got %q", line)
	}
	// Non-flamboyant kid: no line even when trailing late.
	plain := personalityKid("WK-PLAIN", "Plain Pete", "dedicated_pro")
	if line := e.personalityMissLine(plain, "home", 78); line != "" {
		t.Fatalf("dedicated kid must not get the flamboyant line, got %q", line)
	}
}

func TestBottlesSuperCupFinal(t *testing.T) {
	home, away := personalityClubs()
	e := NewLiveMatchEngine(home, away, &managers.ManagerProfile{}, &managers.ManagerProfile{}, 5)
	e.Competition = "super-cup"
	e.Stage = "Final"
	kid := personalityKid("WK-DED", "Steady Sam", "dedicated_pro")
	line := e.personalityMissLine(kid, "away", 55)
	if line == "" || !strings.Contains(line, "Steady Sam") {
		t.Fatalf("super-cup final miss needs a bottle line tied to the kid, got %q", line)
	}
	// Same competition, earlier round: silent.
	e.Stage = "Play-in"
	kid.Composure = 75
	if line := e.personalityMissLine(kid, "away", 55); line != "" {
		t.Fatalf("play-in miss must stay generic, got %q", line)
	}
	if kid.Composure != 75 {
		t.Fatal("no hook outside the final")
	}
}

func TestDedicatedKidOnTenMen(t *testing.T) {
	home, away := personalityClubs()
	e := NewLiveMatchEngine(home, away, &managers.ManagerProfile{}, &managers.ManagerProfile{}, 5)
	e.SetClubs(home, away, &managers.ManagerProfile{}, &managers.ManagerProfile{})
	// Away starters include the dedicated kid (away squad built above).
	line := e.dedicatedTenManLine("away", 63)
	if line == "" || !strings.Contains(line, "Steady Sam") {
		t.Fatalf("10-man dedicated kid needs a rally line, got %q", line)
	}
	// Home has no dedicated kid (only flamboyant): silent.
	if line := e.dedicatedTenManLine("home", 63); line != "" {
		t.Fatalf("side without a dedicated kid must stay silent, got %q", line)
	}
}
