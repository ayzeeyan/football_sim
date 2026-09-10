package matchengine

import (
	"strings"
	"testing"

	"football_sim/pkg/growth"
	"football_sim/pkg/managers"
	"football_sim/pkg/matchreport"
	"football_sim/pkg/models"
)

// Chunk 3 coverage: headless live phase machine (advance/resolve/book/subs,
// AI tactics, fast-forward, possession, payload) ported from match_engine.py.

func chunk3LiveEngine(seed int64) *LiveMatchEngine {
	home, away := createTestClubs()
	hm := &managers.ManagerProfile{ClubID: home.ClubID, Name: "Home Gaffer", Style: "possession"}
	am := &managers.ManagerProfile{ClubID: away.ClubID, Name: "Away Gaffer", Style: "high_press"}
	e := NewLiveMatchEngine(home, away, hm, am, seed)
	e.ResetMatch()
	e.StartKickoff()
	return e
}

func TestChunk3LiveResetAndKickoff(t *testing.T) {
	home, away := createTestClubs()
	hm, am := chunk2Mgrs()
	e := NewLiveMatchEngine(home, away, hm, am, 31337)
	e.ResetMatch()
	if e.State != "NOT_STARTED" || e.CurrentMinute != 0 || e.Phase != "BUILDUP" {
		t.Errorf("reset state wrong: %+v", e.State)
	}
	if e.HomePossessionTicks != 1 || e.AwayPossessionTicks != 1 {
		t.Errorf("possession ticks not reset: %d/%d", e.HomePossessionTicks, e.AwayPossessionTicks)
	}
	if len(e.HomeStarters) != 11 || len(e.HomeKickoffXI) != 11 || len(e.HomeBench) == 0 {
		t.Errorf("rosters not snapshotted: %d/%d/%d", len(e.HomeStarters), len(e.HomeKickoffXI), len(e.HomeBench))
	}
	if len(e.PlannedSubs) == 0 {
		t.Errorf("planned subs not scheduled")
	}
	if len(e.Commentary) != 1 || !strings.HasPrefix(e.Commentary[0].Text, "Welcome to ") {
		t.Errorf("welcome commentary wrong: %+v", e.Commentary)
	}
	if e.InstanceID != 1 {
		t.Errorf("instance id = %d; want 1", e.InstanceID)
	}

	// StartKickoff gates on state.
	e.StartKickoff()
	if e.State != "PLAYING" {
		t.Errorf("kickoff should start play, got %s", e.State)
	}
	if len(e.Commentary) != 2 || e.Commentary[1].Text != "The referee blows the whistle and we are underway!" {
		t.Errorf("underway call wrong: %+v", e.Commentary)
	}
	e.StartKickoff() // already playing: no-op
	if len(e.Commentary) != 2 {
		t.Errorf("repeat kickoff should be a no-op, got %d items", len(e.Commentary))
	}
	// Paused matches resume without a second underway call.
	e.CurrentMinute = 10
	e.State = "PAUSED"
	e.StartKickoff()
	if e.State != "PLAYING" || len(e.Commentary) != 2 {
		t.Errorf("paused resume wrong: %s / %d items", e.State, len(e.Commentary))
	}
	// Nil-RNG fallback path.
	e.RNG = nil
	if e.rng() == nil {
		t.Errorf("rng fallback should install a source")
	}
}

func TestChunk3ContextualRadarUsesSelectedXI(t *testing.T) {
	home, away := createTestClubs()
	examKid := home.Squad[9]
	examKid.Age = 16
	examKid.Education = "high_school"
	e := NewLiveMatchEngine(home, away, nil, nil, 7331)
	e.SetFixtureContext("super-league", 12)

	for _, p := range e.HomeStarters {
		if p.PlayerID == examKid.PlayerID {
			t.Fatalf("exam player appeared in contextual live starters")
		}
	}
	if len(e.HomePlayers) != len(e.HomeStarters) {
		t.Fatalf("home radar count %d does not match selected XI count %d", len(e.HomePlayers), len(e.HomeStarters))
	}
	if len(e.AwayPlayers) != len(e.AwayStarters) {
		t.Fatalf("away radar count %d does not match selected XI count %d", len(e.AwayPlayers), len(e.AwayStarters))
	}
	for i, p := range e.HomePlayers {
		if p.PlayerID == "" || p.PlayerID != e.HomeStarters[i].PlayerID {
			t.Fatalf("home radar actor %d does not match selected XI: %+v", i, p)
		}
	}
	for i, p := range e.AwayPlayers {
		if p.PlayerID == "" || p.PlayerID != e.AwayStarters[i].PlayerID {
			t.Fatalf("away radar actor %d does not match selected XI: %+v", i, p)
		}
	}

	e.StartKickoff()
	e.InstantSimulate()
	for _, shot := range e.LiveShots {
		if shot.Shooter.PlayerID == examKid.PlayerID {
			t.Fatalf("exam player appeared in live shot map: %+v", shot)
		}
	}
	for _, ev := range e.Events {
		for _, p := range []*matchreport.MiniPlayer{ev.Scorer, ev.Assister, ev.Player, ev.PlayerOut, ev.PlayerIn} {
			if p != nil && p.PlayerID == examKid.PlayerID {
				t.Fatalf("exam player appeared in live event: %+v", ev)
			}
		}
	}
}

