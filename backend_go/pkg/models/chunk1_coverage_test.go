package models

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"testing"
)

// Supplemental Chunk 1 coverage: Club domain (UnmarshalJSON clamps, morale
// caps/floors, XI edge paths, bench defaults, RecalculateRatings).

func TestChunk1CovClubUnmarshalMoraleClamps(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  string
		want int
	}{
		{"negative clamps to 0", `{"club_id":"T1","morale":-5}`, 0},
		{"over 100 clamps to 100", `{"club_id":"T2","morale":150}`, 100},
		{"explicit preserved", `{"club_id":"T3","morale":55,"form":["W"]}`, 55},
	} {
		var c Club
		if err := json.Unmarshal([]byte(tc.raw), &c); err != nil {
			t.Fatalf("%s: unmarshal failed: %v", tc.name, err)
		}
		if c.Morale != tc.want {
			t.Errorf("%s: morale = %d; want %d", tc.name, c.Morale, tc.want)
		}
	}
	var c Club
	if err := json.Unmarshal([]byte(`{"club_id":"T3","morale":55,"form":["W"]}`), &c); err != nil {
		t.Fatal(err)
	}
	if len(c.Form) != 1 || c.Form[0] != "W" {
		t.Errorf("explicit form should be preserved, got %v", c.Form)
	}
}

func TestChunk1CovClubUnmarshalSquadLinking(t *testing.T) {
	raw := `{"club_id":"LINK","short_name":"ZZQ","squad":[{"player_id":"P1","full_name":"Link Test","position":"ST","ovr":70}]}`
	var c Club
	if err := json.Unmarshal([]byte(raw), &c); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(c.Squad) != 1 {
		t.Fatalf("squad len = %d; want 1", len(c.Squad))
	}
	if c.Squad[0].ClubID != "LINK" {
		t.Errorf("ClubID not linked: %q", c.Squad[0].ClubID)
	}
	if c.Squad[0].OriginalClubID != "LINK" {
		t.Errorf("OriginalClubID not linked: %q", c.Squad[0].OriginalClubID)
	}
	if c.Squad[0].Category != "FWD" {
		t.Errorf("Category not derived: %q", c.Squad[0].Category)
	}
	// Procedural kit colors for unknown short name.
	if c.PrimaryColor == [3]uint8{0, 0, 0} {
		t.Errorf("expected procedural kit colors, got zero value")
	}
}

func TestChunk1CovUpdateMoraleCapsFloors(t *testing.T) {
	c := &Club{Morale: 99}
	c.UpdateMorale("W") // 99+3 -> capped 100
	if c.Morale != 100 {
		t.Errorf("W cap: morale = %d; want 100", c.Morale)
	}
	c.UpdateMorale("W") // already 100, stays (form empty so no streak)
	if c.Morale != 100 {
		t.Errorf("W cap hold: morale = %d; want 100", c.Morale)
	}

	d := &Club{Morale: 30}
	d.UpdateMorale("D") // 29 -> floor 30
	if d.Morale != 30 {
		t.Errorf("D floor: morale = %d; want 30", d.Morale)
	}
	d2 := &Club{Morale: 50}
	d2.UpdateMorale("D")
	if d2.Morale != 49 {
		t.Errorf("D normal: morale = %d; want 49", d2.Morale)
	}

	l := &Club{Morale: 22}
	l.UpdateMorale("L") // 19 -> floor 20
	if l.Morale != 20 {
		t.Errorf("L floor: morale = %d; want 20", l.Morale)
	}

	u := &Club{Morale: 70}
	u.UpdateMorale("X") // unknown result: no-op
	if u.Morale != 70 {
		t.Errorf("unknown result should be no-op, morale = %d", u.Morale)
	}
}

