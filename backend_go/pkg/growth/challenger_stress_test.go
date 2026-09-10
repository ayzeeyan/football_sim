package growth

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
)

// TestChallenger_GrowthEngine_PotentialBounds_Wonderkids tests that canonical wonderkids
// (potentials 93 to 96) NEVER grow beyond their potential ceiling under extreme XP and seasonal progression.
func TestChallenger_GrowthEngine_PotentialBounds_Wonderkids(t *testing.T) {
	ge := NewGrowthEngine(99999)

	wonderkids := []struct {
		id        string
		name      string
		potential int
		posCat    string
	}{
		{"WK_01", "Wonderkid 93", 93, "FWD"},
		{"WK_02", "Wonderkid 94", 94, "MID"},
		{"WK_03", "Wonderkid 95", 95, "DEF"},
		{"WK_04", "Wonderkid 96", 96, "FWD"},
	}

	for _, wk := range wonderkids {
		bio, attrs := ge.RegisterProdigy(
			wk.id, wk.name, 14, 168.0, 58.0, wk.posCat, 76, wk.potential, 19,
		)
		if bio.Potential != wk.potential {
			t.Fatalf("Player %s potential registered as %d, expected %d", wk.id, bio.Potential, wk.potential)
		}

		// 1. Extreme Match XP Bombardment: 500 match appearances with max rating
		mentorOVR := 95
		mentorName := "Grandmaster"
		pers := "dedicated_pro"
		opts := MatchXPOptions{
			MentorOVR:   &mentorOVR,
			MentorName:  &mentorName,
			Personality: &pers,
		}

		for match := 0; match < 500; match++ {
			ge.ApplyMatchXP(wk.id, wk.name, wk.posCat, 10.0, 5, 5, opts)
			currentOVR := ge.CalculateOVR(wk.id, wk.posCat)
			if currentOVR > wk.potential {
				t.Fatalf("VIOLATION: Player %s OVR exceeded potential during MatchXP: OVR=%d, Potential=%d",
					wk.id, currentOVR, wk.potential)
			}
			if currentOVR >= 99 && wk.potential < 99 {
				t.Fatalf("VIOLATION: Wonderkid %s reached 99 OVR! OVR=%d, Potential=%d",
					wk.id, currentOVR, wk.potential)
			}
		}

		// 2. Extreme Seasonal Growth: 50 consecutive seasons with 50 appearances each
		for season := 0; season < 50; season++ {
			age := 14 + season
			grownOVR := ge.ApplySeasonalGrowth(wk.id, age, 50, wk.potential, wk.posCat)
			if grownOVR > wk.potential {
				t.Fatalf("VIOLATION: ApplySeasonalGrowth returned %d, exceeding potential %d for %s",
					grownOVR, wk.potential, wk.id)
			}
			calcOVR := ge.CalculateOVR(wk.id, wk.posCat)
			if calcOVR > wk.potential {
				t.Fatalf("VIOLATION: CalculateOVR returned %d, exceeding potential %d for %s",
					calcOVR, wk.potential, wk.id)
			}
		}

		// 3. Adversarial Attribute Manipulation: Force all attributes to 99
		// Even if all raw attributes are maxed to 99, CalculateOVR MUST enforce the potential cap!
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
		ge.mu.Unlock()

		forcedOVR := ge.CalculateOVR(wk.id, wk.posCat)
		if forcedOVR > wk.potential {
			t.Fatalf("VIOLATION: CalculateOVR returned %d with maxed attributes; MUST be capped at potential %d",
				forcedOVR, wk.potential)
		}
		if forcedOVR >= 99 && wk.potential < 99 {
			t.Fatalf("VIOLATION: Wonderkid %s reached 99 with maxed attributes! Cap was bypassed!", wk.id)
		}
	}
}

