package tournament

import "football_sim/pkg/models"

const (
	maxAnnualReputationGain       = 8
	maxAnnualReputationLoss       = 6
	elitePrestigeAnnualLoss       = 3
	highPrestigeAnnualLoss        = 4
	reputationPerformanceWeight   = 70
	reputationPrestigeWeight      = 30
	leagueChampionReputationBonus = 4
	continentalChampionBonus      = 3
	superCupChampionBonus         = 1
)

// ClubReputationSeasonInput contains only achievements the simulator actually models.
type ClubReputationSeasonInput struct {
	CurrentReputation   int
	HistoricalPrestige int
	LeagueFinish        int
	ClubCount           int
	LeagueChampion      bool
	ContinentalChampion bool
	SuperCupChampion    bool
}

// CalculateClubReputationChange returns the bounded annual reputation delta.
// Historical prestige supplies inertia but never a match-engine bonus.
func CalculateClubReputationChange(in ClubReputationSeasonInput) int {
	current := models.ClampClubRating(in.CurrentReputation)
	prestige := models.ClampClubRating(in.HistoricalPrestige)
	count := in.ClubCount
	if count < 2 {
		count = 2
	}
	finish := in.LeagueFinish
	if finish < 1 {
		finish = count
	}
	if finish > count {
		finish = count
	}

	// 1st = 100 performance, last = 0. Current stature still matters through
	// inertia, but actual results dominate the target.
	performance := ((count - finish) * 100) / (count - 1)
	target := (performance*reputationPerformanceWeight + prestige*reputationPrestigeWeight) / 100
	delta := (target - current) / 4
	if in.LeagueChampion {
		delta += leagueChampionReputationBonus
	}
	if in.ContinentalChampion {
		delta += continentalChampionBonus
	}
	if in.SuperCupChampion {
		delta += superCupChampionBonus
	}

	maxLoss := maxAnnualReputationLoss
	if prestige >= 90 {
		maxLoss = elitePrestigeAnnualLoss
	} else if prestige >= 80 {
		maxLoss = highPrestigeAnnualLoss
	}
	if delta > maxAnnualReputationGain {
		delta = maxAnnualReputationGain
	}
	if delta < -maxLoss {
		delta = -maxLoss
	}
	if current+delta > models.ClubRatingMax {
		delta = models.ClubRatingMax - current
	}
	if current+delta < models.ClubRatingMin {
		delta = models.ClubRatingMin - current
	}
	return delta
}

func (tm *TournamentManager) updateClubReputationsUnlocked() {
	standings := tm.standingsUnlocked()
	if len(standings) == 0 {
		return
	}
	superCupWinnerID := tm.superCupWinnerIDUnlocked()
	position := make(map[string]int, len(standings))
	for i, club := range standings {
		position[club.ClubID] = i + 1
	}
	for _, club := range tm.ClubsList {
		if club == nil {
			continue
		}
		in := ClubReputationSeasonInput{
			CurrentReputation: club.Identity.Reputation,
			HistoricalPrestige: club.Identity.HistoricalPrestige,
			LeagueFinish: position[club.ClubID],
			ClubCount: len(standings),
			LeagueChampion: position[club.ClubID] == 1,
			ContinentalChampion: tm.UCLChampionID == club.ClubID,
		}
		in.SuperCupChampion = superCupWinnerID == club.ClubID
		club.Identity.Reputation = models.ClampClubRating(club.Identity.Reputation + CalculateClubReputationChange(in))
	}
}

func (tm *TournamentManager) superCupWinnerIDUnlocked() string {
	if tm.SuperCupChampionID != "" {
		return tm.SuperCupChampionID
	}
	for _, f := range tm.SuperCupFixtures {
		if (f.Stage != "final" && f.Stage != "Final") || f.Status != "finished" || f.HomeGoals == nil || f.AwayGoals == nil {
			continue
		}
		winner := f.HomeID
		if *f.AwayGoals > *f.HomeGoals || (*f.AwayGoals == *f.HomeGoals && len(f.Penalties) >= 2 && f.Penalties[1] > f.Penalties[0]) {
			winner = f.AwayID
		}
		return winner
	}
	return ""
}

// completedSeasonInputsAvailableUnlocked reports whether the current
// transfer-window state has all results needed for an annual reputation
// update. In particular, the final league week can enter the transfer phase
// before same-week cup fixtures are applied, so the phase alone is not enough.
func (tm *TournamentManager) completedSeasonInputsAvailableUnlocked() bool {
	if tm == nil || tm.SeasonName == "" || tm.SeasonPhase != "transfer_window" || len(tm.ClubsList) == 0 {
		return false
	}

	leagueFixtures := 0
	for _, f := range tm.Fixtures {
		if f.Competition != "" && f.Competition != "super-league" {
			continue
		}
		leagueFixtures++
		if f.Status != "finished" || f.HomeGoals == nil || f.AwayGoals == nil {
			return false
		}
	}
	// A legacy/manual state may not retain its league fixture list. The
	// calendar boundary is the only safe completion signal in that case.
	if leagueFixtures == 0 && tm.CurrentMatchweek <= tm.MaxMatchweeks {
		return false
	}

	if len(tm.UCLFixtures) > 0 {
		for _, f := range tm.UCLFixtures {
			if f.Status != "finished" || f.HomeGoals == nil || f.AwayGoals == nil {
				return false
			}
		}
		if tm.UCLChampionID == "" || tm.Clubs[tm.UCLChampionID] == nil {
			return false
		}
	}
	if len(tm.SuperCupFixtures) > 0 {
		for _, f := range tm.SuperCupFixtures {
			if f.Status != "finished" || f.HomeGoals == nil || f.AwayGoals == nil {
				return false
			}
		}
		winnerID := tm.superCupWinnerIDUnlocked()
		if winnerID == "" || tm.Clubs[winnerID] == nil {
			return false
		}
	}
	return true
}

// ApplyCompletedSeasonReputation applies the completed campaign exactly once.
// The public wrapper owns the manager lock; callers already holding it should
// use applyCompletedSeasonReputationUnlocked instead.
func (tm *TournamentManager) ApplyCompletedSeasonReputation() bool {
	if tm == nil {
		return false
	}
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return tm.applyCompletedSeasonReputationUnlocked()
}

func (tm *TournamentManager) applyCompletedSeasonReputationUnlocked() bool {
	if tm == nil || tm.ReputationAppliedSeason == tm.SeasonName || !tm.completedSeasonInputsAvailableUnlocked() {
		return false
	}
	tm.updateClubReputationsUnlocked()
	tm.ReputationAppliedSeason = tm.SeasonName
	return true
}
