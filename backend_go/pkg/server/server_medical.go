package server

import (
	"net/http"
	"sort"

	"football_sim/pkg/medical"
	"football_sim/pkg/models"
)

// handleGetClubMedical serves the club medical view: current injuries with
// rehab roadmaps, the squad's risk assessments, and the season's injury
// history summary. Read-only — the medical staff report, nobody is treated
// by decree.
func (s *Server) handleGetClubMedical(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	defer s.worldMu.RUnlock()
	cid := r.PathValue("club_id")
	club := s.TournamentManager.Clubs[cid]
	if club == nil {
		http.Error(w, "Club not found", http.StatusNotFound)
		return
	}
	season := s.TournamentManager.SeasonName
	matchweek := s.TournamentManager.CurrentMatchweek
	highPress := false
	if mgr := s.TournamentManager.Managers[cid]; mgr != nil && mgr.CanonicalStyle() == "high_press" {
		highPress = true
	}

	type injuredRow struct {
		PlayerID   string            `json:"player_id"`
		FullName   string            `json:"full_name"`
		Position   string            `json:"position"`
		OVR        int               `json:"ovr"`
		Kind       string            `json:"kind"`
		MatchesOut int               `json:"matches_out"`
		Rehab      medical.RehabPlan `json:"rehab"`
	}
	type riskRow struct {
		PlayerID   string             `json:"player_id"`
		FullName   string             `json:"full_name"`
		Position   string             `json:"position"`
		OVR        int                `json:"ovr"`
		Fitness    int                `json:"fitness"`
		Assessment medical.Assessment `json:"assessment"`
	}

	injured := []injuredRow{}
	risks := []riskRow{}
	var allHistory []medical.Record
	for _, p := range club.Squad {
		if p == nil {
			continue
		}
		allHistory = append(allHistory, p.InjuryHistory...)
		if p.InjuredMatches > 0 {
			injured = append(injured, injuredRow{
				PlayerID: p.PlayerID, FullName: p.FullName, Position: p.Position, OVR: p.OVR,
				Kind: p.Injury, MatchesOut: p.InjuredMatches,
				Rehab: medical.RehabPlanFor(medical.Injury{
					Severity: severityForHistory(p, p.Injury), Kind: p.Injury, MatchesOut: p.InjuredMatches,
				}),
			})
			continue
		}
		density := 0.0
		if matchweek > 0 {
			density = float64(p.Appearances) / float64(matchweek)
		}
		risks = append(risks, riskRow{
			PlayerID: p.PlayerID, FullName: p.FullName, Position: p.Position, OVR: p.OVR, Fitness: p.Fitness,
			Assessment: medical.Assess(medical.RiskInput{
				Age: p.Age, Fitness: p.Fitness, MatchDensity: density,
				IsWonderkid: p.UniverseWonderkid, HighPress: highPress,
			}),
		})
	}

	// Deterministic ordering: injured by matches out descending then id;
	// risks by score descending then id.
	sort.SliceStable(injured, func(i, j int) bool {
		if injured[i].MatchesOut != injured[j].MatchesOut {
			return injured[i].MatchesOut > injured[j].MatchesOut
		}
		return injured[i].PlayerID < injured[j].PlayerID
	})
	sort.SliceStable(risks, func(i, j int) bool {
		if risks[i].Assessment.RiskScore != risks[j].Assessment.RiskScore {
			return risks[i].Assessment.RiskScore > risks[j].Assessment.RiskScore
		}
		return risks[i].PlayerID < risks[j].PlayerID
	})
	if len(risks) > 8 {
		risks = risks[:8]
	}

	writeJSON(w, map[string]interface{}{
		"club_id":   cid,
		"club_name": club.ClubName,
		"season":    season,
		"injured":   injured,
		"top_risks": risks,
		"history":   medical.SummarizeHistory(season, allHistory),
	})
}

// severityForHistory recovers the severity tier of a current injury from the
// player's ledger, falling back to the layoff-length tiers.
func severityForHistory(p *models.Player, kind string) string {
	for i := len(p.InjuryHistory) - 1; i >= 0; i-- {
		if p.InjuryHistory[i].Kind == kind {
			return p.InjuryHistory[i].Severity
		}
	}
	switch {
	case p.InjuredMatches > 10:
		return medical.SeveritySerious
	case p.InjuredMatches > 3:
		return medical.SeverityModerate
	default:
		return medical.SeverityMinor
	}
}