// TestChallenger_GrowthEngine_ApplySeasonalGrowth_PotentialCeiling tests edge cases:
// when player OVR is already at or near potential, or potential passed is lower/higher.
func TestChallenger_GrowthEngine_ApplySeasonalGrowth_PotentialCeiling(t *testing.T) {
	ge := NewGrowthEngine(42)

	// Unregistered player static bump
	// Base 93, Pot 94, Apps 30 (+3) -> Clamped to 94
	res1 := ge.ApplySeasonalGrowth("unreg_1", 18, 30, 94, "FWD", 93)
	if res1 != 94 {
		t.Fatalf("Expected 94, got %d", res1)
	}

	// Base 94, Pot 94, Apps 30 (+3) -> Stays 94
	res2 := ge.ApplySeasonalGrowth("unreg_2", 18, 30, 94, "FWD", 94)
	if res2 != 94 {
		t.Fatalf("Expected 94, got %d", res2)
	}

	// Base 95, Pot 94 (anomalous input where base > pot) -> Clamped to 94
	res3 := ge.ApplySeasonalGrowth("unreg_3", 18, 30, 94, "FWD", 95)
	if res3 != 94 {
		t.Fatalf("Expected 94 when base exceeds potential, got %d", res3)
	}

	// Registered player with potential 93
	ge.RegisterProdigy("reg_p", "Reg Prodigy", 16, 175.0, 68.0, "MID", 92, 93, 19)
	// Try to push beyond potential by passing potential=99 in seasonal growth
	// The BiometricProfile has potential 93, so CalculateOVR and internalCalculateOVR will cap at 93!
	resReg := ge.ApplySeasonalGrowth("reg_p", 16, 40, 93, "MID")
	if resReg > 93 {
		t.Fatalf("Registered prodigy grew to %d, exceeding potential 93!", resReg)
	}
}

// TestChallenger_AgingDecline_VeteransFloor35 rigorously tests that aging decline decays
// physical attributes down to floor 35, NEVER below 35, across all veteran ages and multiple passes.
func TestChallenger_AgingDecline_VeteransFloor35(t *testing.T) {
	ge := NewGrowthEngine(1234)

	targetAges := []struct {
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
		{40, 3},
		{45, 3},
	}

	for _, ta := range targetAges {
		playerID := fmt.Sprintf("vet_%d", ta.age)
		ge.Attributes[playerID] = &TechnicalAttributes{
			Pace:            60,
			Stamina:         60,
			Strength:        60,
			Physicality:     60,
			Shooting:        80, // non-physical
			Passing:         80, // non-physical
			Dribbling:       80, // non-physical
			Defending:       80, // non-physical
			Composure:       80, // non-physical
		}

		// First decline
		changed := ge.ApplyAgingDecline(playerID, ta.age)
		if len(changed) != 4 {
			t.Fatalf("Age %d: expected 4 physical attributes changed, got %d (%v)",
				ta.age, len(changed), changed)
		}

		attrs := ge.Attributes[playerID]
		expectedVal := 60 - ta.expectedDrop
		if attrs.Pace != expectedVal || attrs.Stamina != expectedVal ||
			attrs.Strength != expectedVal || attrs.Physicality != expectedVal {
			t.Fatalf("Age %d: expected physical attributes %d, got pace=%d, stamina=%d, strength=%d, phys=%d",
				ta.age, expectedVal, attrs.Pace, attrs.Stamina, attrs.Strength, attrs.Physicality)
		}

		// Verify non-physical attributes are strictly UNTOUCHED
		if attrs.Shooting != 80 || attrs.Passing != 80 || attrs.Dribbling != 80 ||
			attrs.Defending != 80 || attrs.Composure != 80 {
			t.Fatalf("Age %d: non-physical attributes were modified by aging decline!", ta.age)
		}

		// Run aging decline 50 more times to drive attributes into the floor
		for iter := 0; iter < 50; iter++ {
			ge.ApplyAgingDecline(playerID, ta.age)
		}

		// Verify hard floor of 35 is held strictly
		if attrs.Pace != 35 {
			t.Fatalf("Age %d: Pace floor violated! Expected 35, got %d", ta.age, attrs.Pace)
		}
		if attrs.Stamina != 35 {
			t.Fatalf("Age %d: Stamina floor violated! Expected 35, got %d", ta.age, attrs.Stamina)
		}
		if attrs.Strength != 35 {
			t.Fatalf("Age %d: Strength floor violated! Expected 35, got %d", ta.age, attrs.Strength)
		}
		if attrs.Physicality != 35 {
			t.Fatalf("Age %d: Physicality floor violated! Expected 35, got %d", ta.age, attrs.Physicality)
		}

		// Further calls when floored at 35 must return empty slice
		flooredChanged := ge.ApplyAgingDecline(playerID, ta.age)
		if len(flooredChanged) != 0 {
			t.Fatalf("Age %d: Floored veteran returned changes: %v", ta.age, flooredChanged)
		}
	}
}

