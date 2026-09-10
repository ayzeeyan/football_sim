package tournament

import (
	"fmt"
	"math"

	"football_sim/pkg/models"
)

// ValidateWorldState checks durable universe invariants without mutating state.
// It is intended for save loading, tests, soak runs, and developer diagnostics.
// Callers that share a TournamentManager across goroutines should hold the
// appropriate external world lock while validating.
func (tm *TournamentManager) ValidateWorldState() error {
	if tm == nil {
		return fmt.Errorf("world validation: tournament manager is nil")
	}
	if tm.SeasonName == "" {
		return fmt.Errorf("world validation: season name is empty")
	}
	switch tm.SeasonPhase {
	case "season", "transfer_window":
	default:
		return fmt.Errorf("world validation: unknown season phase %q", tm.SeasonPhase)
	}
	if tm.MaxMatchweeks < 1 {
		return fmt.Errorf("world validation: max matchweeks must be positive, got %d", tm.MaxMatchweeks)
	}
	if tm.CurrentMatchweek < 1 || tm.CurrentMatchweek > tm.MaxMatchweeks+1 {
		return fmt.Errorf("world validation: current matchweek %d outside legal range 1..%d", tm.CurrentMatchweek, tm.MaxMatchweeks+1)
	}

	clubIDs := make(map[string]struct{}, len(tm.ClubsList))
	playerIDs := make(map[string]string)
	for i, club := range tm.ClubsList {
		if club == nil {
			return fmt.Errorf("world validation: clubs list contains nil club at index %d", i)
		}
		if club.ClubID == "" {
			return fmt.Errorf("world validation: club at index %d has empty id", i)
		}
		if _, exists := clubIDs[club.ClubID]; exists {
			return fmt.Errorf("world validation: duplicate club id %q", club.ClubID)
		}
		clubIDs[club.ClubID] = struct{}{}
		if mapped := tm.Clubs[club.ClubID]; mapped != club {
			if mapped == nil {
				return fmt.Errorf("world validation: club %q missing from club map", club.ClubID)
			}
			return fmt.Errorf("world validation: club %q map/list references disagree", club.ClubID)
		}
		if err := validateClubState(club, playerIDs); err != nil {
			return err
		}
	}
	if len(tm.Clubs) != len(clubIDs) {
		return fmt.Errorf("world validation: club map/list size mismatch map=%d list=%d", len(tm.Clubs), len(clubIDs))
	}
	for id, club := range tm.Clubs {
		if club == nil {
			return fmt.Errorf("world validation: club map contains nil club for %q", id)
		}
		if id != club.ClubID {
			return fmt.Errorf("world validation: club map key %q does not match club id %q", id, club.ClubID)
		}
		if _, ok := clubIDs[id]; !ok {
			return fmt.Errorf("world validation: club %q exists in map but not clubs list", id)
		}
	}

	for clubID, manager := range tm.Managers {
		if _, ok := clubIDs[clubID]; !ok {
			return fmt.Errorf("world validation: manager references unknown club %q", clubID)
		}
		if manager == nil {
			return fmt.Errorf("world validation: active manager for club %q is nil", clubID)
		}
		if manager.ClubID != "" && manager.ClubID != clubID {
			return fmt.Errorf("world validation: manager %q club id %q does not match map key %q", manager.Name, manager.ClubID, clubID)
		}
		if manager.BudgetEur < 0 {
			return fmt.Errorf("world validation: manager %q has negative budget %d", manager.Name, manager.BudgetEur)
		}
	}
	for _, entry := range tm.ManagerHistory {
		if entry.ClubID != "" {
			if _, ok := clubIDs[entry.ClubID]; !ok {
				return fmt.Errorf("world validation: manager history references unknown club %q", entry.ClubID)
			}
		}
		if entry.Matchweek < 0 {
			return fmt.Errorf("world validation: manager history has negative matchweek for club %q", entry.ClubID)
		}
	}

	fixtureIDs := make(map[string]struct{})
	for name, fixtures := range map[string][]Fixture{
		"league": tm.Fixtures,
		"ucl": tm.UCLFixtures,
		"super_cup": tm.SuperCupFixtures,
	} {
		for i := range fixtures {
			if err := validateFixtureState(name, i, &fixtures[i], clubIDs, fixtureIDs); err != nil {
				return err
			}
		}
	}

	if tm.TransferEngine != nil {
		te := tm.TransferEngine
		if te.CurrentWeek < 1 || te.CurrentWeek > 13 {
			return fmt.Errorf("world validation: transfer week %d outside legal range 1..13", te.CurrentWeek)
		}
		if te.CurrentDay < 0 || te.CurrentMatchweek < 0 {
			return fmt.Errorf("world validation: transfer counters cannot be negative (day=%d matchweek=%d)", te.CurrentDay, te.CurrentMatchweek)
		}
		for _, transfer := range te.CompletedTransfers {
			if transfer.FeeEUR < 0 {
				return fmt.Errorf("world validation: completed transfer for player %q has negative fee %d", transfer.PlayerID, transfer.FeeEUR)
			}
			if transfer.SellerID != "" {
				if _, ok := clubIDs[transfer.SellerID]; !ok {
					return fmt.Errorf("world validation: completed transfer references unknown seller %q", transfer.SellerID)
				}
			}
			if transfer.BuyerID != "" {
				if _, ok := clubIDs[transfer.BuyerID]; !ok {
					return fmt.Errorf("world validation: completed transfer references unknown buyer %q", transfer.BuyerID)
				}
			}
		}
		for _, negotiation := range te.ActiveNegotiations {
			if negotiation == nil {
				return fmt.Errorf("world validation: active transfer negotiations contain nil entry")
			}
			if negotiation.CurrentBid < 0 || negotiation.AskingPrice < 0 {
				return fmt.Errorf("world validation: negotiation %q has negative financial value", negotiation.NegotiationID)
			}
			if negotiation.Buyer != nil {
				if _, ok := clubIDs[negotiation.Buyer.ClubID]; !ok {
					return fmt.Errorf("world validation: negotiation %q references unknown buyer %q", negotiation.NegotiationID, negotiation.Buyer.ClubID)
				}
			}
			if negotiation.Seller != nil {
				if _, ok := clubIDs[negotiation.Seller.ClubID]; !ok {
					return fmt.Errorf("world validation: negotiation %q references unknown seller %q", negotiation.NegotiationID, negotiation.Seller.ClubID)
				}
			}
		}
	}

	if tm.GrowthEngine != nil {
		for playerID, bio := range tm.GrowthEngine.Biometrics {
			if bio == nil {
				return fmt.Errorf("world validation: biometric profile for player %q is nil", playerID)
			}
			if bio.PlayerID != "" && bio.PlayerID != playerID {
				return fmt.Errorf("world validation: biometric map key %q does not match profile player id %q", playerID, bio.PlayerID)
			}
			values := []struct {
				name  string
				value float64
			}{
				{"current height", bio.CurrentHeightCM},
				{"baseline height", bio.BaselineHeightCM},
				{"current weight", bio.CurrentWeightKG},
				{"baseline weight", bio.BaselineWeightKG},
				{"growth velocity", bio.GrowthVelocity},
				{"accumulated XP", bio.AccumulatedXP},
				{"level XP target", bio.LevelXPTarget},
				{"yearly height taken", bio.YearlyHeightTaken},
			}
			for _, item := range values {
				if math.IsNaN(item.value) || math.IsInf(item.value, 0) {
					return fmt.Errorf("world validation: player %q biometric %s is not finite", playerID, item.name)
				}
			}
			if bio.Age < 0 || bio.Potential < 0 || bio.Potential > 100 || bio.CurrentHeightCM <= 0 || bio.CurrentWeightKG <= 0 || bio.LevelXPTarget < 0 || bio.AccumulatedXP < 0 {
				return fmt.Errorf("world validation: player %q has invalid biometric bounds", playerID)
			}
		}
	}

	return nil
}

