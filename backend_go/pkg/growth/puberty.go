package growth

import (
	"fmt"
	"math"
)

// SimulatePubertyCycle executes a probabilistic puberty growth tick for a player.
// week is an optional parameter matching tournament simulation matchweek signatures.
func (ge *GrowthEngine) SimulatePubertyCycle(playerID string, week ...int) []string {
	ge.mu.Lock()
	defer ge.mu.Unlock()

	bio := ge.Biometrics[playerID]
	attrs := ge.Attributes[playerID]
	if bio == nil || attrs == nil {
		return nil
	}
	posCat := bio.PositionCategory
	if posCat == "" {
		posCat = "FWD"
	}
	ge.enforceSeasonOVRCapUnlocked(playerID, posCat)

	playerName := bio.FullName
	events := make([]string, 0)
	if !ge.StillGrowing(bio) {
		return events
	}

	var yearCap float64
	if bio.Age <= 14 {
		yearCap = 2.6
	} else if bio.Age <= 16 {
		yearCap = 2.4
	} else {
		yearCap = 1.8
	}

	// Height spurt: 24% chance if within yearly cap
	if ge.rng.Float64() < 0.24 && bio.YearlyHeightTaken < yearCap {
		remaining := math.Max(0.0, bio.GrowthVelocity-bio.HeightGainCM())
		randInc := 0.2 + ge.rng.Float64()*(0.45-0.2)
		capRemaining := yearCap - bio.YearlyHeightTaken
		hGain := math.Round(math.Min(remaining, math.Min(capRemaining, randInc))*10) / 10

		if hGain >= 0.1 {
			bio.CurrentHeightCM = math.Round((bio.CurrentHeightCM+hGain)*10) / 10
			bio.YearlyHeightTaken = math.Round((bio.YearlyHeightTaken+hGain)*10) / 10

			if ge.rng.Float64() < 0.45 {
				ge.tryIncrementAttributeUnlocked(playerID, posCat, "aerial_reach")
			}

			msg := fmt.Sprintf("Growth spurt: %s grew +%.1f cm (%s).",
				playerName, hGain, FormatHeightCMFt(bio.CurrentHeightCM))
			events = append(events, msg)

			milestone := GrowthMilestone{
				Timestamp:   "Matchweek Growth",
				PlayerName:  playerName,
				EventType:   "BIOMETRIC",
				Description: msg,
				BadgeColor:  "cyan",
			}
			ge.Milestones = append([]GrowthMilestone{milestone}, ge.Milestones...)
		}
	}

	// Weight gain: 20% chance if under 5.0 kg total gain limit
	if ge.rng.Float64() < 0.20 && bio.WeightGainKG() < 5.0 {
		randWeight := 0.2 + ge.rng.Float64()*(0.4-0.2)
		wGain := math.Round(randWeight*10) / 10
		if bio.WeightGainKG()+wGain > 5.0 {
			wGain = math.Round((5.0-bio.WeightGainKG())*10) / 10
		}

		if wGain >= 0.1 {
			bio.CurrentWeightKG = math.Round((bio.CurrentWeightKG+wGain)*10) / 10

			if ge.rng.Float64() < 0.40 {
				ge.tryIncrementAttributeUnlocked(playerID, posCat, "stamina")
			}

			msg := fmt.Sprintf("Athletic framing: %s gained +%.1f kg (%.1f kg).",
				playerName, wGain, bio.CurrentWeightKG)
			events = append(events, msg)

			milestone := GrowthMilestone{
				Timestamp:   "Matchweek Growth",
				PlayerName:  playerName,
				EventType:   "BIOMETRIC",
				Description: msg,
				BadgeColor:  "green",
			}
			ge.Milestones = append([]GrowthMilestone{milestone}, ge.Milestones...)
		}
	}

	// Transition to Adult Frame if growth criteria satisfied
	if !ge.StillGrowing(bio) {
		bio.PubertyStage = "Adult frame"
		events = append(events, fmt.Sprintf("%s has reached adult height (%s).",
			playerName, FormatHeightCMFt(bio.CurrentHeightCM)))
	}

	return events
}

// ResetYearlyHeightTaken resets the annual height accumulator upon season advancement.
func (ge *GrowthEngine) ResetYearlyHeightTaken(playerID string) {
	ge.mu.Lock()
	defer ge.mu.Unlock()

	bio := ge.Biometrics[playerID]
	if bio != nil {
		bio.YearlyHeightTaken = 0.0
	}
}
