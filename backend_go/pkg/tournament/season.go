package tournament

import (
	"fmt"
	"strconv"
	"strings"

	"football_sim/pkg/datamanager"
	"football_sim/pkg/models"
)

func (tm *TournamentManager) archiveSeasonUnlocked() {
	allZero := true
	for _, c := range tm.ClubsList {
		if c.Played > 0 {
			allZero = false
			break
		}
	}
	if allZero {
		return
	}
	awards := tm.seasonAwardsUnlocked()
	table := tm.standingsUnlocked()
	row := map[string]interface{}{
		"season_name":          tm.SeasonName,
		"champion":             awards["super_league_champion"],
		"runner_up":            awards["super_league_runner_up"],
		"ucl_champion":         awards["ucl_champion"],
		"top_scorer":           awards["top_scorer"],
		"top_assister":         awards["top_assister"],
		"golden_boy":           awards["golden_boy"],
		"ballon_dor":           awards["ballon_dor"],
		"player_of_the_season": awards["player_of_the_season"],
		"super_cup_champion":   awards["super_cup_champion"],
		"monthly_awards":       tm.MonthlyAwards,
	}
	var tableTop []map[string]interface{}
	for i, c := range table {
		if i >= 3 {
			break
		}
		tableTop = append(tableTop, map[string]interface{}{
			"club_name": c.ClubName, "short_name": c.ShortName,
			"pts": c.Points, "gd": c.GoalDifference, "w": c.Won, "d": c.Drawn, "l": c.Lost,
		})
	}
	row["table"] = tableTop
	row["recap"] = composeSeasonRecap(row)
	tm.SeasonHistory = append(tm.SeasonHistory, row)

	if tm.ClubSeasonHistory == nil {
		tm.ClubSeasonHistory = map[string][]map[string]interface{}{}
	}
	posMap := map[string]int{}
	for i, c := range table {
		posMap[c.ClubID] = i + 1
	}
	sl, _ := awards["super_league_champion"].(map[string]interface{})
	ucl, _ := awards["ucl_champion"].(map[string]interface{})
	sc, _ := awards["super_cup_champion"].(map[string]interface{})
	for _, club := range tm.ClubsList {
		trophies := []string{}
		if sl != nil && sl["short_name"] == club.ShortName {
			trophies = append(trophies, "Super League Champion")
		}
		if ucl != nil && ucl["short_name"] == club.ShortName {
			trophies = append(trophies, "Champions Cup")
		}
		if sc != nil && sc["short_name"] == club.ShortName {
			trophies = append(trophies, "Super Cup")
		}
		tm.ClubSeasonHistory[club.ClubID] = append(tm.ClubSeasonHistory[club.ClubID], map[string]interface{}{
			"season_name": tm.SeasonName,
			"position":    posMap[club.ClubID],
			"pts":         club.Points,
			"w":           club.Won, "d": club.Drawn, "l": club.Lost,
			"gf": club.GoalsFor, "ga": club.GoalsAgainst, "gd": club.GoalDifference,
			"trophies": trophies,
		})
	}
}

func composeSeasonRecap(row map[string]interface{}) string {
	var bits []string
	champ, _ := row["champion"].(map[string]interface{})
	runner, _ := row["runner_up"].(map[string]interface{})
	if champ != nil && champ["club_name"] != nil {
		line := fmt.Sprintf("%v won the Super League", champ["club_name"])
		if pts, ok := champ["pts"]; ok && pts != nil {
			line += fmt.Sprintf(" on %v points", pts)
		}
		if runner != nil && runner["club_name"] != nil {
			line += fmt.Sprintf(", ahead of %v", runner["club_name"])
		}
		bits = append(bits, line+".")
	}
	if ucl, ok := row["ucl_champion"].(map[string]interface{}); ok && ucl != nil && ucl["club_name"] != nil {
		if champ != nil && champ["club_name"] == ucl["club_name"] {
			bits = append(bits, "They added the Champions Cup.")
		} else {
			bits = append(bits, fmt.Sprintf("%v took the Champions Cup.", ucl["club_name"]))
		}
	}
	if sc, ok := row["super_cup_champion"].(map[string]interface{}); ok && sc != nil && sc["club_name"] != nil {
		bits = append(bits, fmt.Sprintf("%v lifted the Super Cup.", sc["club_name"]))
	}
	if boot, ok := row["top_scorer"].(map[string]interface{}); ok && boot != nil && boot["full_name"] != nil {
		bits = append(bits, fmt.Sprintf("%v won the golden boot with %v.", boot["full_name"], boot["goals"]))
	}
	return strings.Join(bits, " ")
}

