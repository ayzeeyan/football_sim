package server

import (
	"encoding/json"
	"io"
	"net/http"
	"sort"

	"football_sim/pkg/models"
)

// Post-match resolution: instant simulations, scoring races, awards and history.
func (s *Server) handleSimulateFixture(w http.ResponseWriter, r *http.Request) {
	s.worldMu.Lock()
	held := true
	defer func() {
		if held {
			s.worldMu.Unlock()
		}
	}()

	fid := r.PathValue("fixture_id")
	res := s.TournamentManager.SimulateFixture(fid)
	if res["status"] == "error" {
		msg, _ := res["message"].(string)
		if msg == "" {
			msg = "Could not simulate."
		}
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	snap, gen := s.takeCareerSnapshotLocked()
	s.worldMu.Unlock()
	held = false
	s.commitCareerSnapshot(snap, gen)
	writeJSON(w, res)
}

func (s *Server) handleSimulateRemaining(w http.ResponseWriter, r *http.Request) {
	// Optional watch-one exclusion: the live-committed fixture must never be
	// replayed. Empty bodies (legacy callers) simulate the whole slate.
	var req struct {
		ExcludeFixtureID string `json:"exclude_fixture_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		writeErrorJSON(w, http.StatusBadRequest, "Invalid simulate-remaining request")
		return
	}
	s.worldMu.Lock()
	held := true
	defer func() {
		if held {
			s.worldMu.Unlock()
		}
	}()
	exclude := req.ExcludeFixtureID
	if exclude == "" {
		exclude = r.URL.Query().Get("exclude_fixture_id")
	}
	// Safety net: never resim the currently-selected live fixture, even when
	// the client forgets to pass it.
	if exclude == "" {
		exclude = s.liveFixtureID
	}
	var res map[string]interface{}
	if exclude != "" {
		res = s.TournamentManager.SimulateRemainingExcluding(exclude)
	} else {
		res = s.TournamentManager.SimulateRemaining()
	}
	snap, gen := s.takeCareerSnapshotLocked()
	s.worldMu.Unlock()
	held = false
	s.commitCareerSnapshot(snap, gen)
	writeJSON(w, res)
}

// handleGetFavourite returns the legacy persisted viewing preference (may be empty).
func (s *Server) handleGetScoringRace(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	var all []*models.Player
	for _, c := range s.TournamentManager.ClubsList {
		all = append(all, c.Squad...)
	}

	sort.Slice(all, func(i, j int) bool {
		if all[i].Goals != all[j].Goals {
			return all[i].Goals > all[j].Goals
		}
		if all[i].Assists != all[j].Assists {
			return all[i].Assists > all[j].Assists
		}
		if all[i].OVR != all[j].OVR {
			return all[i].OVR > all[j].OVR
		}
		return all[i].PlayerID < all[j].PlayerID
	})

	var race []map[string]interface{}
	for i := 0; i < 5 && i < len(all); i++ {
		p := all[i]
		club := s.TournamentManager.Clubs[p.ClubID]
		cName := p.ClubID
		cShort := ""
		if club != nil {
			cName = club.ClubName
			cShort = club.ShortName
		}
		race = append(race, map[string]interface{}{
			"player_id":    p.PlayerID,
			"full_name":    p.FullName,
			"position":     p.Position,
			"ovr":          p.OVR,
			"goals":        p.Goals,
			"assists":      p.Assists,
			"club_name":    cName,
			"club_short":   cShort,
			"is_wonderkid": p.UniverseWonderkid,
		})
	}
	s.worldMu.RUnlock()
	writeJSON(w, race)
}

func (s *Server) handleGetTrophies(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	cabinet := s.TournamentManager.GetTrophyCabinet()
	s.worldMu.RUnlock()
	writeJSON(w, map[string]interface{}{"status": "success", "cabinet": cabinet})
}

func (s *Server) handleGetRecords(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	payload := map[string]interface{}{
		"status":  "success",
		"records": s.TournamentManager.GetAllTimeRecords(),
	}
	s.worldMu.RUnlock()
	writeJSON(w, payload)
}

func (s *Server) handleGetSeasonAwards(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	awards := s.TournamentManager.GetSeasonAwards()
	s.worldMu.RUnlock()
	writeJSON(w, awards)
}

// handleGetAwardsCeremony serves the gala contract: ranked categories with
// nominees and winners, plus Ballon d'Or shortlist, Team of the Season, and Manager of the Year.
// Guarded: returns HTTP 409 Conflict if called mid-season or after season rollover.
func (s *Server) handleGetAwardsCeremony(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	if !s.TournamentManager.AwardsCeremonyReady() {
		s.worldMu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "awards ceremony is only available at the conclusion of the season before rollover",
		})
		return
	}
	ceremony := s.TournamentManager.GetAwardsCeremony()
	awards := s.TournamentManager.GetSeasonAwards()
	s.worldMu.RUnlock()
	ceremony["ballon_dor"] = awards["ballon_dor"]
	ceremony["team_of_the_season"] = awards["team_of_the_season"]
	ceremony["manager_of_the_year"] = awards["manager_of_the_year"]
	writeJSON(w, ceremony)
}

func (s *Server) handleGetSeasonHistory(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	defer s.worldMu.RUnlock()
	writeJSON(w, s.careerHistoryPayload())
}

// careerHistoryPayload builds the History-tab contract: current season
// awards, live table, archived past seasons, trophy cabinet, and records.
// Caller must hold worldMu (TournamentManager methods take their own locks).
func (s *Server) careerHistoryPayload() map[string]interface{} {
	tm := s.TournamentManager
	table := []map[string]interface{}{}
	for _, c := range tm.GetStandings() {
		table = append(table, map[string]interface{}{
			"club_name":  c.ClubName,
			"short_name": c.ShortName,
			"pts":        c.Points,
			"gd":         c.GoalDifference,
			"p":          c.Played,
		})
	}
	past := []map[string]interface{}{}
	past = append(past, tm.SeasonHistory...)
	return map[string]interface{}{
		"season_name":       tm.SeasonName,
		"current_matchweek": tm.CurrentMatchweek,
		"max_matchweeks":    tm.MaxMatchweeks,
		"season_phase":      tm.SeasonPhase,
		"ucl_stage":         tm.UCLStage,
		"super_cup_stage":   tm.SuperCupStage,
		"current":           tm.GetSeasonAwards(),
		"table":             table,
		"past":              past,
		"trophy_cabinet":    tm.GetTrophyCabinet(),
		"all_time_records":  tm.GetAllTimeRecords(),
		"recent_results":    tm.RecentFinishedSummaries(16),
	}
}

func (s *Server) handleGetSeasonStats(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	stats := s.TournamentManager.GetSeasonStats()
	s.worldMu.RUnlock()
	writeJSON(w, stats)
}

// handleGetAdvancedSeasonStats serves the statistics centre aggregation.
func (s *Server) handleGetAdvancedSeasonStats(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	payload := s.TournamentManager.GetAdvancedSeasonStats()
	s.worldMu.RUnlock()
	writeJSON(w, payload)
}
