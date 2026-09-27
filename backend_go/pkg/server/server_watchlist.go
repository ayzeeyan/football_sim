package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"football_sim/pkg/tournament"
)

// handleGetWatchlist returns the persisted multi-entity watchlist with
// display names resolved for the panel.
func (s *Server) handleGetWatchlist(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	defer s.worldMu.RUnlock()

	state := s.TournamentManager.Watchlist()
	clubs := make([]map[string]interface{}, 0, len(state.Clubs))
	for _, id := range state.Clubs {
		club := s.TournamentManager.Clubs[id]
		if club == nil {
			continue
		}
		clubs = append(clubs, map[string]interface{}{
			"id": club.ClubID, "name": club.ClubName, "short_name": club.ShortName,
		})
	}
	players := make([]map[string]interface{}, 0, len(state.Players))
	for _, id := range state.Players {
		p, club := s.findPlayer(id)
		if p == nil {
			continue
		}
		row := map[string]interface{}{
			"id": p.PlayerID, "name": p.FullName, "position": p.Position, "ovr": p.OVR,
		}
		if club != nil {
			row["club_short"] = club.ShortName
		}
		players = append(players, row)
	}
	competitions := make([]map[string]interface{}, 0, len(state.Competitions))
	if s.TournamentManager.World != nil {
		for _, id := range state.Competitions {
			comp := s.TournamentManager.World.Competitions[id]
			if comp == nil {
				continue
			}
			competitions = append(competitions, map[string]interface{}{
				"id": comp.ID, "name": comp.Name, "kind": comp.Kind,
			})
		}
	}
	writeJSON(w, map[string]interface{}{
		"clubs":        clubs,
		"players":      players,
		"competitions": competitions,
	})
}

// handleToggleWatchlist adds or removes one watchlist entity and persists
// the change. The watchlist is observational only.
func (s *Server) handleToggleWatchlist(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Entity string `json:"entity"`
		ID     string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeErrorJSON(w, http.StatusBadRequest, "Invalid watchlist request")
		return
	}
	if req.ID == "" {
		req.ID = r.URL.Query().Get("id")
	}
	if req.Entity == "" {
		req.Entity = r.URL.Query().Get("entity")
	}

	s.worldMu.Lock()
	state, watched, ok := s.TournamentManager.ToggleWatchlist(tournament.WatchlistEntity(req.Entity), req.ID)
	if ok {
		snap, gen := s.takeCareerSnapshotLocked()
		s.worldMu.Unlock()
		s.commitCareerSnapshot(snap, gen)
	} else {
		s.worldMu.Unlock()
		http.Error(w, "Unknown watchlist entity", http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]interface{}{
		"status":    "success",
		"entity":    req.Entity,
		"id":        req.ID,
		"watched":   watched,
		"watchlist": state,
	})
}
