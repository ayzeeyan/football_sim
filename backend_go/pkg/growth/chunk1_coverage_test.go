package growth

import (
	"math"
	"strings"
	"testing"
)

// Supplemental Chunk 1 coverage: growth engine edge paths, helpers, and
// adversarial seasonal-growth / mentorship / puberty branches.

func TestChunk1CovEngineBasics(t *testing.T) {
	ge := NewGrowthEngine(0) // time-seeded path
	if ge == nil {
		t.Fatal("NewGrowthEngine(0) returned nil")
	}
	if GetDefaultGrowthEngine() != ge {
		t.Errorf("default engine not registered")
	}
	ge.TrainingEnergy = 0
	ge.ReplenishTrainingEnergy()
	if ge.TrainingEnergy != ge.MaxTrainingEnergy {
		t.Errorf("energy not replenished: %d", ge.TrainingEnergy)
	}
	custom := NewGrowthEngine(7)
	SetDefaultGrowthEngine(custom)
	if GetDefaultGrowthEngine() != custom {
		t.Errorf("SetDefaultGrowthEngine failed")
	}
}

func TestChunk1CovStillGrowingBranches(t *testing.T) {
	ge := NewGrowthEngine(11)
	// Adult by age: sets Adult frame, returns false.
	adult := &BiometricProfile{Age: 20, AdultHeightAge: 19, PubertyStage: "Late-puberty"}
	if ge.StillGrowing(adult) {
		t.Errorf("adult should not still be growing")
	}
	if adult.PubertyStage != "Adult frame" {
		t.Errorf("adult stage = %q", adult.PubertyStage)
	}
	// Growth budget exhausted.
	done := &BiometricProfile{Age: 15, AdultHeightAge: 19, CurrentHeightCM: 180, BaselineHeightCM: 170, GrowthVelocity: 9.9}
	if ge.StillGrowing(done) {
		t.Errorf("budget-exhausted should not still be growing")
	}
	// Still growing.
	young := &BiometricProfile{Age: 14, AdultHeightAge: 19, CurrentHeightCM: 170, BaselineHeightCM: 170, GrowthVelocity: 9.0}
	if !ge.StillGrowing(young) {
		t.Errorf("young prodigy should still be growing")
	}
}

func TestChunk1CovAttrAccessors(t *testing.T) {
	a := &TechnicalAttributes{}
	keys := []string{"pace", "shooting", "passing", "dribbling", "defending", "physicality",
		"aerial_reach", "heading_power", "strength", "shielding", "press_resistance", "stamina", "composure"}
	for i, k := range keys {
		setAttr(a, k, 50+i)
		if got := getAttr(a, k); got != 50+i {
			t.Errorf("attr %s round-trip = %d; want %d", k, got, 50+i)
		}
	}
	if getAttr(a, "bogus") != 0 {
		t.Errorf("unknown attr should read 0")
	}
	before := *a
	setAttr(a, "bogus", 99) // no-op, must not panic or mutate
	if *a != before {
		t.Errorf("unknown attr write must be a no-op")
	}
	if minInt(2, 5) != 2 || minInt(5, 2) != 2 || maxInt(2, 5) != 5 || maxInt(5, 2) != 5 {
		t.Errorf("min/max helpers wrong")
	}
}

