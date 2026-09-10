package tournament

import (
	"math"
	"math/rand"

	"football_sim/pkg/matchreport"
	"football_sim/pkg/models"
)

func (tm *TournamentManager) applyKnockoutDecider(f *Fixture, home, away *models.Club, payload *matchreport.InstantPayload) {
	rng := tm.RNG
	if rng == nil {
		rng = rand.New(rand.NewSource(1))
	}
	tm.applyKnockoutDeciderWithRNG(f, home, away, payload, rng)
}

// applyKnockoutDeciderWithRNG resolves extra time and shootouts with an
// explicit stream so slate workers never share tm.RNG. Cup-state reads are
// safe during parallel compute because application (the only writer) runs
// afterwards, serially.
func (tm *TournamentManager) applyKnockoutDeciderWithRNG(f *Fixture, home, away *models.Club, payload *matchreport.InstantPayload, rng *rand.Rand) {
	if f == nil || payload == nil {
		return
	}
	if f.Competition != "ucl" && f.Competition != "super-cup" {
		return
	}
	if rng == nil {
		rng = rand.New(rand.NewSource(1))
	}

	needsET := false
	if f.Competition == "super-cup" && payload.HomeGoals == payload.AwayGoals {
		needsET = true
	} else if f.Stage == "Final" && payload.HomeGoals == payload.AwayGoals {
		needsET = true
	} else if f.Leg == 2 {
		leg1 := tm.uclLeg(f.TieID, 1)
		if leg1 != nil && leg1.Status == "finished" && leg1.HomeGoals != nil && leg1.AwayGoals != nil {
			aggH := *leg1.HomeGoals + payload.AwayGoals
			aggA := *leg1.AwayGoals + payload.HomeGoals
			needsET = aggH == aggA
		}
	}
	if !needsET {
		return
	}

	minutes := []int{97, 105, 112, 118}
	rng.Shuffle(len(minutes), func(i, j int) { minutes[i], minutes[j] = minutes[j], minutes[i] })
	extra := 0
	for _, minute := range minutes {
		if extra >= 2 {
			break
		}
		if rng.Float64() >= 0.42 {
			continue
		}
		side := "home"
		if rng.Float64() >= 0.52 {
			side = "away"
		}
		tm.addETGoal(payload, side, minute, rng, f.DerbyName != "" || f.Competition == "ucl")
		extra++
	}

	stillLevel := false
	if f.Competition == "super-cup" || f.Stage == "Final" {
		stillLevel = payload.HomeGoals == payload.AwayGoals
	} else {
		leg1 := tm.uclLeg(f.TieID, 1)
		if leg1 != nil && leg1.HomeGoals != nil && leg1.AwayGoals != nil {
			stillLevel = (*leg1.HomeGoals + payload.AwayGoals) == (*leg1.AwayGoals + payload.HomeGoals)
		} else {
			stillLevel = payload.HomeGoals == payload.AwayGoals
		}
	}

	if stillLevel {
		hPen, aPen := penaltyShootout(home, away, rng, models.FixtureContext(f.Competition, f.Matchweek))
		dec := "penalties"
		payload.DecidedBy = &dec
		payload.Penalties = []int{hPen, aPen}
	} else {
		dec := "extra_time"
		payload.DecidedBy = &dec
		payload.Penalties = nil
	}
}

func (tm *TournamentManager) addETGoal(payload *matchreport.InstantPayload, side string, minute int, rng *rand.Rand, bigGame ...bool) {
	on := matchreport.OnFieldPlayers(*payload, side, minute)
	if len(on) == 0 {
		if side == "home" {
			on = payload.HomeXI
		} else {
			on = payload.AwayXI
		}
	}
	if len(on) == 0 {
		return
	}
	inBigGame := len(bigGame) > 0 && bigGame[0]
	scorer := matchreport.PickScorer(on, minute, true, rng, inBigGame)
	if scorer == nil {
		return
	}
	assister := matchreport.PickAssister(on, scorer, rng)
	seq := 0
	for _, e := range payload.Events {
		if e.Seq > seq {
			seq = e.Seq
		}
	}
	seq++
	if side == "home" {
		payload.HomeGoals++
	} else {
		payload.AwayGoals++
	}
	sMini := matchreport.ToMiniPlayer(scorer)
	ev := matchreport.MatchEventItem{
		Minute:    minute,
		Display:   matchreport.MinuteDisplay(minute, rng),
		Seq:       seq,
		Type:      "goal",
		Side:      side,
		Scorer:    &sMini,
		HomeScore: payload.HomeGoals,
		AwayScore: payload.AwayGoals,
		Period:    "et",
	}
	if assister != nil {
		am := matchreport.ToMiniPlayer(assister)
		ev.Assister = &am
	}
	payload.Events = append(payload.Events, ev)
}

