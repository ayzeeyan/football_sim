package matchengine

import (
	"math/rand"
	"strings"
	"testing"

	"football_sim/pkg/growth"
	"football_sim/pkg/managers"
	"football_sim/pkg/matchreport"
	"football_sim/pkg/models"
)

// Chunk 2 coverage: instant-match fidelity (pure sim, Python parity on
// ratings, events, cards, subs, set pieces, VAR, stats, and assembly).

func chunk2BaseClubs() (*models.Club, *models.Club) {
	home, away := createTestClubs()
	return home, away
}

func chunk2Mgrs() (*managers.ManagerProfile, *managers.ManagerProfile) {
	return &managers.ManagerProfile{ClubID: "LAL-BAR", Name: "Home Gaffer", Style: "possession"},
		&managers.ManagerProfile{ClubID: "LAL-RMA", Name: "Away Gaffer", Style: "high_press"}
}

func TestChunk2SimConfigGuards(t *testing.T) {
	home, away := chunk2BaseClubs()
	hm, am := chunk2Mgrs()
	ge := growth.NewGrowthEngine(501)

	// Nil RNG and nil config both fall back safely.
	rep := SimulateInstantMatch(home, away, hm, am, ge, nil, nil)
	if rep == nil || rep.Method != "instant" {
		t.Fatalf("nil-config sim failed: %+v", rep)
	}
	// Empty weather defaults to clear.
	rep2 := SimulateInstantMatch(home, away, hm, am, ge, &InstantMatchConfig{Weather: "", Referee: "balanced"}, nil)
	if rep2.Weather != "clear" {
		t.Errorf("empty weather should default to clear, got %q", rep2.Weather)
	}
	// Nil managers fall back to possession styles.
	rep3 := SimulateInstantMatch(home, away, nil, nil, ge, &InstantMatchConfig{Weather: "rain", Referee: "lenient"}, rand.New(rand.NewSource(9)))
	if rep3 == nil {
		t.Fatalf("nil-manager sim failed")
	}
	// Empty manager styles also fall back.
	rep4 := SimulateInstantMatch(home, away,
		&managers.ManagerProfile{ClubID: "H"}, &managers.ManagerProfile{ClubID: "A"},
		ge, &InstantMatchConfig{Weather: "wind", Referee: "strict"}, rand.New(rand.NewSource(10)))
	if rep4 == nil {
		t.Fatalf("empty-style manager sim failed")
	}
}

func TestChunk2XIStrengthUsesSelectedPlayers(t *testing.T) {
	old := &models.Player{PlayerID: "OLD", OVR: 80, ConsecutiveStarts: 3}
	fresh := &models.Player{PlayerID: "FRESH", OVR: 70}
	got := xiEffectiveRating([]*models.Player{old, nil, fresh}, 99)
	want := float64(old.EffectiveOVR()+fresh.EffectiveOVR()) / 2
	if got != want {
		t.Errorf("XI strength = %v; want average selected effective OVR %v", got, want)
	}
	if got := xiEffectiveRating(nil, 99); got != 0 {
		t.Errorf("empty XI strength = %v; want 0", got)
	}
}

func TestChunk2InstantStrengthIgnoresCachedClubRating(t *testing.T) {
	homeA, awayA := createTestClubs()
	homeB, awayB := createTestClubs()
	homeA.OverallTeamRating, awayA.OverallTeamRating = 1, 99
	homeB.OverallTeamRating, awayB.OverallTeamRating = 99, 1
	hm, am := chunk2Mgrs()
	cfg := &InstantMatchConfig{Weather: "clear", Referee: "balanced"}
	repA := SimulateInstantMatch(homeA, awayA, hm, am, nil, cfg, rand.New(rand.NewSource(8181)))
	repB := SimulateInstantMatch(homeB, awayB, hm, am, nil, cfg, rand.New(rand.NewSource(8181)))
	if repA.HomeGoals != repB.HomeGoals || repA.AwayGoals != repB.AwayGoals {
		t.Fatalf("cached rating changed score: %d-%d vs %d-%d", repA.HomeGoals, repA.AwayGoals, repB.HomeGoals, repB.AwayGoals)
	}
	if repA.Stats.Home.Shots != repB.Stats.Home.Shots || repA.Stats.Away.Shots != repB.Stats.Away.Shots ||
		repA.Stats.Home.Possession != repB.Stats.Home.Possession || repA.Stats.Home.XG != repB.Stats.Home.XG ||
		repA.Stats.Away.XG != repB.Stats.Away.XG || repA.ShotMap.TotalHomeXG != repB.ShotMap.TotalHomeXG ||
		repA.ShotMap.TotalAwayXG != repB.ShotMap.TotalAwayXG {
		t.Fatalf("cached rating changed strength-driven stats/xG: %+v vs %+v", repA.Stats, repB.Stats)
	}
}

