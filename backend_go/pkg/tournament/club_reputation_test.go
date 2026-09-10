package tournament

import "testing"

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
