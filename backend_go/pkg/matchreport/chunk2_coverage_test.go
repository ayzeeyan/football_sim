package matchreport

import (
	"math"
	"math/rand"
	"strings"
	"testing"

	"football_sim/pkg/growth"
	"football_sim/pkg/models"
)

// Chunk 2 coverage: simulation helpers (sampling, picks, subs, stats,
// assembly) ported from match_report.py.

func TestChunk2RNGPrimitives(t *testing.T) {
	if ensureRNG(nil) == nil {
		t.Errorf("ensureRNG(nil) should return a fresh RNG")
	}
	fixed := rand.New(rand.NewSource(7))
	if ensureRNG(fixed) != fixed {
		t.Errorf("ensureRNG should echo a supplied RNG")
	}
	if got := Poisson(fixed, 0); got != 0 {
		t.Errorf("Poisson(0) = %d; want 0", got)
	}
	if got := Poisson(fixed, -2); got != 0 {
		t.Errorf("Poisson(-2) = %d; want 0", got)
	}
	seenPositive := false
	for i := 0; i < 200; i++ {
		v := Poisson(fixed, 5)
		if v < 0 || v > 30 {
			t.Fatalf("Poisson(5) out of range: %d", v)
		}
		if v > 0 {
			seenPositive = true
		}
	}
	if !seenPositive {
		t.Errorf("Poisson(5) never produced a positive count in 200 draws")
	}

	if got := weightedIndex(fixed, nil); got != -1 {
		t.Errorf("weightedIndex(empty) = %d; want -1", got)
	}
	for i := 0; i < 50; i++ {
		if got := weightedIndex(fixed, []float64{0, 0, 0}); got < 0 || got > 2 {
			t.Fatalf("uniform fallback out of range: %d", got)
		}
	}
	if got := weightedIndex(fixed, []float64{0, 0, 5}); got != 2 {
		t.Errorf("weightedIndex should pick the only positive weight, got %d", got)
	}
	if got := WeightedChoice(fixed, []float64{1, 2, 3}); got < 0 || got > 2 {
		t.Errorf("WeightedChoice out of range: %d", got)
	}

	if got := SampleMinutes(fixed, 0); got != nil {
		t.Errorf("SampleMinutes(0) should be nil, got %v", got)
	}
	many := SampleMinutes(fixed, 100)
	if len(many) != 90 {
		t.Errorf("SampleMinutes(100) capped = %d; want 90", len(many))
	}
	five := SampleMinutes(fixed, 5)
	if len(five) != 5 {
		t.Fatalf("SampleMinutes(5) = %d; want 5", len(five))
	}
	seen := map[int]bool{}
	prev := 0
	for _, m := range five {
		if m < 1 || m > 90 || m <= prev || seen[m] {
			t.Fatalf("minutes not unique sorted in 1..90: %v", five)
		}
		seen[m] = true
		prev = m
	}

	// Gamma/beta samplers: bounds plus the alpha<1 boost path.
	for i := 0; i < 100; i++ {
		if b := Beta(fixed, 2.8, 2.2); b <= 0 || b >= 1 {
			t.Fatalf("Beta(2.8,2.2) out of (0,1): %v", b)
		}
		if b := Beta(fixed, 0.5, 0.5); b <= 0 || b >= 1 {
			t.Fatalf("Beta(0.5,0.5) out of (0,1): %v", b)
		}
	}
	for i := 0; i < 3000; i++ {
		_ = gammaSample(fixed, 1.0)
	}
}

func TestChunk2MinuteDisplay(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	if got := MinuteDisplay(121, rng); got != "120+1'" {
		t.Errorf(">120 display = %q", got)
	}
	if got := MinuteDisplay(95, rng); got != "95'" {
		t.Errorf(">90 display = %q", got)
	}
	if got := MinuteDisplay(44, rng); !strings.HasPrefix(got, "45+") {
		t.Errorf("44 display = %q; want 45+N'", got)
	}
	if got := MinuteDisplay(45, rng); !strings.HasPrefix(got, "45+") {
		t.Errorf("45 display = %q; want 45+N'", got)
	}
	if got := MinuteDisplay(89, rng); !strings.HasPrefix(got, "90+") {
		t.Errorf("89 display = %q; want 90+N'", got)
	}
	if got := MinuteDisplay(90, rng); !strings.HasPrefix(got, "90+") {
		t.Errorf("90 display = %q; want 90+N'", got)
	}
	if got := MinuteDisplay(30, rng); got != "30'" {
		t.Errorf("plain display = %q", got)
	}
	if got := MinuteDisplay(10, nil); got != "10'" {
		t.Errorf("nil-rng display = %q", got)
	}
}

func TestChunk2Referees(t *testing.T) {
	if len(Referees) != 12 {
		t.Errorf("referee roster = %d; want 12", len(Referees))
	}
	if RefereePersonality("Michael Oliver") != "strict" {
		t.Errorf("Oliver should be strict")
	}
	if RefereePersonality("Daniele Orsato") != "lenient" {
		t.Errorf("Orsato should be lenient")
	}
	if RefereePersonality("Szymon Marciniak") != "balanced" {
		t.Errorf("Marciniak should be balanced")
	}
	if RefereePersonality("Nobody") != "balanced" {
		t.Errorf("unknown official should default to balanced")
	}
	rng := rand.New(rand.NewSource(9))
	if got := PickRefereeName(rng, "Michael Oliver"); got != "Michael Oliver" {
		t.Errorf("name hint should be honored, got %q", got)
	}
	for _, pers := range []string{"strict", "lenient", "balanced"} {
		got := PickRefereeName(rng, pers)
		if RefereePersonality(got) != pers {
			t.Errorf("personality hint %q drew %q (%s)", pers, got, RefereePersonality(got))
		}
	}
	drawn := PickRefereeName(rng, "???")
	found := false
	for _, n := range Referees {
		if n == drawn {
			found = true
		}
	}
	if !found {
		t.Errorf("random draw %q not on roster", drawn)
	}
	if RefCardMult("strict") != 1.20 || RefCardMult("lenient") != 0.80 || RefCardMult("balanced") != 1.0 || RefCardMult("??") != 1.0 {
		t.Errorf("card multipliers wrong")
	}
}

func chunk2mkPlayer(id, name, pos, cat string, ovr int) *models.Player {
	return &models.Player{PlayerID: id, FullName: name, Position: pos, Category: cat, OVR: ovr, Age: 25}
}

