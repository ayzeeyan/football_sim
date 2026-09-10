package growth

import (
	"math"
	"math/rand"
	"sync"
	"time"
)

// GrowthEngine coordinates biometric progression, attribute training,
// puberty cycles, aging decline, and senior mentorship.
type GrowthEngine struct {
	mu                sync.RWMutex
	Biometrics        map[string]*BiometricProfile    `json:"biometrics"`
	Attributes        map[string]*TechnicalAttributes `json:"attributes"`
	Milestones        []GrowthMilestone               `json:"milestones"`
	Timeline          map[string][]TimelineEntry      `json:"timeline"`
	TrainingEnergy    int                             `json:"training_energy"`
	MaxTrainingEnergy int                             `json:"max_training_energy"`
	rng               *rand.Rand
}

var (
	defaultEngineMu sync.RWMutex
	defaultEngine   *GrowthEngine
)

// SetDefaultGrowthEngine registers the global singleton instance.
func SetDefaultGrowthEngine(ge *GrowthEngine) {
	defaultEngineMu.Lock()
	defer defaultEngineMu.Unlock()
	defaultEngine = ge
}

// GetDefaultGrowthEngine returns the global singleton instance.
func GetDefaultGrowthEngine() *GrowthEngine {
	defaultEngineMu.RLock()
	defer defaultEngineMu.RUnlock()
	return defaultEngine
}

// NewGrowthEngine initializes a thread-safe GrowthEngine.
// If seed is 0, the current time is used for seeding.
func NewGrowthEngine(seed int64) *GrowthEngine {
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	ge := &GrowthEngine{
		Biometrics:        make(map[string]*BiometricProfile),
		Attributes:        make(map[string]*TechnicalAttributes),
		Milestones:        make([]GrowthMilestone, 0),
		Timeline:          make(map[string][]TimelineEntry),
		TrainingEnergy:    3,
		MaxTrainingEnergy: 3,
		rng:               rand.New(rand.NewSource(seed)),
	}
	SetDefaultGrowthEngine(ge)
	return ge
}

// ReplenishTrainingEnergy restores available training sessions to maximum.
func (ge *GrowthEngine) ReplenishTrainingEnergy() {
	ge.mu.Lock()
	defer ge.mu.Unlock()
	ge.TrainingEnergy = ge.MaxTrainingEnergy
}

// RegisterProdigy initializes a prodigy's biometric profile and attribute matrix.
func (ge *GrowthEngine) RegisterProdigy(
	id, name string,
	age int,
	height, weight float64,
	posCat string,
	baseOvr, potential, adultAge int,
) (*BiometricProfile, *TechnicalAttributes) {
	ge.mu.Lock()
	defer ge.mu.Unlock()

	adultAgeResolved := AdultHeightAgeFor(name, adultAge)
	yearsLeft := adultAgeResolved - age
	if yearsLeft < 0 {
		yearsLeft = 0
	}

	var remaining float64
	var puberty string
	if yearsLeft <= 0 {
		remaining = 0.0
		puberty = "Adult frame"
	} else {
		var perYear float64
		if age <= 14 {
			perYear = 1.85
		} else if age <= 16 {
			perYear = 1.7
		} else {
			perYear = 1.25
		}
		remaining = math.Round(float64(yearsLeft)*perYear*10) / 10
		if age <= 14 {
			puberty = "Early-puberty"
		} else if age <= 16 {
			puberty = "Mid-puberty"
		} else {
			puberty = "Late-puberty"
		}
	}

	bio := &BiometricProfile{
		PlayerID:          id,
		FullName:          name,
		Age:               age,
		CurrentHeightCM:   height,
		BaselineHeightCM:  height,
		CurrentWeightKG:   weight,
		BaselineWeightKG:  weight,
		Potential:         potential,
		GrowthVelocity:    remaining,
		PubertyStage:      puberty,
		AccumulatedXP:     0.0,
		LevelXPTarget:     160.0,
		AdultHeightAge:    adultAgeResolved,
		BaselineOVR:       baseOvr,
		SeasonStartOVR:    baseOvr,
		YearlyHeightTaken: 0.0,
	}

	attrs := ge.seedAttributes(baseOvr, posCat, height)

	ge.Biometrics[id] = bio
	ge.Attributes[id] = attrs

	ge.internalNudgeToOVR(id, posCat, baseOvr)

	return bio, attrs
}