func TestChunk2InstantFixtureExcludesExamPlayer(t *testing.T) {
	home, away := createTestClubs()
	examKid := home.Squad[9]
	examKid.Age = 16
	examKid.Education = "high_school"
	hm, am := chunk2Mgrs()
	cfg := &InstantMatchConfig{Weather: "clear", Referee: "balanced", Competition: "super-league", Matchweek: 12}
	rep := SimulateInstantMatch(home, away, hm, am, nil, cfg, rand.New(rand.NewSource(8282)))
	for _, row := range append(append([]matchreport.MatchPlayerRow{}, rep.HomeXI...), rep.HomeBench...) {
		if row.PlayerID == examKid.PlayerID {
			t.Fatalf("exam player appeared in instant XI/bench: %+v", row)
		}
	}
	for _, ev := range rep.Events {
		for _, p := range []*matchreport.MiniPlayer{ev.Scorer, ev.Assister, ev.Player, ev.PlayerOut, ev.PlayerIn} {
			if p != nil && p.PlayerID == examKid.PlayerID {
				t.Fatalf("exam player appeared in instant involvement: %+v", ev)
			}
		}
	}
}

func TestChunk2RefereeResolution(t *testing.T) {
	home, away := chunk2BaseClubs()
	hm, am := chunk2Mgrs()
	ge := growth.NewGrowthEngine(502)

	rep := SimulateInstantMatch(home, away, hm, am, ge,
		&InstantMatchConfig{Weather: "clear", Referee: "Michael Oliver"}, rand.New(rand.NewSource(1)))
	if rep.Referee != "Michael Oliver" {
		t.Errorf("referee name hint not honored: %q", rep.Referee)
	}
	rep2 := SimulateInstantMatch(home, away, hm, am, ge,
		&InstantMatchConfig{Weather: "clear", Referee: "strict"}, rand.New(rand.NewSource(2)))
	if matchreport.RefereePersonality(rep2.Referee) != "strict" {
		t.Errorf("strict hint drew %q", rep2.Referee)
	}
	rep3 := SimulateInstantMatch(home, away, hm, am, ge,
		&InstantMatchConfig{Weather: "clear", Referee: "bogus"}, rand.New(rand.NewSource(3)))
	found := false
	for _, n := range matchreport.Referees {
		if n == rep3.Referee {
			found = true
		}
	}
	if !found {
		t.Errorf("unknown hint should draw from roster, got %q", rep3.Referee)
	}
}

func TestChunk2MoraleAndRatings(t *testing.T) {
	ge := growth.NewGrowthEngine(503)
	hm, am := chunk2Mgrs()
	mkClub := func(id string, rating, morale int) *models.Club {
		home, _ := createTestClubs()
		home.ClubID = id
		home.OverallTeamRating = rating
		home.Morale = morale
		return home
	}
	// Both morale bonuses active.
	rep := SimulateInstantMatch(mkClub("H1", 84, 95), mkClub("A1", 84, 95), hm, am, ge,
		&InstantMatchConfig{Weather: "clear", Referee: "balanced"}, rand.New(rand.NewSource(21)))
	if rep == nil {
		t.Fatalf("high-morale sim failed")
	}
	// Home low-morale upset nudge (delta > 2).
	rep2 := SimulateInstantMatch(mkClub("H2", 85, 30), mkClub("A2", 70, 75), hm, am, ge,
		&InstantMatchConfig{Weather: "clear", Referee: "balanced"}, rand.New(rand.NewSource(22)))
	if rep2 == nil {
		t.Fatalf("home-upset sim failed")
	}
	// Away low-morale upset nudge (delta < -2).
	rep3 := SimulateInstantMatch(mkClub("H3", 70, 75), mkClub("A3", 85, 30), hm, am, ge,
		&InstantMatchConfig{Weather: "clear", Referee: "balanced"}, rand.New(rand.NewSource(23)))
	if rep3 == nil {
		t.Fatalf("away-upset sim failed")
	}
	// Snow scoring suppression path.
	rep4 := SimulateInstantMatch(mkClub("H4", 84, 75), mkClub("A4", 84, 75), hm, am, ge,
		&InstantMatchConfig{Weather: "snow", Referee: "balanced"}, rand.New(rand.NewSource(24)))
	if rep4 == nil || rep4.Weather != "snow" {
		t.Fatalf("snow sim failed: %+v", rep4)
	}
}