func TestChunk2PickScorer(t *testing.T) {
	rng := rand.New(rand.NewSource(11))
	if got := PickScorer(nil, 45, false, rng); got != nil {
		t.Errorf("empty XI scorer should be nil")
	}
	if got := PickScorer([]*models.Player{}, 45, false, nil); got != nil {
		t.Errorf("empty XI scorer (nil rng) should be nil")
	}
	xi := []*models.Player{
		chunk2mkPlayer("S1", "Striker", "ST", "FWD", 85),
		chunk2mkPlayer("M1", "Mid", "CM", "MID", 84),
		chunk2mkPlayer("D1", "Def", "CB", "DEF", 90),
		chunk2mkPlayer("G1", "Keeper", "GK", "GK", 90),
	}
	// Only FWD/MID are candidates: the 90-rated DEF/GK must never score.
	for i := 0; i < 100; i++ {
		s := PickScorer(xi, 45, false, rng)
		if s.Category != "FWD" && s.Category != "MID" {
			t.Fatalf("non-attacker picked as scorer: %+v", s)
		}
	}
	// All-DEF squad falls back to the whole XI.
	defs := []*models.Player{chunk2mkPlayer("D1", "A", "CB", "DEF", 80), chunk2mkPlayer("D2", "B", "CB", "DEF", 81)}
	if s := PickScorer(defs, 10, false, rng); s == nil {
		t.Errorf("fallback scorer should not be nil")
	}
	// Wonderkid boosts: mentor, big-game clutch/minute, flamboyant, composure.
	wk := &models.Player{PlayerID: "WK1", FullName: "Wonderkid", Position: "ST", Category: "FWD", OVR: 76,
		UniverseWonderkid: true, MentorName: "Vet", Personality: "big_game_performer", Composure: 85, Age: 16}
	plain := chunk2mkPlayer("P9", "Plain", "ST", "FWD", 76)
	wkWins, plainWins := 0, 0
	for i := 0; i < 300; i++ {
		if PickScorer([]*models.Player{wk, plain}, 80, true, rng) == wk {
			wkWins++
		} else {
			plainWins++
		}
	}
	if wkWins <= plainWins {
		t.Errorf("boosted wonderkid should outscore plain peer: %d vs %d", wkWins, plainWins)
	}
	// All four clutch/minute combinations execute.
	for _, tc := range [][2]interface{}{{80, true}, {80, false}, {10, true}, {10, false}} {
		_ = PickScorer([]*models.Player{wk, plain}, tc[0].(int), tc[1].(bool), rng)
	}
	flam := &models.Player{PlayerID: "WK2", FullName: "Flair", Position: "LW", Category: "FWD", OVR: 75,
		UniverseWonderkid: true, Personality: "flamboyant_star", Composure: 70, Age: 17}
	_ = PickScorer([]*models.Player{flam, plain}, 20, false, rng)
	quiet := &models.Player{PlayerID: "WK3", FullName: "Quiet", Position: "ST", Category: "FWD", OVR: 75,
		UniverseWonderkid: true, Personality: "dedicated_pro", Composure: 70, Age: 17}
	_ = PickScorer([]*models.Player{quiet, plain}, 20, false, rng)
}

func TestChunk2PickAssister(t *testing.T) {
	rng := rand.New(rand.NewSource(13))
	xi := []*models.Player{
		chunk2mkPlayer("S1", "Striker", "ST", "FWD", 85),
		chunk2mkPlayer("M1", "Mid", "CM", "MID", 84),
		chunk2mkPlayer("G1", "Keeper", "GK", "GK", 90),
	}
	if got := PickAssister(xi[:1], xi[0], rng); got != nil {
		t.Errorf("single-man XI should yield no assister")
	}
	// 72% assist rate: both outcomes must appear across seeds.
	var assisted, solo int
	for seed := int64(0); seed < 60; seed++ {
		r := rand.New(rand.NewSource(seed))
		if PickAssister(xi, xi[0], r) != nil {
			assisted++
		} else {
			solo++
		}
	}
	if assisted == 0 || solo == 0 {
		t.Errorf("assist rate should produce both outcomes: assisted=%d solo=%d", assisted, solo)
	}
	if assisted*100/(assisted+solo) < 55 || assisted*100/(assisted+solo) > 85 {
		t.Errorf("assist rate out of plausible band: %d%%", assisted*100/(assisted+solo))
	}
	// Two-man XI: the only teammate (even a GK) assists.
	duo := []*models.Player{xi[0], xi[2]}
	r2 := rand.New(rand.NewSource(1))
	hitGK := false
	for i := 0; i < 20 && !hitGK; i++ {
		if a := PickAssister(duo, duo[0], r2); a == duo[1] {
			hitGK = true
		}
	}
	if !hitGK {
		t.Errorf("lone teammate should assist sometimes")
	}
}

func TestChunk2AerialAndCornerTakers(t *testing.T) {
	rng := rand.New(rand.NewSource(15))
	if got := PickAerialTarget(nil, nil, rng); got != nil {
		t.Errorf("empty aerial XI should be nil")
	}
	xi := []*models.Player{
		chunk2mkPlayer("D1", "Def", "CB", "DEF", 82),
		chunk2mkPlayer("M1", "Mid", "CM", "MID", 84),
		chunk2mkPlayer("F1", "Fwd", "ST", "FWD", 83),
	}
	// No growth engine: OVR fallback with DEF/FWD bonus.
	for i := 0; i < 30; i++ {
		if got := PickAerialTarget(xi, nil, rng); got == nil {
			t.Fatalf("aerial target should not be nil")
		}
	}
	// Growth engine with full biometrics.
	ge := growth.NewGrowthEngine(21)
	ge.RegisterProdigy("D1", "Def", 20, 195, 88, "DEF", 82, 90, 19)
	ge.RegisterProdigy("M1", "Mid", 20, 170, 65, "MID", 84, 90, 19)
	ge.RegisterProdigy("F1", "Fwd", 20, 188, 80, "FWD", 83, 90, 19)
	giantWins := 0
	for i := 0; i < 100; i++ {
		if PickAerialTarget(xi, ge, rng) == xi[0] {
			giantWins++
		}
	}
	if giantWins < 40 {
		t.Errorf("195cm defender should dominate aerial picks, won %d/100", giantWins)
	}
	// Attributes without biometrics: partial-score branch.
	delete(ge.Biometrics, "D1")
	_ = PickAerialTarget(xi, ge, rng)
	// Unknown IDs: OVR fallback branch.
	_ = PickAerialTarget([]*models.Player{chunk2mkPlayer("ZZ", "Stranger", "CB", "DEF", 80)}, ge, rng)
	// All-GK squad falls back to the whole XI.
	gks := []*models.Player{chunk2mkPlayer("G1", "A", "GK", "GK", 80), chunk2mkPlayer("G2", "B", "GK", "GK", 81)}
	if got := PickAerialTarget(gks, nil, rng); got == nil {
		t.Errorf("GK-only aerial fallback should not be nil")
	}

	// Corner taker never selects the aerial target.
	target := xi[0]
	for i := 0; i < 50; i++ {
		if got := PickCornerTaker(xi, target, rng); got == target {
			t.Fatalf("corner taker must exclude the aerial target")
		}
	}
	if got := PickCornerTaker([]*models.Player{target}, target, rng); got != nil {
		t.Errorf("corner taker with only the target available should be nil")
	}
	if got := PickCornerTaker(nil, nil, nil); got != nil {
		t.Errorf("empty corner XI should be nil")
	}
}

