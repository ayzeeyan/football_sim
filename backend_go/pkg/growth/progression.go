package growth

import (
	"errors"
	"fmt"
	"math"
)

// MatchXPOptions allows overriding mentorship parameters for match XP calculations.
type MatchXPOptions struct {
	MentorOVR   *int
	MentorName  *string
	Personality *string
}

// MentorshipOptions allows overriding mentorship parameters for weekly clinics.
type MentorshipOptions struct {
	MentorName  *string
	MentorOVR   *int
	Personality *string
}

// ApplyMatchXP awards match performance XP and triggers attribute level-ups.
func (ge *GrowthEngine) ApplyMatchXP(
	playerID, playerName, posCat string,
	matchRating float64,
	goals, assists int,
	opts ...MatchXPOptions,
) []string {
	ge.mu.Lock()
	defer ge.mu.Unlock()

	bio := ge.Biometrics[playerID]
	attrs := ge.Attributes[playerID]
	if bio == nil || attrs == nil {
		return nil
	}
	if posCat == "" {
		posCat = bio.PositionCategory
	}
	if posCat == "" {
		posCat = "FWD"
	}
	ge.enforceSeasonOVRCapUnlocked(playerID, posCat)

	effMentorOVR := bio.MentorOVR
	effMentorName := bio.MentorName
	effPersonality := bio.Personality

	if len(opts) > 0 {
		if opts[0].MentorOVR != nil {
			effMentorOVR = *opts[0].MentorOVR
		}
		if opts[0].MentorName != nil {
			effMentorName = *opts[0].MentorName
		}
		if opts[0].Personality != nil {
			effPersonality = *opts[0].Personality
		}
	}

	// Age multiplier: prime development is 17-21
	var ageMult float64
	if bio.Age <= 16 {
		ageMult = 1.05
	} else if bio.Age <= 18 {
		ageMult = 1.00
	} else if bio.Age <= 21 {
		ageMult = 0.90
	} else if bio.Age <= 24 {
		ageMult = 0.75
	} else {
		ageMult = 0.50
	}

	// Mentor multiplier
	mentorMult := 1.0
	if effMentorOVR > 0 {
		bonus := float64(effMentorOVR-70) * 0.01
		if bonus < 0.05 {
			bonus = 0.05
		} else if bonus > 0.25 {
			bonus = 0.25
		}
		mentorMult += bonus
		if effPersonality == "dedicated_pro" {
			mentorMult += 0.05
		}
	}

	baseXP := matchRating * 2.2
	goalXP := float64(goals) * 5.0
	assistXP := float64(assists) * 3.0
	totalXP := (baseXP + goalXP + assistXP) * ageMult * mentorMult

	bio.AccumulatedXP += totalXP
	events := make([]string, 0)
	cap := bio.Potential
	upgrades := 0

	for bio.AccumulatedXP >= bio.LevelXPTarget && upgrades < 1 {
		bio.AccumulatedXP -= bio.LevelXPTarget
		bio.LevelXPTarget = math.Round(bio.LevelXPTarget*1.04*10) / 10
		upgrades++

		var pool []string
		switch posCat {
		case "FWD":
			pool = []string{"shooting", "pace", "dribbling"}
		case "MID":
			pool = []string{"passing", "dribbling", "composure"}
		default: // DEF
			pool = []string{"defending", "physicality", "pace"}
		}

		chosenAttr := pool[ge.rng.Intn(len(pool))]
		chosenApplied := ge.tryIncrementAttributeUnlocked(playerID, posCat, chosenAttr)

		// Senior mentor composure transfer on level-up
		composureApplied := false
		if effMentorName != "" && effMentorOVR > 0 {
			compChance := 0.35
			if effPersonality == "big_game_performer" {
				compChance = 0.50
			}
			if ge.rng.Float64() < compChance && attrs.Composure < minInt(99, effMentorOVR) {
				composureApplied = ge.tryIncrementAttributeUnlocked(playerID, posCat, "composure")
				if composureApplied {
					compMsg := fmt.Sprintf("Mentorship poise: Under %s's guidance, %s's composure increased to %d.",
						effMentorName, playerName, attrs.Composure)
					events = append(events, compMsg)
					milestone := GrowthMilestone{
						Timestamp:   "Mentorship Wisdom",
						PlayerName:  playerName,
						EventType:   "ATTRIBUTE",
						Description: compMsg,
						BadgeColor:  "gold",
					}
					ge.Milestones = append([]GrowthMilestone{milestone}, ge.Milestones...)
				}
			}
		}

		newOVR := ge.internalCalculateOVR(playerID, posCat)
		if newOVR >= cap {
			events = append(events, fmt.Sprintf("%s is at his ceiling (%d OVR).", playerName, cap))
			break
		}
		if !chosenApplied && !composureApplied {
			break
		}

		msg := fmt.Sprintf("Skill upgrade: %s ticked up. OVR is now %d.", playerName, newOVR)
		events = append(events, msg)
		milestone := GrowthMilestone{
			Timestamp:   "Match Performance",
			PlayerName:  playerName,
			EventType:   "OVR_UPGRADE",
			Description: msg,
			BadgeColor:  "gold",
		}
		ge.Milestones = append([]GrowthMilestone{milestone}, ge.Milestones...)
	}

	return events
}

