package tournament

import (
	"sort"

	"football_sim/pkg/models"
)

// GetProdigyWatch returns the franchise wonderkids in published Golden Boy order.
func (tm *TournamentManager) GetProdigyWatch() []map[string]interface{} {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	type item struct {
		p     *models.Player
		c     *models.Club
		score float64
	}
	seen := map[string]bool{}
	var rows []item
	for _, c := range tm.ClubsList {
		for _, p := range c.Squad {
			if p == nil || !p.UniverseWonderkid || seen[p.PlayerID] {
				continue
			}
			seen[p.PlayerID] = true
			rows = append(rows, item{p: p, c: c, score: tm.goldenBoyScoreUnlocked(p)})
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].score != rows[j].score {
			return rows[i].score > rows[j].score
		}
		if rows[i].p.OVR != rows[j].p.OVR {
			return rows[i].p.OVR > rows[j].p.OVR
		}
		return rows[i].p.FullName < rows[j].p.FullName
	})
	out := make([]map[string]interface{}, 0, len(rows))
	for i, row := range rows {
		p, c := row.p, row.c
		entry := map[string]interface{}{
			"rank": i + 1, "player_id": p.PlayerID, "full_name": p.FullName,
			"club_id": p.ClubID, "age": p.Age, "position": p.Position,
			"ovr": p.OVR, "goals": p.Goals, "assists": p.Assists,
			"appearances": p.Appearances, "golden_boy_score": row.score,
			"career_goals": p.CareerGoals + p.Goals, "career_assists": p.CareerAssists + p.Assists,
			"career_apps": p.CareerApps + p.Appearances,
		}
		if c != nil {
			entry["club_name"], entry["club_short"] = c.ClubName, c.ShortName
		}
		if tm.GrowthEngine != nil {
			if bio := tm.GrowthEngine.Biometrics[p.PlayerID]; bio != nil {
				entry["potential"] = bio.Potential
				entry["height_cm"] = bio.CurrentHeightCM
				entry["weight_kg"] = bio.CurrentWeightKG
				entry["height_gain_cm"] = bio.HeightGainCM()
				entry["weight_gain_kg"] = bio.WeightGainKG()
				entry["puberty_stage"] = bio.PubertyStage
			}
		}
		out = append(out, entry)
	}
	return out
}
