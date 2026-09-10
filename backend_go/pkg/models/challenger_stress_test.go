package models

import (
	"fmt"
	"math"
	"sync"
	"testing"
)

// TestChallenger_ValuationClamping_ExtremeNegatives verifies that extreme negative valuations
// (-€100B, -€100T, MinInt64, -1) never produce negative values and are strictly clamped >= €300k.
func TestChallenger_ValuationClamping_ExtremeNegatives(t *testing.T) {
	extremeNegatives := []int64{
		-100_000_000_000,     // -€100B
		-100_000_000_000_000, // -€100T
		math.MinInt64,        // Minimum possible int64
		-1_000_000_000_000_000,
		-500_000_000,
		-300_000,
		-1,
		0,
	}

	ovrList := []int{40, 50, 60, 65, 75, 85, 90, 95, 99}
	ageList := []int{14, 15, 18, 21, 25, 30, 32, 33, 40, 45}
	wkList := []bool{false, true}

	for _, current := range extremeNegatives {
		for _, ovr := range ovrList {
			for _, age := range ageList {
				for _, wk := range wkList {
					clamped := ClampValue(current, ovr, age, wk)
					if clamped < 300_000 {
						t.Fatalf("VIOLATION: ClampValue(%d, OVR=%d, Age=%d, WK=%v) = %d; MUST be >= 300,000",
							current, ovr, age, wk, clamped)
					}
					if clamped > 500_000_000 {
						t.Fatalf("VIOLATION: ClampValue(%d, OVR=%d, Age=%d, WK=%v) = %d; MUST be <= 500,000,000",
							current, ovr, age, wk, clamped)
					}
				}
			}
		}
	}
}

// TestChallenger_ValuationClamping_AbsurdPositives verifies that absurd positive valuations
// (€100T, 1 quintillion, MaxInt64) never exceed the absolute ceiling of €500M.
func TestChallenger_ValuationClamping_AbsurdPositives(t *testing.T) {
	absurdPositives := []int64{
		500_000_001,
		1_000_000_000,        // €1B
		100_000_000_000,      // €100B
		100_000_000_000_000,  // €100T
		1_000_000_000_000_000_000,
		math.MaxInt64,        // Maximum possible int64
	}

	ovrList := []int{40, 50, 65, 75, 85, 90, 95, 99}
	ageList := []int{14, 18, 21, 25, 32, 33, 40, 45}
	wkList := []bool{false, true}

	for _, current := range absurdPositives {
		for _, ovr := range ovrList {
			for _, age := range ageList {
				for _, wk := range wkList {
					clamped := ClampValue(current, ovr, age, wk)
					if clamped > 500_000_000 {
						t.Fatalf("VIOLATION: ClampValue(%d, OVR=%d, Age=%d, WK=%v) = %d; MUST NOT exceed 500,000,000",
							current, ovr, age, wk, clamped)
					}
					if clamped < 300_000 {
						t.Fatalf("VIOLATION: ClampValue(%d, OVR=%d, Age=%d, WK=%v) = %d; MUST be >= 300,000",
							current, ovr, age, wk, clamped)
					}
				}
			}
		}
	}
}

// TestChallenger_ValuationClamping_CorridorBounds tests that within valid ranges,
// values are properly clamped to [0.35 * anchor, 3.0 * anchor] corridor.
func TestChallenger_ValuationClamping_CorridorBounds(t *testing.T) {
	cases := []struct {
		ovr int
		age int
		wk  bool
	}{
		{65, 25, false}, // anchor = 11M, low = 3.85M, high = 33M
		{75, 14, true},  // anchor = ~38.96M, low = ~13.6M, high = ~116.9M
		{90, 28, false}, // anchor = ~83.2M, low = ~29.1M, high = ~249.6M
		{55, 38, false}, // anchor = 500k, low = 175k -> 300k, high = 1.5M
	}

	for _, c := range cases {
		anchor := float64(BaselineValue(c.ovr, c.age, c.wk))
		lowCorridor := int64(math.Round(anchor * 0.35))
		highCorridor := int64(math.Round(anchor * 3.0))

		// 1. Value within corridor: preserved
		midVal := int64(anchor)
		if got := ClampValue(midVal, c.ovr, c.age, c.wk); got != midVal {
			t.Errorf("ClampValue inside corridor: got %d, want %d", got, midVal)
		}

		// 2. Value below corridor: clamped to max(300k, lowCorridor)
		veryLow := int64(100)
		expectedLow := lowCorridor
		if expectedLow < 300_000 {
			expectedLow = 300_000
		}
		if got := ClampValue(veryLow, c.ovr, c.age, c.wk); got != expectedLow {
			t.Errorf("ClampValue below corridor: got %d, want %d", got, expectedLow)
		}

		// 3. Value above corridor: clamped to min(500M, highCorridor)
		veryHigh := int64(999_999_999_999)
		expectedHigh := highCorridor
		if expectedHigh > 500_000_000 {
			expectedHigh = 500_000_000
		}
		if got := ClampValue(veryHigh, c.ovr, c.age, c.wk); got != expectedHigh {
			t.Errorf("ClampValue above corridor: got %d, want %d", got, expectedHigh)
		}
	}
}

