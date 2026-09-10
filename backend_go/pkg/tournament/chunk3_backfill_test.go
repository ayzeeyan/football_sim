package tournament

import (
	"encoding/json"
	"math/rand"
	"testing"

	"football_sim/pkg/growth"
	"football_sim/pkg/managers"
	"football_sim/pkg/models"
)

// Chunk 3 backfill: pre-existing season-sim surface (fixtures, mentorship,
// milestones, standings flows, head-to-head, records).

func TestChunk3BackfillFixtures(t *testing.T) {
	if got := GenerateLeagueFixtures(nil, rand.New(rand.NewSource(1))); got != nil {
		t.Errorf("nil clubs should yield nil fixtures")
	}
	solo := []*models.Club{{ClubID: "SOLO"}}
	if got := GenerateLeagueFixtures(solo, rand.New(rand.NewSource(1))); got != nil {
		t.Errorf("single club should yield nil fixtures")
	}
	var f Fixture
	if err := f.UnmarshalJSON([]byte(`{oops`)); err == nil {
		t.Errorf("malformed fixture JSON should error")
	}
	var f2 Fixture
	if err := json.Unmarshal([]byte(`{"id": "FX9", "competition": "super-league"}`), &f2); err != nil {
		t.Fatalf("fixture unmarshal failed: %v", err)
	}
	if f2.FixtureID != "FX9" {
		t.Errorf("id fallback wrong: %q", f2.FixtureID)
	}
	var f3 Fixture
	if err := json.Unmarshal([]byte(`{"fixture_id": "FX10", "id": "IGNORED"}`), &f3); err != nil || f3.FixtureID != "FX10" {
		t.Errorf("explicit fixture_id should win: %+v / %v", f3, err)
	}
}

func TestChunk3BackfillMentorship(t *testing.T) {
	ge := growth.NewGrowthEngine(801)
	// Existing mentor still at the club: refreshed, both engine paths.
	vet := &models.Player{PlayerID: "V1", FullName: "Veteran", Position: "ST", Category: "FWD", OVR: 84, Age: 32}
	wk := &models.Player{PlayerID: "WK1", FullName: "Kid", Position: "ST", Category: "FWD", OVR: 78, Age: 16,
		UniverseWonderkid: true, MentorID: "V1", Personality: "dedicated_pro"}
	club := &models.Club{ClubID: "C1", Squad: []*models.Player{vet, wk}}
	PairSeniorMentors([]*models.Club{club}, ge)
	if wk.MentorName != "Veteran" || wk.MentorOVR != 84 {
		t.Errorf("existing mentor not refreshed: %+v", wk)
	}
	PairSeniorMentors([]*models.Club{club}, nil)
	if wk.MentorName != "Veteran" {
		t.Errorf("nil-engine refresh should still stamp fields")
	}
	// Stale mentor ID: cleared and re-paired.
	wk.MentorID = "GONE"
	PairSeniorMentors([]*models.Club{club}, ge)
	if wk.MentorID != "V1" {
		t.Errorf("stale mentor should re-pair to V1, got %q", wk.MentorID)
	}
	// Young-only squad: fallback pool (no 26+ veterans).
	young := &models.Club{ClubID: "C2", Squad: []*models.Player{
		{PlayerID: "WK2", FullName: "Kid2", Position: "CM", Category: "MID", OVR: 76, Age: 16, UniverseWonderkid: true},
		{PlayerID: "Y1", FullName: "Young", Position: "CM", Category: "MID", OVR: 70, Age: 21},
	}}
	PairSeniorMentors([]*models.Club{young}, nil)
	if young.Squad[0].MentorID != "Y1" {
		t.Errorf("young fallback pairing wrong: %+v", young.Squad[0])
	}
	// Lone wonderkid: nobody to pair.
	lone := &models.Club{ClubID: "C3", Squad: []*models.Player{
		{PlayerID: "WK3", FullName: "Solo", Position: "ST", Category: "FWD", OVR: 76, Age: 16, UniverseWonderkid: true},
	}}
	PairSeniorMentors([]*models.Club{lone}, nil)
	if lone.Squad[0].MentorID != "" {
		t.Errorf("lone wonderkid must stay unpaired")
	}
	// OVR tie prefers the older veteran.
	tie := &models.Club{ClubID: "C4", Squad: []*models.Player{
		{PlayerID: "WK4", FullName: "Kid4", Position: "ST", Category: "FWD", OVR: 76, Age: 16, UniverseWonderkid: true},
		{PlayerID: "O1", FullName: "Old", Position: "ST", Category: "FWD", OVR: 82, Age: 34},
		{PlayerID: "O2", FullName: "Mid", Position: "ST", Category: "FWD", OVR: 82, Age: 28},
	}}
	PairSeniorMentors([]*models.Club{tie}, nil)
	if tie.Squad[0].MentorID != "O1" {
		t.Errorf("OVR tie should prefer older veteran, got %q", tie.Squad[0].MentorID)
	}
}

