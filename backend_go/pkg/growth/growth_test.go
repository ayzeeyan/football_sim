package growth

import (
	"fmt"
	"math"
	"sync"
	"testing"
)

func TestBiometricProfile_CalculatedProperties(t *testing.T) {
	bio := &BiometricProfile{
		PlayerID:         "test_1",
		FullName:         "Test Prodigy",
		Age:              14,
		CurrentHeightCM:  168.4,
		BaselineHeightCM: 165.0,
		CurrentWeightKG:  62.7,
		BaselineWeightKG: 60.0,
		Potential:        95,
	}

	expectedHeightGain := 3.4
	if math.Abs(bio.HeightGainCM()-expectedHeightGain) > 0.001 {
		t.Fatalf("Expected HeightGainCM %.1f, got %.1f", expectedHeightGain, bio.HeightGainCM())
	}

	expectedWeightGain := 2.7
	if math.Abs(bio.WeightGainKG()-expectedWeightGain) > 0.001 {
		t.Fatalf("Expected WeightGainKG %.1f, got %.1f", expectedWeightGain, bio.WeightGainKG())
	}
}

func TestFormatHeightCMFt(t *testing.T) {
	cases := []struct {
		cm       float64
		expected string
	}{
		{178.0, "178 cm (5'10\")"},
		{182.88, "182.9 cm (6'0\")"},
		{183.0, "183 cm (6'0\")"},
		{165.4, "165.4 cm (5'5\")"},
	}

	for _, c := range cases {
		got := FormatHeightCMFt(c.cm)
		if got != c.expected {
			t.Errorf("FormatHeightCMFt(%.2f) = %q, expected %q", c.cm, got, c.expected)
		}
	}
}

func TestAdultHeightAgeFor(t *testing.T) {
	// Explicit configured values
	if got := AdultHeightAgeFor("Player One", 18); got != 18 {
		t.Errorf("Expected 18, got %d", got)
	}
	if got := AdultHeightAgeFor("Player Two", 19); got != 19 {
		t.Errorf("Expected 19, got %d", got)
	}
	if got := AdultHeightAgeFor("Player Three", 20); got != 20 {
		t.Errorf("Expected 20, got %d", got)
	}

	// Dynamic deterministic hash
	age1 := AdultHeightAgeFor("Venjamin Valerio")
	age2 := AdultHeightAgeFor("Venjamin Valerio")
	if age1 != age2 {
		t.Errorf("AdultHeightAgeFor should be deterministic: %d != %d", age1, age2)
	}
	if age1 < 18 || age1 > 20 {
		t.Errorf("AdultHeightAgeFor returned %d, out of [18, 20] range", age1)
	}
}