func (tm *TournamentManager) ageAllPlayersUnlocked() {
	for _, club := range tm.ClubsList {
		for _, p := range club.Squad {
			p.Age++
			if tm.GrowthEngine != nil {
				if bio := tm.GrowthEngine.Biometrics[p.PlayerID]; bio != nil {
					bio.Age = p.Age
					bio.YearlyHeightTaken = 0
					if p.Age >= bio.AdultHeightAge {
						bio.PubertyStage = "Adult frame"
					}
				}
			}
		}
	}
}

func (tm *TournamentManager) applySeasonalChangesUnlocked() {
	for _, club := range tm.ClubsList {
		for _, p := range club.Squad {
			if p.Age >= 30 {
				drop := 1
				if p.Age >= 36 {
					drop = 3
				} else if p.Age >= 34 {
					drop = 2
				}
				p.OVR = p.OVR - drop
				if p.OVR < 55 {
					p.OVR = 55
				}
				if tm.GrowthEngine != nil {
					tm.GrowthEngine.ApplyAgingDecline(p.PlayerID, p.Age)
				}
			} else if p.Age < 25 && tm.GrowthEngine != nil {
				pot := p.OVR + (25-p.Age)*2
				if pot > 99 {
					pot = 99
				}
				if bio := tm.GrowthEngine.Biometrics[p.PlayerID]; bio != nil {
					pot = bio.Potential
				} else if p.UniverseWonderkid {
					pot = 96
				}
				if p.OVR < pot {
					newOVR := tm.GrowthEngine.ApplySeasonalGrowth(p.PlayerID, p.Age, p.Appearances, pot, p.Category, p.OVR)
					if newOVR > p.OVR {
						p.OVR = newOVR
					}
				}
			}
		}
	}
}

func (tm *TournamentManager) processRetirementsUnlocked() {
	for _, club := range tm.ClubsList {
		kept := club.Squad[:0]
		for _, p := range club.Squad {
			if p.Age < 36 {
				kept = append(kept, p)
				continue
			}
			chance := 0.10
			switch {
			case p.Age >= 42:
				chance = 1.0
			case p.Age >= 39:
				chance = 0.90
			case p.Age == 38:
				chance = 0.50
			case p.Age == 37:
				chance = 0.25
			}
			roll := 0.0
			if tm.RNG != nil {
				roll = tm.RNG.Float64()
			}
			if roll < chance {
				tm.PushInbox("honour",
					fmt.Sprintf("%s hangs up his boots", p.FullName),
					fmt.Sprintf("After a career at %s, %s retires at %d.", club.ClubName, p.FullName, p.Age),
					1, []string{club.ClubID}, p.PlayerID, "")
				continue
			}
			kept = append(kept, p)
		}
		club.Squad = kept
	}
}

