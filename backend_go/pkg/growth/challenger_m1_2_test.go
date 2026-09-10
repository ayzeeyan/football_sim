package growth

import (
	"fmt"
	"math/rand"
	"testing"
)

// referenceLegacySeasonalGrowth computes the exact legacy OVR bump for players without attrs (attrs == nil).
func referenceLegacySeasonalGrowth(age, appearances, potential int, currentOVR ...int) int {
	if age >= 25 {
		if len(currentOVR) > 0 {
			return currentOVR[0]
		}
		return 75
	}

	var bump int
	if appearances >= 20 {
		bump = 3
	} else if appearances >= 8 {
		bump = 2
	} else {
		bump = 1
	}

	base := 70
	if len(currentOVR) > 0 {
		base = currentOVR[0]
	}
	return minInt(potential, base+bump)
}

// TestChallenger2_GenericPlayer_ApplySeasonalGrowth_LegacyEquivalence exhaustively verifies
// that generic players with no biometric profile (attrs == nil) produce results 100% identical
// to the legacy logic across all permutations of age, appearances, potential, and currentOVR.
func TestChallenger2_GenericPlayer_ApplySeasonalGrowth_LegacyEquivalence(t *testing.T) {
	ge := NewGrowthEngine(12345)

	testAges := []int{14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 30, 35, 40}
	testApps := []int{0, 1, 5, 7, 8, 9, 15, 19, 20, 21, 25, 30, 38, 44, 55, 100}
	testPots := []int{65, 70, 72, 75, 80, 85, 90, 93, 94, 95, 96, 99}
	testCats := []string{"FWD", "MID", "DEF", ""}
	testOVRs := [][]int{
		nil, // no currentOVR passed
		{60},
		{70},
		{72},
		{75},
		{80},
		{85},
		{90},
		{94},
		{96},
	}

	permutationsChecked := 0

	for _, age := range testAges {
		for _, apps := range testApps {
			for _, pot := range testPots {
				for _, cat := range testCats {
					for _, ovrSlice := range testOVRs {
						playerID := fmt.Sprintf("unregistered_%d_%d_%d", age, apps, pot)

						// 1. Test GrowthEngine.ApplySeasonalGrowth
						var actual int
						var expected int
						if ovrSlice == nil {
							actual = ge.ApplySeasonalGrowth(playerID, age, apps, pot, cat)
							expected = referenceLegacySeasonalGrowth(age, apps, pot)
						} else {
							actual = ge.ApplySeasonalGrowth(playerID, age, apps, pot, cat, ovrSlice[0])
							expected = referenceLegacySeasonalGrowth(age, apps, pot, ovrSlice[0])
						}

						if actual != expected {
							t.Fatalf("[ApplySeasonalGrowth] Discrepancy for generic player (age=%d, apps=%d, pot=%d, cat=%s, ovr=%v): got %d, want %d",
								age, apps, pot, cat, ovrSlice, actual, expected)
						}

						// 2. Test ApplySeasonalGrowthHelper with engine
						var helperEngineActual int
						if ovrSlice == nil {
							helperEngineActual = ApplySeasonalGrowthHelper(playerID, age, pot, apps, cat, ge)
						} else {
							helperEngineActual = ApplySeasonalGrowthHelper(playerID, age, pot, apps, cat, ge, ovrSlice[0])
						}
						if helperEngineActual != expected {
							t.Fatalf("[ApplySeasonalGrowthHelper w/ ge] Discrepancy: got %d, want %d", helperEngineActual, expected)
						}

						// 3. Test ApplySeasonalGrowthHelper without engine (nil)
						var helperNilActual int
						if ovrSlice == nil {
							helperNilActual = ApplySeasonalGrowthHelper(playerID, age, pot, apps, cat, nil)
						} else {
							helperNilActual = ApplySeasonalGrowthHelper(playerID, age, pot, apps, cat, nil, ovrSlice[0])
						}
						if helperNilActual != expected {
							t.Fatalf("[ApplySeasonalGrowthHelper nil] Discrepancy: got %d, want %d", helperNilActual, expected)
						}

						// Invariant: for young players (< 25) undergoing seasonal growth, returned OVR must NEVER exceed potential
						if age < 25 && actual > pot {
							t.Fatalf("CRITICAL INVARIANT VIOLATION: actual %d > pot %d for age %d", actual, pot, age)
						}

						permutationsChecked++
					}
				}
			}
		}
	}

	t.Logf("Successfully verified %d generic player permutations against legacy logic with 0 discrepancies.", permutationsChecked)
}

