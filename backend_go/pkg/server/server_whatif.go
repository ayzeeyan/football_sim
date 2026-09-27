package server

import (
	"net/http"
	"strconv"
)

// handleGetWhatIf resolves one fixture under a scratch seed without touching
// the real universe. The recorded result always stands; this is an
// observational sandbox, never a replay.
func (s *Server) handleGetWhatIf(w http.ResponseWriter, r *http.Request) {
	fid := r.PathValue("fixture_id")
	seed := int64(0)
	if raw := r.URL.Query().Get("seed"); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil {
			seed = parsed
		}
	}
	s.worldMu.RLock()
	result, errMsg := s.TournamentManager.WhatIfSandbox(fid, seed)
	s.worldMu.RUnlock()

	if result == nil {
		if errMsg == "Fixture not found." {
			http.Error(w, errMsg, http.StatusNotFound)
			return
		}
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}
	writeJSON(w, result)
}
