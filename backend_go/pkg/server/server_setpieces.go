package server

import (
	"net/http"

	"football_sim/pkg/matchreport"
)

// handleGetClubSetPieces serves the read-only set-piece briefing for a club's
// probable XI: who the engine picks for penalties, free kicks, corner
// delivery, and aerial targets, and why. Inspection only — the engine keeps
// its own RNG-driven picks during play.
func (s *Server) handleGetClubSetPieces(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	defer s.worldMu.RUnlock()
	cid := r.PathValue("club_id")
	club := s.TournamentManager.Clubs[cid]
	if club == nil {
		http.Error(w, "Club not found", http.StatusNotFound)
		return
	}
	style, focus := "", ""
	if mgr := s.TournamentManager.Managers[cid]; mgr != nil {
		style, focus = mgr.Style, mgr.Focus
	}
	xi := club.GetStartingElevenWithBias(style, focus, s.clubFixtureContext(cid))
	writeJSON(w, map[string]interface{}{
		"club_id":    cid,
		"club_name":  club.ClubName,
		"short_name": club.ShortName,
		"set_pieces": matchreport.InspectSetPieces(xi, s.GrowthEngine),
	})
}