func TestGrowthEngine_RegisterProdigy(t *testing.T) {
	ge := NewGrowthEngine(42)

	// Test Under-14 FWD
	bio14, attrs14 := ge.RegisterProdigy(
		"WK_Venjamin_Valerio", "Venjamin Valerio",
		14, 168.0, 58.0,
		"FWD", 76, 95, 19,
	)

	if bio14.PubertyStage != "Early-puberty" {
		t.Errorf("Expected Early-puberty, got %s", bio14.PubertyStage)
	}
	// yearsLeft = 19 - 14 = 5. perYear = 1.85. 5 * 1.85 = 9.25 -> 9.3
	expectedVelocity := 9.3
	if math.Abs(bio14.GrowthVelocity-expectedVelocity) > 0.05 {
		t.Errorf("Expected velocity ~%.1f, got %.1f", expectedVelocity, bio14.GrowthVelocity)
	}

	// Verify initial OVR equals baseline_ovr
	currentOVR := ge.CalculateOVR("WK_Venjamin_Valerio", "FWD")
	if currentOVR != 76 {
		t.Errorf("Expected baseline nudged OVR 76, got %d", currentOVR)
	}

	// Verify FWD attribute seeding biases
	if attrs14.Shooting < 75 || attrs14.Pace < 75 {
		t.Errorf("Expected high pace and shooting for FWD, got pace %d, shooting %d", attrs14.Pace, attrs14.Shooting)
	}

	// Test Age 15 MID
	bio15, _ := ge.RegisterProdigy(
		"WK_Mid", "Mid Prodigy",
		15, 172.0, 64.0,
		"MID", 75, 94, 19,
	)
	if bio15.PubertyStage != "Mid-puberty" {
		t.Errorf("Expected Mid-puberty, got %s", bio15.PubertyStage)
	}

	// Test Age 17 DEF
	bio17, _ := ge.RegisterProdigy(
		"WK_Def", "Def Prodigy",
		17, 185.0, 78.0,
		"DEF", 75, 93, 19,
	)
	if bio17.PubertyStage != "Late-puberty" {
		t.Errorf("Expected Late-puberty, got %s", bio17.PubertyStage)
	}

	// Test Age 20 Adult
	bio20, _ := ge.RegisterProdigy(
		"WK_Adult", "Adult Prodigy",
		20, 188.0, 82.0,
		"DEF", 78, 93, 19,
	)
	if bio20.PubertyStage != "Adult frame" || bio20.GrowthVelocity != 0.0 {
		t.Errorf("Expected Adult frame with 0 velocity, got %s and %.1f", bio20.PubertyStage, bio20.GrowthVelocity)
	}
}

func TestGrowthEngine_CalculateOVR(t *testing.T) {
	ge := NewGrowthEngine(42)

	// Manually set attributes to test exact mathematical weights
	ge.Attributes["fwd_test"] = &TechnicalAttributes{
		Pace:        80, // 80 * 0.25 = 20
		Shooting:    80, // 80 * 0.35 = 28
		Dribbling:   80, // 80 * 0.20 = 16
		Passing:     80, // 80 * 0.10 = 8
		Physicality: 80, // 80 * 0.10 = 8 -> Sum = 80
	}
	ovrFWD := ge.CalculateOVR("fwd_test", "FWD")
	if ovrFWD != 80 {
		t.Errorf("Expected FWD OVR 80, got %d", ovrFWD)
	}

	ge.Attributes["mid_test"] = &TechnicalAttributes{
		Passing:     90, // 90 * 0.30 = 27
		Dribbling:   80, // 80 * 0.25 = 20
		Pace:        70, // 70 * 0.15 = 10.5
		Shooting:    70, // 70 * 0.15 = 10.5
		Physicality: 70, // 70 * 0.15 = 10.5 -> Sum = 78.5 -> 79
	}
	ovrMID := ge.CalculateOVR("mid_test", "MID")
	if ovrMID != 79 {
		t.Errorf("Expected MID OVR 79, got %d", ovrMID)
	}

	ge.Attributes["def_test"] = &TechnicalAttributes{
		Defending:   90, // 90 * 0.40 = 36
		Physicality: 80, // 80 * 0.25 = 20
		Pace:        70, // 70 * 0.15 = 10.5
		Passing:     70, // 70 * 0.15 = 10.5
		Dribbling:   60, // 60 * 0.05 = 3 -> Sum = 80
	}
	ovrDEF := ge.CalculateOVR("def_test", "DEF")
	if ovrDEF != 80 {
		t.Errorf("Expected DEF OVR 80, got %d", ovrDEF)
	}

	// Potential cap clipping
	ge.Biometrics["fwd_cap"] = &BiometricProfile{Potential: 82}
	ge.Attributes["fwd_cap"] = &TechnicalAttributes{
		Pace: 99, Shooting: 99, Dribbling: 99, Passing: 99, Physicality: 99,
	}
	ovrCapped := ge.CalculateOVR("fwd_cap", "FWD")
	if ovrCapped != 82 {
		t.Errorf("Expected capped OVR 82, got %d", ovrCapped)
	}

	// Minimum clamp (60)
	ge.Attributes["low_test"] = &TechnicalAttributes{
		Pace: 30, Shooting: 30, Dribbling: 30, Passing: 30, Physicality: 30,
	}
	ovrLow := ge.CalculateOVR("low_test", "FWD")
	if ovrLow != 60 {
		t.Errorf("Expected minimum floor OVR 60, got %d", ovrLow)
	}
}

