package models

import (
	"testing"
)

func TestCanonicalProdigyPersonalities(t *testing.T) {
	canonicalExpected := map[string]string{
		"Maverick Cantalejo":       "big_game_performer",
		"Reid Randell Libatan":     "dedicated_pro",
		"Reid Libatan":             "dedicated_pro",
		"Venjamin Valerio":         "academic_dual",
		"Izyan Levin Bantol":       "flamboyant_star",
		"James Bernard Rizon":      "big_game_performer",
		"Yeshua Emmanuel Gocotano": "dedicated_pro",
		"Cliergy Jave Lanticse":    "flamboyant_star",
		"Earl Josh Hernando":       "dedicated_pro",
		"Ezail Zamora":             "academic_dual",
		"Ashle Zylle Baguio":       "flamboyant_star",
		"Jhed Anthony Guinita":     "big_game_performer",
		"Rich Lorenz Suico":        "dedicated_pro",
	}

	for name, expected := range canonicalExpected {
		actual := PersonalityFor(name)
		if actual != expected {
			t.Errorf("PersonalityFor(%q) = %q; want %q", name, actual, expected)
		}
	}
}

func TestPersonalityFallback(t *testing.T) {
	// Random name not in canonical list
	p1 := PersonalityFor("Generic Player A")
	p2 := PersonalityFor("Generic Player A")
	if p1 != p2 {
		t.Errorf("PersonalityFor must be deterministic: %q != %q", p1, p2)
	}

	validArchetypes := map[string]bool{
		"dedicated_pro":      true,
		"flamboyant_star":    true,
		"academic_dual":      true,
		"big_game_performer": true,
	}
	if !validArchetypes[p1] {
		t.Errorf("PersonalityFor returned unrecognized archetype: %q", p1)
	}
}

func TestArchetypeForKey(t *testing.T) {
	arch := ArchetypeForKey("flamboyant_star")
	if arch.Badge != "Flamboyant Star" || arch.Icon != "sparkle" {
		t.Errorf("ArchetypeForKey(flamboyant_star) incorrect: %+v", arch)
	}

	fallback := ArchetypeForKey("unknown_key")
	if fallback.Key != "dedicated_pro" {
		t.Errorf("ArchetypeForKey(unknown) should default to dedicated_pro; got %q", fallback.Key)
	}
}

func TestSchoolWant(t *testing.T) {
	want := SchoolWantFor("Maverick Cantalejo")
	if want != "school" && want != "football" && want != "open" {
		t.Errorf("SchoolWantFor returned invalid state: %q", want)
	}

	// Determinism check
	for i := 0; i < 5; i++ {
		if SchoolWantFor("Maverick Cantalejo") != want {
			t.Errorf("SchoolWantFor is not deterministic across invocations")
		}
	}

	if SchoolWantLabel("school") != "Wants to finish school" {
		t.Errorf("SchoolWantLabel(school) unexpected: %q", SchoolWantLabel("school"))
	}
	if SchoolWantLabel("football") != "Wants football first" {
		t.Errorf("SchoolWantLabel(football) unexpected: %q", SchoolWantLabel("football"))
	}
	if SchoolWantLabel("open") != "Has not made his mind up" {
		t.Errorf("SchoolWantLabel(open) unexpected: %q", SchoolWantLabel("open"))
	}
}