func penaltyShootout(home, away *models.Club, rng *rand.Rand, fixture ...string) (int, int) {
	if rng == nil {
		rng = rand.New(rand.NewSource(1))
	}
	hXI := home.GetStartingEleven(fixture...)
	aXI := away.GetStartingEleven(fixture...)
	hOut := outfield(hXI)
	aOut := outfield(aXI)
	if len(hOut) == 0 {
		hOut = hXI
	}
	if len(aOut) == 0 {
		aOut = aXI
	}
	if len(hOut) == 0 || len(aOut) == 0 {
		return 0, 0
	}
	hScore, aScore := 0, 0
	for roundN := 1; roundN <= 20; roundN++ {
		hp := hOut[(roundN-1)%len(hOut)]
		ap := aOut[(roundN-1)%len(aOut)]
		hP := math.Max(0.55, math.Min(0.90, 0.72+float64(hp.OVR-75)*0.005))
		aP := math.Max(0.55, math.Min(0.90, 0.72+float64(ap.OVR-75)*0.005))
		if rng.Float64() < hP {
			hScore++
		}
		if rng.Float64() < aP {
			aScore++
		}
		if roundN >= 5 && hScore != aScore {
			return hScore, aScore
		}
	}
	if hScore == aScore {
		hScore++
	}
	return hScore, aScore
}

func outfield(xi []*models.Player) []*models.Player {
	var out []*models.Player
	for _, p := range xi {
		if p != nil && p.Category != "GK" {
			out = append(out, p)
		}
	}
	return out
}

func (tm *TournamentManager) uclLeg(tieID string, leg int) *Fixture {
	if tieID == "" {
		return nil
	}
	for i := range tm.UCLFixtures {
		f := &tm.UCLFixtures[i]
		if f.TieID == tieID && f.Leg == leg {
			return f
		}
	}
	return nil
}

func (tm *TournamentManager) uclTieLegs(tieID string) []*Fixture {
	var out []*Fixture
	for i := range tm.UCLFixtures {
		f := &tm.UCLFixtures[i]
		if f.TieID == tieID {
			out = append(out, f)
		}
	}
	return out
}

func (tm *TournamentManager) scTieFixture(tieID string) *Fixture {
	for i := range tm.SuperCupFixtures {
		f := &tm.SuperCupFixtures[i]
		if f.TieID == tieID {
			return f
		}
	}
	return nil
}

func (tm *TournamentManager) uclLegBlocked(f *Fixture) string {
	if f == nil || f.Competition != "ucl" || f.Leg != 2 {
		return ""
	}
	if f.TieID == "" {
		return ""
	}
	leg1 := tm.uclLeg(f.TieID, 1)
	if leg1 == nil {
		return "Champions Cup leg 1 has not been scheduled."
	}
	if leg1.Status != "finished" {
		return "Play Champions Cup leg 1 before the return."
	}
	return ""
}

func (tm *TournamentManager) cupWeek(planned int) int {
	cur := tm.CurrentMatchweek
	if cur < 1 {
		cur = 1
	}
	if cur > tm.MaxMatchweeks {
		cur = tm.MaxMatchweeks
	}
	if planned < cur {
		planned = cur
	}
	if planned > tm.MaxMatchweeks {
		planned = tm.MaxMatchweeks
	}
	return planned
}