func TestChunk3RadarHandlesShortAndEmptyXI(t *testing.T) {
	makeClub := func(id string, p *models.Player) *models.Club {
		return &models.Club{ClubID: id, ClubName: id, ShortName: id, Squad: []*models.Player{p}}
	}
	short := makeClub("SHORT", &models.Player{PlayerID: "SHORT_P", Category: "FWD", OVR: 70})
	empty := makeClub("EMPTY", &models.Player{PlayerID: "EMPTY_P", Category: "FWD", OVR: 70, UniverseWonderkid: true, Age: 16, Education: "high_school"})
	for _, tc := range []struct {
		name  string
		club  *models.Club
		count int
	}{
		{name: "short", club: short, count: 1},
		{name: "empty", club: empty, count: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, opponent := createTestClubs()
			e := NewLiveMatchEngine(tc.club, opponent, nil, nil, 7442)
			e.SetFixtureContext("super-league", 12)
			if len(e.HomePlayers) != tc.count {
				t.Fatalf("radar count = %d; want %d", len(e.HomePlayers), tc.count)
			}
			for _, p := range e.HomePlayers {
				if p.PlayerID == "" {
					t.Fatalf("blank radar actor: %+v", p)
				}
			}
			e.StartKickoff()
			for i := 0; i < 3; i++ {
				e.Tick(1)
			}
			_ = e.BuildLivePayload()
		})
	}
}

func TestChunk3LiveUpdateMachine(t *testing.T) {
	e := chunk3LiveEngine(1001)

	// Non-playing update is a no-op.
	e.State = "PAUSED"
	e.Update(1.0)
	if e.CurrentMinute != 0 {
		t.Errorf("paused update advanced the clock")
	}
	e.State = "PLAYING"

	// GOAL_PAUSE countdown resumes play and rebuilds.
	e.State = "GOAL_PAUSE"
	e.BannerTimer = 0.5
	e.Banner = "GOAL"
	e.Update(1.0)
	if e.State != "PLAYING" || e.Banner != "" || e.Phase != "BUILDUP" {
		t.Errorf("goal pause should resume: %s/%q/%s", e.State, e.Banner, e.Phase)
	}
	e.State = "GOAL_PAUSE"
	e.BannerTimer = 5.0
	e.Update(1.0)
	if e.State != "GOAL_PAUSE" || e.BannerTimer != 4.0 {
		t.Errorf("goal pause should count down: %s/%v", e.State, e.BannerTimer)
	}
	e.State = "PLAYING"

	// Speed-999 fast path.
	e.Speed = 999
	e.Update(0.016)
	if e.State != "FULL_TIME" || e.CurrentMinute != 90 {
		t.Errorf("999x should fast-forward to full time: %s/%v", e.State, e.CurrentMinute)
	}
	foundInstant := false
	for _, c := range e.Commentary {
		if strings.Contains(c.Text, "Instant Sim") {
			foundInstant = true
		}
	}
	if !foundInstant {
		t.Errorf("fast-forward should log the instant-sim whistle")
	}
}

func TestChunk3LiveFullMatchFlow(t *testing.T) {
	e := chunk3LiveEngine(2002)
	ticks := 0
	for e.State != "FULL_TIME" && ticks < 500 {
		e.Update(1.0)
		if e.State == "HALF_TIME" {
			e.ResumeHalfTime("home", "", "", "")
		}
		ticks++
	}
	if e.State != "FULL_TIME" {
		t.Fatalf("sim never reached full time in %d ticks", ticks)
	}
	if e.CurrentMinute != 90 {
		t.Errorf("clock = %v; want 90", e.CurrentMinute)
	}
	if e.HomePossessionTicks+e.AwayPossessionTicks <= 2 {
		t.Errorf("possession ticks never accumulated")
	}
	if len(e.Commentary) == 0 || e.Commentary[len(e.Commentary)-1].Category != "FULLTIME" {
		t.Errorf("missing full-time call")
	}
	t.Logf("live flow: %d-%d in %d ticks, %d events, %d shots", e.HomeScore, e.AwayScore, ticks, len(e.Events), len(e.LiveShots))

	// Payload + assembly integration over a real live match.
	payload := e.BuildLivePayload()
	if payload.HomeGoals != e.HomeScore || payload.AwayGoals != e.AwayScore {
		t.Errorf("payload scores %d-%d vs engine %d-%d", payload.HomeGoals, payload.AwayGoals, e.HomeScore, e.AwayScore)
	}
	report := matchreport.AssembleReport(payload, "live", e.RNG)
	if report.Method != "live" || report.MOTM == nil {
		t.Errorf("live assembly wrong: %+v", report.MOTM)
	}
	if report.Stats.Home.Shots != e.HomeShots || report.Stats.Away.Corners != e.AwayCorners {
		t.Errorf("payload counters wrong: %+v vs %d/%d", report.Stats, e.HomeShots, e.AwayCorners)
	}
}

