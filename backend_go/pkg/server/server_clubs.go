package server

import (
	"math"
	"net/http"

	"football_sim/pkg/growth"
	"football_sim/pkg/models"
	"football_sim/pkg/tournament"
)

// Club and player read endpoints (squads, profiles, history, head-to-head).
func (s *Server) handleGetClubs(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	var result []map[string]interface{}
	for _, c := range s.TournamentManager.ClubsList {
		result = append(result, s.serializeClub(c))
	}
	s.worldMu.RUnlock()
	writeJSON(w, result)
}

func (s *Server) handleGetClubSquad(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	cid := r.PathValue("club_id")
	club := s.TournamentManager.Clubs[cid]
	var squad []map[string]interface{}
	if club != nil {
		for _, p := range club.Squad {
			squad = append(squad, s.serializePlayer(p))
		}
	}
	s.worldMu.RUnlock()

	if club == nil {
		http.Error(w, "Club not found", http.StatusNotFound)
		return
	}
	writeJSON(w, squad)
}

func (s *Server) handleGetClubXI(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	cid := r.PathValue("club_id")
	club := s.TournamentManager.Clubs[cid]
	startersJSON := []map[string]interface{}{}
	if club != nil {
		style, focus := "", ""
		if mgr := s.TournamentManager.Managers[cid]; mgr != nil {
			style, focus = mgr.Style, mgr.Focus
		}
		slots := club.GetStartingElevenSlotsWithBias(style, focus, s.clubFixtureContext(cid))
		for _, entry := range slots {
			p := entry.Player
			if p == nil {
				continue
			}
			row := s.serializePlayer(p)
			row["starting_slot"] = entry.Slot
			startersJSON = append(startersJSON, row)
		}
	}
	s.worldMu.RUnlock()

	if club == nil {
		http.Error(w, "Club not found", http.StatusNotFound)
		return
	}
	writeJSON(w, startersJSON)
}

func (s *Server) handleGetClubHistory(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	cid := r.PathValue("club_id")
	club := s.TournamentManager.Clubs[cid]
	var payload map[string]interface{}
	if club != nil {
		hist := append([]map[string]interface{}{}, s.TournamentManager.ClubSeasonHistory[cid]...)
		if hist == nil {
			hist = []map[string]interface{}{}
		}
		payload = map[string]interface{}{
			"club_id":          club.ClubID,
			"club_name":        club.ClubName,
			"short_name":       club.ShortName,
			"primary_color":    club.PrimaryColor,
			"history":          hist,
			"trophies_summary": clubHistoryTrophySummary(hist),
			"historical":       tournament.HistoricalClubTrophies[cid],
		}
	}
	s.worldMu.RUnlock()

	if club == nil {
		http.Error(w, "Club not found", http.StatusNotFound)
		return
	}
	writeJSON(w, payload)
}

func clubHistoryTrophySummary(hist []map[string]interface{}) map[string]int {
	out := map[string]int{"super_league": 0, "ucl": 0, "super_cup": 0}
	for _, row := range hist {
		var titles []string
		switch v := row["trophies"].(type) {
		case []string:
			titles = v
		case []interface{}:
			for _, item := range v {
				if s, ok := item.(string); ok {
					titles = append(titles, s)
				}
			}
		}
		for _, title := range titles {
			switch title {
			case "Super League Champion":
				out["super_league"]++
			case "Champions Cup":
				out["ucl"]++
			case "Super Cup":
				out["super_cup"]++
			}
		}
	}
	return out
}

func (s *Server) handleGetH2H(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	clubA := r.PathValue("club_a")
	clubB := r.PathValue("club_b")
	res := s.TournamentManager.GetHeadToHead(clubA, clubB)
	s.worldMu.RUnlock()

	if res == nil {
		http.Error(w, "Club not found", http.StatusNotFound)
		return
	}
	writeJSON(w, res)
}

func (s *Server) handleGetPlayerProfile(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	pid := r.PathValue("player_id")
	p, club := s.findPlayer(pid)
	var payload map[string]interface{}
	if p != nil {
		var bio *growth.BiometricProfile
		var attrs *growth.TechnicalAttributes
		var timeline []growth.TimelineEntry
		if s.GrowthEngine != nil {
			// Snapshot structs by value so the encode below cannot race an
			// in-place training update.
			if live := s.GrowthEngine.Biometrics[p.PlayerID]; live != nil {
				cp := *live
				bio = &cp
			}
			if live := s.GrowthEngine.Attributes[p.PlayerID]; live != nil {
				cp := *live
				attrs = &cp
			}
			timeline = append([]growth.TimelineEntry{}, s.GrowthEngine.Timeline[p.PlayerID]...)
		}

		unrestLevel, unrestReason := s.TournamentManager.PlayerUnrestLevel(pid)
		log := s.TournamentManager.PlayerMatchLog(pid, 8)
		var avg interface{}
		rated := 0
		sum := 0.0
		for _, row := range log {
			if rating, ok := row["rating"].(float64); ok {
				sum += rating
				rated++
			}
		}
		if rated > 0 {
			avg = math.Round(sum/float64(rated)*100) / 100
		}
		payload = map[string]interface{}{
			"player":         s.serializePlayer(p),
			"club":           s.serializeClub(club),
			"biometrics":     bio,
			"attributes":     attrs,
			"timeline":       timeline,
			"last_matches":   log,
			"avg_rating":     avg,
			"apps_rated":     rated,
			"previous_clubs": playerPreviousClubs(p),
			"unrest_level":   unrestLevel,
			"unrest_reason":  unrestReason,
		}
	}
	s.worldMu.RUnlock()

	if p == nil {
		http.Error(w, "Player not found", http.StatusNotFound)
		return
	}
	writeJSON(w, payload)
}

