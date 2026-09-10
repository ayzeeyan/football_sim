package server

import (
	"net/http"
	"time"
)

type DiagnosticsReport struct {
	Timestamp                 string `json:"timestamp"`
	SaveVersion               string `json:"save_version"`
	UniverseSeed              int64  `json:"universe_seed"`
	SeasonName                string `json:"season_name"`
	SeasonPhase               string `json:"season_phase"`
	CurrentMatchweek          int    `json:"current_matchweek"`
	MaxMatchweeks             int    `json:"max_matchweeks"`
	TransferCurrentWeek       int    `json:"transfer_current_week"`
	TotalClubs                int    `json:"total_clubs"`
	TotalPlayers              int    `json:"total_players"`
	TotalManagers             int    `json:"total_managers"`
	LiveFixtureID             string `json:"live_fixture_id"`
	LiveEngineState           string `json:"live_engine_state"`
	LiveEngineInstanceID      int    `json:"live_engine_instance_id"`
	LastCommittedLiveInstance int    `json:"last_committed_live_instance"`
	WorldStateValid           bool   `json:"world_state_valid"`
	ValidationError           string `json:"validation_error,omitempty"`
}

func (s *Server) handleDiagnostics(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	defer s.worldMu.RUnlock()

	tm := s.TournamentManager
	totalPlayers := 0
	if tm != nil {
		for _, c := range tm.ClubsList {
			if c != nil {
				totalPlayers += len(c.Squad)
			}
		}
	}

	liveState := ""
	liveInstanceID := 0
	if s.LiveMatchEngine != nil {
		liveState = s.LiveMatchEngine.State
		liveInstanceID = s.LiveMatchEngine.InstanceID
	}

	transferWeek := 0
	if s.TransferEngine != nil {
		transferWeek = s.TransferEngine.CurrentWeek
	}

	var valErrStr string
	valErr := tm.ValidateWorldState()
	if valErr != nil {
		valErrStr = valErr.Error()
	}

	report := DiagnosticsReport{
		Timestamp:                 time.Now().UTC().Format(time.RFC3339),
		SaveVersion:               "1.0",
		UniverseSeed:              tm.Seed,
		SeasonName:                tm.SeasonName,
		SeasonPhase:               tm.SeasonPhase,
		CurrentMatchweek:          tm.CurrentMatchweek,
		MaxMatchweeks:             tm.MaxMatchweeks,
		TransferCurrentWeek:       transferWeek,
		TotalClubs:                len(tm.ClubsList),
		TotalPlayers:              totalPlayers,
		TotalManagers:             len(tm.Managers),
		LiveFixtureID:             s.liveFixtureID,
		LiveEngineState:           liveState,
		LiveEngineInstanceID:      liveInstanceID,
		LastCommittedLiveInstance: s.lastCommittedLiveInstance,
		WorldStateValid:           valErr == nil,
		ValidationError:           valErrStr,
	}

	writeJSON(w, report)
}
