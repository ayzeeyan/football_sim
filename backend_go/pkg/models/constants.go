package models

import (
	"math"
	"strings"
)

// ExamWeeks contains the domestic matchweeks when enrolled prodigies must sit exams.
var ExamWeeks = map[int]bool{
	12: true,
	13: true,
	24: true,
	25: true,
	32: true,
	33: true,
}

// IsExamWeek reports whether the given matchweek is an examination week.
func IsExamWeek(matchweek int) bool {
	return ExamWeeks[matchweek]
}

// PositionPaths maps primary positions to secondary positions a prodigy can learn.
var PositionPaths = map[string][]string{
	"CAM": {"CF", "ST"},
	"CF":  {"ST", "CAM"},
	"ST":  {"CF", "LW"},
	"LW":  {"CAM", "ST"},
	"RW":  {"CAM", "ST"},
}

// PositionOptions returns a copy of learnable secondary positions for the given position.
func PositionOptions(pos string) []string {
	paths, ok := PositionPaths[strings.ToUpper(strings.TrimSpace(pos))]
	if !ok {
		return nil
	}
	out := make([]string, len(paths))
	copy(out, paths)
	return out
}

// GetPositionCategory maps a specific position to GK, DEF, MID, or FWD.
// NOTE: CAM is explicitly categorized as FWD per the match engine design.
func GetPositionCategory(pos string) string {
	p := strings.ToUpper(strings.TrimSpace(pos))
	switch p {
	case "GK":
		return "GK"
	case "CB", "LB", "RB", "LWB", "RWB":
		return "DEF"
	case "CDM", "CM":
		return "MID"
	default:
		// Includes CAM, CF, ST, LW, RW, etc.
		return "FWD"
	}
}

// KnownKitColors maps club short names to their canonical primary and secondary RGB colors.
var KnownKitColors = map[string][2][3]uint8{
	"ARS": {{219, 0, 7}, {255, 255, 255}},     // Arsenal red / white
	"CHE": {{3, 70, 148}, {255, 255, 255}},     // Chelsea blue / white
	"LIV": {{200, 16, 46}, {0, 178, 169}},      // Liverpool red / teal
	"MCI": {{108, 171, 221}, {28, 44, 91}},     // Man City sky / navy
	"MUN": {{218, 41, 28}, {251, 225, 34}},     // Man Utd red / gold
	"NEW": {{30, 30, 30}, {255, 255, 255}},     // Newcastle black / white
	"TOT": {{19, 34, 87}, {255, 255, 255}},     // Spurs navy / white
	"AVL": {{149, 177, 218}, {103, 14, 54}},    // Villa claret / blue
	"BAR": {{0, 77, 152}, {165, 0, 68}},        // Barca blau / grana
	"RMA": {{245, 245, 245}, {254, 190, 16}},   // Real Madrid white / gold
	"ATM": {{203, 53, 36}, {39, 46, 97}},       // Atletico red / blue
	"RSO": {{0, 102, 204}, {255, 255, 255}},    // Real Sociedad blue / white
	"ATH": {{238, 37, 35}, {255, 255, 255}},    // Athletic Club red / white
	"BAY": {{220, 5, 45}, {0, 102, 178}},       // Bayern red / blue
	"BVB": {{253, 225, 0}, {20, 20, 20}},       // Dortmund yellow / black
	"LEV": {{227, 34, 33}, {20, 20, 20}},       // Leverkusen red / black
	"RBL": {{227, 24, 55}, {255, 255, 255}},    // Leipzig red / white
	"INT": {{0, 20, 137}, {20, 20, 20}},        // Inter blue / black
	"MIL": {{251, 9, 11}, {20, 20, 20}},        // Milan red / black
	"JUV": {{240, 240, 240}, {20, 20, 20}},     // Juventus white / black
	"NAP": {{18, 160, 215}, {255, 255, 255}},   // Napoli azure / white
	"ROM": {{138, 30, 43}, {241, 158, 31}},     // Roma carmine / gold
	"PSG": {{0, 65, 112}, {218, 41, 28}},       // PSG navy / red
	"OM":  {{47, 174, 224}, {255, 255, 255}},   // Marseille sky / white
	"ASM": {{226, 0, 26}, {255, 255, 255}},     // Monaco red / white
	"OL":  {{29, 66, 138}, {218, 41, 28}},      // Lyon blue / red
}

// HSVToRGB converts HSV values (h in [0, 360), s in [0, 1], v in [0, 1]) to an RGB uint8 array.
func HSVToRGB(h, s, v float64) [3]uint8 {
	c := v * s
	hp := math.Mod(h, 360.0) / 60.0
	x := c * (1.0 - math.Abs(math.Mod(hp, 2.0)-1.0))
	m := v - c

	var r1, g1, b1 float64
	switch {
	case hp >= 0 && hp < 1:
		r1, g1, b1 = c, x, 0
	case hp >= 1 && hp < 2:
		r1, g1, b1 = x, c, 0
	case hp >= 2 && hp < 3:
		r1, g1, b1 = 0, c, x
	case hp >= 3 && hp < 4:
		r1, g1, b1 = 0, x, c
	case hp >= 4 && hp < 5:
		r1, g1, b1 = x, 0, c
	case hp >= 5 && hp < 6:
		r1, g1, b1 = c, 0, x
	default:
		r1, g1, b1 = 0, 0, 0
	}

	clampUint8 := func(val float64) uint8 {
		rounded := math.Round((val + m) * 255.0)
		if rounded < 0 {
			return 0
		}
		if rounded > 255 {
			return 255
		}
		return uint8(rounded)
	}

	return [3]uint8{
		clampUint8(r1),
		clampUint8(g1),
		clampUint8(b1),
	}
}

// hashShortName produces a stable hash integer for color generation.
func hashShortName(s string) uint32 {
	var h uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
}

// KitColorsForClub returns canonical or procedurally generated kit colors for a club.
func KitColorsForClub(shortName string) (primary [3]uint8, secondary [3]uint8) {
	upper := strings.ToUpper(strings.TrimSpace(shortName))
	if colors, ok := KnownKitColors[upper]; ok {
		return colors[0], colors[1]
	}

	// Deterministic distinct color based on club short name (HSV: h % 360, s: 85%, v: 90%)
	h := float64(hashShortName(upper) % 360)
	primary = HSVToRGB(h, 0.85, 0.90)
	secondary = [3]uint8{255, 255, 255}
	return primary, secondary
}
