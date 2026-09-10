package tournament

import (
	"fmt"

	"football_sim/pkg/managers"
	"football_sim/pkg/models"
)

// boardHotSeatWeeks converts BoardPatience into a bounded persistence window.
// The performance/crisis detector is unchanged: patience only decides how
// many consecutive Hot Seat assessments the board tolerates before acting.
func boardHotSeatWeeks(club *models.Club) int {
	if club == nil {
		return 2
	}
	p := club.Identity.BoardPatience
	switch {
	case p <= 35:
		return 1
	case p >= 75:
		return 3
	default:
		return 2
	}
}

func (tm *TournamentManager) evaluateManagerTenureWithPatience(completedMW int) {
	if completedMW < 8 {
		return
	}
	if tm.ManagerConsecutiveHot == nil {
		tm.ManagerConsecutiveHot = map[string]int{}
	}
	if tm.ManagerLastChange == nil {
		tm.ManagerLastChange = map[string]int{}
	}
	for _, club := range tm.ClubsList {
		mgr := tm.Managers[club.ClubID]
		if mgr == nil {
			continue
		}
		status, reason, sackCandidate := tm.managerSecurityUnlocked(club, completedMW)
		mgr.JobSecurity = status
		if !sackCandidate {
			if status == "Safe" {
				tm.ManagerConsecutiveHot[club.ClubID] = 0
			}
			continue
		}
		tm.ManagerConsecutiveHot[club.ClubID]++
		required := boardHotSeatWeeks(club)
		if tm.ManagerConsecutiveHot[club.ClubID] < required {
			continue
		}
		if last := tm.ManagerLastChange[club.ClubID]; last > 0 && completedMW-last < 6 {
			continue
		}

		oldStyle := mgr.CanonicalStyle()
		old, next := managers.AppointManager(tm.Managers, club, tm.RNG)
		if old == nil || next == nil {
			continue
		}
		next.History = append([]managers.ManagerHistoryEntry(nil), old.History...)
		next.History = append(next.History, managers.ManagerHistoryEntry{
			ClubID: club.ClubID, ClubName: club.ClubName, ManagerName: old.Name, Style: oldStyle,
			AppointedSeason: old.AppointedSeason, AppointedMatchweek: old.AppointedMatchweek,
			DepartedSeason: tm.SeasonName, DepartedMatchweek: completedMW, Reason: reason,
		})
		next.AppointedSeason = tm.SeasonName
		next.AppointedMatchweek = completedMW + 1
		next.JobSecurity = "Safe"
		if tm.TransferEngine != nil && tm.TransferEngine.Managers != nil {
			tm.TransferEngine.Managers[club.ClubID] = next
		}
		entry := ManagerHistoryEntry{
			SeasonName: tm.SeasonName, Matchweek: completedMW, ClubID: club.ClubID, ClubName: club.ClubName,
			Action: "sacked", OldManager: old.Name, NewManager: next.Name, OldStyle: oldStyle,
			NewStyle: next.CanonicalStyle(), Reason: reason, JobSecurity: "Hot Seat",
		}
		tm.ManagerHistory = append(tm.ManagerHistory, entry)
		tm.ManagerLastChange[club.ClubID] = completedMW
		tm.ManagerConsecutiveHot[club.ClubID] = 0
		tm.PushInbox(
			"manager",
			fmt.Sprintf("BREAKING: %s sack %s; %s appointed", club.ClubName, old.Name, next.Name),
			fmt.Sprintf("%s. %s arrive with a %s identity after the board moved on from %s.", reason, next.Name, next.Tactic(), old.Name),
			completedMW, []string{club.ClubID}, "", "",
		)
	}
}