func TestChunk3BackfillMilestones(t *testing.T) {
	mkWK := func(id string, age int) *models.Player {
		return &models.Player{PlayerID: id, FullName: "Kid " + id, Position: "ST", Category: "FWD",
			OVR: 78, Age: age, UniverseWonderkid: true, Personality: "dedicated_pro", MentorName: "Vet"}
	}
	club := &models.Club{ClubID: "C1", ClubName: "Club One", ShortName: "CONE", Squad: []*models.Player{
		mkWK("W15", 15), mkWK("W17", 17), mkWK("W18", 18), mkWK("W14", 14),
	}}
	fired := map[string]map[string]bool{}
	news := CheckWonderkidMilestones(20, "2026-27", []*models.Club{club}, fired, rand.New(rand.NewSource(2)))
	if len(news) != 3 {
		t.Fatalf("milestones fired = %d; want 3 (15/17/18)", len(news))
	}
	// Re-fire is idempotent.
	if again := CheckWonderkidMilestones(20, "2026-27", []*models.Club{club}, fired, rand.New(rand.NewSource(2))); len(again) != 0 {
		t.Errorf("milestones should fire once, got %d", len(again))
	}
	// Below-threshold matchweeks stay quiet.
	fresh := map[string]map[string]bool{}
	if got := CheckWonderkidMilestones(3, "2026-27", []*models.Club{club}, fresh, rand.New(rand.NewSource(2))); len(got) != 0 {
		t.Errorf("mw 3 should fire nothing, got %d", len(got))
	}
	fresh2 := map[string]map[string]bool{}
	if got := CheckWonderkidMilestones(9, "2026-27", []*models.Club{club}, fresh2, rand.New(rand.NewSource(2))); len(got) != 1 {
		t.Errorf("mw 9 should fire only the age-15 deal, got %d", len(got))
	}
	fresh3 := map[string]map[string]bool{}
	if got := CheckWonderkidMilestones(19, "2026-27", []*models.Club{club}, fresh3, rand.New(rand.NewSource(2))); len(got) != 2 {
		t.Errorf("mw 19 should fire 15+17 deals, got %d", len(got))
	}
}

func TestChunk3BackfillNXGNBreak(t *testing.T) {
	club := &models.Club{ClubID: "C1", ClubName: "Club One", ShortName: "CONE"}
	for i := 0; i < 55; i++ {
		club.Squad = append(club.Squad, &models.Player{
			PlayerID: string(rune('A'+i%26)) + string(rune('a'+i/26)) + "X",
			FullName: "Prospect", Position: "CM", Category: "MID",
			OVR: 60 + i%20, Age: 16 + i%4, ClubID: "C1",
		})
	}
	rankings := GenerateNXGN50([]*models.Club{club}, nil)
	if len(rankings) != 50 {
		t.Errorf("rankings capped = %d; want 50", len(rankings))
	}
	for i := 1; i < len(rankings); i++ {
		if rankings[i].Rank != i+1 {
			t.Errorf("rank numbering broken at %d", i)
		}
	}
}

