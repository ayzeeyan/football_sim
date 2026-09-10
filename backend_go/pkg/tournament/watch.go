package tournament

// Watch-one / sim-the-rest week (Feature 2).
//
// My club is a persisted favourite club id. The client watches that club's
// fixture live; at full time the rest of the slate is simulated instantly,
// excluding the already-committed live result. Same-week cup fixtures ride
// the same slate and are offered as jump targets.

// SetFavouriteClubID pins the watched club. Returns false for unknown clubs.
func (tm *TournamentManager) SetFavouriteClubID(clubID string) bool {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if _, ok := tm.Clubs[clubID]; !ok {
		return false
	}
	tm.FavouriteClubID = clubID
	return true
}

// FavouriteClub returns the persisted favourite club id (may be empty).
func (tm *TournamentManager) FavouriteClub() string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.FavouriteClubID
}

// WeekWatch resolves the favourite's fixture on the current slate plus any
// same-week cup fixtures available as jump targets. No simulation occurs.
func (tm *TournamentManager) WeekWatch() (favID string, fav *Fixture, cups []Fixture, mw int) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	mw = tm.CurrentMatchweek
	if mw > tm.MaxMatchweeks {
		mw = tm.MaxMatchweeks
	}
	if mw < 1 {
		mw = 1
	}
	favID = tm.FavouriteClubID
	slate := tm.slateUnlocked(mw)
	var favPtr *Fixture
	for _, f := range slate {
		if favID != "" && (f.HomeID == favID || f.AwayID == favID) {
			// Prefer the league fixture when the favourite has two (league + cup).
			if favPtr == nil || (favPtr.Competition != "super-league" && f.Competition == "super-league") {
				favPtr = f
			}
		}
	}
	if favPtr != nil {
		cp := *favPtr
		fav = &cp
	}
	for _, f := range slate {
		if f.Competition != "ucl" && f.Competition != "super-cup" {
			continue
		}
		if fav != nil && f.FixtureID == fav.FixtureID {
			continue
		}
		cups = append(cups, *f)
	}
	if cups == nil {
		cups = []Fixture{}
	}
	return favID, fav, cups, mw
}

// SimulateRemainingExcluding plays every scheduled game on this week's slate
// except excludeID (the watched live fixture). Finished fixtures are never
// replayed, so an already-committed live result cannot be simulated again.
func (tm *TournamentManager) SimulateRemainingExcluding(excludeID string) map[string]interface{} {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return tm.simulateRemainingExcludingUnlocked(excludeID)
}

func (tm *TournamentManager) simulateRemainingExcludingUnlocked(excludeID string) map[string]interface{} {
	mw, ids := tm.slateBatchUnlocked()
	filtered := make([]string, 0, len(ids))
	for _, id := range ids {
		if excludeID != "" && id == excludeID {
			continue
		}
		filtered = append(filtered, id)
	}
	var base int64 = 1
	if tm.RNG != nil {
		base = tm.RNG.Int63()
	}
	for _, id := range filtered {
		if fx := tm.findFixtureUnlocked(id); fx != nil && fx.Status != "finished" {
			tm.EnsureFixtureWeather(fx)
		}
	}
	computed := tm.computeSlateWaves(groupSlateWaves(tm, filtered), base)
	played, skipped := 0, 0
	for _, id := range filtered {
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
	if excludeID != "" {
		skipped++
	}
	return map[string]interface{}{
		"status":              "success",
		"played":              played,
		"skipped":             skipped,
		"simulated_count":     played,
		"matchweek":           mw,
		"next_matchweek":      tm.CurrentMatchweek,
		"is_finished":         tm.CurrentMatchweek > tm.MaxMatchweeks,
		"champion":            champIf(tm.CurrentMatchweek > tm.MaxMatchweeks, tm.championNameUnlocked()),
		"excluded_fixture_id": excludeID,
	}
}
