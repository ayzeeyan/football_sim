package tournament

import (
	"fmt"
	"math/rand"
	"sort"

	"football_sim/pkg/growth"
	"football_sim/pkg/models"
)

type NXGNRankingItem struct {
	Rank             int    `json:"rank"`
	PlayerID         string `json:"player_id"`
	FullName         string `json:"full_name"`
	ClubID           string `json:"club_id"`
	ClubName         string `json:"club_name"`
	ShortName        string `json:"short_name"`
	Age              int    `json:"age"`
	Position         string `json:"position"`
	Category         string `json:"category"`
	OVR              int    `json:"ovr"`
	Potential        int    `json:"potential"`
	Goals            int    `json:"goals"`
	Assists          int    `json:"assists"`
	Appearances      int    `json:"appearances"`
	Personality      string `json:"personality"`
	PersonalityTitle string `json:"personality_title"`
	MentorName       string `json:"mentor_name,omitempty"`
	MentorOVR        int    `json:"mentor_ovr,omitempty"`
	IsWonderkid      bool   `json:"is_wonderkid"`
	ScoutVerdict     string `json:"scout_verdict"`
}

// GenerateNXGN50 produces the annual March wonderkid rankings using authentic growth biometrics.
func GenerateNXGN50(clubs []*models.Club, ge *growth.GrowthEngine) []NXGNRankingItem {
	var candidates []*models.Player
	for _, club := range clubs {
		for _, p := range club.Squad {
			if p.Age <= 19 {
				candidates = append(candidates, p)
			}
		}
	}

	getPot := func(p *models.Player) int {
		if ge != nil {
			if bio, ok := ge.Biometrics[p.PlayerID]; ok && bio != nil {
				return bio.Potential
			}
		}
		pot := p.OVR + (22-p.Age)*3
		if pot > 99 {
			pot = 99
		}
		return pot
	}

	sort.Slice(candidates, func(i, j int) bool {
		potI := getPot(candidates[i])
		potJ := getPot(candidates[j])
		if potI != potJ {
			return potI > potJ
		}
		if candidates[i].OVR != candidates[j].OVR {
			return candidates[i].OVR > candidates[j].OVR
		}
		return candidates[i].Age < candidates[j].Age
	})

	var rankings []NXGNRankingItem
	for idx, p := range candidates {
		if idx >= 50 {
			break
		}
		pot := getPot(p)
		clubName := p.ClubID
		shortName := p.ClubID
		for _, c := range clubs {
			if c.ClubID == p.ClubID {
				clubName = c.ClubName
				shortName = c.ShortName
				break
			}
		}

		verdict := fmt.Sprintf("High-ceiling prospect with projected %d OVR potential.", pot)
		if p.UniverseWonderkid {
			verdict = fmt.Sprintf("Generational talent. Starting at age %d with a world-class %d potential ceiling.", p.Age, pot)
			if p.MentorName != "" {
				verdict += fmt.Sprintf(" Mentored by %s.", p.MentorName)
			}
		}

		rankings = append(rankings, NXGNRankingItem{
			Rank:             idx + 1,
			PlayerID:         p.PlayerID,
			FullName:         p.FullName,
			ClubID:           p.ClubID,
			ClubName:         clubName,
			ShortName:        shortName,
			Age:              p.Age,
			Position:         p.Position,
			Category:         p.Category,
			OVR:              p.OVR,
			Potential:        pot,
			Goals:            p.Goals,
			Assists:          p.Assists,
			Appearances:      p.Appearances,
			Personality:      p.Personality,
			PersonalityTitle: models.ArchetypeForKey(p.Personality).Title,
			MentorName:       p.MentorName,
			MentorOVR:        p.MentorOVR,
			IsWonderkid:      p.UniverseWonderkid,
			ScoutVerdict:     verdict,
		})
	}

	return rankings
}

// CheckWonderkidMilestones triggers autonomous milestone events at specific ages.
func CheckWonderkidMilestones(
	completedMW int,
	seasonName string,
	clubs []*models.Club,
	milestonesFired map[string]map[string]bool,
	rng *rand.Rand,
) []InboxItem {
	var news []InboxItem

	for _, club := range clubs {
		for _, p := range club.Squad {
			if !p.UniverseWonderkid {
				continue
			}

			if milestonesFired[p.PlayerID] == nil {
				milestonesFired[p.PlayerID] = make(map[string]bool)
			}
			fired := milestonesFired[p.PlayerID]

			// Age 15: First Professional Contract
			if p.Age == 15 && !fired["pro_contract"] && completedMW >= 4 {
				fired["pro_contract"] = true
				headline := fmt.Sprintf("%s commits future to %s with first pro contract", p.FullName, club.ShortName)
				body := fmt.Sprintf("At just 15, %s has put pen to paper on his first professional deal. Coaching staff cite his %s approach as foundational.", p.FullName, models.ArchetypeForKey(p.Personality).Badge)
				news = append(news, NewInboxItem(
					fmt.Sprintf("WK_PRO_%s_%d", p.PlayerID, completedMW),
					"wonderkid",
					headline,
					body,
					completedMW,
					seasonName,
					[]string{club.ClubID},
					p.PlayerID,
					"",
				))
			}

			// Age 17: Youth International Call-up
			if p.Age == 17 && !fired["national_team"] && completedMW >= 10 {
				fired["national_team"] = true
				headline := fmt.Sprintf("BREAKING: %s receives first international call-up for Philippines U-19", p.FullName)
				body := fmt.Sprintf("%s's scintillating club form at %s has earned him a maiden call-up to the national youth squad for the upcoming AFC qualifiers.", p.FullName, club.ShortName)
				news = append(news, NewInboxItem(
					fmt.Sprintf("WK_NAT_%s_%d", p.PlayerID, completedMW),
					"wonderkid",
					headline,
					body,
					completedMW,
					seasonName,
					[]string{club.ClubID},
					p.PlayerID,
					"",
				))
			}

			// Age 18: Senior Graduation & Golden Boy shortlist
			if p.Age == 18 && !fired["senior_graduation"] && completedMW >= 20 {
				fired["senior_graduation"] = true
				headline := fmt.Sprintf("%s shortlisted for Golden Boy as he graduates from academy ranks", p.FullName)
				body := fmt.Sprintf("No longer just a prodigy: 18-year-old %s is now an established senior pillar at %s and features on Tuttosport's 20-man Golden Boy shortlist.", p.FullName, club.ShortName)
				news = append(news, NewInboxItem(
					fmt.Sprintf("WK_GRAD_%s_%d", p.PlayerID, completedMW),
					"wonderkid",
					headline,
					body,
					completedMW,
					seasonName,
					[]string{club.ClubID},
					p.PlayerID,
					"",
				))
			}
		}
	}

	return news
}
