package tournament

import "testing"

func TestFinalizeSeasonTransitionRejectsRepeatedFinalizationWithoutMutation(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	if result := tm.SimulateMatchweek(1); result["status"] != "success" {
		t.Fatalf("failed to prepare completed campaign state: %v", result)
	}
	tm.SeasonPhase = "transfer_window"

	firstPlayer := tm.ClubsList[0].Squad[0]
	ageBefore := firstPlayer.Age
	first := tm.FinalizeSeasonTransition()
	if first["status"] != "success" {
		t.Fatalf("first transition failed: %v", first)
	}
	if firstPlayer.Age != ageBefore+1 {
		t.Fatalf("first transition did not age player once: got %d want %d", firstPlayer.Age, ageBefore+1)
	}

	seasonAfterFirst := tm.SeasonName
	ageAfterFirst := firstPlayer.Age
	careerGoalsAfterFirst := firstPlayer.CareerGoals
	historyAfterFirst := len(tm.SeasonHistory)
	fixturesAfterFirst := len(tm.Fixtures)
	inboxAfterFirst := len(tm.Inbox)

	second := tm.FinalizeSeasonTransition()
	if second["status"] != "error" {
		t.Fatalf("repeated season transition should be explicitly rejected: %v", second)
	}
	if tm.SeasonName != seasonAfterFirst {
		t.Fatalf("repeated transition advanced season: %s -> %s", seasonAfterFirst, tm.SeasonName)
	}
	if firstPlayer.Age != ageAfterFirst || firstPlayer.CareerGoals != careerGoalsAfterFirst {
		t.Fatalf("repeated transition mutated player continuity: age=%d/%d goals=%d/%d", firstPlayer.Age, ageAfterFirst, firstPlayer.CareerGoals, careerGoalsAfterFirst)
	}
	if len(tm.SeasonHistory) != historyAfterFirst || len(tm.Fixtures) != fixturesAfterFirst || len(tm.Inbox) != inboxAfterFirst {
		t.Fatalf("repeated transition mutated world collections: history=%d/%d fixtures=%d/%d inbox=%d/%d", len(tm.SeasonHistory), historyAfterFirst, len(tm.Fixtures), fixturesAfterFirst, len(tm.Inbox), inboxAfterFirst)
	}
}

func TestFinalizeSeasonTransitionRejectsBeforeOffseasonPhase(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	season := tm.SeasonName
	age := tm.ClubsList[0].Squad[0].Age
	result := tm.FinalizeSeasonTransition()
	if result["status"] != "error" {
		t.Fatalf("expected transition outside transfer phase to be rejected: %v", result)
	}
	if tm.SeasonName != season || tm.ClubsList[0].Squad[0].Age != age {
		t.Fatal("rejected transition mutated universe")
	}
}
