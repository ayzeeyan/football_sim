package tournament

import (
	"testing"

	"football_sim/pkg/models"
)

// buildRelegationTestWorld constructs a minimal world whose final tables are
// fully controlled, so the swap semantics can be asserted deterministically.
func buildRelegationTestWorld() *TournamentManager {
	tm := &TournamentManager{
		Clubs:     map[string]*models.Club{},
		ClubsList: []*models.Club{},
		World: &EuropeanWorld{
			Competitions: map[string]*Competition{}, CompetitionOrder: []string{},
		},
	}
	for _, def := range domesticLeagueDefinitions {
		comp := &Competition{ID: def.ID, Name: def.Name, Kind: CompetitionLeague, ParticipantIDs: []string{}}
		tm.World.Competitions[def.ID] = comp
		tm.World.CompetitionOrder = append(tm.World.CompetitionOrder, def.ID)
		for i := 0; i < 8; i++ {
			// Points strictly decrease so the table order is exactly the
			// insertion order: index 0 is the champion, index 7 is last.
			id := def.ID + "-" + string(rune('A'+i))
			club := &models.Club{
				ClubID: id, ClubName: id, ShortName: id, League: def.League,
				Points: 100 - i,
			}
			tm.Clubs[id] = club
			tm.ClubsList = append(tm.ClubsList, club)
			comp.ParticipantIDs = append(comp.ParticipantIDs, id)
		}
	}
	return tm
}

func TestPlanDomesticPromotionRelegationSwapsBoundaries(t *testing.T) {
	tm := buildRelegationTestWorld()
	moves := tm.planDomesticPromotionRelegationUnlocked()

	chain := domesticRelegationChain()
	// 4 boundaries x (3 down + 3 up) = 24 moves.
	if len(moves) != 24 {
		t.Fatalf("moves=%d want 24", len(moves))
	}
	byClub := map[string]RelegationMove{}
	for _, move := range moves {
		if _, dup := byClub[move.ClubID]; dup {
			t.Fatalf("club %s appears in two moves", move.ClubID)
		}
		byClub[move.ClubID] = move
	}
	for i := 0; i+1 < len(chain); i++ {
		stronger, weaker := chain[i], chain[i+1]
		// Bottom three of the stronger league drop into the weaker league.
		for _, suffix := range []string{"F", "G", "H"} {
			id := stronger.ID + "-" + suffix
			move, ok := byClub[id]
			if !ok {
				t.Fatalf("%s (bottom three of %s) was not relegated", id, stronger.Name)
			}
			if move.Direction != "relegated" || move.ToLeague != weaker.League {
				t.Fatalf("%s wrong move: %+v", id, move)
			}
		}
		// Top three of the weaker league rise into the stronger league.
		for _, suffix := range []string{"A", "B", "C"} {
			id := weaker.ID + "-" + suffix
			move, ok := byClub[id]
			if !ok {
				t.Fatalf("%s (top three of %s) was not promoted", id, weaker.Name)
			}
			if move.Direction != "promoted" || move.ToLeague != stronger.League {
				t.Fatalf("%s wrong move: %+v", id, move)
			}
		}
	}
	// The base league's bottom three never move: the pyramid has no lower tier.
	base := chain[len(chain)-1]
	for _, suffix := range []string{"F", "G", "H"} {
		id := base.ID + "-" + suffix
		if _, moved := byClub[id]; moved {
			t.Fatalf("base-league club %s must not be relegated out of the pyramid", id)
		}
	}
	// Mid-table clubs never move.
	if _, moved := byClub[chain[0].ID+"-D"]; moved {
		t.Fatal("mid-table club must not move")
	}
}

func TestPlanDomesticPromotionRelegationIsDeterministic(t *testing.T) {
	a := buildRelegationTestWorld().planDomesticPromotionRelegationUnlocked()
	b := buildRelegationTestWorld().planDomesticPromotionRelegationUnlocked()
	if len(a) != len(b) {
		t.Fatalf("plan length differs: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("move %d differs: %+v vs %+v", i, a[i], b[i])
		}
	}
}

func TestApplyPlannedRelegationRewritesLeagues(t *testing.T) {
	tm := buildRelegationTestWorld()
	moves := tm.planDomesticPromotionRelegationUnlocked()
	tm.applyPlannedRelegationUnlocked(moves)

	// League sizes are conserved.
	counts := map[string]int{}
	for _, club := range tm.ClubsList {
		counts[club.League]++
	}
	for _, def := range domesticLeagueDefinitions {
		if counts[def.League] != 8 {
			t.Fatalf("%s size=%d want 8", def.League, counts[def.League])
		}
	}
	// A relegated club now carries its new league.
	if club := tm.Clubs["premier-league-H"]; club.League != "La Liga" {
		t.Fatalf("relegated club league=%q want La Liga", club.League)
	}
	if club := tm.Clubs["ligue-1-A"]; club.League != "Serie A" {
		t.Fatalf("promoted club league=%q want Serie A", club.League)
	}
	// Planning again after application is idempotent-safe: the base league's
	// bottom three still never move.
	chain := domesticRelegationChain()
	base := chain[len(chain)-1]
	for _, move := range tm.planDomesticPromotionRelegationUnlocked() {
		if move.FromLeague == base.League && move.Direction == "relegated" {
			t.Fatalf("base league cannot relegate: %+v", move)
		}
	}
}