func TestChunk1CovCalculateOVREdges(t *testing.T) {
	ge := NewGrowthEngine(21)
	if got := ge.CalculateOVR("ghost", "FWD"); got != 75 {
		t.Errorf("unknown player OVR = %d; want 75", got)
	}
	// Floor at 60: seed then crush attributes to minimums.
	ge.RegisterProdigy("low1", "Low One", 16, 175, 70, "MID", 60, 90, 19)
	attrs := ge.Attributes["low1"]
	for _, k := range []string{"pace", "shooting", "passing", "dribbling", "defending", "physicality"} {
		setAttr(attrs, k, 30)
	}
	if got := ge.CalculateOVR("low1", "MID"); got != 60 {
		t.Errorf("OVR floor = %d; want 60", got)
	}
	// Potential cap: raw well above potential clamps down.
	ge.RegisterProdigy("cap1", "Cap One", 16, 180, 75, "FWD", 85, 86, 19)
	bio := ge.Biometrics["cap1"]
	bio.Potential = 80
	if got := ge.CalculateOVR("cap1", "FWD"); got != 80 {
		t.Errorf("potential cap = %d; want 80", got)
	}
	// DEF weighting path with defaults.
	ge.RegisterProdigy("def1", "Def One", 15, 185, 78, "DEF", 70, 92, 19)
	if got := ge.CalculateOVR("def1", "DEF"); got < 60 || got > 92 {
		t.Errorf("DEF OVR out of range: %d", got)
	}
	// internalNudgeToOVR on unknown player is a no-op.
	ge.internalNudgeToOVR("ghost", "FWD", 90)
	// Nudge downward path (current > target, step -1).
	ge.internalNudgeToOVR("cap1", "FWD", 70)
	if got := ge.CalculateOVR("cap1", "FWD"); got > 80 {
		t.Errorf("downward nudge should respect potential cap: %d", got)
	}
	// Saturated attributes: all at 99, nudge cannot move (covers no-change path).
	sat := ge.Attributes["cap1"]
	for _, k := range []string{"pace", "shooting", "passing", "dribbling", "defending", "physicality"} {
		setAttr(sat, k, 99)
	}
	ge.internalNudgeToOVR("cap1", "FWD", 95)
}

func TestChunk1CovRegisterProdigyAgeBuckets(t *testing.T) {
	ge := NewGrowthEngine(31)
	// Adult registration: no remaining growth.
	bio, _ := ge.RegisterProdigy("adult1", "Adult One", 25, 185, 80, "DEF", 75, 85, 19)
	if bio.GrowthVelocity != 0 || bio.PubertyStage != "Adult frame" {
		t.Errorf("adult reg wrong: %+v", bio)
	}
	// yearsLeft < 0 clamps to 0.
	bio2, _ := ge.RegisterProdigy("old1", "Old One", 30, 185, 82, "DEF", 75, 85, 19)
	if bio2.GrowthVelocity != 0 {
		t.Errorf("over-age reg should have 0 velocity: %+v", bio2)
	}
	// Mid-puberty and late-puberty buckets.
	mid, _ := ge.RegisterProdigy("mid1", "Mid One", 16, 175, 68, "MID", 70, 90, 20)
	if mid.PubertyStage != "Mid-puberty" {
		t.Errorf("age 16 stage = %q", mid.PubertyStage)
	}
	late, _ := ge.RegisterProdigy("late1", "Late One", 18, 180, 74, "FWD", 72, 90, 20)
	if late.PubertyStage != "Late-puberty" {
		t.Errorf("age 18 stage = %q", late.PubertyStage)
	}
	early, _ := ge.RegisterProdigy("early1", "Early One", 13, 165, 55, "FWD", 68, 93, 19)
	if early.PubertyStage != "Early-puberty" {
		t.Errorf("age 13 stage = %q", early.PubertyStage)
	}
	if v := AdultHeightAgeFor("x", 17); v < 18 || v > 20 {
		t.Errorf("invalid configured age should hash: %d", v)
	}
}

func TestChunk1CovAgingDeclineEdges(t *testing.T) {
	ge := NewGrowthEngine(41)
	// Unknown player returns empty.
	if got := ge.ApplyAgingDecline("ghost", 35); len(got) != 0 {
		t.Errorf("unknown player decline = %v; want empty", got)
	}
	ge.RegisterProdigy("vet1", "Vet One", 34, 182, 78, "DEF", 80, 88, 19)
	// Mid band (34-35): drop 2.
	if got := ge.ApplyAgingDecline("vet1", 35); len(got) != 4 {
		t.Errorf("age 35 decline = %v; want 4 attrs", got)
	}
	// Senior band (36+): drop 3.
	if got := ge.ApplyAgingDecline("vet1", 37); len(got) != 4 {
		t.Errorf("age 37 decline = %v; want 4 attrs", got)
	}
	// At floor 35: no changes reported.
	attrs := ge.Attributes["vet1"]
	for _, k := range []string{"pace", "stamina", "strength", "physicality"} {
		setAttr(attrs, k, 35)
	}
	if got := ge.ApplyAgingDecline("vet1", 38); len(got) != 0 {
		t.Errorf("floored decline = %v; want empty", got)
	}
	// Helpers delegate to explicit and default engines.
	if got := ApplyAgingDeclineHelper("vet1", 36, ge); len(got) != 0 {
		t.Errorf("helper decline on floored = %v", got)
	}
	prev := GetDefaultGrowthEngine()
	SetDefaultGrowthEngine(ge)
	defer SetDefaultGrowthEngine(prev)
	if got := ApplyAgingDeclineHelper("ghost", 36); len(got) != 0 {
		t.Errorf("default-engine helper = %v", got)
	}
	SetDefaultGrowthEngine(nil)
	if got := ApplyAgingDeclineHelper("ghost", 36); len(got) != 0 {
		t.Errorf("nil-engine helper should be empty, got %v", got)
	}
	SetDefaultGrowthEngine(ge)
}

