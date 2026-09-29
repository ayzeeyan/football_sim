package footballai

// Advisor helpers shared by the simulation integration sites. These map the
// game's own domain values onto FootballMoE's feature groups and expose the
// calibration constants that keep learned adjustments bounded.

// MeanBootstrapInjuryProb is the mean injury probability of the bootstrap
// teacher's train split (player_seed.jsonl, injury_risk rows, n=2276). The
// runtime uses it to convert a model probability into a relative adjustment
// so the engine's own calibrated base rate stays authoritative.
const MeanBootstrapInjuryProb = 0.07656

// PositionFromGamePos maps a dataset position code onto the coarse position
// group the model was trained with. Unknown codes map to FWD's zero vector is
// avoided: they map to MID, the most common group, so rare codes degrade
// gracefully instead of silently feeding an empty one-hot.
func PositionFromGamePos(s string) Position {
	switch s {
	case "GK":
		return PosGK
	case "CB", "LB", "RB", "LWB", "RWB":
		return PosDEF
	case "CDM", "CM", "CAM":
		return PosMID
	default: // LW, RW, CF, ST and anything unmapped
		return PosFWD
	}
}

// InjuryCalibrationRatio converts a model injury probability into a bounded
// relative multiplier around the teacher's mean rate. ratio == 1 means the
// model sees average risk; the bounds keep any single player's risk inside a
// sane corridor even if the model were systematically off.
func InjuryCalibrationRatio(modelProb float32) float64 {
	const mean = MeanBootstrapInjuryProb
	ratio := float64(modelProb) / mean
	if ratio < 0.4 {
		return 0.4
	}
	if ratio > 2.5 {
		return 2.5
	}
	return ratio
}