// TestChallenger2_VeteranAgingDecline_IntegrityAndIsolation verifies that aging decline
// behaves strictly as specified and was not regressed by wonderkid progression rebalancing.
func TestChallenger2_VeteranAgingDecline_IntegrityAndIsolation(t *testing.T) {
	ge := NewGrowthEngine(54321)

	// 1. Verify Non-veterans (< 30) suffer ZERO attribute drop and ZERO OVR drop
	for age := 14; age < 30; age++ {
		pID := fmt.Sprintf("youth_vet_test_%d", age)
		ge.Attributes[pID] = &TechnicalAttributes{
			Pace:        80,
			Stamina:     80,
			Strength:    80,
			Physicality: 80,
			Shooting:    80,
			Passing:     80,
			Dribbling:   80,
			Defending:   80,
			Composure:   80,
		}

		changed := ge.ApplyAgingDecline(pID, age)
		if len(changed) != 0 {
			t.Fatalf("Age %d (< 30) suffered aging decline: %v", age, changed)
		}

		// Also verify SeasonalOVRDrop
		ovrDrop := SeasonalOVRDrop(age, 80)
		if ovrDrop != 80 {
			t.Fatalf("Age %d (< 30) suffered SeasonalOVRDrop: want 80, got %d", age, ovrDrop)
		}
	}

	// 2. Verify Exact Decline Rates for Veterans (30+)
	veteranTests := []struct {
		age          int
		expectedDrop int
	}{
		{30, 1},
		{31, 1},
		{32, 1},
		{33, 1},
		{34, 2},
		{35, 2},
		{36, 3},
		{37, 3},
		{38, 3},
		{40, 3},
		{45, 3},
	}

	for _, vt := range veteranTests {
		pID := fmt.Sprintf("veteran_test_%d", vt.age)
		ge.Attributes[pID] = &TechnicalAttributes{
			Pace:        70,
			Stamina:     70,
			Strength:    70,
			Physicality: 70,
			Shooting:    85,
			Passing:     85,
			Dribbling:   85,
			Defending:   85,
			Composure:   85,
		}

		// Single season decline
		changed := ge.ApplyAgingDecline(pID, vt.age)
		if len(changed) != 4 {
			t.Fatalf("Age %d: expected 4 physical attributes changed, got %d (%v)", vt.age, len(changed), changed)
		}

		attrs := ge.Attributes[pID]
		expectedVal := 70 - vt.expectedDrop
		if attrs.Pace != expectedVal || attrs.Stamina != expectedVal ||
			attrs.Strength != expectedVal || attrs.Physicality != expectedVal {
			t.Fatalf("Age %d: expected physical attributes %d, got pace=%d, stamina=%d, strength=%d, phys=%d",
				vt.age, expectedVal, attrs.Pace, attrs.Stamina, attrs.Strength, attrs.Physicality)
		}

		// Non-physical attributes MUST remain completely untouched
		if attrs.Shooting != 85 || attrs.Passing != 85 || attrs.Dribbling != 85 ||
			attrs.Defending != 85 || attrs.Composure != 85 {
			t.Fatalf("Age %d: non-physical attributes were modified by aging decline!", vt.age)
		}

		// SeasonalOVRDrop matches exact rate
		dropOVR := SeasonalOVRDrop(vt.age, 85)
		if dropOVR != 85-vt.expectedDrop {
			t.Fatalf("Age %d SeasonalOVRDrop: want %d, got %d", vt.age, 85-vt.expectedDrop, dropOVR)
		}

		// Verify 35 hard floor for physical attributes under extreme repeated decline
		for rep := 0; rep < 50; rep++ {
			ge.ApplyAgingDecline(pID, vt.age)
		}
		if attrs.Pace != 35 || attrs.Stamina != 35 || attrs.Strength != 35 || attrs.Physicality != 35 {
			t.Fatalf("Age %d: attributes fell below hard floor 35! pace=%d, stamina=%d, strength=%d, phys=%d",
				vt.age, attrs.Pace, attrs.Stamina, attrs.Strength, attrs.Physicality)
		}

		// Verify 55 hard floor for SeasonalOVRDrop
		flooredOVR := 56
		for rep := 0; rep < 50; rep++ {
			flooredOVR = SeasonalOVRDrop(vt.age, flooredOVR)
		}
		if flooredOVR != 55 {
			t.Fatalf("Age %d SeasonalOVRDrop fell below hard floor 55: got %d", vt.age, flooredOVR)
		}
	}

	// 3. Verify that veterans registered with biometrics do NOT gain seasonal growth
	ge.RegisterProdigy("WK_Veteran_Reg", "Old Prodigy", 32, 185.0, 80.0, "FWD", 82, 90, 19)
	initialOVR := ge.CalculateOVR("WK_Veteran_Reg", "FWD")
	seasonalRes := ge.ApplySeasonalGrowth("WK_Veteran_Reg", 32, 44, 90, "FWD", initialOVR)
	if seasonalRes > initialOVR {
		t.Fatalf("Veteran (age 32) received positive youth seasonal growth! Initial=%d, Result=%d", initialOVR, seasonalRes)
	}
}

