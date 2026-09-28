package persistence

import (
	"path/filepath"
	"testing"

	"football_sim/pkg/datamanager"
	"football_sim/pkg/growth"
	"football_sim/pkg/tournament"
	"football_sim/pkg/transfers"
)

// setupEuropeanWorld builds a full five-league world (96 clubs, national
// teams, European fields) for tests that need the complete career shape.
func setupEuropeanWorld(t *testing.T) (*growth.GrowthEngine, *tournament.TournamentManager, *transfers.TransferEngine) {
	t.Helper()
	ge := growth.NewGrowthEngine(8181)
	dm := datamanager.NewDataManager(filepath.Join("..", "..", "..", "dataset.json"), ge)
	if len(dm.ClubsList) != 96 {
		t.Fatalf("dataset clubs=%d want 96", len(dm.ClubsList))
	}
	tm := tournament.NewEuropeanWorldManager(dm.ClubsList, ge, 8181)
	te := transfers.NewTransferEngine(dm.ClubsList, tm.Managers, 9191)
	tm.TransferEngine = te
	return ge, tm, te
}
