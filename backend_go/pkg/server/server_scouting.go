package server

import (
	"net/http"
	"strconv"
)

// handleGetClubScouting serves the AI recruitment shortlist for one club.
// Purely observational: the scouting desk reports on the world, it never
// acts for the club.
func (s *Server) handleGetClubScouting(w http.ResponseWriter, r *http.Request) {
	clubID := r.PathValue("club_id")
	limit := 12
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 50 {
			limit = parsed
		}
	}
	s.worldMu.RLock()
	shortlist := s.TournamentManager.ScoutingShortlist(clubID, limit)
	s.worldMu.RUnlock()

	if shortlist == nil {
		http.Error(w, "Club not found", http.StatusNotFound)
		return
	}
	writeJSON(w, map[string]interface{}{
		"club_id":   clubID,
		"shortlist": shortlist,
	})
}
