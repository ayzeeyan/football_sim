package models

import (
	"testing"
)

// The formation catalogue is rigid: every canonical shape owns exactly
// eleven unique slots and every slot has a position-fit entry.
func TestFormationCatalogueIsRigid(t *testing.T) {
	for _, formation := range []string{
		Formation433, Formation433Attack, Formation4231, Formation442,
		Formation343, Formation352, Formation4141, Formation532,
	} {
		slots := FormationSlots(formation)
		if len(slots) != 11 {
			t.Fatalf("%s has %d slots, want 11", formation, len(slots))
		}
		seen := map[string]bool{}
		for _, slot := range slots {
			if seen[slot] {
				t.Fatalf("%s repeats slot %s", formation, slot)
			}
			seen[slot] = true
			if _, ok := primaryPositionFits[slot]; !ok {
				t.Fatalf("%s uses slot %q with no position-fit entry", formation, slot)
			}
		}
		if NormalizeFormation(formation) != formation {
			t.Fatalf("NormalizeFormation(%q) != %q", formation, formation)
		}
	}
}

// FormationForManager varies the shape per club within a philosophy while
// staying deterministic: the same club always fields the same shape.
func TestFormationForManagerVariesPerClubAndIsDeterministic(t *testing.T) {
	styles := []string{"high_press", "possession", "low_block", "free_flowing"}
	for _, style := range styles {
		shapes := map[string]bool{}
		for _, clubID := range []string{"EPL-ARS", "EPL-CHE", "LAL-RMA", "LAL-BAR", "BUN-BAY", "BUN-DOR", "SER-JUV", "SER-MIL", "FRA-PSG", "FRA-LYO", "EPL-LIV", "EPL-MCI"} {
			first := FormationForManager(style, clubID)
			again := FormationForManager(style, clubID)
			if first != again {
				t.Fatalf("FormationForManager(%s, %s) is not deterministic: %q vs %q", style, clubID, first, again)
			}
			variants := styleFormationVariants[style]
			found := false
			for _, v := range variants {
				if v == first {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("FormationForManager(%s, %s) returned %q, not a variant of the style", style, clubID, first)
			}
			shapes[first] = true
		}
		if len(shapes) < 2 {
			t.Fatalf("style %s produced a single shape across 12 clubs — no variety", style)
		}
	}
	// Unknown styles fall back to the legacy style mapping.
	if FormationForManager("unknown-style", "EPL-ARS") != FormationForStyle("unknown-style") {
		t.Fatal("unknown style must fall back to FormationForStyle")
	}
}
