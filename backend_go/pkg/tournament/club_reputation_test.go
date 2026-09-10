package tournament

import (
	"testing"

	"football_sim/pkg/models"
)

func TestReputationChampionImprovesButMiracleIsBounded(t *testing.T) {
	in := ClubReputationSeasonInput{CurrentReputation: 45, HistoricalPrestige: 40, LeagueFinish: 1, ClubCount: 12, LeagueChampion: true}
	delta := CalculateClubReputationChange(in)
	if delta <= 0 {
		t.Fatalf("champion delta=%d want positive", delta)
	}
	if delta > maxAnnualReputationGain {
		t.Fatalf("miracle title delta=%d exceeds cap %d", delta, maxAnnualReputationGain)
	}
}

func TestReputationPoorSeasonDeclines(t *testing.T) {
	delta := CalculateClubReputationChange(ClubReputationSeasonInput{CurrentReputation: 80, HistoricalPrestige: 70, LeagueFinish: 12, ClubCount: 12})
	if delta >= 0 {
		t.Fatalf("poor-season delta=%d want negative", delta)
	}
}

func TestPrestigiousClubDeclinesMoreSlowly(t *testing.T) {
	elite := CalculateClubReputationChange(ClubReputationSeasonInput{CurrentReputation: 95, HistoricalPrestige: 98, LeagueFinish: 12, ClubCount: 12})
	ordinary := CalculateClubReputationChange(ClubReputationSeasonInput{CurrentReputation: 95, HistoricalPrestige: 60, LeagueFinish: 12, ClubCount: 12})
	if elite < -elitePrestigeAnnualLoss {
		t.Fatalf("elite decline=%d exceeds inertia cap", elite)
	}
	if elite <= ordinary {
		t.Fatalf("elite decline %d should be gentler than ordinary %d", elite, ordinary)
	}
}

func TestRepeatedSuccessCompoundsAndRemainsBounded(t *testing.T) {
	rep := 55
	for season := 0; season < 5; season++ {
		delta := CalculateClubReputationChange(ClubReputationSeasonInput{CurrentReputation: rep, HistoricalPrestige: 50, LeagueFinish: 1, ClubCount: 12, LeagueChampion: true})
		if delta < 0 || delta > maxAnnualReputationGain {
			t.Fatalf("season %d delta=%d", season+1, delta)
		}
		rep += delta
		if rep < 0 || rep > 100 {
			t.Fatalf("season %d rep=%d outside 0-100", season+1, rep)
		}
	}
	if rep <= 55 {
		t.Fatalf("repeated success did not compound: %d", rep)
	}
}

func TestReputationNeverExceedsBounds(t *testing.T) {
	if delta := CalculateClubReputationChange(ClubReputationSeasonInput{CurrentReputation: 99, HistoricalPrestige: 100, LeagueFinish: 1, ClubCount: 12, LeagueChampion: true, ContinentalChampion: true, SuperCupChampion: true}); 99+delta > 100 {
		t.Fatalf("upper bound breached: delta=%d", delta)
	}
	if delta := CalculateClubReputationChange(ClubReputationSeasonInput{CurrentReputation: 1, HistoricalPrestige: 0, LeagueFinish: 12, ClubCount: 12}); 1+delta < 0 {
		t.Fatalf("lower bound breached: delta=%d", delta)
	}
}

func TestSuperCupPenaltyWinnerUsesAwaySide(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	completeLeagueForReputationTest(tm)
	home, away := tm.ClubsList[0], tm.ClubsList[1]
	hg, ag := 1, 1
	tm.SuperCupFixtures = []Fixture{{
		FixtureID: "REP-SC-FINAL", Competition: "super-cup", Stage: "Final",
		HomeID: home.ClubID, AwayID: away.ClubID, Status: "finished",
		HomeGoals: &hg, AwayGoals: &ag, Penalties: []int{3, 5},
	}}
	tm.SuperCupChampionID = ""
	for _, club := range tm.ClubsList {
		club.Identity.Reputation = 50
		club.Identity.HistoricalPrestige = 50
	}
	standings := tm.standingsUnlocked()
	positions := make(map[string]int, len(standings))
	for i, club := range standings {
		positions[club.ClubID] = i + 1
	}
	if !tm.ApplyCompletedSeasonReputation() {
		t.Fatal("completed Super Cup penalty final should allow reputation application")
	}
	// Recompute from the known baseline so the penalty winner assertion checks
	// the full annual formula, including the one-point Super Cup bonus.
	homeExpected := 50 + CalculateClubReputationChange(ClubReputationSeasonInput{
		CurrentReputation: 50, HistoricalPrestige: 50, LeagueFinish: positions[home.ClubID],
		ClubCount: len(standings), LeagueChampion: positions[home.ClubID] == 1,
	})
	awayExpected := 50 + CalculateClubReputationChange(ClubReputationSeasonInput{
		CurrentReputation: 50, HistoricalPrestige: 50, LeagueFinish: positions[away.ClubID],
		ClubCount: len(standings), LeagueChampion: positions[away.ClubID] == 1,
		SuperCupChampion: true,
	})
	if home.Identity.Reputation != homeExpected || away.Identity.Reputation != awayExpected {
		t.Fatalf("penalty winner reputation home=%d want %d away=%d want %d", home.Identity.Reputation, homeExpected, away.Identity.Reputation, awayExpected)
	}
}