func TestChunk3LivePhaseCoverage(t *testing.T) {
	// Exercise every phase branch across seeded iterations.
	var sawMid, sawAttack, sawShot, sawTurnover, sawCorner, sawCreatorNote bool
	for seed := int64(0); seed < 120; seed++ {
		e := chunk3LiveEngine(seed)
		e.Phase = "BUILDUP"
		e.AdvancePhase()
		if e.Phase == "MIDFIELD" {
			sawMid = true
		} else if e.Phase == "BUILDUP" && e.PossessionTeam == "away" {
			sawTurnover = true
		}
		e.Phase = "MIDFIELD"
		before := e.Phase
		e.AdvancePhase()
		if e.Phase == "ATTACKING" {
			sawAttack = true
		}
		_ = before
		e.Phase = "ATTACKING"
		e.AdvancePhase()
		if e.Phase == "SHOT" {
			sawShot = true
		}
		if e.HomeCorners+e.AwayCorners > 0 {
			sawCorner = true
		}
		for _, c := range e.Commentary {
			if strings.Contains(c.Text, "penetrative pass") {
				sawCreatorNote = true
			}
			if strings.Contains(c.Text, "corner kick.") {
				sawCorner = true
			}
		}
		e.Phase = "SHOT"
		e.AdvancePhase()
		if e.Phase != "BUILDUP" {
			t.Fatalf("SHOT must fall back to BUILDUP, got %s", e.Phase)
		}
	}
	for name, seen := range map[string]bool{"MIDFIELD": sawMid, "turnover": sawTurnover, "ATTACKING": sawAttack, "SHOT": sawShot, "corner": sawCorner, "creator note": sawCreatorNote} {
		if !seen {
			t.Errorf("phase outcome %q never observed in 120 seeds", name)
		}
	}
	// Stance-modified deltas run without error.
	e := chunk3LiveEngine(7)
	e.HomeStance = "OVERLOAD"
	e.AwayStance = "PARK_BUS"
	e.Phase = "BUILDUP"
	e.AdvancePhase()
	e.PossessionTeam = "away"
	e.HomeStance = "PARK_BUS"
	e.AwayStance = "OVERLOAD"
	e.Phase = "MIDFIELD"
	e.AdvancePhase()
	// Empty creator pool turns over.
	e.HomeStarters = nil
	e.AwayStarters = nil
	e.Bookings = map[string]int{}
	e.Phase = "MIDFIELD"
	e.PossessionTeam = "home"
	e.AdvancePhase()
}

func TestChunk3LiveResolveShotMatrix(t *testing.T) {
	var sawGoal, sawSave, sawMiss, sawOG, sawPenalty, sawBooking bool
	var sawWKGoal, sawMentorTag, sawVision, sawPlain bool
	for seed := int64(0); seed < 400 && !(sawGoal && sawSave && sawMiss && sawOG && sawPenalty && sawBooking); seed++ {
		e := chunk3LiveEngine(seed)
		for _, p := range e.HomeStarters {
			if p.UniverseWonderkid {
				p.MentorName = "Veteran"
			}
		}
		// Rotate morale/stance/score contexts to hit conv branches.
		switch seed % 4 {
		case 0:
			e.HomeClub.Morale = 95
			e.HomeScore, e.AwayScore = 1, 1
		case 1:
			e.HomeClub.Morale = 30
		case 2:
			e.HomeStance = "OVERLOAD"
			e.AwayStance = "PARK_BUS"
		}
		e.ResolveShot(e.HomeClub, e.AwayClub, e.HomeStarters, e.AwayStarters)
		for _, s := range e.LiveShots {
			switch s.Outcome {
			case "goal":
				sawGoal = true
			case "save":
				sawSave = true
			case "miss":
				sawMiss = true
			}
		}
		for _, ev := range e.Events {
			switch ev.Type {
			case "goal":
				sawGoal = true
			case "own_goal":
				sawOG = true
			case "penalty", "penalty_miss":
				sawPenalty = true
			case "yellow", "red":
				sawBooking = true
			}
		}
		for _, c := range e.Commentary {
			switch {
			case strings.HasPrefix(c.Text, "WONDERKID GOAL!"):
				sawWKGoal = true
			case strings.Contains(c.Text, "mentored by"):
				sawMentorTag = true
			case strings.HasPrefix(c.Text, "WONDERKID VISION!"):
				sawVision = true
			case strings.HasPrefix(c.Text, "GOAAAL!"):
				sawPlain = true
			}
		}
	}
	for name, seen := range map[string]bool{"goal": sawGoal, "save": sawSave, "miss": sawMiss, "own_goal": sawOG, "penalty path": sawPenalty, "booking": sawBooking} {
		if !seen {
			t.Errorf("shot outcome %q never observed in 400 shots", name)
		}
	}
	// Mentored wonderkid XI: every goal is a WK goal, so the mentor tag must fire.
	tagged := false
	for seed := int64(0); seed < 60 && !tagged; seed++ {
		home, away := createTestClubs()
		for _, p := range home.Squad {
			if p.Category == "FWD" || p.Category == "MID" {
				p.UniverseWonderkid = true
				p.MentorName = "Veteran"
			}
		}
		he := NewLiveMatchEngine(home, away, nil, nil, seed)
		he.ResetMatch()
		he.StartKickoff()
		for i := 0; i < 12; i++ {
			he.ResolveShot(he.HomeClub, he.AwayClub, he.HomeStarters, he.AwayStarters)
		}
		for _, c := range he.Commentary {
			if strings.Contains(c.Text, "mentored by") {
				tagged = true
			}
			if strings.HasPrefix(c.Text, "WONDERKID VISION!") {
				sawVision = true
			}
			if strings.HasPrefix(c.Text, "GOAAAL!") {
				sawPlain = true
			}
		}
	}
	if !tagged {
		t.Errorf("mentored wonderkid goal tag never observed")
	}
	for name, seen := range map[string]bool{"wonderkid goal": sawWKGoal, "mentor tag": tagged || sawMentorTag, "vision": sawVision, "plain": sawPlain} {
		if !seen {
			t.Errorf("commentary variant %q never observed", name)
		}
	}
	// Empty candidate pools are safe no-ops.
	e := chunk3LiveEngine(9)
	e.ResolveShot(e.HomeClub, e.AwayClub, nil, nil)
	e.ResolvePenalty(e.HomeClub, nil)
}

