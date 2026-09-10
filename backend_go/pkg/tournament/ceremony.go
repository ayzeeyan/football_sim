package tournament

import (
	"fmt"
	"sort"

	"football_sim/pkg/models"
)

// Awards ceremony (React AwardsCeremonyModal contract): four ranked
// categories with nominees plus a winner, built from the live season tables.
// Team of the season and manager of the year are not computed server-side;
// the client guards their absence.

// GetAwardsCeremony compiles gala categories from current season data.
func (tm *TournamentManager) GetAwardsCeremony() map[string]interface{} {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.awardsCeremonyUnlocked()
}

func (tm *TournamentManager) awardsCeremonyUnlocked() map[string]interface{} {
	var all []*models.Player
	for _, c := range tm.ClubsList {
		all = append(all, c.Squad...)
	}
	nominee := func(p *models.Player) map[string]interface{} {
		clubName, short := p.ClubID, p.ClubID
		if c := tm.Clubs[p.ClubID]; c != nil {
			clubName, short = c.ClubName, c.ShortName
		}
		return map[string]interface{}{
			"full_name":    p.FullName,
			"position":     p.Position,
			"ovr":          p.OVR,
			"age":          p.Age,
			"goals":        p.Goals,
			"assists":      p.Assists,
			"club_name":    clubName,
			"short_name":   short,
			"is_wonderkid": p.UniverseWonderkid,
			"stats_line":   fmt.Sprintf("%d G · %d A · %d OVR", p.Goals, p.Assists, p.OVR),
		}
	}
	top := func(less func(a, b *models.Player) bool, n int) []map[string]interface{} {
		pool := append([]*models.Player{}, all...)
		sort.Slice(pool, func(i, j int) bool { return less(pool[i], pool[j]) })
		out := []map[string]interface{}{}
		for i := 0; i < n && i < len(pool); i++ {
			out = append(out, nominee(pool[i]))
		}
		return out
	}
	moreGoals := func(a, b *models.Player) bool {
		if a.Goals != b.Goals {
			return a.Goals > b.Goals
		}
		return a.OVR > b.OVR
	}
	moreAssists := func(a, b *models.Player) bool {
		if a.Assists != b.Assists {
			return a.Assists > b.Assists
		}
		return a.OVR > b.OVR
	}
	potsScore := func(p *models.Player) float64 {
		return float64(p.OVR) + float64(p.Goals)*2.5 + float64(p.Assists)*1.5
	}
	bestSeason := func(a, b *models.Player) bool { return potsScore(a) > potsScore(b) }
	wonderScore := func(p *models.Player) int { return p.OVR*100 + p.Goals*3 + p.Assists*2 + p.Appearances }
	bestKid := func(a, b *models.Player) bool {
		aw, bw := 0, 0
		if a.UniverseWonderkid {
			aw = wonderScore(a)
		}
		if b.UniverseWonderkid {
			bw = wonderScore(b)
		}
		if aw != bw {
			return aw > bw
		}
		return a.OVR > b.OVR
	}
	category := func(key, title, blurb string, nominees []map[string]interface{}) map[string]interface{} {
		var winner interface{}
		if len(nominees) > 0 {
			winner = nominees[0]
		}
		return map[string]interface{}{
			"key": key, "title": title, "blurb": blurb,
			"nominees": nominees, "winner": winner,
		}
	}
	categories := []map[string]interface{}{
		category("golden_boot", "Golden Boot", "Most Super League goals this season.", top(moreGoals, 4)),
		category("playmaker", "Playmaker Award", "Most assists this season.", top(moreAssists, 4)),
		category("golden_boy", "Golden Boy", "Best U-14 prodigy this season.", top(bestKid, 4)),
		category("player_of_the_season", "Player of the Season", "Open to all: goals, assists and aura.", top(bestSeason, 4)),
	}
	return map[string]interface{}{
		"season_name": tm.SeasonName,
		"categories":  categories,
	}
}
