package tournament

import (
	"os"
	"testing"

	"football_sim/pkg/models"
	"football_sim/pkg/transfers"
)

// TestEuropeanWorldFiveSeasonSoak runs a 5-season verification ensuring long-term
// competition completion, financial bounds, contract health, and roster stability.
func TestEuropeanWorldFiveSeasonSoak(t *testing.T) {
	runEuropeanWorldSoak(t, 5)
}

// TestEuropeanWorldTenSeasonSoak is opt-in because it resolves the complete
// 96-club calendar ten times. Run with FOOTBALL_SIM_LONG_SOAK=1 during release
// verification; the ordinary suite already covers one-season world behavior.
func TestEuropeanWorldTenSeasonSoak(t *testing.T) {
	if os.Getenv("FOOTBALL_SIM_LONG_SOAK") != "1" {
		t.Skip("set FOOTBALL_SIM_LONG_SOAK=1 for the full release soak")
	}
	runEuropeanWorldSoak(t, 10)
}

func runEuropeanWorldSoak(t *testing.T, targetSeasons int) {
	tm, _, te := loadEuropeanWorldForTest(t)
	startingPlayers := 0
	for _, club := range tm.ClubsList {
		startingPlayers += len(club.Squad)
	}

	for season := 1; season <= targetSeasons; season++ {
		batch := tm.SimulateBatchWeeks(tm.MaxMatchweeks)
		if batch.Status != "success" || !batch.SeasonFinished || tm.SeasonPhase != "transfer_window" {
			t.Fatalf("season %d did not complete: batch=%+v phase=%s", season, batch, tm.SeasonPhase)
		}
		for _, id := range tm.World.CompetitionOrder {
			comp := tm.World.Competitions[id]
			if comp == nil || comp.ChampionID == "" || tm.Clubs[comp.ChampionID] == nil {
				t.Fatalf("season %d competition %s has no valid champion", season, id)
			}
		}
		assertLongSoakWorldHealth(t, tm, season)

		te.BeginOffSeasonWindow()
		for te.IsWindowOpen() {
			te.AdvanceOpenWindow()
		}
		if te.CurrentWeek != transfers.TransferWindowWeeks || te.ProcessedWeeks != transfers.TransferWindowWeeks {
			t.Fatalf("season %d summer window ended at week=%d processed=%d", season, te.CurrentWeek, te.ProcessedWeeks)
		}
		if transition := tm.FinalizeSeasonTransition(); transition["status"] != "success" {
			t.Fatalf("season %d transition failed: %v", season, transition)
		}
		if te.WindowType != transfers.WindowClosed || te.CurrentWeek != 0 || te.ProcessedWeeks != 0 {
			t.Fatalf("season %d new campaign has ambiguous market state: type=%s week=%d processed=%d", season, te.WindowType, te.CurrentWeek, te.ProcessedWeeks)
		}
		assertLongSoakWorldHealth(t, tm, season)
	}

	endingPlayers := 0
	for _, club := range tm.ClubsList {
		endingPlayers += len(club.Squad)
	}
	// Transfers redistribute players, while retirement and the sustainable
	// academy target keep the universe from converging on 96 maxed-out squads.
	if ceiling := len(tm.ClubsList) * (models.AcademyIntakeSquadTarget + 2); endingPlayers > ceiling {
		t.Fatalf("%d-season player population saturated: players=%d ceiling=%d", targetSeasons, endingPlayers, ceiling)
	}
	t.Logf("%d-season 96-club soak complete: players=%d→%d, transfers=%d, retired=%d", targetSeasons, startingPlayers, endingPlayers, len(te.AllTimeTransfers), len(tm.RetiredPlayerIDs))
}

func assertLongSoakWorldHealth(t *testing.T, tm *TournamentManager, season int) {
	t.Helper()
	if err := tm.ValidateWorldState(); err != nil {
		t.Fatalf("season %d world invalid: %v", season, err)
	}
	if len(tm.ClubsList) != 96 || len(tm.Managers) != 96 {
		t.Fatalf("season %d universe size clubs=%d managers=%d", season, len(tm.ClubsList), len(tm.Managers))
	}
	seen := make(map[string]string)
	for _, club := range tm.ClubsList {
		if club == nil {
			t.Fatalf("season %d contains nil club", season)
		}
		if club.Finances.Balance < 0 || club.Finances.TransferBudget < 0 || club.Finances.TransferBudget > club.Finances.Balance {
			t.Fatalf("season %d invalid finances for %s: %+v", season, club.ClubID, club.Finances)
		}
		if len(club.Squad) < 11 || len(club.Squad) > models.MaxSeniorSquadSize {
			t.Fatalf("season %d implausible squad size for %s: %d", season, club.ClubID, len(club.Squad))
		}
		for _, player := range club.Squad {
			if player == nil || player.PlayerID == "" {
				t.Fatalf("season %d %s contains nil/id-less player", season, club.ClubID)
			}
			if owner, exists := seen[player.PlayerID]; exists {
				t.Fatalf("season %d duplicate player %s at %s and %s", season, player.PlayerID, owner, club.ClubID)
			}
			seen[player.PlayerID] = club.ClubID
			if player.ClubID != club.ClubID || player.ContractYears <= 0 {
				t.Fatalf("season %d ownership/contract mismatch for %s: roster=%s player=%s years=%d", season, player.PlayerID, club.ClubID, player.ClubID, player.ContractYears)
			}
			if player.Morale < 0 || player.Morale > 100 || player.Fitness < 0 || player.Fitness > 100 || player.Sharpness < 0 || player.Sharpness > 100 {
				t.Fatalf("season %d dynamics out of range for %s", season, player.PlayerID)
			}
			if player.MarketValueEUR < models.MinPlayerValueEUR || player.MarketValueEUR > models.MaxPlayerValueEUR {
				t.Fatalf("season %d valuation out of range for %s: %d", season, player.PlayerID, player.MarketValueEUR)
			}
		}
	}
}