func TestChunk3BackfillManagerFlows(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	_ = tm
	// Zero seed takes the deterministic default.
	tm0 := NewTournamentManager(nil, nil, 0)
	if tm0.RNG == nil || len(tm0.ClubsList) != 0 {
		t.Errorf("zero-seed construction wrong")
	}
	// Invalid matchweeks are rejected.
	if res := tm0.SimulateMatchweek(0); res["status"] != "error" {
		t.Errorf("mw 0 should error: %v", res)
	}
	if res := tm0.SimulateMatchweek(99); res["status"] != "error" {
		t.Errorf("mw 99 should error: %v", res)
	}
	// Fixtures pointing at missing clubs are skipped safely.
	ghost := Fixture{FixtureID: "GHOST", Matchweek: 1, Competition: "super-league", HomeID: "NOPE", AwayID: "MISSING"}
	tm0.Fixtures = []Fixture{ghost}
	tm0.MaxMatchweeks = 33
	if res := tm0.SimulateMatchweek(1); res["simulated_count"] != 0 {
		t.Errorf("ghost fixture should be skipped: %v", res)
	}

	// Derby heat movement across margins plus the 100 cap.
	heatSeen := map[int]bool{}
	for seed := int64(0); seed < 30; seed++ {
		h := chunk3mkClub("DH")
		a := chunk3mkClub("DA")
		mini := &TournamentManager{
			Clubs: map[string]*models.Club{"DH": h, "DA": a}, ClubsList: []*models.Club{h, a},
			Managers: map[string]*managers.ManagerProfile{}, RNG: rand.New(rand.NewSource(seed)),
			DerbyHeat: map[string]int{}, MilestonesFired: map[string]map[string]bool{},
			SeasonName: "2026-27", MaxMatchweeks: 33, CurrentMatchweek: 1,
			Fixtures: []Fixture{{FixtureID: "DB1", Matchweek: 1, Competition: "super-league",
				HomeID: "DH", AwayID: "DA", DerbyName: "Test Derby"}},
		}
		res := mini.SimulateMatchweek(1)
		if res["simulated_count"] != 1 {
			t.Fatalf("seed %d: derby fixture not simulated: %v", seed, res)
		}
		heatSeen[mini.DerbyHeat["DH_DA"]] = true
		if mini.DerbyHeat["DA_DH"] != mini.DerbyHeat["DH_DA"] {
			t.Errorf("seed %d: reverse heat not mirrored", seed)
		}
	}
	for h := range heatSeen {
		if h < 50 || h > 100 {
			t.Errorf("unexpected heat value %d (want 50-100)", h)
		}
	}
	if !heatSeen[55] && !heatSeen[65] {
		t.Errorf("one-goal margin (+5, or +15 with a red) never observed in 30 derbies")
	}
	// Cap at 100 from a hot start.
	capped := false
	for seed := int64(0); seed < 30 && !capped; seed++ {
		h := chunk3mkClub("DH")
		a := chunk3mkClub("DA")
		mini := &TournamentManager{
			Clubs: map[string]*models.Club{"DH": h, "DA": a}, ClubsList: []*models.Club{h, a},
			Managers: map[string]*managers.ManagerProfile{}, RNG: rand.New(rand.NewSource(900 + seed)),
			DerbyHeat: map[string]int{"DH_DA": 99, "DA_DH": 99}, MilestonesFired: map[string]map[string]bool{},
			SeasonName: "2026-27", MaxMatchweeks: 33, CurrentMatchweek: 1,
			Fixtures: []Fixture{{FixtureID: "DB2", Matchweek: 1, Competition: "super-league",
				HomeID: "DH", AwayID: "DA", DerbyName: "Test Derby"}},
		}
		mini.SimulateMatchweek(1)
		if mini.DerbyHeat["DH_DA"] == 100 {
			capped = true
		} else if mini.DerbyHeat["DH_DA"] > 100 {
			t.Fatalf("heat exceeded 100: %d", mini.DerbyHeat["DH_DA"])
		}
	}
	if !capped {
		t.Errorf("heat cap 100 never observed in 30 hot derbies")
	}
}

