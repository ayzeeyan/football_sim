package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// Tier B transfer control (B3): viewer offers and negotiation responses
// through the five-stage FSM.

// handleTransferOffer opens a negotiation with the viewer's opening offer.
func (s *Server) handleTransferOffer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlayerID string `json:"player_id"`
		BuyerID  string `json:"buyer_id"`
		Amount   int64  `json:"amount"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeErrorJSON(w, http.StatusBadRequest, "Invalid offer request")
		return
	}

	s.worldMu.Lock()
	held := true
	defer func() {
		if held {
			s.worldMu.Unlock()
		}
	}()
	if !s.careerWindowOpen() {
		s.worldMu.Unlock()
		held = false
		writeErrorJSON(w, http.StatusBadRequest, "The window opens when the season ends.")
		return
	}
	neg, reason := s.TransferEngine.ViewerOffer(req.PlayerID, req.BuyerID, req.Amount)
	if neg == nil {
		s.worldMu.Unlock()
		held = false
		writeErrorJSON(w, http.StatusBadRequest, reason)
		return
	}
	payload := s.serializeNegotiation(neg)
	snap, gen := s.takeCareerSnapshotLocked()
	s.worldMu.Unlock()
	held = false
	s.commitCareerSnapshot(snap, gen)
	writeJSON(w, payload)
}

// handleNegotiationRespond improves or withdraws an active negotiation.
func (s *Server) handleNegotiationRespond(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action string `json:"action"`
		Amount int64  `json:"amount"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeErrorJSON(w, http.StatusBadRequest, "Invalid response request")
		return
	}

	s.worldMu.Lock()
	held := true
	defer func() {
		if held {
			s.worldMu.Unlock()
		}
	}()
	neg, reason := s.TransferEngine.ViewerRespond(r.PathValue("negotiation_id"), req.Action, req.Amount)
	if neg == nil {
		s.worldMu.Unlock()
		held = false
		writeErrorJSON(w, http.StatusNotFound, reason)
		return
	}
	if reason != "" {
		payload := s.serializeNegotiation(neg)
		payload["status"] = "error"
		payload["message"] = reason
		snap, gen := s.takeCareerSnapshotLocked()
		s.worldMu.Unlock()
		held = false
		s.commitCareerSnapshot(snap, gen)
		writeJSON(w, payload)
		return
	}
	payload := s.serializeNegotiation(neg)
	payload["status"] = "success"
	snap, gen := s.takeCareerSnapshotLocked()
	s.worldMu.Unlock()
	held = false
	s.commitCareerSnapshot(snap, gen)
	writeJSON(w, payload)
}
