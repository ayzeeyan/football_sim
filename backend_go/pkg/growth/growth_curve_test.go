package growth

import (
	"math/rand"
	"testing"
)

// canonicalProdigyTestDef represents a canonical wonderkid definition for test verification.
type canonicalProdigyTestDef struct {
	id        string
	name      string
	startOVR  int
	potential int
}

// 12 Canonical Outfield Franchise Wonderkids from dataset / prodigies.go
var canonicalTestProdigies = []canonicalProdigyTestDef{
	{"WK_Venjamin_Valerio", "Venjamin Valerio", 78, 96},
	{"WK_Maverick_Cantalejo", "Maverick Cantalejo", 77, 95},
	{"WK_Yeshua_Gocotano", "Yeshua Emmanuel Gocotano", 75, 93},
	{"WK_Izyan_Bantol", "Izyan Levin Bantol", 76, 95},
	{"WK_James_Rizon", "James Bernard Rizon", 76, 94},
	{"WK_Reid_Libatan", "Reid Randell Libatan", 76, 95},
	{"WK_Ashle_Baguio", "Ashle Zylle Baguio", 75, 95},
	{"WK_Cliergy_Lanticse", "Cliergy Jave Lanticse", 75, 94},
	{"WK_Ezail_Zamora", "Ezail Zamora", 77, 96},
	{"WK_Earl_Hernando", "Earl Josh Hernando", 75, 94},
	{"WK_Rich_Suico", "Rich Lorenz Suico", 75, 94},
	{"WK_Jhed_Guinita", "Jhed Anthony Guinita", 75, 94},
}

// simulateSeasonMatches executes a full 44-week season for a wonderkid with regular starts.
func simulateSeasonMatches(ge *GrowthEngine, id, name string, pot, age int, rng *rand.Rand) (int, int) {
	examWeeks := map[int]bool{12: true, 13: true, 24: true, 25: true, 32: true, 33: true}
	appearances := 0

	mentorOVR := 82
	mentorName := "Senior Captain"
	pers := "dedicated_pro"
	ge.SetMentorship(id, "SENIOR_1", mentorName, mentorOVR, pers)

	for mw := 1; mw <= 44; mw++ {
		ge.ReplenishTrainingEnergy()
		ge.SimulatePubertyCycle(id, mw)

		ge.ApplyMentorshipTick(id, name, MentorshipOptions{
			MentorName:  &mentorName,
			MentorOVR:   &mentorOVR,
			Personality: &pers,
		})

		// Middle school exams (MW 12, 13, 24, 25, 32, 33) apply only when age <= 14
		isExamWeek := age <= 14 && examWeeks[mw]
		if !isExamWeek {
			appearances++
			rating := 6.8 + rng.Float64()*(8.0-6.8)
			goals := 0
			if rng.Float64() < 0.35 {
				goals = 1
			}
			assists := 0
			if rng.Float64() < 0.22 {
				assists = 1
			}
			ge.ApplyMatchXP(id, name, "FWD", rating, goals, assists)
		}
	}

	startOVR := ge.CalculateOVR(id, "FWD")
	endOVR := ge.ApplySeasonalGrowth(id, age, appearances, pot, "FWD", startOVR)
	return appearances, endOVR
}

// TestWonderkid_SingleSeasonGrowthCurve verifies that simulating a full 44-week season
// with regular starts results in a wonderkid gaining +2 to +4 OVR (never exceeding +5 OVR).
func TestWonderkid_SingleSeasonGrowthCurve(t *testing.T) {
	for _, wk := range canonicalTestProdigies {
		for seed := int64(1001); seed <= 1005; seed++ {
			ge := NewGrowthEngine(seed)
			rng := rand.New(rand.NewSource(seed))

			ge.RegisterProdigy(wk.id, wk.name, 14, 168.0, 58.0, "FWD", wk.startOVR, wk.potential, 19)

			apps, endOVR := simulateSeasonMatches(ge, wk.id, wk.name, wk.potential, 14, rng)
			gain := endOVR - wk.startOVR

			if gain < 2 || gain > 4 {
				t.Errorf("[%s seed %d] Expected season gain in [+2, +4], got +%d (from %d to %d, apps %d)",
					wk.name, seed, gain, wk.startOVR, endOVR, apps)
			}
			if gain > 5 {
				t.Fatalf("[%s seed %d] CRITICAL VIOLATION: Season gain +%d exceeded hard ceiling of +5!",
					wk.name, seed, gain)
			}
			if endOVR > wk.potential {
				t.Fatalf("[%s seed %d] CRITICAL VIOLATION: End OVR %d exceeded potential %d!",
					wk.name, seed, endOVR, wk.potential)
			}
		}
	}
}

