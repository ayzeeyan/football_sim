package tournament

import (
	"testing"

	"football_sim/pkg/growth"
	"football_sim/pkg/models"
)

func TestEffectiveAppearancesFallsBackWithoutMinuteTracking(t *testing.T) {
	p := &models.Player{PlayerID: "X", Appearances: 30}
	if got := effectiveAppearances(p); got != 30 {
		t.Fatalf("fallback got %d want 30", got)
	}
	if got := effectiveAppearances(nil); got != 0 {
		t.Fatalf("nil got %d want 0", got)
	}
	zero := &models.Player{PlayerID: "Z", Appearances: 0, CompetitionStats: map[string]*models.CompetitionSeasonStats{
		"premier-league": {CompetitionID: "premier-league", Appearances: 0, Minutes: 0},
	}}
	if got := effectiveAppearances(zero); got != 0 {
		t.Fatalf("zero apps got %d want 0", got)
	}
}

func TestEffectiveAppearancesWeightsMinutesAndNeverInflates(t *testing.T) {
	fullTimer := &models.Player{PlayerID: "F", Appearances: 30, CompetitionStats: map[string]*models.CompetitionSeasonStats{
		"premier-league": {CompetitionID: "premier-league", Appearances: 30, Starts: 30, Minutes: 2700},
	}}
	if got := effectiveAppearances(fullTimer); got != 30 {
		t.Fatalf("full timer got %d want 30", got)
	}
	cameo := &models.Player{PlayerID: "C", Appearances: 30, CompetitionStats: map[string]*models.CompetitionSeasonStats{
		"premier-league": {CompetitionID: "premier-league", Appearances: 30, Starts: 0, Minutes: 600},
	}}
	if got := effectiveAppearances(cameo); got != 7 {
		t.Fatalf("cameo got %d want 7", got)
	}
	// Extra-time padding across competitions never manufactures appearances.
	padded := &models.Player{PlayerID: "P", Appearances: 5, CompetitionStats: map[string]*models.CompetitionSeasonStats{
		"premier-league": {CompetitionID: "premier-league", Appearances: 3, Minutes: 320},
		"fa-cup":         {CompetitionID: "fa-cup", Appearances: 2, Minutes: 250},
	}}
	if got := effectiveAppearances(padded); got != 5 {
		t.Fatalf("padded got %d want 5 (capped at appearances)", got)
	}
}

// Playing time must move development through the same capped engine: a
// 30-game starter grows more than a 30-cameo bench player of identical age
// and rating, and neither escapes the generic youth curve.
func TestPlayingTimeAffectsSeasonGrowthWithinCap(t *testing.T) {
	mkClub := func(id string, p *models.Player) *models.Club {
		return &models.Club{ClubID: id, ClubName: id, ShortName: id, League: "Premier League", Squad: []*models.Player{p}}
	}
	starter := &models.Player{
		PlayerID: "ST", FullName: "Starter", OVR: 70, Age: 18, Category: "FWD",
		ClubID: "C1", Appearances: 30,
		CompetitionStats: map[string]*models.CompetitionSeasonStats{
			"premier-league": {CompetitionID: "premier-league", Appearances: 30, Starts: 30, Minutes: 2700},
		},
	}
	bench := &models.Player{
		PlayerID: "BE", FullName: "Bench", OVR: 70, Age: 18, Category: "FWD",
		ClubID: "C2", Appearances: 30,
		CompetitionStats: map[string]*models.CompetitionSeasonStats{
			"premier-league": {CompetitionID: "premier-league", Appearances: 30, Starts: 0, Minutes: 600},
		},
	}
	tm := &TournamentManager{
		ClubsList:    []*models.Club{mkClub("C1", starter), mkClub("C2", bench)},
		GrowthEngine: growth.NewGrowthEngine(7),
	}
	tm.applySeasonalChangesUnlocked()
	if starter.OVR != 73 {
		t.Fatalf("starter OVR=%d want 73", starter.OVR)
	}
	if bench.OVR != 71 {
		t.Fatalf("cameo OVR=%d want 71", bench.OVR)
	}
}
