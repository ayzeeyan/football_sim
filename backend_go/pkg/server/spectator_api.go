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
	WindowOpen   bool
	Processed    int
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
		Season:    s.TournamentManager.SeasonName,
		Phase:     s.TournamentManager.SeasonPhase,
		Matchweek: s.TournamentManager.CurrentMatchweek,
		TransferWeek: func() int {
			if s.TransferEngine == nil {
				return 0
			}
			return s.TransferEngine.CurrentWeek
		}(),
		WindowOpen: func() bool {
			return s.TransferEngine != nil && s.TransferEngine.IsWindowOpen()
		}(),
		Processed: func() int {
			if s.TransferEngine == nil {
				return 0
			}
			return s.TransferEngine.ProcessedWeeks
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
			out.Message = "Season complete. The awards ceremony is ready. Transfer Week 1 must be processed before the next season."
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

	// The tournament changing phase is the authority for opening the offseason.
	// Initialize the transfer domain exactly once on the first transfer-phase
	// action: Week 1, fresh eligibility markers, and club-owned window budgets.
	// Subsequent Sim Week/Month calls must never reset that state.
	if !s.TransferEngine.IsOffSeason {
		// Apply the completed campaign before budgets are derived from club
		// reputation. The tournament helper verifies that all league and cup
		// inputs are available and is idempotent for repeated phase actions.
		if !tm.ApplyCompletedSeasonReputation() && tm.ReputationAppliedSeason != tm.SeasonName {
			out.Status = "error"
			out.Message = "The completed season is not ready for the transfer window; finish all league and cup results first."
			return out, http.StatusConflict
		}
		s.TransferEngine.BeginOffSeasonWindow()
	}
	advance := 1
	switch mode {
	case "month":
		advance = 4
		// Do not let a four-week macro skip past the final open deadline
		// day. From Week 10, for example, it processes Weeks 10–11 and
		// leaves Week 12 open for inspection; a later click processes that
		// final week and performs the season transition.
		if current := s.TransferEngine.CurrentWeek; current < s.TransferEngine.WindowWeeks() {
			remainingBeforeDeadline := s.TransferEngine.WindowWeeks() - current
			if advance > remainingBeforeDeadline {
				advance = remainingBeforeDeadline
			}
		}
	case "season":
		advance = s.TransferEngine.WindowWeeks() - s.TransferEngine.ProcessedWeeks
	}
	if advance < 0 {
		advance = 0
	}

	for i := 0; i < advance && s.TransferEngine.IsWindowOpen(); i++ {
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
	if !s.TransferEngine.IsWindowOpen() {
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
		out.Message = "All 12 transfer weeks were processed. A new season has started."
		s.clearLiveFixtureSelection()
		s.lastCommittedLiveInstance = -1
		return out, http.StatusOK
	}
	out.SeasonName = tm.SeasonName
	out.SeasonPhase = tm.SeasonPhase
	out.CurrentMatchweek = tm.CurrentMatchweek
	out.Message = fmt.Sprintf("Transfer window advanced to week %d of %d.", s.TransferEngine.CurrentWeek, s.TransferEngine.WindowWeeks())
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

func (s *Server) handleGetWorldDashboard(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	payload := s.TournamentManager.WorldDashboard()
	s.worldMu.RUnlock()
	writeJSON(w, payload)
}

type continueResult struct {
	tournament.BatchSimResult
	StopReason      string                 `json:"stop_reason"`
	ContinueHint    string                 `json:"continue_hint,omitempty"`
	NextFixture     map[string]interface{} `json:"next_fixture,omitempty"`
	FavouriteClubID string                 `json:"favourite_club_id,omitempty"`
}

func (s *Server) handleSimContinue(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	defer func() { recordMacroSimulationDuration(s, time.Since(started)) }()

	s.worldMu.Lock()
	if msg, statusCode := s.validateMacroSimAllowedLocked(); statusCode != 0 {
		s.worldMu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": msg})
		return
	}

	tm := s.TournamentManager
	if tm.SeasonPhase == "season" {
		favID, fav, _, _ := tm.WeekWatch()
		if fav != nil && fav.Status == "scheduled" {
			res := tm.SimulateRemainingExcluding(fav.FixtureID)
			played, _ := res["played"].(int)
			skipped, _ := res["skipped"].(int)
			hint := "Your club is ready to play. Watch live or simulate the fixture."
			out := continueResult{
				BatchSimResult: tournament.BatchSimResult{
					Status:           "success",
					Mode:             "continue",
					SeasonName:       tm.SeasonName,
					SeasonPhase:      tm.SeasonPhase,
					CurrentMatchweek: tm.CurrentMatchweek,
					Played:           played,
					Skipped:          skipped,
					Digests:          []tournament.MatchweekDigest{},
					Message:          hint,
					CalendarLabel:    tm.SeasonName,
				},
				StopReason:      "watched_club_match",
				ContinueHint:    hint,
				NextFixture:     s.serializeFixture(fav),
				FavouriteClubID: favID,
			}
			snap, gen := s.takeCareerSnapshotLocked()
			s.worldMu.Unlock()
			s.commitCareerSnapshot(snap, gen)
			writeJSON(w, out)
			return
		}
	}

	batch, statusCode := s.runMacroSimulationLocked("week")
	batch.Mode = "continue"
	stop := "week"
	hint := batch.Message
	if batch.AwardsReady {
		stop = "season_event"
		if hint == "" {
			hint = "Season complete. The awards ceremony is ready."
		}
	} else if tm.SeasonPhase == "transfer_window" && s.TransferEngine != nil {
		if !s.TransferEngine.IsWindowOpen() {
			stop = "transfer_deadline"
			if hint == "" {
				hint = "The transfer window has closed."
			}
		} else if s.TransferEngine.CurrentWeek >= s.TransferEngine.WindowWeeks() {
			stop = "transfer_deadline"
			if hint == "" {
				hint = "Deadline day has arrived."
			}
		} else {
			stop = "transfer_week"
			if hint == "" {
				hint = fmt.Sprintf("Transfer week %d of %d processed.", s.TransferEngine.CurrentWeek, s.TransferEngine.WindowWeeks())
			}
		}
	} else if hint == "" {
		hint = "Advanced to the next matchweek."
	}
	out := continueResult{
		BatchSimResult:  batch,
		StopReason:      stop,
		ContinueHint:    hint,
		FavouriteClubID: tm.FavouriteClubID,
	}
	if statusCode == http.StatusOK {
		snap, gen := s.takeCareerSnapshotLocked()
		s.worldMu.Unlock()
		s.commitCareerSnapshot(snap, gen)
		writeJSON(w, out)
		return
	}
	s.worldMu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": batch.Message})
}