// TestChallenger2_PotentialClamping_UniversalStrictness verifies that under no circumstance
// (bombardment of match XP, max mentor bonuses, repeated seasonal growth, direct attribute tampering)
// can any player's OVR exceed their assigned potential or 96 for canonical wonderkids.
func TestChallenger2_PotentialClamping_UniversalStrictness(t *testing.T) {
	ge := NewGrowthEngine(98765)

	// Canonical Wonderkids to stress test
	testWonderkids := []struct {
		id        string
		name      string
		startOVR  int
		potential int
		posCat    string
	}{
		{"WK_Valerio", "Venjamin Valerio", 78, 96, "FWD"},
		{"WK_Cantalejo", "Maverick Cantalejo", 77, 95, "MID"},
		{"WK_Gocotano", "Yeshua Emmanuel Gocotano", 75, 93, "FWD"},
		{"WK_Bantol", "Izyan Levin Bantol", 76, 95, "MID"},
		{"WK_Rizon", "James Bernard Rizon", 76, 94, "FWD"},
		{"WK_Libatan", "Reid Randell Libatan", 76, 95, "FWD"},
		{"WK_Baguio", "Ashle Zylle Baguio", 75, 95, "MID"},
		{"WK_Lanticse", "Cliergy Jave Lanticse", 75, 94, "MID"},
		{"WK_Zamora", "Ezail Zamora", 77, 96, "FWD"},
		{"WK_Hernando", "Earl Josh Hernando", 75, 94, "MID"},
		{"WK_Suico", "Rich Lorenz Suico", 75, 94, "DEF"},
		{"WK_Guinita", "Jhed Anthony Guinita", 75, 94, "FWD"},
	}

	for _, wk := range testWonderkids {
		// Verify canonical potential invariant: must be in [93, 96]
		if wk.potential < 93 || wk.potential > 96 {
			t.Fatalf("CRITICAL INVARIANT VIOLATION: Prodigy %s potential %d not in [93, 96]", wk.name, wk.potential)
		}

		bio, attrs := ge.RegisterProdigy(
			wk.id, wk.name, 14, 172.0, 62.0, wk.posCat, wk.startOVR, wk.potential, 19,
		)
		if bio.Potential != wk.potential {
			t.Fatalf("Prodigy %s potential mismatch: registered %d, want %d", wk.name, bio.Potential, wk.potential)
		}

		// Phase 1: Massive Match XP Bombardment (200 matches with 10.0 rating and 5 goals)
		mentorOVR := 95
		mentorName := "Senior Legend"
		pers := "dedicated_pro"
		opts := MatchXPOptions{
			MentorOVR:   &mentorOVR,
			MentorName:  &mentorName,
			Personality: &pers,
		}

		for m := 1; m <= 200; m++ {
			ge.ApplyMatchXP(wk.id, wk.name, wk.posCat, 10.0, 5, 5, opts)
			currentOVR := ge.CalculateOVR(wk.id, wk.posCat)
			if currentOVR > wk.potential {
				t.Fatalf("Match XP Overgrowth: %s OVR %d exceeded potential %d on match %d!",
					wk.name, currentOVR, wk.potential, m)
			}
			if currentOVR > 96 {
				t.Fatalf("CRITICAL: %s OVR %d exceeded absolute ceiling of 96!", wk.name, currentOVR)
			}
		}

		// Phase 2: Massive Seasonal Growth Calls (30 consecutive seasons with 50 appearances)
		for s := 1; s <= 30; s++ {
			curOVR := ge.CalculateOVR(wk.id, wk.posCat)
			grownOVR := ge.ApplySeasonalGrowth(wk.id, 14+s, 50, wk.potential, wk.posCat, curOVR)
			if grownOVR > wk.potential {
				t.Fatalf("Seasonal Overgrowth: %s seasonal OVR %d exceeded potential %d in season %d!",
					wk.name, grownOVR, wk.potential, s)
			}
			if grownOVR > 96 {
				t.Fatalf("CRITICAL: %s seasonal OVR %d exceeded absolute ceiling of 96!", wk.name, grownOVR)
			}
			calcOVR := ge.CalculateOVR(wk.id, wk.posCat)
			if calcOVR > wk.potential {
				t.Fatalf("CalculateOVR Overgrowth: %s OVR %d exceeded potential %d in season %d!",
					wk.name, calcOVR, wk.potential, s)
			}
		}

		// Phase 3: Adversarial Attribute Maxing (force all raw attributes to 99 in memory)
		ge.mu.Lock()
		attrs.Pace = 99
		attrs.Shooting = 99
		attrs.Passing = 99
		attrs.Dribbling = 99
		attrs.Defending = 99
		attrs.Physicality = 99
		attrs.Composure = 99
		attrs.Strength = 99
		attrs.Stamina = 99
		attrs.HeadingPower = 99
		attrs.Shielding = 99
		attrs.PressResistance = 99
		ge.mu.Unlock()

		// CalculateOVR MUST enforce the potential ceiling even when attributes are 99
		forcedOVR := ge.CalculateOVR(wk.id, wk.posCat)
		if forcedOVR > wk.potential {
			t.Fatalf("Adversarial Tampering: %s OVR %d with maxed attributes exceeded potential %d!",
				wk.name, forcedOVR, wk.potential)
		}
		if forcedOVR > 96 {
			t.Fatalf("CRITICAL: %s OVR %d with maxed attributes exceeded absolute ceiling 96!",
				wk.name, forcedOVR)
		}

		// ApplySeasonalGrowth with maxed attributes MUST also enforce potential ceiling
		seasonalForced := ge.ApplySeasonalGrowth(wk.id, 20, 50, wk.potential, wk.posCat, forcedOVR)
		if seasonalForced > wk.potential {
			t.Fatalf("Seasonal growth with maxed attributes: %d > %d", seasonalForced, wk.potential)
		}
		if seasonalForced > 96 {
			t.Fatalf("CRITICAL: Seasonal growth with maxed attributes exceeded 96: %d", seasonalForced)
		}
	}
}