func TestChunk3LiveBookingMatrix(t *testing.T) {
	var sawYellow, sawStraightRed, sawSentOff bool
	for seed := int64(0); seed < 600 && !(sawYellow && sawStraightRed && sawSentOff); seed++ {
		e := chunk3LiveEngine(seed)
		// Seed a prior booking to open the second-yellow path.
		if seed%2 == 0 && len(e.AwayStarters) > 3 {
			e.Bookings[e.AwayStarters[3].PlayerID] = 1
		}
		if seed%3 == 0 {
			e.IsHighHeatDerby = true
		}
		before := len(e.Events)
		e.MaybeBookPlayer(e.AwayStarters)
		if len(e.Events) == before {
			continue
		}
		switch ev := e.Events[len(e.Events)-1]; {
		case ev.Type == "yellow":
			sawYellow = true
		case ev.Type == "red":
			if !ev.SentOff {
				t.Fatalf("red event must dismiss player: %+v", ev)
			}
			if ev.Seq == 0 {
				t.Fatalf("live red event missing seq: %+v", ev)
			}
			sawSentOff = true
			if ev.Detail == "straight_red" {
				sawStraightRed = true
			}
		}
	}
	for name, seen := range map[string]bool{"yellow": sawYellow, "straight red": sawStraightRed, "sent off": sawSentOff} {
		if !seen {
			t.Errorf("booking outcome %q never observed in 600 rolls", name)
		}
	}
	// Empty pools are safe no-ops.
	e := chunk3LiveEngine(11)
	e.MaybeBookPlayer(nil)
	e.MaybeBookPlayer([]*models.Player{chunk3GKOnly()})
}

func TestDismissedRadarStopsMovingWithTheShape(t *testing.T) {
	e := chunk3LiveEngine(21)
	e.State = "PLAYING"
	e.HomeStance = "OVERLOAD"
	if len(e.HomePlayers) < 3 {
		t.Fatal("expected radar actors")
	}
	sent := e.HomePlayers[2]
	e.Bookings[sent.PlayerID] = 2
	x0, y0 := e.HomePlayers[2].X, e.HomePlayers[2].Y
	live0 := e.HomePlayers[1]
	e.Tick(0.5)
	if e.HomePlayers[2].X != x0 || e.HomePlayers[2].Y != y0 {
		t.Fatalf("sent-off actor kept moving: (%v,%v) -> (%v,%v)", x0, y0, e.HomePlayers[2].X, e.HomePlayers[2].Y)
	}
	if e.HomePlayers[1].X == live0.X && e.HomePlayers[1].Y == live0.Y && e.HomePlayers[1].PlayerID == live0.PlayerID {
		t.Fatalf("on-pitch actors should still take the overload shape")
	}
}

func chunk3GKOnly() *models.Player {
	return &models.Player{PlayerID: "GKSOLO", FullName: "Solo Keeper", Position: "GK", Category: "GK", OVR: 80}
}

func TestChunk3LivePenaltyMatrix(t *testing.T) {
	var sawScored, sawMissed bool
	for seed := int64(0); seed < 120 && !(sawScored && sawMissed); seed++ {
		e := chunk3LiveEngine(seed)
		e.ResolvePenalty(e.HomeClub, e.HomeStarters)
		last := e.Events[len(e.Events)-1]
		if last.Type == "penalty" {
			sawScored = true
		} else if last.Type == "penalty_miss" {
			sawMissed = true
		} else {
			t.Fatalf("penalty must emit penalty/penalty_miss, got %q", last.Type)
		}
	}
	if !sawScored || !sawMissed {
		t.Errorf("penalty outcomes incomplete: scored=%v missed=%v", sawScored, sawMissed)
	}
	// Mentored wonderkid taker exercises the conversion caps.
	home, _ := createTestClubs()
	for _, p := range home.Squad {
		p.UniverseWonderkid = true
		p.MentorName = "Vet"
		p.Composure = 85
	}
	away, _ := createTestClubs()
	e := NewLiveMatchEngine(home, away, nil, nil, 77)
	e.ResetMatch()
	e.StartKickoff()
	e.ResolvePenalty(e.HomeClub, e.HomeStarters)
}

func TestChunk3LiveTurnoverAndSubs(t *testing.T) {
	e := chunk3LiveEngine(31)
	e.PossessionTeam = "home"
	e.Turnover("test")
	if e.PossessionTeam != "away" || e.Phase != "BUILDUP" {
		t.Errorf("turnover wrong: %s/%s", e.PossessionTeam, e.Phase)
	}
	if e.BallTarget.X != 0.75 {
		t.Errorf("away buildup target x = %v; want 0.75", e.BallTarget.X)
	}
	e.Turnover("test")
	if e.PossessionTeam != "home" || e.BallTarget.X != 0.25 {
		t.Errorf("turnover back wrong: %s/%v", e.PossessionTeam, e.BallTarget.X)
	}

	// Planned sub execution and skip paths.
	e.CurrentMinute = 90
	e.PlannedSubs = []PlannedSub{{Minute: 60, Out: e.HomeStarters[2], In: e.HomeBench[0], Side: "home"}}
	e.MaybeSubstitute()
	if !e.PlannedSubs[0].Done || e.SubstitutionsMade["home"] != 1 {
		t.Errorf("planned sub should execute: %+v", e.PlannedSubs[0])
	}
	if e.HomeStarters[2].PlayerID != e.HomeBench[0].PlayerID {
		t.Errorf("sub swap did not land")
	}
	// Dismissed out-player: skipped but marked done.
	e2 := chunk3LiveEngine(32)
	e2.CurrentMinute = 90
	e2.PlannedSubs = []PlannedSub{{Minute: 60, Out: e2.HomeStarters[4], In: e2.HomeBench[1], Side: "home"}}
	e2.Bookings[e2.HomeStarters[4].PlayerID] = 2
	e2.MaybeSubstitute()
	if !e2.PlannedSubs[0].Done || e2.SubstitutionsMade["home"] != 0 {
		t.Errorf("dismissed sub should skip: %+v", e2.PlannedSubs[0])
	}
	// Missing out-player: skipped but marked done.
	e3 := chunk3LiveEngine(33)
	e3.CurrentMinute = 90
	ghost := &models.Player{PlayerID: "GHOST", FullName: "Ghost"}
	e3.PlannedSubs = []PlannedSub{{Minute: 60, Out: ghost, In: e3.HomeBench[1], Side: "home"}}
	e3.MaybeSubstitute()
	if !e3.PlannedSubs[0].Done {
		t.Errorf("ghost sub should be marked done")
	}
	// Future sub waits.
	e4 := chunk3LiveEngine(34)
	e4.CurrentMinute = 10
	e4.PlannedSubs = []PlannedSub{{Minute: 60, Out: e4.HomeStarters[1], In: e4.HomeBench[0], Side: "home"}}
	e4.MaybeSubstitute()
	if e4.PlannedSubs[0].Done {
		t.Errorf("future sub must wait")
	}
	// OnPitch filters the dismissed.
	if got := e2.OnPitch(e2.HomeStarters); len(got) != 10 {
		t.Errorf("on-pitch = %d; want 10 after dismissal", len(got))
	}
}

