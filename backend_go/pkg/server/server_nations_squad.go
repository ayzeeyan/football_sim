package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// Tier B national team control (B6): the viewer picks one nation's squad
// for the Nations Cup. Eligibility is enforced by the domain — only players
// whose original club country matches the nation are callable — and the
// AI selection stays the default.

// handleGetNationsSquad returns one nation's squad, the eligible pool, and
// whether the squad is viewer-selected.
func (s *Server) handleGetNationsSquad(w http.ResponseWriter, r *http.Request) {
	teamID := r.PathValue("team_id")
	s.worldMu.RLock()
	payload := s.TournamentManager.NationalSquadSelection(teamID)
	s.worldMu.RUnlock()
	if payload == nil {
		http.Error(w, "National team not found", http.StatusNotFound)
		return
	}
	writeJSON(w, payload)
}

// handleSetNationsSquad installs a viewer-selected squad.
func (s *Server) handleSetNationsSquad(w http.ResponseWriter, r *http.Request) {
	teamID := r.PathValue("team_id")
	var req struct {
		PlayerIDs []string `json:"player_ids"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeErrorJSON(w, http.StatusBadRequest, "Invalid squad request")
		return
	}

	s.worldMu.Lock()
	held := true
	defer func() {
		if held {
			s.worldMu.Unlock()
		}
	}()
	msg, team := s.TournamentManager.SetNationalSquad(teamID, req.PlayerIDs)
	if msg != "" {
		s.worldMu.Unlock()
		held = false
		writeErrorJSON(w, http.StatusBadRequest, msg)
		return
	}
	snap, gen := s.takeCareerSnapshotLocked()
	s.worldMu.Unlock()
	held = false
	s.commitCareerSnapshot(snap, gen)
	writeJSON(w, map[string]interface{}{"status": "success", "team": team})
}

// handleClearNationsSquad returns a nation to the AI selection.
func (s *Server) handleClearNationsSquad(w http.ResponseWriter, r *http.Request) {
	teamID := r.PathValue("team_id")
	s.worldMu.Lock()
	held := true
	defer func() {
		if held {
			s.worldMu.Unlock()
		}
	}()
	msg, team := s.TournamentManager.ClearNationalSquad(teamID)
	if msg != "" {
		s.worldMu.Unlock()
		held = false
		writeErrorJSON(w, http.StatusBadRequest, msg)
		return
	}
	snap, gen := s.takeCareerSnapshotLocked()
	s.worldMu.Unlock()
	held = false
	s.commitCareerSnapshot(snap, gen)
	writeJSON(w, map[string]interface{}{"status": "success", "team": team})
}
