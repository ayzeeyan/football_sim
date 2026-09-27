package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"football_sim/pkg/models"
)

// Pre-match reads: competitions, fixtures, calendar, favourites and watch lists.
func (s *Server) handleGetSuperLeague(w http.ResponseWriter, r *http.Request) {
	// Optional ?league= selector (name or competition ID) picks one domestic
	// league's table; the default remains the compatibility view.
	league := r.URL.Query().Get("league")
	s.worldMu.RLock()
	var standings []map[string]interface{}
	for _, c := range s.TournamentManager.GetStandingsForLeague(league) {
		standings = append(standings, s.serializeClub(c))
	}
	payload := map[string]interface{}{
		"clubs":             standings,
		"standings":         standings,
		"season_name":       s.TournamentManager.SeasonName,
		"current_matchweek": s.TournamentManager.CurrentMatchweek,
		"max_matchweeks":    s.TournamentManager.MaxMatchweeks,
		"season_phase":      s.TournamentManager.SeasonPhase,
		"recent_results":    append([]string{}, s.TournamentManager.RecentResults...),
		"world":             s.TournamentManager.World != nil,
	}
	s.worldMu.RUnlock()
	writeJSON(w, payload)
}

func (s *Server) handleGetCompetitions(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	payload := s.TournamentManager.GetCompetitions()
	world := s.TournamentManager.World != nil
	s.worldMu.RUnlock()
	writeJSON(w, map[string]interface{}{"world": world, "competitions": payload})
}

func (s *Server) handleGetCompetition(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	payload := s.TournamentManager.GetCompetition(r.PathValue("competition_id"))
	s.worldMu.RUnlock()
	if payload == nil {
		http.Error(w, "Competition not found", http.StatusNotFound)
		return
	}
	writeJSON(w, payload)
}

func (s *Server) handleGetCalendar(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	calendar := s.TournamentManager.GetCalendar()
	s.worldMu.RUnlock()
	writeJSON(w, calendar)
}

func (s *Server) handleGetFixtures(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	mw := s.TournamentManager.CurrentMatchweek
	if mwStr := r.URL.Query().Get("matchweek"); mwStr != "" {
		if parsed, err := strconv.Atoi(mwStr); err == nil {
			mw = parsed
		}
	}
	slate := s.TournamentManager.GetSlate(mw)
	useSummary := r.URL.Query().Get("summary") == "1"
	serialized := make([]map[string]interface{}, 0, len(slate))
	for i := range slate {
		if useSummary {
			serialized = append(serialized, s.serializeFixtureSummary(&slate[i]))
		} else {
			serialized = append(serialized, s.serializeFixture(&slate[i]))
		}
	}
	payload := map[string]interface{}{
		"current_matchweek": s.TournamentManager.CurrentMatchweek,
		"max_matchweeks":    s.TournamentManager.MaxMatchweeks,
		"season_phase":      s.TournamentManager.SeasonPhase,
		"season_name":       s.TournamentManager.SeasonName,
		"matchweek":         mw,
		"fixtures":          serialized,
		"ucl_pending_ids":   s.TournamentManager.PendingUCL(),
	}
	s.worldMu.RUnlock()
	writeJSON(w, payload)
}

func (s *Server) handleGetFixture(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	fid := r.PathValue("fixture_id")
	var payload map[string]interface{}
	if f := s.TournamentManager.FindFixture(fid); f != nil {
		payload = s.serializeFixture(f)
	}
	s.worldMu.RUnlock()

	if payload == nil {
		http.Error(w, "Fixture not found", http.StatusNotFound)
		return
	}
	writeJSON(w, payload)
}

func (s *Server) handleGetFavourite(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	defer s.worldMu.RUnlock()
	writeJSON(w, map[string]interface{}{"favourite_club_id": s.TournamentManager.FavouriteClubID})
}