func TestChunk3LiveAIDecisions(t *testing.T) {
	mk := func(minute, hs, as int) *LiveMatchEngine {
		e := chunk3LiveEngine(41)
		e.State = "PLAYING"
		e.CurrentMinute = float64(minute)
		e.HomeScore, e.AwayScore = hs, as
		return e
	}
	// Pre-58 gate.
	e := mk(30, 0, 2)
	e.EvaluateGameStateTactics()
	if e.HomeStance != "NORMAL" {
		t.Errorf("pre-58 must hold stances")
	}
	// Non-playing gate.
	e.State = "PAUSED"
	e.CurrentMinute = 80
	e.HomeScore, e.AwayScore = 0, 2
	e.EvaluateGameStateTactics()
	if e.HomeStance != "NORMAL" {
		t.Errorf("paused must hold stances")
	}
	// Trailing OVERLOAD with dugout sub (home).
	e = mk(75, 0, 1)
	e.ApplyAIManagerDecision("home", -1, 75)
	if e.HomeStance != "OVERLOAD" || e.LatestTacticalShift == nil || e.LatestTacticalShift.Label != "All-Out Overload" {
		t.Errorf("overload wrong: %s / %+v", e.HomeStance, e.LatestTacticalShift)
	}
	if e.SubstitutionsMade["home"] != 1 {
		t.Errorf("overload should sub a forward on: %d", e.SubstitutionsMade["home"])
	}
	// Already engaged: no double shift.
	e.ApplyAIManagerDecision("home", -1, 76)
	if e.SubstitutionsMade["home"] != 1 {
		t.Errorf("engaged stance must not re-fire")
	}
	// Leading PARK_BUS with lockdown sub (away).
	e = mk(80, 0, 1)
	e.ApplyAIManagerDecision("away", 1, 80)
	if e.AwayStance != "PARK_BUS" || e.LatestTacticalShift.Label != "Low-Block Lockdown" {
		t.Errorf("lockdown wrong: %s / %+v", e.AwayStance, e.LatestTacticalShift)
	}
	// Lead too big: no lockdown.
	e = mk(80, 0, 3)
	e.ApplyAIManagerDecision("away", 3, 80)
	if e.AwayStance != "NORMAL" {
		t.Errorf("3-goal lead should not park: %s", e.AwayStance)
	}
	// Card-risk hook with same-category bench cover (first lucky RNG wins).
	hooked := false
	for seed := int64(41); seed < 120 && !hooked; seed++ {
		he := chunk3LiveEngine(seed)
		he.State = "PLAYING"
		he.CurrentMinute = 65
		he.HomeScore, he.AwayScore = 1, 1
		var hbooked *models.Player
		for _, p := range he.HomeStarters {
			if p.Category == "DEF" {
				hbooked = p
				break
			}
		}
		he.Bookings[hbooked.PlayerID] = 1
		// Bench cover in the same category (fixtures carry none by default).
		he.HomeBench = append(he.HomeBench, &models.Player{PlayerID: "COVERDEF", FullName: "Cover Def", Position: "CB", Category: "DEF", OVR: 75})
		beforeHook := he.SubstitutionsMade["home"]
		he.ApplyAIManagerDecision("home", 0, 65)
		hooked = he.SubstitutionsMade["home"] == beforeHook+1
	}
	if !hooked {
		t.Errorf("booked defender should be hooked on some seed")
	}
	// Hook with no cover available.
	e = mk(65, 1, 1)
	var booked2 *models.Player
	for _, p := range e.HomeStarters {
		if p.Category == "DEF" {
			booked2 = p
			break
		}
	}
	e.Bookings[booked2.PlayerID] = 1
	e.HomeBench = []*models.Player{}
	before := e.SubstitutionsMade["home"]
	e.ApplyAIManagerDecision("home", 0, 65)
	if e.SubstitutionsMade["home"] != before {
		t.Errorf("hook without cover must not fire")
	}
	// Substitution cap blocks dugout changes (stance still shifts).
	e = mk(75, 0, 1)
	e.SubstitutionsMade = map[string]int{"home": 5, "away": 0}
	e.ApplyAIManagerDecision("home", -1, 75)
	if e.HomeStance != "OVERLOAD" || e.SubstitutionsMade["home"] != 5 {
		t.Errorf("capped dugout should shift but not sub")
	}
	// Direct sub with missing out-player is a safe no-op.
	e.ExecuteDirectSub("home", &models.Player{PlayerID: "GHOST"}, e.HomeBench[0], 70, "test")
	// Direct sub without reason renders cleanly.
	e.ExecuteDirectSub("away", e.AwayStarters[9], e.AwayBench[0], 70, "")
	// Manager-name fallbacks.
	if got := e.managerName("home"); got == "" {
		t.Errorf("manager name should fall back, got empty")
	}
	e.HomeManager = nil
	if got := e.managerName("home"); got != e.HomeClub.ShortName+" Manager" {
		t.Errorf("nil manager fallback wrong: %q", got)
	}
	e.HomeClub = nil
	if got := e.managerName("home"); got != "Manager" {
		t.Errorf("clubless fallback wrong: %q", got)
	}
	// Accessor coverage.
	if e.stance("home") != e.HomeStance || e.stance("away") != e.AwayStance {
		t.Errorf("stance accessor wrong")
	}
	e.setStance("away", "OVERLOAD")
	if e.AwayStance != "OVERLOAD" {
		t.Errorf("setStance failed")
	}
	if len(e.starters("home")) != 11 || len(e.bench("away")) == 0 {
		t.Errorf("roster accessors wrong")
	}
	if e.plannedInUse("nobody") {
		t.Errorf("plannedInUse false positive")
	}
	if got := maxOVRPick(nil, true); got != nil {
		t.Errorf("empty pick should be nil")
	}
	if got := maxOVRPick(e.AwayBench, false); got == nil {
		t.Errorf("plain pick should resolve")
	}
}

