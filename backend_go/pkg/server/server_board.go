package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// boardObjectives is the closed set the season-assignment logic uses; the
// viewer may pick any of them but never free text.
var boardObjectives = map[string]bool{
	"Title challenge":                true,
	"Champions League qualification": true,
	"European qualification":         true,
	"Top-half finish":                true,
	"Survival":                       true,
	"Competitive finish":             true,
}

// Tier B board & finance control (B4): the viewer sets one club's transfer
// budget, wage cap, and board expectation. The domain invariants still bind:
// a budget can never exceed the club's available balance, and a wage cap can
// never sit below the committed wage bill.
func (s *Server) handleSetClubBoard(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TransferBudget *int64  `json:"transfer_budget"`
		WageCap        *int64  `json:"wage_cap"`
		BoardObjective *string `json:"board_objective"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeErrorJSON(w, http.StatusBadRequest, "Invalid board request")
		return
	}
	if req.TransferBudget == nil && req.WageCap == nil && req.BoardObjective == nil {
		writeErrorJSON(w, http.StatusBadRequest, "Nothing to change")
		return
	}

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

	if req.TransferBudget != nil {
		if *req.TransferBudget < 0 {
			s.worldMu.Unlock()
			held = false
			writeErrorJSON(w, http.StatusBadRequest, "The transfer budget cannot be negative.")
			return
		}
		if *req.TransferBudget > club.Finances.Balance {
			s.worldMu.Unlock()
			held = false
			writeErrorJSON(w, http.StatusBadRequest, "The transfer budget cannot exceed the club's available balance.")
			return
		}
		club.Finances.TransferBudget = *req.TransferBudget
		if mgr := s.TournamentManager.Managers[club.ClubID]; mgr != nil {
			mgr.BudgetEur = *req.TransferBudget
		}
	}
	if req.WageCap != nil {
		if *req.WageCap < 0 {
			s.worldMu.Unlock()
			held = false
			writeErrorJSON(w, http.StatusBadRequest, "The wage cap cannot be negative.")
			return
		}
		if club.WageBill() > 0 && *req.WageCap < club.WageBill() {
			s.worldMu.Unlock()
			held = false
			writeErrorJSON(w, http.StatusBadRequest, "The wage cap cannot sit below the committed wage bill.")
			return
		}
		club.Finances.WageCap = *req.WageCap
	}
	if req.BoardObjective != nil {
		if !boardObjectives[*req.BoardObjective] {
			s.worldMu.Unlock()
			held = false
			writeErrorJSON(w, http.StatusBadRequest, "Unknown board objective.")
			return
		}
		club.BoardObjective = *req.BoardObjective
	}

	payload := map[string]interface{}{
		"status":          "success",
		"club_id":         club.ClubID,
		"finances":        club.Finances,
		"board_objective": club.BoardObjective,
	}
	snap, gen := s.takeCareerSnapshotLocked()
	s.worldMu.Unlock()
	held = false
	s.commitCareerSnapshot(snap, gen)
	writeJSON(w, payload)
}
