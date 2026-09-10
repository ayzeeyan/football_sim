package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"football_sim/pkg/tournament"
)

const liveFixtureConflictMessage = "Finish the currently selected live fixture before simulating the universe."

// progressSignature captures the minimum logical state required to prove that
// a macro-simulation iteration actually advanced the universe.
type progressSignature struct {
	Season       string
	Phase        string
	Matchweek    int
	TransferWeek int
	Completed    int
}

// validateMacroSimAllowed checks backend invariants before macro simulation advances state.
// Returns an error message and whether a conflict (HTTP 409) or server error occurred.
// Caller MUST hold worldMu.
func (s *Server) validateMacroSimAllowedLocked() (string, int) {
	e := s.LiveMatchEngine
	if e == nil {
		return "", 0
	}

	// Active live engine (non-terminal) with a selected fixture.
	if s.liveFixtureID != "" && e.State != "NOT_STARTED" && e.State != "FULL_TIME" {
		return liveFixtureConflictMessage, http.StatusConflict
	}

	// FULL_TIME reached but engine instance not yet committed to tournament state.
	if s.liveFixtureID != "" && e.State == "FULL_TIME" && e.InstanceID != s.lastCommittedLiveInstance {
		return liveFixtureConflictMessage, http.StatusConflict
	}

	return "", 0
}

func (s *Server) macroProgressSignatureLocked() progressSignature {
	completed := 0
	if s.TransferEngine != nil {
		completed = len(s.TransferEngine.CompletedTransfers)
	}
	return progressSignature{
		Season:       s.TournamentManager.SeasonName,
		Phase:        s.TournamentManager.SeasonPhase,
		Matchweek:    s.TournamentManager.CurrentMatchweek,
		TransferWeek: func() int {
			if s.TransferEngine == nil {
				return 0
			}
			return s.TransferEngine.CurrentWeek
		}(),
		Completed: completed,
	}
}

func (s *Server) runMacroSimulationLocked(mode string) (tournament.BatchSimResult, int) {
	if msg, statusCode := s.validateMacroSimAllowedLocked(); statusCode != 0 {
		return tournament.BatchSimResult{
			Status:  "error",
			Mode:    mode,
			Message: msg,
		}, statusCode
	}

	tm := s.TournamentManager
	out := tournament.BatchSimResult{
		Status: "success", Mode: mode, SeasonName: tm.SeasonName,
		SeasonPhase: tm.SeasonPhase, CurrentMatchweek: tm.CurrentMatchweek,
		Digests: []tournament.MatchweekDigest{},
	}

	if tm.SeasonPhase == "season" {
		count := 1
		switch mode {
		case "month":
			count = 4
		case "season":
			count = tm.MaxMatchweeks - tm.CurrentMatchweek + 1
			if count < 1 {
				count = 1
			}
		}
		before := s.macroProgressSignatureLocked()
		out = tm.SimulateBatchWeeks(count)
		out.Mode = mode
		if out.Status == "error" {
			return out, http.StatusInternalServerError
		}
		after := s.macroProgressSignatureLocked()
		if !out.SeasonFinished && before == after {
			out.Status = "error"
			out.Message = "Macro simulation made no progress"
			return out, http.StatusInternalServerError
		}
		if out.AwardsReady {
			out.Message = "Season complete. The awards ceremony is ready."
		}
		s.clearLiveFixtureSelection()
		s.lastCommittedLiveInstance = -1
		return out, http.StatusOK
	}

	if tm.SeasonPhase != "transfer_window" {
		out.Status = "error"
		out.Message = fmt.Sprintf("Unknown or unhandled season phase: %s", tm.SeasonPhase)
		return out, http.StatusInternalServerError
	}

	if s.TransferEngine == nil {
		out.Status = "error"
		out.Message = "Transfer window state is unavailable."
		return out, http.StatusInternalServerError
	}
	if s.TransferEngine.CurrentWeek < 1 {
		s.TransferEngine.CurrentWeek = 1
	}
	advance := 1
	switch mode {
	case "month":
		advance = 4
	case "season":
		advance = 13 - s.TransferEngine.CurrentWeek
	}
	if remaining := 13 - s.TransferEngine.CurrentWeek; advance > remaining {
		advance = remaining
	}
	if advance < 0 {
		advance = 0
	}

	for i := 0; i < advance && s.TransferEngine.CurrentWeek <= 12; i++ {
		beforeSig := s.macroProgressSignatureLocked()
		beforeTransfers := len(s.TransferEngine.CompletedTransfers)
		s.TransferEngine.AdvanceOpenWindow()
		afterSig := s.macroProgressSignatureLocked()
		if beforeSig == afterSig {
			out.Status = "error"
			out.Message = "Macro simulation made no progress during transfer-window advancement"
			return out, http.StatusInternalServerError
		}
		out.WeeksAdvanced++
		if beforeTransfers < len(s.TransferEngine.CompletedTransfers) {
			for _, tr := range s.TransferEngine.CompletedTransfers[beforeTransfers:] {
				tm.NoteMentorDeparture(tr.PlayerID, tr.PlayerName, tr.SellerID, tr.BuyerName, tm.CurrentMatchweek)
			}
			tournament.PairSeniorMentors(tm.ClubsList, s.GrowthEngine)
		}
	}
	if s.TransferEngine.CurrentWeek > 12 {
		transition := tm.FinalizeSeasonTransition()
		if transition["status"] != "success" {
			out.Status = "error"
			if msg, ok := transition["message"].(string); ok && msg != "" {
				out.Message = msg
			} else {
				out.Message = "Season transition failed."
			}
			return out, http.StatusInternalServerError
		}
		out.NewSeasonStarted = true
		out.SeasonName = tm.SeasonName
		out.SeasonPhase = tm.SeasonPhase
		out.CurrentMatchweek = tm.CurrentMatchweek
		out.Message = "Off-season complete. A new season has started."
		s.clearLiveFixtureSelection()
		s.lastCommittedLiveInstance = -1
		return out, http.StatusOK
	}
	out.SeasonName = tm.SeasonName
	out.SeasonPhase = tm.SeasonPhase
	out.CurrentMatchweek = tm.CurrentMatchweek
	out.Message = fmt.Sprintf("Transfer window advanced to week %d of 12.", s.TransferEngine.CurrentWeek)
	s.clearLiveFixtureSelection()
	s.lastCommittedLiveInstance = -1
	return out, http.StatusOK
}

func (s *Server) handleMacroSimulation(w http.ResponseWriter, mode string) {
	started := time.Now()
	defer func() { recordMacroSimulationDuration(s, time.Since(started)) }()

	s.worldMu.Lock()
	out, statusCode := s.runMacroSimulationLocked(mode)
	if statusCode == http.StatusOK {
		snap, gen := s.takeCareerSnapshotLocked()
		s.worldMu.Unlock()
		s.commitCareerSnapshot(snap, gen)
	} else {
		s.worldMu.Unlock()
	}

	if statusCode != http.StatusOK {
		if statusCode >= http.StatusInternalServerError {
			log.Printf("[MacroSim] mode=%s internal failure: %s", mode, out.Message)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": out.Message})
		return
	}
	writeJSON(w, out)
}

func (s *Server) handleSimWeek(w http.ResponseWriter, r *http.Request) {
	s.handleMacroSimulation(w, "week")
}

func (s *Server) handleSimMonth(w http.ResponseWriter, r *http.Request) {
	s.handleMacroSimulation(w, "month")
}

func (s *Server) handleSimSeason(w http.ResponseWriter, r *http.Request) {
	s.handleMacroSimulation(w, "season")
}