// ApplyMentorshipTick performs the weekly autonomous senior mentorship interaction.
func (ge *GrowthEngine) ApplyMentorshipTick(
	playerID, playerName string,
	opts ...MentorshipOptions,
) []string {
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

	mName := bio.MentorName
	mOVR := bio.MentorOVR
	pers := bio.Personality
	if pers == "" {
		pers = "dedicated_pro"
	}

	if len(opts) > 0 {
		if opts[0].MentorName != nil {
			mName = *opts[0].MentorName
		}
		if opts[0].MentorOVR != nil {
			mOVR = *opts[0].MentorOVR
		}
		if opts[0].Personality != nil {
			pers = *opts[0].Personality
		}
	}

	if mName == "" || mOVR <= 0 {
		return nil
	}

	events := make([]string, 0)
	procChance := 0.32
	if pers == "dedicated_pro" {
		procChance = 0.45
	}
	if ge.rng.Float64() > procChance {
		return events
	}

	// 1. Composure transfer
	if attrs.Composure < minInt(95, mOVR) {
		if ge.tryIncrementAttributeUnlocked(playerID, posCat, "composure") {
			msg := fmt.Sprintf("Mentorship clinic: %s (%d OVR) coached %s in pressure management and match composure (Composure: %d).",
				mName, mOVR, playerName, attrs.Composure)
			events = append(events, msg)
			milestone := GrowthMilestone{
				Timestamp:   "Senior Mentorship",
				PlayerName:  playerName,
				EventType:   "ATTRIBUTE",
				Description: msg,
				BadgeColor:  "purple",
			}
			ge.Milestones = append([]GrowthMilestone{milestone}, ge.Milestones...)
		}
	}

	// 2. Synergistic attribute drill based on archetype (30% chance)
	if ge.rng.Float64() < 0.30 {
		var skillLabel string
		var applied bool
		switch pers {
		case "academic_dual":
			applied = ge.tryIncrementAttributeUnlocked(playerID, posCat, "press_resistance")
			if applied {
				skillLabel = fmt.Sprintf("Press resistance +1 (%d)", attrs.PressResistance)
			}
		case "big_game_performer":
			applied = ge.tryIncrementAttributeUnlocked(playerID, posCat, "shooting")
			if applied {
				skillLabel = fmt.Sprintf("Shooting +1 (%d)", attrs.Shooting)
			}
		case "flamboyant_star":
			applied = ge.tryIncrementAttributeUnlocked(playerID, posCat, "dribbling")
			if applied {
				skillLabel = fmt.Sprintf("Dribbling +1 (%d)", attrs.Dribbling)
			}
		default:
			applied = ge.tryIncrementAttributeUnlocked(playerID, posCat, "shielding")
			if applied {
				skillLabel = fmt.Sprintf("Shielding +1 (%d)", attrs.Shielding)
			}
		}

		if applied {
			drillMsg := fmt.Sprintf("%s ran advanced 1-on-1 drills with %s (%s).", mName, playerName, skillLabel)
			events = append(events, drillMsg)
			milestone := GrowthMilestone{
				Timestamp:   "Mentorship Drills",
				PlayerName:  playerName,
				EventType:   "ATTRIBUTE",
				Description: drillMsg,
				BadgeColor:  "green",
			}
			ge.Milestones = append([]GrowthMilestone{milestone}, ge.Milestones...)
		}
	}

	// 3. Direct XP injection from veteran guidance
	randXP := 6.0 + ge.rng.Float64()*(12.0-6.0)
	xpInjection := math.Round(randXP*(float64(mOVR)/80.0)*10) / 10
	bio.AccumulatedXP += xpInjection

	return events
}

