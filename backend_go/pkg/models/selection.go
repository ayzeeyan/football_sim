package models

import "strings"

// CompetitionImportance ranks how much a fixture should use first-choice players.
// 0–100. Early domestic cups sit low so managers rotate; European knockouts sit high.
func CompetitionImportance(competition string, matchweek int) int {
	c := strings.ToLower(strings.TrimSpace(competition))
	switch c {
	case "champions-league", "ucl":
		if matchweek >= 29 {
			return 92
		}
		return 78
	case "europa-league":
		if matchweek >= 29 {
			return 84
		}
		return 68
	case "conference-league":
		if matchweek >= 29 {
			return 76
		}
		return 62
	case "super-cup":
		return 74
	case "fa-cup", "copa-del-rey", "dfb-pokal", "coppa-italia", "coupe-de-france":
		if matchweek >= 27 {
			return 72
		}
		return 46
	case "super-league":
		return 70
	default:
		if strings.Contains(c, "cup") {
			if matchweek >= 27 {
				return 70
			}
			return 46
		}
		return 66
	}
}