func TestChunk2FreeKickTaker(t *testing.T) {
	xi := []*models.Player{
		chunk2mkPlayer("F1", "Forward", "ST", "FWD", 86),  // sh = 86 qualifies
		chunk2mkPlayer("M1", "Mid", "CM", "MID", 90),      // sh = 85 qualifies
		chunk2mkPlayer("D1", "Def", "CB", "DEF", 88),      // sh = 68 no
		chunk2mkPlayer("G1", "Keeper", "GK", "GK", 99),    // sh = 79 no
		chunk2mkPlayer("F2", "Forward2", "ST", "FWD", 88), // sh = 88 top
	}
	got := PickFreeKickTaker(xi, nil)
	if got == nil || got.PlayerID != "F2" {
		t.Errorf("top shooting should take free kicks, got %+v", got)
	}
	duds := []*models.Player{
		chunk2mkPlayer("D1", "Def", "CB", "DEF", 80),
		chunk2mkPlayer("G1", "Keeper", "GK", "GK", 80),
	}
	if got := PickFreeKickTaker(duds, nil); got != nil {
		t.Errorf("no qualifier should yield nil, got %+v", got)
	}
	// Growth-engine shooting overrides the OVR fallback.
	ge := growth.NewGrowthEngine(23)
	ge.RegisterProdigy("D1", "Def", 22, 185, 80, "DEF", 80, 88, 19)
	ge.Attributes["D1"].Shooting = 92
	if got := PickFreeKickTaker(duds, ge); got == nil || got.PlayerID != "D1" {
		t.Errorf("growth shooting 92 should qualify, got %+v", got)
	}
	ge.Attributes["D1"].Shooting = 60
	if got := PickFreeKickTaker(duds, ge); got != nil {
		t.Errorf("growth shooting 60 with weak fallback should yield nil, got %+v", got)
	}
	// Unknown IDs with a nil-attribute engine: pure OVR fallback.
	if got := PickFreeKickTaker(xi, growth.NewGrowthEngine(24)); got == nil || got.PlayerID != "F2" {
		t.Errorf("OVR fallback should still pick F2, got %+v", got)
	}
}

func TestChunk2PickBooked(t *testing.T) {
	rng := rand.New(rand.NewSource(25))
	if got := PickBooked(nil, rng); got != nil {
		t.Errorf("empty booked pool should be nil")
	}
	xi := []*models.Player{
		chunk2mkPlayer("D1", "Def", "CB", "DEF", 80),
		chunk2mkPlayer("M1", "Mid", "CM", "MID", 80),
		chunk2mkPlayer("F1", "Fwd", "ST", "FWD", 90),
		chunk2mkPlayer("G1", "Keeper", "GK", "GK", 90),
	}
	for i := 0; i < 50; i++ {
		if got := PickBooked(xi, rng); got.Category != "DEF" && got.Category != "MID" {
			t.Fatalf("bookings should prefer DEF/MID, got %+v", got)
		}
	}
	fwds := []*models.Player{chunk2mkPlayer("F1", "A", "ST", "FWD", 80)}
	if got := PickBooked(fwds, rng); got != fwds[0] {
		t.Errorf("FWD-only fallback should pick the forward")
	}
}