// TestWonderkid_MultiYearTrajectory verifies that wonderkids develop along a realistic multi-year
// trajectory reaching ~79–82 OVR by age 16, ~85–88 OVR by age 18, and approaching canonical
// 93–96 ceiling in their early 20s.
func TestWonderkid_MultiYearTrajectory(t *testing.T) {
	for seed := int64(2001); seed <= 2005; seed++ {
		ge := NewGrowthEngine(seed)
		rng := rand.New(rand.NewSource(seed))

		// Test typical 75 OVR starting wonderkid
		wk := canonicalProdigyTestDef{"WK_Test_Career", "Test Prodigy", 75, 95}
		ge.RegisterProdigy(wk.id, wk.name, 14, 168.0, 58.0, "FWD", wk.startOVR, wk.potential, 19)

		ovrHistory := make(map[int]int)
		ovrHistory[14] = wk.startOVR

		for age := 14; age <= 21; age++ {
			seasonStartOVR := ge.CalculateOVR(wk.id, "FWD")
			_, endOVR := simulateSeasonMatches(ge, wk.id, wk.name, wk.potential, age, rng)
			seasonGain := endOVR - seasonStartOVR
			if seasonGain > 5 {
				t.Fatalf("[Seed %d, Age %d] Single-season gain +%d exceeded +5 ceiling!", seed, age, seasonGain)
			}

			bio := ge.Biometrics[wk.id]
			bio.Age = age + 1
			ge.ResetYearlyHeightTaken(wk.id)
			ovrHistory[age+1] = endOVR
		}

		// Milestone 1: Age 16 (after 2 seasons): ~79–82 OVR
		ovr16 := ovrHistory[16]
		if ovr16 < 79 || ovr16 > 83 {
			t.Errorf("[Seed %d] At age 16 expected ~79–82 OVR, got %d", seed, ovr16)
		}

		// Milestone 2: Age 18 (after 4 seasons): ~85–88 OVR
		ovr18 := ovrHistory[18]
		if ovr18 < 85 || ovr18 > 89 {
			t.Errorf("[Seed %d] At age 18 expected ~85–88 OVR, got %d", seed, ovr18)
		}

		// Milestone 3: Early 20s (Age 21-22): approaching canonical ceiling [93, 96]
		ovr22 := ovrHistory[22]
		if ovr22 < 91 || ovr22 > wk.potential {
			t.Errorf("[Seed %d] In early 20s expected approaching ceiling (>= 91, <= %d), got %d",
				seed, wk.potential, ovr22)
		}
	}
}

// TestWonderkid_PotentialBoundsStrictness verifies that all canonical wonderkid potentials
// are strictly within [93, 96] and cannot be exceeded even under extreme circumstances.
func TestWonderkid_PotentialBoundsStrictness(t *testing.T) {
	for _, wk := range canonicalTestProdigies {
		if wk.potential < 93 || wk.potential > 96 {
			t.Errorf("Prodigy %s potential %d is out of strict range [93, 96]", wk.name, wk.potential)
		}

		ge := NewGrowthEngine(42)
		ge.RegisterProdigy(wk.id, wk.name, 14, 168.0, 58.0, "FWD", wk.startOVR, wk.potential, 19)

		// Overload with 100 consecutive matches with 10.0 ratings and hat-tricks
		for i := 0; i < 100; i++ {
			ge.ApplyMatchXP(wk.id, wk.name, "FWD", 10.0, 3, 3)
		}

		// Check CalculateOVR never exceeds potential
		ovr := ge.CalculateOVR(wk.id, "FWD")
		if ovr > wk.potential {
			t.Fatalf("Prodigy %s OVR %d exceeded potential %d after extreme matches!", wk.name, ovr, wk.potential)
		}

		// Seasonal growth also never exceeds potential
		seasonalOVR := ge.ApplySeasonalGrowth(wk.id, 18, 50, wk.potential, "FWD", ovr)
		if seasonalOVR > wk.potential {
			t.Fatalf("Prodigy %s seasonal OVR %d exceeded potential %d!", wk.name, seasonalOVR, wk.potential)
		}
	}
}