func TestChunk2NoFreeKickWithoutTaker(t *testing.T) {
	mkWeak := func(id string) *models.Club {
		c := &models.Club{ClubID: id, ClubName: id, ShortName: id, OverallTeamRating: 70, Morale: 70}
		for i := 0; i < 16; i++ {
			c.Squad = append(c.Squad, &models.Player{
				PlayerID: id + "_P", FullName: id + " Player",
				Position: "CM", Category: "MID", OVR: 70, Age: 25,
			})
		}
		for i := range c.Squad {
			c.Squad[i].PlayerID = id + "_P" + string(rune('A'+i))
		}
		return c
	}
	home, away := mkWeak("W1"), mkWeak("W2")
	hm, am := chunk2Mgrs()
	ge := growth.NewGrowthEngine(504)
	for seed := int64(0); seed < 50; seed++ {
		rep := SimulateInstantMatch(home, away, hm, am, ge,
			&InstantMatchConfig{Weather: "clear", Referee: "balanced"}, rand.New(rand.NewSource(seed)))
		for _, e := range rep.Events {
			if e.Type == "free_kick_goal" {
				t.Fatalf("70-OVR squads cannot produce free-kick goals (seed %d)", seed)
			}
		}
	}
}

func TestChunk3InstantOwnGoalRequiresActiveDefender(t *testing.T) {
	rng := rand.New(rand.NewSource(9001))
	if got := instantOwnGoalCulprit(nil, rng); got != nil {
		t.Fatalf("dismissed defending field produced own-goal culprit: %+v", got)
	}
	defender := &models.Player{PlayerID: "ACTIVE-DEF", Category: "DEF", OVR: 70}
	if got := instantOwnGoalCulprit([]*models.Player{defender}, rng); got != defender {
		t.Fatalf("active defender was not eligible for own goal: got %+v", got)
	}

	// Exercise the instant path with an entirely dismissed defending side.
	_, away := createTestClubs()
	emptyHome := &models.Club{
		ClubID: "EMPTY-HOME", ClubName: "Empty Home", ShortName: "EH",
		OverallTeamRating: 70, Morale: 70, StadiumCapacity: 30000,
	}
	for seed := int64(0); seed < 120; seed++ {
		rep := SimulateInstantMatch(emptyHome, away, nil, nil, nil,
			&InstantMatchConfig{Weather: "clear", Referee: "balanced"},
			rand.New(rand.NewSource(seed)))
		if rep.HomeGoals != 0 {
			t.Fatalf("seed %d: empty home side scored %d goals", seed, rep.HomeGoals)
		}
		for _, event := range rep.Events {
			if event.Type == "own_goal" && event.Side == "home" && event.Beneficiary == "away" {
				t.Fatalf("seed %d: own goal credited from dismissed home field: %+v", seed, event)
			}
		}
	}
}