func TestChunk2PlanSubstitutions(t *testing.T) {
	rng := rand.New(rand.NewSource(27))
	mkXI := func() []*models.Player {
		return []*models.Player{
			chunk2mkPlayer("G1", "Keeper", "GK", "GK", 82),
			chunk2mkPlayer("D1", "D1", "CB", "DEF", 80),
			chunk2mkPlayer("D2", "D2", "CB", "DEF", 81),
			chunk2mkPlayer("M1", "M1", "CM", "MID", 83),
			chunk2mkPlayer("F1", "F1", "ST", "FWD", 84),
		}
	}
	mkBench := func() []*models.Player {
		return []*models.Player{
			chunk2mkPlayer("B1", "BDef", "CB", "DEF", 78),
			chunk2mkPlayer("B2", "BMid", "CM", "MID", 79),
			chunk2mkPlayer("B3", "BFwd", "ST", "FWD", 80),
			chunk2mkPlayer("B4", "BFwd2", "LW", "FWD", 77),
		}
	}
	if got := PlanSubstitutions(mkXI(), nil, 5, rng); got != nil {
		t.Errorf("empty bench should yield no subs")
	}
	if got := PlanSubstitutions(nil, mkBench(), 5, rng); got != nil {
		t.Errorf("empty XI should yield no subs")
	}
	// maxN below 3 is clamped.
	subs := PlanSubstitutions(mkXI(), mkBench(), 1, rng)
	if len(subs) == 0 || len(subs) > 3 {
		t.Errorf("clamped maxN should plan 1..3 subs, got %d", len(subs))
	}
	// Standard plan shape: 3-5 subs, HT sub and double-headers appear.
	var sawHT, sawDouble bool
	for seed := int64(0); seed < 120 && (!sawHT || !sawDouble); seed++ {
		r := rand.New(rand.NewSource(seed))
		for _, s := range PlanSubstitutions(mkXI(), mkBench(), 5, r) {
			if s.Minute == 45 {
				sawHT = true
			}
		}
		// Double-header detection needs minute multiset per plan.
		r2 := rand.New(rand.NewSource(seed))
		mins := map[int]int{}
		for _, s := range PlanSubstitutions(mkXI(), mkBench(), 5, r2) {
			mins[s.Minute]++
			if mins[s.Minute] > 1 {
				sawDouble = true
			}
			if s.Out.Category == "GK" || s.In.Category == "GK" {
				t.Errorf("keepers must never be subbed: %+v", s)
			}
			if s.Out.PlayerID == s.In.PlayerID {
				t.Errorf("sub out/in must differ: %+v", s)
			}
		}
	}
	if !sawHT {
		t.Errorf("expected a half-time sub across 120 seeds")
	}
	if !sawDouble {
		t.Errorf("expected a double-header across 120 seeds")
	}
	plan := PlanSubstitutions(mkXI(), mkBench(), 5, rng)
	if len(plan) < 3 || len(plan) > 5 {
		t.Errorf("plan size = %d; want 3..5", len(plan))
	}
	for i := 1; i < len(plan); i++ {
		if plan[i].Minute < plan[i-1].Minute {
			t.Errorf("plan minutes not sorted: %+v", plan)
		}
	}
	// All-keeper XI: nobody can come off.
	gkXI := []*models.Player{chunk2mkPlayer("G1", "A", "GK", "GK", 80), chunk2mkPlayer("G2", "B", "GK", "GK", 81)}
	if got := PlanSubstitutions(gkXI, mkBench(), 5, rng); len(got) != 0 {
		t.Errorf("keeper-only XI should yield no subs, got %+v", got)
	}
	// All-keeper bench: nobody can come on.
	gkBench := []*models.Player{chunk2mkPlayer("B9", "C", "GK", "GK", 80)}
	if got := PlanSubstitutions(mkXI(), gkBench, 5, rng); len(got) != 0 {
		t.Errorf("keeper-only bench should yield no subs, got %+v", got)
	}
	// No like-for-like match: any outfield player may enter.
	midBench := []*models.Player{chunk2mkPlayer("B2", "BMid", "CM", "MID", 79)}
	defXI := []*models.Player{
		chunk2mkPlayer("G1", "Keeper", "GK", "GK", 82),
		chunk2mkPlayer("D1", "D1", "CB", "DEF", 80),
	}
	got := PlanSubstitutions(defXI, midBench, 5, rand.New(rand.NewSource(77)))
	if len(got) != 1 || got[0].In.Category != "MID" {
		t.Errorf("cross-category sub should still happen, got %+v", got)
	}
	// Wonderkid flow: stayers tend to stay, prospects tend to enter.
	wkXI := mkXI()
	wkXI = append(wkXI, &models.Player{PlayerID: "WKX", FullName: "Kid", Position: "ST", Category: "FWD", OVR: 76, UniverseWonderkid: true, Age: 16})
	wkBench := append(mkBench(), &models.Player{PlayerID: "WKB", FullName: "KidB", Position: "CM", Category: "MID", OVR: 74, UniverseWonderkid: true, Age: 16})
	_ = PlanSubstitutions(wkXI, wkBench, 5, rng)
	if a, b := minInt(2, 5), minInt(5, 2); a != 2 || b != 2 {
		t.Errorf("minInt wrong: %d %d", a, b)
	}
	if a, b := maxInt(2, 5), maxInt(5, 2); a != 5 || b != 5 {
		t.Errorf("maxInt wrong: %d %d", a, b)
	}
}

func TestChunk2PassesAndAccuracy(t *testing.T) {
	rng := rand.New(rand.NewSource(29))
	for i := 0; i < 50; i++ {
		h, a := PassesSplit(60, rng)
		if h < 0 || a < 120 {
			t.Fatalf("passes split out of bounds: %d/%d", h, a)
		}
	}
	if acc := PassAccuracy(99, "clear", rng); acc != 94 {
		t.Errorf("elite accuracy should clamp at 94, got %d", acc)
	}
	if acc := PassAccuracy(30, "clear", rng); acc != 65 {
		t.Errorf("weak accuracy should clamp at 65, got %d", acc)
	}
	if acc := PassAccuracy(30, "rain", rng); acc != 65 {
		t.Errorf("rain accuracy should still clamp at 65, got %d", acc)
	}
	mid := PassAccuracy(78, "clear", rng)
	if mid < 65 || mid > 94 {
		t.Errorf("mid accuracy out of corridor: %d", mid)
	}
	// Rain penalty is exactly 5 on the same draw stream.
	r1 := rand.New(rand.NewSource(5))
	r2 := rand.New(rand.NewSource(5))
	if dry, wet := PassAccuracy(80, "clear", r1), PassAccuracy(80, "rain", r2); dry-wet != 5 {
		t.Errorf("rain penalty should be 5, got dry=%d wet=%d", dry, wet)
	}
}

func chunk2BuildClubs() (*models.Club, *models.Club) {
	home := &models.Club{ClubID: "H", ClubName: "Home FC", ShortName: "HOM", HomeStadium: "Home Park", OverallTeamRating: 84, StadiumCapacity: 60000}
	away := &models.Club{ClubID: "A", ClubName: "Away FC", ShortName: "AWY", OverallTeamRating: 82, StadiumCapacity: 40000}
	return home, away
}

