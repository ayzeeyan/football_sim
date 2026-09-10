package server

import (
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"football_sim/pkg/persistence"
	"football_sim/pkg/tournament"
)

var macroSimulationDurations sync.Map // map[*Server]time.Duration

func recordMacroSimulationDuration(s *Server, duration time.Duration) {
	if s != nil {
		macroSimulationDurations.Store(s, duration)
	}
}

func lastMacroSimulationDuration(s *Server) time.Duration {
	if s == nil {
		return 0
	}
	value, ok := macroSimulationDurations.Load(s)
	if !ok {
		return 0
	}
	duration, _ := value.(time.Duration)
	return duration
}

func readUniverseSeedForDiagnostics(savePath string) int64 {
	if savePath == "" {
		return 0
	}
	data, err := os.ReadFile(persistence.UniverseSeedPath(savePath))
	if err != nil {
		return 0
	}
	seed, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0
	}
	return seed
}

// DiagnosticsSnapshot is a compact developer-facing view of the mutable
// universe. It intentionally excludes host, filesystem, and other system data.
type DiagnosticsSnapshot struct {
	SaveVersion                   int     `json:"save_version"`
	UniverseSeed                  int64   `json:"universe_seed"`
	SeasonName                    string  `json:"season_name"`
	SeasonPhase                   string  `json:"season_phase"`
	CurrentMatchweek              int     `json:"current_matchweek"`
	TransferWeek                  int     `json:"transfer_week"`
	FixturesRemaining             int     `json:"fixtures_remaining"`
	CompletedFixtureCount         int     `json:"completed_fixture_count"`
	LiveFixtureID                 string  `json:"live_fixture_id,omitempty"`
	LiveEngineState               string  `json:"live_engine_state,omitempty"`
	LiveEngineInstanceID          int     `json:"live_engine_instance_id"`
	LastCommittedLiveInstance     int     `json:"last_committed_live_instance"`
	TotalClubs                    int     `json:"total_clubs"`
	TotalPlayers                  int     `json:"total_players"`
	TotalManagers                 int     `json:"total_managers"`
	LastMacroSimulationDurationMS float64 `json:"last_macro_simulation_duration_ms"`
}

// Diagnostics returns a consistent snapshot under the world read lock. Seed
// file I/O is performed before acquiring the world lock so unrelated disk work
// never extends the critical section.
func (s *Server) Diagnostics() DiagnosticsSnapshot {
	seed := readUniverseSeedForDiagnostics(s.savePath)
	duration := lastMacroSimulationDuration(s)

	s.worldMu.RLock()
	defer s.worldMu.RUnlock()

	out := DiagnosticsSnapshot{
		SaveVersion:                   persistence.SaveVersion,
		UniverseSeed:                  seed,
		LastMacroSimulationDurationMS: float64(duration) / float64(time.Millisecond),
	}
	if s.TournamentManager != nil {
		tm := s.TournamentManager
		out.SeasonName = tm.SeasonName
		out.SeasonPhase = tm.SeasonPhase
		out.CurrentMatchweek = tm.CurrentMatchweek
		out.TotalClubs = len(tm.ClubsList)
		out.TotalManagers = len(tm.Managers)
		for _, club := range tm.ClubsList {
			if club != nil {
				out.TotalPlayers += len(club.Squad)
			}
		}
		for _, fixtures := range [][]tournament.Fixture{tm.Fixtures, tm.UCLFixtures, tm.SuperCupFixtures} {
			for i := range fixtures {
				if fixtures[i].Status == "finished" {
					out.CompletedFixtureCount++
				} else {
					out.FixturesRemaining++
				}
			}
		}
	}
	if s.TransferEngine != nil {
		out.TransferWeek = s.TransferEngine.CurrentWeek
	}
	out.LiveFixtureID = s.liveFixtureID
	out.LastCommittedLiveInstance = s.lastCommittedLiveInstance
	if s.LiveMatchEngine != nil {
		out.LiveEngineState = s.LiveMatchEngine.State
		out.LiveEngineInstanceID = s.LiveMatchEngine.InstanceID
	}
	return out
}