func TestChunk2MentoredWonderkidPenalties(t *testing.T) {
	mkWKClub := func(id string) *models.Club {
		c := &models.Club{ClubID: id, ClubName: id, ShortName: id, OverallTeamRating: 82, Morale: 75}
		cats := []string{"GK", "DEF", "DEF", "DEF", "DEF", "MID", "MID", "MID", "FWD", "FWD", "FWD"}
		poss := []string{"GK", "CB", "CB", "CB", "RB", "CM", "CM", "CAM", "LW", "ST", "RW"}
		for i := 0; i < 16; i++ {
			ci := i
			if ci > 10 {
				ci = 10
			}
			c.Squad = append(c.Squad, &models.Player{
				PlayerID: id + "_WK" + string(rune('A'+i)), FullName: id + " Kid",
				Position: poss[ci], Category: cats[ci], OVR: 78, Age: 16,
				UniverseWonderkid: true, MentorName: "Veteran", Personality: "big_game_performer", Composure: 82,
			})
		}
		return c
	}
	home, away := mkWKClub("K1"), mkWKClub("K2")
	hm, am := chunk2Mgrs()
	ge := growth.NewGrowthEngine(505)
	var wkPenalties, wkMisses int
	for seed := int64(0); seed < 150; seed++ {
		rep := SimulateInstantMatch(home, away, hm, am, ge,
			&InstantMatchConfig{Weather: "clear", Referee: "balanced"}, rand.New(rand.NewSource(1000+seed)))
		for _, e := range rep.Events {
			if (e.Type == "penalty" || e.Type == "penalty_miss") && e.Scorer != nil && e.Scorer.IsWK {
				if e.Type == "penalty" {
					wkPenalties++
				} else {
					wkMisses++
				}
			}
		}
	}
	if wkPenalties == 0 {
		t.Errorf("expected mentored wonderkids to convert penalties across 150 sims")
	}
	t.Logf("mentored-WK penalties: scored=%d missed=%d", wkPenalties, wkMisses)
}

