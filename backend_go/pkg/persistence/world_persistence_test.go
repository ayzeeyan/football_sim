package persistence

import (
	"path/filepath"
	"testing"

	"football_sim/pkg/datamanager"
	"football_sim/pkg/growth"
	"football_sim/pkg/models"
	"football_sim/pkg/tournament"
	"football_sim/pkg/transfers"
)

func europeanPersistenceWorld(t *testing.T) (*tournament.TournamentManager, *growth.GrowthEngine, *transfers.TransferEngine) {
	t.Helper()
	ge := growth.NewGrowthEngine(704)
	dm := datamanager.NewDataManager(filepath.Join("..", "..", "..", "dataset.json"), ge)
	tm := tournament.NewEuropeanWorldManager(dm.ClubsList, ge, 704)
	te := transfers.NewTransferEngine(dm.ClubsList, tm.Managers, 705)
	tm.TransferEngine = te
	return tm, ge, te
}

func TestEuropeanWorldSnapshotRestoresSharedCompetitionState(t *testing.T) {
	tm, ge, te := europeanPersistenceWorld(t)
	_ = tm.SimulateRemaining()
	_ = tm.SimulateRemaining()
	_ = tm.SimulateRemaining()
	path := filepath.Join(t.TempDir(), "career.json")
	if _, err := SaveCareer(tm, ge, te, path); err != nil {
		t.Fatalf("save world: %v", err)
	}
	snap, err := LoadCareer(path)
	if err != nil {
		t.Fatalf("load world: %v", err)
	}
	if snap.Version != SaveVersion || snap.World == nil || len(snap.World.Fixtures) == 0 {
		t.Fatalf("world snapshot missing: version=%d world=%v fixtures=%d", snap.Version, snap.World != nil, len(snap.World.Fixtures))
	}
	if err := ValidateCareerSnapshot(snap); err != nil {
		t.Fatalf("validate saved world: %v", err)
	}
	fresh, freshGE, freshTE := europeanPersistenceWorld(t)
	if err := RestoreCareer(fresh, freshGE, freshTE, snap); err != nil {
		t.Fatalf("restore world: %v", err)
	}
	if fresh.World == nil || len(fresh.World.Fixtures) != len(tm.World.Fixtures) || fresh.CurrentMatchweek != tm.CurrentMatchweek {
		t.Fatalf("restored world shape differs: fixtures=%d/%d mw=%d/%d", len(fresh.World.Fixtures), len(tm.World.Fixtures), fresh.CurrentMatchweek, tm.CurrentMatchweek)
	}
	if err := fresh.ValidateWorldState(); err != nil {
		t.Fatalf("restored world invalid: %v", err)
	}
}

// New ledgers and draw metadata must survive a save/load cycle: coefficients,
// the European revenue ledger, full finances, loan state plus buy clauses,
// per-competition minutes, and pot/tie scaffolding.
func TestEuropeanWorldSnapshotRestoresNewLedgers(t *testing.T) {
	tm, ge, te := europeanPersistenceWorld(t)
	_ = tm.SimulateRemaining()
	donor := tm.ClubsList[0]
	donor.Coefficient = 17
	donor.Finances.EuropeanRevenue = 8_500_000
	// A synthetic loanee with clause and tracked minutes (never a wonderkid:
	// clauses on prodigies are rejected by validation).
	var loanee *models.Player
	for _, p := range donor.Squad {
		if p != nil && !p.UniverseWonderkid {
			loanee = p
			break
		}
	}
	if loanee == nil {
		t.Fatal("no loanable player in donor squad")
	}
	loanee.OnLoan = true
	loanee.ParentClubID = "ZZ-PARENT"
	loanee.LoanBuyClauseEUR = 12_000_000
	loanee.CompetitionStats = map[string]*models.CompetitionSeasonStats{
		"premier-league": {CompetitionID: "premier-league", Appearances: 3, Starts: 2, Minutes: 200, Goals: 1},
	}
	path := filepath.Join(t.TempDir(), "career.json")
	if _, err := SaveCareer(tm, ge, te, path); err != nil {
		t.Fatalf("save world: %v", err)
	}
	snap, err := LoadCareer(path)
	if err != nil {
		t.Fatalf("load world: %v", err)
	}
	if err := ValidateCareerSnapshot(snap); err != nil {
		t.Fatalf("validate saved world: %v", err)
	}
	fresh, freshGE, freshTE := europeanPersistenceWorld(t)
	if err := RestoreCareer(fresh, freshGE, freshTE, snap); err != nil {
		t.Fatalf("restore world: %v", err)
	}
	got := fresh.Clubs[donor.ClubID]
	if got.Coefficient != 17 {
		t.Fatalf("coefficient=%d want 17", got.Coefficient)
	}
	if got.Finances.EuropeanRevenue != 8_500_000 {
		t.Fatalf("european revenue=%d want 8500000", got.Finances.EuropeanRevenue)
	}
	found := false
	for _, p := range got.Squad {
		if p.PlayerID == loanee.PlayerID {
			found = true
			if !p.OnLoan || p.ParentClubID != "ZZ-PARENT" || p.LoanBuyClauseEUR != 12_000_000 {
				t.Fatalf("loan state lost: on=%v parent=%q clause=%d", p.OnLoan, p.ParentClubID, p.LoanBuyClauseEUR)
			}
			row := p.CompetitionStats["premier-league"]
			if row == nil || row.Minutes != 200 || row.Goals != 1 {
				t.Fatalf("competition minutes lost: %+v", row)
			}
		}
	}
	if !found {
		t.Fatal("loanee missing after restore")
	}
	ucl := fresh.World.Competitions["champions-league"]
	if len(ucl.Pots) != 4 {
		t.Fatalf("pots lost on restore: %d", len(ucl.Pots))
	}
}