// handleSetFavourite stores an observational club filter and persists it.
func (s *Server) handleSetFavourite(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ClubID string `json:"club_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeErrorJSON(w, http.StatusBadRequest, "Invalid favourite-club request")
		return
	}
	if req.ClubID == "" {
		req.ClubID = r.URL.Query().Get("club_id")
	}
	s.worldMu.Lock()
	ok := false
	if req.ClubID != "" {
		if _, exists := s.TournamentManager.Clubs[req.ClubID]; exists {
			s.TournamentManager.FavouriteClubID = req.ClubID
			ok = true
		}
	}
	if !ok {
		s.worldMu.Unlock()
		http.Error(w, "Unknown club", http.StatusBadRequest)
		return
	}
	snap, gen := s.takeCareerSnapshotLocked()
	fav := s.TournamentManager.FavouriteClubID
	s.worldMu.Unlock()
	s.commitCareerSnapshot(snap, gen)
	writeJSON(w, map[string]interface{}{"status": "success", "favourite_club_id": fav})
}

// handleWeekWatch resolves the favourite's fixture plus same-week cup jump
// targets. No simulation occurs here.
func (s *Server) handleWeekWatch(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	defer s.worldMu.RUnlock()
	favID, fav, cups, mw := s.TournamentManager.WeekWatch()
	var favPayload interface{}
	if fav != nil {
		favPayload = s.serializeFixture(fav)
	}
	cupPayload := make([]map[string]interface{}, 0, len(cups))
	for i := range cups {
		cupPayload = append(cupPayload, s.serializeFixture(&cups[i]))
	}
	writeJSON(w, map[string]interface{}{
		"favourite_club_id": favID,
		"matchweek":         mw,
		"fixture":           favPayload,
		"same_week_cups":    cupPayload,
	})
}

func (s *Server) handleGetUCL(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	state := s.TournamentManager.GetUCLState()
	groupA, _ := state["group_a"].([]*models.Club)
	groupB, _ := state["group_b"].([]*models.Club)
	records, _ := state["records"].(map[string]*models.CompetitionRecord)
	serGroup := func(clubs []*models.Club) []map[string]interface{} {
		var out []map[string]interface{}
		for _, c := range clubs {
			rec := records[c.ClubID]
			row := s.serializeClubRecord(c, rec)
			row["cup_status"] = s.TournamentManager.UCLGroupStatus(c.ClubID)
			out = append(out, row)
		}
		return out
	}
	payload := map[string]interface{}{
		"stage":          state["stage"],
		"group_a":        serGroup(groupA),
		"group_b":        serGroup(groupB),
		"quarter_finals": s.serializeTieMap(state["quarter_finals"]),
		"semi_finals":    s.serializeTieMap(state["semi_finals"]),
		"final":          s.serializeFinal(state["final"]),
		"champion":       s.serializeClubPtr(state["champion"]),
	}
	s.worldMu.RUnlock()
	writeJSON(w, payload)
}

func (s *Server) handleGetSuperCup(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	state := s.TournamentManager.GetSuperCupState()
	var byes []map[string]interface{}
	if raw, ok := state["byes"].([]*models.Club); ok {
		for _, c := range raw {
			byes = append(byes, s.serializeClub(c))
		}
	}
	payload := map[string]interface{}{
		"stage":          state["stage"],
		"byes":           byes,
		"play_in":        s.serializeTieMap(state["play_in"]),
		"quarter_finals": s.serializeTieMap(state["quarter_finals"]),
		"semi_finals":    s.serializeTieMap(state["semi_finals"]),
		"final":          s.serializeFinal(state["final"]),
		"champion":       s.serializeClubPtr(state["champion"]),
	}
	s.worldMu.RUnlock()
	writeJSON(w, payload)
}

func (s *Server) handleGetUCLFixtures(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	fx := s.TournamentManager.UCLFixturesCopy()
	out := make([]map[string]interface{}, 0, len(fx))
	for i := range fx {
		// This endpoint feeds the UCL calendar and cards. Full previews are
		// generated only after the viewer opens a specific scheduled fixture.
		out = append(out, s.serializeFixtureSummary(&fx[i]))
	}
	s.worldMu.RUnlock()
	writeJSON(w, out)
}

func (s *Server) serializeClubPtr(v interface{}) map[string]interface{} {
	c, _ := v.(*models.Club)
	return s.serializeClub(c)
}

func (s *Server) serializeTieMap(v interface{}) map[string]interface{} {
	raw, _ := v.(map[string]interface{})
	out := map[string]interface{}{}
	for k, item := range raw {
		m, _ := item.(map[string]interface{})
		out[k] = map[string]interface{}{
			"home":       s.serializeClubPtr(m["home"]),
			"away":       s.serializeClubPtr(m["away"]),
			"leg1":       m["leg1"],
			"leg2":       m["leg2"],
			"winner":     s.serializeClubPtr(m["winner"]),
			"decided_by": m["decided_by"],
			"penalties":  m["penalties"],
		}
	}
	return out
}

func (s *Server) serializeFinal(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	m, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}
	return map[string]interface{}{
		"team1":      s.serializeClubPtr(m["team1"]),
		"team2":      s.serializeClubPtr(m["team2"]),
		"score":      m["score"],
		"winner":     s.serializeClubPtr(m["winner"]),
		"decided_by": m["decided_by"],
		"penalties":  m["penalties"],
	}
}