func TestChunk1CovSeasonalGrowthBranches(t *testing.T) {
	ge := NewGrowthEngine(51)
	ge.RegisterProdigy("yg1", "Young One", 17, 175, 68, "FWD", 72, 94, 19)

	// Auto-detect category from currentOVR.
	got := ge.ApplySeasonalGrowth("yg1", 17, 10, 94, "", 72)
	if got < 72 || got > 94 {
		t.Errorf("auto-cat growth = %d; want in [72,94]", got)
	}
	// Age 25+ with attrs returns current OVR (no youth growth).
	ge.RegisterProdigy("prime1", "Prime One", 26, 182, 78, "MID", 80, 88, 19)
	cur := ge.CalculateOVR("prime1", "MID")
	if got := ge.ApplySeasonalGrowth("prime1", 26, 30, 88, "MID"); got != cur {
		t.Errorf("veteran growth = %d; want current %d", got, cur)
	}
	// Age 25+, no attrs, with currentOVR -> echo.
	if got := ge.ApplySeasonalGrowth("ghost", 27, 30, 88, "MID", 81); got != 81 {
		t.Errorf("ghost veteran echo = %d; want 81", got)
	}
	// Age 25+, no attrs, no OVR -> 75 default.
	if got := ge.ApplySeasonalGrowth("ghost", 27, 30, 88, "MID"); got != 75 {
		t.Errorf("ghost veteran default = %d; want 75", got)
	}
	// No attrs, young, explicit currentOVR + appearances bump.
	if got := ge.ApplySeasonalGrowth("ghost", 18, 25, 90, "FWD", 70); got != 73 {
		t.Errorf("ghost youth bump = %d; want 73", got)
	}
	// No attrs, young, no OVR -> 70 base + bump.
	if got := ge.ApplySeasonalGrowth("ghost", 18, 3, 90, "FWD"); got != 71 {
		t.Errorf("ghost youth base = %d; want 71", got)
	}
	// currentOVR above computed base lifts the base.
	high := ge.ApplySeasonalGrowth("yg1", 17, 10, 94, "FWD", 90)
	if high < 90 || high > 94 {
		t.Errorf("lifted-base growth = %d; want in [90,94]", high)
	}
	// Guarantee branch: inflated currentOVR with a big gap the 24-step nudge
	// cannot close in one season -> exactly base+1 (capped by potential).
	ge2 := NewGrowthEngine(52)
	ge2.RegisterProdigy("gap1", "Gap One", 18, 176, 69, "FWD", 70, 95, 19)
	res := ge2.ApplySeasonalGrowth("gap1", 18, 25, 95, "FWD", 85)
	if res < 85 || res > 95 {
		t.Errorf("guarantee growth = %d; want in [85,95]", res)
	}
	// Static helper with nil engine (no default registered).
	prev := GetDefaultGrowthEngine()
	SetDefaultGrowthEngine(nil)
	defer SetDefaultGrowthEngine(prev)
	if got := ApplySeasonalGrowthHelper("x", 18, 90, 25, "FWD", nil, 70); got != 73 {
		t.Errorf("static helper bump = %d; want 73", got)
	}
	if got := ApplySeasonalGrowthHelper("x", 18, 90, 1, "FWD", nil); got != 71 {
		t.Errorf("static helper base = %d; want 71", got)
	}
	if got := ApplySeasonalGrowthHelper("x", 18, 90, 10, "FWD", nil, 70); got != 72 {
		t.Errorf("static helper mid bump = %d; want 72", got)
	}
	// Helper delegates to explicit engine.
	ge3 := NewGrowthEngine(53)
	if got := ApplySeasonalGrowthHelper("ghost", 27, 88, 5, "MID", ge3, 81); got != 81 {
		t.Errorf("explicit-engine helper = %d; want 81", got)
	}
	SetDefaultGrowthEngine(ge3)
	if got := ApplySeasonalGrowthHelper("ghost", 27, 88, 5, "MID", nil, 81); got != 81 {
		t.Errorf("default-engine helper = %d; want 81", got)
	}
}