func validateClubState(club *models.Club, playerIDs map[string]string) error {
	if club.Played < 0 || club.Won < 0 || club.Drawn < 0 || club.Lost < 0 || club.GoalsFor < 0 || club.GoalsAgainst < 0 || club.Points < 0 {
		return fmt.Errorf("world validation: club %q has negative standings values", club.ClubID)
	}
	if club.Played != club.Won+club.Drawn+club.Lost {
		return fmt.Errorf("world validation: club %q played=%d but W+D+L=%d", club.ClubID, club.Played, club.Won+club.Drawn+club.Lost)
	}
	if club.GoalDifference != club.GoalsFor-club.GoalsAgainst {
		return fmt.Errorf("world validation: club %q goal difference=%d but GF-GA=%d", club.ClubID, club.GoalDifference, club.GoalsFor-club.GoalsAgainst)
	}
	if club.OverallTeamRating < 0 || club.OverallTeamRating > 100 {
		return fmt.Errorf("world validation: club %q team rating %d outside 0..100", club.ClubID, club.OverallTeamRating)
	}
	for i, player := range club.Squad {
		if player == nil {
			return fmt.Errorf("world validation: club %q squad contains nil player at index %d", club.ClubID, i)
		}
		if player.PlayerID == "" {
			return fmt.Errorf("world validation: club %q has player with empty id at index %d", club.ClubID, i)
		}
		if previousClub, exists := playerIDs[player.PlayerID]; exists {
			return fmt.Errorf("world validation: duplicate player id %q in clubs %q and %q", player.PlayerID, previousClub, club.ClubID)
		}
		playerIDs[player.PlayerID] = club.ClubID
		if player.ClubID != "" && player.ClubID != club.ClubID {
			return fmt.Errorf("world validation: player %q claims club %q but is in squad %q", player.PlayerID, player.ClubID, club.ClubID)
		}
		if player.Goals < 0 || player.Assists < 0 || player.Appearances < 0 || player.CareerGoals < 0 || player.CareerAssists < 0 || player.CareerApps < 0 {
			return fmt.Errorf("world validation: player %q has negative statistics", player.PlayerID)
		}
		if player.OVR < 0 || player.OVR > 100 {
			return fmt.Errorf("world validation: player %q OVR %d outside 0..100", player.PlayerID, player.OVR)
		}
		if player.Age < 0 || player.MarketValueEUR < 0 || player.WageEUR < 0 {
			return fmt.Errorf("world validation: player %q has invalid age/value/wage", player.PlayerID)
		}
	}
	return nil
}

