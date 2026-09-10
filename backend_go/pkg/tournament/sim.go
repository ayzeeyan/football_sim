package tournament

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"

	"football_sim/pkg/matchengine"
	"football_sim/pkg/matchreport"
	"football_sim/pkg/models"
)

func (tm *TournamentManager) findFixtureUnlocked(id string) *Fixture {
	for i := range tm.Fixtures {
		if tm.Fixtures[i].FixtureID == id {
			return &tm.Fixtures[i]
		}
	}
	for i := range tm.UCLFixtures {
		if tm.UCLFixtures[i].FixtureID == id {
			return &tm.UCLFixtures[i]
		}
	}
	for i := range tm.SuperCupFixtures {
		if tm.SuperCupFixtures[i].FixtureID == id {
			return &tm.SuperCupFixtures[i]
		}
	}
	return nil
}

// FindFixture looks up a league or cup fixture by ID.
func (tm *TournamentManager) FindFixture(id string) *Fixture {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.findFixtureUnlocked(id)
}

func (tm *TournamentManager) leagueFixturesUnlocked(mw int) []Fixture {
	var out []Fixture
	for _, f := range tm.Fixtures {
		if f.Matchweek == mw {
			out = append(out, f)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].FixtureID < out[j].FixtureID })
	return out
}

func (tm *TournamentManager) slateUnlocked(mw int) []*Fixture {
	var league []*Fixture
	for i := range tm.Fixtures {
		if tm.Fixtures[i].Matchweek == mw {
			league = append(league, &tm.Fixtures[i])
		}
	}
	sort.Slice(league, func(i, j int) bool { return league[i].FixtureID < league[j].FixtureID })

	var cups []*Fixture
	for i := range tm.UCLFixtures {
		f := &tm.UCLFixtures[i]
		if f.Matchweek == mw {
			cups = append(cups, f)
		}
	}
	for i := range tm.SuperCupFixtures {
		f := &tm.SuperCupFixtures[i]
		if f.Matchweek == mw {
			cups = append(cups, f)
		}
	}
	view := tm.CurrentMatchweek
	if view > tm.MaxMatchweeks {
		view = tm.MaxMatchweeks
	}
	if mw == view {
		for i := range tm.UCLFixtures {
			f := &tm.UCLFixtures[i]
			if f.Status == "scheduled" && f.Matchweek < mw {
				cups = append(cups, f)
			}
		}
		for i := range tm.SuperCupFixtures {
			f := &tm.SuperCupFixtures[i]
			if f.Status == "scheduled" && f.Matchweek < mw {
				cups = append(cups, f)
			}
		}
	}
	sort.Slice(cups, func(i, j int) bool {
		if cups[i].Matchweek != cups[j].Matchweek {
			return cups[i].Matchweek < cups[j].Matchweek
		}
		return cups[i].FixtureID < cups[j].FixtureID
	})
	seen := map[string]bool{}
	var slate []*Fixture
	for _, f := range append(league, cups...) {
		if seen[f.FixtureID] {
			continue
		}
		seen[f.FixtureID] = true
		slate = append(slate, f)
	}
	return slate
}

// GetSlate returns league + cup fixtures for a matchweek (overdue cups hitch a ride).
func (tm *TournamentManager) GetSlate(mw int) []Fixture {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	ptrs := tm.slateUnlocked(mw)
	out := make([]Fixture, len(ptrs))
	for i, f := range ptrs {
		out[i] = *f
	}
	return out
}

// PendingUCL returns scheduled Champions Cup fixture IDs up to the current week.
func (tm *TournamentManager) PendingUCL() []string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.pendingUCLUnlocked()
}

func (tm *TournamentManager) pendingUCLUnlocked() []string {
	mw := tm.CurrentMatchweek
	var ids []string
	for i := range tm.UCLFixtures {
		f := &tm.UCLFixtures[i]
		if f.Status == "scheduled" && f.Matchweek <= mw {
			ids = append(ids, f.FixtureID)
		}
	}
	sort.Strings(ids)
	return ids
}

