package models

import (
	"encoding/json"
	"testing"
)

func TestPlayerEffectiveOVR(t *testing.T) {
	p := &Player{
		OVR: 85,
	}

	// Less than 3 consecutive starts -> no fatigue drop
	p.ConsecutiveStarts = 0
	if p.EffectiveOVR() != 85 {
		t.Errorf("EffectiveOVR(0 starts) = %d; want 85", p.EffectiveOVR())
	}
	p.ConsecutiveStarts = 2
	if p.EffectiveOVR() != 85 {
		t.Errorf("EffectiveOVR(2 starts) = %d; want 85", p.EffectiveOVR())
	}

	// Consecutive starts = 3 -> drop 2
	p.ConsecutiveStarts = 3
	if p.EffectiveOVR() != 83 {
		t.Errorf("EffectiveOVR(3 starts) = %d; want 83", p.EffectiveOVR())
	}

	// Consecutive starts = 4 -> drop 3
	p.ConsecutiveStarts = 4
	if p.EffectiveOVR() != 82 {
		t.Errorf("EffectiveOVR(4 starts) = %d; want 82", p.EffectiveOVR())
	}

	// Consecutive starts = 5 -> drop 4 (capped)
	p.ConsecutiveStarts = 5
	if p.EffectiveOVR() != 81 {
		t.Errorf("EffectiveOVR(5 starts) = %d; want 81", p.EffectiveOVR())
	}
	p.ConsecutiveStarts = 10
	if p.EffectiveOVR() != 81 {
		t.Errorf("EffectiveOVR(10 starts) = %d; want 81", p.EffectiveOVR())
	}

	// Floor at 40
	pLow := &Player{OVR: 42, ConsecutiveStarts: 5}
	if pLow.EffectiveOVR() != 40 {
		t.Errorf("EffectiveOVR() should floor at 40; got %d", pLow.EffectiveOVR())
	}
}

func TestPlayerJSONUnmarshal(t *testing.T) {
	rawJSON := `{
		"player_id": "P001",
		"full_name": "Test Striker",
		"position": "CAM",
		"ovr": 80,
		"age": 22,
		"estimated_market_value_eur": 25000000,
		"season_stats": {
			"goals": 5,
			"assists": 3,
			"appearances": 10,
			"season": "2026-27"
		}
	}`

	var p Player
	if err := json.Unmarshal([]byte(rawJSON), &p); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if p.PlayerID != "P001" {
		t.Errorf("PlayerID = %q; want P001", p.PlayerID)
	}
	// Verify CAM is mapped to FWD
	if p.Category != "FWD" {
		t.Errorf("Category for CAM = %q; want FWD", p.Category)
	}
	if p.MarketValueEUR != 25000000 {
		t.Errorf("MarketValueEUR = %d; want 25000000", p.MarketValueEUR)
	}
	if p.Goals != 5 || p.Assists != 3 || p.Appearances != 10 {
		t.Errorf("Stats mismatch: G=%d A=%d App=%d", p.Goals, p.Assists, p.Appearances)
	}
	if p.WageEUR == 0 {
		t.Errorf("WageEUR should have been populated from OVR")
	}
}

