package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"football_sim/pkg/tournament"
)

// validateMacroSimAllowed checks backend invariants before macro simulation advances state.
// Returns an error message and whether a conflict (HTTP 409) or server error occurred.
// Caller MUST hold worldMu.
func (s *Server) validateMacroSimAllowedLocked() (string, int) {
	e := s.LiveMatchEngine
	if e == nil {
		return "", 0
	}

	// Active live engine (non-terminal) with a selected fixture
	if s.liveFixtureID != "" && e.State != "NOT_STARTED" && e.State != "FULL_TIME" {
		return "Finish the currently selected live fixture before simulating the universe.", http.StatusConflict
	}

	// FULL_TIME reached but engine instance not yet committed to tournament state
	if s.liveFixtureID != "" && e.State == "FULL_TIME" && e.InstanceID != s.lastCommittedLiveInstance {
		return "Finish the currently selected live fixture before simulating the universe.", http.StatusConflict
	}

	return "", 0
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
		out = tm.SimulateBatchWeeks(count)
		out.Mode = mode
		if out.Status == "error" {
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

	type offSeasonSig struct {
		SeasonName   string
		TransferWeek int
		Day          int
	}

	maxIterations := advance + 5
	iterations := 0
	for i := 0; i < advance && s.TransferEngine.CurrentWeek <= 12; i++ {
		iterations++
		if iterations > maxIterations {
			out.Status = "error"
			out.Message = "Macro simulation iteration limit exceeded without progress"
			return out, http.StatusInternalServerError
		}

		beforeSig := offSeasonSig{
			SeasonName:   tm.SeasonName,
			TransferWeek: s.TransferEngine.CurrentWeek,
			Day:          s.TransferEngine.CurrentDay,
		}

		before := len(s.TransferEngine.CompletedTransfers)
		s.TransferEngine.AdvanceOpenWindow()
		out.WeeksAdvanced++
		if before < len(s.TransferEngine.CompletedTransfers) {
			for _, tr := range s.TransferEngine.CompletedTransfers[before:] {
				tm.NoteMentorDeparture(tr.PlayerID, tr.PlayerName, tr.SellerID, tr.BuyerName, tm.CurrentMatchweek)
			}
			tournament.PairSeniorMentors(tm.ClubsList, s.GrowthEngine)
		}

		afterSig := offSeasonSig{
			SeasonName:   tm.SeasonName,
			TransferWeek: s.TransferEngine.CurrentWeek,
			Day:          s.TransferEngine.CurrentDay,
		}

		if beforeSig == afterSig {
			out.Status = "error"
			out.Message = "Transfer window macro simulation stalled without progress"
			return out, http.StatusInternalServerError
		}
	}
	if s.TransferEngine.CurrentWeek > 12 {
		tm.ResetNewSeason()
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
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  out.Status,
			"mode":    out.Mode,
			"message": out.Message,
			"detail":  out.Message,
		})
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
