package tournament

import (
	"football_sim/pkg/footballai"
	"football_sim/pkg/managers"
	"football_sim/pkg/matchengine"
	"football_sim/pkg/models"
)

// FootballMoE bridge: the only places the learned model touches the live
// simulation. Every path here is gated by a brain feature flag, bounded by a
// corridor, and fails open (model error -> existing behavior). The rules,
// constraints, RNG, and world progression remain the simulator's own.

// applyMatchModelHint optionally overlays learned expected goals onto one
// fixture's match configuration. The hint is advisory: the engine blends it
// inside a bounded corridor and the RNG still resolves the scoreline.
func (tm *TournamentManager) applyMatchModelHint(cfg *matchengine.InstantMatchConfig, home, away *models.Club, f *Fixture) {
	brain := tm.AIBrain
	if brain == nil || !brain.Config().UseMatchModel {
		return
	}
	model := brain.Model()
	if model == nil {
		return
	}
	pred, err := model.PredictMatch(footballai.MatchRequest{
		Match:   matchContextFromClubs(home, away, f),
		World:   worldContextFromFixture(f),
		Manager: managerFeaturesFromProfile(tm.Managers[f.HomeID]),
	})
	if err != nil {
		return // never break a fixture on a model error
	}
	// Reject implausible expectations outright: a broken or mismatched model
	// must never bend the engine's rates. Sane xG lives in (0, 6].
	if pred.HomeXG <= 0 || pred.HomeXG > 6 || pred.AwayXG <= 0 || pred.AwayXG > 6 {
		return
	}
	cfg.XGHint = &matchengine.XGHint{
		Home: float64(pred.HomeXG),
		Away: float64(pred.AwayXG),
	}
}

// adjustInjuryChance optionally tilts one player's injury chance with the
// model's calibrated probability. The base chance (medical risk model) stays
// authoritative; the model only bends it inside a narrow corridor, and the
// RNG still decides whether an injury actually happens.
func (tm *TournamentManager) adjustInjuryChance(chance float64, player *models.Player, fixtureID string) float64 {
	brain := tm.AIBrain
	if brain == nil || !brain.Config().UseInjuryModel {
		return chance
	}
	model := brain.Model()
	if model == nil {
		return chance
	}
	pred, err := model.PredictInjury(footballai.InjuryRequest{
		Player: tm.playerFeatures(player),
		World:  footballai.WorldContext{MatchImportance: fixtureImportance(fixtureID)},
	})
	if err != nil {
		return chance
	}
	ratio := footballai.InjuryCalibrationRatio(pred.Probability)
	// 0.7 + 0.3*ratio keeps the adjustment inside [+/-30%] of the base
	// chance for the ratio corridor [0.4, 2.5]: [0.82x, 1.45x].
	return chance * (0.7 + 0.3*ratio)
}

// matchContextFromClubs builds the match feature group from club state.
// Unavailable signals stay zero: the shared encoder treats unfilled slots
// as missing rather than fabricating values.
func matchContextFromClubs(home, away *models.Club, f *Fixture) footballai.MatchContext {
	cfg := footballai.MatchContext{
		HomeRating:    float32(home.OverallTeamRating),
		AwayRating:    float32(away.OverallTeamRating),
		HomeForm:      clubFormModifier(home),
		AwayForm:      clubFormModifier(away),
		HomeFitness:   squadAvgFitness(home),
		AwayFitness:   squadAvgFitness(away),
		HomeAbsences:  squadInjuries(home),
		AwayAbsences:  squadInjuries(away),
		Derby:         f.DerbyName != "",
		EuropeanNight: f.Competition == "ucl",
	}
	if len(f.Referee) > 0 {
		cfg.MatchImportance = fixtureImportance(f.FixtureID)
	}
	return cfg
}

// worldContextFromFixture fills the coarse routing context.
func worldContextFromFixture(f *Fixture) footballai.WorldContext {
	return footballai.WorldContext{
		MatchImportance: fixtureImportance(f.FixtureID),
		ClubPressure:    derbyHeatNorm(f),
	}
}

func derbyHeatNorm(f *Fixture) float32 {
	if f.DerbyHeat <= 0 {
		return 0
	}
	if f.DerbyHeat > 100 {
		return 1
	}
	return float32(f.DerbyHeat) / 100
}

// fixtureImportance maps fixture identity onto the 0..1 importance scale the
// model was trained with: knockout and European fixtures matter most.
func fixtureImportance(fixtureID string) float32 {
	switch {
	case containsStr(fixtureID, "final"), containsStr(fixtureID, "sf"), containsStr(fixtureID, "qf"):
		return 1
	case containsStr(fixtureID, "ucl"):
		return 0.8
	default:
		return 0.4
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// clubFormModifier converts the recent-results ring into the signed modifier
// the match feature group expects (W +3, D 0, L -3, last five).
func clubFormModifier(c *models.Club) float32 {
	if c == nil {
		return 0
	}
	mod := float32(0)
	n := len(c.Form)
	if n > 5 {
		n = 5
	}
	for _, r := range c.Form[len(c.Form)-n:] {
		switch r {
		case "W":
			mod += 3
		case "L":
			mod -= 3
		}
	}
	return mod
}

// squadAvgFitness averages squad fitness (0-100).
func squadAvgFitness(c *models.Club) float32 {
	if c == nil || len(c.Squad) == 0 {
		return 0
	}
	sum := 0
	for _, p := range c.Squad {
		if p != nil {
			sum += p.Fitness
		}
	}
	return float32(sum) / float32(len(c.Squad))
}

// squadInjuries counts currently injured squad players.
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

// playerFeatures maps a live player onto the player feature group. Potential
// comes from the growth engine's biometrics when known. Fields the live model
// cannot see cheaply (fatigue, workload) stay zero rather than fabricated.
func (tm *TournamentManager) playerFeatures(p *models.Player) footballai.PlayerFeatures {
	if p == nil {
		return footballai.PlayerFeatures{}
	}
	f := footballai.PlayerFeatures{
		Age:          p.Age,
		OVR:          float32(p.OVR),
		Fitness:      float32(p.Fitness),
		Sharpness:    float32(p.Sharpness),
		Morale:       float32(p.Morale),
		FormModifier: float32(p.FormModifier()),
		Position:     footballai.PositionFromGamePos(p.Position),
	}
	if tm.GrowthEngine != nil {
		if pot, ok := tm.GrowthEngine.PotentialFor(p.PlayerID); ok {
			f.Potential = float32(pot)
		}
	}
	if f.Potential == 0 {
		f.Potential = f.OVR // no biometrics yet: neutral, not misleading
	}
	if p.Injury != "" || p.InjuredMatches > 0 {
		f.RecentInjury = true
	}
	return f
}

// managerFeaturesFromProfile maps manager traits, not identity, onto the
// manager feature group.
func managerFeaturesFromProfile(m *managers.ManagerProfile) footballai.ManagerFeatures {
	if m == nil {
		return footballai.ManagerFeatures{}
	}
	p := managers.PersonalityForManager(m.Style, m.Name)
	return footballai.ManagerFeatures{
		Risk:               float32(p.RiskAppetite),
		YouthPreference:    float32(p.YouthTrust),
		FinancialCaution:   float32(1 - p.TransferAggression),
		Patience:           float32(p.Patience) / 10,
		RotationPreference: float32(p.Rotation),
		Attacking:          float32(p.RiskAppetite),
		Pressing:           float32(p.PressIntensity) / 10,
	}
}
