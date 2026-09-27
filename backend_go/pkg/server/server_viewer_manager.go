package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"
)

// Tier B manager career (B5): the viewer can accept a club's dugout, be
// sacked by the patience system, win trophies, and resign. The world keeps
// running AI-first for every other club.

// handleGetViewerManager returns the viewer's career state, the current
// job's security, and the full list of clubs with their managers so the UI
// can offer every open dugout.
func (s *Server) handleGetViewerManager(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	vm := s.TournamentManager.GetViewerManager()
	var job map[string]interface{}
	if vm != nil && vm.ClubID != "" {
		job = s.serializeManager(s.TournamentManager.Managers[vm.ClubID])
	}
	clubs := make([]map[string]interface{}, 0, len(s.TournamentManager.Clubs))
	for _, club := range s.TournamentManager.Clubs {
		row := map[string]interface{}{
			"club_id":   club.ClubID,
			"club_name": club.ClubName,
			"league":    club.League,
		}
		if mgr := s.TournamentManager.Managers[club.ClubID]; mgr != nil {
			row["manager_name"] = mgr.Name
			row["job_security"] = mgr.JobSecurity
		}
		clubs = append(clubs, row)
	}
	s.worldMu.RUnlock()

	sort.Slice(clubs, func(i, j int) bool {
		a, _ := clubs[i]["club_id"].(string)
		b, _ := clubs[j]["club_id"].(string)
		return a < b
	})
	writeJSON(w, map[string]interface{}{
		"manager": vm,
		"job":     job,
		"clubs":   clubs,
	})
}

// handleAcceptViewerJob installs the viewer in one club's dugout.
func (s *Server) handleAcceptViewerJob(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ClubID string `json:"club_id"`
		Name   string `json:"name"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeErrorJSON(w, http.StatusBadRequest, "Invalid job request")
		return
	}
	if req.ClubID == "" {
		writeErrorJSON(w, http.StatusBadRequest, "A club_id is required.")
		return
	}

	s.worldMu.Lock()
	held := true
	defer func() {
		if held {
			s.worldMu.Unlock()
		}
	}()
	club := s.TournamentManager.Clubs[req.ClubID]
	if club == nil {
		s.worldMu.Unlock()
		held = false
		http.Error(w, "Club not found", http.StatusNotFound)
		return
	}
	vm, msg := s.TournamentManager.AcceptViewerJob(req.ClubID, req.Name)
	if msg != "" {
		s.worldMu.Unlock()
		held = false
		writeErrorJSON(w, http.StatusBadRequest, msg)
		return
	}
	payload := map[string]interface{}{
		"status":  "success",
		"manager": vm,
		"job":     s.serializeManager(s.TournamentManager.Managers[req.ClubID]),
	}
	snap, gen := s.takeCareerSnapshotLocked()
	s.worldMu.Unlock()
	held = false
	s.commitCareerSnapshot(snap, gen)
	writeJSON(w, payload)
}

// handleResignViewerJob steps down; the club appoints a deterministic
// replacement so the world keeps running.
func (s *Server) handleResignViewerJob(w http.ResponseWriter, r *http.Request) {
	s.worldMu.Lock()
	held := true
	defer func() {
		if held {
			s.worldMu.Unlock()
		}
	}()
	vm, msg := s.TournamentManager.ResignViewerJob()
	if msg != "" {
		s.worldMu.Unlock()
		held = false
		writeErrorJSON(w, http.StatusBadRequest, msg)
		return
	}
	payload := map[string]interface{}{
		"status":  "success",
		"manager": vm,
	}
	snap, gen := s.takeCareerSnapshotLocked()
	s.worldMu.Unlock()
	held = false
	s.commitCareerSnapshot(snap, gen)
	writeJSON(w, payload)
}
