package tournament

import (
	"fmt"
	"sort"
)

// Country-pure league realignment.
//
// Saves written by the retired cross-country swap era can carry domestic
// league registries that no longer match the dataset-derived club leagues
// (for example Atletico Madrid registered to the Premier League). The
// invariant is now "no club ever changes league", and club.League is
// dataset-derived on every load, so such a save must be realigned before the
// world can validate: every domestic competition is re-seeded from the
// club's national league, the calendar is rebuilt, and the season restarts
// from matchweek 1 with all-time history kept.

// domesticMembershipMisalignedUnlocked reports whether any domestic league
// registry disagrees with the clubs' dataset-derived League fields.
// The caller must hold tm.mu.
func (tm *TournamentManager) domesticMembershipMisalignedUnlocked() bool {
	if tm.World == nil || len(tm.World.Competitions) == 0 {
		return false
	}
	for _, def := range domesticLeagueDefinitions {
		comp := tm.World.Competitions[def.ID]
		if comp == nil || comp.Kind != CompetitionLeague {
			return true
		}
		members := make([]string, 0)
		for _, club := range tm.ClubsList {
			if club != nil && club.League == def.League {
				members = append(members, club.ClubID)
			}
		}
		sort.Strings(members)
		if len(members) != len(comp.ParticipantIDs) {
			return true
		}
		for i, id := range comp.ParticipantIDs {
			if members[i] != id {
				return true
			}
		}
	}
	return false
}

// RealignCountryPureWorld detects a swap-era domestic registry and rebuilds
// the country-pure calendar. It returns true when a realignment happened.
// The season restarts from matchweek 1; all-time history is kept.
func (tm *TournamentManager) RealignCountryPureWorld() bool {
	if tm == nil {
		return false
	}
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if !tm.domesticMembershipMisalignedUnlocked() {
		return false
	}
	tm.realignCountryPureWorldUnlocked()
	return true
}

func (tm *TournamentManager) realignCountryPureWorldUnlocked() {
	// Zero the season the same way a season restart does: the rebuilt
	// calendar invalidates every saved fixture, so stale results and
	// standings must not survive.
	for _, club := range tm.ClubsList {
		for _, p := range club.Squad {
			p.Goals, p.Assists, p.Appearances = 0, 0, 0
			p.OwnGoals, p.SuspendedMatches, p.InjuredMatches = 0, 0, 0
			p.Injury = ""
			p.ConsecutiveStarts = 0
			p.ResetSeasonCompetitionStats()
			p.Fitness = 80
			p.Sharpness = 65
		}
		club.Played, club.Won, club.Drawn, club.Lost = 0, 0, 0, 0
		club.GoalsFor, club.GoalsAgainst, club.GoalDifference, club.Points = 0, 0, 0, 0
		club.Form = []string{}
		club.Morale = 70
		club.RecalculateRatings()
	}

	// Re-seed every domestic and European competition from the clubs'
	// national leagues, with fresh-world European fields (the zeroed tables
	// have no qualification to carry over). The international competition
	// survives the rebuild with its history and viewer squads intact.
	tm.rebuildEuropeanWorldCalendarUnlocked(nil)
	tm.initializeNationalTeamsUnlocked()

	tm.CurrentMatchweek = 1
	tm.SeasonPhase = "season"
	tm.RecentResults = nil
	tm.GrowthNotifications = nil
	tm.PlayerOfTheWeek = nil
	tm.MonthlyAwards = nil
	tm.ReputationAppliedSeason = ""
	tm.ManagerConsecutiveHot = map[string]int{}
	tm.ManagerLastChange = map[string]int{}
	tm.MatchweekWeather = map[int]string{}
	for mw := 1; mw <= tm.MaxMatchweeks; mw++ {
		tm.MatchweekWeather[mw] = tm.weatherUnlocked(mw)
	}
	if tm.GrowthEngine != nil {
		tm.GrowthEngine.ReplenishTrainingEnergy()
	}
	tm.PushInbox(
		MsgCategorySystem,
		"League realignment: every club returns to its national league",
		fmt.Sprintf("The closed pyramid is country-pure: no club ever changes league. %s restarts from matchweek 1 with the domestic fields re-seeded; all-time history is kept.", tm.SeasonName),
		1, nil, "", "",
	)
}