func completeLeagueForReputationTest(tm *TournamentManager) {
	for i := range tm.Fixtures {
		hg, ag := 0, 0
		if i == 0 {
			hg = 3
		}
		tm.Fixtures[i].Status = "finished"
		tm.Fixtures[i].HomeGoals = &hg
		tm.Fixtures[i].AwayGoals = &ag
	}
	for i, club := range tm.ClubsList {
		club.Played = len(tm.Fixtures) / len(tm.ClubsList) * 2
		club.Points = 0
		club.GoalDifference = 0
		if i == 0 {
			club.Points = 100
		}
	}
	tm.CurrentMatchweek = tm.MaxMatchweeks + 1
	tm.SeasonPhase = "transfer_window"
	tm.UCLFixtures = nil
	tm.SuperCupFixtures = nil
}

func TestCompletedSeasonReputationWaitsForCupInputsAndIsIdempotent(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	completeLeagueForReputationTest(tm)
	champion := tm.ClubsList[0]
	before := champion.Identity.Reputation
	season := tm.SeasonName
	tm.UCLFixtures = []Fixture{{
		FixtureID: "REP-UCL-FINAL", Competition: "ucl", Stage: "Final",
		HomeID: champion.ClubID, AwayID: tm.ClubsList[1].ClubID, Status: "scheduled",
	}}

	if tm.ApplyCompletedSeasonReputation() {
		t.Fatal("reputation applied before the outstanding cup result")
	}
	if champion.Identity.Reputation != before || tm.ReputationAppliedSeason != "" {
		t.Fatalf("incomplete cup changed reputation=%d marker=%q", champion.Identity.Reputation, tm.ReputationAppliedSeason)
	}

	hg, ag := 2, 1
	tm.UCLFixtures[0].Status = "finished"
	tm.UCLFixtures[0].HomeGoals = &hg
	tm.UCLFixtures[0].AwayGoals = &ag
	tm.UCLChampionID = champion.ClubID
	if !tm.ApplyCompletedSeasonReputation() {
		t.Fatal("completed season reputation was not applied")
	}
	if champion.Identity.Reputation == before || tm.ReputationAppliedSeason != season {
		t.Fatalf("completed season update missing: reputation=%d/%d marker=%q", champion.Identity.Reputation, before, tm.ReputationAppliedSeason)
	}
	after := champion.Identity.Reputation
	if tm.ApplyCompletedSeasonReputation() {
		t.Fatal("same season reputation applied twice")
	}
	if champion.Identity.Reputation != after {
		t.Fatalf("same season reputation changed on retry: %d -> %d", after, champion.Identity.Reputation)
	}
}

func TestCompletedSeasonReputationPreservesBoundsAndPrestigeInertia(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	completeLeagueForReputationTest(tm)
	tm.UCLFixtures = nil
	tm.SuperCupFixtures = nil
	for _, club := range tm.ClubsList {
		club.Identity.Reputation = 100
		club.Identity.HistoricalPrestige = 100
	}
	if !tm.ApplyCompletedSeasonReputation() {
		t.Fatal("completed season reputation was not applied")
	}
	for _, club := range tm.ClubsList {
		if club.Identity.Reputation < models.ClubRatingMin || club.Identity.Reputation > models.ClubRatingMax {
			t.Fatalf("club %s reputation=%d outside bounds", club.ClubID, club.Identity.Reputation)
		}
	}
}

func TestSeasonBoundaryAppliesReputationAfterCanonicalCupSlate(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	before := make(map[string]int, len(tm.ClubsList))
	for _, club := range tm.ClubsList {
		before[club.ClubID] = club.Identity.Reputation
	}
	season := tm.SeasonName
	result := tm.SimulateBatchWeeks(LeagueRounds)
	if result.Status != "success" || !result.SeasonFinished {
		t.Fatalf("canonical season did not finish: %+v", result)
	}
	if tm.ReputationAppliedSeason != season {
		t.Fatalf("boundary marker=%q want %q", tm.ReputationAppliedSeason, season)
	}
	changed := false
	for _, club := range tm.ClubsList {
		if club.Identity.Reputation != before[club.ClubID] {
			changed = true
			break
		}
	}
	if !changed {
		t.Fatal("canonical completed season did not update any reputation")
	}
}
