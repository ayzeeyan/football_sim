package tournament

// Gen2ManagerSecuritySnapshot exposes the simulator's existing manager-security
// assessment to the offline dataset generator. It is read-only and does not
// alter gameplay state, RNG, or production decisions.
func (tm *TournamentManager) Gen2ManagerSecuritySnapshot(clubID string, completedMW int) (status, reason string, sackCandidate bool) {
	if tm == nil {
		return "", "", false
	}
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	club := tm.Clubs[clubID]
	if club == nil {
		return "", "", false
	}
	return tm.managerSecurityUnlocked(club, completedMW)
}
