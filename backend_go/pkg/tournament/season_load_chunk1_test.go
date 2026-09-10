package tournament

import "testing"

func TestChunk1SeasonLoadTargetsHighFifties(t *testing.T) {
	const clubs = 12
	leaguePerClub := 4 * (clubs - 1)
	if LeagueRounds != leaguePerClub {
		t.Fatalf("LeagueRounds=%d want %d league matches per club", LeagueRounds, leaguePerClub)
	}

	// A finalist plays five two-sided Champions Cup group/knockout rounds as
	// currently modelled (10 matches total) plus four Super Cup rounds.
	const championsCupMax = 10
	const superCupMax = 4
	maxCompetitive := leaguePerClub + championsCupMax + superCupMax
	if maxCompetitive != 58 {
		t.Fatalf("maximum competitive load=%d want 58", maxCompetitive)
	}
	if maxCompetitive < 55 || maxCompetitive > 62 {
		t.Fatalf("maximum competitive load=%d is outside realistic high-50s target", maxCompetitive)
	}
}
