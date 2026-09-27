package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"football_sim/pkg/models"
)

// Tier B lineup control: the viewer sets one club's formation and starting
// XI. The rigid-slot contract is enforced by models.ValidateLineupOverride
// and the effective XI is materialised through the same StartingSlot
// structure the AI selection uses.

// handleSetClubLineup stores a viewer lineup override for one club.
func (s *Server) handleSetClubLineup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Formation string            `json:"formation"`
		Players   map[string]string `json:"players"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeErrorJSON(w, http.StatusBadRequest, "Invalid lineup request")
		return
	}
	override := &models.LineupOverride{Formation: req.Formation, Players: req.Players}

	s.worldMu.Lock()
	held := true
	defer func() {
		if held {
			s.worldMu.Unlock()
		}
	}()
	club := s.TournamentManager.Clubs[r.PathValue("club_id")]
	if club == nil {
		s.worldMu.Unlock()
		held = false
		http.Error(w, "Club not found", http.StatusNotFound)
		return
	}
	if err := models.ValidateLineupOverride(club.Squad, override); err != nil {
		s.worldMu.Unlock()
		held = false
		writeErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	club.LineupOverride = override
	style, focus := "", ""
	if mgr := s.TournamentManager.Managers[club.ClubID]; mgr != nil {
		style, focus = mgr.Style, mgr.Focus
	}
	slots := club.GetStartingElevenSlotsWithBias(style, focus, s.clubFixtureContext(club.ClubID))
	snap, gen := s.takeCareerSnapshotLocked()
	s.worldMu.Unlock()
	held = false
	s.commitCareerSnapshot(snap, gen)

	serialized := make([]map[string]interface{}, 0, len(slots))
	for _, entry := range slots {
		if entry.Player == nil {
			continue
		}
		row := s.serializePlayer(entry.Player)
		row["starting_slot"] = entry.Slot
		row["position_fit"] = string(entry.PositionFit)
		serialized = append(serialized, row)
	}
	writeJSON(w, map[string]interface{}{
		"status":   "success",
		"club_id":  club.ClubID,
		"lineup":   serialized,
		"override": club.LineupOverride,
	})
}

// handleClearClubLineup returns a club to AI selection.
func (s *Server) handleClearClubLineup(w http.ResponseWriter, r *http.Request) {
	s.worldMu.Lock()
	held := true
	defer func() {
		if held {
			s.worldMu.Unlock()
		}
	}()
	club := s.TournamentManager.Clubs[r.PathValue("club_id")]
	if club == nil {
		s.worldMu.Unlock()
		held = false
		http.Error(w, "Club not found", http.StatusNotFound)
		return
	}
	club.LineupOverride = nil
	snap, gen := s.takeCareerSnapshotLocked()
	s.worldMu.Unlock()
	held = false
	s.commitCareerSnapshot(snap, gen)
	writeJSON(w, map[string]interface{}{"status": "success", "club_id": club.ClubID, "override": nil})
}
