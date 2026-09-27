package server

import (
	"net/http"

	"football_sim/pkg/tournament"
)

// handleGetAchievements serves the milestone ledger: the fixed catalogue with
// unlocked flags plus the unlocked entries and the youngest-scorer record.
func (s *Server) handleGetAchievements(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	unlocked := s.TournamentManager.GetAchievements()
	definitions := tournament.AchievementDefinitions()
	youngest := s.TournamentManager.GetYoungestScorerRecord()
	s.worldMu.RUnlock()

	unlockedIDs := map[string]bool{}
	for _, a := range unlocked {
		unlockedIDs[a.ID] = true
	}
	catalogue := make([]map[string]interface{}, 0, len(definitions))
	for _, def := range definitions {
		catalogue = append(catalogue, map[string]interface{}{
			"id": def.ID, "kind": def.Kind, "title": def.Title,
			"description": def.Description, "unlocked": unlockedIDs[def.ID],
		})
	}
	writeJSON(w, map[string]interface{}{
		"definitions":            catalogue,
		"achievements":           unlocked,
		"youngest_scorer_record": youngest,
	})
}