func (ge *GrowthEngine) seedAttributes(baselineOvr int, posCat string, heightCM float64) *TechnicalAttributes {
	t := baselineOvr
	hReach := int(heightCM * 0.40)

	if posCat == "FWD" {
		return &TechnicalAttributes{
			Pace:            minInt(99, t+3),
			Shooting:        minInt(99, t+4),
			Passing:         maxInt(48, t-10),
			Dribbling:       minInt(99, t+2),
			Defending:       maxInt(38, t-32),
			Physicality:     maxInt(52, t-8),
			AerialReach:     hReach,
			HeadingPower:    maxInt(52, t-6),
			Strength:        maxInt(52, t-8),
			Shielding:       maxInt(52, t-6),
			PressResistance: maxInt(54, t-4),
			Stamina:         maxInt(58, t-2),
			Composure:       maxInt(54, t-5),
		}
	}
	if posCat == "MID" {
		return &TechnicalAttributes{
			Pace:            minInt(99, t+1),
			Shooting:        maxInt(50, t-4),
			Passing:         minInt(99, t+5),
			Dribbling:       minInt(99, t+3),
			Defending:       maxInt(48, t-12),
			Physicality:     maxInt(54, t-6),
			AerialReach:     int(heightCM * 0.38),
			HeadingPower:    maxInt(50, t-10),
			Strength:        maxInt(52, t-8),
			Shielding:       maxInt(56, t-4),
			PressResistance: minInt(99, t+2),
			Stamina:         minInt(99, t+2),
			Composure:       minInt(99, t+1),
		}
	}
	// DEF
	return &TechnicalAttributes{
		Pace:            minInt(99, t+1),
		Shooting:        maxInt(42, t-22),
		Passing:         maxInt(52, t-6),
		Dribbling:       maxInt(50, t-8),
		Defending:       minInt(99, t+5),
		Physicality:     minInt(99, t+2),
		AerialReach:     int(heightCM * 0.44),
		HeadingPower:    minInt(99, t+2),
		Strength:        minInt(99, t+3),
		Shielding:       minInt(99, t+1),
		PressResistance: maxInt(52, t-6),
		Stamina:         maxInt(58, t-2),
		Composure:       maxInt(54, t-5),
	}
}

// internalNudgeToOVR adjusts attributes incrementally until target OVR is reached.
// Caller must hold ge.mu write lock.
func (ge *GrowthEngine) internalNudgeToOVR(playerID string, posCat string, target int) {
	attrs := ge.Attributes[playerID]
	if attrs == nil {
		return
	}

	keys := []string{"pace", "shooting", "passing", "dribbling", "defending", "physicality"}
	for i := 0; i < 24; i++ {
		current := ge.internalCalculateOVR(playerID, posCat)
		if current == target {
			return
		}
		step := 1
		if current > target {
			step = -1
		}

		// Shuffle keys
		shuffled := make([]string, len(keys))
		copy(shuffled, keys)
		ge.rng.Shuffle(len(shuffled), func(a, b int) {
			shuffled[a], shuffled[b] = shuffled[b], shuffled[a]
		})

		for _, k := range shuffled {
			val := getAttr(attrs, k)
			nxt := maxInt(30, minInt(99, val+step))
			if nxt != val {
				setAttr(attrs, k, nxt)
				break
			}
		}
	}
}

// CalculateOVR computes current tactical OVR based on positional attribute weighting.
func (ge *GrowthEngine) CalculateOVR(playerID string, posCat string) int {
	ge.mu.RLock()
	defer ge.mu.RUnlock()
	return ge.internalCalculateOVR(playerID, posCat)
}

