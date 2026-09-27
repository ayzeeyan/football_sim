package tournament

import (
	"encoding/json"
	"fmt"
	"football_sim/pkg/models"
	"testing"
)

func findClubByShort(t *testing.T, tm *TournamentManager, short string) *models.Club {
	t.Helper()
	for _, c := range tm.ClubsList {
		if c.ShortName == short {
			return c
		}
	}
	t.Fatalf("club %s not found", short)
	return nil
}

// TestWeeklyAchievementsTrackUnbeatenRunsAndYoungestScorer runs one real
// matchweek and verifies the ledger bookkeeping: unbeaten runs move with the
// results, the youngest-scorer record is populated from report events, and
// re-running the same week never duplicates ledger entries.
func TestWeeklyAchievementsTrackUnbeatenRunsAndYoungestScorer(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	if out := tm.SimulateRemaining(); out["status"] != "success" {
		t.Fatalf("slate simulation failed: %v", out)
	}

	tm.mu.RLock()
	runs := map[string]int{}
	for k, v := range tm.ClubUnbeatenRuns {
		runs[k] = v
	}
	ledger := len(tm.Achievements)
	tm.mu.RUnlock()

	if len(runs) == 0 {
		t.Fatal("unbeaten runs must be populated after a matchweek")
	}
	total := 0
	for _, v := range runs {
		if v < 0 {
			t.Fatal("unbeaten runs must never be negative")
		}
		total += v
	}
	// Every played league fixture increments at least one run (the winner or
	// both on a draw), so the sum is at least the number of league fixtures.
	if total < 10 {
		t.Fatalf("unbeaten run total suspiciously low: %d", total)
	}

	// The youngest-scorer record must exist and come from a real report event.
	if tm.GetYoungestScorerRecord() == nil {
		t.Fatal("youngest scorer record must be set after a scored matchweek")
	}

	// Re-running the same week's evaluation must not duplicate entries.
	tm.mu.Lock()
	tm.evaluateWeeklyAchievementsUnlocked(1)
	dup := len(tm.Achievements)
	tm.mu.Unlock()
	if dup != ledger {
		t.Fatalf("re-evaluation duplicated ledger entries: %d -> %d", ledger, dup)
	}
}

// TestThirtyGoalSeasonAchievementFiresOnce covers the player milestone and
// its per-season dedupe.
func TestThirtyGoalSeasonAchievementFiresOnce(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	club := tm.ClubsList[0]
	var scorer *models.Player
	for _, p := range club.Squad {
		if p != nil {
			scorer = p
			break
		}
	}
	if scorer == nil {
		t.Fatal("no player found")
	}
	tm.mu.Lock()
	scorer.Goals = 30
	tm.evaluateWeeklyAchievementsUnlocked(5)
	first := len(tm.Achievements)
	tm.evaluateWeeklyAchievementsUnlocked(6)
	second := len(tm.Achievements)
	tm.mu.Unlock()

	if first != 1 {
		t.Fatalf("30-goal season must fire exactly once, ledger=%d", first)
	}
	if second != first {
		t.Fatal("30-goal season must not fire twice in one season")
	}
	if got := tm.GetAchievements(); len(got) != 1 || got[0].ID != "goals_30_season" || got[0].SubjectID != scorer.PlayerID {
		t.Fatalf("unexpected ledger: %+v", got)
	}
}

// TestSackingAchievementOnlyBeforeSeasonEnd pins the mid-season condition.
func TestSackingAchievementOnlyBeforeSeasonEnd(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	club := tm.ClubsList[0]
	tm.mu.Lock()
	tm.unlockSackingAchievementUnlocked(club, "Test Manager", 10)
	mid := len(tm.Achievements)
	tm.unlockSackingAchievementUnlocked(club, "Another Manager", tm.MaxMatchweeks)
	end := len(tm.Achievements)
	tm.mu.Unlock()

	if mid != 1 {
		t.Fatalf("mid-season sacking must unlock, ledger=%d", mid)
	}
	if end != mid {
		t.Fatal("final-week sacking must not unlock")
	}
}

// TestSeasonAchievementsTrebleAndWorstToChampion exercises the end-of-season
// checks with hand-seeded champions and history.
func TestSeasonAchievementsTrebleAndWorstToChampion(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	club := tm.ClubsList[0]

	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Treble: league + domestic cup + Champions League for one club.
	for _, id := range []string{"premier-league", "fa-cup", "champions-league"} {
		comp := tm.worldCompetitionUnlocked(id)
		if comp == nil {
			t.Fatalf("competition %s missing in world", id)
		}
		comp.ChampionID = club.ClubID
	}
	// Worst-to-champion: 18th last season (Ligue 1 has 18 clubs), champion now.
	tm.ClubSeasonHistory[club.ClubID] = append(tm.ClubSeasonHistory[club.ClubID],
		map[string]interface{}{"season_name": "2025-26", "position": 18},
		map[string]interface{}{"season_name": tm.SeasonName, "position": 1},
	)

	tm.evaluateSeasonAchievementsUnlocked()
	ids := map[string]bool{}
	for _, a := range tm.Achievements {
		ids[a.ID] = true
	}
	if !ids["treble"] {
		t.Fatalf("treble must unlock for %s, ledger=%+v", club.ClubID, tm.Achievements)
	}
	if !ids["worst_to_champion"] {
		t.Fatalf("worst-to-champion must unlock, ledger=%+v", tm.Achievements)
	}

	// Re-running the same season evaluation must not duplicate.
	before := len(tm.Achievements)
	tm.evaluateSeasonAchievementsUnlocked()
	if len(tm.Achievements) != before {
		t.Fatal("season achievements must dedupe by unlock key")
	}
}

// TestAchievementLedgerDeterministic runs two identical worlds for a full
// season and asserts byte-identical ledgers.
func TestAchievementLedgerDeterministic(t *testing.T) {
	runSeason := func() []byte {
		tm, _, _ := loadEuropeanWorldForTest(t)
		for tm.CurrentMatchweek <= tm.MaxMatchweeks {
			if out := tm.SimulateRemaining(); out["status"] != "success" {
				t.Fatalf("slate simulation failed: %v", out)
			}
		}
		ledger := tm.GetAchievements()
		out, err := json.Marshal(ledger)
		if err != nil {
			t.Fatalf("marshal ledger: %v", err)
		}
		return out
	}
	first := runSeason()
	second := runSeason()
	if string(first) != string(second) {
		fmt.Printf("first=%s\nsecond=%s\n", first, second)
		t.Fatal("achievement ledger diverged across identical seasons")
	}
}
