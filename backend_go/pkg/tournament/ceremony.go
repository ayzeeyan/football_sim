package tournament

import (
	"fmt"
	"hash/fnv"
	"sort"

	"football_sim/pkg/models"
)

// AwardsCeremonyReady indicates whether the season matches have completed
// and the post-season awards ceremony is available before advancing to the next season.
func (tm *TournamentManager) AwardsCeremonyReady() bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.awardsCeremonyReadyUnlocked()
}

func (tm *TournamentManager) awardsCeremonyReadyUnlocked() bool {
	return tm.CurrentMatchweek > tm.MaxMatchweeks
}

// Awards ceremony data deliberately separates winner identity from nominee
// presentation order. The winner is selected from football criteria first;
// finalists are then deterministically shuffled for presentation so card
// position can never reveal the result.
func (tm *TournamentManager) GetAwardsCeremony() map[string]interface{} {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.awardsCeremonyUnlocked()
}

type rankedPlayer struct {
	player *models.Player
	score  float64
}

// Deterministic ranking tie-breakers:
// 1. score desc
// 2. OVR desc
// 3. goals + assists desc
// 4. PlayerID asc
func rankPlayers(pool []*models.Player, score func(*models.Player) float64, n int) []rankedPlayer {
	rows := make([]rankedPlayer, 0, len(pool))
	for _, p := range pool {
		if p != nil {
			rows = append(rows, rankedPlayer{player: p, score: score(p)})
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].score != rows[j].score {
			return rows[i].score > rows[j].score
		}
		if rows[i].player.OVR != rows[j].player.OVR {
			return rows[i].player.OVR > rows[j].player.OVR
		}
		gaI := rows[i].player.Goals + rows[i].player.Assists
		gaJ := rows[j].player.Goals + rows[j].player.Assists
		if gaI != gaJ {
			return gaI > gaJ
		}
		return rows[i].player.PlayerID < rows[j].player.PlayerID
	})
	if n > 0 && len(rows) > n {
		rows = rows[:n]
	}
	return rows
}

func goalScore(p *models.Player) float64 {
	if p == nil {
		return 0
	}
	return float64(p.Goals)*10000 + float64(p.Assists)*10 + float64(p.OVR)/100
}

func assistScore(p *models.Player) float64 {
	if p == nil {
		return 0
	}
	return float64(p.Assists)*10000 + float64(p.Goals)*10 + float64(p.OVR)/100
}

func seasonScore(p *models.Player) float64 {
	if p == nil {
		return 0
	}
	return float64(p.Goals)*3 + float64(p.Assists)*2 + float64(p.Appearances)*0.5 + float64(p.OVR)
}

func (tm *TournamentManager) ballonScoreUnlocked(p *models.Player) float64 {
	if p == nil {
		return 0
	}
	teamBonus := 0.0
	standings := tm.standingsUnlocked()
	if len(standings) > 0 && standings[0].ClubID == p.ClubID {
		teamBonus += 10
	}
	if tm.UCLChampionID != "" && tm.UCLChampionID == p.ClubID {
		teamBonus += 12
	}
	if tm.SuperCupChampionID != "" && tm.SuperCupChampionID == p.ClubID {
		teamBonus += 5
	}
	for i, c := range standings {
		if c.ClubID == p.ClubID {
			teamBonus += float64(maxInt(0, len(standings)-i)) * 0.75
			break
		}
	}
	return round2(float64(p.OVR)*1.5 + float64(p.Goals)*4 + float64(p.Assists)*2.5 + float64(p.Appearances)*0.35 + teamBonus)
}

func (tm *TournamentManager) allPlayersUnlocked() []*models.Player {
	all := make([]*models.Player, 0)
	for _, c := range tm.ClubsList {
		for _, p := range c.Squad {
			if p != nil {
				all = append(all, p)
			}
		}
	}
	return all
}