func (tm *TournamentManager) resolveTwoLegged(tie *CupTie, tieID string) {
	if tie == nil || tie.HomeID == "" {
		return
	}
	leg1 := tm.uclLeg(tieID, 1)
	leg2 := tm.uclLeg(tieID, 2)
	if leg1 != nil && leg1.Status == "finished" && leg1.HomeGoals != nil && leg1.AwayGoals != nil {
		tie.Leg1 = []int{*leg1.HomeGoals, *leg1.AwayGoals}
	}
	if leg2 != nil && leg2.Status == "finished" && leg2.HomeGoals != nil && leg2.AwayGoals != nil {
		tie.Leg2 = []int{*leg2.HomeGoals, *leg2.AwayGoals}
	}
	if leg1 == nil || leg2 == nil || leg1.Status != "finished" || leg2.Status != "finished" {
		return
	}
	aggH := *leg1.HomeGoals + *leg2.AwayGoals
	aggA := *leg1.AwayGoals + *leg2.HomeGoals
	if aggH != aggA {
		if aggH > aggA {
			tie.WinnerID = tie.HomeID
		} else {
			tie.WinnerID = tie.AwayID
		}
		tie.DecidedBy = leg2.DecidedBy
		return
	}
	tie.DecidedBy = leg2.DecidedBy
	tie.Penalties = append([]int(nil), leg2.Penalties...)
	if len(leg2.Penalties) >= 2 {
		if leg2.Penalties[0] > leg2.Penalties[1] {
			tie.WinnerID = tie.HomeID
		} else {
			tie.WinnerID = tie.AwayID
		}
	} else {
		tie.WinnerID = tie.HomeID
	}
}

func (tm *TournamentManager) resolveFinal() {
	var final *Fixture
	for i := range tm.UCLFixtures {
		f := &tm.UCLFixtures[i]
		if f.Stage == "Final" && f.Status == "finished" {
			final = f
			break
		}
	}
	if final == nil || tm.UCLFinal.HomeID == "" {
		return
	}
	if final.HomeGoals == nil || final.AwayGoals == nil {
		return
	}
	s1, s2 := *final.HomeGoals, *final.AwayGoals
	tm.UCLFinal.Leg1 = []int{s1, s2}
	tm.UCLFinal.DecidedBy = final.DecidedBy
	tm.UCLFinal.Penalties = append([]int(nil), final.Penalties...)
	t1, t2 := tm.UCLFinal.HomeID, tm.UCLFinal.AwayID
	if s1 != s2 {
		if s1 > s2 {
			tm.UCLFinal.WinnerID = t1
		} else {
			tm.UCLFinal.WinnerID = t2
		}
	} else if len(final.Penalties) >= 2 {
		if final.Penalties[0] > final.Penalties[1] {
			tm.UCLFinal.WinnerID = t1
		} else {
			tm.UCLFinal.WinnerID = t2
		}
	} else {
		tm.UCLFinal.WinnerID = t1
	}
	tm.UCLChampionID = tm.UCLFinal.WinnerID
}

func (tm *TournamentManager) syncUCLBracket() {
	for k, tie := range tm.UCLQuarterFinals {
		t := tie
		tm.resolveTwoLegged(&t, k)
		tm.UCLQuarterFinals[k] = t
	}
	for k, tie := range tm.UCLSemiFinals {
		t := tie
		tm.resolveTwoLegged(&t, k)
		tm.UCLSemiFinals[k] = t
	}
	tm.resolveFinal()
}

func (tm *TournamentManager) drawUCLQuarterfinals() string {
	a, b := tm.uclStandingsUnlocked()
	if len(a) < 4 || len(b) < 4 {
		return ""
	}
	pairs := []struct {
		key        string
		home, away *models.Club
	}{
		{"qf_1", a[0], b[3]},
		{"qf_2", b[0], a[3]},
		{"qf_3", a[1], b[2]},
		{"qf_4", b[1], a[2]},
	}
	tm.UCLQuarterFinals = map[string]CupTie{}
	names := make([]string, 0, 8)
	w1 := tm.cupWeek(UCLQFWeeks[0])
	w2 := w1 + 1
	if w2 < UCLQFWeeks[1] {
		w2 = UCLQFWeeks[1]
	}
	if w2 > tm.MaxMatchweeks {
		w2 = tm.MaxMatchweeks
	}
	for _, p := range pairs {
		tm.UCLQuarterFinals[p.key] = emptyTie(p.home, p.away)
		tm.makeUCLLeg(p.key, "Quarter-final", 1, w1, p.home, p.away)
		tm.makeUCLLeg(p.key, "Quarter-final", 2, w2, p.away, p.home)
		names = append(names, p.home.ShortName, p.away.ShortName)
	}
	return "UCL Group Stage Complete! Quarter-Finalists: " + joinShort(names) + "."
}