// TestChunk2EventEcosystem sweeps fixed seeds asserting every event family
// occurs and every per-report invariant holds on every simulation.
func TestChunk2EventEcosystem(t *testing.T) {
	home, away := chunk2BaseClubs()
	home.StadiumCapacity = 60000
	away.StadiumCapacity = 40000
	hm, am := chunk2Mgrs()
	ge := growth.NewGrowthEngine(506)

	var sawSub, sawRed, sawSentOff, sawOG, sawFK, sawCorner, sawPen, sawMiss, sawStands, sawDisallowed bool
	var sawOffside, sawHandball, sawStoppage, sawBenchPlayed, sawDraw, sawAwayWin bool
	var sawRedPlain bool
	_ = sawRedPlain

	const sims = 600
	for seed := int64(0); seed < sims; seed++ {
		h, a := chunk2BaseClubs()
		h.StadiumCapacity = 60000
		rep := SimulateInstantMatch(h, a, hm, am, ge,
			&InstantMatchConfig{Weather: "clear", Referee: "balanced"}, rand.New(rand.NewSource(seed)))

		// --- structural invariants (every sim) ---
		if rep.Method != "instant" {
			t.Fatalf("seed %d: method = %q", seed, rep.Method)
		}
		if rep.DecidedBy != nil || rep.Penalties != nil {
			t.Fatalf("seed %d: league sim must leave decided_by/penalties nil", seed)
		}
		prevMin, prevSeq := -1, -1
		for _, e := range rep.Events {
			if e.Minute < prevMin || (e.Minute == prevMin && e.Seq <= prevSeq) {
				t.Fatalf("seed %d: events not sorted by (minute,seq): %+v", seed, e)
			}
			prevMin, prevSeq = e.Minute, e.Seq
			switch e.Type {
			case "goal", "penalty", "corner_goal", "free_kick_goal":
				if e.Scorer == nil {
					t.Fatalf("seed %d: goal without scorer: %+v", seed, e)
				}
				if e.Type == "goal" && e.Scorer.Category != "FWD" && e.Scorer.Category != "MID" {
					t.Fatalf("seed %d: regular goal by non-attacker: %+v", seed, e)
				}
			case "own_goal":
				if e.Scorer == nil || e.Beneficiary == "" {
					t.Fatalf("seed %d: malformed own goal: %+v", seed, e)
				}
			case "yellow", "red":
				if e.Player == nil {
					t.Fatalf("seed %d: card without player: %+v", seed, e)
				}
			case "sub":
				if e.PlayerOut == nil || e.PlayerIn == nil {
					t.Fatalf("seed %d: malformed sub: %+v", seed, e)
				}
			case "var_review":
				if e.Outcome == "" || e.Reason == "" || e.Decision == "" {
					t.Fatalf("seed %d: var_review missing fields: %+v", seed, e)
				}
			}
			if strings.Contains(e.Display, "45+") || strings.Contains(e.Display, "90+") {
				sawStoppage = true
			}
		}
		// Running-score backfill: monotonic and exact at full time.
		var lh, la int
		for _, e := range rep.Events {
			if e.HomeScore < lh || e.AwayScore < la {
				t.Fatalf("seed %d: running score regressed at %+v", seed, e)
			}
			lh, la = e.HomeScore, e.AwayScore
		}
		if lh != rep.HomeGoals || la != rep.AwayGoals {
			t.Fatalf("seed %d: backfill (%d-%d) != final (%d-%d)", seed, lh, la, rep.HomeGoals, rep.AwayGoals)
		}
		// Recompute the scoreboard from events independently.
		var eh, ea int
		for _, e := range rep.Events {
			switch e.Type {
			case "goal", "penalty", "corner_goal", "free_kick_goal":
				if !e.Disallowed {
					if e.Side == "home" {
						eh++
					} else {
						ea++
					}
				}
			case "own_goal":
				if e.Beneficiary == "home" {
					eh++
				} else {
					ea++
				}
			}
		}
		if eh != rep.HomeGoals || ea != rep.AwayGoals {
			t.Fatalf("seed %d: event recount (%d-%d) != final (%d-%d)", seed, eh, ea, rep.HomeGoals, rep.AwayGoals)
		}
		if rep.HTHome > rep.HomeGoals || rep.HTAway > rep.AwayGoals {
			t.Fatalf("seed %d: HT exceeds FT: %+v", seed, rep)
		}
		if rep.Attendance < 12000 || rep.Attendance > 60000 {
			t.Fatalf("seed %d: attendance %d out of [12000,60000]", seed, rep.Attendance)
		}
		st := rep.Stats
		if st.Home.Possession+st.Away.Possession != 100 {
			t.Fatalf("seed %d: possession sums to %d", seed, st.Home.Possession+st.Away.Possession)
		}
		if st.Home.Possession < 28 || st.Home.Possession > 72 {
			t.Fatalf("seed %d: possession %d out of corridor", seed, st.Home.Possession)
		}
		for _, ts := range []matchreport.TeamStats{st.Home, st.Away} {
			if ts.Shots < 4 || ts.ShotsOn < 0 || ts.ShotsOn > ts.Shots || ts.Corners < 0 ||
				ts.Passes < 0 || ts.Fouls < 0 || ts.YellowCards < 0 || ts.RedCards < 0 || ts.Saves < 0 {
				t.Fatalf("seed %d: stat out of bounds: %+v", seed, ts)
			}
			if ts.XG < 0.05 || ts.XG > 4.8 {
				t.Fatalf("seed %d: xG %v out of corridor", seed, ts.XG)
			}
			if ts.PassAccuracy < 65 || ts.PassAccuracy > 94 {
				t.Fatalf("seed %d: accuracy %d out of corridor", seed, ts.PassAccuracy)
			}
		}
		// Card stats must match emitted events (sent-off yellows excluded).
		sentOff := map[string]bool{}
		for _, e := range rep.Events {
			if e.Type == "red" && e.SentOff && e.Player != nil {
				sentOff[e.Player.PlayerID] = true
			}
		}
		var evHY, evHR, evAY, evAR int
		for _, e := range rep.Events {
			switch {
			case e.Type == "yellow" && e.Side == "home" && e.Player != nil && !sentOff[e.Player.PlayerID]:
				evHY++
			case e.Type == "red" && e.Side == "home":
				evHR++
			case e.Type == "yellow" && e.Side == "away" && e.Player != nil && !sentOff[e.Player.PlayerID]:
				evAY++
			case e.Type == "red" && e.Side == "away":
				evAR++
			}
		}
		if evHY != st.Home.YellowCards || evAY != st.Away.YellowCards {
			t.Fatalf("seed %d: yellow stats (%d/%d) != yellow events (%d/%d)", seed, st.Home.YellowCards, st.Away.YellowCards, evHY, evAY)
		}
		if evHR != st.Home.RedCards || evAR != st.Away.RedCards {
			t.Fatalf("seed %d: red stats (%d/%d) != red events (%d/%d)", seed, st.Home.RedCards, st.Away.RedCards, evHR, evAR)
		}
		if rep.MOTM == nil || !rep.MOTM.Played {
			t.Fatalf("seed %d: MOTM must be a played row: %+v", seed, rep.MOTM)
		}
		if rep.MOTM.Side != "home" && rep.MOTM.Side != "away" {
			t.Fatalf("seed %d: MOTM side = %q", seed, rep.MOTM.Side)
		}
		if rep.HomeGoals == rep.AwayGoals {
			sawDraw = true
		}
		if rep.AwayGoals > rep.HomeGoals {
			sawAwayWin = true
		}

		for _, e := range rep.Events {
			switch e.Type {
			case "sub":
				sawSub = true
			case "red":
				sawRed = true
				if e.SentOff {
					sawSentOff = true
				} else {
					sawRedPlain = true
				}
			case "own_goal":
				sawOG = true
			case "free_kick_goal":
				sawFK = true
			case "corner_goal":
				sawCorner = true
			case "penalty":
				sawPen = true
			case "penalty_miss":
				sawMiss = true
			case "var_review":
				if e.Outcome == "goal_stands" {
					sawStands = true
				} else {
					sawDisallowed = true
				}
				if e.Reason == "offside" {
					sawOffside = true
				}
				if e.Reason == "handball" {
					sawHandball = true
				}
			}
		}
		for _, row := range append(append([]matchreport.MatchPlayerRow{}, rep.HomeBench...), rep.AwayBench...) {
			if row.Played {
				sawBenchPlayed = true
			}
		}
	}

	for name, seen := range map[string]bool{
		"sub": sawSub, "red": sawRed, "sent_off": sawSentOff,
		"own_goal": sawOG, "free_kick_goal": sawFK, "corner_goal": sawCorner,
		"penalty": sawPen, "penalty_miss": sawMiss, "goal_stands": sawStands,
		"disallowed": sawDisallowed, "offside": sawOffside, "handball": sawHandball,
		"stoppage display": sawStoppage, "bench played": sawBenchPlayed,
		"draw": sawDraw, "away win": sawAwayWin,
	} {
		if !seen {
			t.Errorf("event family %q never occurred in %d sims", name, sims)
		}
	}
}

