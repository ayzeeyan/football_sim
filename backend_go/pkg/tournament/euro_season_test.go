package tournament

import (
	"testing"

	"football_sim/pkg/transfers"
)

// TestEuropeanSeasonBanksCoefficientsAndSeparateRevenue plays a full world
// season, then checks: prizes and coefficients bank at the season transition
// (not before); European participants earned coefficient points and tracked
// European revenue (at least the participation floor for the Champions League
// winner); clubs outside Europe earned neither; and coefficients persist into
// the new campaign while the revenue ledger holds exactly the just-finished
// intake.
func TestEuropeanSeasonBanksCoefficientsAndSeparateRevenue(t *testing.T) {
	tm, _, te := loadEuropeanWorldForTest(t)
	for _, c := range tm.ClubsList {
		if c.Coefficient != 0 || c.Finances.EuropeanRevenue != 0 {
			t.Fatalf("fresh world %s has coefficient=%d revenue=%d", c.ClubID, c.Coefficient, c.Finances.EuropeanRevenue)
		}
	}
	batch := tm.SimulateBatchWeeks(38)
	if batch.Status != "success" || !batch.SeasonFinished {
		t.Fatalf("season did not finish: %+v", batch)
	}
	ucl := tm.World.Competitions["champions-league"]
	if ucl.ChampionID == "" {
		t.Fatal("no UCL champion")
	}
	// Banking happens at the transition, alongside domestic prizes.
	for _, c := range tm.ClubsList {
		if c.Coefficient != 0 || c.Finances.EuropeanRevenue != 0 {
			t.Fatalf("%s banked before transition: coefficient=%d revenue=%d", c.ClubID, c.Coefficient, c.Finances.EuropeanRevenue)
		}
	}
	// Season-1 European membership must be captured before the transition
	// rebuilds the competition registry for the new campaign.
	season1Europe := map[string]bool{}
	for _, eid := range europeanDefinitions {
		for _, pid := range tm.World.Competitions[eid.ID].ParticipantIDs {
			season1Europe[pid] = true
		}
	}
	te.BeginOffSeasonWindow()
	for i := 0; i < transfers.TransferWindowWeeks; i++ {
		te.AdvanceOpenWindow()
	}
	if transition := tm.FinalizeSeasonTransition(); transition["status"] != "success" {
		t.Fatalf("transition failed: %v", transition)
	}
	champ := tm.Clubs[ucl.ChampionID]
	if champ.Coefficient <= 0 {
		t.Fatalf("champion %s has no coefficient", champ.ClubID)
	}
	// Participation floor alone guarantees this for any UCL club, and the
	// champion additionally banked rank, knockout, and winner money.
	if champ.Finances.EuropeanRevenue < 12_000_000 {
		t.Fatalf("champion %s European revenue=%d, want >= participation", champ.ClubID, champ.Finances.EuropeanRevenue)
	}
	outsideFound := false
	for _, c := range tm.ClubsList {
		if c.Coefficient < 0 || c.Finances.EuropeanRevenue < 0 {
			t.Fatalf("%s negative coefficient/revenue", c.ClubID)
		}
		if !season1Europe[c.ClubID] {
			outsideFound = true
			if c.Coefficient != 0 || c.Finances.EuropeanRevenue != 0 {
				t.Fatalf("non-European %s has coefficient=%d revenue=%d", c.ClubID, c.Coefficient, c.Finances.EuropeanRevenue)
			}
		}
	}
	if !outsideFound {
		t.Fatal("expected some clubs outside Europe")
	}
	if err := tm.ValidateWorldState(); err != nil {
		t.Fatalf("next world season invalid: %v", err)
	}
	// The rebuilt season re-drew and re-snapshotted its Swiss pots.
	if pots := tm.World.Competitions["champions-league"].Pots; len(pots) != 4 {
		t.Fatalf("rebuilt UCL pots=%d want 4", len(pots))
	}
}

// TestEuropeanRevenueLedgerReplacesNeverAccumulates pins the ledger semantic
// directly: zeroing then banking yields exactly the fresh intake.
func TestEuropeanRevenueLedgerReplacesNeverAccumulates(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	club := tm.ClubsList[0]
	club.Finances.EuropeanRevenue = 999_000_000
	tm.resetEuropeanRevenueLedgerUnlocked()
	if club.Finances.EuropeanRevenue != 0 {
		t.Fatalf("ledger not zeroed: %d", club.Finances.EuropeanRevenue)
	}
	creditEuropeanPrize(club, 5_000_000)
	if club.Finances.EuropeanRevenue != 5_000_000 {
		t.Fatalf("ledger=%d want exactly the fresh intake", club.Finances.EuropeanRevenue)
	}
}