func TestChunk1CovParseFixtureArgs(t *testing.T) {
	c := &Club{ClubID: "PFA"}
	p := &Player{PlayerID: "P1", Category: "FWD", OVR: 70}
	c.Squad = []*Player{p}

	if got := c.AvailableSquad(); len(got) != 1 {
		t.Errorf("no args: got %d players; want 1", len(got))
	}
	if got := c.AvailableSquad(""); len(got) != 1 {
		t.Errorf("empty arg: got %d players; want 1", len(got))
	}
	if got := c.AvailableSquad("ucl"); len(got) != 1 {
		t.Errorf("comp only: got %d players; want 1", len(got))
	}
	if got := c.AvailableSquad("ucl:abc"); len(got) != 1 {
		t.Errorf("bad week: got %d players; want 1", len(got))
	}
	if got := c.AvailableSquad("ucl:5"); len(got) != 1 {
		t.Errorf("good week: got %d players; want 1", len(got))
	}
	// GetStartingEleven fixture passthrough (single-player squad yields a
	// duplicate-free XI of 1).
	if xi := c.GetStartingEleven("super-league:5"); len(xi) != 1 {
		t.Errorf("XI passthrough: got %d; want 1", len(xi))
	}
}

func TestChunk1CovStartingElevenNoGKAndFill(t *testing.T) {
	// No GK in squad: falls back to pool[0]; unknown category -> FWD bucket;
	// short squad exercises the fill-to-11 path.
	c := &Club{ClubID: "NOGK"}
	c.Squad = []*Player{
		{PlayerID: "D1", Category: "DEF", OVR: 75},
		{PlayerID: "D2", Category: "DEF", OVR: 74},
		{PlayerID: "M1", Category: "MID", OVR: 76},
		{PlayerID: "X1", Category: "XXX", OVR: 80},
	}
	xi := c.GetStartingEleven()
	if len(xi) != 4 {
		t.Errorf("short squad XI = %d; want 4", len(xi))
	}
	if xi[0].PlayerID != "D1" {
		t.Errorf("no-GK fallback should start pool[0], got %s", xi[0].PlayerID)
	}

	// All-FWD squad of 12: take cap at 3, fill path tops up toward 11.
	c2 := &Club{ClubID: "ALLFWD"}
	for i := 0; i < 12; i++ {
		c2.Squad = append(c2.Squad, &Player{
			PlayerID: string(rune('A'+i)) + "_FWD",
			Category: "FWD",
			OVR:      60 + i,
		})
	}
	xi2 := c2.GetStartingEleven()
	if len(xi2) != 11 {
		t.Errorf("filled XI = %d; want 11", len(xi2))
	}
}

func TestChunk1CovGetBenchDefaults(t *testing.T) {
	c := &Club{ClubID: "BENCH2"}
	for i := 0; i < 14; i++ {
		c.Squad = append(c.Squad, &Player{
			PlayerID: fmt.Sprintf("M_%02d", i),
			Category: "MID",
			OVR:      60 + i,
		})
	}
	starters := c.GetStartingEleven()
	// n <= 0 defaults to 7; pool smaller than n returns whole pool.
	bench := c.GetBench(starters, 0)
	if len(bench) != len(c.Squad)-len(starters) {
		t.Errorf("bench default n: got %d; want %d", len(bench), len(c.Squad)-len(starters))
	}
	for _, b := range bench {
		for _, s := range starters {
			if b.PlayerID == s.PlayerID {
				t.Errorf("overlap: %s in starters and bench", b.PlayerID)
			}
		}
	}
	// Wonderkid-first bench ordering (tired wonderkid misses the XI via
	// lost priority, then heads the bench despite the lowest OVR).
	c.Squad = append(c.Squad, &Player{PlayerID: "wk_b", Category: "MID", OVR: 60, UniverseWonderkid: true, ConsecutiveStarts: 5})
	bench2 := c.GetBench(c.GetStartingEleven(), 7)
	if len(bench2) == 0 || bench2[0].PlayerID != "wk_b" {
		t.Errorf("wonderkid should head the bench, got %+v", bench2)
	}
}

