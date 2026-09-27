package tournament

// FinalizeSeasonTransition is the guarded mutation boundary for moving from a
// completed campaign/offseason into the next season. Callers should use this
// instead of invoking ResetNewSeason directly as part of autonomous
// progression. A repeated call after the transition has already completed is
// explicitly rejected without mutating the universe.
func (tm *TournamentManager) FinalizeSeasonTransition() map[string]interface{} {
	if tm == nil {
		return map[string]interface{}{
			"status":  "error",
			"message": "Cannot finalize season transition: tournament manager is unavailable.",
		}
	}
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if failure := tm.seasonTransitionErrorUnlocked(); failure != nil {
		return failure
	}
	return tm.resetNewSeasonUnlocked()
}

// seasonTransitionErrorUnlocked applies the lifecycle gates shared by every
// transition path. The caller must hold tm.mu.
func (tm *TournamentManager) seasonTransitionErrorUnlocked() map[string]interface{} {
	if tm.SeasonPhase != "transfer_window" {
		return map[string]interface{}{
			"status":            "error",
			"message":           "Season transition is not pending.",
			"season_name":       tm.SeasonName,
			"current_matchweek": tm.CurrentMatchweek,
		}
	}
	if tm.TransferEngine != nil && !tm.TransferEngine.HasCompletedSummerWindow() {
		return map[string]interface{}{
			"status":            "error",
			"message":           "Season transition cannot finalize before all 12 transfer weeks are processed.",
			"season_name":       tm.SeasonName,
			"current_matchweek": tm.CurrentMatchweek,
		}
	}

	// The normal server path applies reputation before opening Week 1 so budgets
	// use the completed campaign's stature. Keep this guarded fallback for
	// legacy/manual callers that finalize after the window without going through
	// that path. Incomplete result inputs must leave the campaign untouched.
	ready := tm.completedSeasonInputsAvailableUnlocked()
	if ready && tm.ReputationAppliedSeason != tm.SeasonName {
		ready = tm.applyCompletedSeasonReputationUnlocked()
	}
	markerReady := tm.ReputationAppliedSeason == tm.SeasonName
	if !ready || !markerReady {
		return map[string]interface{}{
			"status":            "error",
			"message":           "Season transition cannot finalize before all league and cup results are complete.",
			"season_name":       tm.SeasonName,
			"current_matchweek": tm.CurrentMatchweek,
		}
	}
	return nil
}
