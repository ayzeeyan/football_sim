package tournament

import (
	"strings"

	"football_sim/pkg/models"
)

// MatchImportance represents the prestige, rivalry, and stakes of a scheduled match.
type MatchImportance string

const (
	ImportanceRoutine          MatchImportance = "Routine"
	ImportanceNotable          MatchImportance = "Notable"
	ImportanceImportant        MatchImportance = "Important"
	ImportanceBigMatch         MatchImportance = "Big Match"
	ImportanceDerby            MatchImportance = "Derby"
	ImportanceTitleRace        MatchImportance = "Title Race"
	ImportanceEuropeanDecider  MatchImportance = "European Decider"
	ImportanceRelegationBattle MatchImportance = "Relegation Battle"
	ImportanceCupTie           MatchImportance = "Cup Tie"
	ImportanceSemiFinal        MatchImportance = "Semi-Final"
	ImportanceFinal            MatchImportance = "Final"
	ImportanceCupFinal         MatchImportance = "Final"
	ImportanceSixPointer       MatchImportance = "Title Race"
)

// Rank is a stable numeric ordering used to feature fixtures. Higher is more
// important. Tie-breaks belong to the caller (club strength, fixture id).
func (imp MatchImportance) Rank() int {
	switch imp {
	case ImportanceFinal:
		return 100
	case ImportanceSemiFinal:
		return 90
	case ImportanceEuropeanDecider:
		return 85
	case ImportanceDerby:
		return 80
	case ImportanceTitleRace:
		return 75
	case ImportanceRelegationBattle:
		return 72
	case ImportanceBigMatch:
		return 70
	case ImportanceCupTie:
		return 55
	case ImportanceImportant:
		return 40
	case ImportanceNotable:
		return 25
	default:
		return 10
	}
}

// EvaluateMatchImportance calculates the contextual stakes of a fixture from
// competition stage, derby heat, rivalry history, and league standings proximity.
func (tm *TournamentManager) EvaluateMatchImportance(f *Fixture) MatchImportance {
	if f == nil || tm == nil {
		return ImportanceRoutine
	}

	stage := strings.ToLower(strings.TrimSpace(f.Stage))
	comp := strings.ToLower(strings.TrimSpace(f.Competition))

	// 1. Cup / continental finals and semis
	if stage == "final" {
		return ImportanceFinal
	}
	if stage == "sf" || stage == "semi-final" || strings.Contains(stage, "semi") {
		return ImportanceSemiFinal
	}

	// 2. High-Heat Derbies and Local Rivalries
	derby := f.DerbyName
	if derby == "" {
		derby = GetDerbyName(f.HomeID, f.AwayID)
	}
	heat := f.DerbyHeat
	if derby != "" && heat == 0 {
		heat = 50
		if v, ok := tm.DerbyHeat[derby]; ok {
			heat = v
		}
	}
	if f.IsHighHeatDerby || heat >= 65 || derby != "" {
		return ImportanceDerby
	}

	// 3. Continental & cup knockout stakes
	european := comp == "ucl" || comp == "champions-league" || comp == "europa-league" || comp == "conference-league"
	if european {
		if stage == "qf" || stage == "quarter-final" || strings.Contains(stage, "quarter") || (f.Leg == 2 && stage != "league" && stage != "league phase") {
			return ImportanceEuropeanDecider
		}
		if stage == "league phase" || strings.HasPrefix(stage, "group") {
			return ImportanceNotable
		}
		return ImportanceImportant
	}
	if strings.HasSuffix(comp, "-cup") || strings.Contains(comp, "cup") {
		if stage == "qf" || stage == "quarter-final" || f.Leg == 2 {
			return ImportanceCupTie
		}
		if stage != "league" && stage != "group" && stage != "" {
			return ImportanceCupTie
		}
	}

	// 4. League Standings Proximity and Table Stakes
	homeClub := tm.Clubs[f.HomeID]
	awayClub := tm.Clubs[f.AwayID]
	if homeClub != nil && awayClub != nil && homeClub.League == awayClub.League {
		leagueID := leagueIDForName(homeClub.League)
		var standings []*models.Club
		if tm.World != nil && leagueID != "" {
			standings = tm.worldLeagueStandingsUnlocked(leagueID)
		} else {
			standings = tm.standingsUnlocked()
		}

		homePos, awayPos := -1, -1
		for idx, c := range standings {
			if c.ClubID == f.HomeID {
				homePos = idx + 1
			}
			if c.ClubID == f.AwayID {
				awayPos = idx + 1
			}
		}

		if homePos > 0 && awayPos > 0 {
			diffPos := homePos - awayPos
			if diffPos < 0 {
				diffPos = -diffPos
			}
			ptsDiff := homeClub.Points - awayClub.Points
			if ptsDiff < 0 {
				ptsDiff = -ptsDiff
			}

			// Title clash: both top 3 and within 6 points
			if homePos <= 3 && awayPos <= 3 && ptsDiff <= 6 {
				return ImportanceTitleRace
			}

			// Top-four battle: both top 6 and within 4 points
			if homePos <= 6 && awayPos <= 6 && ptsDiff <= 4 {
				return ImportanceTitleRace
			}

			// Relegation six-pointer: late in season and both in bottom 4
			totalTeams := len(standings)
			if totalTeams >= 12 && tm.CurrentMatchweek >= 20 {
				if homePos >= totalTeams-3 && awayPos >= totalTeams-3 {
					return ImportanceRelegationBattle
				}
			}

			if diffPos <= 2 && ptsDiff <= 3 {
				return ImportanceImportant
			}
		}
	}

	// 5. Late-Season Decisive Matchweeks
	if tm.CurrentMatchweek >= tm.MaxMatchweeks-3 && tm.MaxMatchweeks > 0 {
		return ImportanceImportant
	}

	return ImportanceRoutine
}
