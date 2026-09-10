package matchengine

// registerTacticalAliases extends the live-pitch slot table to the complete
// position vocabulary used by the universe. The player's stored Position is
// never mutated; these are presentation/tactical slots only.
func init() {
	// Home attacks left -> right. Y=0 is the home team's left touchline,
	// Y=1 is the right touchline.
	home := map[string][2]float64{
		"LCB": {0.18, 0.38}, "RCB": {0.18, 0.62},
		"LDM": {0.32, 0.38}, "RDM": {0.32, 0.62},
		"LCM": {0.40, 0.34}, "RCM": {0.40, 0.66},
		"LAM": {0.50, 0.30}, "RAM": {0.50, 0.70},
		"LF":  {0.60, 0.30}, "RF":  {0.60, 0.70},
	}
	for pos, xy := range home {
		positionHomeCoords[pos] = xy
	}

	// Away coordinates are a 180-degree orientation mirror. A player's true
	// footballing left/right therefore remains visually correct relative to the
	// direction that team attacks.
	away := map[string][2]float64{
		"LCB": {0.82, 0.62}, "RCB": {0.82, 0.38},
		"LDM": {0.68, 0.62}, "RDM": {0.68, 0.38},
		"LCM": {0.60, 0.66}, "RCM": {0.60, 0.34},
		"LAM": {0.50, 0.70}, "RAM": {0.50, 0.30},
		"LF":  {0.40, 0.70}, "RF":  {0.40, 0.30},
	}
	for pos, xy := range away {
		positionAwayCoords[pos] = xy
	}
}
