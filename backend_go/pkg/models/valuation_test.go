package models

import (
	"math"
	"testing"
)

func TestBaselineValue(t *testing.T) {
	// 1. Standard player at baseline (OVR 65, age 25, non-wonderkid)
	val65 := BaselineValue(65, 25, false)
	if val65 != 11000000 {
		t.Errorf("BaselineValue(65, 25, false) = %d; want 11000000", val65)
	}

	// 2. Youth boost (age <= 21 gets 1.15x)
	valYouth := BaselineValue(65, 20, false)
	expectedYouth := int64(11000000.0 * 1.15)
	if valYouth != expectedYouth {
		t.Errorf("BaselineValue(65, 20, false) = %d; want %d", valYouth, expectedYouth)
	}

	// 3. Wonderkid boost (1.35x)
	valWK := BaselineValue(65, 24, true)
	expectedWK := int64(11000000.0 * 1.35)
	if valWK != expectedWK {
		t.Errorf("BaselineValue(65, 24, true) = %d; want %d", valWK, expectedWK)
	}

	// 4. Combined Youth & Wonderkid (age 14, OVR 75)
	valCombined := BaselineValue(75, 14, true)
	if valCombined <= 35000000 || valCombined >= 42000000 {
		t.Errorf("BaselineValue(75, 14, true) = %d; want ~38.96M", valCombined)
	}

	// 5. Veteran depreciation (age >= 33, 0.75^(age-32))
	val32 := BaselineValue(70, 32, false)
	val33 := BaselineValue(70, 33, false)
	expected33 := int64(math.Round(float64(val32) * 0.75))
	if val33 != expected33 {
		t.Errorf("BaselineValue(70, 33, false) = %d; want %d (val32: %d)", val33, expected33, val32)
	}

	// 6. Absolute floor: very low rating returns at least 500,000
	valFloor := BaselineValue(30, 40, false)
	if valFloor < 500000 {
		t.Errorf("BaselineValue(30, 40, false) = %d; want at least 500000", valFloor)
	}
}

func TestClampValue(t *testing.T) {
	// Anchor for OVR 65, age 25, false = 11,000,000
	// Low corridor = 11M * 0.35 = 3,850,000
	// High corridor = 11M * 3.0 = 33,000,000

	// 1. Normal value within corridor should remain untouched
	normal := ClampValue(15000000, 65, 25, false)
	if normal != 15000000 {
		t.Errorf("ClampValue(15M) = %d; want 15000000", normal)
	}

	// 2. Below corridor floor should clamp to 0.35 * anchor
	below := ClampValue(1000000, 65, 25, false)
	if below != 3850000 {
		t.Errorf("ClampValue(1M) = %d; want 3850000", below)
	}

	// 3. Above corridor ceiling should clamp to 3.0 * anchor
	above := ClampValue(50000000, 65, 25, false)
	if above != 33000000 {
		t.Errorf("ClampValue(50M) = %d; want 33000000", above)
	}

	// 4. Runaway / Trillion value should never exceed absolute ceiling €500M
	runaway := ClampValue(1000000000000, 95, 25, false)
	if runaway > 500000000 {
		t.Errorf("ClampValue(1T) = %d; must not exceed 500,000,000", runaway)
	}

	// 5. Negative value should be clamped safely above €300k
	negative := ClampValue(-50000000, 65, 25, false)
	if negative < 300000 {
		t.Errorf("ClampValue(-50M) = %d; must be at least 300,000", negative)
	}
}

func TestClampPlayer(t *testing.T) {
	// Nil safe
	if ClampPlayer(nil) != 0 {
		t.Errorf("ClampPlayer(nil) should return 0")
	}

	p := &Player{
		OVR:               70,
		Age:               24,
		MarketValueEUR:    9999999999,
		UniverseWonderkid: false,
	}
	res := ClampPlayer(p)
	if p.MarketValueEUR != res {
		t.Errorf("ClampPlayer should update player's MarketValueEUR in place")
	}
	anchor := BaselineValue(70, 24, false)
	maxCorridor := int64(float64(anchor) * 3.0)
	if res > maxCorridor {
		t.Errorf("Clamped value %d exceeded high corridor %d", res, maxCorridor)
	}
}

func TestWageForOVR(t *testing.T) {
	// OVR <= 60
	w60 := WageForOVR(60)
	if w60 != 2000 {
		t.Errorf("WageForOVR(60) = %d; want 2000", w60)
	}
	w50 := WageForOVR(50)
	if w50 != 2000 {
		t.Errorf("WageForOVR(50) = %d; want 2000", w50)
	}

	// OVR 70
	w70 := WageForOVR(70)
	if w70 <= 15000 || w70 >= 16500 {
		t.Errorf("WageForOVR(70) = %d; want ~15851", w70)
	}
}

func TestFormatCurrency(t *testing.T) {
	tests := []struct {
		val  int64
		want string
	}{
		{0, "€0"},
		{500, "€500"},
		{500000, "€500K"},
		{1500000, "€1.5M"},
		{35000000, "€35.0M"},
		{1200000000, "€1.20B"},
		{2500000000000, "€2.50T"},
	}

	for _, tt := range tests {
		got := FormatCurrency(tt.val)
		if got != tt.want {
			t.Errorf("FormatCurrency(%d) = %q; want %q", tt.val, got, tt.want)
		}
	}
}

func TestFormatWage(t *testing.T) {
	tests := []struct {
		val  int64
		want string
	}{
		{25000, "€25k/wk"},
		{1250000, "€1.25M/wk"},
	}

	for _, tt := range tests {
		got := FormatWage(tt.val)
		if got != tt.want {
			t.Errorf("FormatWage(%d) = %q; want %q", tt.val, got, tt.want)
		}
	}
}
