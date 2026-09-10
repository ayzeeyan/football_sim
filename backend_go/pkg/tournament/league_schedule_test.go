package tournament

import (
	"fmt"
	"math/rand"
	"testing"

	"football_sim/pkg/models"
)

func TestLeagueScheduleRealUniverseIntegrity(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	clubs := len(tm.ClubsList)
	if clubs != 12 {
		t.Fatalf("test universe has %d clubs; want 12", clubs)
	}
	wantFixtures := clubs * (clubs - 1)
	wantRounds := 2 * (clubs - 1)
	wantPerRound := clubs / 2
	if len(tm.Fixtures) != wantFixtures {
		t.Fatalf("fixtures=%d want %d", len(tm.Fixtures), wantFixtures)
	}
	if tm.MaxMatchweeks != wantRounds {
		t.Fatalf("MaxMatchweeks=%d want %d", tm.MaxMatchweeks, wantRounds)
	}

	perWeek := map[int][]Fixture{}
	orderedPair := map[string]int{}
	clubGames := map[string]int{}
	for _, f := range tm.Fixtures {
		if f.HomeID == f.AwayID {
			t.Fatalf("self fixture %s", f.FixtureID)
		}
		perWeek[f.Matchweek] = append(perWeek[f.Matchweek], f)
		orderedPair[fmt.Sprintf("%s>%s", f.HomeID, f.AwayID)]++
		clubGames[f.HomeID]++
		clubGames[f.AwayID]++
	}
	if len(perWeek) != wantRounds {
		t.Fatalf("rounds=%d want %d", len(perWeek), wantRounds)
	}
	for mw := 1; mw <= wantRounds; mw++ {
		fixtures := perWeek[mw]
		if len(fixtures) != wantPerRound {
			t.Fatalf("MW%d fixtures=%d want %d", mw, len(fixtures), wantPerRound)
		}
		seen := map[string]bool{}
		for _, f := range fixtures {
			if seen[f.HomeID] || seen[f.AwayID] {
				t.Fatalf("MW%d schedules a club more than once", mw)
			}
			seen[f.HomeID], seen[f.AwayID] = true, true
		}
		if len(seen) != clubs {
			t.Fatalf("MW%d has %d participating clubs want %d", mw, len(seen), clubs)
		}
	}
	for _, c := range tm.ClubsList {
		if clubGames[c.ClubID] != 2*(clubs-1) {
			t.Errorf("%s games=%d want %d", c.ShortName, clubGames[c.ClubID], 2*(clubs-1))
		}
	}
	for i := range tm.ClubsList {
		for j := i + 1; j < len(tm.ClubsList); j++ {
			a, b := tm.ClubsList[i].ClubID, tm.ClubsList[j].ClubID
			if orderedPair[a+">"+b] != 1 || orderedPair[b+">"+a] != 1 {
				t.Fatalf("pair %s/%s is not exactly one home + one away: %d/%d", a, b, orderedPair[a+">"+b], orderedPair[b+">"+a])
			}
		}
	}
}

func TestLeagueScheduleOddClubCountHasExactlyOneByePerRound(t *testing.T) {
	clubs := make([]*models.Club, 5)
	for i := range clubs {
		clubs[i] = &models.Club{ClubID: fmt.Sprintf("C%d", i), ClubName: fmt.Sprintf("Club %d", i)}
	}
	fixtures := GenerateLeagueFixtures(clubs, rand.New(rand.NewSource(1)))
	if len(fixtures) != 5*4 {
		t.Fatalf("fixtures=%d want 20", len(fixtures))
	}
	perWeek := map[int][]Fixture{}
	for _, f := range fixtures {
		perWeek[f.Matchweek] = append(perWeek[f.Matchweek], f)
	}
	if len(perWeek) != 10 {
		t.Fatalf("rounds=%d want 10", len(perWeek))
	}
	for mw := 1; mw <= 10; mw++ {
		seen := map[string]bool{}
		for _, f := range perWeek[mw] {
			seen[f.HomeID], seen[f.AwayID] = true, true
		}
		if len(perWeek[mw]) != 2 || len(seen) != 4 {
			t.Fatalf("MW%d fixtures=%d participating=%d; want 2 and 4 (one bye)", mw, len(perWeek[mw]), len(seen))
		}
	}
}

func TestLeagueScheduleDeterministicWithoutConsumingRNG(t *testing.T) {
	clubs := []*models.Club{{ClubID:"A"},{ClubID:"B"},{ClubID:"C"},{ClubID:"D"}}
	r1 := rand.New(rand.NewSource(99))
	r2 := rand.New(rand.NewSource(99))
	got := GenerateLeagueFixtures(clubs, r1)
	_ = GenerateLeagueFixtures(clubs, r2)
	if len(got) != 12 {
		t.Fatalf("fixtures=%d want 12", len(got))
	}
	if r1.Int63() != r2.Int63() {
		t.Fatal("fixture generation unexpectedly consumed universe RNG")
	}
}
