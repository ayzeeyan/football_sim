package tournament

import (
	"sort"

	"football_sim/pkg/scouting"
)

// ScoutingShortlist builds the recruitment shortlist for one club: a
// deterministic, read-only ranking of players at other clubs. Candidates are
// gathered in sorted club order, tracked growth potentials are read through
// the growth engine, and the scouting package performs the final ordered
// selection.
func (tm *TournamentManager) ScoutingShortlist(clubID string, limit int) []scouting.Report {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	club := tm.Clubs[clubID]
	if club == nil {
		return nil
	}
	clubIDs := make([]string, 0, len(tm.Clubs))
	for id := range tm.Clubs {
		clubIDs = append(clubIDs, id)
	}
	sort.Strings(clubIDs)

	candidates := make([]scouting.Candidate, 0, 2048)
	for _, id := range clubIDs {
		other := tm.Clubs[id]
		if other == nil || id == clubID {
			continue
		}
		for _, p := range other.Squad {
			if p == nil {
				continue
			}
			potential, tracked := 0, false
			if tm.GrowthEngine != nil {
				potential, tracked = tm.GrowthEngine.PotentialFor(p.PlayerID)
			}
			candidates = append(candidates, scouting.Candidate{Player: p, Club: other, Potential: potential, Tracked: tracked})
		}
	}
	return scouting.Shortlist(clubID, candidates, limit)
}
