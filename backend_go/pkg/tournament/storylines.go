package tournament

import (
	"fmt"
	"strings"

	"football_sim/pkg/models"
)

// GenerateSeasonStorylines produces factual narrative headlines from live standings,
// streaks, scoring races, and European progression. Never generates fabricated facts.
func (tm *TournamentManager) GenerateSeasonStorylines(clubID string) []string {
	if tm == nil {
		return nil
	}

	var storylines []string
	club := tm.Clubs[clubID]

	// 1. Club-specific form streaks
	if club != nil && len(club.Form) >= 3 {
		form := club.Form
		winStreak := 0
		for i := len(form) - 1; i >= 0; i-- {
			if form[i] == "W" {
				winStreak++
			} else {
				break
			}
		}
		unbeatenStreak := 0
		for i := len(form) - 1; i >= 0; i-- {
			if form[i] == "W" || form[i] == "D" {
				unbeatenStreak++
			} else {
				break
			}
		}

		if winStreak >= 4 {
			storylines = append(storylines, fmt.Sprintf("%s have won %d consecutive matches across all competitions.", club.ShortName, winStreak))
		} else if unbeatenStreak >= 5 {
			storylines = append(storylines, fmt.Sprintf("%s are unbeaten in their last %d matches.", club.ShortName, unbeatenStreak))
		}
	}

	// 2. League Standing & Title Race Narratives
	if club != nil {
		leagueID := leagueIDForName(club.League)
		var standings []*models.Club
		if tm.World != nil && leagueID != "" {
			standings = tm.worldLeagueStandingsUnlocked(leagueID)
		} else {
			standings = tm.standingsUnlocked()
		}

		pos := -1
		for idx, c := range standings {
			if c.ClubID == club.ClubID {
				pos = idx + 1
				break
			}
		}

		if pos == 1 && len(standings) > 1 {
			lead := club.Points - standings[1].Points
			if lead >= 6 {
				storylines = append(storylines, fmt.Sprintf("%s command a comfortable %d-point cushion at the summit of %s.", club.ShortName, lead, club.League))
			} else if lead > 0 {
				storylines = append(storylines, fmt.Sprintf("%s hold a tight %d-point lead over %s in the title race.", club.ShortName, lead, standings[1].ShortName))
			} else {
				storylines = append(storylines, fmt.Sprintf("%s top %s on goal difference in a thrilling title battle.", club.ShortName, club.League))
			}
		} else if pos == 2 && len(standings) > 0 {
			gap := standings[0].Points - club.Points
			if gap <= 3 {
				storylines = append(storylines, fmt.Sprintf("%s sit just %d point(s) behind leaders %s entering the weekend.", club.ShortName, gap, standings[0].ShortName))
			}
		} else if pos > 0 && pos <= 4 && tm.CurrentMatchweek >= 15 {
			storylines = append(storylines, fmt.Sprintf("%s occupy %s Champions League qualification spot in %d position.", club.ShortName, club.League, pos))
		} else if pos > len(standings)-3 && len(standings) >= 12 && tm.CurrentMatchweek >= 15 {
			storylines = append(storylines, fmt.Sprintf("%s face mounting pressure in the relegation battle, sitting %d in %s.", club.ShortName, pos, club.League))
		}
	}

	// 3. Top Scorer / Wonderkid Spotlight
	if tm.GrowthEngine != nil && club != nil {
		var topClubScorer *models.Player
		for _, p := range club.Squad {
			if p != nil && p.Goals > 0 {
				if topClubScorer == nil || p.Goals > topClubScorer.Goals {
					topClubScorer = p
				}
			}
		}
		if topClubScorer != nil && topClubScorer.Goals >= 8 {
			storylines = append(storylines, fmt.Sprintf("%s has struck %d goals in %d appearances for %s this campaign.", topClubScorer.FullName, topClubScorer.Goals, topClubScorer.Appearances, club.ShortName))
		}
	}

	// 4. European World Highlights
	if tm.World != nil {
		for _, id := range tm.World.CompetitionOrder {
			comp := tm.World.Competitions[id]
			if comp != nil && comp.Kind == CompetitionEuropean {
				if comp.Stage == "final" && comp.ChampionID != "" {
					champ := tm.Clubs[comp.ChampionID]
					if champ != nil {
						storylines = append(storylines, fmt.Sprintf("%s were crowned %s champions after a historic campaign.", champ.ShortName, comp.Name))
					}
				}
			}
		}
	}

	// 5. Transfer Window Narratives
	if tm.TransferEngine != nil && tm.TransferEngine.IsWindowOpen() {
		tw := tm.TransferEngine
		if tw.CurrentWeek >= tw.WindowWeeks() {
			storylines = append(storylines, "Deadline Day is underway: clubs scramble to finalize last-minute registrations.")
		} else if tw.CurrentWeek >= tw.WindowWeeks()-2 {
			storylines = append(storylines, fmt.Sprintf("Transfer Window entering final stages (Week %d of %d).", tw.CurrentWeek, tw.WindowWeeks()))
		}
	}

	if len(storylines) == 0 {
		storylines = append(storylines, fmt.Sprintf("Season %s continues as clubs battle across domestic and continental campaigns.", tm.SeasonName))
	}

	return storylines
}

