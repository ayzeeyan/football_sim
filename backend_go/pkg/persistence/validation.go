package persistence

import (
	"fmt"

	"football_sim/pkg/models"
	"football_sim/pkg/tournament"
)

// ValidateCareerSnapshot rejects malformed critical save state before it is
// overlaid onto a live universe. Zero-valued additive fields are deliberately
// accepted so older saves can continue to use RestoreCareer's defaults.
func ValidateCareerSnapshot(snap *CareerSnapshot) error {
	if snap == nil {
		return fmt.Errorf("career snapshot is nil")
	}
	if snap.Version < 0 {
		return fmt.Errorf("career snapshot has invalid version %d", snap.Version)
	}
	if snap.CurrentMatchweek < 0 {
		return fmt.Errorf("career snapshot has negative matchweek %d", snap.CurrentMatchweek)
	}
	if snap.MaxMatchweeks < 0 {
		return fmt.Errorf("career snapshot has negative max matchweeks %d", snap.MaxMatchweeks)
	}
	if snap.CurrentMatchweek > 0 && snap.MaxMatchweeks > 0 && snap.CurrentMatchweek > snap.MaxMatchweeks+1 {
		return fmt.Errorf("career snapshot matchweek %d exceeds legal maximum %d", snap.CurrentMatchweek, snap.MaxMatchweeks+1)
	}
	if snap.SeasonPhase != "" && snap.SeasonPhase != "season" && snap.SeasonPhase != "transfer_window" {
		return fmt.Errorf("career snapshot has unknown season phase %q", snap.SeasonPhase)
	}
	if snap.Transfers.CurrentDay < 0 || snap.Transfers.CurrentMatchweek < 0 || snap.Transfers.CurrentWeek < 0 {
		return fmt.Errorf("career snapshot has negative transfer counters")
	}
	if snap.Transfers.CurrentWeek > 13 {
		return fmt.Errorf("career snapshot transfer week %d exceeds legal maximum 13", snap.Transfers.CurrentWeek)
	}

	// Super League invariant: exactly 12 clubs
	if len(snap.Clubs) != 12 {
		return fmt.Errorf("career snapshot must contain exactly 12 clubs, got %d", len(snap.Clubs))
	}

	clubIDs := make(map[string]struct{}, len(snap.Clubs))
	playerIDs := make(map[string]string)
	for mapID, club := range snap.Clubs {
		if club == nil {
			return fmt.Errorf("career snapshot club %q is nil", mapID)
		}
		if mapID == "" || club.ClubID == "" {
			return fmt.Errorf("career snapshot contains club with empty id")
		}
		if mapID != club.ClubID {
			return fmt.Errorf("career snapshot club map key %q does not match club id %q", mapID, club.ClubID)
		}
		if _, exists := clubIDs[club.ClubID]; exists {
			return fmt.Errorf("career snapshot contains duplicate club id %q", club.ClubID)
		}
		clubIDs[club.ClubID] = struct{}{}
		if club.Played < 0 || club.Won < 0 || club.Drawn < 0 || club.Lost < 0 || club.GoalsFor < 0 || club.GoalsAgainst < 0 || club.Points < 0 {
			return fmt.Errorf("career snapshot club %q has negative standings values", club.ClubID)
		}
		for _, player := range club.Squad {
			if player == nil {
				return fmt.Errorf("career snapshot club %q contains nil player", club.ClubID)
			}
			if player.PlayerID == "" {
				return fmt.Errorf("career snapshot club %q contains player with empty id", club.ClubID)
			}
			if previous, exists := playerIDs[player.PlayerID]; exists {
				return fmt.Errorf("career snapshot duplicate player id %q in clubs %q and %q", player.PlayerID, previous, club.ClubID)
			}
			playerIDs[player.PlayerID] = club.ClubID
			if player.Goals < 0 || player.Assists < 0 || player.Appearances < 0 || player.CareerGoals < 0 || player.CareerAssists < 0 || player.CareerApps < 0 {
				return fmt.Errorf("career snapshot player %q has negative statistics", player.PlayerID)
			}
			if player.OVR < 0 || player.OVR > 100 || player.Age < 0 || player.MarketValueEUR < 0 || player.WageEUR < 0 {
				return fmt.Errorf("career snapshot player %q has invalid rating, age, value, or wage", player.PlayerID)
			}
		}
	}

	for clubID, manager := range snap.Managers {
		if manager == nil {
			return fmt.Errorf("career snapshot manager for club %q is nil", clubID)
		}
		if _, ok := clubIDs[clubID]; len(clubIDs) > 0 && !ok {
			return fmt.Errorf("career snapshot manager references unknown club %q", clubID)
		}
		if manager.ClubID != "" && manager.ClubID != clubID {
			return fmt.Errorf("career snapshot manager %q references club %q but is keyed by %q", manager.Name, manager.ClubID, clubID)
		}
		if manager.BudgetEur < 0 {
			return fmt.Errorf("career snapshot manager %q has negative budget", manager.Name)
		}
	}

	fixtureIDs := make(map[string]struct{})
	for scope, fixtures := range map[string][]tournamentFixtureView{
		"league":    fixtureViews(snap.Fixtures),
		"ucl":       fixtureViews(snap.UCLFixtures),
		"super_cup": fixtureViews(snap.SuperCupFixtures),
	} {
		for _, fixture := range fixtures {
			if fixture.ID != "" {
				if _, exists := fixtureIDs[fixture.ID]; exists {
					return fmt.Errorf("career snapshot contains duplicate fixture id %q", fixture.ID)
				}
				fixtureIDs[fixture.ID] = struct{}{}
			}
			if fixture.HomeID == "" || fixture.AwayID == "" || fixture.HomeID == fixture.AwayID {
				return fmt.Errorf("career snapshot %s fixture %q has invalid home/away clubs", scope, fixture.ID)
			}
			if len(clubIDs) > 0 {
				if _, ok := clubIDs[fixture.HomeID]; !ok {
					return fmt.Errorf("career snapshot fixture %q references unknown home club %q", fixture.ID, fixture.HomeID)
				}
				if _, ok := clubIDs[fixture.AwayID]; !ok {
					return fmt.Errorf("career snapshot fixture %q references unknown away club %q", fixture.ID, fixture.AwayID)
				}
			}
			if fixture.Status != "" && fixture.Status != "scheduled" && fixture.Status != "playing" && fixture.Status != "finished" {
				return fmt.Errorf("career snapshot fixture %q has unknown status %q", fixture.ID, fixture.Status)
			}
			if fixture.OneScoreOnly || (fixture.Status == "finished" && !fixture.HasScore) || fixture.NegativeScore {
				return fmt.Errorf("career snapshot fixture %q has impossible result state", fixture.ID)
			}
		}
	}

	// Knockout cup structure validation
	if snap.UCLFinal.WinnerID != "" && snap.UCLFinal.WinnerID != snap.UCLFinal.HomeID && snap.UCLFinal.WinnerID != snap.UCLFinal.AwayID {
		return fmt.Errorf("career snapshot ucl final winner %q must be one of finalists (%s, %s)", snap.UCLFinal.WinnerID, snap.UCLFinal.HomeID, snap.UCLFinal.AwayID)
	}
	if snap.SuperCupFinal.WinnerID != "" && snap.SuperCupFinal.WinnerID != snap.SuperCupFinal.HomeID && snap.SuperCupFinal.WinnerID != snap.SuperCupFinal.AwayID {
		return fmt.Errorf("career snapshot super cup final winner %q must be one of finalists (%s, %s)", snap.SuperCupFinal.WinnerID, snap.SuperCupFinal.HomeID, snap.SuperCupFinal.AwayID)
	}

	// Growth & progression validations
	if snap.Growth.TrainingEnergy < 0 {
		return fmt.Errorf("career snapshot training energy is negative")
	}
	if snap.Growth.MaxTrainingEnergy > 0 && snap.Growth.TrainingEnergy > snap.Growth.MaxTrainingEnergy {
		return fmt.Errorf("career snapshot training energy %d exceeds maximum %d", snap.Growth.TrainingEnergy, snap.Growth.MaxTrainingEnergy)
	}
	for pid, bio := range snap.Growth.Biometrics {
		if bio == nil {
			continue
		}
		if bio.Potential < 50 || bio.Potential > 99 {
			return fmt.Errorf("career snapshot player %q has invalid potential %d", pid, bio.Potential)
		}
		canonical := pid
		if c, ok := ProdigyMap[pid]; ok {
			canonical = c
		}
		if models.IsCanonicalWonderkidID(canonical) {
			if bio.Potential < 93 || bio.Potential > 96 {
				return fmt.Errorf("career snapshot canonical wonderkid %q potential %d outside [93, 96]", pid, bio.Potential)
			}
		}
		if bio.CurrentHeightCM < 0 || bio.BaselineHeightCM < 0 || bio.CurrentWeightKG < 0 {
			return fmt.Errorf("career snapshot player %q has negative biometrics", pid)
		}
	}

	// Transfer validation: current-window deals & negotiations
	for _, transfer := range snap.Transfers.Completed {
		if transfer.FeeEUR < 0 {
			return fmt.Errorf("career snapshot completed transfer for player %q has negative fee", transfer.PlayerID)
		}
		if transfer.PlayerID == "" {
			return fmt.Errorf("career snapshot completed transfer has empty player id")
		}
		// Historical buyers/sellers outside the active 12 clubs are explicitly permitted
		// to prevent false rejections of multi-season saves or foreign signings.
	}

	for _, neg := range snap.Transfers.ActiveNegotiations {
		if neg == nil {
			continue
		}
		if neg.NegotiationID == "" {
			return fmt.Errorf("career snapshot negotiation has empty id")
		}
		if neg.CurrentBid < 0 || neg.AskingPrice < 0 {
			return fmt.Errorf("career snapshot negotiation %q has negative bid or asking price", neg.NegotiationID)
		}
		// Expired or rejected negotiations must never be rejected
		if neg.StageName == "expired" || neg.StageName == "rejected" {
			continue
		}
	}

	return nil
}

type tournamentFixtureView struct {
	ID            string
	HomeID        string
	AwayID        string
	Status        string
	HasScore      bool
	OneScoreOnly  bool
	NegativeScore bool
}

func fixtureViews(fixtures []tournament.Fixture) []tournamentFixtureView {
	out := make([]tournamentFixtureView, 0, len(fixtures))
	for i := range fixtures {
		f := &fixtures[i]
		hasHome, hasAway := f.HomeGoals != nil, f.AwayGoals != nil
		negative := (hasHome && *f.HomeGoals < 0) || (hasAway && *f.AwayGoals < 0)
		out = append(out, tournamentFixtureView{
			ID: f.FixtureID, HomeID: f.HomeID, AwayID: f.AwayID, Status: f.Status,
			HasScore: hasHome && hasAway, OneScoreOnly: hasHome != hasAway, NegativeScore: negative,
		})
	}
	return out
}