// TestChallenger2_SingleSeasonGain_BoundedNeverExceeds5 tests 200 random seasons
// across canonical wonderkids and verifies single-season growth is strictly between +2 and +4
// under standard starts, and NEVER exceeds +5 under any condition.
func TestChallenger2_SingleSeasonGain_BoundedNeverExceeds5(t *testing.T) {
	for seed := int64(3001); seed <= 3010; seed++ {
		ge := NewGrowthEngine(seed)
		rng := rand.New(rand.NewSource(seed))

		for _, wk := range canonicalTestProdigies {
			ge.RegisterProdigy(wk.id, wk.name, 14, 168.0, 58.0, "FWD", wk.startOVR, wk.potential, 19)

			startOVR := ge.CalculateOVR(wk.id, "FWD")
			appearances, endOVR := simulateSeasonMatches(ge, wk.id, wk.name, wk.potential, 14, rng)
			gain := endOVR - startOVR

			if appearances >= 30 {
				if gain < 2 || gain > 4 {
					t.Errorf("[%s seed %d] Standard regular starter (apps=%d) gained +%d OVR, expected +2 to +4 (start=%d, end=%d)",
						wk.name, seed, appearances, gain, startOVR, endOVR)
				}
			}

			if gain > 5 {
				t.Fatalf("[%s seed %d] CRITICAL FAILURE: Single-season gain +%d exceeded hard ceiling +5! (start=%d, end=%d)",
					wk.name, seed, gain, startOVR, endOVR)
			}
			if gain < 0 {
				t.Fatalf("[%s seed %d] Youth wonderkid had negative growth in season 1: %d", wk.name, seed, gain)
			}
			if endOVR > wk.potential {
				t.Fatalf("[%s seed %d] End OVR %d exceeded potential %d!", wk.name, seed, endOVR, wk.potential)
			}
			if endOVR > 96 {
				t.Fatalf("[%s seed %d] End OVR %d exceeded absolute ceiling 96!", wk.name, seed, endOVR)
			}
		}
	}
}

