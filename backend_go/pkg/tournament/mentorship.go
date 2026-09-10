package tournament

import (
	"sort"

	"football_sim/pkg/growth"
	"football_sim/pkg/models"
)

// PairSeniorMentors pairs each franchise wonderkid with an elite senior mentor from their club squad.
func PairSeniorMentors(clubs []*models.Club, ge *growth.GrowthEngine) {
	for _, club := range clubs {
		var prodigies []*models.Player
		for _, p := range club.Squad {
			if p.UniverseWonderkid {
				prodigies = append(prodigies, p)
			}
		}

		for _, prodigy := range prodigies {
			// Check if existing mentor is still at the club
			var currentMentor *models.Player
			if prodigy.MentorID != "" {
				for _, p := range club.Squad {
					if p.PlayerID == prodigy.MentorID && p.PlayerID != prodigy.PlayerID {
						currentMentor = p
						break
					}
				}
			}

			if currentMentor != nil {
				prodigy.MentorName = currentMentor.FullName
				prodigy.MentorOVR = currentMentor.OVR
				if ge != nil {
					ge.SetMentorship(
						prodigy.PlayerID,
						currentMentor.PlayerID,
						currentMentor.FullName,
						currentMentor.OVR,
						prodigy.Personality,
					)
				}
				continue
			}

			// Clear invalid mentor
			prodigy.MentorID = ""
			prodigy.MentorName = ""
			prodigy.MentorOVR = 0

			// Find candidate senior veterans (age >= 26)
			var veterans []*models.Player
			for _, p := range club.Squad {
				if !p.UniverseWonderkid && p.Age >= 26 {
					veterans = append(veterans, p)
				}
			}
			if len(veterans) == 0 {
				for _, p := range club.Squad {
					if !p.UniverseWonderkid && p.PlayerID != prodigy.PlayerID {
						veterans = append(veterans, p)
					}
				}
			}
			if len(veterans) == 0 {
				continue
			}

			// Prefer matching position category
			var sameCategory []*models.Player
			for _, v := range veterans {
				if v.Category == prodigy.Category {
					sameCategory = append(sameCategory, v)
				}
			}

			candidates := veterans
			if len(sameCategory) > 0 {
				candidates = sameCategory
			}

			// Sort by OVR desc, then Age desc
			sort.Slice(candidates, func(i, j int) bool {
				if candidates[i].OVR == candidates[j].OVR {
					return candidates[i].Age > candidates[j].Age
				}
				return candidates[i].OVR > candidates[j].OVR
			})

			bestMentor := candidates[0]
			prodigy.MentorID = bestMentor.PlayerID
			prodigy.MentorName = bestMentor.FullName
			prodigy.MentorOVR = bestMentor.OVR

			if ge != nil {
				ge.SetMentorship(
					prodigy.PlayerID,
					bestMentor.PlayerID,
					bestMentor.FullName,
					bestMentor.OVR,
					prodigy.Personality,
				)
			}
		}
	}
}