func TestChunk3LivePossessionAndPayload(t *testing.T) {
	e := chunk3LiveEngine(55)
	if got := e.HomePossessionPct(); got != 50 {
		t.Errorf("fresh ticks should read 50/50, got %d", got)
	}
	e.HomePossessionTicks, e.AwayPossessionTicks = 60, 40
	if got := e.HomePossessionPct(); got != 60 {
		t.Errorf("possession = %d; want 60", got)
	}
	if got := e.AwayPossessionPct(); got != 40 {
		t.Errorf("away possession = %d; want 40", got)
	}

	// Payload HT quirk: only goal/penalty (+OG) count, not corner/FK types.
	sc := matchreport.ToMiniPlayer(e.HomeStarters[9])
	e.Events = []matchreport.MatchEventItem{
		{Minute: 10, Seq: 1, Type: "corner_goal", Side: "home", Scorer: &sc},
		{Minute: 20, Seq: 2, Type: "goal", Side: "home", Scorer: &sc},
		{Minute: 30, Seq: 3, Type: "free_kick_goal", Side: "away", Scorer: &sc},
		{Minute: 50, Seq: 4, Type: "goal", Side: "away", Scorer: &sc},
		{Minute: 44, Seq: 5, Type: "own_goal", Side: "away", Beneficiary: "home", Scorer: &sc},
	}
	e.HomeScore, e.AwayScore = 3, 2
	payload := e.BuildLivePayload()
	if payload.HTHome != 2 || payload.HTAway != 0 {
		t.Errorf("live HT quirk wrong: %d-%d (corner/FK excluded)", payload.HTHome, payload.HTAway)
	}
	if payload.Referee == "" || payload.Weather != "clear" {
		t.Errorf("payload defaults wrong: %+v", payload)
	}
	if payload.Attendance < 12000 {
		t.Errorf("attendance floor wrong: %d", payload.Attendance)
	}
	// Live shots/touches flow into maps when present.
	e.LiveShots = []matchreport.ShotMapItem{
		{Minute: 20, Team: "home", Shooter: matchreport.ShotShooter{FullName: "X", Position: "ST", OVR: 80}, X: 0.9, Y: 0.5, XG: 0.4, Outcome: "goal"},
	}
	for i := 0; i < 10; i++ {
		e.LiveTouches["home"] = append(e.LiveTouches["home"], [2]float64{0.7, 0.5})
		e.LiveTouches["away"] = append(e.LiveTouches["away"], [2]float64{0.3, 0.5})
	}
	payload2 := e.BuildLivePayload()
	if len(payload2.ShotMap.Shots) != 1 || payload2.ShotMap.Shots[0].XG != 0.4 {
		t.Errorf("live shots should pass through: %+v", payload2.ShotMap.Shots)
	}
	if len(payload2.Heatmap.HomePoints) != 10 {
		t.Errorf("live touches should pass through: %d", len(payload2.Heatmap.HomePoints))
	}
	// Display/seq backfill on raw engine events.
	e.Events = []matchreport.MatchEventItem{{Minute: 5, Type: "goal", Side: "home", Scorer: &sc}}
	payload3 := e.BuildLivePayload()
	if payload3.Events[0].Display != "5'" {
		t.Errorf("display default wrong: %q", payload3.Events[0].Display)
	}
	// Zero-capacity club falls back to 50000.
	e.HomeClub.StadiumCapacity = 0
	_ = e.BuildLivePayload()
	_ = growth.NewGrowthEngine(1)
}