func TestChunk3BackfillWeekTriggers(t *testing.T) {
	mkMini := func(mw int, seed int64) *TournamentManager {
		h := chunk3mkClub("WH")
		a := chunk3mkClub("WA")
		return &TournamentManager{
			Clubs: map[string]*models.Club{"WH": h, "WA": a}, ClubsList: []*models.Club{h, a},
			Managers: map[string]*managers.ManagerProfile{}, RNG: rand.New(rand.NewSource(seed)),
			DerbyHeat: map[string]int{}, MilestonesFired: map[string]map[string]bool{},
			SeasonName: "2026-27", MaxMatchweeks: 33, CurrentMatchweek: mw,
			Fixtures: []Fixture{{FixtureID: "WX", Matchweek: mw, Competition: "super-league",
				HomeID: "WH", AwayID: "WA"}},
		}
	}
	// NXGN wire at matchweek 24.
	nx := mkMini(24, 11)
	nx.SimulateMatchweek(24)
	foundNXGN := false
	for _, item := range nx.Inbox {
		if item.Category == "nxgn" {
			foundNXGN = true
		}
	}
	if !foundNXGN {
		t.Errorf("mw 24 should push the NXGN wire")
	}
	// Championship coronation at the final round.
	ch := mkMini(33, 12)
	ch.SimulateMatchweek(33)
	if ch.SeasonPhase != "transfer_window" {
		t.Errorf("season should close after mw 33, phase=%q", ch.SeasonPhase)
	}
	foundHonour := false
	for _, item := range ch.Inbox {
		if item.Category == "honour" {
			foundHonour = true
		}
	}
	if !foundHonour {
		t.Errorf("final round should crown a champion")
	}
}

func TestChunk3BackfillHeadToHead(t *testing.T) {
	intPtr := func(v int) *int { return &v }
	mkClub := func(id, short string) *models.Club {
		return &models.Club{ClubID: id, ClubName: id + " FC", ShortName: short}
	}
	tm := &TournamentManager{
		Clubs:     map[string]*models.Club{"A": mkClub("A", "AAA"), "B": mkClub("B", "BBB")},
		DerbyHeat: map[string]int{},
		Fixtures: []Fixture{
			{FixtureID: "F1", Matchweek: 1, HomeID: "A", AwayID: "B", Status: "finished", HomeGoals: intPtr(2), AwayGoals: intPtr(1)},
			{FixtureID: "F2", Matchweek: 2, HomeID: "B", AwayID: "A", Status: "finished", HomeGoals: intPtr(0), AwayGoals: intPtr(3)},
			{FixtureID: "F3", Matchweek: 3, HomeID: "A", AwayID: "B", Status: "finished", HomeGoals: intPtr(1), AwayGoals: intPtr(1)},
			{FixtureID: "F4", Matchweek: 4, HomeID: "A", AwayID: "B", Status: "scheduled"},
			{FixtureID: "F5", Matchweek: 5, HomeID: "A", AwayID: "B", Status: "finished"},
		},
	}
	if got := tm.GetHeadToHead("A", "ZZZ"); got != nil {
		t.Errorf("unknown club H2H should be nil")
	}
	if got := tm.GetHeadToHead("ZZZ", "A"); got != nil {
		t.Errorf("unknown club H2H (reversed) should be nil")
	}
	h2h := tm.GetHeadToHead("A", "B")
	if h2h == nil {
		t.Fatalf("H2H should resolve")
	}
	if h2h["matches_played"] != 3 || h2h["wins_a"] != 2 || h2h["wins_b"] != 0 || h2h["draws"] != 1 {
		t.Errorf("H2H tally wrong: %+v", h2h)
	}
	if h2h["goals_a"] != 6 || h2h["goals_b"] != 2 {
		t.Errorf("H2H goals wrong: %+v", h2h)
	}
	if h2h["derby_heat"] != 50 {
		t.Errorf("missing heat should default 50: %+v", h2h["derby_heat"])
	}
	if recent, ok := h2h["recent_matches"].([]map[string]interface{}); !ok || len(recent) != 3 {
		t.Errorf("recent matches wrong: %+v", h2h["recent_matches"])
	}
	// Reversed perspective swaps the ledger.
	rev := tm.GetHeadToHead("B", "A")
	if rev["wins_a"] != 0 || rev["wins_b"] != 2 || rev["goals_a"] != 2 || rev["goals_b"] != 6 {
		t.Errorf("reversed H2H wrong: %+v", rev)
	}
}