func TestPlayerAvailabilityAndSchool(t *testing.T) {
	// Normal veteran player
	vet := &Player{
		PlayerID:          "P_VET",
		Age:               28,
		UniverseWonderkid: false,
	}
	if vet.IsUnavailable("super-league", 12) {
		t.Errorf("Veteran should not have school conflicts on exam week")
	}
	if vet.AvailabilityNote("super-league", 12) != "Available" {
		t.Errorf("Veteran availability note should be Available")
	}

	// Suspended player
	vet.SuspendedMatches = 1
	if !vet.IsUnavailable("super-league", 1) {
		t.Errorf("Suspended player must be unavailable")
	}
	if vet.AvailabilityNote("super-league", 1) != "Suspended · out 1 match" {
		t.Errorf("Unexpected suspension note: %q", vet.AvailabilityNote("super-league", 1))
	}
	vet.SuspendedMatches = 0

	// Injured player
	vet.InjuredMatches = 3
	vet.Injury = "hamstring strain"
	if !vet.IsUnavailable("super-league", 1) {
		t.Errorf("Injured player must be unavailable")
	}
	if vet.AvailabilityNote("super-league", 1) != "Hamstring strain · out 3 matches" {
		t.Errorf("Unexpected injury note: %q", vet.AvailabilityNote("super-league", 1))
	}

	// Wonderkid in middle school
	wk := &Player{
		PlayerID:          "WK_01",
		FullName:          "Young Prodigy",
		Age:               14,
		UniverseWonderkid: true,
		Education:         "middle_school",
	}

	// Regular league matchweek (e.g. week 5) -> available
	if wk.IsUnavailable("super-league", 5) {
		t.Errorf("Wonderkid should be available in regular league matchweek 5")
	}

	// Exam week (e.g. week 12) -> unavailable
	if !wk.IsUnavailable("super-league", 12) {
		t.Errorf("Wonderkid should be unavailable during exam week 12")
	}
	if wk.AvailabilityNote("super-league", 12) != "Exams" {
		t.Errorf("Expected note 'Exams', got: %q", wk.AvailabilityNote("super-league", 12))
	}

	// UCL fixture for middle school wonderkid -> unavailable
	if !wk.IsUnavailable("ucl", 3) {
		t.Errorf("Middle school wonderkid should be unavailable for UCL")
	}
	if wk.AvailabilityNote("ucl", 3) != "School" {
		t.Errorf("Expected note 'School', got: %q", wk.AvailabilityNote("ucl", 3))
	}

	// Dropout wonderkid -> no school conflict
	wk.Education = "dropout"
	if wk.IsUnavailable("ucl", 12) {
		t.Errorf("Dropout wonderkid should not have school conflicts")
	}
}

func TestPlayerEducationDecisions(t *testing.T) {
	// Want: football, loyalty 95, top 4 team -> high_school
	p1 := &Player{
		FullName:          "Football Kid",
		UniverseWonderkid: true,
		SchoolWant:        "football",
		Loyalty:           95,
	}
	if p1.DecideEducation(2, 5) != "high_school" {
		t.Errorf("High loyalty top team football kid should choose high_school")
	}

	// Want: football, normal loyalty -> dropout
	p2 := &Player{
		FullName:          "Football Kid 2",
		UniverseWonderkid: true,
		SchoolWant:        "football",
		Loyalty:           60,
	}
	if p2.DecideEducation(8, 15) != "dropout" {
		t.Errorf("Regular football kid should choose dropout")
	}

	// AdvanceEducation at 16
	p3 := &Player{
		FullName:          "Age 16 Kid",
		Age:               16,
		UniverseWonderkid: true,
		Education:         "middle_school",
		SchoolWant:        "football",
		Loyalty:           60,
	}
	action := p3.AdvanceEducation(6)
	if action != "left" || p3.Education != "dropout" {
		t.Errorf("AdvanceEducation at 16 failed: action=%q, edu=%q", action, p3.Education)
	}

	// AdvanceEducation at 18
	p4 := &Player{
		FullName:          "Age 18 Kid",
		Age:               18,
		UniverseWonderkid: true,
		Education:         "high_school",
	}
	action4 := p4.AdvanceEducation(6)
	if action4 != "graduated" || p4.Education != "graduated" {
		t.Errorf("AdvanceEducation at 18 failed: action=%q, edu=%q", action4, p4.Education)
	}
}

func TestPlayerCareerStatsAndLedger(t *testing.T) {
	p := &Player{
		Goals:         3,
		Assists:       2,
		Appearances:   5,
		CareerGoals:   20,
		CareerAssists: 15,
		CareerApps:    60,
		BestGoals:     10,
		BestAssists:   8,
		BestSeason:    "2024-25",
		Season:        "2026-27",
	}

	p.RecordGoal()
	p.RecordAssist()
	p.RecordAppearance()

	if p.AllTimeGoals() != 24 {
		t.Errorf("AllTimeGoals = %d; want 24", p.AllTimeGoals())
	}
	if p.AllTimeAssists() != 18 {
		t.Errorf("AllTimeAssists = %d; want 18", p.AllTimeAssists())
	}
	if p.AllTimeApps() != 66 {
		t.Errorf("AllTimeApps = %d; want 66", p.AllTimeApps())
	}

	// Live best season
	g, a, s := p.LiveBest("")
	if g != 10 || a != 8 || s != "2024-25" {
		t.Errorf("LiveBest should still be historical best: G=%d, A=%d, S=%s", g, a, s)
	}

	p.Goals = 15
	g2, _, s2 := p.LiveBest("2026-27")
	if g2 != 15 || s2 != "2026-27" {
		t.Errorf("LiveBest should reflect current season when exceeding best: G=%d, S=%s", g2, s2)
	}
}