func TestChunk1CovMatchXPAgeAndMentorBranches(t *testing.T) {
	mk := func(seed int64, id, name string, age int) *GrowthEngine {
		ge := NewGrowthEngine(seed)
		ge.RegisterProdigy(id, name, age, 176, 69, "FWD", 72, 94, 19)
		return ge
	}
	// Nil bio/attrs -> nil.
	ge0 := NewGrowthEngine(61)
	if out := ge0.ApplyMatchXP("ghost", "Ghost", "FWD", 7.5, 1, 0); out != nil {
		t.Errorf("ghost XP should be nil, got %v", out)
	}
	// Each age multiplier bucket.
	for _, age := range []int{15, 19, 23, 27} {
		ge := mk(100+int64(age), "xp1", "XP One", age)
		_ = ge.ApplyMatchXP("xp1", "XP One", "FWD", 7.0, 1, 1)
	}
	// MID and DEF pools.
	geM := NewGrowthEngine(62)
	geM.RegisterProdigy("midX", "Mid X", 17, 176, 69, "MID", 72, 94, 19)
	bio := geM.Biometrics["midX"]
	bio.AccumulatedXP = bio.LevelXPTarget // force level-up
	_ = geM.ApplyMatchXP("midX", "Mid X", "MID", 9.5, 2, 2)
	geD := NewGrowthEngine(63)
	geD.RegisterProdigy("defX", "Def X", 17, 182, 76, "DEF", 72, 94, 19)
	bioD := geD.Biometrics["defX"]
	bioD.AccumulatedXP = bioD.LevelXPTarget
	_ = geD.ApplyMatchXP("defX", "Def X", "DEF", 9.5, 0, 1)

	// Mentor low-OVR clamp (bonus floor 0.05) + dedicated_pro kicker.
	geL := mk(64, "lo1", "Lo One", 18)
	lowOVR := 60
	_ = geL.ApplyMatchXP("lo1", "Lo One", "FWD", 7.0, 0, 0, MatchXPOptions{MentorOVR: &lowOVR})
	// Mentor high-OVR clamp (bonus cap 0.25).
	geH := mk(65, "hi1", "Hi One", 18)
	hiOVR := 99
	pers := "dedicated_pro"
	mname := "Old Master"
	_ = geH.ApplyMatchXP("hi1", "Hi One", "FWD", 7.0, 0, 0,
		MatchXPOptions{MentorOVR: &hiOVR, MentorName: &mname, Personality: &pers})
	// big_game_performer composure chance with mentor at ceiling loop:
	// force many level-ups across seeds to hit composure + ceiling branches.
	bgp := "big_game_performer"
	for seed := int64(200); seed < 220; seed++ {
		g := NewGrowthEngine(seed)
		g.RegisterProdigy("c1", "Ceil One", 17, 176, 69, "FWD", 90, 91, 19)
		b := g.Biometrics["c1"]
		b.AccumulatedXP = b.LevelXPTarget * 3
		b.MentorName = "Vet"
		b.MentorOVR = 90
		b.Personality = bgp
		_ = g.ApplyMatchXP("c1", "Ceil One", "FWD", 10.0, 3, 3,
			MatchXPOptions{MentorName: &mname, MentorOVR: &hiOVR, Personality: &bgp})
	}
	// No level-up (XP below target) returns without events.
	geN := mk(66, "no1", "No One", 18)
	if out := geN.ApplyMatchXP("no1", "No One", "FWD", 6.0, 0, 0); len(out) != 0 {
		t.Errorf("sub-threshold XP should yield no events, got %v", out)
	}
}

