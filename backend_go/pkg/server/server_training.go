package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"football_sim/pkg/growth"
	"football_sim/pkg/scouting"
)

// Tier B training control (B2): any squad player can be trained, not just
// the twelve prodigies. Unregistered players get a growth profile on first
// use; their projected ceiling never reaches the wonderkid-only 99 marker.

// handleTrainPlayer runs one training regimen for any player, registering a
// growth profile on demand the first time the player is trained.
func (s *Server) handleTrainPlayer(w http.ResponseWriter, r *http.Request) {
	s.worldMu.Lock()
	held := true
	defer func() {
		if held {
			s.worldMu.Unlock()
		}
	}()

	pid := r.PathValue("player_id")
	var req struct {
		Focus string `json:"focus"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		s.worldMu.Unlock()
		held = false
		writeErrorJSON(w, http.StatusBadRequest, "Invalid training request")
		return
	}
	if req.Focus == "" {
		req.Focus = "technical"
	}
	if req.Focus != "hypertrophy" && req.Focus != "technical" && req.Focus != "tactical" {
		s.worldMu.Unlock()
		held = false
		writeErrorJSON(w, http.StatusBadRequest, "Unknown training focus")
		return
	}

	p, clubID := s.findPlayer(pid)
	if p == nil {
		s.worldMu.Unlock()
		held = false
		http.Error(w, "Player not found", http.StatusNotFound)
		return
	}

	// On-demand registration for players the growth engine does not track
	// yet. The projected ceiling stays below the wonderkid-only 99 marker.
	if _, tracked := s.GrowthEngine.PotentialFor(pid); !tracked {
		height, weight := 178.0, 73.0
		if p.Age < 19 {
			height, weight = 168.0, 58.0
		}
		ceiling := scouting.ProjectedCeiling(p.Age, p.OVR)
		if !p.UniverseWonderkid && ceiling > 98 {
			ceiling = 98
		}
		s.GrowthEngine.RegisterProdigy(pid, p.FullName, p.Age, height, weight, p.Category, p.OVR, ceiling, growth.AdultHeightAgeFor(p.FullName))
	}

	res, err := s.GrowthEngine.RunTrainingCycle(pid, req.Focus)
	if err != nil {
		s.worldMu.Unlock()
		held = false
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	p.OVR = s.GrowthEngine.EnforceSeasonOVRCap(pid, p.Category)
	res["ovr"] = p.OVR
	res["player_id"] = pid
	if clubID != nil {
		res["club_id"] = clubID.ClubID
	}
	snap, gen := s.takeCareerSnapshotLocked()
	s.worldMu.Unlock()
	held = false
	s.commitCareerSnapshot(snap, gen)
	writeJSON(w, res)
}
