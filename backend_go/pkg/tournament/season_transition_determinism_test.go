package tournament

import (
	"reflect"
	"sort"
	"testing"
)

type seasonDeterminismSnapshot struct {
	LeagueScores []string
	CupScores    []string
	Managers     []string
	AcademyIDs   []string
}

func completedFixtureSignatures(tm *TournamentManager) (league, cups []string) {
	fixtureSig := func(f Fixture) string {
		hg, ag := -1, -1
		if f.HomeGoals != nil { hg = *f.HomeGoals }
		if f.AwayGoals != nil { ag = *f.AwayGoals }
		return f.FixtureID + ":" + f.Status + ":" + string(rune(hg+65)) + ":" + string(rune(ag+65))
	}
	for _, f := range tm.Fixtures { league = append(league, fixtureSig(f)) }
	for _, f := range tm.UCLFixtures { cups = append(cups, fixtureSig(f)) }
	for _, f := range tm.SuperCupFixtures { cups = append(cups, fixtureSig(f)) }
	sort.Strings(league)
	sort.Strings(cups)
	return league, cups
}

func transitionDeterminismSnapshot(t *testing.T, seed int64) seasonDeterminismSnapshot {
	t.Helper()
	tm := deterministicUniverse(t, seed)
	res := tm.SimulateBatchWeeks(LeagueRounds)
	if res.Status == "error" { t.Fatalf("season simulation failed: %+v", res) }
	league, cups := completedFixtureSignatures(tm)
	managers := make([]string, 0, len(tm.ClubsList))
	for _, c := range tm.ClubsList {
		m := tm.Managers[c.ClubID]
		if m != nil { managers = append(managers, c.ClubID+":"+m.Name+":"+m.Style) }
	}
	sort.Strings(managers)
	if got := tm.ResetNewSeason(); got["status"] != "success" { t.Fatalf("reset failed: %+v", got) }
	academy := []string{}
	for _, c := range tm.ClubsList {
		for _, p := range c.Squad {
			if p != nil && p.PlayerSource == "academy" { academy = append(academy, c.ClubID+":"+p.PlayerID) }
		}
	}
	sort.Strings(academy)
	return seasonDeterminismSnapshot{LeagueScores: league, CupScores: cups, Managers: managers, AcademyIDs: academy}
}

func TestSeasonTransitionDeterminismBoundary(t *testing.T) {
	a := transitionDeterminismSnapshot(t, 424242)
	b := transitionDeterminismSnapshot(t, 424242)
	if !reflect.DeepEqual(a.LeagueScores, b.LeagueScores) { t.Fatal("same-seed league fixture results diverged before season transition") }
	if !reflect.DeepEqual(a.CupScores, b.CupScores) { t.Fatal("same-seed cup fixture results diverged before season transition") }
	if !reflect.DeepEqual(a.Managers, b.Managers) { t.Fatal("same-seed manager state diverged before season transition") }
	if !reflect.DeepEqual(a.AcademyIDs, b.AcademyIDs) { t.Fatalf("same-seed academy intake diverged at season transition\nA=%v\nB=%v", a.AcademyIDs, b.AcademyIDs) }
}