func TestChunk1CovMentorshipTickBranches(t *testing.T) {
	ge := NewGrowthEngine(71)
	if out := ge.ApplyMentorshipTick("ghost", "Ghost"); out != nil {
		t.Errorf("ghost tick should be nil, got %v", out)
	}
	ge.RegisterProdigy("men1", "Men One", 16, 174, 66, "MID", 70, 92, 19)
	// No mentor assigned -> nil.
	if out := ge.ApplyMentorshipTick("men1", "Men One"); out != nil {
		t.Errorf("mentorless tick should be nil, got %v", out)
	}
	// Explicit overrides without bio mentor.
	mname := "Senior Pro"
	movr := 88
	pers := "academic_dual"
	seenProc := false
	seenMiss := false
	for seed := int64(300); seed < 340; seed++ {
		g := NewGrowthEngine(seed)
		g.RegisterProdigy("m2", "Men Two", 16, 174, 66, "MID", 70, 92, 19)
		out := g.ApplyMentorshipTick("m2", "Men Two",
			MentorshipOptions{MentorName: &mname, MentorOVR: &movr, Personality: &pers})
		if len(out) > 0 {
			seenProc = true
		} else {
			seenMiss = true
		}
	}
	if !seenProc || !seenMiss {
		t.Errorf("mentorship proc chance should hit both paths (proc=%v miss=%v)", seenProc, seenMiss)
	}
	// Each drill archetype across seeds.
	for _, p := range []string{"academic_dual", "big_game_performer", "flamboyant_star", "dedicated_pro"} {
		pp := p
		hit := false
		for seed := int64(400); seed < 460; seed++ {
			g := NewGrowthEngine(seed)
			g.RegisterProdigy("m3", "Men Three", 16, 174, 66, "MID", 70, 92, 19)
			// Lower composure so transfer path is open; empty personality on bio
			// exercises the dedicated_pro default.
			g.Attributes["m3"].Composure = 60
			out := g.ApplyMentorshipTick("m3", "Men Three",
				MentorshipOptions{MentorName: &mname, MentorOVR: &movr, Personality: &pp})
			if len(out) > 0 {
				hit = true
				break
			}
		}
		if !hit {
			t.Errorf("archetype %s never procced a tick", p)
		}
	}
	// Composure already maxed: skips transfer, still injects XP.
	g := NewGrowthEngine(72)
	g.RegisterProdigy("m4", "Men Four", 16, 174, 66, "MID", 70, 92, 19)
	g.Attributes["m4"].Composure = 99
	b := g.Biometrics["m4"]
	b.MentorName = mname
	b.MentorOVR = movr
	before := b.AccumulatedXP
	_ = g.ApplyMentorshipTick("m4", "Men Four")
	if b.AccumulatedXP <= before && len(g.Milestones) == 0 {
		t.Log("note: maxed-composure tick produced no milestone (acceptable)")
	}
}

func TestChunk1CovTrainingCycleBranches(t *testing.T) {
	ge := NewGrowthEngine(81)
	// No energy -> error path.
	ge.TrainingEnergy = 0
	if _, err := ge.RunTrainingCycle("ghost", "technical"); err == nil {
		t.Errorf("expected no-energy error")
	}
	// Unknown prodigy -> not-found error.
	ge.TrainingEnergy = 3
	if _, err := ge.RunTrainingCycle("ghost", "technical"); err == nil {
		t.Errorf("expected prodigy-not-found error")
	}
	ge.RegisterProdigy("tr1", "Train One", 14, 170, 60, "FWD", 70, 93, 20)
	// Hypertrophy with growth remaining (weight path).
	if _, err := ge.RunTrainingCycle("tr1", "hypertrophy", false); err != nil {
		t.Errorf("hypertrophy failed: %v", err)
	}
	// Hypertrophy consuming energy.
	if _, err := ge.RunTrainingCycle("tr1", "hypertrophy"); err != nil {
		t.Errorf("hypertrophy (consume) failed: %v", err)
	}
	// Hypertrophy at weight cap (no weight gain branch).
	bio := ge.Biometrics["tr1"]
	bio.CurrentWeightKG = bio.BaselineWeightKG + 5.0
	if _, err := ge.RunTrainingCycle("tr1", "hypertrophy", false); err != nil {
		t.Errorf("capped hypertrophy failed: %v", err)
	}
	// Technical focus.
	if res, err := ge.RunTrainingCycle("tr1", "technical", false); err != nil || res["status"] != "success" {
		t.Errorf("technical failed: %v %v", res, err)
	}
	// Tactical default + unknown focus string.
	if res, err := ge.RunTrainingCycle("tr1", "tactical", false); err != nil || res["status"] != "success" {
		t.Errorf("tactical failed: %v %v", res, err)
	}
	if _, err := ge.RunTrainingCycle("tr1", "mystery", false); err != nil {
		t.Errorf("unknown focus should default to tactical: %v", err)
	}
	// Adult (not growing) hypertrophy skips weight branch.
	ge.RegisterProdigy("tr2", "Train Two", 25, 185, 80, "DEF", 75, 85, 19)
	if _, err := ge.RunTrainingCycle("tr2", "hypertrophy", false); err != nil {
		t.Errorf("adult hypertrophy failed: %v", err)
	}
}