func (tm *TournamentManager) weatherUnlocked(mw int) string {
	if tm.MatchweekWeather == nil {
		tm.MatchweekWeather = map[int]string{}
	}
	if w, ok := tm.MatchweekWeather[mw]; ok && w != "" {
		return w
	}
	opts := WeatherOptions
	w := opts[mw%len(opts)]
	if tm.RNG != nil {
		w = opts[tm.RNG.Intn(len(opts))]
	}
	tm.MatchweekWeather[mw] = w
	return w
}

// ResolveWeather returns the fixture's recorded weather, then the matchweek
// climate table, then dry default. GET serialization uses this so a blank
// fixture still shows rain without consuming RNG.
func (tm *TournamentManager) ResolveWeather(f *Fixture) string {
	if f != nil && strings.TrimSpace(f.Weather) != "" {
		return f.Weather
	}
	mw := 1
	if f != nil && f.Matchweek > 0 {
		mw = f.Matchweek
	}
	if tm != nil && tm.MatchweekWeather != nil {
		if w := tm.MatchweekWeather[mw]; strings.TrimSpace(w) != "" {
			return w
		}
	}
	return "clear"
}

// EnsureFixtureWeather writes climate onto a blank fixture so live and instant
// play the same weather and later reports keep it.
func (tm *TournamentManager) EnsureFixtureWeather(f *Fixture) string {
	if f == nil {
		return "clear"
	}
	if strings.TrimSpace(f.Weather) != "" {
		return f.Weather
	}
	w := "clear"
	if tm != nil {
		w = tm.weatherUnlocked(f.Matchweek)
		if strings.TrimSpace(w) == "" {
			w = "clear"
		}
	}
	f.Weather = w
	return w
}

func (tm *TournamentManager) derbyHeatUnlocked(name string) int {
	if name == "" {
		return 0
	}
	if tm.DerbyHeat == nil {
		return 50
	}
	if v, ok := tm.DerbyHeat[name]; ok {
		return v
	}
	return 50
}

// SimulateFixture instantly simulates one scheduled fixture. Finished games stay locked.
func (tm *TournamentManager) SimulateFixture(fixtureID string) map[string]interface{} {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return tm.simulateFixtureUnlocked(fixtureID)
}

// CommitLiveFixture records a completed live engine result against the
// scheduled fixture for the current matchweek. Finished fixtures are ignored
// so a repeated full-time tick cannot replay the result. Callers that already
// selected a fixture should use CommitLiveFixtureByID so a rollover or a
// same-pair fixture in another competition cannot change the target.
func (tm *TournamentManager) CommitLiveFixture(homeID, awayID string, engine *matchengine.LiveMatchEngine) map[string]interface{} {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return tm.commitLiveFixtureUnlocked(homeID, awayID, engine)
}

// CommitLiveFixtureByID records a completed live engine result against the
// exact selected fixture. The fixture must still be scheduled and its clubs
// must match the engine; no current-slate or pair fallback is attempted.
func (tm *TournamentManager) CommitLiveFixtureByID(fixtureID string, engine *matchengine.LiveMatchEngine) map[string]interface{} {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return tm.commitLiveFixtureByIDUnlocked(fixtureID, engine)
}

func (tm *TournamentManager) commitLiveFixtureByIDUnlocked(fixtureID string, engine *matchengine.LiveMatchEngine) map[string]interface{} {
	if fixtureID == "" {
		return map[string]interface{}{"status": "ignored", "recorded": false, "reason": "missing-fixture-id", "terminal": true}
	}
	fixture := tm.findFixtureUnlocked(fixtureID)
	if fixture == nil {
		return map[string]interface{}{"status": "ignored", "recorded": false, "reason": "fixture-not-found", "fixture_id": fixtureID, "terminal": true}
	}
	if fixture.Status != "scheduled" {
		return map[string]interface{}{"status": "ignored", "recorded": false, "reason": "fixture-not-scheduled", "fixture_id": fixture.FixtureID, "terminal": true}
	}
	return tm.commitLiveFixtureOnFixtureUnlocked(fixture, engine)
}