func TestChunk1CovRecalculateRatings(t *testing.T) {
	empty := &Club{ClubID: "EMPTY", OverallTeamRating: 80}
	empty.RecalculateRatings()
	if empty.SquadSize != 0 {
		t.Errorf("empty squad size = %d; want 0", empty.SquadSize)
	}

	c := &Club{ClubID: "RAT"}
	ovrs := []int{80, 82, 78, 84, 76, 81, 79, 83, 77, 85, 75, 74}
	cats := []string{"GK", "DEF", "DEF", "DEF", "DEF", "MID", "MID", "MID", "FWD", "FWD", "FWD", "FWD"}
	for i := range ovrs {
		c.Squad = append(c.Squad, &Player{PlayerID: string(rune('A' + i)), Category: cats[i], OVR: ovrs[i]})
	}
	c.RecalculateRatings()
	if c.SquadSize != 12 {
		t.Errorf("SquadSize = %d; want 12", c.SquadSize)
	}
	if c.SquadAvgOVR <= 0 {
		t.Errorf("SquadAvgOVR not computed: %v", c.SquadAvgOVR)
	}
	if c.OverallTeamRating <= 0 {
		t.Errorf("OverallTeamRating not computed: %d", c.OverallTeamRating)
	}
	// ToStandingsRow form copy must be independent.
	c.Form = []string{"W"}
	row := c.ToStandingsRow(2)
	row.Form[0] = "L"
	if c.Form[0] != "W" {
		t.Errorf("ToStandingsRow must deep-copy form")
	}
}

func TestChunk1CovSortKeyFatigueAndSortClubs(t *testing.T) {
	tired := &Player{PlayerID: "T", OVR: 85, ConsecutiveStarts: 6}
	if _, adj := sortKey(tired); adj != 85-12 {
		t.Errorf("fatigue drop wrong: %d", adj)
	}
	fresh := &Player{PlayerID: "F", OVR: 80}
	if _, adj := sortKey(fresh); adj != 80 {
		t.Errorf("fresh player should have no drop: %d", adj)
	}
	// sortPlayersForXI tiebreak on PlayerID.
	a := &Player{PlayerID: "A", Category: "MID", OVR: 75}
	b := &Player{PlayerID: "B", Category: "MID", OVR: 75}
	pool := []*Player{b, a}
	sortPlayersForXI(pool)
	if pool[0].PlayerID != "A" {
		t.Errorf("ID tiebreak failed: %v", pool)
	}

	// SortClubs: exercise GD, GF, rating, and name tiebreaks.
	mk := func(id, name string, pts, gd, gf, rating int) *Club {
		return &Club{ClubID: id, ClubName: name, Points: pts, GoalDifference: gd, GoalsFor: gf, OverallTeamRating: rating}
	}
	clubs := []*Club{
		mk("C1", "Zulu", 10, 5, 12, 80),
		mk("C2", "Alpha", 10, 5, 12, 80),
		mk("C3", "Mike", 10, 5, 15, 80),
		mk("C4", "Oscar", 10, 8, 9, 80),
		mk("C5", "Papa", 10, 8, 9, 85),
		mk("C6", "Quebec", 12, 0, 5, 70),
	}
	SortClubs(clubs)
	wantOrder := []string{"C6", "C5", "C4", "C3", "C2", "C1"}
	for i, want := range wantOrder {
		if clubs[i].ClubID != want {
			t.Errorf("pos %d = %s; want %s (full order %v)", i, clubs[i].ClubID, want, clubs)
			break
		}
	}
}

func TestChunk1CovHSVToRGBSectors(t *testing.T) {
	// One hue per sector plus gray (s=0).
	for _, h := range []float64{30, 90, 150, 210, 270, 330} {
		rgb := HSVToRGB(h, 0.85, 0.90)
		if rgb == [3]uint8{0, 0, 0} {
			t.Errorf("hue %v produced black: %v", h, rgb)
		}
	}
	gray := HSVToRGB(0, 0, 0.5)
	if gray[0] != gray[1] || gray[1] != gray[2] {
		t.Errorf("gray should have equal channels: %v", gray)
	}
	// Clamp paths: over-range and under-range.
	hi := HSVToRGB(0, 0, 2)
	if hi != [3]uint8{255, 255, 255} {
		t.Errorf("over-range should clamp to white: %v", hi)
	}
	lo := HSVToRGB(0, 0, -1)
	if lo != [3]uint8{0, 0, 0} {
		t.Errorf("under-range should clamp to black: %v", lo)
	}
	// NaN hue exercises the default switch branch without panicking.
	_ = HSVToRGB(math.NaN(), 0.5, 0.5)
	// Negative hue wraps via Mod.
	_ = HSVToRGB(-30, 0.85, 0.90)
	_, _ = KitColorsForClub("  ars ")
	if p, s := KitColorsForClub("ZZQ"); p == [3]uint8{0, 0, 0} || s != [3]uint8{255, 255, 255} {
		t.Errorf("procedural colors wrong: %v / %v", p, s)
	}
}

