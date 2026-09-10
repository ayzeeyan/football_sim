package tournament

import (
	"path/filepath"
	"reflect"
	"testing"

	"football_sim/pkg/datamanager"
	"football_sim/pkg/growth"
)

type deterministicFixtureResult struct {
	ID        string
	HomeGoals int
	AwayGoals int
	Status    string
}

type deterministicTableRow struct {
	ClubID string
	P      int
	W      int
	D      int
	L      int
	GF     int
	GA     int
	Pts    int
}

func deterministicUniverse(t *testing.T, rootSeed int64) *TournamentManager {
	t.Helper()
	streams := NewSubsystemRNG(rootSeed)
	ge := growth.NewGrowthEngine(streams.SeedFor("development"))
	dm := datamanager.NewDataManager(filepath.Join("..", "..", "..", "dataset.json"), ge)
	if dm == nil || len(dm.Clubs) == 0 {
		t.Fatal("failed to load deterministic test dataset")
	}
	return NewTournamentManager(dm.GetEliteClubs(), ge, streams.SeedFor("matches"))
}

func deterministicSnapshot(tm *TournamentManager) ([]deterministicFixtureResult, []deterministicTableRow) {
	fixtures := make([]deterministicFixtureResult, 0)
	for i := range tm.Fixtures {
		f := &tm.Fixtures[i]
		if f.Status != "finished" || f.HomeGoals == nil || f.AwayGoals == nil {
			continue
		}
		fixtures = append(fixtures, deterministicFixtureResult{
			ID: f.FixtureID, HomeGoals: *f.HomeGoals, AwayGoals: *f.AwayGoals, Status: f.Status,
		})
	}
	table := make([]deterministicTableRow, 0, len(tm.ClubsList))
	for _, club := range tm.GetStandings() {
		table = append(table, deterministicTableRow{
			ClubID: club.ClubID, P: club.Played, W: club.Won, D: club.Drawn, L: club.Lost,
			GF: club.GoalsFor, GA: club.GoalsAgainst, Pts: club.Points,
		})
	}
	return fixtures, table
}

func TestSameSeedSameActionsProduceSameMeaningfulOutput(t *testing.T) {
	a := deterministicUniverse(t, 918273645)
	b := deterministicUniverse(t, 918273645)

	outA := a.SimulateBatchWeeks(8)
	outB := b.SimulateBatchWeeks(8)
	if outA.Status != "success" || outB.Status != "success" {
		t.Fatalf("deterministic simulations failed: A=%+v B=%+v", outA, outB)
	}
	fa, ta := deterministicSnapshot(a)
	fb, tb := deterministicSnapshot(b)
	if !reflect.DeepEqual(fa, fb) {
		t.Fatalf("same-seed fixture results diverged\nA=%+v\nB=%+v", fa, fb)
	}
	if !reflect.DeepEqual(ta, tb) {
		t.Fatalf("same-seed standings diverged\nA=%+v\nB=%+v", ta, tb)
	}
	if a.CurrentMatchweek != b.CurrentMatchweek {
		t.Fatalf("same-seed macro progression diverged: %d vs %d", a.CurrentMatchweek, b.CurrentMatchweek)
	}
}

func TestDifferentSeedsAreNotForcedToIdenticalOutput(t *testing.T) {
	a := deterministicUniverse(t, 101)
	b := deterministicUniverse(t, 202)
	if out := a.SimulateBatchWeeks(8); out.Status != "success" {
		t.Fatalf("seed A simulation failed: %+v", out)
	}
	if out := b.SimulateBatchWeeks(8); out.Status != "success" {
		t.Fatalf("seed B simulation failed: %+v", out)
	}
	fa, ta := deterministicSnapshot(a)
	fb, tb := deterministicSnapshot(b)
	if reflect.DeepEqual(fa, fb) && reflect.DeepEqual(ta, tb) {
		t.Fatal("different seeds were forced into identical match and table output")
	}
}
