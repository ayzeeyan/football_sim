package tournament

import (
	"testing"
)

func directedKey(home, away string) string {
	return home + ">" + away
}

func pairKey(a, b string) string {
	if a < b {
		return a + "|" + b
	}
	return b + "|" + a
}

func cycleFixtures(fixtures []Fixture, lo, hi int) []Fixture {
	out := make([]Fixture, 0, 66)
	for _, f := range fixtures {
		if f.Matchweek >= lo && f.Matchweek <= hi {
			out = append(out, f)
		}
	}
	return out
}

func directedSet(fixtures []Fixture) map[string]int {
	out := map[string]int{}
	for _, f := range fixtures {
		out[directedKey(f.HomeID, f.AwayID)]++
	}
	return out
}

func TestPacket9QuadrupleRoundRobinShape(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	if len(tm.Fixtures) != 44*6 {
		t.Fatalf("league fixtures = %d; want %d", len(tm.Fixtures), 44*6)
	}
	perWeek := map[int]int{}
	homes := map[string]int{}
	pairs := map[string]int{}
	for _, f := range tm.Fixtures {
		if f.Competition != "super-league" {
			t.Fatalf("league calendar contains %q", f.Competition)
		}
		perWeek[f.Matchweek]++
		homes[f.HomeID]++
		pairs[pairKey(f.HomeID, f.AwayID)]++
		if f.HomeID == "" || f.AwayID == "" || f.HomeID == f.AwayID {
			t.Fatalf("invalid pairing: %+v", f)
		}
	}
	if len(perWeek) != 44 {
		t.Fatalf("matchweeks = %d; want 44", len(perWeek))
	}
	for mw := 1; mw <= 44; mw++ {
		if perWeek[mw] != 6 {
			t.Errorf("MW %d has %d league games; want 6", mw, perWeek[mw])
		}
	}
	if got := LeaguePhase(1); got != "Opening series" {
		t.Fatalf("MW 1 phase = %q; want Opening series", got)
	}
	if got := LeaguePhase(12); got != "Return series" {
		t.Fatalf("MW 12 phase = %q; want Return series", got)
	}
	if got := LeaguePhase(23); got != "Third series" {
		t.Fatalf("MW 23 phase = %q; want Third series", got)
	}
	if got := LeaguePhase(34); got != "Final stretch" {
		t.Fatalf("MW 34 phase = %q; want Final stretch", got)
	}
	for _, f := range cycleFixtures(tm.Fixtures, 34, 44) {
		if f.Stage != "Final stretch" {
			t.Fatalf("MW %d stage = %q; want Final stretch", f.Matchweek, f.Stage)
		}
	}
	if len(pairs) != 66 {
		t.Fatalf("unique pairs = %d; want 66", len(pairs))
	}
	for pair, n := range pairs {
		if n != 4 {
			t.Errorf("pair %s played %d times; want 4", pair, n)
		}
	}
	for _, club := range tm.ClubsList {
		h := homes[club.ClubID]
		if h != 22 {
			t.Errorf("%s home games = %d; want exactly 22", club.ShortName, h)
		}
	}
}

func TestPacket9CycleVenuesAndSymmetry(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	c1 := directedSet(cycleFixtures(tm.Fixtures, 1, 11))
	c2 := directedSet(cycleFixtures(tm.Fixtures, 12, 22))
	c3 := directedSet(cycleFixtures(tm.Fixtures, 23, 33))
	c4 := directedSet(cycleFixtures(tm.Fixtures, 34, 44))

	if len(c1) != 66 || len(c2) != 66 || len(c3) != 66 || len(c4) != 66 {
		t.Fatalf("cycle sizes c1=%d c2=%d c3=%d c4=%d; want 66 directed meetings each", len(c1), len(c2), len(c3), len(c4))
	}

	// Cycle 1 equals Cycle 3 (H -> A)
	for key, count := range c1 {
		if c3[key] != count {
			t.Errorf("Cycle 3 directed key %s = %d; want %d (matching Cycle 1)", key, c3[key], count)
		}
	}

	// Cycle 2 equals Cycle 4 (A -> H)
	for key, count := range c2 {
		if c4[key] != count {
			t.Errorf("Cycle 4 directed key %s = %d; want %d (matching Cycle 2)", key, c4[key], count)
		}
	}

	// Across all 4 cycles, every directed matchup A > B occurs exactly 2 times
	allDirected := directedSet(tm.Fixtures)
	for key, count := range allDirected {
		if count != 2 {
			t.Errorf("directed matchup %s count = %d; want exactly 2", key, count)
		}
	}
}

func TestPacket9AdoptLongSeasonKeepsOpeningResultsWhenAddingHomeStretch(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	short := tm.Fixtures[:22*6]
	kept := 0
	for i := range short {
		if short[i].Matchweek == 1 {
			g := 2
			short[i].Status = "finished"
			short[i].HomeGoals = &g
			short[i].AwayGoals = new(int)
			kept++
		}
	}
	tm.Fixtures = short
	tm.MaxMatchweeks = 22
	tm.SuperCupFixtures = tm.SuperCupFixtures[:0]
	if !tm.AdoptLongSeason() {
		t.Fatal("22-week calendar should stretch to 44 weeks")
	}
	if tm.MaxMatchweeks != 44 || len(tm.Fixtures) != 44*6 {
		t.Fatalf("stretched calendar weeks=%d fixtures=%d; want weeks=44 fixtures=264", tm.MaxMatchweeks, len(tm.Fixtures))
	}
	finished := 0
	for _, f := range tm.Fixtures {
		if f.Matchweek == 1 && f.Status == "finished" {
			finished++
		}
	}
	if finished != kept || kept == 0 {
		t.Fatalf("opening results dropped on stretch: kept=%d finished=%d", kept, finished)
	}
	stretch := 0
	for _, f := range tm.Fixtures {
		if f.Matchweek >= 34 && f.Matchweek <= 44 {
			stretch++
		}
	}
	if stretch != 11*6 {
		t.Fatalf("final stretch games = %d; want 66", stretch)
	}
}