func TestChunk1CovEducationForAgeAndLabels(t *testing.T) {
	plain := &Player{FullName: "Plain Joe", Age: 14}
	if got := plain.EducationForAge(); got != "none" {
		t.Errorf("non-wonderkid education = %q; want none", got)
	}
	for _, tc := range []struct {
		age  int
		want string
	}{
		{14, "middle_school"}, {15, "middle_school"},
		{16, "high_school"}, {17, "high_school"},
		{18, "graduated"}, {25, "graduated"},
	} {
		wk := &Player{FullName: "Kid", Age: tc.age, UniverseWonderkid: true}
		if got := wk.EducationForAge(); got != tc.want {
			t.Errorf("age %d: got %q; want %q", tc.age, got, tc.want)
		}
	}

	mid := &Player{Education: "middle_school", SchoolWant: "school", UniverseWonderkid: true}
	if !strings.Contains(mid.EducationLabel(), "Middle school") {
		t.Errorf("middle label wrong: %q", mid.EducationLabel())
	}
	midNoWant := &Player{Education: "middle_school"}
	if !strings.Contains(midNoWant.EducationLabel(), "Middle school") {
		t.Errorf("middle label without want wrong: %q", midNoWant.EducationLabel())
	}
	for edu, want := range map[string]string{
		"high_school": "High school",
		"dropout":     "Left school",
		"graduated":   "Finished school",
		"none":        "—",
		"":            "—",
	} {
		p := &Player{Education: edu}
		if got := p.EducationLabel(); !strings.Contains(got, want) {
			t.Errorf("edu %q: got %q; want substring %q", edu, got, want)
		}
	}
}

func TestChunk1CovAvailabilityNoteVariants(t *testing.T) {
	inj := &Player{InjuredMatches: 2, Injury: "knock"}
	if got := inj.AvailabilityNote("x", 1); !strings.Contains(got, "2 matches") {
		t.Errorf("injured plural wrong: %q", got)
	}
	inj1 := &Player{InjuredMatches: 1}
	if got := inj1.AvailabilityNote("x", 1); !strings.Contains(got, "1 match") {
		t.Errorf("injured singular wrong: %q", got)
	}
	sus := &Player{SuspendedMatches: 3}
	if got := sus.AvailabilityNote("x", 1); !strings.Contains(got, "3 matches") {
		t.Errorf("suspended plural wrong: %q", got)
	}
	sus1 := &Player{SuspendedMatches: 1}
	if got := sus1.AvailabilityNote("x", 1); !strings.Contains(got, "1 match") {
		t.Errorf("suspended singular wrong: %q", got)
	}
	wkExam := &Player{UniverseWonderkid: true, Education: "middle_school", Age: 14}
	if got := wkExam.AvailabilityNote("super-league", 12); got != "Exams" {
		t.Errorf("exam note = %q; want Exams", got)
	}
	wkSchool := &Player{UniverseWonderkid: true, Education: "middle_school", Age: 14}
	if got := wkSchool.AvailabilityNote("ucl", 5); got != "School" {
		t.Errorf("school note = %q; want School", got)
	}
	free := &Player{}
	if got := free.AvailabilityNote("x", 1); got != "Available" {
		t.Errorf("available note = %q", got)
	}
	if free.SchoolWantLine() == "" {
		t.Errorf("SchoolWantLine should not be empty")
	}
}