func validateFixtureState(scope string, index int, fixture *Fixture, clubIDs map[string]struct{}, fixtureIDs map[string]struct{}) error {
	if fixture == nil {
		return fmt.Errorf("world validation: %s fixture %d is nil", scope, index)
	}
	if fixture.FixtureID != "" {
		if _, exists := fixtureIDs[fixture.FixtureID]; exists {
			return fmt.Errorf("world validation: duplicate fixture id %q", fixture.FixtureID)
		}
		fixtureIDs[fixture.FixtureID] = struct{}{}
	}
	if fixture.HomeID == "" || fixture.AwayID == "" {
		return fmt.Errorf("world validation: %s fixture %q has empty home/away club id", scope, fixture.FixtureID)
	}
	if fixture.HomeID == fixture.AwayID {
		return fmt.Errorf("world validation: fixture %q has identical home and away club %q", fixture.FixtureID, fixture.HomeID)
	}
	if _, ok := clubIDs[fixture.HomeID]; !ok {
		return fmt.Errorf("world validation: fixture %q references unknown home club %q", fixture.FixtureID, fixture.HomeID)
	}
	if _, ok := clubIDs[fixture.AwayID]; !ok {
		return fmt.Errorf("world validation: fixture %q references unknown away club %q", fixture.FixtureID, fixture.AwayID)
	}
	if fixture.Home != nil && fixture.Home.ClubID != fixture.HomeID {
		return fmt.Errorf("world validation: fixture %q home pointer id %q does not match home id %q", fixture.FixtureID, fixture.Home.ClubID, fixture.HomeID)
	}
	if fixture.Away != nil && fixture.Away.ClubID != fixture.AwayID {
		return fmt.Errorf("world validation: fixture %q away pointer id %q does not match away id %q", fixture.FixtureID, fixture.Away.ClubID, fixture.AwayID)
	}
	switch fixture.Status {
	case "", "scheduled", "playing", "finished":
	default:
		return fmt.Errorf("world validation: fixture %q has unknown status %q", fixture.FixtureID, fixture.Status)
	}
	if (fixture.HomeGoals == nil) != (fixture.AwayGoals == nil) {
		return fmt.Errorf("world validation: fixture %q has only one goal value set", fixture.FixtureID)
	}
	if fixture.HomeGoals != nil && (*fixture.HomeGoals < 0 || *fixture.AwayGoals < 0) {
		return fmt.Errorf("world validation: fixture %q has negative score", fixture.FixtureID)
	}
	if fixture.Status == "finished" && (fixture.HomeGoals == nil || fixture.AwayGoals == nil) {
		return fmt.Errorf("world validation: finished fixture %q has no result", fixture.FixtureID)
	}
	return nil
}
