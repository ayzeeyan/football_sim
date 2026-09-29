package server

import (
	"football_sim/pkg/footballai"
	"football_sim/pkg/models"
)

// FootballMoE integration: purely observational outcome recording around
// single-fixture simulations. Nothing in this file can change a simulation
// result — it captures pre-match state, attaches the actual outcome
// afterwards, and hands both to the recorder for future offline training.
// The runtime never trains; weights only change via cmd/train.

// recordMatchObservation captures the pre-match state of one fixture.
// Returns 0 when recording is disabled or the fixture is not pending.
func (s *Server) recordMatchObservation(fixtureID string) uint64 {
	rec := s.AIBrain().Recorder()
	if rec == nil {
		return 0
	}
	fix := s.TournamentManager.FindFixture(fixtureID)
	if fix == nil || fix.Status == "finished" || fix.Home == nil || fix.Away == nil {
		return 0
	}
	// Stamp the simulation clock so recorded rows carry their matchweek.
	rec.SetClock(0, 0, s.TournamentManager.CurrentMatchweek)
	req := footballai.MatchRequest{
		Match: footballai.MatchContext{
			HomeRating:    float32(fix.Home.OverallTeamRating),
			AwayRating:    float32(fix.Away.OverallTeamRating),
			HomeForm:      clubFormModifier(fix.Home),
			AwayForm:      clubFormModifier(fix.Away),
			HomeFitness:   squadAvgFitness(fix.Home),
			AwayFitness:   squadAvgFitness(fix.Away),
			HomeAbsences:  squadInjuries(fix.Home),
			AwayAbsences:  squadInjuries(fix.Away),
			EuropeanNight: fix.Competition == "ucl",
		},
	}
	return rec.Observe(footballai.TaskMatchPrediction, req.Encode())
}

// completeMatchObservation attaches the actual scoreline to a pending
// observation. It is only called after a successful simulation; unknown IDs
// are ignored inside the recorder.
func (s *Server) completeMatchObservation(id uint64, res map[string]interface{}) {
	if id == 0 {
		return
	}
	rec := s.AIBrain().Recorder()
	if rec == nil {
		return
	}
	home := toFloat(res["home_goals"])
	away := toFloat(res["away_goals"])
	rec.Complete(id, map[string]float64{
		"home_goals": home,
		"away_goals": away,
		"goal_diff":  home - away,
	})
}

// clubFormModifier converts the club's recent-results ring into the signed
// modifier used by the match feature group (W +3, D 0, L -3).
func clubFormModifier(c *models.Club) float32 {
	if c == nil {
		return 0
	}
	mod := float32(0)
	for _, r := range c.Form {
		switch r {
		case "W":
			mod += 3
		case "L":
			mod -= 3
		}
	}
	return mod
}

// squadAvgFitness averages squad fitness (0-100). Returns 0 for empty squads
// so the slot stays unfilled rather than lying with a fabricated value.
func squadAvgFitness(c *models.Club) float32 {
	if c == nil || len(c.Squad) == 0 {
		return 0
	}
	sum := 0
	for _, p := range c.Squad {
		sum += p.Fitness
	}
	return float32(sum) / float32(len(c.Squad))
}

// squadInjuries counts currently injured players.
func squadInjuries(c *models.Club) int {
	if c == nil {
		return 0
	}
	n := 0
	for _, p := range c.Squad {
		if p != nil && p.Injury != "" {
			n++
		}
	}
	return n
}

// toFloat coerces a JSON-ish number.
func toFloat(v interface{}) float64 {
	switch n := v.(type) {
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case float64:
		return n
	case float32:
		return float64(n)
	}
	return 0
}