func TestChunk1CovDecideAndAdvanceEducation(t *testing.T) {
	// football want + loyal + top club -> high_school.
	fb := &Player{SchoolWant: "football", Loyalty: 95, Appearances: 5}
	if got := fb.DecideEducation(2, 5); got != "high_school" {
		t.Errorf("loyal football want = %q; want high_school", got)
	}
	// football want + disloyal -> dropout.
	fb2 := &Player{SchoolWant: "football", Loyalty: 50, Appearances: 20}
	if got := fb2.DecideEducation(12, 20); got != "dropout" {
		t.Errorf("disloyal football want = %q; want dropout", got)
	}
	// school want + bad scene -> dropout.
	sc := &Player{SchoolWant: "school", Loyalty: 30, Appearances: 20}
	if got := sc.DecideEducation(12, 20); got != "dropout" {
		t.Errorf("bad-scene school want = %q; want dropout", got)
	}
	// school want default -> high_school.
	sc2 := &Player{SchoolWant: "school", Loyalty: 80, Appearances: 5}
	if got := sc2.DecideEducation(3, 5); got != "high_school" {
		t.Errorf("school want = %q; want high_school", got)
	}
	// open want: defaults for loyalty/place (0 -> 50/6), few apps at mid club -> high.
	op := &Player{SchoolWant: "open", Appearances: 5}
	if got := op.DecideEducation(0, 5); got != "high_school" {
		t.Errorf("open want good scene = %q; want high_school", got)
	}
	// open want: ever-present at relegated club -> dropout.
	op2 := &Player{SchoolWant: "open", Loyalty: 60, Appearances: 20}
	if got := op2.DecideEducation(14, 20); got != "dropout" {
		t.Errorf("open want bad scene = %q; want dropout", got)
	}
	// empty want resolves via SchoolWantFor.
	blank := &Player{FullName: "Blank Slate Junior", Loyalty: 80, Appearances: 3}
	_ = blank.DecideEducation(4, 3)

	// AdvanceEducation branches.
	if got := (&Player{}).AdvanceEducation(5); got != "" {
		t.Errorf("non-wonderkid advance = %q; want empty", got)
	}
	stay := &Player{FullName: "Stay Kid", UniverseWonderkid: true, Age: 16, Education: "middle_school", SchoolWant: "school", Loyalty: 80, Appearances: 4}
	if got := stay.AdvanceEducation(3); got != "stayed" {
		t.Errorf("stay advance = %q; want stayed", got)
	}
	left := &Player{FullName: "Left Kid", UniverseWonderkid: true, Age: 16, Education: "middle_school", SchoolWant: "football", Loyalty: 50, Appearances: 20}
	if got := left.AdvanceEducation(15); got != "left" {
		t.Errorf("left advance = %q; want left", got)
	}
	grad := &Player{UniverseWonderkid: true, Age: 18, Education: "high_school"}
	if got := grad.AdvanceEducation(5); got != "graduated" {
		t.Errorf("grad advance = %q; want graduated", got)
	}
	gradMid := &Player{UniverseWonderkid: true, Age: 19, Education: "high_school"}
	if got := gradMid.AdvanceEducation(5); got != "graduated" {
		t.Errorf("late grad advance = %q; want graduated", got)
	}
	// Age 18+ but still in middle school re-enters the age-16 decision track first.
	stillMid := &Player{FullName: "Still Mid", UniverseWonderkid: true, Age: 19, Education: "middle_school", SchoolWant: "football", Loyalty: 50, Appearances: 20}
	if got := stillMid.AdvanceEducation(15); got != "left" {
		t.Errorf("over-age middle advance = %q; want left", got)
	}
	young := &Player{UniverseWonderkid: true, Age: 14, Education: "middle_school"}
	if got := young.AdvanceEducation(5); got != "" {
		t.Errorf("young advance = %q; want empty", got)
	}
}

