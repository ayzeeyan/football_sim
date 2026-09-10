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
	if tm.SeasonPhase != "transfer_window" {
		return map[string]interface{}{
			"status":            "error",
			"message":           "Season transition is not pending.",
			"season_name":       tm.SeasonName,
			"current_matchweek": tm.CurrentMatchweek,
		}
	}
	if tm.TransferEngine != nil && tm.TransferEngine.CurrentWeek <= TransferWindowWeeks {
		return map[string]interface{}{
			"status":            "error",
			"message":           "Season transition cannot finalize before all 12 transfer weeks are processed.",
			"season_name":       tm.SeasonName,
			"current_matchweek": tm.CurrentMatchweek,
		}
	}

	// Reputation changes once at the same guarded boundary that archives the
	// completed campaign. ResetNewSeason switches SeasonPhase back to season, so
	// a repeated finalization call cannot apply this mutation twice.
	tm.mu.Lock()
	tm.updateClubReputationsUnlocked()
	tm.mu.Unlock()
	return tm.ResetNewSeason()
}
