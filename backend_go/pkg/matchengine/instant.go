package matchengine

import (
	"fmt"
	"math"
	"math/rand"
	"sort"

	"football_sim/pkg/growth"
	"football_sim/pkg/managers"
	"football_sim/pkg/matchreport"
	"football_sim/pkg/models"
)

// InstantMatchConfig sets optional rules for instant match simulation.
type InstantMatchConfig struct {
	Weather     string // clear, rain, snow, wind
	DerbyHeat   int    // 0 - 100
	IsDerby     bool   // true if high-stakes rivalry
	Referee     string // official name or personality: strict, lenient, balanced
	Competition string // super-league, ucl, super-cup
	Matchweek   int
}

// shootingAccuracyWeatherFactor returns the existing on-target accuracy
// adjustment shared by instant and live open-play shooting.
func shootingAccuracyWeatherFactor(weather string) float64 {
	switch weather {
	case "rain":
		return 0.96
	case "wind":
		return 0.92
	default:
		return 1.0
	}
}

// weatherCardClimateFactor returns the rain card-climate multiplier shared
// by instant foul volume and live post-shot bookings.
func weatherCardClimateFactor(weather string) float64 {
	if weather == "rain" {
		return 1.05
	}
	return 1.0
}

// actionRank orders simultaneous events the way the Python engine does.
var actionRank = map[string]int{
	"red": 0, "yellow": 1, "card": 1, "sub": 2, "var_review": 3,
	"own_goal": 4, "goal": 5, "corner_goal": 5, "free_kick_goal": 5,
	"penalty": 5, "penalty_miss": 5,
}

type simAction struct {
	minute   int
	kind     string // goal, penalty, corner_goal, free_kick_goal, card, sub
	side     string
	out      *models.Player
	in       *models.Player
	tiebreak float64
}

func managerDisplayName(m *managers.ManagerProfile, short string) string {
	if m != nil && m.Name != "" {
		return m.Name
	}
	return short + " Manager"
}

func instantOwnGoalCulprit(defendersOnPitch []*models.Player, rng *rand.Rand) *models.Player {
	if len(defendersOnPitch) == 0 {
		return nil
	}
	return matchreport.PickOwnGoalCulprit(defendersOnPitch, nil, rng)
}