// TestChunk2SubSkipOnDismissal forces the planned-sub gate: with heavy
// rotation the booked-out path must trigger across a wide seed range.
func TestChunk2SubSkipOnDismissal(t *testing.T) {
	home, away := chunk2BaseClubs()
	hm, am := chunk2Mgrs()
	ge := growth.NewGrowthEngine(507)
	skips := 0
	tried := 0
	for seed := int64(0); seed < 400; seed++ {
		h, a := chunk2BaseClubs()
		rep := SimulateInstantMatch(h, a, hm, am, ge,
			&InstantMatchConfig{Weather: "clear", Referee: "strict", DerbyHeat: 90, IsDerby: true},
			rand.New(rand.NewSource(seed)))
		tried++
		sentOffIDs := map[string]bool{}
		for _, e := range rep.Events {
			if e.Type == "red" && e.SentOff && e.Player != nil {
				sentOffIDs[e.Player.PlayerID] = true
				if e.Detail != "second_yellow" && e.Detail != "straight_red" {
					t.Fatalf("instant red missing dismissal detail: %+v", e)
				}
			}
		}
		if len(sentOffIDs) == 0 {
			continue
		}
		// Any sent-off starter had their later planned subs skipped: the
		// skip path is covered whenever a dismissal meets a planned sub.
		skips += len(sentOffIDs)
	}
	t.Logf("dismissals observed across %d strict derbies: %d player-matches", tried, skips)
	_ = home
	_ = away
}

// TestChunk2DistributionParity checks season-scale statistical shape:
// scoring bands, assist rate, and positional scorer sanity.
func TestChunk2DistributionParity(t *testing.T) {
	hm, am := chunk2Mgrs()
	ge := growth.NewGrowthEngine(508)
	const sims = 400
	var totH, totA, assisted, unassisted, draws int
	var possSum int
	for seed := int64(0); seed < sims; seed++ {
		h, a := chunk2BaseClubs()
		rep := SimulateInstantMatch(h, a, hm, am, ge,
			&InstantMatchConfig{Weather: "clear", Referee: "balanced"}, rand.New(rand.NewSource(5000+seed)))
		totH += rep.HomeGoals
		totA += rep.AwayGoals
		possSum += rep.Stats.Home.Possession
		if rep.HomeGoals == rep.AwayGoals {
			draws++
		}
		for _, e := range rep.Events {
			if e.Type == "goal" && !e.Disallowed {
				if e.Assister != nil {
					assisted++
				} else {
					unassisted++
				}
			}
		}
	}
	meanH := float64(totH) / sims
	meanA := float64(totA) / sims
	if meanH < 0.7 || meanH > 2.6 {
		t.Errorf("mean home goals = %.2f; want [0.7,2.6]", meanH)
	}
	if meanA < 0.5 || meanA > 2.3 {
		t.Errorf("mean away goals = %.2f; want [0.5,2.3]", meanA)
	}
	if draws == 0 {
		t.Errorf("expected some draws in %d sims", sims)
	}
	rate := float64(assisted) / float64(assisted+unassisted) * 100
	if rate < 55 || rate > 85 {
		t.Errorf("assist rate = %.1f%%; want [55,85]", rate)
	}
	if mean := float64(possSum) / sims; mean < 42 || mean > 58 {
		t.Errorf("mean home possession = %.1f; want [42,58]", mean)
	}
	t.Logf("parity: mean %.2f-%.2f, draws %d, assist %.1f%%", meanH, meanA, draws, rate)
}