func (tm *TournamentManager) commitLiveFixtureUnlocked(homeID, awayID string, engine *matchengine.LiveMatchEngine) map[string]interface{} {
	if engine == nil {
		return map[string]interface{}{"status": "ignored", "recorded": false, "reason": "missing-engine", "terminal": false}
	}

	view := tm.CurrentMatchweek
	if view > tm.MaxMatchweeks {
		view = tm.MaxMatchweeks
	}
	var fixture *Fixture
	for _, f := range tm.slateUnlocked(view) {
		if f.Status == "scheduled" && f.HomeID == homeID && f.AwayID == awayID {
			fixture = f
			break
		}
	}
	if fixture == nil {
		return map[string]interface{}{"status": "ignored", "recorded": false, "reason": "not-a-league-fixture", "terminal": true}
	}
	return tm.commitLiveFixtureOnFixtureUnlocked(fixture, engine)
}

func (tm *TournamentManager) commitLiveFixtureOnFixtureUnlocked(fixture *Fixture, engine *matchengine.LiveMatchEngine) map[string]interface{} {
	if fixture == nil {
		return map[string]interface{}{"status": "ignored", "recorded": false, "reason": "fixture-not-found", "terminal": true}
	}
	if fixture.Status != "scheduled" {
		return map[string]interface{}{"status": "ignored", "recorded": false, "reason": "fixture-not-scheduled", "fixture_id": fixture.FixtureID, "terminal": true}
	}
	if engine == nil {
		return map[string]interface{}{"status": "ignored", "recorded": false, "reason": "missing-engine", "fixture_id": fixture.FixtureID, "terminal": false}
	}
	if engine.HomeClub == nil || engine.AwayClub == nil || engine.HomeClub.ClubID != fixture.HomeID || engine.AwayClub.ClubID != fixture.AwayID {
		return map[string]interface{}{"status": "error", "recorded": false, "reason": "fixture-club-mismatch", "fixture_id": fixture.FixtureID, "terminal": true}
	}

	home := tm.Clubs[fixture.HomeID]
	away := tm.Clubs[fixture.AwayID]
	if home == nil || away == nil {
		return map[string]interface{}{"status": "error", "recorded": false, "reason": "club-not-found", "message": "Club not found.", "fixture_id": fixture.FixtureID, "terminal": false}
	}
	if blocked := tm.uclLegBlocked(fixture); blocked != "" {
		return map[string]interface{}{"status": "error", "recorded": false, "reason": blocked, "message": blocked, "fixture_id": fixture.FixtureID, "terminal": false}
	}

	weather := fixture.Weather
	if weather == "" {
		weather = engine.Weather
		if weather == "" {
			weather = tm.EnsureFixtureWeather(fixture)
		} else {
			fixture.Weather = weather
		}
	}
	engine.SetFixtureWeather(weather)
	payload := engine.BuildLivePayload()
	tm.applyKnockoutDecider(fixture, home, away, &payload)
	assembled := matchreport.AssembleReport(payload, "live", tm.RNG)
	tm.applyFinishedFixture(fixture, home, away, &assembled, "live")

	cupEvent := ""
	if fixture.Competition == "ucl" {
		cupEvent = tm.maybeAdvanceUCL()
	} else if fixture.Competition == "super-cup" {
		cupEvent = tm.maybeAdvanceSuperCup()
	}
	if cupEvent != "" {
		tm.PushInbox("cup", strings.TrimRight(cupEvent, "."), cupEvent, fixture.Matchweek, []string{home.ClubID, away.ClubID}, "", fixture.FixtureID)
	}

	rolled := tm.maybeRolloverUnlocked()
	champ, _ := rolled["champion"].(string)
	return map[string]interface{}{
		"status":      "success",
		"recorded":    true,
		"fixture_id":  fixture.FixtureID,
		"ucl_event":   nilIfEmpty(cupEvent),
		"rolled_over": rolled["rolled"],
		"is_finished": rolled["is_finished"],
		"champion":    champ,
		"terminal":    true,
	}
}

func (tm *TournamentManager) simulateFixtureUnlocked(fixtureID string) map[string]interface{} {
	return tm.simulateFixtureWithRNGUnlocked(fixtureID, tm.RNG)
}

