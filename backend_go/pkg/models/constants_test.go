package models

import (
	"testing"
)

func TestGetPositionCategory(t *testing.T) {
	tests := []struct {
		pos  string
		want string
	}{
		{"GK", "GK"},
		{"CB", "DEF"},
		{"LB", "DEF"},
		{"RB", "DEF"},
		{"LWB", "DEF"},
		{"RWB", "DEF"},
		{"CDM", "MID"},
		{"CM", "MID"},
		// CRITICAL: CAM is explicitly categorized as FWD in the simulation engine
		{"CAM", "FWD"},
		{"CF", "FWD"},
		{"ST", "FWD"},
		{"LW", "FWD"},
		{"RW", "FWD"},
	}

	for _, tt := range tests {
		got := GetPositionCategory(tt.pos)
		if got != tt.want {
			t.Errorf("GetPositionCategory(%q) = %q; want %q", tt.pos, got, tt.want)
		}
	}
}

func TestExamWeeks(t *testing.T) {
	expectedWeeks := []int{12, 13, 24, 25, 32, 33}
	for _, w := range expectedWeeks {
		if !IsExamWeek(w) {
			t.Errorf("IsExamWeek(%d) = false; want true", w)
		}
	}

	nonExamWeeks := []int{1, 5, 11, 14, 20, 26, 31, 34, 38}
	for _, w := range nonExamWeeks {
		if IsExamWeek(w) {
			t.Errorf("IsExamWeek(%d) = true; want false", w)
		}
	}
}

func TestPositionOptions(t *testing.T) {
	camOpts := PositionOptions("CAM")
	if len(camOpts) != 2 || camOpts[0] != "CF" || camOpts[1] != "ST" {
		t.Errorf("PositionOptions(CAM) = %v; want [CF, ST]", camOpts)
	}

	stOpts := PositionOptions("ST")
	if len(stOpts) != 2 || stOpts[0] != "CF" || stOpts[1] != "LW" {
		t.Errorf("PositionOptions(ST) = %v; want [CF, LW]", stOpts)
	}

	noneOpts := PositionOptions("CB")
	if len(noneOpts) != 0 {
		t.Errorf("PositionOptions(CB) = %v; want empty", noneOpts)
	}
}

func TestKitColors(t *testing.T) {
	// Canonical club
	p, s := KitColorsForClub("ARS")
	if p != [3]uint8{219, 0, 7} || s != [3]uint8{255, 255, 255} {
		t.Errorf("KitColorsForClub(ARS) = (%v, %v); unexpected", p, s)
	}

	// Generated club
	pGen, sGen := KitColorsForClub("XYZ_CUSTOM")
	if sGen != [3]uint8{255, 255, 255} {
		t.Errorf("Generated secondary color should be white; got %v", sGen)
	}
	if pGen == [3]uint8{0, 0, 0} {
		t.Errorf("Generated primary color should not be black")
	}
}
