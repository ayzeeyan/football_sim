package models

import (
	"strings"
)

// PersonalityArchetype represents a player's behavioural profile.
type PersonalityArchetype struct {
	Key         string `json:"key"`
	Title       string `json:"title"`
	Badge       string `json:"badge"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

// PersonalityArchetypes holds the definition of the 4 archetypes.
var PersonalityArchetypes = map[string]PersonalityArchetype{
	"dedicated_pro": {
		Key:         "dedicated_pro",
		Title:       "The Dedicated Professional",
		Badge:       "Dedicated Pro",
		Description: "Focuses entirely on football, trains extra hours, handles praise with humility, high loyalty.",
		Icon:        "shield",
	},
	"flamboyant_star": {
		Key:         "flamboyant_star",
		Title:       "The Flamboyant Prodigy",
		Badge:       "Flamboyant Star",
		Description: "Loves the spotlight, attempts high-risk skill moves, attracts global boot & brand sponsorships.",
		Icon:        "sparkle",
	},
	"academic_dual": {
		Key:         "academic_dual",
		Title:       "The Academic Dual-Threat",
		Badge:       "Academic Dual-Threat",
		Description: "Balances elite academy football with rigorous schooling, calm demeanor, high tactical discipline.",
		Icon:        "book",
	},
	"big_game_performer": {
		Key:         "big_game_performer",
		Title:       "The Big-Game Performer",
		Badge:       "Big-Game Performer",
		Description: "Rises to the occasion in derbies and European nights, ice-cold under pressure, clutch finisher.",
		Icon:        "flame",
	},
}

// ProdigyPersonalities maps canonical prodigy player names to their archetypes.
var ProdigyPersonalities = map[string]string{
	"maverick cantalejo":       "big_game_performer",
	"reid randell libatan":     "dedicated_pro",
	"reid libatan":             "dedicated_pro",
	"venjamin valerio":         "academic_dual",
	"izyan levin bantol":       "flamboyant_star",
	"james bernard rizon":      "big_game_performer",
	"yeshua emmanuel gocotano": "dedicated_pro",
	"cliergy jave lanticse":    "flamboyant_star",
	"earl josh hernando":       "dedicated_pro",
	"ezail zamora":             "academic_dual",
	"ashle zylle baguio":       "flamboyant_star",
	"jhed anthony guinita":     "big_game_performer",
	"rich lorenz suico":        "dedicated_pro",
}

// PersonalityFor resolves the archetype key for a given player name.
func PersonalityFor(name string) string {
	key := strings.ToLower(strings.TrimSpace(name))
	if arch, ok := ProdigyPersonalities[key]; ok {
		return arch
	}

	options := []string{"dedicated_pro", "flamboyant_star", "academic_dual", "big_game_performer"}
	sum := 0
	for _, c := range key {
		sum += int(c)
	}
	return options[sum%len(options)]
}

// ArchetypeForKey returns the archetype struct for a given key, defaulting to dedicated_pro.
func ArchetypeForKey(key string) PersonalityArchetype {
	if arch, ok := PersonalityArchetypes[key]; ok {
		return arch
	}
	return PersonalityArchetypes["dedicated_pro"]
}

// SchoolWantFor determines deterministic preference between school and football.
func SchoolWantFor(name string) string {
	n := 0
	lower := strings.ToLower(strings.TrimSpace(name))
	for i, ch := range lower {
		n += (int(ch) * (i + 5)) % 97
	}
	r := n % 10
	if r <= 3 {
		return "school"
	}
	if r <= 7 {
		return "football"
	}
	return "open"
}

// SchoolWantLabel provides human-readable text for a player's school want.
func SchoolWantLabel(want string) string {
	switch strings.ToLower(strings.TrimSpace(want)) {
	case "school":
		return "Wants to finish school"
	case "football":
		return "Wants football first"
	default:
		return "Has not made his mind up"
	}
}