func TestGrowthEngine_ApplyAgingDecline(t *testing.T) {
	ge := NewGrowthEngine(42)

	ge.Attributes["vet"] = &TechnicalAttributes{
		Pace:        70,
		Stamina:     70,
		Strength:    70,
		Physicality: 70,
		Shooting:    85, // Technical attributes should not drop directly
	}

	// Age 28 (<30): zero drop
	changed28 := ge.ApplyAgingDecline("vet", 28)
	if len(changed28) != 0 {
		t.Errorf("Age < 30 should have 0 changed attributes, got %v", changed28)
	}
	if ge.Attributes["vet"].Pace != 70 {
		t.Errorf("Age 28 pace should remain 70, got %d", ge.Attributes["vet"].Pace)
	}

	// Age 31 (30-33): -1 drop
	changed31 := ge.ApplyAgingDecline("vet", 31)
	if len(changed31) != 4 {
		t.Errorf("Age 31 should drop 4 physical attributes, got %v", changed31)
	}
	if ge.Attributes["vet"].Pace != 69 || ge.Attributes["vet"].Stamina != 69 {
		t.Errorf("Expected pace 69, stamina 69, got pace %d, stamina %d",
			ge.Attributes["vet"].Pace, ge.Attributes["vet"].Stamina)
	}
	if ge.Attributes["vet"].Shooting != 85 {
		t.Errorf("Shooting should remain unchanged at 85, got %d", ge.Attributes["vet"].Shooting)
	}

	// Age 34 (34-35): -2 drop
	changed34 := ge.ApplyAgingDecline("vet", 34)
	if len(changed34) != 4 {
		t.Errorf("Age 34 should drop 4 physical attributes, got %v", changed34)
	}
	if ge.Attributes["vet"].Pace != 67 {
		t.Errorf("Expected pace 67, got %d", ge.Attributes["vet"].Pace)
	}

	// Age 37 (36+): -3 drop
	changed37 := ge.ApplyAgingDecline("vet", 37)
	if len(changed37) != 4 {
		t.Errorf("Age 37 should drop 4 physical attributes, got %v", changed37)
	}
	if ge.Attributes["vet"].Pace != 64 {
		t.Errorf("Expected pace 64, got %d", ge.Attributes["vet"].Pace)
	}

	// Test Floor 35 enforcement
	ge.Attributes["old_vet"] = &TechnicalAttributes{
		Pace:        36,
		Stamina:     35,
		Strength:    34,
		Physicality: 38,
	}
	ge.ApplyAgingDecline("old_vet", 38) // -3 drop
	attrs := ge.Attributes["old_vet"]
	if attrs.Pace != 35 {
		t.Errorf("Expected pace floor 35, got %d", attrs.Pace)
	}
	if attrs.Stamina != 35 {
		t.Errorf("Expected stamina floor 35, got %d", attrs.Stamina)
	}
	if attrs.Strength != 35 {
		t.Errorf("Expected strength floor 35 (clamped from 34), got %d", attrs.Strength)
	}
	if attrs.Physicality != 35 {
		t.Errorf("Expected physicality 35 (38 - 3), got %d", attrs.Physicality)
	}

	// Running aging on an already-floored player should return 0 changes
	changedFloored := ge.ApplyAgingDecline("old_vet", 38)
	if len(changedFloored) != 0 {
		t.Errorf("Expected 0 changes when already at floor 35, got %v", changedFloored)
	}
}