func TestChunk3R3ZeroValueEngine(t *testing.T) {
	// Zero-value engine: lazy map init must not panic.
	e := &LiveMatchEngine{}
	e.ResetMatch()
	if e.Bookings == nil || e.LiveTouches == nil || e.SubstitutionsMade == nil {
		t.Errorf("lazy maps not initialized")
	}
	// Populated maps are cleared (not reallocated) on re-reset.
	e.Bookings["X"] = 1
	e.LiveTouches["home"] = append(e.LiveTouches["home"], [2]float64{0.5, 0.5})
	e.SubstitutionsMade["home"] = 3
	e.ResetMatch()
	if len(e.Bookings) != 0 || len(e.LiveTouches["home"]) != 0 || e.SubstitutionsMade["home"] != 0 {
		t.Errorf("re-reset should clear maps")
	}
	// Untouched zero engine still turns over via lazy touch-map init.
	// (Zero possession "" flips to "home", mirroring Python semantics.)
	z0 := &LiveMatchEngine{}
	z0.Turnover("test")
	if z0.PossessionTeam != "home" || len(z0.LiveTouches["home"]) != 1 {
		t.Errorf("zero turnover wrong: %s / %v", z0.PossessionTeam, z0.LiveTouches)
	}
	e.Turnover("test")
	if e.PossessionTeam != "away" {
		t.Errorf("turnover should flip to away: %s", e.PossessionTeam)
	}
	if got := (&LiveMatchEngine{}).HomePossessionPct(); got != 50 {
		t.Errorf("zero possession = %d; want 50", got)
	}
	// Fresh zero engines (nil maps) so each sub path hits its nil-map guard.
	home, _ := createTestClubs()
	z := &LiveMatchEngine{HomeStarters: append([]*models.Player(nil), home.Squad[:11]...)}
	// Planned sub first so ExecuteSub hits the nil-map guard.
	z.PlannedSubs = []PlannedSub{{Minute: 0, Out: z.HomeStarters[2], In: home.Squad[13], Side: "home"}}
	z.ExecuteSub(0) // CurrentMinute 0 exercises the minute floor
	if z.SubstitutionsMade["home"] != 1 {
		t.Errorf("zero-engine planned sub not counted")
	}
	if z.Events[len(z.Events)-1].Minute != 1 {
		t.Errorf("sub minute should floor at 1: %+v", z.Events[len(z.Events)-1])
	}
	zd := &LiveMatchEngine{HomeStarters: append([]*models.Player(nil), home.Squad[:11]...)}
	out, in := zd.HomeStarters[1], home.Squad[12]
	zd.ExecuteDirectSub("home", out, in, 70, "test")
	if zd.SubstitutionsMade["home"] != 1 {
		t.Errorf("zero-engine direct sub not counted")
	}
}

func TestChunk3R3ShotConvBranches(t *testing.T) {
	mkEngine := func() *LiveMatchEngine { return chunk3LiveEngine(909) }
	// No-GK defending starters fall back to the first defender.
	e := mkEngine()
	defs := []*models.Player{}
	for _, p := range e.AwayStarters {
		if p.Category != "GK" {
			defs = append(defs, p)
		}
	}
	e.ResolveShot(e.HomeClub, e.AwayClub, e.HomeStarters, defs)
	// Empty defending starters: safe early return.
	e.ResolveShot(e.HomeClub, e.AwayClub, e.HomeStarters, nil)

	// Personality conversion branches.
	personalities := []string{"big_game_performer", "flamboyant_star", "dedicated_pro"}
	for _, pers := range personalities {
		he := mkEngine()
		for _, p := range he.HomeStarters {
			if p.Category == "FWD" {
				p.UniverseWonderkid = true
				p.Personality = pers
				p.MentorName = "Vet"
				p.Composure = 85
			}
		}
		he.HomeScore, he.AwayScore = 2, 2 // clutch for big-game
		he.CurrentMinute = 75             // late for big-game
		for i := 0; i < 10; i++ {
			he.ResolveShot(he.HomeClub, he.AwayClub, he.HomeStarters, he.AwayStarters)
		}
	}
	// OG fallback when defenders lack DEF/GK categories.
	ogHit := false
	for seed := int64(0); seed < 300 && !ogHit; seed++ {
		he := chunk3LiveEngine(seed)
		mids := []*models.Player{}
		for _, p := range he.AwayStarters {
			if p.Category == "MID" {
				mids = append(mids, p)
			}
		}
		// Swap in an all-midfield defending unit with a keeper-less shape.
		he.AwayStarters = append(append([]*models.Player{}, mids...), mids...)
		before := len(he.Events)
		for i := 0; i < 6; i++ {
			he.ResolveShot(he.HomeClub, he.AwayClub, he.HomeStarters, he.AwayStarters)
		}
		for _, ev := range he.Events[before:] {
			if ev.Type == "own_goal" && ev.Scorer != nil && ev.Scorer.Category == "MID" {
				ogHit = true
			}
		}
	}
	if !ogHit {
		t.Errorf("OG fallback culprit never observed in 300 seeds")
	}
}

func TestChunk3R3BookingFallbacks(t *testing.T) {
	// GK-only pool forces the live fallback; loop seeds to fire the 8.5%.
	fired := false
	for seed := int64(0); seed < 80 && !fired; seed++ {
		e := chunk3LiveEngine(seed)
		gk := chunk3GKOnly()
		before := len(e.Events)
		e.MaybeBookPlayer([]*models.Player{gk})
		for _, ev := range e.Events[before:] {
			if (ev.Type == "yellow" || ev.Type == "red") && ev.Player != nil && ev.Player.PlayerID == "GKSOLO" {
				fired = true
			}
		}
	}
	if !fired {
		t.Errorf("GK fallback booking never fired in 80 seeds")
	}
	// Empty pools are safe no-ops (loop seeds to cross the rate gate).
	for seed := int64(0); seed < 80; seed++ {
		e := chunk3LiveEngine(seed)
		e.MaybeBookPlayer(nil)
	}
}