func TestChunk2BuildStats(t *testing.T) {
	home, away := chunk2BuildClubs()
	sc := ToMiniPlayer(chunk2mkPlayer("S1", "Striker", "ST", "FWD", 85))
	as := ToMiniPlayer(chunk2mkPlayer("M1", "Mid", "CM", "MID", 84))
	bk := ToMiniPlayer(chunk2mkPlayer("D1", "Def", "CB", "DEF", 82))
	og := ToMiniPlayer(chunk2mkPlayer("D9", "OppDef", "CB", "DEF", 80))
	events := []MatchEventItem{
		{Minute: 10, Seq: 1, Type: "goal", Side: "home", Scorer: &sc, Assister: &as},
		{Minute: 20, Seq: 2, Type: "goal", Side: "away", Scorer: &sc, Display: "x"},
		{Minute: 25, Seq: 3, Type: "goal", Side: "home", Scorer: &sc, Disallowed: true},
		{Minute: 30, Seq: 4, Type: "own_goal", Side: "away", Beneficiary: "home", Scorer: &og},
		{Minute: 40, Seq: 5, Type: "yellow", Side: "home", Player: &bk},
		{Minute: 50, Seq: 6, Type: "yellow", Side: "home", Player: &bk},
		{Minute: 55, Seq: 7, Type: "red", Side: "home", Player: &bk, SentOff: true},
		{Minute: 60, Seq: 8, Type: "red", Side: "away", Player: &bk},
	}
	rng := rand.New(rand.NewSource(31))
	stats := BuildStats(home, away, 12, 9, 5, 3, 58, 6, 4, events, "rain", rng)
	if stats.Home.Possession != 58 || stats.Away.Possession != 42 {
		t.Errorf("possession split wrong: %+v", stats)
	}
	if stats.Home.YellowCards != 0 {
		t.Errorf("sent-off player's yellows must be excluded, got %d", stats.Home.YellowCards)
	}
	if stats.Home.RedCards != 1 || stats.Away.RedCards != 1 {
		t.Errorf("red counts wrong: %+v", stats)
	}
	if stats.Home.Corners != 6 || stats.Away.Corners != 4 {
		t.Errorf("corners not passed through: %+v", stats)
	}
	if stats.Home.XG < 0.05 || stats.Home.XG > 4.8 || stats.Away.XG < 0.05 || stats.Away.XG > 4.8 {
		t.Errorf("xG out of corridor: %+v", stats)
	}
	if stats.Home.PassAccuracy < 65 || stats.Home.PassAccuracy > 94 {
		t.Errorf("accuracy out of corridor: %+v", stats)
	}
	if stats.Home.Fouls < 0 || stats.Away.Fouls < 0 {
		t.Errorf("fouls negative: %+v", stats)
	}
	if stats.Home.Saves != 1 { // awayOn(3) - awayGoals(1 allowed + ... )
		t.Logf("home saves = %d", stats.Home.Saves)
	}
	// Dry-weather variant for the foul-base branch.
	statsDry := BuildStats(home, away, 12, 9, 5, 3, 58, 6, 4, nil, "clear", rng)
	if statsDry.Home.Shots != 12 || statsDry.Away.Shots != 9 {
		t.Errorf("shot totals not passed through: %+v", statsDry)
	}
}

func TestChunk2AssembleReport(t *testing.T) {
	_, _ = chunk2BuildClubs()
	sc := ToMiniPlayer(chunk2mkPlayer("S1", "Striker", "ST", "FWD", 85))
	benchScorer := ToMiniPlayer(chunk2mkPlayer("B7", "SuperSub", "ST", "FWD", 80))
	events := []MatchEventItem{
		{Minute: 70, Seq: 2, Type: "goal", Side: "home", Scorer: &benchScorer},
		{Minute: 20, Seq: 1, Type: "goal", Side: "home", Scorer: &sc},
	}
	xi := []*models.Player{chunk2mkPlayer("S1", "Striker", "ST", "FWD", 85)}
	bench := []*models.Player{chunk2mkPlayer("B7", "SuperSub", "ST", "FWD", 80)}
	payload := InstantPayload{
		HomeGoals: 2, AwayGoals: 0, Events: events,
		HomeXI: xi, AwayXI: nil, HomeBench: bench, AwayBench: nil,
		HTHome: 1, HTAway: 0, Attendance: 50000, Referee: "Michael Oliver", Weather: "rain",
	}
	rng := rand.New(rand.NewSource(33))
	report := AssembleReport(payload, "instant", nil)
	if report.Method != "instant" {
		t.Errorf("method = %q", report.Method)
	}
	if len(report.Events) != 2 || report.Events[0].Minute != 20 {
		t.Errorf("events should be chronologically sorted: %+v", report.Events)
	}
	if report.HTHome != 1 || report.Attendance != 50000 || report.Referee != "Michael Oliver" || report.Weather != "rain" {
		t.Errorf("payload fields not carried: %+v", report)
	}
	if report.MOTM == nil || !report.MOTM.Played {
		t.Errorf("MOTM should be a played row: %+v", report.MOTM)
	}
	if report.MOTM.Side != "home" {
		t.Errorf("MOTM side = %q", report.MOTM.Side)
	}
	// Empty payload: MOTM is nil but assembly still succeeds.
	empty := AssembleReport(InstantPayload{}, "live", rng)
	if empty.MOTM != nil {
		t.Errorf("empty payload MOTM should be nil")
	}
	if empty.Method != "live" {
		t.Errorf("empty method = %q", empty.Method)
	}
	// Nil-RNG assembly path.
	_ = AssembleReport(payload, "instant", nil)
}

func TestChunk2ShotMapConstants(t *testing.T) {
	home, away := chunk2BuildClubs()
	sc := ToMiniPlayer(chunk2mkPlayer("S1", "Striker", "ST", "FWD", 85))
	pen := MatchEventItem{Minute: 30, Type: "penalty", Side: "away", Scorer: &sc}
	reg := MatchEventItem{Minute: 60, Type: "goal", Side: "home", Scorer: &sc}
	anon := MatchEventItem{Minute: 70, Type: "goal", Side: "home"}
	rng := rand.New(rand.NewSource(35))
	sm := GenerateShotMap(home, away, []MatchEventItem{pen, reg, anon}, 8, 6, 3, 2, rng, nil)
	byMinute := map[int]ShotMapItem{}
	for _, s := range sm.Shots {
		byMinute[s.Minute] = s
	}
	if byMinute[30].XG != 0.76 {
		t.Errorf("penalty xG = %v; want 0.76", byMinute[30].XG)
	}
	if byMinute[30].X != 0.07 || byMinute[30].Y < 0.44 || byMinute[30].Y > 0.56 {
		t.Errorf("away penalty coords = (%v,%v); want (0.07,[0.44,0.56])", byMinute[30].X, byMinute[30].Y)
	}
	if byMinute[60].XG < 0.32 || byMinute[60].XG > 0.62 {
		t.Errorf("regular goal xG = %v; want [0.32,0.62]", byMinute[60].XG)
	}
	if byMinute[60].X != 0.93 {
		t.Errorf("home goal x = %v; want 0.93", byMinute[60].X)
	}
	if byMinute[60].Y < 0.44 || byMinute[60].Y > 0.56 {
		t.Errorf("home goal y = %v; want [0.44,0.56]", byMinute[60].Y)
	}
	if byMinute[70].Shooter.FullName != "Striker" || byMinute[70].Shooter.OVR != 80 {
		t.Errorf("anonymous shooter fallback wrong: %+v", byMinute[70].Shooter)
	}
	// Reconstructed placeholders carry the club short name.
	foundPlaceholder := false
	for _, s := range sm.Shots {
		if s.Outcome != "goal" && (s.Shooter.FullName == "HOM Player" || s.Shooter.FullName == "AWY Player") {
			foundPlaceholder = true
		}
	}
	if !foundPlaceholder {
		t.Errorf("expected {SHORT} Player placeholders among reconstructed shots")
	}
	// Flow tail always reaches at least 90'.
	tail := sm.XGFlow[len(sm.XGFlow)-1]
	if tail.Minute < 90 {
		t.Errorf("flow tail = %d; want >= 90", tail.Minute)
	}
}