func (tm *TournamentManager) ballonDorUnlocked() []map[string]interface{} {
	all := tm.allPlayersUnlocked()
	ranked := rankPlayers(all, tm.ballonScoreUnlocked, 10)
	out := make([]map[string]interface{}, 0, len(ranked))
	for idx, r := range ranked {
		p := r.player
		clubShort := p.ClubID
		if c := tm.Clubs[p.ClubID]; c != nil {
			clubShort = c.ShortName
		}
		out = append(out, map[string]interface{}{
			"rank":         idx + 1,
			"player_id":    p.PlayerID,
			"full_name":    p.FullName,
			"club_short":   clubShort,
			"position":     p.Position,
			"ovr":          p.OVR,
			"goals":        p.Goals,
			"assists":      p.Assists,
			"score":        round2(r.score),
			"is_wonderkid": p.UniverseWonderkid,
		})
	}
	return out
}

func (tm *TournamentManager) totsCardUnlocked(p *models.Player, role string) map[string]interface{} {
	if p == nil {
		return nil
	}
	clubName, clubShort := p.ClubID, p.ClubID
	var color [3]uint8
	if c := tm.Clubs[p.ClubID]; c != nil {
		clubName = c.ClubName
		clubShort = c.ShortName
		color = c.PrimaryColor
	}
	return map[string]interface{}{
		"player_id":     p.PlayerID,
		"full_name":     p.FullName,
		"position":      p.Position,
		"role":          role,
		"ovr":           p.OVR,
		"age":           p.Age,
		"club_id":       p.ClubID,
		"club_name":     clubName,
		"club_short":    clubShort,
		"primary_color": []int{int(color[0]), int(color[1]), int(color[2])},
		"goals":         p.Goals,
		"assists":       p.Assists,
		"appearances":   p.Appearances,
		"is_wonderkid":  p.UniverseWonderkid,
	}
}

func (tm *TournamentManager) teamOfTheSeasonUnlocked() map[string]interface{} {
	all := tm.allPlayersUnlocked()
	ranked := rankPlayers(all, seasonScore, 0)
	used := make(map[string]bool)

	pickOne := func(primary func(*models.Player) bool, fallback func(*models.Player) bool) *models.Player {
		for _, r := range ranked {
			if !used[r.player.PlayerID] && primary(r.player) {
				used[r.player.PlayerID] = true
				return r.player
			}
		}
		if fallback != nil {
			for _, r := range ranked {
				if !used[r.player.PlayerID] && fallback(r.player) {
					used[r.player.PlayerID] = true
					return r.player
				}
			}
		}
		for _, r := range ranked {
			if !used[r.player.PlayerID] {
				used[r.player.PlayerID] = true
				return r.player
			}
		}
		return nil
	}

	gk := pickOne(func(p *models.Player) bool { return models.GetPositionCategory(p.Position) == "GK" }, nil)
	lb := pickOne(func(p *models.Player) bool { return p.Position == "LB" || p.Position == "LWB" }, func(p *models.Player) bool { return models.GetPositionCategory(p.Position) == "DEF" })
	cb1 := pickOne(func(p *models.Player) bool { return p.Position == "CB" }, func(p *models.Player) bool { return models.GetPositionCategory(p.Position) == "DEF" })
	cb2 := pickOne(func(p *models.Player) bool { return p.Position == "CB" }, func(p *models.Player) bool { return models.GetPositionCategory(p.Position) == "DEF" })
	rb := pickOne(func(p *models.Player) bool { return p.Position == "RB" || p.Position == "RWB" }, func(p *models.Player) bool { return models.GetPositionCategory(p.Position) == "DEF" })
	mid1 := pickOne(func(p *models.Player) bool { return models.GetPositionCategory(p.Position) == "MID" }, nil)
	mid2 := pickOne(func(p *models.Player) bool { return models.GetPositionCategory(p.Position) == "MID" }, nil)
	mid3 := pickOne(func(p *models.Player) bool { return models.GetPositionCategory(p.Position) == "MID" }, nil)
	fwd1 := pickOne(func(p *models.Player) bool { return models.GetPositionCategory(p.Position) == "FWD" }, nil)
	fwd2 := pickOne(func(p *models.Player) bool { return models.GetPositionCategory(p.Position) == "FWD" }, nil)
	fwd3 := pickOne(func(p *models.Player) bool { return models.GetPositionCategory(p.Position) == "FWD" }, nil)

	gkCard := tm.totsCardUnlocked(gk, "GK")
	lbCard := tm.totsCardUnlocked(lb, "LB")
	cb1Card := tm.totsCardUnlocked(cb1, "CB")
	cb2Card := tm.totsCardUnlocked(cb2, "CB")
	rbCard := tm.totsCardUnlocked(rb, "RB")
	mid1Card := tm.totsCardUnlocked(mid1, "CM")
	mid2Card := tm.totsCardUnlocked(mid2, "CM")
	mid3Card := tm.totsCardUnlocked(mid3, "CM")
	fwd1Card := tm.totsCardUnlocked(fwd1, "FWD")
	fwd2Card := tm.totsCardUnlocked(fwd2, "FWD")
	fwd3Card := tm.totsCardUnlocked(fwd3, "FWD")

	xi := []map[string]interface{}{
		gkCard, lbCard, cb1Card, cb2Card, rbCard,
		mid1Card, mid2Card, mid3Card,
		fwd1Card, fwd2Card, fwd3Card,
	}

	return map[string]interface{}{
		"formation": "4-3-3",
		"gk":        gkCard,
		"lb":        lbCard,
		"cb1":       cb1Card,
		"cb2":       cb2Card,
		"rb":        rbCard,
		"mid1":      mid1Card,
		"mid2":      mid2Card,
		"mid3":      mid3Card,
		"fwd1":      fwd1Card,
		"fwd2":      fwd2Card,
		"fwd3":      fwd3Card,
		"xi":        xi,
	}
}