func (tm *TournamentManager) drawUCLSemifinals() string {
	w := map[string]*models.Club{}
	for k, t := range tm.UCLQuarterFinals {
		w[k] = tm.Clubs[t.WinnerID]
	}
	if w["qf_1"] == nil || w["qf_2"] == nil || w["qf_3"] == nil || w["qf_4"] == nil {
		return ""
	}
	tm.UCLSemiFinals = map[string]CupTie{
		"semi_1": emptyTie(w["qf_1"], w["qf_2"]),
		"semi_2": emptyTie(w["qf_3"], w["qf_4"]),
	}
	w1 := tm.cupWeek(UCLSFWeeks[0])
	w2 := w1 + 1
	if w2 < UCLSFWeeks[1] {
		w2 = UCLSFWeeks[1]
	}
	if w2 > tm.MaxMatchweeks {
		w2 = tm.MaxMatchweeks
	}
	tm.makeUCLLeg("semi_1", "Semi-final", 1, w1, w["qf_1"], w["qf_2"])
	tm.makeUCLLeg("semi_1", "Semi-final", 2, w2, w["qf_2"], w["qf_1"])
	tm.makeUCLLeg("semi_2", "Semi-final", 1, w1, w["qf_3"], w["qf_4"])
	tm.makeUCLLeg("semi_2", "Semi-final", 2, w2, w["qf_4"], w["qf_3"])
	return "UCL Semi-Finalists Confirmed: " + w["qf_1"].ShortName + ", " + w["qf_2"].ShortName + ", " + w["qf_3"].ShortName + ", " + w["qf_4"].ShortName + "!"
}

func (tm *TournamentManager) drawUCLFinal() string {
	t1 := tm.Clubs[tm.UCLSemiFinals["semi_1"].WinnerID]
	t2 := tm.Clubs[tm.UCLSemiFinals["semi_2"].WinnerID]
	if t1 == nil || t2 == nil {
		return ""
	}
	tm.UCLFinal = emptyTie(t1, t2)
	tm.makeUCLLeg("final", "Final", 1, tm.cupWeek(UCLFinalWeek), t1, t2)
	return "UCL Finalists Confirmed: " + t1.ClubName + " vs " + t2.ClubName + "!"
}

func (tm *TournamentManager) refreshUCLStage() {
	if tm.UCLChampionID != "" {
		tm.UCLStage = "CHAMPION_CROWNED"
		return
	}
	if tm.UCLFinal.HomeID != "" {
		tm.UCLStage = "GRAND_FINAL"
		return
	}
	if len(tm.UCLSemiFinals) > 0 {
		legs1 := 0
		for _, id := range []string{"semi_1", "semi_2"} {
			if f := tm.uclLeg(id, 1); f != nil && f.Status == "finished" {
				legs1++
			}
		}
		if legs1 >= 2 {
			tm.UCLStage = "SEMI_FINALS_LEG2"
		} else {
			tm.UCLStage = "SEMI_FINALS_LEG1"
		}
		return
	}
	if len(tm.UCLQuarterFinals) > 0 {
		legs1 := 0
		for i := range tm.UCLFixtures {
			f := &tm.UCLFixtures[i]
			if len(f.TieID) >= 3 && f.TieID[:3] == "qf_" && f.Status == "finished" && f.Leg == 1 {
				legs1++
			}
		}
		if legs1 >= 4 {
			tm.UCLStage = "QUARTER_FINALS_LEG2"
		} else {
			tm.UCLStage = "QUARTER_FINALS_LEG1"
		}
		return
	}
	tm.UCLStage = "GROUP_STAGE"
}