// TestWonderkid_AppearanceThresholds verifies the calibrated appearance bump thresholds in ApplySeasonalGrowth.
func TestWonderkid_AppearanceThresholds(t *testing.T) {
	ge := NewGrowthEngine(99)
	ge.RegisterProdigy("WK_App_Test", "App Tester", 16, 175.0, 68.0, "FWD", 75, 95, 19)

	// Sub-threshold (< 12 apps): bump = 0
	ovr0 := ge.ApplySeasonalGrowth("WK_App_Test", 16, 5, 95, "FWD", 75)
	if ovr0 != 75 {
		t.Errorf("Expected 0 bump for 5 apps (< 12), got %d (want 75)", ovr0)
	}

	// Moderate threshold (12-24 apps): bump = 1
	ovr1 := ge.ApplySeasonalGrowth("WK_App_Test", 16, 15, 95, "FWD", 75)
	if ovr1 != 76 {
		t.Errorf("Expected +1 bump for 15 apps, got %d (want 76)", ovr1)
	}

	// Regular starter threshold (25-31 apps): bump = 1
	ovr2 := ge.ApplySeasonalGrowth("WK_App_Test", 16, 28, 95, "FWD", 76)
	if ovr2 != 77 {
		t.Errorf("Expected +1 bump for 28 apps, got %d (want 77)", ovr2)
	}

	// Heavy starter threshold (>= 32 apps and OVR < 88): bump = 2
	ovr3 := ge.ApplySeasonalGrowth("WK_App_Test", 16, 38, 95, "FWD", 77)
	if ovr3 != 79 {
		t.Errorf("Expected +2 bump for 38 apps with OVR < 88, got %d (want 79)", ovr3)
	}

	// Heavy starter threshold with OVR >= 88: bump = 1
	ge.RegisterProdigy("WK_High_OVR", "High OVR Tester", 19, 180.0, 75.0, "FWD", 89, 95, 19)
	ovr4 := ge.ApplySeasonalGrowth("WK_High_OVR", 19, 38, 95, "FWD", 89)
	if ovr4 != 90 {
		t.Errorf("Expected +1 bump for 38 apps with OVR >= 88, got %d (want 90)", ovr4)
	}
}

// TestEmpirical_NormalSeason_75OVR tests 1,000 normal seasons with standard starter performance
// (38 appearances, 6.8-8.0 ratings, 0.35 goals/match, 82 OVR mentor)
// and strictly verifies every season gain is in [+2, +4] (no +5 in normal play).
func TestEmpirical_NormalSeason_75OVR(t *testing.T) {
	minGain := 99
	maxGain := -99
	gains := make(map[int]int)

	for seed := int64(1); seed <= 1000; seed++ {
		ge := NewGrowthEngine(seed)
		rng := rand.New(rand.NewSource(seed))

		ge.RegisterProdigy("WK_Norm_75", "Normal Wonderkid", 14, 168.0, 58.0, "FWD", 75, 95, 19)

		apps, endOVR := simulateSeasonMatches(ge, "WK_Norm_75", "Normal Wonderkid", 95, 14, rng)
		gain := endOVR - 75

		if seed == 880 {
			t.Logf("[Seed 880 Debug] startOVR=75, endOVR=%d, gain=%d, apps=%d", endOVR, gain, apps)
		}

		if gain < minGain {
			minGain = gain
		}
		if gain > maxGain {
			maxGain = gain
		}
		gains[gain]++

		if gain < 2 || gain > 4 {
			t.Errorf("[Seed %d] Gain +%d outside expected normal [+2, +4] (apps: %d, endOVR: %d)",
				seed, gain, apps, endOVR)
		}
		if gain >= 5 {
			t.Errorf("[Seed %d] Gain +%d breached normal corridor (must be <= +4, no +5 in normal play)",
				seed, gain)
		}
		if endOVR > 95 {
			t.Errorf("[Seed %d] End OVR %d exceeded potential 95!", seed, endOVR)
		}
	}

	t.Logf("=== TestEmpirical_NormalSeason_75OVR Results (1000 runs) ===")
	t.Logf("Min Gain: +%d, Max Gain: +%d", minGain, maxGain)
	for g := 2; g <= 5; g++ {
		cnt := gains[g]
		pct := float64(cnt) / 1000.0 * 100.0
		t.Logf("  Gain +%d: %d (%.1f%%)", g, cnt, pct)
	}

	if maxGain > 4 {
		t.Fatalf("CRITICAL FAILURE: Max normal gain was +%d, exceeding normal corridor [+2, +4]", maxGain)
	}
}