// ResetNewSeason archives the campaign, ages the squad, and rebuilds the calendar.
// Current squad membership is authoritative; OriginalClubID is historical metadata
// and is never used to reconstruct rosters.
func (tm *TournamentManager) ResetNewSeason() map[string]interface{} {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.archiveSeasonUnlocked()
	finalOrder := tm.standingsUnlocked()
	finalIDs := make([]string, 0, len(finalOrder))
	for _, c := range finalOrder {
		finalIDs = append(finalIDs, c.ClubID)
	}

	posMap := map[string]int{}
	for i, cid := range finalIDs {
		posMap[cid] = i + 1
	}
	for _, club := range tm.ClubsList {
		place := posMap[club.ClubID]
		if place == 0 {
			place = 12
		}
		for _, p := range club.Squad {
			p.CareerGoals += p.Goals
			p.CareerAssists += p.Assists
			p.CareerApps += p.Appearances
			if p.Goals > p.BestGoals {
				p.BestGoals = p.Goals
				p.BestAssists = p.Assists
				p.BestSeason = tm.SeasonName
			}
			if p.UniverseWonderkid {
				if tm.GrowthEngine != nil {
					h, w := 170.0, 65.0
					mName := p.MentorName
					if bio := tm.GrowthEngine.Biometrics[p.PlayerID]; bio != nil {
						h, w = bio.CurrentHeightCM, bio.CurrentWeightKG
						if bio.MentorName != "" {
							mName = bio.MentorName
						}
					}
					tm.GrowthEngine.RecordTimelineEntry(p.PlayerID, tm.SeasonName, p.Age, p.OVR, h, w, p.Goals, p.Assists, p.Appearances, club.ShortName, mName)
				}
				ev := p.AdvanceEducation(place)
				switch ev {
				case "stayed":
					tm.PushInbox("youth", p.FullName+" stays in school", p.SchoolWantLine()+" High school from here — exam weeks only.", 1, []string{p.ClubID}, p.PlayerID, "")
				case "left":
					tm.PushInbox("youth", p.FullName+" leaves school for football", p.SchoolWantLine()+" Full-time with the first team.", 1, []string{p.ClubID}, p.PlayerID, "")
				case "graduated":
					tm.PushInbox("youth", p.FullName+" finished school", "Full-time football from here. No more exam weeks.", 1, []string{p.ClubID}, p.PlayerID, "")
				}
			}
			if place <= 4 {
				p.Loyalty += 5
				if p.Loyalty > 95 {
					p.Loyalty = 95
				}
			} else if place >= 10 {
				p.Loyalty -= 5
				if p.Loyalty < 5 {
					p.Loyalty = 5
				}
			} else {
				p.Loyalty += 2
				if p.Loyalty > 95 {
					p.Loyalty = 95
				}
			}
		}
	}

	tm.ageAllPlayersUnlocked()
	tm.applySeasonalChangesUnlocked()
	tm.processRetirementsUnlocked()

	for _, club := range tm.ClubsList {
		for _, p := range club.Squad {
			p.Goals, p.Assists, p.Appearances = 0, 0, 0
			p.OwnGoals, p.SuspendedMatches, p.InjuredMatches = 0, 0, 0
			p.Injury = ""
			p.ConsecutiveStarts = 0
		}
		club.Played, club.Won, club.Drawn, club.Lost = 0, 0, 0, 0
		club.GoalsFor, club.GoalsAgainst, club.GoalDifference, club.Points = 0, 0, 0, 0
		club.Form = []string{}
		club.Morale = 70
		club.RecalculateRatings()
	}

	parts := strings.Split(tm.SeasonName, "-")
	if len(parts) >= 1 {
		if y, err := strconv.Atoi(parts[0]); err == nil {
			tm.SeasonName = fmt.Sprintf("%d-%02d", y+1, (y+2)%100)
		}
	}

	if tm.TransferEngine != nil {
		tm.TransferEngine.ResetForNewSeason()
	}

	tm.CurrentMatchweek = 1
	tm.RecentResults = nil
	tm.GrowthNotifications = nil
	tm.PlayerOfTheWeek = nil
	tm.MonthlyAwards = nil
	tm.SeasonPhase = "season"
	tm.Fixtures = GenerateLeagueFixtures(tm.ClubsList, tm.RNG)
	ordered := make([]*models.Club, 0, len(finalIDs))
	for _, cid := range finalIDs {
		if c := tm.Clubs[cid]; c != nil {
			ordered = append(ordered, c)
		}
	}
	if len(ordered) >= 12 {
		tm.reseedUCLGroups(ordered)
	} else {
		tm.UCLGroupA = append([]*models.Club(nil), tm.ClubsList[:len(tm.ClubsList)/2]...)
		tm.UCLGroupB = append([]*models.Club(nil), tm.ClubsList[len(tm.ClubsList)/2:]...)
	}
	tm.UCLStage = "GROUP_STAGE"
	tm.UCLQuarterFinals = map[string]CupTie{}
	tm.UCLSemiFinals = map[string]CupTie{}
	tm.UCLFinal = CupTie{}
	tm.UCLChampionID = ""
	tm.initUCLRecords()
	tm.buildUCLGroupFixtures()
	if len(ordered) == 0 {
		ordered = append([]*models.Club(nil), tm.ClubsList...)
		sortClubsByRating(ordered)
	}
	tm.drawSuperCup(ordered)

	if tm.GrowthEngine != nil {
		tm.GrowthEngine.ReplenishTrainingEnergy()
	}
	tm.ManagerConsecutiveHot = map[string]int{}
	tm.ManagerLastChange = map[string]int{}
	tm.MatchweekWeather = map[int]string{}
	for mw := 1; mw <= LeagueRounds; mw++ {
		tm.MatchweekWeather[mw] = tm.weatherUnlocked(mw)
	}

	if signed, err := datamanager.RunYouthIntakeClubs(tm.ClubsList, nil, tm.GrowthEngine, tm.RNG); err == nil {
		for _, p := range signed {
			club := tm.Clubs[p.ClubID]
			name := p.ClubID
			if club != nil {
				name = club.ShortName
			}
			tm.PushInbox("youth",
				fmt.Sprintf("Academy intake: %s signs for %s", p.FullName, name),
				fmt.Sprintf("%d, %s, %d OVR. Not a franchise prodigy — a kid from the system.", p.Age, p.Position, p.OVR),
				1, []string{p.ClubID}, p.PlayerID, "")
		}
	}
	PairSeniorMentors(tm.ClubsList, tm.GrowthEngine)
	tm.seedOpeningInbox()

	return map[string]interface{}{
		"status":            "success",
		"message":           "New European Super League season initialized. Permanent transfers and current squads were preserved.",
		"current_matchweek": 1,
		"max_matchweeks":    tm.MaxMatchweeks,
	}
}