func TestSeasonalOVRDrop(t *testing.T) {
	// Age < 30: unchanged
	if got := SeasonalOVRDrop(27, 85); got != 85 {
		t.Errorf("Expected 85 for age 27, got %d", got)
	}

	// Age 30-33: -1
	if got := SeasonalOVRDrop(32, 85); got != 84 {
		t.Errorf("Expected 84 for age 32, got %d", got)
	}

	// Age 34-35: -2
	if got := SeasonalOVRDrop(35, 85); got != 83 {
		t.Errorf("Expected 83 for age 35, got %d", got)
	}

	// Age 36+: -3
	if got := SeasonalOVRDrop(38, 85); got != 82 {
		t.Errorf("Expected 82 for age 38, got %d", got)
	}

	// Floor 55
	if got := SeasonalOVRDrop(38, 56); got != 55 {
		t.Errorf("Expected floor 55, got %d", got)
	}
	if got := SeasonalOVRDrop(38, 55); got != 55 {
		t.Errorf("Expected floor 55, got %d", got)
	}
}

func TestGrowthEngine_ApplySeasonalGrowth(t *testing.T) {
	ge := NewGrowthEngine(42)

	// Age 25+: no growth
	if got := ge.ApplySeasonalGrowth("non_engine_vet", 26, 30, 95, "FWD", 80); got != 80 {
		t.Errorf("Player age 26 should not grow, got %d", got)
	}

	// Young player (<25) without engine attrs
	// apps < 8 -> +1
	if got := ge.ApplySeasonalGrowth("p_low_apps", 18, 5, 90, "FWD", 72); got != 73 {
		t.Errorf("Expected 72 + 1 = 73, got %d", got)
	}
	// apps 8-19 -> +2
	if got := ge.ApplySeasonalGrowth("p_mid_apps", 18, 12, 90, "FWD", 72); got != 74 {
		t.Errorf("Expected 72 + 2 = 74, got %d", got)
	}
	// apps >= 20 -> +3
	if got := ge.ApplySeasonalGrowth("p_high_apps", 18, 25, 90, "FWD", 72); got != 75 {
		t.Errorf("Expected 72 + 3 = 75, got %d", got)
	}

	// Potential ceiling bounds check (never exceeds potential ceiling, never 99)
	potCapped := ge.ApplySeasonalGrowth("p_pot_cap", 18, 30, 94, "FWD", 93)
	if potCapped != 94 {
		t.Errorf("Expected potential ceiling clamp to 94, got %d", potCapped)
	}

	alreadyAtPot := ge.ApplySeasonalGrowth("p_at_pot", 18, 30, 94, "FWD", 94)
	if alreadyAtPot != 94 {
		t.Errorf("Expected 94 when already at potential, got %d", alreadyAtPot)
	}

	// Test with player registered in engine
	ge.RegisterProdigy("reg_kid", "Reg Kid", 16, 175.0, 68.0, "FWD", 78, 95, 19)
	grownOVR := ge.ApplySeasonalGrowth("reg_kid", 16, 25, 95, "FWD")
	if grownOVR < 79 {
		t.Errorf("Expected registered prodigy to grow from 78 to at least 79, got %d", grownOVR)
	}
	if grownOVR > 95 {
		t.Errorf("Registered prodigy should never exceed potential 95, got %d", grownOVR)
	}
}

func TestGrowthEngine_PubertySimulation(t *testing.T) {
	// Seed deterministically
	ge := NewGrowthEngine(12345)

	ge.RegisterProdigy("WK_Test", "Test Wonderkid", 14, 165.0, 55.0, "FWD", 76, 95, 19)
	bio := ge.Biometrics["WK_Test"]

	initialHeight := bio.CurrentHeightCM
	initialWeight := bio.CurrentWeightKG

	// Run multiple puberty cycles to simulate a season (e.g. 38 matchweeks)
	for week := 1; week <= 38; week++ {
		ge.SimulatePubertyCycle("WK_Test", week)
	}

	// Verify height gained is within annual cap of 2.6 cm for age 14
	if bio.YearlyHeightTaken > 2.65 {
		t.Errorf("Yearly height taken %.2f exceeded 2.6 cm cap", bio.YearlyHeightTaken)
	}
	if bio.HeightGainCM() > bio.GrowthVelocity {
		t.Errorf("Height gain %.2f exceeded velocity %.2f", bio.HeightGainCM(), bio.GrowthVelocity)
	}

	// Verify weight gain limit (5.0 kg)
	if bio.WeightGainKG() > 5.05 {
		t.Errorf("Weight gain %.2f exceeded 5.0 kg cap", bio.WeightGainKG())
	}

	t.Logf("Simulated 38 weeks: Height %.1f -> %.1f (+%.1f cm), Weight %.1f -> %.1f (+%.1f kg), Milestones: %d",
		initialHeight, bio.CurrentHeightCM, bio.HeightGainCM(),
		initialWeight, bio.CurrentWeightKG, bio.WeightGainKG(),
		len(ge.Milestones))
}

