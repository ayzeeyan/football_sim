package tournament

import (
	"reflect"
	"testing"

	"football_sim/pkg/models"
)

// buildRelegationTestWorld constructs a minimal world whose final tables are
// fully controlled, so the survival-stakes semantics can be asserted
// deterministically.
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
				Finances: models.ClubFinances{
					Balance: 100_000_000, TransferBudget: 50_000_000, WageCap: 60_000_000,
				},
			}
			club.Identity.Reputation = 60
			tm.Clubs[id] = club
			tm.ClubsList = append(tm.ClubsList, club)
			comp.ParticipantIDs = append(comp.ParticipantIDs, id)
		}
	}
	return tm
}

// The survival stakes hit exactly the bottom three of every league.
func TestPlanDomesticRelegationStakesHitsBottomThree(t *testing.T) {
	tm := buildRelegationTestWorld()
	stakes := tm.planDomesticRelegationStakesUnlocked()

	// 5 leagues x 3 staked clubs = 15 stakes.
	if len(stakes) != 15 {
		t.Fatalf("stakes=%d want 15", len(stakes))
	}
	byClub := map[string]RelegationStake{}
	for _, stake := range stakes {
		if _, dup := byClub[stake.ClubID]; dup {
			t.Fatalf("club %s appears in two stakes", stake.ClubID)
		}
		byClub[stake.ClubID] = stake
	}
	for _, def := range domesticLeagueDefinitions {
		for _, suffix := range []string{"F", "G", "H"} {
			id := def.ID + "-" + suffix
			if _, ok := byClub[id]; !ok {
				t.Fatalf("%s (bottom three of %s) was not staked", id, def.Name)
			}
		}
		for _, suffix := range []string{"A", "B", "C", "D", "E"} {
			id := def.ID + "-" + suffix
			if _, ok := byClub[id]; ok {
				t.Fatalf("%s (top five of %s) must not pay stakes", id, def.Name)
			}
		}
	}
}

// The stakes cost reputation and finances, the deduction never pushes a
// balance negative, and the budget invariant (budget <= balance) holds.
func TestApplyRelegationStakesChargesPrice(t *testing.T) {
	tm := buildRelegationTestWorld()
	stakes := tm.planDomesticRelegationStakesUnlocked()

	// A staked club with a tiny balance: the deduction clamps to zero.
	club := tm.Clubs[stakes[0].ClubID]
	club.Finances.Balance = 5
	club.Finances.TransferBudget = 5

	before := map[string]int{}
	for _, c := range tm.ClubsList {
		before[c.ClubID] = c.Identity.Reputation
	}

	tm.applyRelegationStakesUnlocked(stakes)

	for _, stake := range stakes {
		c := tm.Clubs[stake.ClubID]
		if c.Identity.Reputation != before[stake.ClubID]-relegationReputationPenalty {
			t.Fatalf("%s reputation=%d want %d", stake.ClubID, c.Identity.Reputation, before[stake.ClubID]-relegationReputationPenalty)
		}
		if c.Finances.Balance < 0 {
			t.Fatalf("%s balance went negative: %d", stake.ClubID, c.Finances.Balance)
		}
		if c.Finances.TransferBudget > c.Finances.Balance {
			t.Fatalf("%s budget %d exceeds balance %d", stake.ClubID, c.Finances.TransferBudget, c.Finances.Balance)
		}
	}
	// The tiny-balance club paid no finance penalty (10% of 5 rounds down
	// to zero) but still lost reputation and stayed solvent.
	if club.Finances.Balance != 5 || club.Finances.TransferBudget != 5 {
		t.Fatalf("tiny-balance club mischarged: %+v", club.Finances)
	}
	// A full-balance staked club lost exactly 10%.
	full := tm.Clubs[stakes[1].ClubID]
	if full.Finances.Balance != 90_000_000 {
		t.Fatalf("full-balance club balance=%d want 90000000", full.Finances.Balance)
	}
	// Unstaked clubs keep their reputation and finances.
	for _, def := range domesticLeagueDefinitions {
		for _, suffix := range []string{"A", "B", "C", "D", "E"} {
			id := def.ID + "-" + suffix
			c := tm.Clubs[id]
			if c.Identity.Reputation != before[id] {
				t.Fatalf("unstaked %s reputation changed: %d", id, c.Identity.Reputation)
			}
			if c.Finances.Balance != 100_000_000 {
				t.Fatalf("unstaked %s balance changed: %d", id, c.Finances.Balance)
			}
		}
	}
}

// No club ever changes league: the closed country-pure pyramid has no
// boundary swaps, so every club's League field survives the stakes intact.
func TestRelegationStakesNeverMoveClubsBetweenLeagues(t *testing.T) {
	tm := buildRelegationTestWorld()
	startLeague := map[string]string{}
	for _, club := range tm.ClubsList {
		startLeague[club.ClubID] = club.League
	}

	stakes := tm.planDomesticRelegationStakesUnlocked()
	tm.applyRelegationStakesUnlocked(stakes)

	for _, club := range tm.ClubsList {
		if club.League != startLeague[club.ClubID] {
			t.Fatalf("%s changed league: %q -> %q", club.ClubID, startLeague[club.ClubID], club.League)
		}
	}
}

// Identical tables produce identical stakes, in stable league order.
func TestRelegationStakesPlanIsDeterministic(t *testing.T) {
	tm := buildRelegationTestWorld()
	a := tm.planDomesticRelegationStakesUnlocked()
	b := tm.planDomesticRelegationStakesUnlocked()
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("stakes differ between identical tables: %+v vs %+v", a, b)
	}
}