// TestEmpirical_AdversarialCeiling_SingleSeason tests 500 adversarial extreme seasons
// (44 appearances, 10.0 ratings, 40 goals, 25 assists, 92 OVR mentor)
// and strictly verifies that single-season gain NEVER exceeds +5 OVR under any circumstances.
func TestEmpirical_AdversarialCeiling_SingleSeason(t *testing.T) {
	minGain := 99
	maxGain := -99
	gains := make(map[int]int)

	mentorOVR := 92
	mentorName := "Senior Legend"
	pers := "dedicated_pro"
	opts := MatchXPOptions{
		MentorOVR:   &mentorOVR,
		MentorName:  &mentorName,
		Personality: &pers,
	}

	for seed := int64(5001); seed <= 5500; seed++ {
		ge := NewGrowthEngine(seed)
		ge.RegisterProdigy("WK_Adversarial", "Max Prodigy", 14, 168.0, 58.0, "FWD", 75, 95, 19)
		ge.SetMentorship("WK_Adversarial", "SENIOR_M", mentorName, mentorOVR, pers)

		rng := rand.New(rand.NewSource(seed))

		for mw := 1; mw <= 44; mw++ {
			ge.ReplenishTrainingEnergy()
			ge.SimulatePubertyCycle("WK_Adversarial", mw)
			ge.ApplyMentorshipTick("WK_Adversarial", "Max Prodigy", MentorshipOptions{
				MentorName:  &mentorName,
				MentorOVR:   &mentorOVR,
				Personality: &pers,
			})

			goals := 0
			if rng.Float64() < 40.0/44.0 {
				goals = 1
			}
			assists := 0
			if rng.Float64() < 25.0/44.0 {
				assists = 1
			}
			ge.ApplyMatchXP("WK_Adversarial", "Max Prodigy", "FWD", 10.0, goals, assists, opts)
		}

		inSeasonOVR := ge.CalculateOVR("WK_Adversarial", "FWD")
		endOVR := ge.ApplySeasonalGrowth("WK_Adversarial", 14, 44, 95, "FWD", inSeasonOVR)
		gain := endOVR - 75

		if gain < minGain {
			minGain = gain
		}
		if gain > maxGain {
			maxGain = gain
		}
		gains[gain]++

		if gain > 5 {
			t.Fatalf("[Seed %d] VIOLATION: Adversarial extreme season gain was +%d (> +5 ceiling)! in-season OVR: %d, endOVR: %d",
				seed, gain, inSeasonOVR, endOVR)
		}
		if endOVR > 95 {
			t.Fatalf("[Seed %d] VIOLATION: End OVR %d exceeded potential 95!", seed, endOVR)
		}

		if seed == 5389 {
			t.Logf("[Seed 5389 Debug] startOVR=75, inSeasonOVR=%d, endOVR=%d, gain=%d", inSeasonOVR, endOVR, gain)
		}
	}

	t.Logf("=== TestEmpirical_AdversarialCeiling_SingleSeason Results (500 runs) ===")
	t.Logf("Min Gain: +%d, Max Gain: +%d", minGain, maxGain)
	for g := 3; g <= 6; g++ {
		cnt := gains[g]
		pct := float64(cnt) / 500.0 * 100.0
		t.Logf("  Gain +%d: %d (%.1f%%)", g, cnt, pct)
	}

	if maxGain > 5 {
		t.Fatalf("CRITICAL FAILURE: Max adversarial gain was +%d, exceeding hard ceiling of +5", maxGain)
	}
}

// TestWonderkid_HardCeiling_NeverExceeds5_Explicit exhaustively tests that regardless of
// how high in-season OVR reaches (+4, +5, +6, or +10), ApplySeasonalGrowth NEVER exceeds
// seasonStartOVR + 5 under any circumstances.
func TestWonderkid_HardCeiling_NeverExceeds5_Explicit(t *testing.T) {
	testCases := []struct {
		startOVR    int
		inSeasonOVR int
		appearances int
		potential   int
	}{
		{75, 78, 44, 95}, // in-season +3 -> max 79 (+4) or 80 (+5)
		{75, 79, 44, 95}, // in-season +4 (Seed 5389 scenario) -> strictly <= 80 (+5)
		{75, 80, 44, 95}, // in-season +5 -> strictly <= 80 (+5)
		{75, 81, 44, 95}, // in-season +6 -> strictly <= 80 (+5)
		{75, 85, 44, 95}, // in-season +10 -> strictly <= 80 (+5)
		{77, 81, 44, 95}, // in-season +4 for 77 starter -> strictly <= 82 (+5)
		{78, 83, 44, 96}, // in-season +5 for 78 starter -> strictly <= 83 (+5)
	}

	for _, tc := range testCases {
		ge := NewGrowthEngine(9999)
		ge.RegisterProdigy("WK_CeilTest", "Ceiling Tester", 14, 168.0, 58.0, "FWD", tc.startOVR, tc.potential, 19)

		// Nudge to desired in-season OVR
		ge.mu.Lock()
		ge.internalNudgeToOVR("WK_CeilTest", "FWD", tc.inSeasonOVR)
		ge.mu.Unlock()

		calcInSeason := ge.CalculateOVR("WK_CeilTest", "FWD")
		endOVR := ge.ApplySeasonalGrowth("WK_CeilTest", 14, tc.appearances, tc.potential, "FWD", calcInSeason)
		gain := endOVR - tc.startOVR

		if gain > 5 {
			t.Fatalf("[startOVR=%d, inSeasonOVR=%d] VIOLATION: Season gain +%d exceeded +5 ceiling! endOVR=%d",
				tc.startOVR, calcInSeason, gain, endOVR)
		}
		if endOVR > tc.potential {
			t.Fatalf("[startOVR=%d] Exceeded potential %d: got %d", tc.startOVR, tc.potential, endOVR)
		}
	}
}