func TestGrowthEngine_MentorshipAndXP(t *testing.T) {
	ge := NewGrowthEngine(999)

	ge.RegisterProdigy("WK_Mentee", "Mentee Player", 16, 172.0, 62.0, "MID", 75, 95, 19)
	mentorOVR := 88
	mentorName := "Senior Maestro"
	personality := "dedicated_pro"

	ge.SetMentorship("WK_Mentee", "M_01", mentorName, mentorOVR, personality)

	bio := ge.Biometrics["WK_Mentee"]
	attrs := ge.Attributes["WK_Mentee"]
	initialXP := bio.AccumulatedXP

	// Apply match XP with high rating and goals
	opts := MatchXPOptions{
		MentorOVR:   &mentorOVR,
		MentorName:  &mentorName,
		Personality: &personality,
	}
	events := ge.ApplyMatchXP("WK_Mentee", "Mentee Player", "MID", 8.5, 1, 1, opts)

	if bio.AccumulatedXP <= initialXP && len(events) == 0 {
		t.Errorf("Expected accumulated XP or level up event, got XP %.1f", bio.AccumulatedXP)
	}

	// Test weekly mentorship tick
	initialComposure := attrs.Composure
	mOpts := MentorshipOptions{
		MentorName:  &mentorName,
		MentorOVR:   &mentorOVR,
		Personality: &personality,
	}

	// Run multiple ticks to observe composure / drills / XP injection
	for i := 0; i < 20; i++ {
		ge.ApplyMentorshipTick("WK_Mentee", "Mentee Player", mOpts)
	}

	if attrs.Composure < initialComposure {
		t.Errorf("Composure should not decrease under mentorship: %d vs %d",
			attrs.Composure, initialComposure)
	}
}

func TestGrowthEngine_TrainingCycles(t *testing.T) {
	ge := NewGrowthEngine(777)

	ge.RegisterProdigy("WK_Trainee", "Trainee Player", 15, 170.0, 60.0, "FWD", 75, 94, 19)
	attrs := ge.Attributes["WK_Trainee"]

	if ge.TrainingEnergy != 3 {
		t.Fatalf("Initial training energy should be 3, got %d", ge.TrainingEnergy)
	}

	// 1. Hypertrophy
	initialStrength := attrs.Strength
	res1, err := ge.RunTrainingCycle("WK_Trainee", "hypertrophy", true)
	if err != nil {
		t.Fatalf("Hypertrophy cycle failed: %v", err)
	}
	if attrs.Strength != initialStrength+1 {
		t.Errorf("Expected strength %d, got %d", initialStrength+1, attrs.Strength)
	}
	if ge.TrainingEnergy != 2 {
		t.Errorf("Expected 2 remaining energy, got %d", ge.TrainingEnergy)
	}

	// 2. Technical
	initialDribbling := attrs.Dribbling
	res2, err := ge.RunTrainingCycle("WK_Trainee", "technical", true)
	if err != nil {
		t.Fatalf("Technical cycle failed: %v", err)
	}
	if attrs.Dribbling != initialDribbling+1 {
		t.Errorf("Expected dribbling %d, got %d", initialDribbling+1, attrs.Dribbling)
	}
	if ge.TrainingEnergy != 1 {
		t.Errorf("Expected 1 remaining energy, got %d", ge.TrainingEnergy)
	}

	// 3. Tactical
	initialPace := attrs.Pace
	res3, err := ge.RunTrainingCycle("WK_Trainee", "tactical", true)
	if err != nil {
		t.Fatalf("Tactical cycle failed: %v", err)
	}
	if attrs.Pace != initialPace+1 {
		t.Errorf("Expected pace %d, got %d", initialPace+1, attrs.Pace)
	}
	if ge.TrainingEnergy != 0 {
		t.Errorf("Expected 0 remaining energy, got %d", ge.TrainingEnergy)
	}

	// 4. Exhausted energy error check
	_, errExhausted := ge.RunTrainingCycle("WK_Trainee", "technical", true)
	if errExhausted == nil {
		t.Fatalf("Expected error when running training with 0 energy, got nil")
	}

	// 5. Replenish energy
	ge.ReplenishTrainingEnergy()
	if ge.TrainingEnergy != 3 {
		t.Errorf("Expected energy replenished to 3, got %d", ge.TrainingEnergy)
	}

	_ = res1
	_ = res2
	_ = res3
}

