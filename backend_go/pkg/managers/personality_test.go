package managers

import (
	"testing"
)

// The personality model is a post-training artifact: inference only, no
// runtime state. These tests pin the contract.
func TestPersonalityForManagerIsDeterministicAndVaried(t *testing.T) {
	styles := []string{"high_press", "possession", "low_block", "free_flowing"}
	names := []string{"Piotr Lewandowski", "Marco Bianchi", "Hans Gruber", "Pierre Laurent", "Eduardo Silva", "Karl Fischer"}
	for _, style := range styles {
		seen := map[string]bool{}
		for _, name := range names {
			first := PersonalityForManager(style, name)
			again := PersonalityForManager(style, name)
			if first != again {
				t.Fatalf("PersonalityForManager(%s, %s) is not deterministic", style, name)
			}
			// Every fitted profile is fully specified.
			if first.Label == "" || first.Description == "" || first.Key == "" {
				t.Fatalf("incomplete profile for %s/%s: %+v", style, name, first)
			}
			if first.PressIntensity < 1 || first.PressIntensity > 10 || first.Patience < 1 || first.Patience > 10 {
				t.Fatalf("out-of-range ints for %s/%s: %+v", style, name, first)
			}
			for _, v := range []float64{first.Rotation, first.YouthTrust, first.TransferAggression, first.RiskAppetite} {
				if v < 0 || v > 1 {
					t.Fatalf("out-of-range trait for %s/%s: %+v", style, name, first)
				}
			}
			// The profile must belong to the style's fitted set.
			found := false
			for _, p := range personalityTable[style] {
				if p.Key == first.Key {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("profile %q is not a %s variant", first.Key, style)
			}
			seen[first.Key] = true
		}
		if len(seen) < 2 {
			t.Fatalf("style %s produced a single personality across %d managers — no variety", style, len(names))
		}
	}
}

// Unknown philosophies get the balanced neutral profile, never a panic.
func TestPersonalityForUnknownStyleIsNeutral(t *testing.T) {
	p := PersonalityForManager("does-not-exist", "Anyone")
	if p.Key != "neutral" || p.Label != "The Balanced Hand" {
		t.Fatalf("unknown style must return the neutral profile, got %+v", p)
	}
}

// The fitted table is closed and small: three archetypes per philosophy,
// twelve total — the whole model is a few hundred bytes of constants.
func TestPersonalityTableIsClosed(t *testing.T) {
	if len(personalityTable) != 4 {
		t.Fatalf("personality table covers %d styles, want 4", len(personalityTable))
	}
	total := 0
	for style, profiles := range personalityTable {
		if len(profiles) != 3 {
			t.Fatalf("style %s has %d archetypes, want 3", style, len(profiles))
		}
		total += len(profiles)
	}
	if total != 12 {
		t.Fatalf("personality table has %d archetypes, want 12", total)
	}
}