// SimulateInstantMatch simulates a full fixture instantly, mirroring
// match_report.generate_instant_match: Poisson goal model, set pieces with
// aerial/free-kick takers, referee-weighted cards with reds, planned
// substitutions, own goals, penalty shootouts-in-play, VAR reviews, and full
// stat/xG/report assembly.
//
// The simulation is pure: unlike earlier revisions it does NOT update club
// records — callers (tournament season flow, single-fixture API) own the
// UpdateResult step, exactly like the Python engine.
func SimulateInstantMatch(
	homeClub *models.Club,
	awayClub *models.Club,
	homeMgr *managers.ManagerProfile,
	awayMgr *managers.ManagerProfile,
	growthEngine *growth.GrowthEngine,
	cfg *InstantMatchConfig,
	rng *rand.Rand,
) *matchreport.MatchReport {
	if rng == nil {
		rng = rand.New(rand.NewSource(rand.Int63()))
	}
	if cfg == nil {
		cfg = &InstantMatchConfig{Weather: "clear", Referee: "balanced"}
	}
	weather := cfg.Weather
	if weather == "" {
		weather = "clear"
	}
	// NOTE: growthEngine is accepted for signature stability but intentionally
	// unused on the instant path — Python's generate_instant_match likewise
	// resolves aerial and free-kick takers from OVR fallbacks, not growth attrs.
	_ = growthEngine

	// Referee crew and card climate.
	refereeName := matchreport.PickRefereeName(rng, cfg.Referee)
	refMult := matchreport.RefCardMult(matchreport.RefereePersonality(refereeName))
	highHeatDerby := cfg.IsDerby || cfg.DerbyHeat > 70

	homeStyle := "possession"
	if homeMgr != nil && homeMgr.Style != "" {
		homeStyle = homeMgr.Style
	}
	awayStyle := "possession"
	if awayMgr != nil && awayMgr.Style != "" {
		awayStyle = awayMgr.Style
	}
	homeEdge := managers.TacticEdge(homeStyle, awayStyle)

	fxKey := models.FixtureContext(cfg.Competition, cfg.Matchweek)
	homeXI := homeClub.GetStartingEleven(fxKey)
	awayXI := awayClub.GetStartingEleven(fxKey)
	homeBench := homeClub.GetBench(homeXI, 7, fxKey)
	awayBench := awayClub.GetBench(awayXI, 7, fxKey)

	// Rating base uses the actual matchday XI (fatigue + absences), not the
	// cached squad rating. Missing kids on exam week should show up here.
	moraleBonus := func(morale int) float64 {
		if morale > 80 {
			return 0.8
		}
		if morale < 40 {
			return -0.8
		}
		return 0.0
	}
	homeMorale := homeClub.Morale
	awayMorale := awayClub.Morale
	bigGame := bigGameContext(cfg.Competition, cfg.IsDerby)
	rh := xiEffectiveRating(homeXI, 0) + 1.5 + moraleBonus(homeMorale)
	ra := xiEffectiveRating(awayXI, 0) + moraleBonus(awayMorale)
	delta := rh - ra
	if homeMorale < 40 && delta > 2 {
		delta -= 0.5 + rng.Float64()*1.5
	}
	if awayMorale < 40 && delta < -2 {
		delta += 0.5 + rng.Float64()*1.5
	}

	snowFactor := 1.0
	if weather == "snow" {
		snowFactor = 0.95
	}
	homeLambda := math.Max(0.3, (1.45+0.09*delta+homeEdge)*snowFactor)
	awayLambda := math.Max(0.3, (1.20-0.09*delta-homeEdge*0.5)*snowFactor)
	rawHomeGoals := matchreport.Poisson(rng, homeLambda)
	rawAwayGoals := matchreport.Poisson(rng, awayLambda)

	var actions []simAction
	addAction := func(minute int, kind, side string) {
		actions = append(actions, simAction{minute: minute, kind: kind, side: side, tiebreak: rng.Float64()})
	}
	for _, m := range matchreport.SampleMinutes(rng, rawHomeGoals) {
		addAction(m, "goal", "home")
	}
	for _, m := range matchreport.SampleMinutes(rng, rawAwayGoals) {
		addAction(m, "goal", "away")
	}
	// Single in-play penalty kick (22%).
	if rng.Float64() < 0.22 {
		side := "home"
		if rng.Float64() < 0.5 {
			side = "away"
		}
		addAction(20+rng.Intn(69), "penalty", side)
	}

	// Corners with header attempts.
	homeCorners := matchreport.Poisson(rng, 5)
	awayCorners := matchreport.Poisson(rng, 4)
	spConv := 0.30
	if weather == "snow" {
		spConv = 0.33
	}
	for i := 0; i < homeCorners; i++ {
		if rng.Float64() < 0.15 && rng.Float64() < spConv {
			addAction(6+rng.Intn(83), "corner_goal", "home")
		}
	}
	for i := 0; i < awayCorners; i++ {
		if rng.Float64() < 0.15 && rng.Float64() < spConv {
			addAction(6+rng.Intn(83), "corner_goal", "away")
		}
	}

	// Dangerous direct free kicks (instant path uses OVR fallback, as in Python).
	if matchreport.PickFreeKickTaker(homeXI, nil) != nil {
		for i := 0; i < 1+rng.Intn(2); i++ {
			if rng.Float64() < 0.05 {
				addAction(15+rng.Intn(71), "free_kick_goal", "home")
			}
		}
	}
	if matchreport.PickFreeKickTaker(awayXI, nil) != nil {
		for i := 0; i < 1+rng.Intn(2); i++ {
			if rng.Float64() < 0.05 {
				addAction(15+rng.Intn(71), "free_kick_goal", "away")
			}
		}
	}

	// Card fouls shaped by referee and rain.
	cardMult := refMult * weatherCardClimateFactor(weather)
	for _, side := range []string{"home", "away"} {
		for i := 0; i < matchreport.Poisson(rng, 1.6*cardMult); i++ {
			addAction(12+rng.Intn(79), "card", side)
		}
	}

	// Planned substitutions for both dugouts.
	for _, sub := range matchreport.PlanSubstitutions(homeXI, homeBench, 5, rng) {
		actions = append(actions, simAction{minute: sub.Minute, kind: "sub", side: "home", out: sub.Out, in: sub.In, tiebreak: rng.Float64()})
	}
	for _, sub := range matchreport.PlanSubstitutions(awayXI, awayBench, 5, rng) {
		actions = append(actions, simAction{minute: sub.Minute, kind: "sub", side: "away", out: sub.Out, in: sub.In, tiebreak: rng.Float64()})
	}
	sort.SliceStable(actions, func(i, j int) bool {
		if actions[i].minute != actions[j].minute {
			return actions[i].minute < actions[j].minute
		}
		ri, rj := actionRank[actions[i].kind], actionRank[actions[j].kind]
		if ri != rj {
			return ri < rj
		}
		return actions[i].tiebreak < actions[j].tiebreak
	})

	field := map[string][]*models.Player{"home": append([]*models.Player(nil), homeXI...), "away": append([]*models.Player(nil), awayXI...)}
	sideBench := map[string][]*models.Player{"home": append([]*models.Player(nil), homeBench...), "away": append([]*models.Player(nil), awayBench...)}
	bookings := map[string]int{}
	reds := map[string]int{"home": 0, "away": 0}
	live := func(side string) []*models.Player {
		var on []*models.Player
		for _, p := range field[side] {
			if bookings[p.PlayerID] < 2 {
				on = append(on, p)
			}
		}
		return on
	}
	onField := func(side string, pid string) bool {
		for _, p := range live(side) {
			if p.PlayerID == pid {
				return true
			}
		}
		return false
	}

	var events []matchreport.MatchEventItem
	seq := 1
	gh, ga := 0, 0
	emit := func(e matchreport.MatchEventItem) {
		e.Seq = seq
		seq++
		events = append(events, e)
	}
	disp := func(m int) string { return matchreport.MinuteDisplay(m, rng) }

	for _, act := range actions {
		m, side, kind := act.minute, act.side, act.kind

		if kind == "sub" {
			// Planned in-players always remain on the bench (bench cards do
			// not exist and used_in prevents reuse); only the out-player can
			// have left the field via dismissal.
			if !onField(side, act.out.PlayerID) {
				continue
			}
			next := make([]*models.Player, 0, len(field[side]))
			for _, p := range field[side] {
				if p.PlayerID == act.out.PlayerID {
					next = append(next, act.in)
				} else {
					next = append(next, p)
				}
			}
			field[side] = next
			kept := sideBench[side][:0]
			for _, p := range sideBench[side] {
				if p.PlayerID != act.in.PlayerID {
					kept = append(kept, p)
				}
			}
			sideBench[side] = kept
			outMini := matchreport.ToMiniPlayer(act.out)
			inMini := matchreport.ToMiniPlayer(act.in)
			emit(matchreport.MatchEventItem{
				Minute: m, Display: fmt.Sprintf("Substitution %s (%s ➜ %s)", disp(m), act.out.FullName, act.in.FullName),
				Type: "sub", Side: side, PlayerOut: &outMini, PlayerIn: &inMini,
			})
			continue
		}

		if kind == "card" {
			// A side cannot empty its field via cards: bookings only reach 2
			// through reds (~7% of ~2 card actions), so the pool below is
			// never empty in practice.
			pool := live(side)
			if len(pool) == 0 {
				continue
			}
			booked := matchreport.PickBooked(pool, rng)
			if booked == nil {
				continue
			}
			prior := bookings[booked.PlayerID]
			redChance := 0.07
			if highHeatDerby {
				redChance += 0.15
			}
			color, sentOff := "yellow", false
			detail := ""
			if prior >= 1 {
				color, sentOff = "red", true
				detail = "second_yellow"
			} else if rng.Float64() < redChance {
				// Python marks first-card reds as sent_off=false; the career
				// still removes the player. Instant matches follow live.
				color, sentOff = "red", true
				detail = "straight_red"
			}
			if color == "red" {
				bookings[booked.PlayerID] = 2
				reds[side]++
			} else {
				bookings[booked.PlayerID] = 1
			}
			bMini := matchreport.ToMiniPlayer(booked)
			label := "Yellow Card"
			if color == "red" {
				label = "Red Card"
			}
			emit(matchreport.MatchEventItem{
				Minute: m, Display: fmt.Sprintf("%s: %s %s", label, booked.FullName, disp(m)),
				Type: color, Side: side, Player: &bMini, SentOff: sentOff,
				HomeScore: gh, AwayScore: ga, Detail: detail,
			})
			continue
		}

		// live() is empty only after 11 send-offs (see card branch note);
		// PickScorer/PickBooked nil-guards remain the last line of defense.
		on := live(side)
		if len(on) == 0 {
			continue
		}

		// A man down wastes chances: remaining attacking actions for the
		// reduced side are less likely to become shots on goal.
		if kind == "goal" || kind == "penalty" || kind == "corner_goal" || kind == "free_kick_goal" {
			other := "away"
			if side == "away" {
				other = "home"
			}
			extraReds := reds[side] - reds[other]
			if extraReds > 0 && rng.Float64() < math.Min(0.45, 0.18*float64(extraReds)) {
				continue
			}
		}

		if kind == "penalty" {
			taker := matchreport.PickScorer(on, m, gh == ga, rng, bigGame)
			if taker == nil {
				continue
			}
			sideMorale := homeMorale
			if side == "away" {
				sideMorale = awayMorale
			}
			moraleConv := 0.0
			if sideMorale > 80 {
				moraleConv = 0.02
			} else if sideMorale < 40 {
				moraleConv = -0.02
			}
			conv := math.Max(0.60, math.Min(0.88, 0.76+float64(taker.OVR-75)*0.005+moraleConv))
			if taker.UniverseWonderkid {
				if taker.MentorName != "" {
					conv = math.Min(0.92, conv+0.02)
				}
				if taker.Composure > 75 {
					conv = math.Min(0.95, conv+math.Min(0.04, float64(taker.Composure-75)*0.002))
				}
			}
			tMini := matchreport.ToMiniPlayer(taker)
			if rng.Float64() < conv {
				if side == "home" {
					gh++
				} else {
					ga++
				}
				emit(matchreport.MatchEventItem{
					Minute: m, Display: fmt.Sprintf("Penalty: %s %s", taker.FullName, disp(m)),
					Type: "penalty", Side: side, Scorer: &tMini,
				})
			} else {
				emit(matchreport.MatchEventItem{
					Minute: m, Display: fmt.Sprintf("Penalty Miss: %s %s", taker.FullName, disp(m)),
					Type: "penalty_miss", Side: side, Scorer: &tMini,
				})
			}
			continue
		}

		// Own goal (4% of goal actions, charged to the defending side).
		if rng.Float64() < 0.04 {
			defSide := "away"
			if side == "away" {
				defSide = "home"
			}
			defendersOnPitch := live(defSide)
			if len(defendersOnPitch) == 0 {
				// A dismissed side cannot concede an own goal through an active
				// attacker; skip the action without changing the running score.
				continue
			}
			culprit := instantOwnGoalCulprit(defendersOnPitch, rng)
			if culprit == nil {
				continue
			}
			cMini := matchreport.ToMiniPlayer(culprit)
			emit(matchreport.MatchEventItem{
				Minute: m, Display: fmt.Sprintf("Own Goal: %s %s", culprit.FullName, disp(m)),
				Type: "own_goal", Side: defSide, Beneficiary: side, Scorer: &cMini,
			})
			if side == "home" {
				gh++
			} else {
				ga++
			}
			continue
		}

		// Regular, corner-header, and direct free-kick goals.
		var scorer *models.Player
		var assister *models.Player
		etype := "goal"
		switch kind {
		case "corner_goal":
			scorer = matchreport.PickAerialTarget(on, nil, rng)
			assister = matchreport.PickCornerTaker(on, scorer, rng)
			etype = "corner_goal"
		case "free_kick_goal":
			scorer = matchreport.PickFreeKickTaker(on, nil)
			if scorer == nil {
				scorer = matchreport.PickScorer(on, m, false, rng, bigGame)
			}
			etype = "free_kick_goal"
		default:
			clutch := math.Abs(float64(gh-ga)) <= 1
			scorer = matchreport.PickScorer(on, m, clutch, rng, bigGame)
			assister = matchreport.PickAssister(on, scorer, rng)
		}
		if scorer == nil {
			continue
		}
		sMini := matchreport.ToMiniPlayer(scorer)
		var aMini *matchreport.MiniPlayer
		display := fmt.Sprintf("%s %s", scorer.FullName, disp(m))
		if assister != nil {
			am := matchreport.ToMiniPlayer(assister)
			aMini = &am
			display = fmt.Sprintf("%s %s (Assist: %s)", scorer.FullName, disp(m), assister.FullName)
		}
		goalEvent := matchreport.MatchEventItem{
			Minute: m, Display: display, Type: etype, Side: side, Scorer: &sMini, Assister: aMini,
		}
		if side == "home" {
			gh++
		} else {
			ga++
		}

		// VAR review (~5% of goals, ~30% disallowed).
		if rng.Float64() < 0.05 {
			if rng.Float64() < 0.30 {
				goalEvent.Disallowed = true
				if side == "home" {
					gh--
				} else {
					ga--
				}
				emit(goalEvent)
				reason := "offside"
				if rng.Float64() < 0.5 {
					reason = "handball"
				}
				emit(matchreport.MatchEventItem{
					Minute: m, Display: fmt.Sprintf("VAR Review: goal disallowed (%s) %s", reason, disp(m)),
					Type: "var_review", Side: side,
					Outcome: "goal_disallowed", Reason: reason, Decision: "goal_disallowed", Disallowed: true,
				})
			} else {
				emit(goalEvent)
				emit(matchreport.MatchEventItem{
					Minute: m, Display: fmt.Sprintf("VAR Review: goal stands %s", disp(m)),
					Type: "var_review", Side: side,
					Outcome: "goal_stands", Reason: "check complete", Decision: "goal_stands",
				})
			}
		} else {
			emit(goalEvent)
		}
	}

	// Running score backfill in chronological order.
	ordered := append([]matchreport.MatchEventItem(nil), events...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Minute == ordered[j].Minute {
			return ordered[i].Seq < ordered[j].Seq
		}
		return ordered[i].Minute < ordered[j].Minute
	})
	hRun, aRun := 0, 0
	for i := range ordered {
		e := &ordered[i]
		switch e.Type {
		case "goal", "penalty", "corner_goal", "free_kick_goal":
			if !e.Disallowed {
				if e.Side == "home" {
					hRun++
				} else {
					aRun++
				}
			}
		case "own_goal":
			if e.Beneficiary == "home" {
				hRun++
			} else {
				aRun++
			}
		}
		e.HomeScore = hRun
		e.AwayScore = aRun
	}
	events = ordered

	// Volume stats shaped by the actual matchday XI strength.
	hEdge := math.Max(-8, math.Min(8, xiEffectiveRating(homeXI, 0)-xiEffectiveRating(awayXI, 0)))
	homeShots := int(math.Max(4, float64(matchreport.Poisson(rng, 11+hEdge*0.5))))
	awayShots := int(math.Max(4, float64(matchreport.Poisson(rng, 11-hEdge*0.5))))
	homePoss := int(math.Max(28, math.Min(72, 50+hEdge*1.6+rng.Float64()*8-4)))
	accuracyFactor := shootingAccuracyWeatherFactor(weather)
	// uniform(0.30, 0.42) of shots are on target; with at least 4 shots the
	// truncation below is always strictly below the total (no cap needed).
	homeOn := int(float64(homeShots) * (0.30 + rng.Float64()*0.12) * accuracyFactor)
	awayOn := int(float64(awayShots) * (0.30 + rng.Float64()*0.12) * accuracyFactor)

	stats := matchreport.BuildStats(
		homeClub, awayClub,
		homeShots, awayShots, homeOn, awayOn, homePoss,
		homeCorners, awayCorners,
		events, weather, rng,
	)

	// Half-time score (events at minute <= 45).
	htH, htA := 0, 0
	for _, e := range events {
		if e.Minute > 45 {
			continue
		}
		switch e.Type {
		case "goal", "penalty", "corner_goal", "free_kick_goal":
			if !e.Disallowed {
				if e.Side == "home" {
					htH++
				} else {
					htA++
				}
			}
		case "own_goal":
			if e.Beneficiary == "home" {
				htH++
			} else {
				htA++
			}
		}
	}

	cap := homeClub.StadiumCapacity
	if cap <= 0 {
		cap = 50000
	}
	attendance := int(math.Max(12000, math.Min(float64(cap), float64(cap)*(0.76+rng.Float64()*0.21))))

	shotMap := matchreport.GenerateShotMap(homeClub, awayClub, events, homeShots, awayShots, homeOn, awayOn, rng, nil)
	heatmap := matchreport.GenerateTouchHeatmap(homeClub, awayClub, homePoss, rng, nil)
	press := matchreport.GeneratePressConference(
		homeClub, awayClub, gh, ga, events,
		managerDisplayName(homeMgr, homeClub.ShortName),
		managerDisplayName(awayMgr, awayClub.ShortName),
	)

	payload := matchreport.InstantPayload{
		HomeGoals: gh, AwayGoals: ga,
		HomeClubName: homeClub.ClubName, AwayClubName: awayClub.ClubName,
		Events: events,
		HomeXI: homeXI, AwayXI: awayXI,
		HomeBench: homeBench, AwayBench: awayBench,
		Stats:  stats,
		HTHome: htH, HTAway: htA,
		Attendance: attendance, Referee: refereeName, Weather: weather,
		ShotMap: shotMap, Heatmap: heatmap, Press: press,
	}
	report := matchreport.AssembleReport(payload, "instant", rng)
	return &report
}

func xiEffectiveRating(xi []*models.Player, _ int) float64 {
	sum := 0
	count := 0
	for _, p := range xi {
		if p == nil {
			continue
		}
		sum += p.EffectiveOVR()
		count++
	}
	if count == 0 {
		return 0
	}
	return float64(sum) / float64(count)
}