func TestGrowthEngine_TimelineAndProdigyData(t *testing.T) {
	ge := NewGrowthEngine(42)

	ge.RegisterProdigy("WK_Timeline", "Timeline Wonderkid", 14, 168.0, 58.0, "FWD", 76, 95, 19)

	// Record season snapshots
	ge.RecordTimelineEntry("WK_Timeline", "2026/27", 14, 76, 168.0, 58.0, 12, 6, 24, "TOT", "Harry Kane")
	ge.RecordTimelineEntry("WK_Timeline", "2027/28", 15, 80, 170.5, 61.2, 18, 9, 32, "TOT", "Harry Kane")

	history := ge.GetProgressionHistory("WK_Timeline")
	if len(history) != 2 {
		t.Fatalf("Expected 2 timeline entries, got %d", len(history))
	}
	if history[0].Season != "2026/27" || history[1].Season != "2027/28" {
		t.Errorf("Unexpected timeline seasons: %s, %s", history[0].Season, history[1].Season)
	}

	// Overwrite existing season snapshot
	ge.RecordTimelineEntry("WK_Timeline", "2027/28", 15, 81, 171.0, 61.5, 20, 10, 34, "TOT", "Harry Kane")
	updatedHistory := ge.GetProgressionHistory("WK_Timeline")
	if len(updatedHistory) != 2 {
		t.Fatalf("Expected still 2 timeline entries after update, got %d", len(updatedHistory))
	}
	if updatedHistory[1].OVR != 81 || updatedHistory[1].Goals != 20 {
		t.Errorf("Expected updated OVR 81, goals 20, got OVR %d, goals %d",
			updatedHistory[1].OVR, updatedHistory[1].Goals)
	}

	// Verify Wonderkid Lab UI data
	data, ok := ge.GetProdigyData("WK_Timeline", "FWD")
	if !ok {
		t.Fatalf("Expected GetProdigyData to succeed")
	}
	if data["full_name"] != "Timeline Wonderkid" {
		t.Errorf("Expected name 'Timeline Wonderkid', got %v", data["full_name"])
	}
	if data["height_display"] != "168 cm (5'6\")" {
		t.Errorf("Expected height display '168 cm (5'6\")', got %v", data["height_display"])
	}
}

