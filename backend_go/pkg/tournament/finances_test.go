package tournament

import (
	"testing"

	"football_sim/pkg/models"
)

func TestLeagueFinishPrizePaysTheChampion(t *testing.T) {
	club := &models.Club{ClubID: "WIN", ClubName: "Winners", ShortName: "WIN", League: "Premier League", Finances: models.ClubFinances{Balance: 10_000_000, TransferBudget: 5_000_000}}
	tm := &TournamentManager{
		Clubs:     map[string]*models.Club{"WIN": club},
		ClubsList: []*models.Club{club},
		World: &EuropeanWorld{Competitions: map[string]*Competition{
			"premier-league": {ID: "premier-league", Name: "Premier League", Kind: CompetitionLeague, Prestige: 95, ParticipantIDs: []string{"WIN"}, ChampionID: "WIN"},
		}, CompetitionOrder: []string{"premier-league"}},
		SeasonName: "2026-27",
	}
	club.Points = 90
	tm.awardSeasonPrizeMoneyUnlocked()
	if club.Finances.Balance <= 10_000_000 {
		t.Fatalf("champion should receive prize money, balance=%d", club.Finances.Balance)
	}
}

// European prizes must land in the separate season ledger while moving the
// same euros into Balance: for an isolated Euro-only club, the ledger equals
// the Balance delta exactly, and the runner-up banks its own bonus.
func TestEuropeanPrizeMoneyTracksSeparateRevenue(t *testing.T) {
	a := &models.Club{ClubID: "A", ClubName: "Alpha", ShortName: "ALP", League: "Premier League", Finances: models.ClubFinances{Balance: 100_000_000, TransferBudget: 50_000_000}}
	b := &models.Club{ClubID: "B", ClubName: "Beta", ShortName: "BET", League: "Premier League", Finances: models.ClubFinances{Balance: 100_000_000, TransferBudget: 50_000_000}}
	tm := &TournamentManager{
		Clubs:     map[string]*models.Club{"A": a, "B": b},
		ClubsList: []*models.Club{a, b},
		World: &EuropeanWorld{Competitions: map[string]*Competition{
			"europa-league": {
				ID: "europa-league", Name: "UEFA Europa League", Kind: CompetitionEuropean,
				ParticipantIDs: []string{"A", "B"},
				Records: map[string]*models.CompetitionRecord{
					"A": {Won: 4, Drawn: 0, Points: 12},
					"B": {Won: 1, Drawn: 0, Points: 3},
				},
				Rounds: []KnockoutRound{
					{Stage: "Final", EntrantIDs: []string{"A", "B"}, FixtureIDs: []string{"F"}, TieIDs: []string{"TF"}, WinnerIDs: []string{"A"}},
				},
				ChampionID: "A",
			},
		}, CompetitionOrder: []string{"europa-league"}},
		SeasonName: "2026-27",
	}
	tm.awardEuropeanPrizeMoneyUnlocked("europa-league")
	// A: 6M participation + rank 1 (250k + 1*100k) + 2.5M final + 2.5M bonus.
	wantA := int64(6_000_000 + 350_000 + 2_500_000 + 2_500_000)
	if a.Finances.EuropeanRevenue != wantA {
		t.Fatalf("A European revenue=%d want %d", a.Finances.EuropeanRevenue, wantA)
	}
	if a.Finances.Balance-100_000_000 != wantA {
		t.Fatalf("A balance delta=%d want revenue %d", a.Finances.Balance-100_000_000, wantA)
	}
	// B: 6M participation + rank 2 (250k) + 2.5M finalist prize + 1.5M runners-up bonus.
	wantB := int64(6_000_000 + 250_000 + 2_500_000 + 1_500_000)
	if b.Finances.EuropeanRevenue != wantB {
		t.Fatalf("B European revenue=%d want %d", b.Finances.EuropeanRevenue, wantB)
	}
}
