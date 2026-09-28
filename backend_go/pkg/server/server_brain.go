package server

import (
	"net/http"

	"football_sim/pkg/brain"
	"football_sim/pkg/managers"
)

// handleGetBrain exposes the world brain's learned state: the fitted
// weights, training volume, running loss, and the learned-vs-fixed tactical
// edges for every philosophical pairing. Observational — the viewer never
// trains or steers the brain.
func (s *Server) handleGetBrain(w http.ResponseWriter, r *http.Request) {
	state := s.TournamentManager.BrainState()
	if state == nil {
		writeJSON(w, map[string]interface{}{"brain": nil})
		return
	}
	weights := make([]map[string]interface{}, 0, brain.NumFeatures)
	labels := brain.FeatureLabels()
	for i, label := range labels {
		weights = append(weights, map[string]interface{}{
			"feature": label,
			"weight":  state.Weights[i],
		})
	}
	styles := []string{"high_press", "possession", "low_block", "free_flowing"}
	edges := make([]map[string]interface{}, 0, 0)
	for _, home := range styles {
		for _, away := range styles {
			if home == away {
				continue
			}
			learned := s.TournamentManager.BrainTacticEdge(home, away)
			edges = append(edges, map[string]interface{}{
				"home_style": home,
				"away_style": away,
				"fixed_edge": managers.TacticEdge(home, away),
				"edge":       learned,
			})
		}
	}
	writeJSON(w, map[string]interface{}{
		"brain": map[string]interface{}{
			"weights":            weights,
			"bias":               state.Bias,
			"samples":            state.Samples,
			"loss_ema":           state.LossEMA,
			"tactical_edges":     edges,
			"edge_blend_samples": 400,
		},
	})
}
