package tournament

import (
	"hash/fnv"
	"math/rand"
	"sort"
	"strings"
	"sync"

	"football_sim/pkg/matchengine"
	"football_sim/pkg/matchreport"
	"football_sim/pkg/models"
)

// Bounded worker pool for SimulateRemaining (F4).
//
// A league night plus cups is a slate of simultaneous fixtures: the model has
// no staggered kickoff times, so the conflict rule is shared clubs. Fixtures
// that touch the same club (or an unknown fixture, defensively) never compute
// concurrently; club-disjoint fixtures compute in parallel with a per-fixture
// RNG. Application (tables, cup advancement, inbox, rollover) stays serial in
// slate order, so cup leg-order dependencies resolve exactly as before.
//
// Every fixture's random stream derives from one base draw plus its fixture
// ID, making each sim order-independent: a pooled slate matches a serial
// slate fixture-for-fixture.

// slatePoolSize bounds concurrent instant-match computations.
const slatePoolSize = 4

// slateComputed is the pure half of a fixture sim: report payload plus the
// assembled report, ready for serial application.
type slateComputed struct {
	fixtureID string
	homeID    string
	awayID    string
	payload   matchreport.InstantPayload
	assembled matchreport.MatchReport
}

// slateSeed derives a deterministic per-fixture RNG seed from one base draw
// and the fixture ID.
func slateSeed(base int64, fixtureID string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(fixtureID))
	return base ^ int64(h.Sum64())
}

// slateBatchUnlocked collects and orders the remaining slate: the current
// matchweek's unfinished fixtures across league and cups, league first.
// Shared by the pooled path and the serial reference. Caller must hold tm.mu.
func (tm *TournamentManager) slateBatchUnlocked() (int, []string) {
	mw := tm.CurrentMatchweek
	if mw > tm.MaxMatchweeks {
		mw = tm.MaxMatchweeks
	}
	batch := tm.slateUnlocked(mw)
	sort.SliceStable(batch, func(i, j int) bool {
		rank := func(c string) int {
			switch c {
			case "super-league":
				return 0
			case "super-cup":
				return 1
			case "ucl":
				return 2
			default:
				return 3
			}
		}
		ri, rj := rank(batch[i].Competition), rank(batch[j].Competition)
		if ri != rj {
			return ri < rj
		}
		if batch[i].Leg != batch[j].Leg {
			return batch[i].Leg < batch[j].Leg
		}
		if batch[i].Matchweek != batch[j].Matchweek {
			return batch[i].Matchweek < batch[j].Matchweek
		}
		return batch[i].FixtureID < batch[j].FixtureID
	})
	ids := make([]string, 0, len(batch))
	for _, f := range batch {
		if f.Status != "finished" {
			ids = append(ids, f.FixtureID)
		}
	}
	return mw, ids
}

// groupSlateWaves partitions fixture IDs into sequential waves. IDs in one
// wave never share a club, so they may compute concurrently; fixtures sharing
// a club land in different waves and stay serial. Order-preserving and
// deterministic: the concatenation of all waves is the input order.
func groupSlateWaves(tm *TournamentManager, ids []string) [][]string {
	var waves [][]string
	var waveClubs []map[string]bool
	for _, id := range ids {
		f := tm.findFixtureUnlocked(id)
		if f == nil {
			waves = append(waves, []string{id})
			waveClubs = append(waveClubs, map[string]bool{})
			continue
		}
		placed := false
		for w := range waves {
			if !waveClubs[w][f.HomeID] && !waveClubs[w][f.AwayID] {
				waves[w] = append(waves[w], id)
				waveClubs[w][f.HomeID] = true
				waveClubs[w][f.AwayID] = true
				placed = true
				break
			}
		}
		if !placed {
			waves = append(waves, []string{id})
			waveClubs = append(waveClubs, map[string]bool{f.HomeID: true, f.AwayID: true})
		}
	}
	return waves
}