func TestChunk2HeatmapBeta(t *testing.T) {
	home, away := chunk2BuildClubs()
	rng := rand.New(rand.NewSource(37))
	hm := GenerateTouchHeatmap(home, away, 60, rng, nil)
	if len(hm.HomePoints) != 72 || len(hm.AwayPoints) != 48 {
		t.Errorf("possession split points = %d/%d; want 72/48", len(hm.HomePoints), len(hm.AwayPoints))
	}
	var sum float64
	for _, p := range hm.HomePoints {
		sum += p[0]
		if p[1] < 0.08 || p[1] > 0.92 {
			t.Errorf("y out of clamp: %v", p)
		}
	}
	if mean := sum / float64(len(hm.HomePoints)); mean < 0.50 || mean > 0.62 {
		t.Errorf("beta-skewed home x-mean = %v; want attack tilt ~0.56", mean)
	}
	if hm.HomeZones.Defensive+hm.HomeZones.Midfield+hm.HomeZones.Attacking < 95 {
		t.Errorf("zones should sum ~100: %+v", hm.HomeZones)
	}
	// Nil RNG path.
	hm2 := GenerateTouchHeatmap(home, away, 50, nil, nil)
	if len(hm2.HomePoints)+len(hm2.AwayPoints) != 120 {
		t.Errorf("nil-rng heatmap should still scatter 120 points")
	}
}

func TestChunk2PressConferenceExact(t *testing.T) {
	home, away := chunk2BuildClubs()
	rng := rand.New(rand.NewSource(39))
	_ = rng
	draw := GeneratePressConference(home, away, 1, 1, nil, "", "")
	if draw.Headline != "HOM and AWY share spoils in tense tactical clash" {
		t.Errorf("draw headline wrong: %q", draw.Headline)
	}
	if draw.HomeQuote != "Both teams showed immense tactical discipline. A fair point, though we felt we created enough to nick all three." {
		t.Errorf("draw home quote wrong: %q", draw.HomeQuote)
	}
	wantAway := "Coming away from Home Park with a point is a solid foundation. The team showed fighting character until the final whistle."
	if draw.AwayQuote != wantAway {
		t.Errorf("draw away quote wrong: %q", draw.AwayQuote)
	}
	if draw.HomeManager != "HOM Manager" || draw.AwayManager != "AWY Manager" {
		t.Errorf("manager fallbacks wrong: %+v", draw)
	}
	// Wonderkid scorer appends the prodigy line.
	wk := ToMiniPlayer(&models.Player{PlayerID: "WK1", FullName: "Kid Wonder", Position: "ST", Category: "FWD", OVR: 78, UniverseWonderkid: true, Age: 16})
	win := GeneratePressConference(home, away, 2, 0,
		[]MatchEventItem{{Minute: 10, Type: "goal", Side: "home", Scorer: &wk}}, "Gaffer", "Boss")
	if !strings.Contains(win.HomeQuote, "Kid Wonder showed maturity beyond his years") {
		t.Errorf("WK home line missing: %q", win.HomeQuote)
	}
	awayWin := GeneratePressConference(home, away, 0, 1,
		[]MatchEventItem{{Minute: 10, Type: "goal", Side: "away", Scorer: &wk}}, "Gaffer", "Boss")
	if !strings.Contains(awayWin.AwayQuote, "Kid Wonder's composure on the ball was the turning point tonight.") {
		t.Errorf("WK away line missing: %q", awayWin.AwayQuote)
	}
	if awayWin.HomeManager != "Gaffer" || awayWin.AwayManager != "Boss" {
		t.Errorf("named managers not carried: %+v", awayWin)
	}
}

func TestChunk2R2MiniNil(t *testing.T) {
	if got := ToMiniPlayer(nil); got != (MiniPlayer{}) {
		t.Errorf("nil mini player should be zero struct: %+v", got)
	}
}

func TestChunk2R2AppearanceWindow(t *testing.T) {
	inMini := MiniPlayer{PlayerID: "IN1"}
	outMini := MiniPlayer{PlayerID: "OUT1"}
	redMini := MiniPlayer{PlayerID: "RED1"}
	events := []MatchEventItem{
		{Minute: 60, Seq: 1, Type: "sub", Side: "home", PlayerIn: &inMini, PlayerOut: &outMini},
		{Minute: 70, Seq: 2, Type: "sub", Side: "home", PlayerOut: &outMini},
		{Minute: 75, Seq: 3, Type: "red", Side: "away", Player: &redMini},
	}
	// Subbed-in non-starter.
	if mins, on, off := AppearanceWindow("IN1", events, false, false); mins != 30 || on == nil || *on != 60 || off != nil {
		t.Errorf("sub-in window wrong: %d %v %v", mins, on, off)
	}
	// Subbed-out starter takes the earliest off-minute.
	if mins, on, off := AppearanceWindow("OUT1", events, true, false); mins != 60 || on != nil || off == nil || *off != 60 {
		t.Errorf("sub-out window wrong: %d %v %v", mins, on, off)
	}
	// Dismissed starter.
	if mins, _, off := AppearanceWindow("RED1", events, true, false); mins != 75 || off == nil || *off != 75 {
		t.Errorf("red window wrong: %d %v", mins, off)
	}
	// Unused bench player never played.
	if mins, on, off := AppearanceWindow("GHOST", events, false, false); mins != 0 || on != nil || off != nil {
		t.Errorf("ghost window wrong: %d %v %v", mins, on, off)
	}
	// Full-match starter and extra-time starter.
	if mins, on, off := AppearanceWindow("FULL", nil, true, false); mins != 90 || on != nil || off != nil {
		t.Errorf("full window wrong: %d %v %v", mins, on, off)
	}
	if mins, _, _ := AppearanceWindow("FULL", nil, true, true); mins != 120 {
		t.Errorf("ET window = %d; want 120", mins)
	}
	// Sub-on and sub-off in the same minute floors at 1.
	flash := []MatchEventItem{
		{Minute: 89, Seq: 1, Type: "sub", Side: "home", PlayerIn: &inMini},
		{Minute: 89, Seq: 2, Type: "sub", Side: "home", PlayerOut: &inMini},
	}
	if mins, _, _ := AppearanceWindow("IN1", flash, false, false); mins != 1 {
		t.Errorf("flash window = %d; want 1", mins)
	}
}