// TestChallenger2_AdversarialInputs_SeasonalGrowth tests pathological boundary inputs
// such as negative appearances, zero appearances, anomalous base > potential, and extreme ages.
func TestChallenger2_AdversarialInputs_SeasonalGrowth(t *testing.T) {
	ge := NewGrowthEngine(777)

	// 1. Anomalous base > potential for unregistered player
	// Base = 95, Potential = 90. Output MUST be clamped to potential (90), not base+bump.
	res := ge.ApplySeasonalGrowth("anon_over", 18, 30, 90, "FWD", 95)
	if res != 90 {
		t.Fatalf("Anomalous base > pot: expected 90, got %d", res)
	}

	// 2. Negative appearances
	resNeg := ge.ApplySeasonalGrowth("anon_neg", 18, -10, 90, "FWD", 75)
	if resNeg != 76 { // base 75 + bump 1 = 76
		t.Fatalf("Negative appearances: expected 76, got %d", resNeg)
	}

	// 3. Zero appearances for registered wonderkid (bump = 0)
	ge.RegisterProdigy("WK_ZeroApps", "Zero Apps", 16, 175.0, 68.0, "FWD", 76, 95, 19)
	resZero := ge.ApplySeasonalGrowth("WK_ZeroApps", 16, 0, 95, "FWD", 76)
	if resZero != 76 {
		t.Fatalf("Zero appearances for wonderkid: expected 76 (0 bump), got %d", resZero)
	}

	// 4. Extreme age: age 100 (veteran beyond 25)
	res100 := ge.ApplySeasonalGrowth("anon_100", 100, 30, 90, "FWD", 65)
	if res100 != 65 {
		t.Fatalf("Age 100: expected unchanged 65, got %d", res100)
	}

	// 5. Empty position category with unregistered player
	resEmpty := ge.ApplySeasonalGrowth("anon_empty", 18, 25, 90, "", 70)
	if resEmpty != 73 {
		t.Fatalf("Empty category: expected 73, got %d", resEmpty)
	}

	// 6. Registered player with anomalous currentOVR > potential passed in
	ge.RegisterProdigy("WK_OverPot", "Over Pot", 18, 178.0, 70.0, "FWD", 93, 94, 19)
	resOverPot := ge.ApplySeasonalGrowth("WK_OverPot", 18, 40, 94, "FWD", 96)
	if resOverPot > 94 {
		t.Fatalf("Registered player with anomalous base > pot returned %d (> 94)", resOverPot)
	}
}

// TestChallenger2_TrainingAndPuberty_CeilingIntegrity verifies that intensive training
// and puberty cycles cannot cause a wonderkid's OVR to exceed their potential or 96.
func TestChallenger2_TrainingAndPuberty_CeilingIntegrity(t *testing.T) {
	ge := NewGrowthEngine(888)

	for _, wk := range canonicalTestProdigies {
		ge.RegisterProdigy(wk.id, wk.name, 14, 168.0, 58.0, "FWD", wk.startOVR, wk.potential, 19)

		// 100 Puberty cycles
		for mw := 1; mw <= 100; mw++ {
			ge.SimulatePubertyCycle(wk.id, mw)
			ovr := ge.CalculateOVR(wk.id, "FWD")
			if ovr > wk.potential {
				t.Fatalf("Puberty cycle %d: %s OVR %d exceeded potential %d", mw, wk.name, ovr, wk.potential)
			}
			if ovr > 96 {
				t.Fatalf("Puberty cycle %d: %s OVR %d exceeded 96", mw, wk.name, ovr)
			}
		}

		// 150 Intensive Training cycles (50 hypertrophy, 50 technical, 50 tactical)
		focuses := []string{"hypertrophy", "technical", "tactical"}
		for i := 0; i < 150; i++ {
			f := focuses[i%3]
			_, err := ge.RunTrainingCycle(wk.id, f, false) // consumeEnergy=false
			if err != nil {
				t.Fatalf("Training cycle failed: %v", err)
			}

			ovr := ge.CalculateOVR(wk.id, "FWD")
			if ovr > wk.potential {
				t.Fatalf("Training cycle %d (%s): %s OVR %d exceeded potential %d", i, f, wk.name, ovr, wk.potential)
			}
			if ovr > 96 {
				t.Fatalf("Training cycle %d (%s): %s OVR %d exceeded 96", i, f, wk.name, ovr)
			}
		}
	}
}

