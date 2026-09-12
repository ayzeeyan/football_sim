package tournament

import (
	"testing"

	"football_sim/pkg/models"
	"football_sim/pkg/transfers"
)

func TestChunk1TenSeasonIntegrationSoak(t *testing.T) {
	const rootSeed int64 = 77120441
	streams := NewSubsystemRNG(rootSeed)
	tm := deterministicUniverse(t, rootSeed)
	te := transfers.NewTransferEngine(tm.ClubsList, tm.Managers, streams.SeedFor("transfers"))
	tm.TransferEngine = te

	if len(tm.ClubsList) != 12 {
		t.Fatalf("clubs=%d want 12", len(tm.ClubsList))
	}

	startRep := make(map[string]int, len(tm.ClubsList))
	for _, club := range tm.ClubsList {
		startRep[club.ClubID] = club.Identity.Reputation
	}

	minGrowth, maxGrowth := 999, -999
	totalTransfers := 0
	preventedDuplicateChecks := 0
	minBudget, maxBudget := int64(^uint64(0)>>1), int64(0)

	for season := 1; season <= 10; season++ {
		expectedFixtures := len(tm.ClubsList) * LeagueRounds / 2
		if len(tm.Fixtures) != expectedFixtures {
			t.Fatalf("season %d league fixtures=%d want %d", season, len(tm.Fixtures), expectedFixtures)
		}

		leagueApps := make(map[string]int, len(tm.ClubsList))
		for _, f := range tm.Fixtures {
			if f.HomeID == "" || f.AwayID == "" || f.HomeID == f.AwayID {
				t.Fatalf("season %d invalid league fixture: %+v", season, f)
			}
			leagueApps[f.HomeID]++
			leagueApps[f.AwayID]++
		}
		for _, club := range tm.ClubsList {
			if got := leagueApps[club.ClubID]; got != LeagueRounds {
				t.Fatalf("season %d %s league schedule=%d want %d", season, club.ClubID, got, LeagueRounds)
			}
		}

		seasonStartOVR := map[string]int{}
		for _, club := range tm.ClubsList {
			for _, p := range club.Squad {
				if p != nil && models.IsCanonicalWonderkidID(p.PlayerID) {
					seasonStartOVR[p.PlayerID] = p.OVR
				}
			}
		}

		batch := tm.SimulateBatchWeeks(LeagueRounds)
		if batch.Status != "success" || !batch.SeasonFinished {
			t.Fatalf("season %d simulation failed: %+v", season, batch)
		}
		if tm.SeasonPhase != "transfer_window" {
			t.Fatalf("season %d ended in phase %q, want transfer_window", season, tm.SeasonPhase)
		}
		for _, club := range tm.ClubsList {
			if club.Played != LeagueRounds {
				t.Fatalf("season %d %s played=%d want %d", season, club.ClubID, club.Played, LeagueRounds)
			}
		}
		awards := tm.GetSeasonAwards()
		if awards["player_of_the_season"] == nil {
			t.Fatalf("season %d produced no player of the season", season)
		}
		ballon, ok := awards["ballon_dor"].([]map[string]interface{})
		if !ok || len(ballon) == 0 {
			t.Fatalf("season %d produced no Ballon d'Or ranking", season)
		}
		if err := tm.ValidateWorldState(); err != nil {
			t.Fatalf("season %d pre-window world invalid: %v", season, err)
		}

		for _, club := range tm.ClubsList {
			for _, p := range club.Squad {
				if p == nil || !models.IsCanonicalWonderkidID(p.PlayerID) {
					continue
				}
				if !models.IsDesignatedWonderkidClubID(p.ClubID) {
					t.Fatalf("season %d canonical wonderkid %s escaped to %s", season, p.PlayerID, p.ClubID)
				}
				if start, ok := seasonStartOVR[p.PlayerID]; ok {
					growth := p.OVR - start
					if growth < minGrowth {
						minGrowth = growth
					}
					if growth > maxGrowth {
						maxGrowth = growth
					}
					if growth > 5 {
						t.Fatalf("season %d %s grew %+d OVR; hard annual bound is +5", season, p.PlayerID, growth)
					}
				}
			}
		}

		te.BeginOffSeasonWindow()
		if !te.IsWindowOpen() || te.CurrentWeek != 1 {
			t.Fatalf("season %d transfer window did not begin at week 1: open=%v week=%d", season, te.IsWindowOpen(), te.CurrentWeek)
		}

		for processed := 1; processed <= transfers.TransferWindowWeeks; processed++ {
			if te.CurrentWeek != processed {
				t.Fatalf("season %d transfer week before step %d = %d", season, processed, te.CurrentWeek)
			}
			if early := tm.FinalizeSeasonTransition(); early["status"] == "success" {
				t.Fatalf("season %d transitioned before transfer week %d was processed", season, processed)
			}
			te.AdvanceOpenWindow()
			for _, club := range tm.ClubsList {
				if club.Finances.TransferBudget < 0 || club.Finances.Balance < 0 {
					t.Fatalf("season %d week %d %s has negative finances: %+v", season, processed, club.ClubID, club.Finances)
				}
				if club.Finances.TransferBudget < minBudget {
					minBudget = club.Finances.TransferBudget
				}
				if club.Finances.TransferBudget > maxBudget {
					maxBudget = club.Finances.TransferBudget
				}
			}
		}
		if te.CurrentWeek != transfers.TransferWindowWeeks+1 || te.IsWindowOpen() {
			t.Fatalf("season %d transfer window completion state wrong: week=%d open=%v", season, te.CurrentWeek, te.IsWindowOpen())
		}

		seenThisWindow := map[string]bool{}
		for _, done := range te.CompletedTransfers {
			if seenThisWindow[done.PlayerID] {
				t.Fatalf("season %d player %s transferred twice in one window", season, done.PlayerID)
			}
			seenThisWindow[done.PlayerID] = true
			if !te.TransferredThisWindow[done.PlayerID] {
				t.Fatalf("season %d completed transfer %s missing window marker", season, done.PlayerID)
			}
			playerFound := false
			for _, p := range te.Clubs[done.BuyerID].Squad {
				if p != nil && p.PlayerID == done.PlayerID {
					playerFound = true
					if p.ClubID != done.BuyerID {
						t.Fatalf("season %d transfer %s roster/current club disagree: %s vs %s", season, p.PlayerID, done.BuyerID, p.ClubID)
					}
					break
				}
			}
			if !playerFound {
				t.Fatalf("season %d transferred player %s missing from buyer %s", season, done.PlayerID, done.BuyerID)
			}
		}
		totalTransfers += len(te.CompletedTransfers)

		// Commit validation must still reject a second move for a player already
		// marked in this window. TriggerSpecificBid is the exported candidate/
		// eligibility boundary and must not create a second negotiation.
		for playerID := range seenThisWindow {
			var currentClub *models.Club
			for _, club := range tm.ClubsList {
				for _, p := range club.Squad {
					if p != nil && p.PlayerID == playerID {
						currentClub = club
						break
					}
				}
				if currentClub != nil {
					break
				}
			}
			if currentClub == nil {
				continue
			}
			for _, buyer := range tm.ClubsList {
				if buyer.ClubID == currentClub.ClubID {
					continue
				}
				if te.TriggerSpecificBid(buyer.ClubID, currentClub.ClubID, playerID) != nil {
					t.Fatalf("season %d created second-window negotiation for already transferred %s", season, playerID)
				}
				preventedDuplicateChecks++
				break
			}
			break
		}

		transition := tm.FinalizeSeasonTransition()
		if transition["status"] != "success" {
			t.Fatalf("season %d failed final transition after 12 weeks: %#v", season, transition)
		}
		if tm.SeasonPhase != "season" || tm.CurrentMatchweek != 1 {
			t.Fatalf("season %d next season state invalid: phase=%s mw=%d", season, tm.SeasonPhase, tm.CurrentMatchweek)
		}
		if err := tm.ValidateWorldState(); err != nil {
			t.Fatalf("season %d post-transition world invalid: %v", season, err)
		}
		for _, done := range te.AllTimeTransfers {
			if done.PlayerID == "" {
				continue
			}
			if tm.IsPlayerRetired(done.PlayerID) {
				continue
			}
			foundAtBuyer := false
			for _, p := range tm.Clubs[done.BuyerID].Squad {
				if p != nil && p.PlayerID == done.PlayerID {
					foundAtBuyer = p.ClubID == done.BuyerID
					break
				}
			}
			// A later-season transfer can legitimately move a player again. Only
			// demand the historical buyer when it is still the player's latest deal.
			latest := done
			for _, later := range te.AllTimeTransfers {
				if later.PlayerID == done.PlayerID {
					latest = later
				}
			}
			if latest.BuyerID == done.BuyerID && !foundAtBuyer {
				t.Fatalf("season %d permanent transfer %s did not persist at latest buyer %s", season, done.PlayerID, done.BuyerID)
			}
		}
	}

	reputationChanged := false
	lowestRep, highestRep := 101, -1
	for _, club := range tm.ClubsList {
		if club.Identity.Reputation != startRep[club.ClubID] {
			reputationChanged = true
		}
		if club.Identity.Reputation < lowestRep {
			lowestRep = club.Identity.Reputation
		}
		if club.Identity.Reputation > highestRep {
			highestRep = club.Identity.Reputation
		}
	}
	if !reputationChanged {
		t.Fatal("reputation did not evolve across ten seasons")
	}
	if minBudget == int64(^uint64(0)>>1) {
		minBudget = 0
	}
	t.Logf("10-season observations: transfers=%d, prevented duplicate probes=%d, wonderkid annual OVR range=%+d..%+d, reputation range=%d..%d, observed warchests=%s..%s",
		totalTransfers, preventedDuplicateChecks, minGrowth, maxGrowth, lowestRep, highestRep,
		models.FormatCurrency(minBudget), models.FormatCurrency(maxBudget))
}