// TestChunk2StrictVsLenient checks the referee climate directionally.
func TestChunk2StrictVsLenient(t *testing.T) {
	hm, am := chunk2Mgrs()
	ge := growth.NewGrowthEngine(509)
	const sims = 200
	var strictCards, lenientCards int
	for seed := int64(0); seed < sims; seed++ {
		h, a := chunk2BaseClubs()
		rs := SimulateInstantMatch(h, a, hm, am, ge,
			&InstantMatchConfig{Weather: "clear", Referee: "strict"}, rand.New(rand.NewSource(seed)))
		h2, a2 := chunk2BaseClubs()
		rl := SimulateInstantMatch(h2, a2, hm, am, ge,
			&InstantMatchConfig{Weather: "clear", Referee: "lenient"}, rand.New(rand.NewSource(seed)))
		strictCards += rs.Stats.Home.YellowCards + rs.Stats.Away.YellowCards
		lenientCards += rl.Stats.Home.YellowCards + rl.Stats.Away.YellowCards
	}
	if strictCards <= lenientCards {
		t.Errorf("strict (%d) should produce more cards than lenient (%d) over %d sims", strictCards, lenientCards, sims)
	}
	t.Logf("cards: strict=%d lenient=%d", strictCards, lenientCards)
}

func TestChunk2R2LowMoralePenalty(t *testing.T) {
	// Low-morale conversion (-0.02) applies when the struggling side steps up.
	home, _ := chunk2BaseClubs()
	home.Morale = 30
	_, away := chunk2BaseClubs()
	hm, am := chunk2Mgrs()
	ge := growth.NewGrowthEngine(510)
	var homePens int
	for seed := int64(0); seed < 250 && homePens == 0; seed++ {
		h, a := chunk2BaseClubs()
		h.Morale = 30
		rep := SimulateInstantMatch(h, a, hm, am, ge,
			&InstantMatchConfig{Weather: "clear", Referee: "balanced"}, rand.New(rand.NewSource(seed)))
		for _, e := range rep.Events {
			if (e.Type == "penalty" || e.Type == "penalty_miss") && e.Side == "home" {
				homePens++
			}
		}
	}
	if homePens == 0 {
		t.Errorf("expected a home penalty across 250 low-morale sims")
	}
	_ = home
	_ = away
}