func TestChunk3R3AwayPenaltyAndSubs(t *testing.T) {
	e := chunk3LiveEngine(919)
	e.PossessionTeam = "away"
	before := len(e.Events)
	e.ResolvePenalty(e.AwayClub, e.AwayStarters)
	if len(e.Events) == before {
		t.Fatalf("away penalty emitted nothing")
	}
	ev := e.Events[len(e.Events)-1]
	if ev.Type != "penalty" && ev.Type != "penalty_miss" {
		t.Fatalf("away penalty wrong type: %q", ev.Type)
	}
	// WONDERKID direct-sub category.
	he := chunk3LiveEngine(920)
	wkIn := &models.Player{PlayerID: "WKSUB", FullName: "Kid Sub", Position: "ST", Category: "FWD", OVR: 76, UniverseWonderkid: true}
	he.ExecuteDirectSub("home", he.HomeStarters[8], wkIn, 70, "chasing")
	last := he.Events[len(he.Events)-1]
	if last.Reason != "chasing" {
		t.Errorf("direct-sub reason not recorded: %+v", last)
	}
	found := false
	for _, c := range he.Commentary {
		if c.Category == "WONDERKID" && strings.Contains(c.Text, "Kid Sub") {
			found = true
		}
	}
	if !found {
		t.Errorf("WONDERKID direct-sub commentary missing")
	}
	// maxOVRPick prefers boosted wonderkids.
	pool := []*models.Player{
		chunk3PlainForPick("A", 85, false),
		chunk3PlainForPick("B", 80, true),
	}
	if got := maxOVRPick(pool, true); got == nil || got.PlayerID != "B" {
		t.Errorf("boosted pick wrong: %+v", got)
	}
	if got := maxOVRPick(pool, false); got == nil || got.PlayerID != "A" {
		t.Errorf("plain pick wrong: %+v", got)
	}
}

func TestChunk3RedDismissalUsesActiveStrengthAndGuardsSubstitutions(t *testing.T) {
	home, away := createTestClubs()
	e := NewLiveMatchEngine(home, away, nil, nil, 9211)
	e.ResetMatch()

	// Live strength keeps the normal-XI denominator even for short active
	// slices, and dismissals remove the player's effective rating entirely.
	p := &models.Player{PlayerID: "STRENGTH", OVR: 77}
	if got := e.liveStrength([]*models.Player{p}); got != 7 {
		t.Fatalf("short-XI live strength = %v; want 7", got)
	}
	e.Bookings[p.PlayerID] = 2
	if got := e.liveStrength([]*models.Player{p}); got != 0 {
		t.Fatalf("dismissed short-XI strength = %v; want 0", got)
	}
	if got := e.OnPitch([]*models.Player{nil, p}); len(got) != 0 {
		t.Fatalf("dismissed/nil players should not remain on pitch: %v", got)
	}

	// A dismissed attacker cannot take a penalty or create a live event when
	// it is the only available participant.
	e.Events = nil
	e.ResolvePenalty(home, []*models.Player{p})
	if len(e.Events) != 0 {
		t.Fatalf("dismissed player took a penalty: %+v", e.Events)
	}

	// Direct substitutions reject both a sent-off outgoing player and a
	// sent-off incoming player, while an ordinary change remains valid.
	e2 := NewLiveMatchEngine(home, away, nil, nil, 9212)
	e2.ResetMatch()
	out, in := e2.HomeStarters[1], e2.HomeBench[0]
	e2.Bookings[out.PlayerID] = 2
	e2.ExecuteDirectSub("home", out, in, 70, "dismissed")
	if e2.HomeStarters[1].PlayerID != out.PlayerID || e2.SubstitutionsMade["home"] != 0 {
		t.Fatalf("dismissed outgoing player was replaced: starters=%v subs=%v", e2.HomeStarters[1].PlayerID, e2.SubstitutionsMade)
	}
	e2.Bookings[in.PlayerID] = 2
	e2.ExecuteDirectSub("home", out, in, 70, "dismissed incoming")
	if e2.HomeStarters[1].PlayerID != out.PlayerID || e2.SubstitutionsMade["home"] != 0 {
		t.Fatalf("dismissed incoming player re-entered: starters=%v subs=%v", e2.HomeStarters[1].PlayerID, e2.SubstitutionsMade)
	}
	// Clear the artificial incoming dismissal before testing a legitimate sub.
	delete(e2.Bookings, out.PlayerID)
	delete(e2.Bookings, in.PlayerID)
	e2.ExecuteDirectSub("home", out, in, 70, "ordinary")
	if e2.HomeStarters[1].PlayerID != in.PlayerID || e2.SubstitutionsMade["home"] != 1 {
		t.Fatalf("ordinary substitution failed: starters=%v subs=%v", e2.HomeStarters[1].PlayerID, e2.SubstitutionsMade)
	}
	if len(e2.HomePlayers) <= 1 || e2.HomePlayers[1].PlayerID != in.PlayerID {
		t.Fatalf("radar identity did not follow substitution: %+v", e2.HomePlayers)
	}
}

func chunk3PlainForPick(id string, ovr int, wk bool) *models.Player {
	return &models.Player{PlayerID: id, FullName: "Pick " + id, Position: "ST", Category: "FWD", OVR: ovr, UniverseWonderkid: wk}
}

func TestChunk3R3PayloadExtras(t *testing.T) {
	e := chunk3LiveEngine(929)
	sc := matchreport.ToMiniPlayer(e.HomeStarters[9])
	// Away HT goal + away-beneficiary OG + positive capacity + no kickoff XIs.
	e.Events = []matchreport.MatchEventItem{
		{Minute: 20, Seq: 1, Type: "penalty", Side: "away", Scorer: &sc},
		{Minute: 40, Seq: 2, Type: "own_goal", Side: "home", Beneficiary: "away", Scorer: &sc},
	}
	e.HomeScore, e.AwayScore = 0, 2
	e.HomeClub.StadiumCapacity = 60000
	e.HomeKickoffXI = nil
	e.AwayKickoffXI = nil
	payload := e.BuildLivePayload()
	if payload.HTAway != 2 {
		t.Errorf("away HT = %d; want 2", payload.HTAway)
	}
	if payload.Attendance > 60000 {
		t.Errorf("attendance exceeds capacity: %d", payload.Attendance)
	}
	if len(payload.HomeXI) != 11 || len(payload.AwayXI) != 11 {
		t.Errorf("kickoff fallback XIs wrong: %d/%d", len(payload.HomeXI), len(payload.AwayXI))
	}
}
