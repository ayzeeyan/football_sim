package tournament

import (
	"reflect"
	"testing"
)

type fixtureIntegrityClubState struct {
	Played int
	Won    int
	Drawn  int
	Lost   int
	GF     int
	GA     int
	Points int
}

func fixtureIntegritySnapshot(tm *TournamentManager) map[string]fixtureIntegrityClubState {
	out := make(map[string]fixtureIntegrityClubState, len(tm.ClubsList))
	for _, club := range tm.ClubsList {
		out[club.ClubID] = fixtureIntegrityClubState{
			Played: club.Played, Won: club.Won, Drawn: club.Drawn, Lost: club.Lost,
			GF: club.GoalsFor, GA: club.GoalsAgainst, Points: club.Points,
		}
	}
	return out
}

func TestFixtureIntegrityCompletedMatchweekIsIdempotent(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	first := tm.SimulateMatchweek(1)
	if first["status"] != "success" {
		t.Fatalf("first matchweek simulation failed: %v", first)
	}
	before := fixtureIntegritySnapshot(tm)
	finishedBefore := 0
	resultsBefore := make(map[string][2]int)
	for i := range tm.Fixtures {
		f := &tm.Fixtures[i]
		if f.Matchweek != 1 || f.Status != "finished" {
			continue
		}
		if f.HomeGoals == nil || f.AwayGoals == nil {
			t.Fatalf("finished fixture %s has no score", f.FixtureID)
		}
		finishedBefore++
		resultsBefore[f.FixtureID] = [2]int{*f.HomeGoals, *f.AwayGoals}
	}
	if finishedBefore != len(tm.ClubsList)/2 {
		t.Fatalf("matchweek 1 finished fixtures=%d, want %d", finishedBefore, len(tm.ClubsList)/2)
	}

	_ = tm.SimulateMatchweek(1)
	after := fixtureIntegritySnapshot(tm)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("replaying completed matchweek mutated standings\nbefore=%+v\nafter=%+v", before, after)
	}
	for i := range tm.Fixtures {
		f := &tm.Fixtures[i]
		if result, ok := resultsBefore[f.FixtureID]; ok {
			if f.Status != "finished" || f.HomeGoals == nil || f.AwayGoals == nil || *f.HomeGoals != result[0] || *f.AwayGoals != result[1] {
				t.Fatalf("replaying completed matchweek changed fixture %s", f.FixtureID)
			}
		}
	}
}

func TestFixtureIntegrityStandingsMatchFinishedFixtures(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	for mw := 1; mw <= 5; mw++ {
		if result := tm.SimulateMatchweek(mw); result["status"] != "success" {
			t.Fatalf("simulate matchweek %d failed: %v", mw, result)
		}
	}

	type expected struct{ played, won, drawn, lost, gf, ga, points int }
	want := make(map[string]expected, len(tm.ClubsList))
	for i := range tm.Fixtures {
		f := &tm.Fixtures[i]
		if f.Status != "finished" {
			continue
		}
		if f.HomeGoals == nil || f.AwayGoals == nil {
			t.Fatalf("finished fixture %s has missing result", f.FixtureID)
		}
		if f.HomeID == f.AwayID || tm.Clubs[f.HomeID] == nil || tm.Clubs[f.AwayID] == nil {
			t.Fatalf("fixture %s has invalid club references", f.FixtureID)
		}
		h := want[f.HomeID]
		a := want[f.AwayID]
		h.played++
		a.played++
		h.gf += *f.HomeGoals
		h.ga += *f.AwayGoals
		a.gf += *f.AwayGoals
		a.ga += *f.HomeGoals
		switch {
		case *f.HomeGoals > *f.AwayGoals:
			h.won++
			h.points += 3
			a.lost++
		case *f.HomeGoals < *f.AwayGoals:
			a.won++
			a.points += 3
			h.lost++
		default:
			h.drawn++
			a.drawn++
			h.points++
			a.points++
		}
		want[f.HomeID] = h
		want[f.AwayID] = a
	}

	for _, club := range tm.ClubsList {
		e := want[club.ClubID]
		if club.Played != e.played || club.Won != e.won || club.Drawn != e.drawn || club.Lost != e.lost || club.GoalsFor != e.gf || club.GoalsAgainst != e.ga || club.Points != e.points {
			t.Fatalf("standings mismatch for %s: got P/W/D/L/GF/GA/Pts=%d/%d/%d/%d/%d/%d/%d want %d/%d/%d/%d/%d/%d/%d", club.ClubID, club.Played, club.Won, club.Drawn, club.Lost, club.GoalsFor, club.GoalsAgainst, club.Points, e.played, e.won, e.drawn, e.lost, e.gf, e.ga, e.points)
		}
	}
	if err := tm.ValidateWorldState(); err != nil {
		t.Fatalf("fixture-consistent world failed validation: %v", err)
	}
}