func (tm *TournamentManager) managerOfTheYearUnlocked() map[string]interface{} {
	standings := tm.standingsUnlocked()
	actualFinish := map[string]int{}
	for i, c := range standings {
		actualFinish[c.ClubID] = i + 1
	}

	expectedClubs := append([]*models.Club(nil), tm.ClubsList...)
	sort.SliceStable(expectedClubs, func(i, j int) bool {
		repI := expectedClubs[i].Identity.Reputation
		repJ := expectedClubs[j].Identity.Reputation
		if repI != repJ {
			return repI > repJ
		}
		if expectedClubs[i].Identity.HistoricalPrestige != expectedClubs[j].Identity.HistoricalPrestige {
			return expectedClubs[i].Identity.HistoricalPrestige > expectedClubs[j].Identity.HistoricalPrestige
		}
		return expectedClubs[i].ClubID < expectedClubs[j].ClubID
	})

	expectedFinish := map[string]int{}
	for i, c := range expectedClubs {
		expectedFinish[c.ClubID] = i + 1
	}

	trophiesWon := map[string]int{}
	if len(standings) > 0 {
		trophiesWon[standings[0].ClubID]++
	}
	if tm.UCLChampionID != "" {
		trophiesWon[tm.UCLChampionID]++
	}
	if tm.SuperCupChampionID != "" {
		trophiesWon[tm.SuperCupChampionID]++
	}

	type managerCandidate struct {
		club                *models.Club
		actualFinish        int
		expectedFinish      int
		outperformedPlaces int
		trophiesWon         int
		score               float64
	}

	candidates := make([]managerCandidate, 0, len(tm.ClubsList))
	for _, c := range tm.ClubsList {
		act := actualFinish[c.ClubID]
		if act == 0 {
			act = 12
		}
		exp := expectedFinish[c.ClubID]
		if exp == 0 {
			exp = 12
		}
		out := exp - act
		tWon := trophiesWon[c.ClubID]
		score := float64(out)*10.0 + float64(c.Points)*0.5 + float64(tWon)*25.0
		candidates = append(candidates, managerCandidate{
			club:                c,
			actualFinish:        act,
			expectedFinish:      exp,
			outperformedPlaces: out,
			trophiesWon:         tWon,
			score:               score,
		})
	}

	if len(candidates) == 0 {
		return nil
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		if candidates[i].trophiesWon != candidates[j].trophiesWon {
			return candidates[i].trophiesWon > candidates[j].trophiesWon
		}
		if candidates[i].club.Points != candidates[j].club.Points {
			return candidates[i].club.Points > candidates[j].club.Points
		}
		if candidates[i].outperformedPlaces != candidates[j].outperformedPlaces {
			return candidates[i].outperformedPlaces > candidates[j].outperformedPlaces
		}
		return candidates[i].club.ClubID < candidates[j].club.ClubID
	})

	best := candidates[0]
	accolade := ""
	if best.trophiesWon >= 2 {
		accolade = fmt.Sprintf("Delivered a historic multi-trophy campaign for %s, capturing %d major honors.", best.club.ClubName, best.trophiesWon)
	} else if best.trophiesWon == 1 {
		accolade = fmt.Sprintf("Led %s to silverware with tactical discipline and remarkable squad consistency.", best.club.ClubName)
	} else if best.outperformedPlaces > 0 {
		accolade = fmt.Sprintf("Defied expectations to lift %s %d places above their preseason projection.", best.club.ClubName, best.outperformedPlaces)
	} else {
		accolade = fmt.Sprintf("Steered %s through a high-stakes campaign with consummate tactical poise.", best.club.ClubName)
	}

	mgrName := "Interim Manager"
	tactic := "Tactical System"
	style := "Balanced"
	if tm.Managers != nil {
		if mgr := tm.Managers[best.club.ClubID]; mgr != nil {
			mgrName = mgr.Name
			tactic = mgr.Tactic()
			style = mgr.CanonicalStyle()
		}
	}

	return map[string]interface{}{
		"club_id":             best.club.ClubID,
		"club_name":           best.club.ClubName,
		"short_name":          best.club.ShortName,
		"primary_color":       []int{int(best.club.PrimaryColor[0]), int(best.club.PrimaryColor[1]), int(best.club.PrimaryColor[2])},
		"name":                mgrName,
		"manager_name":        mgrName,
		"tactic":              tactic,
		"style":               style,
		"actual_finish":       best.actualFinish,
		"expected_finish":     best.expectedFinish,
		"outperformed_places": best.outperformedPlaces,
		"pts":                 best.club.Points,
		"trophies_won":        best.trophiesWon,
		"score":               round2(best.score),
		"accolade":            accolade,
	}
}

