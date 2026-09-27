package persistence

import (
	"fmt"
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

func TestRestoreLegacyWorldAdoptsNationalCompetition(t *testing.T) {
	tm, ge, te := europeanPersistenceWorld(t)
	snap := BuildSnapshot(tm, ge, te)
	snap.Version = 7
	snap.World.NationalTeams = nil
	fresh, freshGE, freshTE := europeanPersistenceWorld(t)
	if err := RestoreCareer(fresh, freshGE, freshTE, snap); err != nil {
		t.Fatalf("restore legacy world: %v", err)
	}
	if fresh.World == nil || fresh.World.NationalTeams == nil || len(fresh.World.NationalTeams.TeamOrder) != 5 {
		t.Fatalf("national competition missing after migration: %+v", fresh.World)
	}
	if err := fresh.ValidateWorldState(); err != nil {
		t.Fatalf("migrated world invalid: %v", err)
	}
}

func TestRestoreCareerReconcilesCaptainFlagsWithClubIDs(t *testing.T) {
	tm, ge, te := europeanPersistenceWorld(t)
	club := tm.Clubs["EPL-ARS"]
	if club == nil || len(club.Squad) < 2 {
		t.Fatal("need Arsenal squad")
	}
	a, b := club.Squad[0], club.Squad[1]
	club.CaptainID = a.PlayerID
	club.ViceCaptainID = b.PlayerID
	club.Chemistry = 77
	club.MediaPressure = 41
	club.FanExpectation = 80
	a.IsCaptain = false
	b.IsCaptain = true // stale flag, disagrees with captain_id
	b.IsViceCaptain = false

	path := filepath.Join(t.TempDir(), "career.json")
	if _, err := SaveCareer(tm, ge, te, path); err != nil {
		t.Fatalf("save: %v", err)
	}
	snap, err := LoadCareer(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	fresh, freshGE, freshTE := europeanPersistenceWorld(t)
	if err := RestoreCareer(fresh, freshGE, freshTE, snap); err != nil {
		t.Fatalf("restore: %v", err)
	}
	got := fresh.Clubs["EPL-ARS"]
	if got.CaptainID != a.PlayerID {
		t.Fatalf("captain_id=%s want %s", got.CaptainID, a.PlayerID)
	}
	if got.Chemistry != 77 || got.MediaPressure != 41 || got.FanExpectation != 80 {
		t.Fatalf("culture fields not restored: chem=%d media=%d fans=%d", got.Chemistry, got.MediaPressure, got.FanExpectation)
	}
	var flagged string
	for _, p := range got.Squad {
		if p.IsCaptain {
			if flagged != "" {
				t.Fatalf("multiple captains: %s and %s", flagged, p.PlayerID)
			}
			flagged = p.PlayerID
		}
	}
	if flagged != a.PlayerID {
		t.Fatalf("is_captain on %s want %s", flagged, a.PlayerID)
	}
	if err := fresh.ValidateWorldState(); err != nil {
		t.Fatalf("restored world invalid: %v", err)
	}
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

func TestRestoreExpiredWonderkidContractLoads(t *testing.T) {
	tm, ge, te := europeanPersistenceWorld(t)
	var kid *models.Player
	for _, club := range tm.ClubsList {
		for _, p := range club.Squad {
			if p != nil && p.PlayerID == "WK_Izyan_Levin_Bantol" {
				kid = p
			}
		}
	}
	if kid == nil {
		t.Fatal("expected WK_Izyan_Levin_Bantol in the European world")
	}
	kid.Age = 20
	kid.ContractYears = 0
	kid.Loyalty = 80
	kid.Personality = "dedicated_pro"
	kid.TransferRequested = false

	path := filepath.Join(t.TempDir(), "career.json")
	if _, err := SaveCareer(tm, ge, te, path); err != nil {
		t.Fatalf("save: %v", err)
	}
	snap, err := LoadCareer(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if err := ValidateCareerSnapshot(snap); err != nil {
		t.Fatalf("snapshot validation: %v", err)
	}
	fresh, freshGE, freshTE := europeanPersistenceWorld(t)
	if err := RestoreCareer(fresh, freshGE, freshTE, snap); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if err := fresh.ValidateWorldState(); err != nil {
		t.Fatalf("restored world invalid: %v", err)
	}
	var restored *models.Player
	for _, club := range fresh.ClubsList {
		for _, p := range club.Squad {
			if p != nil && p.PlayerID == "WK_Izyan_Levin_Bantol" {
				restored = p
			}
		}
	}
	if restored == nil {
		t.Fatal("wonderkid missing after restore")
	}
	if restored.ContractYears != 0 {
		t.Fatalf("expired deal must survive restore unchanged, years=%d", restored.ContractYears)
	}
}

func TestRestoreCareerUsesSavedSquadNotDatasetOverlay(t *testing.T) {
	tm, ge, te := europeanPersistenceWorld(t)
	inter := tm.Clubs["SEA-INT"]
	if inter == nil || len(inter.Squad) < 3 {
		t.Fatal("need Inter squad")
	}
	departed := inter.Squad[len(inter.Squad)-1]
	inter.Squad = inter.Squad[:len(inter.Squad)-1]
	academy := &models.Player{
		PlayerID: "AC_SEA-INT_Test_Regen_1001", FullName: "Test Regen", Position: "CM",
		Category: "MID", OVR: 62, Age: 17, ClubID: inter.ClubID, OriginalClubID: inter.ClubID,
		ContractYears: 3, WageEUR: 10000, MarketValueEUR: models.MinPlayerValueEUR,
	}
	inter.Squad = append(inter.Squad, academy)
	want := len(inter.Squad)

	path := filepath.Join(t.TempDir(), "career.json")
	if _, err := SaveCareer(tm, ge, te, path); err != nil {
		t.Fatalf("save: %v", err)
	}
	snap, err := LoadCareer(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	fresh, freshGE, freshTE := europeanPersistenceWorld(t)
	if err := RestoreCareer(fresh, freshGE, freshTE, snap); err != nil {
		t.Fatalf("restore: %v", err)
	}
	got := fresh.Clubs["SEA-INT"]
	if len(got.Squad) != want {
		t.Fatalf("Inter squad=%d want saved %d (dataset overlay leaked departed players)", len(got.Squad), want)
	}
	for _, p := range got.Squad {
		if p.PlayerID == departed.PlayerID {
			t.Fatalf("departed dataset player %s still on Inter after restore", departed.PlayerID)
		}
	}
	foundAcademy := false
	for _, p := range got.Squad {
		if p.PlayerID == academy.PlayerID {
			foundAcademy = true
		}
	}
	if !foundAcademy {
		t.Fatal("academy signing missing after restore")
	}
	if err := fresh.ValidateWorldState(); err != nil {
		t.Fatalf("restored world invalid: %v", err)
	}
}

func TestRestoreCareerTrimsSavedSquadOverflow(t *testing.T) {
	tm, ge, te := europeanPersistenceWorld(t)
	inter := tm.Clubs["SEA-INT"]
	if inter == nil {
		t.Fatal("missing Inter")
	}
	for len(inter.Squad) < models.MaxSeniorSquadSize+2 {
		i := len(inter.Squad)
		inter.Squad = append(inter.Squad, &models.Player{
			PlayerID: fmt.Sprintf("AC_SEA-INT_Overflow_%02d", i), FullName: fmt.Sprintf("Overflow %d", i),
			Position: "CM", Category: "MID", OVR: 58, Age: 18, ClubID: inter.ClubID,
			OriginalClubID: inter.ClubID, ContractYears: 2, WageEUR: 8000,
			MarketValueEUR: models.MinPlayerValueEUR,
		})
	}
	path := filepath.Join(t.TempDir(), "career.json")
	if _, err := SaveCareer(tm, ge, te, path); err != nil {
		t.Fatalf("save: %v", err)
	}
	snap, err := LoadCareer(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	fresh, freshGE, freshTE := europeanPersistenceWorld(t)
	if err := RestoreCareer(fresh, freshGE, freshTE, snap); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if got := len(fresh.Clubs["SEA-INT"].Squad); got > models.MaxSeniorSquadSize {
		t.Fatalf("Inter still over cap after restore: %d", got)
	}
	if err := fresh.ValidateWorldState(); err != nil {
		t.Fatalf("restored world invalid: %v", err)
	}
}