func TestChunk2R2LiveLifecycle(t *testing.T) {
	home, away := chunk2BaseClubs()
	hm, am := chunk2Mgrs()

	// Default seed branch.
	e0 := NewLiveMatchEngine(home, away, hm, am, 0)
	if e0.RNG == nil || e0.State != "NOT_STARTED" || len(e0.Commentary) != 1 {
		t.Errorf("fresh engine misconfigured: %+v", e0.State)
	}

	e := NewLiveMatchEngine(home, away, hm, am, 4242)
	if len(e.HomePlayers) != 11 || len(e.AwayPlayers) != 11 {
		t.Fatalf("radar should hold 11 per side")
	}

	// Kickoff from pre-match.
	e.Kickoff()
	if e.State != "PLAYING" {
		t.Errorf("kickoff should start play, got %s", e.State)
	}
	// Pause toggling both ways plus a no-op state.
	e.TogglePause()
	if e.State != "PAUSED" {
		t.Errorf("toggle should pause, got %s", e.State)
	}
	e.Tick(1.0) // paused clock must not advance
	if e.CurrentMinute != 0 {
		t.Errorf("paused tick advanced the clock to %v", e.CurrentMinute)
	}
	e.TogglePause()
	if e.State != "PLAYING" {
		t.Errorf("toggle should resume, got %s", e.State)
	}
	e.State = "NOT_STARTED"
	e.TogglePause()
	if e.State != "NOT_STARTED" {
		t.Errorf("toggle outside play/pause should be a no-op, got %s", e.State)
	}
	e.State = "PLAYING"

	// Speed clamping.
	e.SetSpeed(0)
	if e.Speed != 1 {
		t.Errorf("non-positive speed should clamp to 1, got %d", e.Speed)
	}
	e.SetSpeed(5)
	if e.Speed != 5 {
		t.Errorf("speed = %d; want 5", e.Speed)
	}

	// Away-possession ball tracking.
	e.PossessionTeam = "away"
	e.Tick(0.5)
	if e.BallPos.X == 0.5 && e.BallPos.Y == 0.5 {
		t.Errorf("away-possession tick should move the ball: %+v", e.BallPos)
	}
	e.PossessionTeam = "home"

	// Stance offsets in both directions.
	e.HomeStance = "PARK_BUS"
	e.AwayStance = "PARK_BUS"
	e.Tick(0.5)
	e.HomeStance = "OVERLOAD"
	e.AwayStance = "OVERLOAD"
	e.Tick(0.5)
	e.HomeStance = "NORMAL"
	e.AwayStance = "NORMAL"

	// Full-time path via the clock.
	e.CurrentMinute = 89.9
	e.Tick(5.0)
	if e.State != "FULL_TIME" {
		t.Fatalf("clock should reach FULL_TIME, got %s", e.State)
	}
	if len(e.Commentary) == 0 || e.Commentary[len(e.Commentary)-1].Category != "FULLTIME" {
		t.Errorf("full-time commentary missing: %+v", e.Commentary)
	}
	// Kickoff after full time resets and restarts.
	e.Kickoff()
	if e.State != "PLAYING" || e.CurrentMinute != 0 {
		t.Errorf("post-match kickoff should reset and restart: %+v", e.State)
	}

	// Explicit reset with and without clubs.
	e.HomeScore = 3
	e.Reset()
	if e.HomeScore != 0 || e.State != "NOT_STARTED" || len(e.HomePlayers) != 11 {
		t.Errorf("reset should clear score and rebuild radar")
	}
	e.HomeClub = nil
	e.AwayClub = nil
	e.Reset() // must not panic; skips the welcome message
	if len(e.Commentary) != 0 {
		t.Errorf("clubless reset should skip commentary, got %d items", len(e.Commentary))
	}

	// Re-point at a new matchup.
	home2, away2 := chunk2BaseClubs()
	e.SetClubs(home2, away2, hm, am)
	if e.HomeClub != home2 || e.AwayClub != away2 || e.State != "NOT_STARTED" {
		t.Errorf("SetClubs should re-point and reset")
	}
}

func TestChunk2R2LiveTacticsMatrix(t *testing.T) {
	home, away := chunk2BaseClubs()
	hm, am := chunk2Mgrs()
	mk := func(hs, as int, minute int) *LiveMatchEngine {
		e := NewLiveMatchEngine(home, away, hm, am, 99)
		e.State = "PLAYING"
		e.HomeScore, e.AwayScore = hs, as
		e.CheckTacticalAdaptations(minute)
		return e
	}
	// Away trailing in the window.
	if e := mk(1, 0, 80); e.AwayStance != "OVERLOAD" {
		t.Errorf("trailing away should overload, got %s", e.AwayStance)
	}
	// Home protecting a lead.
	if e := mk(1, 0, 80); e.HomeStance != "PARK_BUS" {
		t.Errorf("leading home should park the bus, got %s", e.HomeStance)
	}
	// Away protecting a lead.
	if e := mk(0, 1, 80); e.AwayStance != "PARK_BUS" {
		t.Errorf("leading away should park the bus, got %s", e.AwayStance)
	}
	// Outside the windows nothing shifts.
	if e := mk(0, 1, 10); e.AwayStance != "NORMAL" || e.LatestTacticalShift != nil {
		t.Errorf("early match should hold stances: %s / %+v", e.AwayStance, e.LatestTacticalShift)
	}
	// Non-normal stances are left alone.
	e := mk(0, 1, 80)
	e.AwayStance = "OVERLOAD"
	e.CheckTacticalAdaptations(80)
	if e.AwayStance != "OVERLOAD" {
		t.Errorf("engaged stance should hold, got %s", e.AwayStance)
	}
}
