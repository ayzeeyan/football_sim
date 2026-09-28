package tournament

import (
	"testing"
)

// A swap-era save carries domestic registries that disagree with the
// dataset-derived club leagues (for example Atletico Madrid registered to
// the Premier League). Restoring such a world must realign it to the
// country-pure pyramid and restart the season.
func TestRealignCountryPureWorldFixesSwapEraRegistry(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)

	// Simulate the user's failing save: Atletico Madrid is registered to the
	// Premier League while its dataset league is La Liga.
	epl := tm.World.Competitions["premier-league"]
	lal := tm.World.Competitions["la-liga"]
	if epl == nil || lal == nil {
		t.Fatal("domestic leagues missing from the fresh world")
	}
	atm := "LAL-ATM"
	found := false
	for _, id := range lal.ParticipantIDs {
		if id == atm {
			found = true
		}
	}
	if !found {
		t.Fatal("test world does not contain LAL-ATM in La Liga")
	}
	tm.mu.Lock()
	epl.ParticipantIDs = append(epl.ParticipantIDs, atm)
	lal.ParticipantIDs = removeStringFromSlice(lal.ParticipantIDs, atm)
	tm.mu.Unlock()

	// The world no longer validates.
	if err := tm.ValidateWorldState(); err == nil {
		t.Fatal("corrupted registry must fail world validation")
	}

	// Realignment fixes it.
	if !tm.RealignCountryPureWorld() {
		t.Fatal("realignment must trigger for a misaligned registry")
	}
	if err := tm.ValidateWorldState(); err != nil {
		t.Fatalf("world invalid after realignment: %v", err)
	}
	// Atletico is back in La Liga's registry only.
	tm.mu.RLock()
	epl = tm.World.Competitions["premier-league"]
	lal = tm.World.Competitions["la-liga"]
	tm.mu.RUnlock()
	for _, id := range epl.ParticipantIDs {
		if id == atm {
			t.Fatal("Atletico still registered to the Premier League after realignment")
		}
	}
	found = false
	for _, id := range lal.ParticipantIDs {
		if id == atm {
			found = true
		}
	}
	if !found {
		t.Fatal("Atletico missing from La Liga after realignment")
	}
	// The season restarted from matchweek 1 with zeroed tables.
	if tm.CurrentMatchweek != 1 || tm.SeasonPhase != "season" {
		t.Fatalf("season state=%d/%q want 1/season", tm.CurrentMatchweek, tm.SeasonPhase)
	}
	for _, club := range tm.ClubsList {
		if club.Played != 0 || club.Points != 0 {
			t.Fatalf("%s kept stale standings after realignment", club.ClubID)
		}
	}
	// The realignment is idempotent.
	if tm.RealignCountryPureWorld() {
		t.Fatal("realignment must not trigger twice")
	}
}

// The international competition survives the realignment with its history
// and viewer squads intact.
func TestRealignPreservesNationalTeams(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	tm.mu.Lock()
	if tm.World.NationalTeams == nil {
		t.Fatal("fresh world has no national teams")
	}
	tm.World.NationalTeams.History = append(tm.World.NationalTeams.History, NationalTeamSeason{
		Season: "2020-21", ChampionID: "england",
	})
	// Corrupt the domestic registry so the realignment path runs.
	epl := tm.World.Competitions["premier-league"]
	epl.ParticipantIDs = append(epl.ParticipantIDs, "LAL-ATM")
	tm.mu.Unlock()

	if !tm.RealignCountryPureWorld() {
		t.Fatal("realignment must trigger")
	}
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	if tm.World.NationalTeams == nil {
		t.Fatal("national teams lost in realignment")
	}
	if len(tm.World.NationalTeams.History) != 1 || tm.World.NationalTeams.History[0].Season != "2020-21" {
		t.Fatalf("nations history lost: %+v", tm.World.NationalTeams.History)
	}
}

func removeStringFromSlice(rows []string, target string) []string {
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		if row != target {
			out = append(out, row)
		}
	}
	return out
}