// TestChallenger_AgingDecline_NonVeteransZeroDecay tests that players under 30 (ages 14-29)
// receive exactly 0 aging decay and their attributes are completely untouched.
func TestChallenger_AgingDecline_NonVeteransZeroDecay(t *testing.T) {
	ge := NewGrowthEngine(5678)

	nonVeteranAges := []int{14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29}

	for _, age := range nonVeteranAges {
		pID := fmt.Sprintf("youth_%d", age)
		ge.Attributes[pID] = &TechnicalAttributes{
			Pace:        75,
			Stamina:     75,
			Strength:    75,
			Physicality: 75,
			Shooting:    75,
			Passing:     75,
		}

		changed := ge.ApplyAgingDecline(pID, age)
		if len(changed) != 0 {
			t.Fatalf("Age %d (< 30) received aging decline: %v", age, changed)
		}

		attrs := ge.Attributes[pID]
		if attrs.Pace != 75 || attrs.Stamina != 75 || attrs.Strength != 75 || attrs.Physicality != 75 {
			t.Fatalf("Age %d (< 30) attributes modified! Expected all 75, got pace=%d, stamina=%d",
				age, attrs.Pace, attrs.Stamina)
		}
	}
}

// TestChallenger_SeasonalOVRDrop_ExhaustiveGrid tests veteran OVR decline and floor 55.
func TestChallenger_SeasonalOVRDrop_ExhaustiveGrid(t *testing.T) {
	// 1. Age < 30: unchanged
	for age := 15; age < 30; age++ {
		for ovr := 60; ovr <= 95; ovr++ {
			if got := SeasonalOVRDrop(age, ovr); got != ovr {
				t.Fatalf("Age %d (<30) OVR changed from %d to %d", age, ovr, got)
			}
		}
	}

	// 2. Age 30-33: -1 OVR down to 55
	for age := 30; age <= 33; age++ {
		if got := SeasonalOVRDrop(age, 80); got != 79 {
			t.Fatalf("Age %d: 80 -> want 79, got %d", age, got)
		}
		if got := SeasonalOVRDrop(age, 56); got != 55 {
			t.Fatalf("Age %d: 56 -> want 55, got %d", age, got)
		}
		if got := SeasonalOVRDrop(age, 55); got != 55 {
			t.Fatalf("Age %d: 55 -> want 55 (floor), got %d", age, got)
		}
	}

	// 3. Age 34-35: -2 OVR down to 55
	for age := 34; age <= 35; age++ {
		if got := SeasonalOVRDrop(age, 80); got != 78 {
			t.Fatalf("Age %d: 80 -> want 78, got %d", age, got)
		}
		if got := SeasonalOVRDrop(age, 56); got != 55 {
			t.Fatalf("Age %d: 56 -> want 55 (floor), got %d", age, got)
		}
		if got := SeasonalOVRDrop(age, 55); got != 55 {
			t.Fatalf("Age %d: 55 -> want 55 (floor), got %d", age, got)
		}
	}

	// 4. Age 36+: -3 OVR down to 55
	for age := 36; age <= 45; age++ {
		if got := SeasonalOVRDrop(age, 80); got != 77 {
			t.Fatalf("Age %d: 80 -> want 77, got %d", age, got)
		}
		if got := SeasonalOVRDrop(age, 57); got != 55 {
			t.Fatalf("Age %d: 57 -> want 55 (floor), got %d", age, got)
		}
		if got := SeasonalOVRDrop(age, 56); got != 55 {
			t.Fatalf("Age %d: 56 -> want 55 (floor), got %d", age, got)
		}
		if got := SeasonalOVRDrop(age, 55); got != 55 {
			t.Fatalf("Age %d: 55 -> want 55 (floor), got %d", age, got)
		}
	}
}