func TestChunk1CovPubertyBranches(t *testing.T) {
	ge := NewGrowthEngine(91)
	if out := ge.SimulatePubertyCycle("ghost"); out != nil {
		t.Errorf("ghost puberty should be nil, got %v", out)
	}
	// Done growing -> empty events.
	ge.RegisterProdigy("pb1", "Pub One", 25, 185, 80, "DEF", 75, 85, 19)
	if out := ge.SimulatePubertyCycle("pb1", 5); len(out) != 0 {
		t.Errorf("adult puberty should be empty, got %v", out)
	}
	// Young prodigies across many seeds/weeks: height spurts, weight gains,
	// aerial/stamina bumps, and adult-frame transitions.
	transitions := 0
	spurts := 0
	framings := 0
	for seed := int64(500); seed < 560; seed++ {
		g := NewGrowthEngine(seed)
		g.RegisterProdigy("pb2", "Pub Two", 14, 165, 55, "FWD", 68, 93, 20)
		for w := 1; w <= 60; w++ {
			for _, e := range g.SimulatePubertyCycle("pb2", w) {
				switch {
				case len(e) >= 12 && e[:12] == "Growth spurt":
					spurts++
				case len(e) >= 16 && e[:16] == "Athletic framing":
					framings++
				default:
					transitions++
				}
			}
			if !g.StillGrowing(g.Biometrics["pb2"]) {
				break
			}
		}
	}
	if spurts == 0 {
		t.Errorf("expected height spurts across 60 seeds")
	}
	if framings == 0 {
		t.Errorf("expected weight gains across 60 seeds")
	}
	t.Logf("puberty sweep: spurts=%d framings=%d transitions=%d", spurts, framings, transitions)
	// Yearly cap exhausted suppresses further height gain.
	g := NewGrowthEngine(92)
	g.RegisterProdigy("pb3", "Pub Three", 14, 165, 55, "FWD", 68, 93, 20)
	b := g.Biometrics["pb3"]
	b.YearlyHeightTaken = 2.6
	_ = g.SimulatePubertyCycle("pb3")
	// Weight cap suppresses further weight gain.
	b.YearlyHeightTaken = 0
	b.CurrentWeightKG = b.BaselineWeightKG + 5.0
	_ = g.SimulatePubertyCycle("pb3")
	// Tiny remaining budget -> hGain < 0.1 no-event path.
	b.CurrentWeightKG = b.BaselineWeightKG + 5.0
	b.GrowthVelocity = 0.05
	_ = g.SimulatePubertyCycle("pb3")
	g.ResetYearlyHeightTaken("pb3")
	if b.YearlyHeightTaken != 0 {
		t.Errorf("yearly height not reset")
	}
	g.ResetYearlyHeightTaken("ghost") // no-op, must not panic
}