// simulateFixtureWithRNGUnlocked is the single-fixture entry point with an
// explicit random stream. The pooled slate path reuses the same
// compute/apply halves with per-fixture streams.
func (tm *TournamentManager) simulateFixtureWithRNGUnlocked(fixtureID string, rng *rand.Rand) map[string]interface{} {
	f := tm.findFixtureUnlocked(fixtureID)
	if f == nil {
		return map[string]interface{}{"status": "error", "message": "Fixture not found."}
	}
	if f.Status == "finished" {
		return map[string]interface{}{"status": "error", "message": "That result already stands and cannot be replayed."}
	}
	if f.Matchweek > tm.CurrentMatchweek && tm.CurrentMatchweek > 0 {
		return map[string]interface{}{"status": "error", "message": "That matchweek has not opened yet."}
	}
	if blocked := tm.uclLegBlocked(f); blocked != "" {
		return map[string]interface{}{"status": "error", "message": blocked}
	}
	computed, errMsg := tm.computeSlateFixture(f, rng)
	if errMsg != "" {
		return map[string]interface{}{"status": "error", "message": errMsg}
	}
	return tm.applySlateFixture(f, computed)
}

func playersFromRows(club *models.Club, rows []matchreport.MatchPlayerRow) []*models.Player {
	if club == nil {
		return nil
	}
	idx := map[string]*models.Player{}
	for _, p := range club.Squad {
		idx[p.PlayerID] = p
	}
	var out []*models.Player
	for _, r := range rows {
		if p := idx[r.PlayerID]; p != nil {
			out = append(out, p)
		}
	}
	return out
}

func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func (tm *TournamentManager) applyFinishedFixture(f *Fixture, home, away *models.Club, report *matchreport.MatchReport, method string) {
	f.Status = "finished"
	f.Method = method
	hg, ag := report.HomeGoals, report.AwayGoals
	f.HomeGoals = &hg
	f.AwayGoals = &ag
	f.Report = report
	f.Referee = report.Referee
	f.Weather = report.Weather
	if report.DecidedBy != nil {
		f.DecidedBy = *report.DecidedBy
	}
	if pens, ok := report.Penalties.([]int); ok {
		f.Penalties = pens
	}

	comp := f.Competition
	if comp == "" {
		comp = "super-league"
	}
	if comp == "super-league" {
		home.UpdateResult(hg, ag)
		away.UpdateResult(ag, hg)
	} else {
		if comp == "ucl" && strings.HasPrefix(f.Stage, "Group") {
			if rec := tm.UCLRecords[home.ClubID]; rec != nil {
				rec.UpdateResult(hg, ag)
			}
			if rec := tm.UCLRecords[away.ClubID]; rec != nil {
				rec.UpdateResult(ag, hg)
			}
		}
		if hg > ag {
			home.UpdateMorale("W")
			away.UpdateMorale("L")
		} else if hg == ag {
			home.UpdateMorale("D")
			away.UpdateMorale("D")
		} else {
			home.UpdateMorale("L")
			away.UpdateMorale("W")
		}
	}

	if tm.MoraleStoryStatus == nil {
		tm.MoraleStoryStatus = map[string]string{}
	}
	for _, club := range []*models.Club{home, away} {
		prev := tm.MoraleStoryStatus[club.ClubID]
		if prev == "" {
			prev = "normal"
		}
		if club.Morale < 35 && prev != "low" {
			tm.MoraleStoryStatus[club.ClubID] = "low"
			tm.PushInbox("club",
				fmt.Sprintf("Dressing room tensions at %s", club.ClubName),
				fmt.Sprintf("Dressing room tensions at %s — sources report senior players frustrated with recent results.", club.ClubName),
				f.Matchweek, []string{club.ClubID}, "", f.FixtureID)
		} else if club.Morale > 90 && prev != "high" {
			tm.MoraleStoryStatus[club.ClubID] = "high"
			tm.PushInbox("club",
				fmt.Sprintf("Team spirit flying high at %s", club.ClubName),
				fmt.Sprintf("%s riding a wave of confidence — team spirit at an all-time high.", club.ClubName),
				f.Matchweek, []string{club.ClubID}, "", f.FixtureID)
		} else if club.Morale >= 35 && club.Morale <= 90 {
			tm.MoraleStoryStatus[club.ClubID] = "normal"
		}
	}

	if f.DerbyName != "" {
		delta := 0
		margin := int(math.Abs(float64(hg - ag)))
		hasRed := false
		for _, e := range report.Events {
			if e.Type == "red" || e.Type == "red_card" {
				hasRed = true
				break
			}
		}
		if margin == 1 {
			delta += 5
		}
		if hasRed {
			delta += 10
		}
		if margin >= 3 {
			delta += 3
		}
		pair := f.HomeID + "_" + f.AwayID
		rev := f.AwayID + "_" + f.HomeID
		cur := 50
		if v, ok := tm.DerbyHeat[f.DerbyName]; ok {
			cur = v
		} else if v, ok := tm.DerbyHeat[pair]; ok {
			cur = v
		} else if v, ok := tm.DerbyHeat[rev]; ok {
			cur = v
		}
		if delta > 0 {
			cur += delta
			if cur > 100 {
				cur = 100
			}
		}
		if tm.DerbyHeat == nil {
			tm.DerbyHeat = map[string]int{}
		}
		tm.DerbyHeat[f.DerbyName] = cur
		tm.DerbyHeat[pair] = cur
		tm.DerbyHeat[rev] = cur
		if tm.DerbiesPlayedThisMW == nil {
			tm.DerbiesPlayedThisMW = map[string]bool{}
		}
		tm.DerbiesPlayedThisMW[f.DerbyName] = true
		tm.DerbiesPlayedThisMW[pair] = true
		tm.DerbiesPlayedThisMW[rev] = true
	}

	growthEvents := tm.ApplyMatchReport(home, away, report, f.Matchweek, f.FixtureID)
	if len(growthEvents) > 0 {
		tm.GrowthNotifications = append(growthEvents, tm.GrowthNotifications...)
		if len(tm.GrowthNotifications) > 8 {
			tm.GrowthNotifications = tm.GrowthNotifications[:8]
		}
	}
	tm.refreshRecentResultsUnlocked()
	tm.inboxMatch(f, home, away, report)
}