func TestChunk1CovLiveBestAndMiscPlayer(t *testing.T) {
	p := &Player{Goals: 10, Assists: 3, Season: "2026-27", BestGoals: 5, BestAssists: 5, BestSeason: "2025-26"}
	g, a, s := p.LiveBest("")
	if g != 10 || s != "2026-27" {
		t.Errorf("LiveBest current = %d/%d/%s", g, a, s)
	}
	_ = a
	p2 := &Player{Goals: 2, BestGoals: 9, BestAssists: 4, BestSeason: "2024-25"}
	if g2, _, _ := p2.LiveBest("2026-27"); g2 != 9 {
		t.Errorf("LiveBest best = %d; want 9", g2)
	}
	p3 := &Player{}
	if g3, _, _ := p3.LiveBest("2026-27"); g3 != 0 {
		t.Errorf("LiveBest empty = %d; want 0", g3)
	}
	// Explicit season name overrides.
	p4 := &Player{Goals: 7, Season: "2026-27", BestGoals: 1}
	if _, _, s4 := p4.LiveBest("2099-00"); s4 != "2099-00" {
		t.Errorf("LiveBest season override = %q", s4)
	}

	// EffectiveOVR floor at 40.
	tired := &Player{OVR: 41, ConsecutiveStarts: 9}
	if got := tired.EffectiveOVR(); got != 40 {
		t.Errorf("EffectiveOVR floor = %d; want 40", got)
	}
	mid := &Player{OVR: 80, ConsecutiveStarts: 3}
	if got := mid.EffectiveOVR(); got != 78 {
		t.Errorf("EffectiveOVR 3 starts = %d; want 78", got)
	}

	// Mentor / personality / records / formatting accessors.
	m := &Player{MentorID: "M1", MentorName: "Mentor", Personality: "flamboyant_star"}
	if !m.HasMentor() {
		t.Errorf("HasMentor should be true")
	}
	if (&Player{}).HasMentor() {
		t.Errorf("HasMentor should be false when empty")
	}
	if m.PersonalityTitle() == "" || m.PersonalityBadge() == "" {
		t.Errorf("personality title/badge should not be empty")
	}
	if got := m.PersonalityInfo().Key; got != "flamboyant_star" {
		t.Errorf("PersonalityInfo key = %q", got)
	}
	r := &Player{CareerGoals: 5, Goals: 2, CareerAssists: 4, Assists: 1, CareerApps: 10, Appearances: 3}
	if r.AllTimeGoals() != 7 || r.AllTimeAssists() != 5 || r.AllTimeApps() != 13 {
		t.Errorf("all-time totals wrong: %d/%d/%d", r.AllTimeGoals(), r.AllTimeAssists(), r.AllTimeApps())
	}
	r.RecordAppearance()
	r.RecordGoal()
	r.RecordAssist()
	if r.Appearances != 4 || r.Goals != 3 || r.Assists != 2 {
		t.Errorf("record increments wrong")
	}
	fw := &Player{WageEUR: 50000, MarketValueEUR: 10000000}
	if fw.FormattedWage() == "" || fw.FormattedValue() == "" {
		t.Errorf("formatted wage/value should not be empty")
	}
	if len((&Player{Position: "CAM"}).PositionOptions()) != 2 {
		t.Errorf("CAM should have 2 position options")
	}
	if PositionOptions("GK") != nil {
		t.Errorf("GK should have nil position options")
	}
}