func TestChunk1CovProdigyDataAndTimeline(t *testing.T) {
	ge := NewGrowthEngine(101)
	if _, ok := ge.GetProdigyData("ghost", "FWD"); ok {
		t.Errorf("ghost prodigy data should be missing")
	}
	ge.RegisterProdigy("pd1", "Prod One", 15, 172, 62, "MID", 71, 93, 19)
	ge.RecordTimelineEntry("pd1", "2026-27", 15, 71, 172, 62, 3, 2, 12, "ARS", "Mentor")
	ge.RecordTimelineEntry("pd1", "2026-27", 15, 72, 172.5, 62.5, 4, 2, 13, "ARS", "Mentor") // overwrite same season
	if h := ge.GetProgressionHistory("pd1"); len(h) != 1 || h[0].OVR != 72 {
		t.Errorf("timeline overwrite wrong: %+v", h)
	}
	data, ok := ge.GetProdigyData("pd1", "MID")
	if !ok {
		t.Fatal("prodigy data missing")
	}
	if data["ovr"] == nil || data["attributes"] == nil || data["height_display"] == nil {
		t.Errorf("prodigy data incomplete: %v", data)
	}
	// Zero XP target path (xpPct guard).
	ge.Biometrics["pd1"].LevelXPTarget = 0
	if _, ok := ge.GetProdigyData("pd1", "MID"); !ok {
		t.Errorf("zero-target prodigy data should still resolve")
	}
	// Mentorship linkage + empty personality preserved.
	ge.SetMentorship("pd1", "M9", "Mentor Nine", 86, "")
	if ge.Biometrics["pd1"].MentorName != "Mentor Nine" {
		t.Errorf("mentorship not set")
	}
	ge.SetMentorship("ghost", "M9", "Mentor Nine", 86, "dedicated_pro") // no-op
	_ = math.Pi
}

func TestChunk1CovSeasonalGrowthDefaultCategory(t *testing.T) {
	ge := NewGrowthEngine(102)
	// Empty posCat with no attributes defaults to FWD before the bump math.
	if got := ge.ApplySeasonalGrowth("ghost", 18, 5, 90, ""); got != 71 {
		t.Errorf("default-cat youth bump = %d; want 71", got)
	}
	if got := ge.ApplySeasonalGrowth("ghost", 27, 5, 88, ""); got != 75 {
		t.Errorf("default-cat veteran = %d; want 75", got)
	}
}

func TestChunk1CovMentorshipDrillArchetypes(t *testing.T) {
	// Each archetype must fire the 30% synergistic drill at least once
	// (identified by the "1-on-1 drills" message).
	mname := "Senior Pro"
	movr := 88
	for _, p := range []string{"academic_dual", "big_game_performer", "flamboyant_star", "dedicated_pro"} {
		pp := p
		hit := false
		for seed := int64(1000); seed < 1400 && !hit; seed++ {
			g := NewGrowthEngine(seed)
			g.RegisterProdigy("dr1", "Drill One", 16, 174, 66, "MID", 70, 92, 19)
			g.Attributes["dr1"].Composure = 60
			out := g.ApplyMentorshipTick("dr1", "Drill One",
				MentorshipOptions{MentorName: &mname, MentorOVR: &movr, Personality: &pp})
			for _, e := range out {
				if strings.Contains(e, "1-on-1 drills") {
					hit = true
					break
				}
			}
		}
		if !hit {
			t.Errorf("archetype %s never fired its mentorship drill", p)
		}
	}
}

func containsAdult(s string) bool {
	return strings.Contains(s, "adult height")
}

func TestChunk1CovPubertyLateCapAndAdultTransition(t *testing.T) {
	// Age > 16 exercises the 1.8cm yearly cap assignment.
	g := NewGrowthEngine(103)
	g.RegisterProdigy("late2", "Late Two", 17, 178, 72, "DEF", 72, 90, 20)
	_ = g.SimulatePubertyCycle("late2", 3)

	// Rig the growth budget so a spurt completes development mid-tick,
	// exercising the Adult-frame transition.
	seen := false
	for seed := int64(1); seed <= 60 && !seen; seed++ {
		e := NewGrowthEngine(seed)
		e.RegisterProdigy("trX", "Trans X", 14, 165, 55, "FWD", 68, 93, 20)
		b := e.Biometrics["trX"]
		b.CurrentHeightCM = b.BaselineHeightCM + 5.0
		b.GrowthVelocity = 5.1
		b.YearlyHeightTaken = 0
		for _, ev := range e.SimulatePubertyCycle("trX", 2) {
			if containsAdult(ev) {
				seen = true
				break
			}
		}
	}
	if !seen {
		t.Errorf("expected an adult-height transition across 60 seeds")
	}
}