func (tm *TournamentManager) inboxMatch(f *Fixture, home, away *models.Club, report *matchreport.MatchReport) {
	if f == nil || home == nil || away == nil || report == nil {
		return
	}
	headline := fmt.Sprintf("%s %d-%d %s", home.ShortName, report.HomeGoals, report.AwayGoals, away.ShortName)
	body := fmt.Sprintf("%s %s %d-%d %s %s.", home.ClubName, home.ShortName, report.HomeGoals, report.AwayGoals, away.ShortName, away.ClubName)
	if report.MOTM != nil {
		body += " MOTM: " + report.MOTM.FullName + "."
	}
	cat := "match"
	if f.Competition == "ucl" {
		cat = "cup"
	} else if f.Competition == "super-cup" {
		cat = "cup"
	}
	tm.PushInbox(cat, headline, body, f.Matchweek, []string{home.ClubID, away.ClubID}, "", f.FixtureID)
}

func (tm *TournamentManager) refreshRecentResultsUnlocked() {
	mw := tm.CurrentMatchweek
	if mw > tm.MaxMatchweeks {
		mw = tm.MaxMatchweeks
	}
	for mw > 1 {
		any := false
		for _, f := range tm.leagueFixturesUnlocked(mw) {
			if f.Status == "finished" {
				any = true
				break
			}
		}
		if any {
			break
		}
		mw--
	}
	var lines []string
	for _, f := range tm.leagueFixturesUnlocked(mw) {
		if f.Status != "finished" || f.HomeGoals == nil || f.AwayGoals == nil {
			continue
		}
		home := tm.Clubs[f.HomeID]
		away := tm.Clubs[f.AwayID]
		if home == nil || away == nil {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s %d - %d %s", home.ShortName, *f.HomeGoals, *f.AwayGoals, away.ShortName))
	}
	tm.RecentResults = lines
}

func (tm *TournamentManager) maybeRolloverUnlocked() map[string]interface{} {
	if tm.CurrentMatchweek > tm.MaxMatchweeks {
		tm.SeasonPhase = "transfer_window"
		return map[string]interface{}{"rolled": false, "is_finished": true, "champion": tm.championNameUnlocked()}
	}
	pending := false
	for _, f := range tm.leagueFixturesUnlocked(tm.CurrentMatchweek) {
		if f.Status == "scheduled" {
			pending = true
			break
		}
	}
	if pending {
		return map[string]interface{}{"rolled": false, "is_finished": false}
	}

	completed := tm.CurrentMatchweek
	tm.runWeeklyTicks(completed)
	tm.CurrentMatchweek++
	tm.refreshRecentResultsUnlocked()

	isFinished := tm.CurrentMatchweek > tm.MaxMatchweeks
	if isFinished {
		tm.SeasonPhase = "transfer_window"
	}
	champ := tm.championNameUnlocked()
	if isFinished && champ != "" {
		standings := tm.standingsUnlocked()
		c := standings[0]
		tm.PushInbox("honour",
			fmt.Sprintf("%s are Super League champions", c.ClubName),
			fmt.Sprintf("%d points · %d wins · %+d goal difference. The window opens.", c.Points, c.Won, c.GoalDifference),
			completed, []string{c.ClubID}, "", "")
	}
	return map[string]interface{}{"rolled": true, "is_finished": isFinished, "champion": champIf(isFinished, champ)}
}

func champIf(ok bool, name string) interface{} {
	if !ok || name == "" {
		return nil
	}
	return name
}

func (tm *TournamentManager) championNameUnlocked() string {
	s := tm.standingsUnlocked()
	if len(s) == 0 {
		return ""
	}
	return s[0].ClubName
}

func (tm *TournamentManager) standingsUnlocked() []*models.Club {
	clubsCopy := make([]*models.Club, len(tm.ClubsList))
	copy(clubsCopy, tm.ClubsList)
	models.SortClubs(clubsCopy)
	return clubsCopy
}

// SimulateRemaining plays every scheduled game on this week's slate.
func (tm *TournamentManager) SimulateRemaining() map[string]interface{} {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return tm.simulateRemainingUnlocked()
}

func (tm *TournamentManager) simulateRemainingUnlocked() map[string]interface{} {
	mw, ids := tm.slateBatchUnlocked()
	// One base draw anchors every fixture's stream, so each sim is
	// order-independent and the pooled slate matches a serial one.
	var base int64 = 1
	if tm.RNG != nil {
		base = tm.RNG.Int63()
	}
	// Resolve climate serially in slate order (mutates the climate table).
	for _, id := range ids {
		if fx := tm.findFixtureUnlocked(id); fx != nil && fx.Status != "finished" {
			tm.EnsureFixtureWeather(fx)
		}
	}
	computed := tm.computeSlateWaves(groupSlateWaves(tm, ids), base)
	// Serial apply in slate order: tables, cup advancement, inbox, and
	// rollover resolve exactly as the legacy loop did.
	played, skipped := 0, 0
	for _, id := range ids {
		fx := tm.findFixtureUnlocked(id)
		if fx == nil || fx.Status == "finished" {
			continue
		}
		if fx.Matchweek > tm.CurrentMatchweek && tm.CurrentMatchweek > 0 {
			skipped++
			continue
		}
		if blocked := tm.uclLegBlocked(fx); blocked != "" {
			skipped++
			continue
		}
		res, ok := computed[id]
		if !ok {
			skipped++
			continue
		}
		if out := tm.applySlateFixture(fx, res); out["status"] == "success" {
			played++
		} else {
			skipped++
		}
	}
	return map[string]interface{}{
		"status":          "success",
		"played":          played,
		"skipped":         skipped,
		"simulated_count": played,
		"matchweek":       mw,
		"next_matchweek":  tm.CurrentMatchweek,
		"is_finished":     tm.CurrentMatchweek > tm.MaxMatchweeks,
		"champion":        champIf(tm.CurrentMatchweek > tm.MaxMatchweeks, tm.championNameUnlocked()),
	}
}