// TestChallenger_ClampPlayer_Stress verifies mutating Player through ClampPlayer
// across extreme market values, nil safety, and boundary properties.
func TestChallenger_ClampPlayer_Stress(t *testing.T) {
	// 1. Nil safety
	if got := ClampPlayer(nil); got != 0 {
		t.Fatalf("ClampPlayer(nil) must return 0, got %d", got)
	}

	// 2. Extreme negative player value
	pNeg := &Player{
		PlayerID:          "P_NEG",
		OVR:               70,
		Age:               20,
		UniverseWonderkid: true,
		MarketValueEUR:    -100_000_000_000,
	}
	resNeg := ClampPlayer(pNeg)
	if resNeg < 300_000 || resNeg > 500_000_000 {
		t.Fatalf("ClampPlayer negative value resulted in %d, out of [300k, 500M]", resNeg)
	}
	if pNeg.MarketValueEUR != resNeg {
		t.Fatalf("Player MarketValueEUR not updated in place: got %d, want %d", pNeg.MarketValueEUR, resNeg)
	}

	// 3. Extreme positive player value
	pPos := &Player{
		PlayerID:          "P_POS",
		OVR:               95,
		Age:               27,
		UniverseWonderkid: false,
		MarketValueEUR:    math.MaxInt64,
	}
	resPos := ClampPlayer(pPos)
	if resPos > 500_000_000 || resPos < 300_000 {
		t.Fatalf("ClampPlayer positive value resulted in %d, out of [300k, 500M]", resPos)
	}
	if pPos.MarketValueEUR != resPos {
		t.Fatalf("Player MarketValueEUR not updated in place: got %d, want %d", pPos.MarketValueEUR, resPos)
	}
}

// TestChallenger_Valuation_ExhaustiveGrid runs an exhaustive combinatorial grid
// testing every OVR in [40..99] x Age in [14..45] x Wonderkid in [false, true].
func TestChallenger_Valuation_ExhaustiveGrid(t *testing.T) {
	testValuations := []int64{
		math.MinInt64,
		-100_000_000_000,
		-1,
		0,
		1,
		100_000,
		300_000,
		1_000_000,
		10_000_000,
		100_000_000,
		500_000_000,
		100_000_000_000_000,
		math.MaxInt64,
	}

	for ovr := 40; ovr <= 99; ovr++ {
		for age := 14; age <= 45; age++ {
			for _, wk := range []bool{false, true} {
				base := BaselineValue(ovr, age, wk)
				if base < 500_000 {
					t.Fatalf("BaselineValue(%d, %d, %v) = %d; MUST be >= 500,000", ovr, age, wk, base)
				}

				for _, tv := range testValuations {
					clamped := ClampValue(tv, ovr, age, wk)
					if clamped < 300_000 || clamped > 500_000_000 {
						t.Fatalf("ClampValue(%d, %d, %d, %v) = %d; OUT OF [300k, 500M] BOUNDS",
							tv, ovr, age, wk, clamped)
					}
				}
			}
		}
	}
}

// TestChallenger_FormatCurrency_Extremes tests formatting across negative, zero, and huge values.
func TestChallenger_FormatCurrency_Extremes(t *testing.T) {
	cases := []struct {
		val      int64
		expected string
	}{
		{-100, "€-100"},
		{0, "€0"},
		{999, "€999"},
		{1000, "€1K"},
		{300000, "€300K"},
		{500000000, "€500.0M"},
		{1000000000000, "€1.00T"},
		{100000000000000, "€100.00T"},
	}

	for _, c := range cases {
		got := FormatCurrency(c.val)
		if got != c.expected {
			t.Errorf("FormatCurrency(%d) = %q, want %q", c.val, got, c.expected)
		}
	}
}

// TestChallenger_Models_Concurrency stress-tests models functions concurrently across goroutines.
func TestChallenger_Models_Concurrency(t *testing.T) {
	var wg sync.WaitGroup
	workers := 50
	iterations := 200

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				ovr := 40 + (i % 60)
				age := 14 + (i % 32)
				wk := (i % 2) == 0

				base := BaselineValue(ovr, age, wk)
				if base < 500_000 {
					panic(fmt.Sprintf("base < 500k: %d", base))
				}

				clamped := ClampValue(int64(i)*1_000_000, ovr, age, wk)
				if clamped < 300_000 || clamped > 500_000_000 {
					panic(fmt.Sprintf("clamped out of bounds: %d", clamped))
				}

				FormatCurrency(clamped)
				FormatWage(WageForOVR(ovr))
			}
		}(w)
	}
	wg.Wait()
}
