package growth

// MaxAnnualOVRGain is the exceptional hard ceiling for one development season.
// Most players should finish below it naturally; this guard exists so weekly
// training, mentorship and match-XP paths cannot stack past the same annual
// boundary before the season-end growth pass runs.
const MaxAnnualOVRGain = 5

// EnforceSeasonOVRCap constrains the technical attribute matrix to the current
// season's hard OVR ceiling and returns the resulting OVR. It intentionally
// operates on attributes as well as the displayed OVR: merely clamping the
// returned number would leave hidden over-development that reappears next
// season.
func (ge *GrowthEngine) EnforceSeasonOVRCap(playerID, posCat string) int {
	ge.mu.Lock()
	defer ge.mu.Unlock()

	bio := ge.Biometrics[playerID]
	attrs := ge.Attributes[playerID]
	if bio == nil || attrs == nil {
		return ge.internalCalculateOVR(playerID, posCat)
	}

	start := bio.SeasonStartOVR
	if start <= 0 {
		start = bio.BaselineOVR
	}
	if start <= 0 {
		return ge.internalCalculateOVR(playerID, posCat)
	}

	ceiling := start + MaxAnnualOVRGain
	if bio.Potential > 0 && ceiling > bio.Potential {
		ceiling = bio.Potential
	}
	if ceiling > 99 {
		ceiling = 99
	}

	// Reduce the attributes that matter most for this positional OVR first.
	// The order is fixed, not random, so enforcing a safety ceiling never
	// consumes the development RNG stream or changes later same-seed outcomes.
	keys := []string{"pace", "shooting", "dribbling", "passing", "physicality"}
	switch posCat {
	case "MID":
		keys = []string{"passing", "dribbling", "pace", "shooting", "physicality"}
	case "DEF", "GK":
		keys = []string{"defending", "physicality", "pace", "passing", "dribbling"}
	}

	for guard := 0; guard < 256 && ge.internalCalculateOVR(playerID, posCat) > ceiling; guard++ {
		changed := false
		for _, key := range keys {
			value := getAttr(attrs, key)
			if value <= 30 {
				continue
			}
			setAttr(attrs, key, value-1)
			changed = true
			if ge.internalCalculateOVR(playerID, posCat) <= ceiling {
				break
			}
		}
		if !changed {
			break
		}
	}

	result := ge.internalCalculateOVR(playerID, posCat)
	if result > ceiling {
		return ceiling
	}
	return result
}