// computeSlateFixture runs the pure half of simulateFixtureUnlocked: heat
// resolution, instant report simulation, XI reconstruction, knockout decider,
// and assembly. It reads only its own fixture plus read-only club views and
// consumes only its own RNG, so concurrent workers on disjoint clubs are
// safe. It returns a non-empty errMsg mirroring the legacy error text when
// the fixture cannot be simulated.
func (tm *TournamentManager) computeSlateFixture(f *Fixture, rng *rand.Rand) (slateComputed, string) {
	home := tm.Clubs[f.HomeID]
	away := tm.Clubs[f.AwayID]
	if home == nil || away == nil {
		return slateComputed{}, "Club not found."
	}

	heat := f.DerbyHeat
	if f.DerbyName != "" {
		heat = tm.derbyHeatUnlocked(f.DerbyName)
		f.DerbyHeat = heat
		f.IsHighHeatDerby = heat > 70
	}
	weather := tm.EnsureFixtureWeather(f)
	cfg := &matchengine.InstantMatchConfig{
		Weather:     weather,
		DerbyHeat:   heat,
		IsDerby:     f.DerbyName != "",
		Referee:     f.Referee,
		Competition: f.Competition,
		Matchweek:   f.Matchweek,
	}
	homeMgr := tm.Managers[f.HomeID]
	awayMgr := tm.Managers[f.AwayID]
	report := matchengine.SimulateInstantMatch(home, away, homeMgr, awayMgr, tm.GrowthEngine, cfg, rng)

	payload := matchreport.InstantPayload{
		HomeGoals:  report.HomeGoals,
		AwayGoals:  report.AwayGoals,
		Events:     report.Events,
		HomeXI:     home.GetStartingEleven(models.FixtureContext(f.Competition, f.Matchweek)),
		AwayXI:     away.GetStartingEleven(models.FixtureContext(f.Competition, f.Matchweek)),
		HomeBench:  home.GetBench(nil, 7, models.FixtureContext(f.Competition, f.Matchweek)),
		AwayBench:  away.GetBench(nil, 7, models.FixtureContext(f.Competition, f.Matchweek)),
		Stats:      report.Stats,
		HTHome:     report.HTHome,
		HTAway:     report.HTAway,
		Attendance: report.Attendance,
		Referee:    report.Referee,
		Weather:    report.Weather,
		DecidedBy:  report.DecidedBy,
		Penalties:  report.Penalties,
		ShotMap:    report.ShotMap,
		Heatmap:    report.Heatmap,
		Press:      report.PressConference,
	}
	// Reconstruct XI from the report's kickoff lists when present.
	if len(report.HomeXI) > 0 {
		payload.HomeXI = playersFromRows(home, report.HomeXI)
		payload.AwayXI = playersFromRows(away, report.AwayXI)
		payload.HomeBench = playersFromRows(home, report.HomeBench)
		payload.AwayBench = playersFromRows(away, report.AwayBench)
	}
	tm.applyKnockoutDeciderWithRNG(f, home, away, &payload, rng)
	assembled := matchreport.AssembleReport(payload, "instant", rng)
	return slateComputed{
		fixtureID: f.FixtureID, homeID: f.HomeID, awayID: f.AwayID,
		payload: payload, assembled: assembled,
	}, ""
}

// computeSlateWaves runs every wave's fixtures concurrently (bounded by the
// pool) while waves themselves run sequentially, so same-club fixtures stay
// serial. Only the results map is shared, under its own mutex; tm is only
// read during compute.
func (tm *TournamentManager) computeSlateWaves(waves [][]string, base int64) map[string]slateComputed {
	computed := make(map[string]slateComputed)
	var computedMu sync.Mutex
	for _, wave := range waves {
		sem := make(chan struct{}, slatePoolSize)
		var wg sync.WaitGroup
		for _, id := range wave {
			f := tm.findFixtureUnlocked(id)
			if f == nil || f.Status == "finished" {
				continue
			}
			rng := rand.New(rand.NewSource(slateSeed(base, id)))
			wg.Add(1)
			sem <- struct{}{}
			go func(fx *Fixture, r *rand.Rand) {
				defer wg.Done()
				defer func() { <-sem }()
				if res, errMsg := tm.computeSlateFixture(fx, r); errMsg == "" {
					computedMu.Lock()
					computed[fx.FixtureID] = res
					computedMu.Unlock()
				}
			}(f, rng)
		}
		wg.Wait()
	}
	return computed
}

// applySlateFixture runs the serial half of simulateFixtureUnlocked: table
// updates, cup advancement, inbox, and rollover. Must be called in slate
// order while holding tm.mu.
func (tm *TournamentManager) applySlateFixture(f *Fixture, computed slateComputed) map[string]interface{} {
	home := tm.Clubs[f.HomeID]
	away := tm.Clubs[f.AwayID]
	assembled := computed.assembled
	tm.applyFinishedFixture(f, home, away, &assembled, "instant")

	cupEvent := ""
	if f.Competition == "ucl" {
		cupEvent = tm.maybeAdvanceUCL()
	} else if f.Competition == "super-cup" {
		cupEvent = tm.maybeAdvanceSuperCup()
	}
	if cupEvent != "" {
		tm.PushInbox("cup", strings.TrimRight(cupEvent, "."), cupEvent, f.Matchweek, []string{home.ClubID, away.ClubID}, "", f.FixtureID)
	}
	rolled := tm.maybeRolloverUnlocked()
	champ, _ := rolled["champion"].(string)
	return map[string]interface{}{
		"status":      "success",
		"fixture_id":  f.FixtureID,
		"home_goals":  assembled.HomeGoals,
		"away_goals":  assembled.AwayGoals,
		"ucl_event":   nilIfEmpty(cupEvent),
		"rolled_over": rolled["rolled"],
		"is_finished": rolled["is_finished"],
		"champion":    champ,
	}
}