func (ge *GrowthEngine) internalCalculateOVR(playerID string, posCat string) int {
	attrs := ge.Attributes[playerID]
	if attrs == nil {
		return 75
	}

	var ovr float64
	switch posCat {
	case "FWD":
		ovr = float64(attrs.Pace)*0.25 + float64(attrs.Shooting)*0.35 + float64(attrs.Dribbling)*0.20 +
			float64(attrs.Passing)*0.10 + float64(attrs.Physicality)*0.10
	case "MID":
		ovr = float64(attrs.Passing)*0.30 + float64(attrs.Dribbling)*0.25 + float64(attrs.Pace)*0.15 +
			float64(attrs.Shooting)*0.15 + float64(attrs.Physicality)*0.15
	default: // DEF
		ovr = float64(attrs.Defending)*0.40 + float64(attrs.Physicality)*0.25 + float64(attrs.Pace)*0.15 +
			float64(attrs.Passing)*0.15 + float64(attrs.Dribbling)*0.05
	}

	cap := 99
	if bio := ge.Biometrics[playerID]; bio != nil {
		cap = bio.Potential
	}

	rounded := int(math.Round(ovr))
	if rounded < 60 {
		rounded = 60
	}
	if rounded > cap {
		rounded = cap
	}
	return rounded
}

// SetMentorship links a prodigy with a senior squad mentor.
func (ge *GrowthEngine) SetMentorship(playerID string, mentorID, mentorName string, mentorOVR int, personality string) {
	ge.mu.Lock()
	defer ge.mu.Unlock()

	bio := ge.Biometrics[playerID]
	if bio != nil {
		bio.MentorID = mentorID
		bio.MentorName = mentorName
		bio.MentorOVR = mentorOVR
		if personality != "" {
			bio.Personality = personality
		}
	}
}

// SetSeasonStartOVR sets the baseline OVR at the start of a season.
func (ge *GrowthEngine) SetSeasonStartOVR(playerID string, ovr int) {
	ge.mu.Lock()
	defer ge.mu.Unlock()

	if bio := ge.Biometrics[playerID]; bio != nil {
		bio.SeasonStartOVR = ovr
	}
}

// StillGrowing returns true if the player has not completed their physical height curve.
func (ge *GrowthEngine) StillGrowing(bio *BiometricProfile) bool {
	if bio.Age >= bio.AdultHeightAge {
		bio.PubertyStage = "Adult frame"
		return false
	}
	if bio.HeightGainCM() >= bio.GrowthVelocity-0.05 {
		return false
	}
	return true
}

// RecordTimelineEntry captures a season progression snapshot.
func (ge *GrowthEngine) RecordTimelineEntry(
	playerID, season string,
	age, ovr int,
	heightCM, weightKG float64,
	goals, assists, appearances int,
	clubShort, mentorName string,
) {
	ge.mu.Lock()
	defer ge.mu.Unlock()

	entry := TimelineEntry{
		Season:      season,
		Age:         age,
		OVR:         ovr,
		HeightCM:    math.Round(heightCM*10) / 10,
		WeightKG:    math.Round(weightKG*10) / 10,
		Goals:       goals,
		Assists:     assists,
		Appearances: appearances,
		ClubShort:   clubShort,
		MentorName:  mentorName,
	}

	list := ge.Timeline[playerID]
	for i, item := range list {
		if item.Season == season {
			list[i] = entry
			ge.Timeline[playerID] = list
			return
		}
	}
	ge.Timeline[playerID] = append(list, entry)
}

// GetProgressionHistory returns the chronological timeline for a wonderkid.
func (ge *GrowthEngine) GetProgressionHistory(playerID string) []TimelineEntry {
	ge.mu.RLock()
	defer ge.mu.RUnlock()

	list := ge.Timeline[playerID]
	result := make([]TimelineEntry, len(list))
	copy(result, list)
	return result
}