// playerPreviousClubs returns the distinct previous club IDs in chronological
// order, excluding the player's current club.
func playerPreviousClubs(p *models.Player) []string {
	if p == nil {
		return nil
	}
	var out []string
	seen := map[string]bool{}
	add := func(id string) {
		if id == "" || id == p.ClubID || seen[id] {
			return
		}
		seen[id] = true
		out = append(out, id)
	}
	add(p.PreviousClubID)
	for _, move := range p.TransferHistory {
		add(move.FromClubID)
		add(move.ToClubID)
	}
	return out
}

func (s *Server) handleGetClubProfile(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	cid := r.PathValue("club_id")
	club := s.TournamentManager.Clubs[cid]
	var payload map[string]interface{}
	if club != nil {
		comps := s.TournamentManager.ClubCompetitionsUnlocked(cid)
		if comps == nil {
			comps = []map[string]interface{}{}
		}
		place, size, _ := s.TournamentManager.ClubLeaguePlaceUnlocked(cid)
		schedule := s.TournamentManager.ClubScheduleUnlocked(cid)
		var next, prev *tournament.Fixture
		for i := range schedule {
			f := &schedule[i]
			if f.Status == "finished" {
				prev = f
			} else if next == nil {
				next = f
			}
		}
		formation := ""
		if mgr := s.TournamentManager.Managers[cid]; mgr != nil {
			formation = mgr.Tactic()
		}
		var sumOVR, sumAge, n int
		var topScorer, topAssister, bestRecent *models.Player
		injuries := []map[string]interface{}{}
		for _, p := range club.Squad {
			if p == nil {
				continue
			}
			sumOVR += p.OVR
			sumAge += p.Age
			n++
			if topScorer == nil || p.Goals > topScorer.Goals {
				topScorer = p
			}
			if topAssister == nil || p.Assists > topAssister.Assists {
				topAssister = p
			}
			if bestRecent == nil || p.Goals+p.Assists > bestRecent.Goals+bestRecent.Assists {
				bestRecent = p
			}
			if p.Injury != "" || p.InjuredMatches > 0 {
				injuries = append(injuries, s.serializePlayer(p))
			}
		}
		avgOVR := 0
		avgAge := 0.0
		if n > 0 {
			avgOVR = int(math.Round(float64(sumOVR) / float64(n)))
			avgAge = float64(sumAge) / float64(n)
		}
		form := append([]string{}, club.Form...)
		if form == nil {
			form = []string{}
		}
		storylines := s.TournamentManager.GenerateSeasonStorylines(cid)
		if storylines == nil {
			storylines = []string{}
		}
		payload = map[string]interface{}{
			"club":            s.serializeClub(club),
			"competitions":    comps,
			"league_position": place,
			"league_size":     size,
			"points":          club.Points,
			"form":            form,
			"next_fixture":    s.serializeFixture(next),
			"previous_result": s.serializeFixture(prev),
			"formation":       formation,
			"squad_avg_ovr":   avgOVR,
			"average_age":     avgAge,
			"squad_morale":    club.Morale,
			"injuries":        injuries,
			"top_scorer":      s.serializePlayer(topScorer),
			"top_assister":    s.serializePlayer(topAssister),
			"best_recent":     s.serializePlayer(bestRecent),
			"storylines":      storylines,
			"season_record": map[string]interface{}{
				"p": club.Played, "w": club.Won, "d": club.Drawn, "l": club.Lost,
				"gf": club.GoalsFor, "ga": club.GoalsAgainst, "gd": club.GoalDifference, "pts": club.Points,
			},
			"board_objective": club.BoardObjective,
			"expected_finish": club.ExpectedFinish,
			"reputation":      club.Identity.Reputation,
		}
	}
	s.worldMu.RUnlock()

	if club == nil {
		http.Error(w, "Club not found", http.StatusNotFound)
		return
	}
	writeJSON(w, payload)
}

func (s *Server) handleGetClubFixtures(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	cid := r.PathValue("club_id")
	club := s.TournamentManager.Clubs[cid]
	var payload map[string]interface{}
	if club != nil {
		schedule := s.TournamentManager.ClubScheduleUnlocked(cid)
		serialized := make([]map[string]interface{}, 0, len(schedule))
		for i := range schedule {
			serialized = append(serialized, s.serializeFixture(&schedule[i]))
		}
		payload = map[string]interface{}{
			"club_id":  cid,
			"fixtures": serialized,
		}
	}
	s.worldMu.RUnlock()

	if club == nil {
		http.Error(w, "Club not found", http.StatusNotFound)
		return
	}
	writeJSON(w, payload)
}

func (s *Server) handleGetClubTransfers(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	cid := r.PathValue("club_id")
	club := s.TournamentManager.Clubs[cid]
	var payload map[string]interface{}
	if club != nil {
		activity := s.TournamentManager.ClubTransferActivityUnlocked(cid)
		activity["club_id"] = cid
		payload = activity
	}
	s.worldMu.RUnlock()

	if club == nil {
		http.Error(w, "Club not found", http.StatusNotFound)
		return
	}
	writeJSON(w, payload)
}

// handleGetClubUnrest returns the derived unrest rows for one club:
// quality players starved of minutes, most severe first. Observational.
func (s *Server) handleGetClubUnrest(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	rows := s.TournamentManager.PlayerUnrestForClub(r.PathValue("club_id"))
	s.worldMu.RUnlock()
	if rows == nil {
		rows = []tournament.PlayerUnrest{}
	}
	writeJSON(w, map[string]interface{}{"unrest": rows})
}
