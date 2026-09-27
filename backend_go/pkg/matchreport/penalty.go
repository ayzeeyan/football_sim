package matchreport

import (
	"sort"

	"football_sim/pkg/growth"
	"football_sim/pkg/models"
)

// DesignatedPenaltyTaker is the first-choice spot-kick taker from a kickoff XI.
func DesignatedPenaltyTaker(xi []*models.Player, ge *growth.GrowthEngine) *models.Player {
	return PickPenaltyTaker(xi, ge, "")
}

// PickPenaltyTaker chooses from CURRENT active players only. A designated
// taker is used when still on the pitch; otherwise the strongest remaining
// option wins. Ties break on PlayerID. Map iteration is never used.
func PickPenaltyTaker(active []*models.Player, ge *growth.GrowthEngine, designatedID string) *models.Player {
	cands := make([]*models.Player, 0, len(active))
	for _, p := range active {
		if p == nil || p.PlayerID == "" {
			continue
		}
		cands = append(cands, p)
	}
	if len(cands) == 0 {
		return nil
	}
	if designatedID != "" {
		for _, p := range cands {
			if p.PlayerID == designatedID {
				return p
			}
		}
	}
	sort.SliceStable(cands, func(i, j int) bool {
		si, sj := penaltyScore(cands[i], ge), penaltyScore(cands[j], ge)
		if si != sj {
			return si > sj
		}
		return cands[i].PlayerID < cands[j].PlayerID
	})
	return cands[0]
}

func penaltyScore(p *models.Player, ge *growth.GrowthEngine) int {
	if p == nil {
		return 0
	}
	shooting := 0
	if ge != nil {
		if attrs := ge.Attributes[p.PlayerID]; attrs != nil {
			shooting = attrs.Shooting
		}
	}
	if shooting == 0 {
		switch p.Category {
		case "FWD":
			shooting = p.OVR
		case "MID":
			shooting = p.OVR - 4
		case "DEF":
			shooting = p.OVR - 12
		default:
			shooting = p.OVR - 22
		}
	}
	pos := 0
	switch p.Category {
	case "FWD":
		pos = 30
	case "MID":
		pos = 16
	case "DEF":
		pos = 6
	}
	return shooting*4 + p.Composure*2 + p.OVR + pos
}
