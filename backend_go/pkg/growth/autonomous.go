package growth

// RollAutonomousTraining owns the probabilistic decision for an autonomous
// staff training session inside the development subsystem. Keeping this draw
// on GrowthEngine's RNG prevents unrelated tournament/admin randomness from
// changing a wonderkid's development path.
func (ge *GrowthEngine) RollAutonomousTraining(chance float64) bool {
	if ge == nil {
		return false
	}
	if chance <= 0 {
		return false
	}
	if chance >= 1 {
		return true
	}
	ge.mu.Lock()
	defer ge.mu.Unlock()
	if ge.rng == nil {
		return false
	}
	return ge.rng.Float64() <= chance
}