func TestChunk1CovPlayerUnmarshalBranches(t *testing.T) {
	// Estimated value fallback + nested season stats + derived category.
	raw := `{"player_id":"U1","full_name":"Unmarshal One","position":"ST","ovr":75,
		"estimated_market_value_eur": 9000000,
		"season_stats": {"goals": 5, "assists": 2, "appearances": 9, "season": "2026-27"}}`
	var p Player
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if p.MarketValueEUR != 9000000 {
		t.Errorf("estimated value fallback = %d", p.MarketValueEUR)
	}
	if p.Goals != 5 || p.Assists != 2 || p.Appearances != 9 || p.Season != "2026-27" {
		t.Errorf("nested season stats wrong: %+v", p)
	}
	if p.Category != "FWD" {
		t.Errorf("derived category = %q", p.Category)
	}
	if p.WageEUR <= 0 || p.ContractYears != 3 || p.Loyalty != 65 {
		t.Errorf("defaults wrong: wage=%d contract=%d loyalty=%d", p.WageEUR, p.ContractYears, p.Loyalty)
	}
	if p.Education != "none" || p.SchoolWant != "open" || p.Personality != "dedicated_pro" {
		t.Errorf("non-WK defaults wrong: %+v", p)
	}
	if p.Composure != 75 || p.OriginalClubID != p.ClubID {
		t.Errorf("composure/original defaults wrong: %+v", p)
	}

	// Wonderkid loyalty bump + age-based education + resolved school want/personality.
	rawWK := `{"player_id":"WK_X","full_name":"Izyan Levin Bantol","position":"CAM","ovr":76,"age":14,
		"loyalty": 60, "universe_wonderkid": true, "club_id": "EPL-ARS"}`
	var wk Player
	if err := json.Unmarshal([]byte(rawWK), &wk); err != nil {
		t.Fatalf("WK unmarshal failed: %v", err)
	}
	if wk.Loyalty != 70 {
		t.Errorf("WK loyalty bump = %d; want 70", wk.Loyalty)
	}
	if wk.Education != "middle_school" {
		t.Errorf("WK education = %q", wk.Education)
	}
	if wk.SchoolWant == "" || wk.Personality == "" {
		t.Errorf("WK school want/personality should resolve: %+v", wk)
	}
	// Malformed JSON surfaces an error.
	var bad Player
	if err := json.Unmarshal([]byte(`{oops`), &bad); err == nil {
		t.Errorf("expected error for malformed JSON")
	}
}

func TestChunk1CovSchoolWantBucketsAndPersonality(t *testing.T) {
	seen := map[string]string{}
	candidates := []string{
		"Alex Carbon", "Bruno Sands", "Carlos Mendez", "Dario Fontana", "Enzo Moretti",
		"Finn Bakker", "Gabriel Ricci", "Hugo Nielsen", "Ivan Petrov", "Jonas Holm",
		"Kai Schneider", "Luka Horvath", "Mateo Vidal", "Nico Costa", "Omar Benali",
		"Pavel Novak", "Rafa Duarte", "Soren Lindqvist", "Theo Camara", "Valentin Gomez",
		"Xavi Hernandez Junior", "Yusuf Demir", "Zayn Hassan", "Amir Traore", "Leon Bauer",
		"Danilo Perez", "Dario Romero", "Samir Keita", "Tariq Mensah", "Elias Stankovic",
	}
	for _, n := range candidates {
		seen[SchoolWantFor(n)] = n
		if len(seen) == 3 {
			break
		}
	}
	for _, want := range []string{"school", "football", "open"} {
		if _, ok := seen[want]; !ok {
			t.Errorf("no candidate produced SchoolWant %q", want)
		}
	}
	if SchoolWantLabel("SCHOOL") != "Wants to finish school" {
		t.Errorf("school label wrong")
	}
	if SchoolWantLabel("Football") != "Wants football first" {
		t.Errorf("football label wrong")
	}
	if SchoolWantLabel("zzz") != "Has not made his mind up" {
		t.Errorf("default label wrong")
	}
	if got := PersonalityFor("Maverick Cantalejo"); got != "big_game_performer" {
		t.Errorf("canonical personality = %q", got)
	}
	if got := PersonalityFor("Some Random Player"); got == "" {
		t.Errorf("hashed personality should not be empty")
	}
	if ArchetypeForKey("nope").Key != "dedicated_pro" {
		t.Errorf("unknown archetype should default to dedicated_pro")
	}
}

func TestChunk1CovStandingsRecordTrim(t *testing.T) {
	var cr CompetitionRecord
	for i := 0; i < 4; i++ {
		cr.UpdateResult(1, 0) // W
	}
	cr.UpdateResult(0, 0) // D
	cr.UpdateResult(0, 2) // L
	cr.UpdateResult(2, 2) // D -> 7 entries, trimmed to 5
	if len(cr.Form) != 5 {
		t.Errorf("form trimmed = %d; want 5", len(cr.Form))
	}
	if cr.Played != 7 || cr.Won != 4 || cr.Drawn != 2 || cr.Lost != 1 || cr.Points != 14 {
		t.Errorf("record wrong: %+v", cr)
	}
	cr.Reset()
	if cr.Played != 0 || len(cr.Form) != 0 || cr.Points != 0 || cr.GoalDifference != 0 {
		t.Errorf("reset wrong: %+v", cr)
	}
}

