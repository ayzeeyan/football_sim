package tournament

import (
	"testing"

	"football_sim/pkg/models"
)

// This default-seed season audit protects the general CF-to-striker fit used
// by Jhed and any other center forward. It identifies the player across the
// world so future seeded wonderkid-home shuffles remain valid.
func TestWorldSeasonGivesCentreForwardWonderkidRegularMinutes(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	batch := tm.SimulateBatchWeeks(tm.MaxMatchweeks)
	if batch.Status != "success" {
		t.Fatalf("world simulation failed: %+v", batch)
	}

	var jhed *models.Player
	appearances, starts, minutes := 0, 0, 0
	topScorerGoals := 0
	wonderkidCopies := 0
	for _, club := range tm.ClubsList {
		for _, player := range club.Squad {
			if player.Goals > topScorerGoals {
				topScorerGoals = player.Goals
			}
			if player.PlayerID != "WK_Jhed_Anthony_Guinita" {
				continue
			}
			wonderkidCopies++
			jhed = player
			for _, stats := range player.CompetitionStats {
				appearances += stats.Appearances
				starts += stats.Starts
				minutes += stats.Minutes
			}
		}
	}
	if wonderkidCopies != 1 || jhed == nil {
		t.Fatalf("Jhed copies=%d; want exactly one in the shuffled club world", wonderkidCopies)
	}
	if appearances < 35 || starts < 25 || minutes < 2500 {
		t.Fatalf("Jhed season usage apps/starts/minutes=%d/%d/%d; want at least 35/25/2500", appearances, starts, minutes)
	}
	if jhed.Goals < 10 {
		t.Fatalf("Jhed scored %d goals in %d apps; want a modest 10-goal floor", jhed.Goals, appearances)
	}
	if topScorerGoals <= 40 {
		t.Fatalf("season top scorer reached only %d goals; want a plausible 40+ elite season", topScorerGoals)
	}
}
