package server

import (
	"net/http"
)

// International fixture delivery: full pre/post-match payloads for the
// European Nations Cup, plus on-demand simulation from the Match Centre.

func (s *Server) handleGetNationsFixture(w http.ResponseWriter, r *http.Request) {
	fixtureID := r.PathValue("fixture_id")
	s.worldMu.RLock()
	payload := s.TournamentManager.NationsFixturePayload(fixtureID)
	s.worldMu.RUnlock()
	if payload == nil {
		http.Error(w, "International fixture not found", http.StatusNotFound)
		return
	}
	writeJSON(w, payload)
}

func (s *Server) handleSimulateNationsFixture(w http.ResponseWriter, r *http.Request) {
	fixtureID := r.PathValue("fixture_id")
	s.worldMu.Lock()
	held := true
	defer func() {
		if held {
			s.worldMu.Unlock()
		}
	}()
	payload := s.TournamentManager.SimulateNationsFixture(fixtureID)
	if payload == nil {
		http.Error(w, "International fixture not found", http.StatusNotFound)
		return
	}
	snap, gen := s.takeCareerSnapshotLocked()
	s.worldMu.Unlock()
	held = false
	s.commitCareerSnapshot(snap, gen)
	writeJSON(w, payload)
}
