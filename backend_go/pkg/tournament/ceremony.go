package tournament

import (
	"fmt"
	"hash/fnv"
	"sort"

	"football_sim/pkg/models"
)

// Awards ceremony data deliberately separates winner identity from nominee
// presentation order. The winner is selected from football criteria first;
// finalists are then deterministically shuffled for presentation so card
// position can never reveal the result.
func (tm *TournamentManager) GetAwardsCeremony() map[string]interface{} {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.awardsCeremonyUnlocked()
}

func (tm *TournamentManager) awardsCeremonyUnlocked() map[string]interface{} {
	all := make([]*models.Player, 0)
	for _, c := range tm.ClubsList {
		for _, p := range c.Squad {
			if p != nil {
				all = append(all, p)
			}
		}
	}

	nominee := func(p *models.Player) map[string]interface{} {
		clubName, short := p.ClubID, p.ClubID
		if c := tm.Clubs[p.ClubID]; c != nil {
			clubName, short = c.ClubName, c.ShortName
		}
		return map[string]interface{}{
			"player_id": p.PlayerID, "full_name": p.FullName, "position": p.Position,
			"ovr": p.OVR, "age": p.Age, "goals": p.Goals, "assists": p.Assists,
			"appearances": p.Appearances, "club_name": clubName, "short_name": short,
			"is_wonderkid": p.UniverseWonderkid,
			"stats_line": fmt.Sprintf("%d G · %d A · %d OVR", p.Goals, p.Assists, p.OVR),
		}
	}

	type rankedPlayer struct {
		player *models.Player
		score  float64
	}
	rank := func(pool []*models.Player, score func(*models.Player) float64, n int) []rankedPlayer {
		rows := make([]rankedPlayer, 0, len(pool))
		for _, p := range pool {
			if p != nil {
				rows = append(rows, rankedPlayer{player: p, score: score(p)})
			}
		}
		sort.SliceStable(rows, func(i, j int) bool {
			if rows[i].score != rows[j].score { return rows[i].score > rows[j].score }
			if rows[i].player.OVR != rows[j].player.OVR { return rows[i].player.OVR > rows[j].player.OVR }
			return rows[i].player.PlayerID < rows[j].player.PlayerID
		})
		if len(rows) > n { rows = rows[:n] }
		return rows
	}

	presentationOrder := func(key string, rows []rankedPlayer) []rankedPlayer {
		out := append([]rankedPlayer(nil), rows...)
		hashKey := func(p *models.Player) uint64 {
			h := fnv.New64a()
			_, _ = h.Write([]byte(tm.SeasonName + "|" + key + "|" + p.PlayerID))
			return h.Sum64()
		}
		sort.SliceStable(out, func(i, j int) bool {
			hi, hj := hashKey(out[i].player), hashKey(out[j].player)
			if hi != hj { return hi < hj }
			return out[i].player.PlayerID < out[j].player.PlayerID
		})
		return out
	}

	category := func(key, title, blurb string, rows []rankedPlayer) map[string]interface{} {
		var winner interface{}
		winnerID := ""
		if len(rows) > 0 {
			winnerID = rows[0].player.PlayerID
			winner = nominee(rows[0].player)
		}
		display := presentationOrder(key, rows)
		nominees := make([]map[string]interface{}, 0, len(display))
		for _, row := range display {
			n := nominee(row.player)
			n["award_score"] = round2(row.score)
			nominees = append(nominees, n)
		}
		return map[string]interface{}{
			"key": key, "title": title, "blurb": blurb,
			"nominees": nominees, "winner": winner, "winner_id": winnerID,
		}
	}

	goldenBoyPool := make([]*models.Player, 0)
	for _, p := range all {
		if p.UniverseWonderkid && p.Age <= 21 { goldenBoyPool = append(goldenBoyPool, p) }
	}

	goalScore := func(p *models.Player) float64 {
		return float64(p.Goals)*10000 + float64(p.Assists)*10 + float64(p.OVR)/100
	}
	assistScore := func(p *models.Player) float64 {
		return float64(p.Assists)*10000 + float64(p.Goals)*10 + float64(p.OVR)/100
	}
	seasonScore := func(p *models.Player) float64 {
		return float64(p.Goals)*3 + float64(p.Assists)*2 + float64(p.Appearances)*0.5 + float64(p.OVR)
	}
	ballonScore := func(p *models.Player) float64 {
		teamBonus := 0.0
		if tm.UCLChampionID == p.ClubID { teamBonus += 12 }
		if tm.SuperCupChampionID == p.ClubID { teamBonus += 5 }
		standings := tm.standingsUnlocked()
		for i, c := range standings {
			if c.ClubID == p.ClubID {
				teamBonus += float64(maxInt(0, len(standings)-i)) * 0.75
				break
			}
		}
		return float64(p.OVR)*1.5 + float64(p.Goals)*4 + float64(p.Assists)*2.5 + float64(p.Appearances)*0.35 + teamBonus
	}

	categories := []map[string]interface{}{
		category("golden_boot", "Golden Boot", "Most Super League goals this season.", rank(all, goalScore, 4)),
		category("playmaker", "Playmaker Award", "Most assists this season.", rank(all, assistScore, 4)),
		category("golden_boy", "Golden Boy", "Best eligible U-21 franchise wonderkid this season.", rank(goldenBoyPool, tm.goldenBoyScoreUnlocked, 4)),
		category("player_of_the_season", "Player of the Season", "Best overall season performance.", rank(all, seasonScore, 4)),
		category("ballon_dor", "European Ballon d'Or", "Top overall player across performance and team achievement.", rank(all, ballonScore, 4)),
	}
	return map[string]interface{}{"season_name": tm.SeasonName, "categories": categories}
}
