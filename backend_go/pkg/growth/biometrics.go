package growth

import (
	"fmt"
	"hash/fnv"
	"math"
)

// BiometricProfile tracks dynamic physical puberty and muscular development.
type BiometricProfile struct {
	PlayerID          string  `json:"player_id"`
	FullName          string  `json:"full_name"`
	Age               int     `json:"age"`
	CurrentHeightCM   float64 `json:"current_height_cm"`
	BaselineHeightCM  float64 `json:"baseline_height_cm"`
	CurrentWeightKG   float64 `json:"current_weight_kg"`
	BaselineWeightKG  float64 `json:"baseline_weight_kg"`
	Potential         int     `json:"potential"`
	PositionCategory  string  `json:"position_category,omitempty"`
	GrowthVelocity    float64 `json:"growth_velocity"` // remaining height budget in cm
	PubertyStage      string  `json:"puberty_stage"`   // Early-puberty / Mid-puberty / Late-puberty / Adult frame
	AccumulatedXP     float64 `json:"accumulated_xp"`
	LevelXPTarget     float64 `json:"level_xp_target"`
	AdultHeightAge    int     `json:"adult_height_age"`
	BaselineOVR       int     `json:"baseline_ovr"`
	SeasonStartOVR    int     `json:"season_start_ovr,omitempty"`
	YearlyHeightTaken float64 `json:"yearly_height_taken"`
	MentorID          string  `json:"mentor_id,omitempty"`
	MentorName        string  `json:"mentor_name,omitempty"`
	MentorOVR         int     `json:"mentor_ovr,omitempty"`
	Personality       string  `json:"personality,omitempty"`
}

// HeightGainCM calculates current height gain rounded to 1 decimal place.
func (b *BiometricProfile) HeightGainCM() float64 {
	return math.Round((b.CurrentHeightCM-b.BaselineHeightCM)*10) / 10
}

// WeightGainKG calculates current weight gain rounded to 1 decimal place.
func (b *BiometricProfile) WeightGainKG() float64 {
	return math.Round((b.CurrentWeightKG-b.BaselineWeightKG)*10) / 10
}

// TechnicalAttributes defines the hexagonal attribute matrix and biometric sub-attributes.
type TechnicalAttributes struct {
	// Core hexagonal attributes (30-99)
	Pace        int `json:"pace"`
	Shooting    int `json:"shooting"`
	Passing     int `json:"passing"`
	Dribbling   int `json:"dribbling"`
	Defending   int `json:"defending"`
	Physicality int `json:"physicality"`

	// Sub-attributes influenced by biometrics and mentorship
	AerialReach     int `json:"aerial_reach"`
	HeadingPower    int `json:"heading_power"`
	Strength        int `json:"strength"`
	Shielding       int `json:"shielding"`
	PressResistance int `json:"press_resistance"`
	Stamina         int `json:"stamina"`
	Composure       int `json:"composure"`
}

// GrowthMilestone records a significant developmental event.
type GrowthMilestone struct {
	Timestamp   string `json:"timestamp"`
	PlayerName  string `json:"player_name"`
	EventType   string `json:"event_type"` // "BIOMETRIC", "ATTRIBUTE", "OVR_UPGRADE"
	Description string `json:"description"`
	BadgeColor  string `json:"badge_color"` // "gold", "green", "cyan", "purple"
}

// TimelineEntry represents a seasonal progression snapshot for wonderkids.
type TimelineEntry struct {
	Season      string  `json:"season"`
	Age         int     `json:"age"`
	OVR         int     `json:"ovr"`
	HeightCM    float64 `json:"height_cm"`
	WeightKG    float64 `json:"weight_kg"`
	Goals       int     `json:"goals"`
	Assists     int     `json:"assists"`
	Appearances int     `json:"appearances"`
	ClubShort   string  `json:"club_short"`
	MentorName  string  `json:"mentor_name"`
}

// FormatHeightCMFt formats metric height into formatted metric + imperial string:
// e.g. "178 cm (5'10")" or "182.5 cm (6'0")"
func FormatHeightCMFt(cm float64) string {
	inchesTotal := cm / 2.54
	feet := int(inchesTotal / 12)
	inches := int(math.Round(math.Mod(inchesTotal, 12)))
	if inches == 12 {
		feet++
		inches = 0
	}
	var metric string
	if math.Abs(cm-math.Round(cm)) < 0.05 {
		metric = fmt.Sprintf("%.0f", cm)
	} else {
		metric = fmt.Sprintf("%.1f", cm)
	}
	return fmt.Sprintf("%s cm (%d'%d\")", metric, feet, inches)
}

// AdultHeightAgeFor determines the age when physical height growth ceases (18, 19, or 20).
func AdultHeightAgeFor(name string, configured ...int) int {
	if len(configured) > 0 && (configured[0] == 18 || configured[0] == 19 || configured[0] == 20) {
		return configured[0]
	}
	h := fnv.New32a()
	h.Write([]byte(name))
	return 18 + int(h.Sum32()%3)
}