// GetProdigyData returns a snapshot view for the Wonderkid Lab UI.
func (ge *GrowthEngine) GetProdigyData(playerID, posCat string) (map[string]interface{}, bool) {
	ge.mu.Lock()
	defer ge.mu.Unlock()

	bio := ge.Biometrics[playerID]
	attrs := ge.Attributes[playerID]
	if bio == nil || attrs == nil {
		return nil, false
	}

	ovr := ge.internalCalculateOVR(playerID, posCat)
	var xpPct float64
	if bio.LevelXPTarget > 0 {
		xpPct = math.Round((bio.AccumulatedXP/bio.LevelXPTarget)*1000) / 10
	}

	attrMap := map[string]int{
		"pace":             attrs.Pace,
		"shooting":         attrs.Shooting,
		"passing":          attrs.Passing,
		"dribbling":        attrs.Dribbling,
		"defending":        attrs.Defending,
		"physicality":      attrs.Physicality,
		"aerial_reach":     attrs.AerialReach,
		"heading_power":    attrs.HeadingPower,
		"strength":         attrs.Strength,
		"shielding":        attrs.Shielding,
		"press_resistance": attrs.PressResistance,
		"stamina":          attrs.Stamina,
		"composure":        attrs.Composure,
	}

	timelineCopy := make([]TimelineEntry, len(ge.Timeline[playerID]))
	copy(timelineCopy, ge.Timeline[playerID])
	stillGrowing := ge.StillGrowing(bio)

	data := map[string]interface{}{
		"player_id":           bio.PlayerID,
		"full_name":           bio.FullName,
		"age":                 bio.Age,
		"current_height_cm":   bio.CurrentHeightCM,
		"baseline_height_cm":  bio.BaselineHeightCM,
		"height_gain_cm":      bio.HeightGainCM(),
		"current_weight_kg":   bio.CurrentWeightKG,
		"baseline_weight_kg":  bio.BaselineWeightKG,
		"weight_gain_kg":      bio.WeightGainKG(),
		"potential":           bio.Potential,
		"puberty_stage":       bio.PubertyStage,
		"growth_velocity":     bio.GrowthVelocity,
		"accumulated_xp":      math.Round(bio.AccumulatedXP*10) / 10,
		"level_xp_target":     math.Round(bio.LevelXPTarget*10) / 10,
		"xp_pct":              xpPct,
		"ovr":                 ovr,
		"height_display":      FormatHeightCMFt(bio.CurrentHeightCM),
		"adult_height_age":    bio.AdultHeightAge,
		"still_growing":       stillGrowing,
		"training_energy":     ge.TrainingEnergy,
		"max_training_energy": ge.MaxTrainingEnergy,
		"mentor_id":           bio.MentorID,
		"mentor_name":         bio.MentorName,
		"mentor_ovr":          bio.MentorOVR,
		"personality":         bio.Personality,
		"attributes":          attrMap,
		"progression_history": timelineCopy,
	}

	return data, true
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func getAttr(a *TechnicalAttributes, key string) int {
	switch key {
	case "pace":
		return a.Pace
	case "shooting":
		return a.Shooting
	case "passing":
		return a.Passing
	case "dribbling":
		return a.Dribbling
	case "defending":
		return a.Defending
	case "physicality":
		return a.Physicality
	case "aerial_reach":
		return a.AerialReach
	case "heading_power":
		return a.HeadingPower
	case "strength":
		return a.Strength
	case "shielding":
		return a.Shielding
	case "press_resistance":
		return a.PressResistance
	case "stamina":
		return a.Stamina
	case "composure":
		return a.Composure
	default:
		return 0
	}
}

func setAttr(a *TechnicalAttributes, key string, val int) {
	switch key {
	case "pace":
		a.Pace = val
	case "shooting":
		a.Shooting = val
	case "passing":
		a.Passing = val
	case "dribbling":
		a.Dribbling = val
	case "defending":
		a.Defending = val
	case "physicality":
		a.Physicality = val
	case "aerial_reach":
		a.AerialReach = val
	case "heading_power":
		a.HeadingPower = val
	case "strength":
		a.Strength = val
	case "shielding":
		a.Shielding = val
	case "press_resistance":
		a.PressResistance = val
	case "stamina":
		a.Stamina = val
	case "composure":
		a.Composure = val
	}
}