func TestChunk2R2RateXIFull(t *testing.T) {
	sc := ToMiniPlayer(chunk2mkPlayer("S1", "Striker", "ST", "FWD", 85))
	as := ToMiniPlayer(chunk2mkPlayer("M1", "Mid", "CM", "MID", 84))
	og := ToMiniPlayer(chunk2mkPlayer("D9", "Opp", "CB", "DEF", 80))
	bk := ToMiniPlayer(chunk2mkPlayer("D1", "Def", "CB", "DEF", 82))
	events := []MatchEventItem{
		{Minute: 10, Seq: 1, Type: "goal", Side: "home", Scorer: &sc, Assister: &as},
		{Minute: 20, Seq: 2, Type: "goal", Side: "home", Scorer: &sc},
		{Minute: 25, Seq: 3, Type: "goal", Side: "home"}, // no scorer/assister recorded
		{Minute: 30, Seq: 4, Type: "goal", Side: "home", Scorer: &sc, Disallowed: true},
		{Minute: 35, Seq: 5, Type: "corner_goal", Side: "home", Scorer: &as},
		{Minute: 40, Seq: 6, Type: "penalty", Side: "home", Scorer: &sc},
		{Minute: 44, Seq: 7, Type: "free_kick_goal", Side: "home", Scorer: &sc},
		{Minute: 50, Seq: 8, Type: "own_goal", Side: "home", Scorer: &og},
		{Minute: 60, Seq: 9, Type: "penalty_miss", Side: "home", Scorer: &sc},
		{Minute: 62, Seq: 10, Type: "penalty_miss", Side: "home", Scorer: &MiniPlayer{PlayerID: "G1"}},
		{Minute: 65, Seq: 11, Type: "yellow", Side: "home", Player: &bk},
		{Minute: 80, Seq: 12, Type: "red", Side: "home", Player: &bk},
		{Minute: 95, Seq: 13, Type: "goal", Side: "away", Scorer: &og},
		{Minute: 70, Seq: 14, Type: "sub", Side: "home", PlayerIn: &MiniPlayer{PlayerID: "SUB1"}, PlayerOut: &MiniPlayer{PlayerID: "S1"}},
	}
	xi := []*models.Player{
		chunk2mkPlayer("S1", "Striker", "ST", "FWD", 85),
		chunk2mkPlayer("M1", "Mid", "CM", "MID", 84),
		chunk2mkPlayer("D1", "Def", "CB", "DEF", 82),
		chunk2mkPlayer("G1", "Keeper", "GK", "GK", 86),
	}
	bench := []*models.Player{chunk2mkPlayer("SUB1", "Sub", "ST", "FWD", 78)}
	rng := rand.New(rand.NewSource(41))
	rows := RateXI(xi, events, "home", 1, true, true, rng)
	if len(rows) != 4 {
		t.Fatalf("rows = %d; want 4", len(rows))
	}
	// Two counted goals + penalty + FK = 4 (disallowed excluded).
	if rows[0].MatchGoals != 4 {
		t.Errorf("striker goals = %d; want 4", rows[0].MatchGoals)
	}
	if rows[0].MatchPenMiss != 1 {
		t.Errorf("striker pen miss = %d; want 1", rows[0].MatchPenMiss)
	}
	// Red overwrites the earlier yellow.
	if rows[2].Card == nil || *rows[2].Card != "red" {
		t.Errorf("defender card = %v; want red", rows[2].Card)
	}
	// Subbed-out starter and extra-time flag via the 95' away goal.
	if rows[0].Minutes != 70 {
		t.Errorf("striker minutes = %d; want 70", rows[0].Minutes)
	}
	// Bench: super-sub played (<20 branch needs a later entrance).
	late := []MatchEventItem{
		{Minute: 78, Seq: 1, Type: "sub", Side: "home", PlayerIn: &MiniPlayer{PlayerID: "SUB1"}},
		{Minute: 95, Seq: 2, Type: "goal", Side: "away", Scorer: &og},
	}
	brows := RateXI(bench, late, "home", 1, true, false, nil)
	if len(brows) != 1 || !brows[0].Played || brows[0].Minutes != 42 {
		t.Errorf("late sub rows wrong (ET extends to 120'): %+v", brows)
	}
	if brows[0].Rating == nil || *brows[0].Rating < 5.0 || *brows[0].Rating > 10.0 {
		t.Errorf("late sub rating out of corridor: %+v", brows[0].Rating)
	}
	// Regulation-time late sub: sub-20 rating band.
	noET := []MatchEventItem{
		{Minute: 78, Seq: 1, Type: "sub", Side: "home", PlayerIn: &MiniPlayer{PlayerID: "SUB1"}},
		{Minute: 80, Seq: 2, Type: "yellow", Side: "home", Player: &MiniPlayer{PlayerID: "SUB1"}},
		{Minute: 79, Seq: 3, Type: "sub", Side: "home", PlayerIn: &MiniPlayer{PlayerID: "SUB2"}},
		{Minute: 85, Seq: 4, Type: "red", Side: "home", Player: &MiniPlayer{PlayerID: "SUB2"}},
	}
	twoSubs := append(bench, chunk2mkPlayer("SUB2", "Sub2", "CM", "MID", 77))
	s20 := RateXI(twoSubs, noET, "home", 0, false, false, rng)
	if len(s20) != 2 || !s20[0].Played || s20[0].Minutes != 12 {
		t.Errorf("sub-20 rows wrong: %+v", s20)
	}
	if s20[0].Card == nil || *s20[0].Card != "yellow" {
		t.Errorf("late-sub yellow not tallied: %+v", s20[0])
	}
	if len(s20) != 2 || !s20[1].Played || s20[1].Minutes != 6 {
		t.Errorf("sent-off sub rows wrong: %+v", s20)
	}
	if s20[1].Card == nil || *s20[1].Card != "red" {
		t.Errorf("late-sub red not tallied: %+v", s20[1])
	}
	// Unused bench: nil-rating rows.
	cold := RateXI(bench, nil, "home", 0, false, false, rng)
	if len(cold) != 1 || cold[0].Played || cold[0].Rating != nil || cold[0].Card != nil {
		t.Errorf("cold bench rows wrong: %+v", cold)
	}
	// Away-side view counts only away events.
	arows := RateXI([]*models.Player{chunk2mkPlayer("D9", "Opp", "CB", "DEF", 80)}, events, "away", 4, false, true, rng)
	if len(arows) != 1 || arows[0].MatchGoals != 1 {
		t.Errorf("away tally wrong: %+v", arows)
	}
}

