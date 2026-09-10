package server

import (
	"football_sim/pkg/persistence"
	"football_sim/pkg/tournament"
)

// DiagnosticsSnapshot is a compact developer-facing view of the mutable
// universe. It intentionally excludes host, filesystem, and other system data.
type DiagnosticsSnapshot struct {
	SaveVersion               int    `json:"save_version"`
	SeasonName                string `json:"season_name"`
	SeasonPhase               string `json:"season_phase"`
	CurrentMatchweek          int    `json:"current_matchweek"`
	TransferWeek              int    `json:"transfer_week"`
	FixturesRemaining         int    `json:"fixtures_remaining"`
	CompletedFixtureCount     int    `json:"completed_fixture_count"`
	LiveFixtureID             string `json:"live_fixture_id,omitempty"`
	LiveEngineState           string `json:"live_engine_state,omitempty"`
	LiveEngineInstanceID      int    `json:"live_engine_instance_id"`
	LastCommittedLiveInstance int    `json:"last_committed_live_instance"`
	TotalClubs                int    `json:"total_clubs"`
	TotalPlayers              int    `json:"total_players"`
	TotalManagers             int    `json:"total_managers"`
}

// Diagnostics returns a consistent snapshot under the world read lock. It is
// intentionally a structure rather than a public endpoint so diagnostics can
// be consumed by tests and local tooling without expanding the public API.
func (s *Server) Diagnostics() DiagnosticsSnapshot {
	s.worldMu.RLock()
	defer s.worldMu.RUnlock()

	out := DiagnosticsSnapshot{SaveVersion: persistence.SaveVersion}
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