// AdoptLongSeason migrates legacy short/long calendar saves onto the current
// mathematically correct quadruple round-robin schedule without wiping results
// that still map to a canonical fixture.
func (tm *TournamentManager) AdoptLongSeason() bool {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return tm.adoptLongSeasonUnlocked()
}

func (tm *TournamentManager) adoptLongSeasonUnlocked() bool {
	target := (len(tm.ClubsList) * LeagueRounds) / 2
	already := tm.MaxMatchweeks == LeagueRounds && len(tm.Fixtures) == target
	if already && len(tm.SuperCupFixtures) > 0 {
		return false
	}
	changed := false
	if len(tm.Fixtures) != target || tm.MaxMatchweeks != LeagueRounds {
		generated := GenerateLeagueFixtures(tm.ClubsList, tm.RNG)
		existing := make(map[string]Fixture, len(tm.Fixtures))
		for _, f := range tm.Fixtures {
			if f.Status == "finished" {
				existing[f.FixtureID] = f
			}
		}
		merged := make([]Fixture, 0, len(generated))
		for _, f := range generated {
			if old, ok := existing[f.FixtureID]; ok {
				merged = append(merged, old)
			} else {
				merged = append(merged, f)
			}
		}
		tm.Fixtures = merged
		tm.MaxMatchweeks = LeagueRounds
		if tm.CurrentMatchweek > LeagueRounds+1 {
			tm.CurrentMatchweek = LeagueRounds + 1
		}
		changed = true
	}
	if len(tm.SuperCupFixtures) == 0 {
		ordered := tm.standingsUnlocked()
		played := false
		for _, c := range tm.ClubsList {
			if c.Played > 0 {
				played = true
				break
			}
		}
		if !played {
			ordered = append([]*models.Club(nil), tm.ClubsList...)
			sortClubsByRating(ordered)
		}
		tm.drawSuperCup(ordered)
		changed = true
	}
	if tm.MatchweekWeather == nil {
		tm.MatchweekWeather = map[int]string{}
	}
	for mw := 1; mw <= LeagueRounds; mw++ {
		if tm.MatchweekWeather[mw] == "" {
			tm.MatchweekWeather[mw] = tm.weatherUnlocked(mw)
		}
	}
	return changed
}