// GenerateWorldStorylines builds factual headlines from the whole European
// world. Club selection must never change this list.
func (tm *TournamentManager) GenerateWorldStorylines() []string {
	if tm == nil {
		return nil
	}
	var storylines []string

	if tm.World != nil {
		for _, def := range domesticLeagueDefinitions {
			table := tm.worldLeagueStandingsUnlocked(def.ID)
			if len(table) < 2 || table[0].Played == 0 {
				continue
			}
			lead := table[0].Points - table[1].Points
			if lead >= 6 {
				storylines = append(storylines, fmt.Sprintf("%s command a %d-point lead at the top of %s.", table[0].ShortName, lead, def.Name))
			} else if lead > 0 {
				storylines = append(storylines, fmt.Sprintf("%s lead %s by %d point(s) in %s.", table[0].ShortName, table[1].ShortName, lead, def.Name))
			} else {
				storylines = append(storylines, fmt.Sprintf("%s and %s are level at the summit of %s.", table[0].ShortName, table[1].ShortName, def.Name))
			}
			if len(table) >= 4 && tm.CurrentMatchweek >= 15 {
				bottom := table[len(table)-1]
				storylines = append(storylines, fmt.Sprintf("%s sit bottom of %s on %d points.", bottom.ShortName, def.Name, bottom.Points))
			}
		}
	}

	var best *models.Club
	bestStreak := 0
	for _, club := range tm.ClubsList {
		if club == nil || len(club.Form) < 3 {
			continue
		}
		streak := 0
		for i := len(club.Form) - 1; i >= 0; i-- {
			if club.Form[i] != "W" {
				break
			}
			streak++
		}
		if streak < 4 {
			continue
		}
		if streak > bestStreak || (streak == bestStreak && (best == nil || club.ClubID < best.ClubID)) {
			bestStreak = streak
			best = club
		}
	}
	if best != nil {
		storylines = append(storylines, fmt.Sprintf("%s have won %d consecutive matches across all competitions.", best.ShortName, bestStreak))
	}

	var topScorer *models.Player
	var topClub *models.Club
	for _, club := range tm.ClubsList {
		if club == nil {
			continue
		}
		for _, p := range club.Squad {
			if p == nil || p.Goals <= 0 {
				continue
			}
			if topScorer == nil || p.Goals > topScorer.Goals || (p.Goals == topScorer.Goals && p.PlayerID < topScorer.PlayerID) {
				topScorer = p
				topClub = club
			}
		}
	}
	if topScorer != nil && topScorer.Goals >= 5 && topClub != nil {
		storylines = append(storylines, fmt.Sprintf("%s leads the scoring charts with %d goals for %s.", topScorer.FullName, topScorer.Goals, topClub.ShortName))
	}

	if tm.World != nil {
		for _, id := range tm.World.CompetitionOrder {
			comp := tm.World.Competitions[id]
			if comp == nil || comp.Kind != CompetitionEuropean {
				continue
			}
			if comp.ChampionID != "" {
				if champ := tm.Clubs[comp.ChampionID]; champ != nil {
					storylines = append(storylines, fmt.Sprintf("%s were crowned %s champions.", champ.ShortName, comp.Name))
				}
				continue
			}
			if comp.Stage != "" && comp.Stage != "league" && comp.Stage != "group" {
				storylines = append(storylines, fmt.Sprintf("%s is in the %s stage.", comp.Name, strings.ReplaceAll(comp.Stage, "-", " ")))
			}
		}
	}

	if tm.TransferEngine != nil && tm.TransferEngine.IsWindowOpen() {
		tw := tm.TransferEngine
		if tw.CurrentWeek >= tw.WindowWeeks() {
			storylines = append(storylines, "Deadline day is underway across the European transfer market.")
		} else {
			storylines = append(storylines, fmt.Sprintf("The transfer window is open (week %d of %d).", tw.CurrentWeek, tw.WindowWeeks()))
		}
	}

	if len(tm.ManagerHistory) > 0 {
		last := tm.ManagerHistory[len(tm.ManagerHistory)-1]
		if last.Action == "sacked" || last.Reason == "results" {
			storylines = append(storylines, fmt.Sprintf("%s have changed manager: %s replaced %s.", last.ClubName, last.NewManager, last.OldManager))
		}
	}

	if len(storylines) == 0 {
		storylines = append(storylines, fmt.Sprintf("Season %s continues across the five leagues and UEFA competitions.", tm.SeasonName))
	}
	if len(storylines) > 8 {
		storylines = storylines[:8]
	}
	return storylines
}

