package tournament

import "football_sim/pkg/models"

// GetCalendar returns the season calendar strip (Python: get_calendar).
func (tm *TournamentManager) GetCalendar() map[string]interface{} {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	view := tm.CurrentMatchweek
	if view > tm.MaxMatchweeks {
		view = tm.MaxMatchweeks
	}
	weeks := make([]map[string]interface{}, 0, tm.MaxMatchweeks)
	for mw := 1; mw <= tm.MaxMatchweeks; mw++ {
		leagueN, uclN, scN := 0, 0, 0
		for i := range tm.Fixtures {
			if tm.Fixtures[i].Matchweek == mw {
				leagueN++
			}
		}
		for i := range tm.UCLFixtures {
			if tm.UCLFixtures[i].Matchweek == mw {
				uclN++
			}
		}
		for i := range tm.SuperCupFixtures {
			if tm.SuperCupFixtures[i].Matchweek == mw {
				scN++
			}
		}
		weeks = append(weeks, map[string]interface{}{
			"matchweek": mw,
			"phase":     LeaguePhase(mw),
			"month":     MonthLabel(mw),
			"year":      CalendarYear(tm.SeasonName, mw),
			"chapter":   WeekChapter(mw),
			"league":    leagueN,
			"ucl":       uclN,
			"super_cup": scN,
			"current":   mw == view,
		})
	}
	nextCup := 0
	overdue := false
	for i := range tm.UCLFixtures {
		f := &tm.UCLFixtures[i]
		if f.Status == "scheduled" && f.Matchweek < view {
			overdue = true
		}
	}
	for i := range tm.SuperCupFixtures {
		f := &tm.SuperCupFixtures[i]
		if f.Status == "scheduled" && f.Matchweek < view {
			overdue = true
		}
	}
	if overdue {
		nextCup = view
	} else {
		for mw := view; mw <= tm.MaxMatchweeks; mw++ {
			found := false
			for i := range tm.UCLFixtures {
				if tm.UCLFixtures[i].Matchweek == mw && tm.UCLFixtures[i].Status == "scheduled" {
					found = true
					break
				}
			}
			if !found {
				for i := range tm.SuperCupFixtures {
					if tm.SuperCupFixtures[i].Matchweek == mw && tm.SuperCupFixtures[i].Status == "scheduled" {
						found = true
						break
					}
				}
			}
			if found {
				nextCup = mw
				break
			}
		}
	}
	var nextCupVal interface{}
	if nextCup > 0 {
		nextCupVal = nextCup
	}
	return map[string]interface{}{
		"current_matchweek": tm.CurrentMatchweek,
		"max_matchweeks":    tm.MaxMatchweeks,
		"season_name":       tm.SeasonName,
		"phase":             LeaguePhase(view),
		"month":             MonthLabel(view),
		"year":              CalendarYear(tm.SeasonName, view),
		"chapter":           WeekChapter(view),
		"this_week":         view,
		"next_cup_night":    nextCupVal,
		"weeks":             weeks,
	}
}

func (tm *TournamentManager) dumpTie(t CupTie) map[string]interface{} {
	return map[string]interface{}{
		"home":       tm.Clubs[t.HomeID],
		"away":       tm.Clubs[t.AwayID],
		"home_id":    t.HomeID,
		"away_id":    t.AwayID,
		"leg1":       t.Leg1,
		"leg2":       t.Leg2,
		"winner":     tm.Clubs[t.WinnerID],
		"winner_id":  t.WinnerID,
		"decided_by": t.DecidedBy,
		"penalties":  t.Penalties,
	}
}

func dumpTieMap(tm *TournamentManager, src map[string]CupTie) map[string]interface{} {
	out := map[string]interface{}{}
	for k, t := range src {
		out[k] = tm.dumpTie(t)
	}
	return out
}

// GetUCLState returns the Champions Cup board (Python: get_ucl).
func (tm *TournamentManager) GetUCLState() map[string]interface{} {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	a, b := tm.uclStandingsUnlocked()
	var final interface{}
	if tm.UCLFinal.HomeID != "" || tm.UCLFinal.AwayID != "" {
		score := interface{}(nil)
		if len(tm.UCLFinal.Leg1) == 2 {
			score = tm.UCLFinal.Leg1
		}
		final = map[string]interface{}{
			"team1":      tm.Clubs[tm.UCLFinal.HomeID],
			"team2":      tm.Clubs[tm.UCLFinal.AwayID],
			"score":      score,
			"winner":     tm.Clubs[tm.UCLFinal.WinnerID],
			"decided_by": tm.UCLFinal.DecidedBy,
			"penalties":  tm.UCLFinal.Penalties,
		}
	}
	var champ *models.Club
	if tm.UCLChampionID != "" {
		champ = tm.Clubs[tm.UCLChampionID]
	}
	return map[string]interface{}{
		"stage":          tm.UCLStage,
		"group_a":        a,
		"group_b":        b,
		"records":        tm.UCLRecords,
		"quarter_finals": dumpTieMap(tm, tm.UCLQuarterFinals),
		"semi_finals":    dumpTieMap(tm, tm.UCLSemiFinals),
		"final":          final,
		"champion":       champ,
	}
}

// GetSuperCupState returns the Super Cup board (Python: get_super_cup).
func (tm *TournamentManager) GetSuperCupState() map[string]interface{} {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	var final interface{}
	if tm.SuperCupFinal.HomeID != "" || tm.SuperCupFinal.AwayID != "" {
		score := interface{}(nil)
		if len(tm.SuperCupFinal.Leg1) == 2 {
			score = tm.SuperCupFinal.Leg1
		}
		final = map[string]interface{}{
			"team1":      tm.Clubs[tm.SuperCupFinal.HomeID],
			"team2":      tm.Clubs[tm.SuperCupFinal.AwayID],
			"score":      score,
			"winner":     tm.Clubs[tm.SuperCupFinal.WinnerID],
			"decided_by": tm.SuperCupFinal.DecidedBy,
			"penalties":  tm.SuperCupFinal.Penalties,
		}
	}
	var champ *models.Club
	if tm.SuperCupChampionID != "" {
		champ = tm.Clubs[tm.SuperCupChampionID]
	}
	return map[string]interface{}{
		"stage":          tm.SuperCupStage,
		"byes":           tm.SuperCupByes,
		"play_in":        dumpTieMap(tm, tm.SuperCupPlayIn),
		"quarter_finals": dumpTieMap(tm, tm.SuperCupQuarterFinals),
		"semi_finals":    dumpTieMap(tm, tm.SuperCupSemiFinals),
		"final":          final,
		"champion":       champ,
	}
}

// UCLFixturesCopy returns a snapshot of Champions Cup fixtures.
func (tm *TournamentManager) UCLFixturesCopy() []Fixture {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	out := make([]Fixture, len(tm.UCLFixtures))
	copy(out, tm.UCLFixtures)
	return out
}