func TestChunk1CovUnmarshalJSONErrors(t *testing.T) {
	var c Club
	if err := c.UnmarshalJSON([]byte(`{oops`)); err == nil {
		t.Errorf("expected Club unmarshal error")
	}
	var p Player
	if err := p.UnmarshalJSON([]byte(`{oops`)); err == nil {
		t.Errorf("expected Player unmarshal error")
	}
}

func TestChunk1CovStartingElevenTinySquad(t *testing.T) {
	// 3-player squad: everyone is picked, no fill needed, no duplicates.
	c := &Club{ClubID: "TINY"}
	c.Squad = []*Player{
		{PlayerID: "G1", Category: "GK", OVR: 70},
		{PlayerID: "D1", Category: "DEF", OVR: 72},
		{PlayerID: "M1", Category: "MID", OVR: 74},
	}
	xi := c.GetStartingEleven()
	if len(xi) != 3 {
		t.Errorf("tiny XI = %d; want 3", len(xi))
	}
	seen := map[*Player]bool{}
	for _, p := range xi {
		if seen[p] {
			t.Errorf("duplicate in tiny XI: %s", p.PlayerID)
		}
		seen[p] = true
	}

	// 6-DEF no-GK squad: the 4-DEF take cap binds, so remain (2) is shorter
	// than needed (7), exercising the needed=len(remain) cap.
	c2 := &Club{ClubID: "DEF6"}
	for i := 0; i < 6; i++ {
		c2.Squad = append(c2.Squad, &Player{PlayerID: fmt.Sprintf("DD%d", i), Category: "DEF", OVR: 70 + i})
	}
	xi2 := c2.GetStartingEleven()
	if len(xi2) != 6 {
		t.Errorf("capped XI = %d; want 6", len(xi2))
	}
	seen2 := map[*Player]bool{}
	for _, p := range xi2 {
		if seen2[p] {
			t.Errorf("duplicate in capped XI: %s", p.PlayerID)
		}
		seen2[p] = true
	}
}

func TestChunk1CovGetBenchWonderkidComparator(t *testing.T) {
	// Bench pool [wkA, T1, T2]: k=1 compares (T1, wkA) with the wonderkid in
	// the j slot (wkJ=1 path), and T1/T2 tie on status+OVR (ID tiebreak).
	c := &Club{ClubID: "WKC"}
	c.Squad = append(c.Squad, &Player{PlayerID: "G1", Category: "GK", OVR: 85})
	for i, ovr := range []int{86, 87, 88, 89} {
		c.Squad = append(c.Squad, &Player{PlayerID: fmt.Sprintf("D%d", i), Category: "DEF", OVR: ovr})
	}
	for i, ovr := range []int{90, 91, 92} {
		c.Squad = append(c.Squad, &Player{PlayerID: fmt.Sprintf("M%d", i), Category: "MID", OVR: ovr})
	}
	for i, ovr := range []int{93, 94, 95} {
		c.Squad = append(c.Squad, &Player{PlayerID: fmt.Sprintf("F%d", i), Category: "FWD", OVR: ovr})
	}
	c.Squad = append(c.Squad,
		&Player{PlayerID: "wkA", Category: "MID", OVR: 60, UniverseWonderkid: true, ConsecutiveStarts: 5},
		&Player{PlayerID: "T1", Category: "MID", OVR: 70},
		&Player{PlayerID: "T2", Category: "MID", OVR: 70},
	)
	starters := c.GetStartingEleven()
	if len(starters) != 11 {
		t.Fatalf("XI = %d; want 11", len(starters))
	}
	bench := c.GetBench(starters, 7)
	if len(bench) != 3 {
		t.Fatalf("bench = %d; want 3", len(bench))
	}
	want := []string{"wkA", "T1", "T2"}
	for i, id := range want {
		if bench[i].PlayerID != id {
			t.Errorf("bench[%d] = %s; want %s", i, bench[i].PlayerID, id)
		}
	}
}
