package tournament

import "testing"

func TestFinalizeSeasonTransitionRejectsRepeatedFinalizationWithoutMutation(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	completeLeagueForReputationTest(tm)

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

func TestFinalizeSeasonTransitionLeavesIncompleteCupUntouched(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	completeLeagueForReputationTest(tm)
	home, away := tm.ClubsList[0], tm.ClubsList[1]
	tm.UCLFixtures = []Fixture{{
		FixtureID: "TRANSITION-UCL-FINAL", Competition: "ucl", Stage: "Final",
		HomeID: home.ClubID, AwayID: away.ClubID, Status: "scheduled",
	}}
	season := tm.SeasonName
	phase := tm.SeasonPhase
	matchweek := tm.CurrentMatchweek
	age := home.Squad[0].Age
	reputation := home.Identity.Reputation
	history := len(tm.SeasonHistory)

	result := tm.FinalizeSeasonTransition()
	if result["status"] != "error" {
		t.Fatalf("incomplete cup should reject finalization: %v", result)
	}
	if tm.SeasonName != season || tm.SeasonPhase != phase || tm.CurrentMatchweek != matchweek {
		t.Fatalf("rejected transition changed calendar: season=%q phase=%q matchweek=%d", tm.SeasonName, tm.SeasonPhase, tm.CurrentMatchweek)
	}
	if home.Squad[0].Age != age || home.Identity.Reputation != reputation || len(tm.SeasonHistory) != history {
		t.Fatalf("rejected transition changed campaign state: age=%d/%d reputation=%d/%d history=%d/%d", home.Squad[0].Age, age, home.Identity.Reputation, reputation, len(tm.SeasonHistory), history)
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

func TestAdoptLongSeasonSequentialPairingAndInboxRemapping(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	home, away := tm.ClubsList[0], tm.ClubsList[1]
	hg, ag := 3, 1
	legacyFixture := Fixture{
		FixtureID:   "LEGACY_UNMAPPED_1",
		Competition: "league",
		Matchweek:   1,
		HomeID:      home.ClubID,
		AwayID:      away.ClubID,
		Status:      "finished",
		HomeGoals:   &hg,
		AwayGoals:   &ag,
	}
	tm.Fixtures = []Fixture{legacyFixture}
	tm.MaxMatchweeks = 1
	tm.Inbox = []InboxItem{
		{ID: "inbox-1", FixtureID: "LEGACY_UNMAPPED_1", Headline: "Big Win"},
	}

	if !tm.AdoptLongSeason() {
		t.Fatal("AdoptLongSeason should report changed")
	}

	if len(tm.Fixtures) != 44*6 || tm.MaxMatchweeks != 44 {
		t.Fatalf("expected 264 fixtures and 44 matchweeks, got %d / %d", len(tm.Fixtures), tm.MaxMatchweeks)
	}

	var remapped *Fixture
	for i := range tm.Fixtures {
		f := &tm.Fixtures[i]
		if f.HomeID == home.ClubID && f.AwayID == away.ClubID && f.Status == "finished" {
			remapped = f
			break
		}
	}
	if remapped == nil {
		t.Fatal("expected finished fixture to be remapped to a canonical fixture")
	}
	if remapped.HomeGoals == nil || *remapped.HomeGoals != 3 || remapped.AwayGoals == nil || *remapped.AwayGoals != 1 {
		t.Fatalf("remapped fixture lost results: %#v", remapped)
	}

	if tm.Inbox[0].FixtureID != remapped.FixtureID {
		t.Fatalf("inbox item not remapped: got %q, want %q", tm.Inbox[0].FixtureID, remapped.FixtureID)
	}
}
