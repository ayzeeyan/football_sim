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
		for _, f := range tm.SuperCupFixtures {
			if f.Stage == "final" && f.Status == "finished" && f.HomeGoals != nil && f.AwayGoals != nil {
				winner := f.HomeID
				if *f.AwayGoals > *f.HomeGoals {
					winner = f.AwayID
				}
				if winner == club.ClubID {
					in.SuperCupChampion = true
				}
			}
		}
		club.Identity.Reputation = models.ClampClubRating(club.Identity.Reputation + CalculateClubReputationChange(in))
	}
}