// TestChallenger2_CanonicalWonderkids_10SeasonCareer_EndToEnd tests full 10-season careers
// for all 12 canonical wonderkids from age 14 to age 24.
func TestChallenger2_CanonicalWonderkids_10SeasonCareer_EndToEnd(t *testing.T) {
	for seed := int64(4001); seed <= 4003; seed++ {
		ge := NewGrowthEngine(seed)
		rng := rand.New(rand.NewSource(seed))

		for _, wk := range canonicalTestProdigies {
			ge.RegisterProdigy(wk.id, wk.name, 14, 168.0, 58.0, "FWD", wk.startOVR, wk.potential, 19)

			ovrHistory := make(map[int]int)
			ovrHistory[14] = wk.startOVR

			for age := 14; age <= 23; age++ {
				startSeasonOVR := ge.CalculateOVR(wk.id, "FWD")
				_, endSeasonOVR := simulateSeasonMatches(ge, wk.id, wk.name, wk.potential, age, rng)
				seasonGain := endSeasonOVR - startSeasonOVR

				// Invariant: Never exceed +5 in a single season
				if seasonGain > 5 {
					t.Fatalf("[%s seed %d age %d] Single season gain +%d exceeded hard ceiling +5! (from %d to %d)",
						wk.name, seed, age, seasonGain, startSeasonOVR, endSeasonOVR)
				}
				if seasonGain < 0 {
					t.Fatalf("[%s seed %d age %d] Negative growth detected: %d -> %d",
						wk.name, seed, age, startSeasonOVR, endSeasonOVR)
				}
				if endSeasonOVR > wk.potential {
					t.Fatalf("[%s seed %d age %d] End season OVR %d exceeded potential %d!",
						wk.name, seed, age, endSeasonOVR, wk.potential)
				}
				if endSeasonOVR > 96 {
					t.Fatalf("[%s seed %d age %d] End season OVR %d exceeded 96!",
						wk.name, seed, age, endSeasonOVR)
				}

				bio := ge.Biometrics[wk.id]
				bio.Age = age + 1
				ge.ResetYearlyHeightTaken(wk.id)
				ovrHistory[age+1] = endSeasonOVR
			}

			// Milestone 1: Age 16 (after 2 seasons)
			// For 75 OVR starters: ~79–82 OVR (up to 83)
			// For 77-78 OVR starters: startOVR + 6..8 (~83–86)
			ovr16 := ovrHistory[16]
			twoYearGain := ovr16 - wk.startOVR
			if twoYearGain < 5 || twoYearGain > 8 {
				t.Errorf("[%s seed %d] 2-year gain at age 16 out of bounds [+5, +8]: start=%d, age16=%d, gain=%d",
					wk.name, seed, wk.startOVR, ovr16, twoYearGain)
			}
			if wk.startOVR == 75 && (ovr16 < 79 || ovr16 > 83) {
				t.Errorf("[%s seed %d] Typical 75 OVR starter at age 16: expected 79-83 OVR, got %d", wk.name, seed, ovr16)
			}

			// Milestone 2: Age 18 (after 4 seasons)
			// For 75 OVR starters: ~85–88 OVR (up to 89)
			// For 77-78 OVR starters: startOVR + 11..13 (~88–91)
			ovr18 := ovrHistory[18]
			fourYearGain := ovr18 - wk.startOVR
			if fourYearGain < 10 || fourYearGain > 14 {
				t.Errorf("[%s seed %d] 4-year gain at age 18 out of bounds [+10, +14]: start=%d, age18=%d, gain=%d",
					wk.name, seed, wk.startOVR, ovr18, fourYearGain)
			}
			if wk.startOVR == 75 && (ovr18 < 85 || ovr18 > 89) {
				t.Errorf("[%s seed %d] Typical 75 OVR starter at age 18: expected 85-89 OVR, got %d", wk.name, seed, ovr18)
			}

			// Milestone 3: Early 20s (Age 22-24) should approach or reach potential ceiling [93, 96]
			ovr24 := ovrHistory[24]
			if ovr24 < 91 || ovr24 > wk.potential {
				t.Errorf("[%s seed %d] At age 24: expected 91-%d OVR, got %d", wk.name, seed, wk.potential, ovr24)
			}
		}
	}
}

