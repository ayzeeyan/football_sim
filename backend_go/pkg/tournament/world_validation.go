package tournament

import (
	"fmt"
	"football_sim/pkg/models"
)

// ValidateWorldState performs deep structural and domain invariant checks on
// the current TournamentManager state. It returns a non-nil error if any invariant
// is violated.
func (tm *TournamentManager) ValidateWorldState() error {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	// 1. Season phase validation
	if tm.SeasonPhase != "season" && tm.SeasonPhase != "transfer_window" {
		return fmt.Errorf("invalid or unknown season phase: %q", tm.SeasonPhase)
	}

	// 2. Matchweek bounds validation
	if tm.CurrentMatchweek < 1 || tm.CurrentMatchweek > tm.MaxMatchweeks+1 {
		return fmt.Errorf("current matchweek %d out of bounds (1..%d)", tm.CurrentMatchweek, tm.MaxMatchweeks+1)
	}

	// 3. Clubs & Squad validation
	seenClubIDs := make(map[string]bool)
	seenPlayerIDs := make(map[string]string) // player_id -> club_id

	for _, club := range tm.ClubsList {
		if club == nil {
			return fmt.Errorf("nil club found in ClubsList")
		}
		if club.ClubID == "" {
			return fmt.Errorf("club has empty ClubID")
		}
		if seenClubIDs[club.ClubID] {
			return fmt.Errorf("duplicate ClubID found in universe: %s", club.ClubID)
		}
		seenClubIDs[club.ClubID] = true

		if mapClub, exists := tm.Clubs[club.ClubID]; !exists || mapClub != club {
			return fmt.Errorf("club %s in ClubsList does not match map entry in tm.Clubs", club.ClubID)
		}

		if club.OverallTeamRating < 1 || club.OverallTeamRating > 99 {
			return fmt.Errorf("club %s overall rating %d out of bounds (1..99)", club.ClubID, club.OverallTeamRating)
		}

		// Squad players validation
		for _, p := range club.Squad {
			if p == nil {
				return fmt.Errorf("nil player found in squad of club %s", club.ClubID)
			}
			if p.PlayerID == "" {
				return fmt.Errorf("player in club %s has empty PlayerID", club.ClubID)
			}
			if existingClub, exists := seenPlayerIDs[p.PlayerID]; exists {
				return fmt.Errorf("duplicate PlayerID %s found in both %s and %s", p.PlayerID, existingClub, club.ClubID)
			}
			seenPlayerIDs[p.PlayerID] = club.ClubID

			if p.ClubID != club.ClubID {
				return fmt.Errorf("player %s has ClubID %q, expected %q", p.PlayerID, p.ClubID, club.ClubID)
			}
			if p.OVR < 1 || p.OVR > 99 {
				return fmt.Errorf("player %s OVR %d out of bounds", p.PlayerID, p.OVR)
			}
			if p.Goals < 0 || p.Assists < 0 || p.Appearances < 0 || p.CareerGoals < 0 || p.CareerAssists < 0 || p.CareerApps < 0 {
				return fmt.Errorf("player %s has negative stat counters", p.PlayerID)
			}
		}
	}

	// 4. Standings / Table validation
	seenStandingsClubs := make(map[string]bool)
	for _, c := range tm.GetStandings() {
		if c == nil {
			return fmt.Errorf("nil club in standings")
		}
		if !seenClubIDs[c.ClubID] {
			return fmt.Errorf("unknown club ID %s in standings", c.ClubID)
		}
		if seenStandingsClubs[c.ClubID] {
			return fmt.Errorf("duplicate club ID %s in standings", c.ClubID)
		}
		seenStandingsClubs[c.ClubID] = true
	}

	// 5. Manager validation
	for clubID, mgr := range tm.Managers {
		if !seenClubIDs[clubID] {
			return fmt.Errorf("manager assigned to non-existent club ID %s", clubID)
		}
		if mgr == nil {
			return fmt.Errorf("nil manager profile for club %s", clubID)
		}
		if mgr.Name == "" {
			return fmt.Errorf("manager for club %s has empty name", clubID)
		}
	}

	// 6. Fixtures validation
	seenFixtureIDs := make(map[string]bool)
	for i := range tm.Fixtures {
		f := &tm.Fixtures[i]
		if f.FixtureID == "" {
			return fmt.Errorf("fixture at index %d has empty FixtureID", i)
		}
		if seenFixtureIDs[f.FixtureID] {
			return fmt.Errorf("duplicate FixtureID: %s", f.FixtureID)
		}
		seenFixtureIDs[f.FixtureID] = true

		if !seenClubIDs[f.HomeID] {
			return fmt.Errorf("fixture %s has unknown HomeID %s", f.FixtureID, f.HomeID)
		}
		if !seenClubIDs[f.AwayID] {
			return fmt.Errorf("fixture %s has unknown AwayID %s", f.FixtureID, f.AwayID)
		}
		if f.HomeID == f.AwayID {
			return fmt.Errorf("fixture %s has identical home and away club %s", f.FixtureID, f.HomeID)
		}
		if f.Status == "finished" {
			if f.HomeGoals == nil || f.AwayGoals == nil {
				return fmt.Errorf("finished fixture %s is missing goals result", f.FixtureID)
			}
			if *f.HomeGoals < 0 || *f.AwayGoals < 0 {
				return fmt.Errorf("finished fixture %s has negative goals", f.FixtureID)
			}
		}
	}

	// 7. Transfer Engine validation
	if tm.TransferEngine != nil {
		if tm.TransferEngine.CurrentWeek < 1 || tm.TransferEngine.CurrentWeek > 13 {
			return fmt.Errorf("transfer engine week %d out of bounds (1..13)", tm.TransferEngine.CurrentWeek)
		}
	}

	return nil
}

// EnsurePlayerPointers returns a reference to a player by ID if they exist.
func (tm *TournamentManager) FindPlayerByID(playerID string) (*models.Player, *models.Club) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	for _, club := range tm.ClubsList {
		for _, p := range club.Squad {
			if p != nil && p.PlayerID == playerID {
				return p, club
			}
		}
	}
	return nil, nil
}