func TestPlayerPersonalityAndMentor(t *testing.T) {
	p := &Player{
		Personality: "flamboyant_star",
		MentorID:    "M001",
		MentorName:  "Senior Mentor",
		MentorOVR:   86,
	}

	info := p.PersonalityInfo()
	if info.Key != "flamboyant_star" {
		t.Errorf("PersonalityInfo.Key = %q; want flamboyant_star", info.Key)
	}
	if p.PersonalityTitle() != "The Flamboyant Prodigy" {
		t.Errorf("PersonalityTitle = %q", p.PersonalityTitle())
	}
	if p.PersonalityBadge() != "Flamboyant Star" {
		t.Errorf("PersonalityBadge = %q", p.PersonalityBadge())
	}
	if !p.HasMentor() {
		t.Errorf("HasMentor should return true when mentor ID and name are set")
	}

	pNoMentor := &Player{}
	if pNoMentor.HasMentor() {
		t.Errorf("HasMentor should return false when no mentor is set")
	}
}

func TestPlayerEducationLabelsAndOptions(t *testing.T) {
	p := &Player{
		FullName:          "Test Kid",
		UniverseWonderkid: true,
		Education:         "middle_school",
		SchoolWant:        "school",
		Position:          "CAM",
		MarketValueEUR:    50000000,
		WageEUR:           150000,
	}

	// EducationLabel middle school
	lbl := p.EducationLabel()
	if lbl == "" || lbl == "—" {
		t.Errorf("EducationLabel for middle school unexpected: %q", lbl)
	}

	// High school
	p.Education = "high_school"
	if p.EducationLabel() != "High school · sits exam weeks" {
		t.Errorf("EducationLabel high_school: %q", p.EducationLabel())
	}

	// Dropout
	p.Education = "dropout"
	if p.EducationLabel() != "Left school · full-time football" {
		t.Errorf("EducationLabel dropout: %q", p.EducationLabel())
	}

	// Graduated
	p.Education = "graduated"
	if p.EducationLabel() != "Finished school" {
		t.Errorf("EducationLabel graduated: %q", p.EducationLabel())
	}

	// SchoolWantLine
	if p.SchoolWantLine() != "Wants to finish school" {
		t.Errorf("SchoolWantLine: %q", p.SchoolWantLine())
	}

	// PositionOptions
	opts := p.PositionOptions()
	if len(opts) != 2 || opts[0] != "CF" || opts[1] != "ST" {
		t.Errorf("PositionOptions for CAM = %v", opts)
	}

	// FormattedWage & FormattedValue
	if p.FormattedWage() != "€150k/wk" {
		t.Errorf("FormattedWage = %q; want €150k/wk", p.FormattedWage())
	}
	if p.FormattedValue() != "€50.0M" {
		t.Errorf("FormattedValue = %q; want €50.0M", p.FormattedValue())
	}
}

func TestPlayerDecideEducationBranches(t *testing.T) {
	// Want school, low loyalty, bad club place -> dropout
	pSchoolDrop := &Player{
		SchoolWant: "school",
		Loyalty:    30,
	}
	if pSchoolDrop.DecideEducation(12, 10) != "dropout" {
		t.Errorf("Low loyalty bad place school kid should dropout")
	}

	// Want school, normal conditions -> high_school
	pSchoolStay := &Player{
		SchoolWant: "school",
		Loyalty:    70,
	}
	if pSchoolStay.DecideEducation(3, 5) != "high_school" {
		t.Errorf("Normal school kid should stay in high_school")
	}

	// Want open, high score -> high_school
	pOpenHigh := &Player{
		SchoolWant: "open",
		Loyalty:    80,
	}
	if pOpenHigh.DecideEducation(4, 5) != "high_school" {
		t.Errorf("High score open kid should stay in high_school")
	}

	// Want open, low score -> dropout
	pOpenLow := &Player{
		SchoolWant: "open",
		Loyalty:    40,
	}
	if pOpenLow.DecideEducation(12, 20) != "dropout" {
		t.Errorf("Low score open kid should dropout")
	}

	// Non-wonderkid AdvanceEducation returns ""
	pNonWk := &Player{Age: 16, UniverseWonderkid: false}
	if pNonWk.AdvanceEducation(5) != "" {
		t.Errorf("Non-wonderkid AdvanceEducation should return empty string")
	}
}
