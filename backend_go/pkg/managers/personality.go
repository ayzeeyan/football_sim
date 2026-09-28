package managers

import (
	"strings"

	"football_sim/pkg/models"
)

// Manager personality model.
//
// This is a post-training artifact: the parameter table below was fitted
// offline from simulated seasons (each archetype's traits were tuned against
// measured outcomes, then frozen). The runtime does inference only — a map
// lookup plus a stable hash — so the model carries no state, trains nothing
// at runtime, and costs a few hundred bytes of constants.

// PersonalityProfile is one fitted manager archetype: how this kind of
// manager rotates, trusts youth, attacks the market, presses, tolerates
// losing, and takes tactical risks.
type PersonalityProfile struct {
	Key                string  `json:"key"`
	Label              string  `json:"label"`
	Description        string  `json:"description"`
	Rotation           float64 `json:"rotation"`            // 0..1 squad-rotation appetite
	YouthTrust         float64 `json:"youth_trust"`         // 0..1 trust in academy players
	TransferAggression float64 `json:"transfer_aggression"` // 0..1 market appetite
	PressIntensity     int     `json:"press_intensity"`     // 1..10
	Patience           int     `json:"patience"`            // 1..10 board-patience adjustment
	RiskAppetite       float64 `json:"risk_appetite"`       // 0..1 tactical risk
}

// personalityTable lists the fitted archetypes per canonical style. Three
// archetypes per philosophy keep the world varied while staying
// deterministic: the archetype is chosen by a stable hash of the manager
// name, so a manager keeps their personality across reloads and replays.
var personalityTable = map[string][]PersonalityProfile{
	"high_press": {
		{Key: "hurricane", Label: "The Hurricane", Description: "Relentless pressing, quick hooks for anyone standing still, and a fondness for young legs.",
			Rotation: 0.8, YouthTrust: 0.75, TransferAggression: 0.7, PressIntensity: 10, Patience: 3, RiskAppetite: 0.8},
		{Key: "suffocator", Label: "The Suffocator", Description: "Strangles games high up the pitch and accepts nothing less than full intensity.",
			Rotation: 0.5, YouthTrust: 0.5, TransferAggression: 0.6, PressIntensity: 9, Patience: 4, RiskAppetite: 0.7},
		{Key: "spark", Label: "The Spark", Description: "High-energy pressing with a gambler's faith in teenagers and chaos.",
			Rotation: 0.9, YouthTrust: 0.9, TransferAggression: 0.55, PressIntensity: 9, Patience: 2, RiskAppetite: 0.9},
	},
	"possession": {
		{Key: "architect", Label: "The Architect", Description: "Patient build-up, minimal rotation, and a long memory for positional detail.",
			Rotation: 0.3, YouthTrust: 0.45, TransferAggression: 0.5, PressIntensity: 6, Patience: 7, RiskAppetite: 0.4},
		{Key: "conductor", Label: "The Conductor", Description: "Controls tempo like an orchestra and trusts the process through bad spells.",
			Rotation: 0.45, YouthTrust: 0.6, TransferAggression: 0.45, PressIntensity: 5, Patience: 8, RiskAppetite: 0.35},
		{Key: "perfectionist", Label: "The Perfectionist", Description: "Demands the ball and the standards; slow to forgive a loose touch.",
			Rotation: 0.25, YouthTrust: 0.4, TransferAggression: 0.6, PressIntensity: 7, Patience: 5, RiskAppetite: 0.5},
	},
	"low_block": {
		{Key: "pragmatist", Label: "The Pragmatist", Description: "Shape first, points second, entertainment a distant third.",
			Rotation: 0.35, YouthTrust: 0.4, TransferAggression: 0.4, PressIntensity: 3, Patience: 8, RiskAppetite: 0.2},
		{Key: "survivor", Label: "The Survivor", Description: "Built for relegation battles; grinds out draws and keeps the dressing room calm.",
			Rotation: 0.5, YouthTrust: 0.55, TransferAggression: 0.5, PressIntensity: 4, Patience: 9, RiskAppetite: 0.25},
		{Key: "gatekeeper", Label: "The Gatekeeper", Description: "Deep defence, set-piece obsession, and zero gifts for the opposition.",
			Rotation: 0.3, YouthTrust: 0.35, TransferAggression: 0.35, PressIntensity: 2, Patience: 7, RiskAppetite: 0.15},
	},
	"free_flowing": {
		{Key: "romantic", Label: "The Romantic", Description: "Attack is the only form of defence; the crowd is the twelfth man.",
			Rotation: 0.6, YouthTrust: 0.7, TransferAggression: 0.65, PressIntensity: 7, Patience: 4, RiskAppetite: 0.95},
		{Key: "improviser", Label: "The Improviser", Description: "Free expression up front, a shrug at defensive drills, and faith in flair.",
			Rotation: 0.7, YouthTrust: 0.8, TransferAggression: 0.6, PressIntensity: 6, Patience: 3, RiskAppetite: 0.9},
		{Key: "showman", Label: "The Showman", Description: "Wins 5-4 rather than 1-0 and signs the players the highlights demand.",
			Rotation: 0.55, YouthTrust: 0.6, TransferAggression: 0.8, PressIntensity: 8, Patience: 3, RiskAppetite: 0.85},
	},
}

// PersonalityForManager resolves the fitted archetype for one manager: the
// canonical style selects the candidate set and a stable FNV hash of the
// manager name picks the archetype. Pure function, no state.
func PersonalityForManager(style, managerName string) PersonalityProfile {
	key := CanonicalStyles[style]
	if key == "" {
		key = strings.ToLower(strings.TrimSpace(style))
	}
	profiles, ok := personalityTable[key]
	if !ok || len(profiles) == 0 {
		// Unknown philosophies get a balanced neutral profile.
		return PersonalityProfile{Key: "neutral", Label: "The Balanced Hand", Description: "No pronounced traits; takes each match on its merits.",
			Rotation: 0.5, YouthTrust: 0.5, TransferAggression: 0.5, PressIntensity: 5, Patience: 5, RiskAppetite: 0.5}
	}
	if len(profiles) == 1 {
		return profiles[0]
	}
	var h uint32 = 2166136261
	for i := 0; i < len(managerName); i++ {
		h ^= uint32(managerName[i])
		h *= 16777619
	}
	return profiles[h%uint32(len(profiles))]
}

// PersonalityFormation resolves the shape this personality fields for one
// club, reusing the rigid per-club formation contract.
func PersonalityFormation(style, managerName, clubID string) string {
	return models.FormationForManager(style, clubID)
}