// GenerateClubPulse compiles executive health vitals for the specified club.
func (tm *TournamentManager) GenerateClubPulse(clubID string) map[string]interface{} {
	if tm == nil {
		return nil
	}
	club := tm.Clubs[clubID]
	if club == nil {
		return nil
	}

	// 1. Board Confidence
	boardConfidence := 75
	manager := tm.Managers[clubID]
	if manager != nil {
		switch strings.ToLower(manager.JobSecurity) {
		case "untouchable":
			boardConfidence = 95
		case "very safe":
			boardConfidence = 88
		case "safe":
			boardConfidence = 78
		case "under pressure":
			boardConfidence = 52
		case "critical":
			boardConfidence = 30
		}
	}

	// 2. Average Squad Morale & Dynamics
	totalMorale, count := 0, 0
	fatigued, unhappy, inForm, expiring := 0, 0, 0, 0
	for _, p := range club.Squad {
		if p == nil {
			continue
		}
		count++
		totalMorale += p.Morale
		if p.Fitness < 70 {
			fatigued++
		}
		if p.Morale < 50 {
			unhappy++
		}
		if p.FormBand() == "Superb" || p.FormBand() == "Excellent" {
			inForm++
		}
		if p.ContractYears <= 1 {
			expiring++
		}
	}
	avgMorale := 70
	if count > 0 {
		avgMorale = totalMorale / count
	}

	// 3. Financial Health
	finState := "Healthy"
	if club.WageBill() > club.WageCap() {
		finState = "Wage Squeezed"
	} else if club.Finances.Balance < 10_000_000 {
		finState = "Tight"
	} else if club.Finances.Balance > 100_000_000 {
		finState = "Rich"
	}

	return map[string]interface{}{
		"club_id":           clubID,
		"club_name":         club.ClubName,
		"short_name":        club.ShortName,
		"board_confidence":  boardConfidence,
		"board_objective":   club.BoardObjective,
		"squad_morale":      avgMorale,
		"recent_form":       club.Form,
		"financial_health":  finState,
		"transfer_budget":   club.Finances.TransferBudget,
		"balance":           club.Finances.Balance,
		"wage_bill":         club.WageBill(),
		"wage_cap":          club.WageCap(),
		"fatigued_count":    fatigued,
		"unhappy_count":     unhappy,
		"in_form_count":     inForm,
		"contract_expiring": expiring,
	}
}