// RunTrainingCycle executes an intensive training regimen for a player.
// focus can be "hypertrophy", "technical", or "tactical".
// consumeEnergy defaults to true if omitted.
func (ge *GrowthEngine) RunTrainingCycle(
	playerID, focus string,
	consumeEnergy ...bool,
) (map[string]interface{}, error) {
	ge.mu.Lock()
	defer ge.mu.Unlock()

	shouldConsume := true
	if len(consumeEnergy) > 0 {
		shouldConsume = consumeEnergy[0]
	}

	if shouldConsume && ge.TrainingEnergy <= 0 {
		return map[string]interface{}{
			"status":           "error",
			"message":          "No training energy remaining this matchweek. Simulate the next matchweek to recover energy.",
			"remaining_energy": 0,
			"max_energy":       ge.MaxTrainingEnergy,
		}, errors.New("no training energy remaining")
	}

	bio := ge.Biometrics[playerID]
	attrs := ge.Attributes[playerID]
	if bio == nil || attrs == nil {
		return map[string]interface{}{
			"status":  "error",
			"message": "Prodigy not found",
		}, errors.New("prodigy not found")
	}
	posCat := bio.PositionCategory
	if posCat == "" {
		posCat = "FWD"
	}
	ge.enforceSeasonOVRCapUnlocked(playerID, posCat)

	if shouldConsume {
		ge.TrainingEnergy--
	}

	gains := make(map[string]interface{})
	var msg string
	var attrGains []string
	switch focus {
	case "hypertrophy":
		if ge.StillGrowing(bio) && bio.WeightGainKG() < 5.0 {
			randW := 0.1 + ge.rng.Float64()*(0.3-0.1)
			wGain := math.Round(randW*10) / 10
			targetW := bio.BaselineWeightKG + 5.0
			bio.CurrentWeightKG = math.Round(math.Min(targetW, bio.CurrentWeightKG+wGain)*10) / 10
			gains["weight"] = fmt.Sprintf("+%.1f kg", wGain)
		}
		if ge.tryIncrementAttributeUnlocked(playerID, posCat, "strength") {
			attrGains = append(attrGains, "Strength +1")
		}
		if ge.tryIncrementAttributeUnlocked(playerID, posCat, "stamina") {
			attrGains = append(attrGains, "Stamina +1")
		}
		if len(attrGains) > 0 {
			msg = fmt.Sprintf("Hypertrophy training: %s added a little strength and stamina.", bio.FullName)
		}

	case "technical":
		if ge.tryIncrementAttributeUnlocked(playerID, posCat, "dribbling") {
			attrGains = append(attrGains, "Dribbling +1")
		}
		if ge.tryIncrementAttributeUnlocked(playerID, posCat, "passing") {
			attrGains = append(attrGains, "Passing +1")
		}
		if ge.tryIncrementAttributeUnlocked(playerID, posCat, "composure") {
			attrGains = append(attrGains, "Composure +1")
		}
		if len(attrGains) > 0 {
			msg = fmt.Sprintf("Technical work: %s sharpened his touch and tactical composure.", bio.FullName)
		}

	default: // "tactical"
		if ge.tryIncrementAttributeUnlocked(playerID, posCat, "pace") {
			attrGains = append(attrGains, "Pace +1")
		}
		if ge.tryIncrementAttributeUnlocked(playerID, posCat, "press_resistance") {
			attrGains = append(attrGains, "Press resistance +1")
		}
		if len(attrGains) > 0 {
			msg = fmt.Sprintf("Tactical work: %s moved a little quicker between lines.", bio.FullName)
		}
	}

	if attrGains == nil {
		attrGains = []string{}
	}
	gains["attributes"] = attrGains

	if len(attrGains) == 0 && gains["weight"] == nil {
		msg = fmt.Sprintf("%s is at his annual development ceiling.", bio.FullName)
	} else {
		milestone := GrowthMilestone{
			Timestamp:   "Training Cycle",
			PlayerName:  bio.FullName,
			EventType:   "ATTRIBUTE",
			Description: msg,
			BadgeColor:  "purple",
		}
		ge.Milestones = append([]GrowthMilestone{milestone}, ge.Milestones...)
	}

	res := map[string]interface{}{
		"status":           "success",
		"message":          msg,
		"gains":            gains,
		"current_height":   bio.CurrentHeightCM,
		"current_weight":   bio.CurrentWeightKG,
		"height_gain":      bio.HeightGainCM(),
		"weight_gain":      bio.WeightGainKG(),
		"remaining_energy": ge.TrainingEnergy,
		"max_energy":       ge.MaxTrainingEnergy,
	}

	return res, nil
}