// TestChallenger_GrowthEngine_ConcurrencyStress runs massive multi-goroutine operations
// across 60 goroutines performing 100 iterations each (6,000 total concurrent cycles).
func TestChallenger_GrowthEngine_ConcurrencyStress(t *testing.T) {
	ge := NewGrowthEngine(8888)

	// Pre-register 12 prodigies
	for i := 1; i <= 12; i++ {
		pID := fmt.Sprintf("WK_%02d", i)
		name := fmt.Sprintf("Stress Prodigy %d", i)
		pot := 93 + (i % 4) // 93, 94, 95, 96
		cat := "FWD"
		if i%3 == 1 {
			cat = "MID"
		} else if i%3 == 2 {
			cat = "DEF"
		}
		ge.RegisterProdigy(pID, name, 14, 166.0+float64(i), 56.0+float64(i), cat, 76, pot, 19)
	}

	var wg sync.WaitGroup
	workers := 60
	iterations := 100

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(workerID * 1000)))

			for it := 0; it < iterations; it++ {
				idx := (rng.Intn(12)) + 1
				pID := fmt.Sprintf("WK_%02d", idx)
				cat := "FWD"
				if idx%3 == 1 {
					cat = "MID"
				} else if idx%3 == 2 {
					cat = "DEF"
				}

				op := it % 8
				switch op {
				case 0:
					ge.CalculateOVR(pID, cat)
				case 1:
					ge.SimulatePubertyCycle(pID, it%38+1)
				case 2:
					mentorOVR := 85
					mentorName := "Veteran"
					pers := "dedicated_pro"
					opts := MatchXPOptions{
						MentorOVR:   &mentorOVR,
						MentorName:  &mentorName,
						Personality: &pers,
					}
					ge.ApplyMatchXP(pID, "Stress Prodigy", cat, 7.5, 1, 0, opts)
				case 3:
					ge.ApplyAgingDecline(pID, 30+(it%10))
				case 4:
					ge.GetProdigyData(pID, cat)
				case 5:
					ge.ApplySeasonalGrowth(pID, 14+(it%15), 15+(it%25), 95, cat)
				case 6:
					ge.RecordTimelineEntry(pID, fmt.Sprintf("202%d", it%5), 15, 78, 168.0, 58.0, 5, 2, 20, "TOT", "Veteran")
					ge.GetProgressionHistory(pID)
				case 7:
					ge.ResetYearlyHeightTaken(pID)
				}
			}
		}(w)
	}

	wg.Wait()

	// Post-concurrency sanity check: all wonderkids must still be within bounds
	for i := 1; i <= 12; i++ {
		pID := fmt.Sprintf("WK_%02d", i)
		pot := 93 + (i % 4)
		cat := "FWD"
		if i%3 == 1 {
			cat = "MID"
		} else if i%3 == 2 {
			cat = "DEF"
		}
		finalOVR := ge.CalculateOVR(pID, cat)
		if finalOVR > pot {
			t.Fatalf("Post-concurrency OVR violation for %s: OVR %d > Potential %d", pID, finalOVR, pot)
		}
		if finalOVR >= 99 && pot < 99 {
			t.Fatalf("Post-concurrency Wonderkid reached 99 OVR! %s", pID)
		}
	}
}