func (tm *TournamentManager) maybeAdvanceUCL() string {
	tm.syncUCLBracket()
	if len(tm.UCLQuarterFinals) == 0 {
		groupGames := 0
		done := 0
		for i := range tm.UCLFixtures {
			f := &tm.UCLFixtures[i]
			if len(f.Stage) >= 5 && f.Stage[:5] == "Group" {
				groupGames++
				if f.Status == "finished" {
					done++
				}
			}
		}
		if groupGames > 0 && groupGames == done {
			msg := tm.drawUCLQuarterfinals()
			tm.refreshUCLStage()
			return msg
		}
	} else if len(tm.UCLSemiFinals) == 0 {
		all := true
		for _, t := range tm.UCLQuarterFinals {
			if t.WinnerID == "" {
				all = false
				break
			}
		}
		if all {
			msg := tm.drawUCLSemifinals()
			tm.refreshUCLStage()
			return msg
		}
	} else if tm.UCLFinal.HomeID == "" {
		all := true
		for _, t := range tm.UCLSemiFinals {
			if t.WinnerID == "" {
				all = false
				break
			}
		}
		if all {
			msg := tm.drawUCLFinal()
			tm.refreshUCLStage()
			return msg
		}
	} else if tm.UCLChampionID == "" {
		for i := range tm.UCLFixtures {
			f := &tm.UCLFixtures[i]
			if f.Stage == "Final" && f.Status == "finished" {
				tm.syncUCLBracket()
				tm.refreshUCLStage()
				if champ := tm.Clubs[tm.UCLChampionID]; champ != nil {
					return "UCL CHAMPIONS: " + champ.ClubName + " lift the Champions Cup!"
				}
			}
		}
	}
	tm.refreshUCLStage()
	return ""
}

func (tm *TournamentManager) resolveSCTie(tie *CupTie, tieID string) {
	fx := tm.scTieFixture(tieID)
	if fx == nil || fx.Status != "finished" || tie.HomeID == "" || fx.HomeGoals == nil || fx.AwayGoals == nil {
		return
	}
	tie.Leg1 = []int{*fx.HomeGoals, *fx.AwayGoals}
	if *fx.HomeGoals != *fx.AwayGoals {
		if *fx.HomeGoals > *fx.AwayGoals {
			tie.WinnerID = tie.HomeID
		} else {
			tie.WinnerID = tie.AwayID
		}
		tie.DecidedBy = fx.DecidedBy
		return
	}
	if len(fx.Penalties) >= 2 {
		if fx.Penalties[0] > fx.Penalties[1] {
			tie.WinnerID = tie.HomeID
		} else {
			tie.WinnerID = tie.AwayID
		}
		tie.DecidedBy = "penalties"
		tie.Penalties = append([]int(nil), fx.Penalties...)
	} else {
		tie.WinnerID = tie.HomeID
	}
}