func TestChunk3BackfillAllTimeRecords(t *testing.T) {
	ge := growth.NewGrowthEngine(606)
	home, away := chunk3mkClub("R1"), chunk3mkClub("R2")
	home.Squad[8].Goals, home.Squad[8].CareerGoals = 9, 20
	home.Squad[5].Assists, home.Squad[5].CareerAssists = 8, 15
	away.Squad[3].Goals = 30 // outright season leader
	wk := home.Squad[9]
	wk.UniverseWonderkid = true
	ge.RegisterProdigy(wk.PlayerID, wk.FullName, 16, 178, 70, "FWD", 80, 94, 19)
	hg, ag := 4, 1
	tm := &TournamentManager{
		Clubs:        map[string]*models.Club{"R1": home, "R2": away},
		ClubsList:    []*models.Club{home, away},
		GrowthEngine: ge,
		Fixtures: []Fixture{
			{FixtureID: "RF1", Matchweek: 1, HomeID: "R1", AwayID: "R2", Status: "finished", HomeGoals: &hg, AwayGoals: &ag},
			{FixtureID: "RF2", Matchweek: 2, HomeID: "R2", AwayID: "R1", Status: "finished", HomeGoals: &ag, AwayGoals: &hg},
			{FixtureID: "RF3", Matchweek: 3, HomeID: "R1", AwayID: "R2", Status: "scheduled"},
		},
	}
	rec := tm.GetAllTimeRecords()
	scorers := rec["top_goalscorers"].([]map[string]interface{})
	if len(scorers) == 0 || scorers[0]["goals"] != 30 {
		t.Errorf("top scorer wrong: %+v", rec["top_goalscorers"])
	}
	assisters := rec["top_assisters"].([]map[string]interface{})
	if len(assisters) == 0 || assisters[0]["assists"] != 23 {
		t.Errorf("top assister wrong: %+v", rec["top_assisters"])
	}
	if hs, ok := rec["highest_scoring_match"].(map[string]interface{}); !ok || hs["goals"] != 5 {
		t.Errorf("highest-scoring match wrong: %+v", rec["highest_scoring_match"])
	}
	if bm, ok := rec["biggest_margin"].(map[string]interface{}); !ok || bm["margin"] != 3 {
		t.Errorf("biggest margin wrong: %+v", rec["biggest_margin"])
	}
	if hk, ok := rec["highest_ovr_wonderkid"].(map[string]interface{}); !ok || hk["potential"] != 94 {
		t.Errorf("wonderkid watch wrong: %+v", rec["highest_ovr_wonderkid"])
	}
	// Empty universe: keys present, features nil.
	empty := (&TournamentManager{}).GetAllTimeRecords()
	if empty["top_goalscorers"] != nil || empty["highest_scoring_match"] != nil ||
		empty["biggest_margin"] != nil || empty["highest_ovr_wonderkid"] != nil {
		t.Errorf("empty records should be nil-valued: %+v", empty)
	}
	// Wonderkid without biometrics falls back to potential 95.
	bare := chunk3mkClub("BW")
	bare.Squad[9].UniverseWonderkid = true
	bareTM := &TournamentManager{
		Clubs: map[string]*models.Club{"BW": bare}, ClubsList: []*models.Club{bare},
		GrowthEngine: growth.NewGrowthEngine(607),
	}
	rec2 := bareTM.GetAllTimeRecords()
	if hk, ok := rec2["highest_ovr_wonderkid"].(map[string]interface{}); !ok || hk["potential"] != 95 {
		t.Errorf("unregistered wonderkid potential should default 95: %+v", rec2["highest_ovr_wonderkid"])
	}
}