func TestChunk2R2ShotMapEdges(t *testing.T) {
	home, away := chunk2BuildClubs()
	sc := ToMiniPlayer(chunk2mkPlayer("S1", "Striker", "ST", "FWD", 85))
	et := MatchEventItem{Minute: 95, Type: "goal", Side: "home", Scorer: &sc}
	etAway := MatchEventItem{Minute: 88, Type: "goal", Side: "away", Scorer: &sc}
	// Nil RNG plus totals below the goal counts (remain caps) plus ET pad.
	sm := GenerateShotMap(home, away, []MatchEventItem{et, etAway}, 0, 0, 0, 0, nil, nil)
	if len(sm.Shots) != 2 {
		t.Fatalf("shots = %d; want 2 (caps at zero remain)", len(sm.Shots))
	}
	if tail := sm.XGFlow[len(sm.XGFlow)-1]; tail.Minute != 95 {
		t.Errorf("ET tail = %d; want 95", tail.Minute)
	}
}

func TestChunk2R2OwnGoalCulprit(t *testing.T) {
	rng := rand.New(rand.NewSource(43))
	defs := []*models.Player{
		chunk2mkPlayer("D1", "A", "CB", "DEF", 80),
		chunk2mkPlayer("M1", "B", "CM", "MID", 80),
		chunk2mkPlayer("G1", "C", "GK", "GK", 80),
	}
	// DEF/GK pool only: the midfielder can never be the culprit.
	for i := 0; i < 50; i++ {
		if got := PickOwnGoalCulprit(defs, defs, rng); got.Category == "MID" {
			t.Fatalf("culprit must come from DEF/GK, got %+v", got)
		}
	}
	// No DEF/GK on the field: anyone available.
	mids := []*models.Player{chunk2mkPlayer("M1", "B", "CM", "MID", 80)}
	if got := PickOwnGoalCulprit(mids, mids, rng); got != mids[0] {
		t.Errorf("fallback should pick the midfielder, got %+v", got)
	}
	// Empty field: explicit fallback list.
	if got := PickOwnGoalCulprit(nil, mids, rng); got != mids[0] {
		t.Errorf("empty field should use fallback, got %+v", got)
	}
	// Nothing anywhere: nil.
	if got := PickOwnGoalCulprit(nil, nil, rng); got != nil {
		t.Errorf("empty everything should be nil, got %+v", got)
	}
	if got := PickOwnGoalCulprit(nil, nil, nil); got != nil {
		t.Errorf("nil-rng empty should be nil, got %+v", got)
	}
}

func TestChunk2R2WeightedNaN(t *testing.T) {
	rng := rand.New(rand.NewSource(45))
	if got := weightedIndex(rng, []float64{math.NaN()}); got != 0 {
		t.Errorf("NaN weights should fall back to the last index, got %d", got)
	}
}

func TestChunk2R2PassesClamp(t *testing.T) {
	rng := rand.New(rand.NewSource(47))
	_, a := PassesSplit(100, rng)
	if a != 120 {
		t.Errorf("full-possession away passes = %d; want floor 120", a)
	}
}

func TestChunk2R2BuildStatsEdges(t *testing.T) {
	home, away := chunk2BuildClubs()
	bk := ToMiniPlayer(chunk2mkPlayer("D1", "Def", "CB", "DEF", 82))
	events := []MatchEventItem{
		{Minute: 40, Seq: 1, Type: "yellow", Side: "home", Player: &bk},
	}
	rng := rand.New(rand.NewSource(49))
	stats := BuildStats(home, away, 10, 8, 4, 3, 55, 5, 4, events, "clear", rng)
	if stats.Home.YellowCards != 1 {
		t.Errorf("non-dismissed yellow should count, got %d", stats.Home.YellowCards)
	}
	// Inconsistent on-target above shots exercises the xG off-target floor.
	stats2 := BuildStats(home, away, 2, 8, 5, 3, 55, 5, 4, nil, "clear", rng)
	if stats2.Home.XG < 0.05 || stats2.Home.XG > 4.8 {
		t.Errorf("floored xG out of corridor: %v", stats2.Home.XG)
	}
}

func TestChunk2R2AssembleSameMinute(t *testing.T) {
	_, _ = chunk2BuildClubs()
	sc := ToMiniPlayer(chunk2mkPlayer("S1", "Striker", "ST", "FWD", 85))
	events := []MatchEventItem{
		{Minute: 70, Seq: 2, Type: "goal", Side: "home", Scorer: &sc},
		{Minute: 70, Seq: 1, Type: "yellow", Side: "home", Player: &sc},
		{Minute: 60, Seq: 3, Type: "sub", Side: "home",
			PlayerIn: &MiniPlayer{PlayerID: "B7"}, PlayerOut: &MiniPlayer{PlayerID: "S1"}},
	}
	xi := []*models.Player{chunk2mkPlayer("S1", "Striker", "ST", "FWD", 85)}
	bench := []*models.Player{chunk2mkPlayer("B7", "SuperSub", "ST", "FWD", 80)}
	payload := InstantPayload{
		HomeGoals: 1, AwayGoals: 0, Events: events,
		HomeXI: xi, HomeBench: bench, HTHome: 0,
		Attendance: 30000, Referee: "Ivan Barton", Weather: "clear",
	}
	report := AssembleReport(payload, "instant", rand.New(rand.NewSource(51)))
	if len(report.Events) != 3 || report.Events[1].Seq != 1 || report.Events[2].Seq != 2 {
		t.Errorf("same-minute events should order by seq: %+v", report.Events)
	}
	if report.MOTM == nil {
		t.Fatalf("MOTM should resolve with a played super-sub")
	}
}