// RestartCurrentSeason rewinds this campaign without aging anyone.
func (tm *TournamentManager) RestartCurrentSeason() map[string]interface{} {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.CurrentMatchweek = 1
	tm.RecentResults = nil
	tm.GrowthNotifications = nil
	tm.PlayerOfTheWeek = nil
	tm.MonthlyAwards = nil
	tm.SeasonPhase = "season"
	var keptInbox []InboxItem
	for _, item := range tm.Inbox {
		if item.SeasonName != tm.SeasonName {
			keptInbox = append(keptInbox, item)
		}
	}
	tm.Inbox = keptInbox
	for _, club := range tm.ClubsList {
		for _, p := range club.Squad {
			p.Goals, p.Assists, p.Appearances = 0, 0, 0
			p.OwnGoals, p.SuspendedMatches, p.InjuredMatches = 0, 0, 0
			p.Injury = ""
			p.ConsecutiveStarts = 0
		}
		club.Played, club.Won, club.Drawn, club.Lost = 0, 0, 0, 0
		club.GoalsFor, club.GoalsAgainst, club.GoalDifference, club.Points = 0, 0, 0, 0
		club.Form = []string{}
		club.Morale = 70
		club.RecalculateRatings()
	}
	if tm.TransferEngine != nil {
		tm.TransferEngine.ResetForNewSeason()
	}
	tm.Fixtures = GenerateLeagueFixtures(tm.ClubsList, tm.RNG)
	tm.UCLGroupA = append([]*models.Club(nil), tm.ClubsList[:len(tm.ClubsList)/2]...)
	tm.UCLGroupB = append([]*models.Club(nil), tm.ClubsList[len(tm.ClubsList)/2:]...)
	tm.UCLStage = "GROUP_STAGE"
	tm.UCLQuarterFinals = map[string]CupTie{}
	tm.UCLSemiFinals = map[string]CupTie{}
	tm.UCLFinal = CupTie{}
	tm.UCLChampionID = ""
	tm.initUCLRecords()
	tm.buildUCLGroupFixtures()
	ordered := append([]*models.Club(nil), tm.ClubsList...)
	sortClubsByRating(ordered)
	tm.drawSuperCup(ordered)
	tm.ManagerConsecutiveHot = map[string]int{}
	tm.ManagerLastChange = map[string]int{}
	tm.MatchweekWeather = map[int]string{}
	for mw := 1; mw <= LeagueRounds; mw++ {
		tm.MatchweekWeather[mw] = tm.weatherUnlocked(mw)
	}
	if tm.GrowthEngine != nil {
		tm.GrowthEngine.ReplenishTrainingEnergy()
	}
	PairSeniorMentors(tm.ClubsList, tm.GrowthEngine)
	tm.seedOpeningInbox()
	return map[string]interface{}{
		"status":            "success",
		"message":           "Season restarted. All-time records kept. Table and this year's goals begin again.",
		"current_matchweek": 1,
		"max_matchweeks":    tm.MaxMatchweeks,
		"season_name":       tm.SeasonName,
	}
}

func removeFromSquad(club *models.Club, player *models.Player) {
	if club == nil || player == nil {
		return
	}
	out := club.Squad[:0]
	for _, p := range club.Squad {
		if p != player && p.PlayerID != player.PlayerID {
			out = append(out, p)
		}
	}
	club.Squad = out
}

func inSquad(club *models.Club, player *models.Player) bool {
	for _, p := range club.Squad {
		if p == player || p.PlayerID == player.PlayerID {
			return true
		}
	}
	return false
}