func TestGrowthEngine_DeterministicSeeding(t *testing.T) {
	ge1 := NewGrowthEngine(123456)
	ge2 := NewGrowthEngine(123456)

	ge1.RegisterProdigy("P1", "Deterministic Player", 14, 165.0, 56.0, "FWD", 76, 95, 19)
	ge2.RegisterProdigy("P1", "Deterministic Player", 14, 165.0, 56.0, "FWD", 76, 95, 19)

	for week := 1; week <= 10; week++ {
		e1 := ge1.SimulatePubertyCycle("P1", week)
		e2 := ge2.SimulatePubertyCycle("P1", week)
		if len(e1) != len(e2) {
			t.Fatalf("Week %d: simulation events count differ: %d vs %d", week, len(e1), len(e2))
		}
	}

	bio1 := ge1.Biometrics["P1"]
	bio2 := ge2.Biometrics["P1"]
	if bio1.CurrentHeightCM != bio2.CurrentHeightCM {
		t.Errorf("Height mismatch with deterministic seed: %.2f vs %.2f",
			bio1.CurrentHeightCM, bio2.CurrentHeightCM)
	}
	if bio1.CurrentWeightKG != bio2.CurrentWeightKG {
		t.Errorf("Weight mismatch with deterministic seed: %.2f vs %.2f",
			bio1.CurrentWeightKG, bio2.CurrentWeightKG)
	}
}

func TestGrowthEngine_ThreadSafety(t *testing.T) {
	ge := NewGrowthEngine(42)

	// Register 5 prodigies
	for i := 1; i <= 5; i++ {
		id := fmt.Sprintf("P_%d", i)
		name := fmt.Sprintf("Parallel Player %d", i)
		ge.RegisterProdigy(id, name, 15, 170.0, 60.0, "FWD", 75, 94, 19)
	}

	var wg sync.WaitGroup
	workers := 20
	iterations := 50

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for it := 0; it < iterations; it++ {
				pID := fmt.Sprintf("P_%d", (it%5)+1)

				switch it % 6 {
				case 0:
					ge.CalculateOVR(pID, "FWD")
				case 1:
					ge.SimulatePubertyCycle(pID, it)
				case 2:
					ge.ApplyMatchXP(pID, "Parallel Player", "FWD", 7.5, 1, 0)
				case 3:
					ge.ApplyAgingDecline(pID, 32)
				case 4:
					ge.GetProdigyData(pID, "FWD")
				case 5:
					ge.RunTrainingCycle(pID, "technical", false)
				}
			}
		}(w)
	}

	wg.Wait()
}

func TestGrowthEngine_ResetYearlyHeightTaken(t *testing.T) {
	ge := NewGrowthEngine(42)
	ge.RegisterProdigy("P_Reset", "Reset Wonderkid", 14, 165.0, 55.0, "FWD", 76, 95, 19)
	bio := ge.Biometrics["P_Reset"]
	bio.YearlyHeightTaken = 2.4

	ge.ResetYearlyHeightTaken("P_Reset")
	if bio.YearlyHeightTaken != 0.0 {
		t.Errorf("Expected YearlyHeightTaken reset to 0.0, got %.2f", bio.YearlyHeightTaken)
	}
}

func TestGrowthEngine_Helpers(t *testing.T) {
	ge := NewGrowthEngine(42)
	ge.RegisterProdigy("P_Helper", "Helper Player", 15, 170.0, 60.0, "FWD", 75, 94, 19)

	// ApplyAgingDeclineHelper
	changed := ApplyAgingDeclineHelper("P_Helper", 32, ge)
	if len(changed) != 4 {
		t.Errorf("Expected 4 attributes dropped for age 32, got %v", changed)
	}

	// ApplyAgingDeclineHelper with nil fallback to default engine
	SetDefaultGrowthEngine(ge)
	changedDefault := ApplyAgingDeclineHelper("P_Helper", 34)
	if len(changedDefault) != 4 {
		t.Errorf("Expected 4 attributes dropped via default engine, got %v", changedDefault)
	}

	// ApplySeasonalGrowthHelper
	newOVR := ApplySeasonalGrowthHelper("P_Helper", 15, 94, 25, "FWD", ge)
	if newOVR < 75 || newOVR > 94 {
		t.Errorf("Expected OVR within [75, 94], got %d", newOVR)
	}

	// Static fallback for ApplySeasonalGrowthHelper when no engine
	staticOVR := ApplySeasonalGrowthHelper("unknown", 18, 90, 22, "FWD", nil, 72)
	if staticOVR != 75 {
		t.Errorf("Expected static growth 72 + 3 = 75, got %d", staticOVR)
	}
}