func (tm *TournamentManager) maybeAdvanceSuperCup() string {
	for k, t := range tm.SuperCupPlayIn {
		tt := t
		tm.resolveSCTie(&tt, k)
		tm.SuperCupPlayIn[k] = tt
	}
	for k, t := range tm.SuperCupQuarterFinals {
		tt := t
		tm.resolveSCTie(&tt, k)
		tm.SuperCupQuarterFinals[k] = tt
	}
	for k, t := range tm.SuperCupSemiFinals {
		tt := t
		tm.resolveSCTie(&tt, k)
		tm.SuperCupSemiFinals[k] = tt
	}

	if len(tm.SuperCupQuarterFinals) == 0 {
		if len(tm.SuperCupPlayIn) == 0 {
			return ""
		}
		for _, t := range tm.SuperCupPlayIn {
			if t.WinnerID == "" {
				return ""
			}
		}
		if len(tm.SuperCupByes) < 4 {
			return ""
		}
		winners := make([]*models.Club, 4)
		for i := 1; i <= 4; i++ {
			winners[i-1] = tm.Clubs[tm.SuperCupPlayIn[fmtSCPI(i)].WinnerID]
			if winners[i-1] == nil {
				return ""
			}
		}
		draw := [][2]*models.Club{
			{tm.SuperCupByes[0], winners[3]},
			{tm.SuperCupByes[1], winners[2]},
			{tm.SuperCupByes[2], winners[1]},
			{tm.SuperCupByes[3], winners[0]},
		}
		week := tm.cupWeek(SuperCupWeeks["qf"])
		tm.SuperCupQuarterFinals = map[string]CupTie{}
		for i, pair := range draw {
			key := fmtSCQF(i + 1)
			tm.SuperCupQuarterFinals[key] = emptyTie(pair[0], pair[1])
			tm.makeSCFixture(key, "Quarter-final", week, pair[0], pair[1])
		}
		tm.SuperCupStage = "QUARTER_FINALS"
		return "Super Cup quarter-finals are set. Seeds 1–4 enter the draw."
	}
	if len(tm.SuperCupSemiFinals) == 0 {
		for _, t := range tm.SuperCupQuarterFinals {
			if t.WinnerID == "" {
				return ""
			}
		}
		w := make([]*models.Club, 4)
		for i := 1; i <= 4; i++ {
			w[i-1] = tm.Clubs[tm.SuperCupQuarterFinals[fmtSCQF(i)].WinnerID]
			if w[i-1] == nil {
				return ""
			}
		}
		week := tm.cupWeek(SuperCupWeeks["sf"])
		tm.SuperCupSemiFinals = map[string]CupTie{
			"sc_sf_1": emptyTie(w[0], w[1]),
			"sc_sf_2": emptyTie(w[2], w[3]),
		}
		tm.makeSCFixture("sc_sf_1", "Semi-final", week, w[0], w[1])
		tm.makeSCFixture("sc_sf_2", "Semi-final", week, w[2], w[3])
		tm.SuperCupStage = "SEMI_FINALS"
		return "Super Cup semi-finals are set."
	}
	if tm.SuperCupFinal.HomeID == "" {
		for _, t := range tm.SuperCupSemiFinals {
			if t.WinnerID == "" {
				return ""
			}
		}
		t1 := tm.Clubs[tm.SuperCupSemiFinals["sc_sf_1"].WinnerID]
		t2 := tm.Clubs[tm.SuperCupSemiFinals["sc_sf_2"].WinnerID]
		if t1 == nil || t2 == nil {
			return ""
		}
		tm.SuperCupFinal = emptyTie(t1, t2)
		tm.makeSCFixture("sc_final", "Final", tm.cupWeek(SuperCupWeeks["final"]), t1, t2)
		tm.SuperCupStage = "FINAL"
		return "Super Cup final: " + t1.ClubName + " vs " + t2.ClubName + "."
	}
	if tm.SuperCupChampionID == "" {
		final := tm.scTieFixture("sc_final")
		if final == nil || final.Status != "finished" || final.HomeGoals == nil || final.AwayGoals == nil {
			return ""
		}
		t1, t2 := tm.SuperCupFinal.HomeID, tm.SuperCupFinal.AwayID
		tm.SuperCupFinal.Leg1 = []int{*final.HomeGoals, *final.AwayGoals}
		tm.SuperCupFinal.DecidedBy = final.DecidedBy
		tm.SuperCupFinal.Penalties = append([]int(nil), final.Penalties...)
		var champID string
		if *final.HomeGoals != *final.AwayGoals {
			if *final.HomeGoals > *final.AwayGoals {
				champID = t1
			} else {
				champID = t2
			}
		} else if len(final.Penalties) >= 2 {
			if final.Penalties[0] > final.Penalties[1] {
				champID = t1
			} else {
				champID = t2
			}
		} else {
			champID = t1
		}
		tm.SuperCupFinal.WinnerID = champID
		tm.SuperCupChampionID = champID
		tm.SuperCupStage = "CROWNED"
		if champ := tm.Clubs[champID]; champ != nil {
			return champ.ClubName + " lift the Super Cup."
		}
	}
	return ""
}

func joinShort(names []string) string {
	out := ""
	for i, n := range names {
		if i > 0 {
			out += ", "
		}
		out += n
	}
	return out
}

func fmtSCPI(i int) string { return "sc_pi_" + itoa(i) }
func fmtSCQF(i int) string { return "sc_qf_" + itoa(i) }

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [8]byte
	n := len(b)
	for i > 0 {
		n--
		b[n] = byte('0' + i%10)
		i /= 10
	}
	return string(b[n:])
}