func (tm *TournamentManager) awardsCeremonyUnlocked() map[string]interface{} {
	all := tm.allPlayersUnlocked()

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

	presentationOrder := func(key string, rows []rankedPlayer) []rankedPlayer {
		out := append([]rankedPlayer(nil), rows...)
		hashKey := func(p *models.Player) uint64 {
			h := fnv.New64a()
			_, _ = h.Write([]byte(tm.SeasonName + "|" + key + "|" + p.PlayerID))
			return h.Sum64()
		}
		sort.SliceStable(out, func(i, j int) bool {
			hi, hj := hashKey(out[i].player), hashKey(out[j].player)
			if hi != hj {
				return hi < hj
			}
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
		if p.UniverseWonderkid && p.Age <= 21 {
			goldenBoyPool = append(goldenBoyPool, p)
		}
	}

	categories := []map[string]interface{}{
		category("golden_boot", "Golden Boot", "Most Super League goals this season.", rankPlayers(all, goalScore, 4)),
		category("playmaker", "Playmaker Award", "Most assists this season.", rankPlayers(all, assistScore, 4)),
		category("golden_boy", "Golden Boy", "Best eligible U-21 franchise wonderkid this season.", rankPlayers(goldenBoyPool, tm.goldenBoyScoreUnlocked, 4)),
		category("player_of_the_season", "Player of the Season", "Best overall season performance.", rankPlayers(all, seasonScore, 4)),
		category("ballon_dor", "European Ballon d'Or", "Top overall player across performance and team achievement.", rankPlayers(all, tm.ballonScoreUnlocked, 4)),
	}
	return map[string]interface{}{
		"season_name":         tm.SeasonName,
		"categories":          categories,
		"ballon_dor":          tm.ballonDorUnlocked(),
		"team_of_the_season":  tm.teamOfTheSeasonUnlocked(),
		"manager_of_the_year": tm.managerOfTheYearUnlocked(),
	}
}
