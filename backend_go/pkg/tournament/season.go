package tournament

import (
	"fmt"
	"sort"
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

// effectiveAppearances converts a season's tracked minutes into full-match
// equivalents for development. Players without minute tracking (legacy saves,
// unit tests) fall back to raw appearances, so behavior there is unchanged.
// The result never exceeds raw appearances: stoppage-time and extra-time
// padding cannot manufacture development. The growth engine's +5 annual cap
// and potential ceilings apply downstream untouched.
func effectiveAppearances(p *models.Player) int {
	if p == nil || p.Appearances <= 0 {
		return 0
	}
	// No minute tracking at all (legacy saves, unit fixtures): fall back to
	// raw appearances. A present-but-empty minutes total means bench minutes
	// only and correctly yields reduced development.
	if len(p.CompetitionStats) == 0 {
		return p.Appearances
	}
	minutes := 0
	for _, row := range p.CompetitionStats {
		if row != nil {
			minutes += row.Minutes
		}
	}
	eff := (minutes + 45) / 90
	if eff < 0 {
		eff = 0
	}
	if eff > p.Appearances {
		eff = p.Appearances
	}
	return eff
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
					// Bioless wonderkids (corrupt/hand-built states) still get
					// their canonical ceiling, never a blanket 96 that could
					// overshoot a true 93-95 potential.
					if canonical, ok := datamanager.CanonicalPotential(p.FullName); ok {
						pot = canonical
					} else {
						pot = 96
					}
				}
				if p.OVR < pot {
					// Minutes-weighted playing time feeds the same engine, so
					// the +5 annual cap and potential ceilings still bind:
					// regular starters develop as before, cameo players less.
					newOVR := tm.GrowthEngine.ApplySeasonalGrowth(p.PlayerID, p.Age, effectiveAppearances(p), pot, p.Category, p.OVR)
					if newOVR > p.OVR {
						p.OVR = newOVR
					}
				}
			}
		}
	}
}

func (tm *TournamentManager) processRetirementsUnlocked() []string {
	if tm.RetiredPlayerIDs == nil {
		tm.RetiredPlayerIDs = make(map[string]bool)
	}
	var retiredIDs []string
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
				retiredIDs = append(retiredIDs, p.PlayerID)
				tm.RetiredPlayerIDs[p.PlayerID] = true
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
	sort.Strings(retiredIDs)
	return retiredIDs
}

// advanceGrowthBaselinesUnlocked commits each registered player's capped OVR
// once at the new-season boundary. Weekly growth and repeated seasonal calls
// must continue to share the same baseline until this point.
func (tm *TournamentManager) advanceGrowthBaselinesUnlocked() {
	if tm.GrowthEngine == nil {
		return
	}
	for _, club := range tm.ClubsList {
		for _, p := range club.Squad {
			if p == nil {
				continue
			}
			tm.GrowthEngine.AdvanceSeasonStartOVR(p.PlayerID, p.Category, p.OVR)
		}
	}
}

// ResetNewSeason archives the campaign, ages the squad, and rebuilds the calendar.
// Current squad membership is authoritative; OriginalClubID is historical metadata
// and is never used to reconstruct rosters.
func (tm *TournamentManager) ResetNewSeason() map[string]interface{} {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if tm.World != nil {
		return tm.resetEuropeanWorldNewSeasonUnlocked()
	}

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
	retiredIDs := tm.processRetirementsUnlocked()
	tm.advanceGrowthBaselinesUnlocked()

	// Mirror the world path: winter loanees come home before squads reset.
	tm.ReturnLoansUnlocked()

	for _, club := range tm.ClubsList {
		for _, p := range club.Squad {
			p.Goals, p.Assists, p.Appearances = 0, 0, 0
			p.OwnGoals, p.SuspendedMatches, p.InjuredMatches = 0, 0, 0
			p.Injury = ""
			p.ConsecutiveStarts = 0
			// Same seasonal wipe as the world path (minutes, ratings,
			// requests) and the same dynamics reset: stale form, leave
			// triggers, and fatigue must not leak into the new season.
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
		billed := map[string]bool{}
		for _, p := range signed {
			club := tm.Clubs[p.ClubID]
			name := p.ClubID
			if club != nil {
				name = club.ShortName
				// Academy wages join the bill immediately so the stored
				// WageBudget never trails the live squad coalescing below.
				// Intake itself is never gated: graduates are the club's own.
				if !billed[club.ClubID] {
					billed[club.ClubID] = true
					club.RecalculateWageBill()
				}
			}
			tm.PushInbox("youth",
				fmt.Sprintf("Academy intake: %s signs for %s", p.FullName, name),
				fmt.Sprintf("%d, %s, %d OVR. Not a franchise prodigy — a kid from the system.", p.Age, p.Position, p.OVR),
				1, []string{p.ClubID}, p.PlayerID, "")
		}
	}
	tm.ArrangeLoansUnlocked()
	PairSeniorMentors(tm.ClubsList, tm.GrowthEngine)
	tm.seedOpeningInbox()

	return map[string]interface{}{
		"status":             "success",
		"message":            "New European Super League season initialized. Permanent transfers and current squads were preserved.",
		"current_matchweek":  1,
		"max_matchweeks":     tm.MaxMatchweeks,
		"retired_player_ids": retiredIDs,
	}
}

func (tm *TournamentManager) nextEuropeanQualificationUnlocked() map[string]map[string]string {
	result := map[string]map[string]string{
		"champions-league": {}, "europa-league": {}, "conference-league": {},
	}
	uclHolder := ""
	if comp := tm.worldCompetitionUnlocked("champions-league"); comp != nil {
		uclHolder = comp.ChampionID
	}
	add := func(compID, clubID, source string) bool {
		if clubID == "" || result[compID][clubID] != "" {
			return false
		}
		for other, sources := range result {
			if other != compID && sources[clubID] != "" {
				return false
			}
		}
		result[compID][clubID] = source
		return true
	}
	for _, league := range domesticLeagueDefinitions {
		table := tm.worldLeagueStandingsUnlocked(league.ID)
		if len(table) == 0 {
			continue
		}
		uclWant := championsLeagueOpeningSlots(len(table))
		uclTaken := 0
		if holder := tm.Clubs[uclHolder]; holder != nil && holder.League == league.League {
			if add("champions-league", holder.ClubID, "Champions League holders") {
				uclTaken++
			}
		}
		for _, club := range table {
			if uclTaken >= uclWant {
				break
			}
			if add("champions-league", club.ClubID, "League position") {
				uclTaken++
			}
		}
		cupWinner := ""
		for _, cup := range domesticCupDefinitions {
			if cup.League == league.League {
				if comp := tm.worldCompetitionUnlocked(cup.ID); comp != nil {
					cupWinner = comp.ChampionID
				}
				break
			}
		}
		elTaken := 0
		if add("europa-league", cupWinner, "Domestic cup winners") {
			elTaken++
		}
		for _, club := range table {
			if elTaken >= 4 {
				break
			}
			if add("europa-league", club.ClubID, "League position") {
				elTaken++
			}
		}
		uelTaken := 0
		for _, club := range table {
			if uelTaken >= 4 {
				break
			}
			if add("conference-league", club.ClubID, "League position") {
				uelTaken++
			}
		}
	}
	return result
}

func (tm *TournamentManager) rebuildEuropeanWorldCalendarUnlocked(qualification map[string]map[string]string) {
	seed := int64(20260907)
	if tm.World != nil && tm.World.Seed != 0 {
		seed = tm.World.Seed
	}
	tm.World = &EuropeanWorld{Version: 1, Seed: seed, Competitions: map[string]*Competition{}, Fixtures: []Fixture{}}
	byLeague := map[string][]*models.Club{}
	for _, club := range tm.ClubsList {
		if club != nil && isTopFiveLeague(club.League) {
			byLeague[club.League] = append(byLeague[club.League], club)
		}
	}
	tm.Fixtures = nil
	tm.MaxMatchweeks = 38
	for _, def := range domesticLeagueDefinitions {
		members := byLeague[def.League]
		if len(members) < 2 {
			continue
		}
		comp := &Competition{ID: def.ID, Name: def.Name, Country: def.Country, Kind: CompetitionLeague, Prestige: def.Prestige, ParticipantIDs: sortedClubIDs(members), Stage: "League"}
		tm.World.Competitions[def.ID] = comp
		tm.World.CompetitionOrder = append(tm.World.CompetitionOrder, def.ID)
		fixtures, rounds := GenerateDoubleRoundRobinFixtures(members, def.ID)
		if rounds > tm.MaxMatchweeks {
			tm.MaxMatchweeks = rounds
		}
		tm.Fixtures = append(tm.Fixtures, fixtures...)
	}
	for _, def := range domesticCupDefinitions {
		members := byLeague[def.League]
		if len(members) < 2 {
			continue
		}
		comp := &Competition{ID: def.ID, Name: def.Name, Country: def.Country, Kind: CompetitionDomestic, Prestige: def.Prestige, ParticipantIDs: sortedClubIDs(members), Stage: "Draw"}
		tm.World.Competitions[def.ID] = comp
		tm.World.CompetitionOrder = append(tm.World.CompetitionOrder, def.ID)
		tm.scheduleWorldCupOpeningRoundUnlocked(comp)
	}
	for _, def := range europeanDefinitions {
		comp := &Competition{ID: def.ID, Name: def.Name, Country: def.Country, Kind: CompetitionEuropean, Prestige: def.Prestige, Records: map[string]*models.CompetitionRecord{}, QualificationSources: map[string]string{}, Stage: "League Phase"}
		if qualification != nil {
			for id, source := range qualification[def.ID] {
				comp.ParticipantIDs = append(comp.ParticipantIDs, id)
				comp.QualificationSources[id] = source
			}
			sort.Strings(comp.ParticipantIDs)
		} else {
			tm.seedOpeningEuropeanParticipantsUnlocked(comp, byLeague)
		}
		tm.World.Competitions[def.ID] = comp
		tm.World.CompetitionOrder = append(tm.World.CompetitionOrder, def.ID)
		tm.scheduleEuropeanLeaguePhaseUnlocked(comp)
	}
}

func (tm *TournamentManager) archiveEuropeanSeasonUnlocked() {
	// Nothing played means nothing to bank or record: a repeated direct
	// reset on a fresh season must not mint prizes, history rows, or inbox
	// cheques for an unplayed campaign (mirrors the legacy allZero guard).
	played := false
	for _, c := range tm.ClubsList {
		if c != nil && c.Played > 0 {
			played = true
			break
		}
	}
	if !played && tm.World != nil {
		for _, comp := range tm.World.Competitions {
			if comp == nil {
				continue
			}
			if comp.ChampionID != "" {
				played = true
				break
			}
			if comp.Kind == CompetitionEuropean {
				for _, rec := range comp.Records {
					if rec != nil && rec.Played > 0 {
						played = true
						break
					}
				}
			}
			if played {
				break
			}
		}
	}
	if !played {
		return
	}
	champions := map[string]string{}
	for _, id := range tm.World.CompetitionOrder {
		if comp := tm.World.Competitions[id]; comp != nil && comp.ChampionID != "" {
			champions[id] = comp.ChampionID
		}
	}
	leagueChampions := map[string]interface{}{}
	for _, def := range domesticLeagueDefinitions {
		if table := tm.worldLeagueStandingsUnlocked(def.ID); len(table) > 0 {
			leagueChampions[def.ID] = tm.Clubs[table[0].ClubID]
		}
	}
	tm.awardSeasonPrizeMoneyUnlocked()
	tm.SeasonHistory = append(tm.SeasonHistory, map[string]interface{}{
		"season_name": tm.SeasonName, "world": true, "league_champions": leagueChampions,
		"competition_champions": champions,
	})
	if tm.ClubSeasonHistory == nil {
		tm.ClubSeasonHistory = map[string][]map[string]interface{}{}
	}
	for _, club := range tm.ClubsList {
		position := 0
		for _, def := range domesticLeagueDefinitions {
			if club.League != def.League {
				continue
			}
			for i, ranked := range tm.worldLeagueStandingsUnlocked(def.ID) {
				if ranked.ClubID == club.ClubID {
					position = i + 1
					break
				}
			}
		}
		trophies := []string{}
		for id, winner := range champions {
			if winner == club.ClubID {
				trophies = append(trophies, tm.World.Competitions[id].Name)
			}
		}
		tm.ClubSeasonHistory[club.ClubID] = append(tm.ClubSeasonHistory[club.ClubID], map[string]interface{}{
			"season_name": tm.SeasonName, "position": position, "pts": club.Points, "w": club.Won, "d": club.Drawn, "l": club.Lost,
			"gf": club.GoalsFor, "ga": club.GoalsAgainst, "gd": club.GoalDifference, "trophies": trophies,
		})
	}
}

func (tm *TournamentManager) resetEuropeanWorldNewSeasonUnlocked() map[string]interface{} {
	qualification := tm.nextEuropeanQualificationUnlocked()
	// Bank coefficient points from the finishing season before anything is
	// rebuilt; next season's Swiss pots read them.
	tm.applyEuropeanCoefficientPointsUnlocked()
	// Open a fresh European revenue ledger before prizes are banked, so the
	// ledger always holds the last completed season's intake (replace, never
	// accumulate) for the season ahead.
	tm.resetEuropeanRevenueLedgerUnlocked()
	tm.archiveEuropeanSeasonUnlocked()
	for _, club := range tm.ClubsList {
		for _, player := range club.Squad {
			if player == nil {
				continue
			}
			player.CareerGoals += player.Goals
			player.CareerAssists += player.Assists
			player.CareerApps += player.Appearances
			if player.Goals > player.BestGoals {
				player.BestGoals, player.BestAssists, player.BestSeason = player.Goals, player.Assists, tm.SeasonName
			}
		}
	}
	tm.ageAllPlayersUnlocked()
	tm.applySeasonalChangesUnlocked()
	retired := tm.processRetirementsUnlocked()
	tm.advanceGrowthBaselinesUnlocked()
	for _, club := range tm.ClubsList {
		for _, player := range club.Squad {
			player.Goals, player.Assists, player.Appearances = 0, 0, 0
			player.OwnGoals, player.SuspendedMatches, player.InjuredMatches = 0, 0, 0
			player.Injury, player.ConsecutiveStarts = "", 0
			player.ResetSeasonCompetitionStats()
			player.Fitness = 80
			player.Sharpness = 65
		}
		club.Played, club.Won, club.Drawn, club.Lost = 0, 0, 0, 0
		club.GoalsFor, club.GoalsAgainst, club.GoalDifference, club.Points = 0, 0, 0, 0
		club.Form, club.Morale = []string{}, 70
		club.RecalculateRatings()
	}
	parts := strings.Split(tm.SeasonName, "-")
	if year, err := strconv.Atoi(parts[0]); err == nil {
		tm.SeasonName = fmt.Sprintf("%d-%02d", year+1, (year+2)%100)
	}
	if tm.TransferEngine != nil {
		tm.TransferEngine.ResetForNewSeason()
	}
	tm.CurrentMatchweek, tm.SeasonPhase = 1, "season"
	tm.RecentResults, tm.GrowthNotifications, tm.PlayerOfTheWeek, tm.MonthlyAwards = nil, nil, nil, nil
	tm.ReputationAppliedSeason = ""
	tm.ReturnLoansUnlocked()
	tm.rebuildEuropeanWorldCalendarUnlocked(qualification)
	tm.AssignSquadRolesUnlocked()
	tm.AssignBoardExpectationsUnlocked()
	tm.ArrangeLoansUnlocked()
	tm.MatchweekWeather = map[int]string{}
	for mw := 1; mw <= tm.MaxMatchweeks; mw++ {
		tm.MatchweekWeather[mw] = tm.weatherUnlocked(mw)
	}
	if tm.GrowthEngine != nil {
		tm.GrowthEngine.ReplenishTrainingEnergy()
	}
	tm.ManagerConsecutiveHot, tm.ManagerLastChange = map[string]int{}, map[string]int{}
	PairSeniorMentors(tm.ClubsList, tm.GrowthEngine)
	tm.PushInbox("system", tm.SeasonName+" European season begins", "Domestic tables reset, national cups are drawn, and qualification has set the European fields.", 1, nil, "", "")
	return map[string]interface{}{"status": "success", "message": "New European season initialized.", "current_matchweek": 1, "max_matchweeks": tm.MaxMatchweeks, "retired_player_ids": retired}
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
	if tm.World != nil {
		// Version-4 careers already own mathematically valid domestic league
		// calendars. The legacy Super League migration must never rewrite them.
		return false
	}
	target := (len(tm.ClubsList) * LeagueRounds) / 2
	already := tm.MaxMatchweeks == LeagueRounds && len(tm.Fixtures) == target
	if already && len(tm.SuperCupFixtures) > 0 {
		return false
	}
	changed := false
	if len(tm.Fixtures) != target || tm.MaxMatchweeks != LeagueRounds {
		generated := GenerateLeagueFixtures(tm.ClubsList, tm.RNG)

		finishedOld := make([]Fixture, 0)
		for _, f := range tm.Fixtures {
			if f.Status == "finished" {
				finishedOld = append(finishedOld, f)
			}
		}
		sort.SliceStable(finishedOld, func(i, j int) bool {
			if finishedOld[i].Matchweek != finishedOld[j].Matchweek {
				return finishedOld[i].Matchweek < finishedOld[j].Matchweek
			}
			return finishedOld[i].FixtureID < finishedOld[j].FixtureID
		})

		claimedGenerated := make(map[int]bool)
		claimedOld := make(map[string]bool)
		remapping := make(map[string]string)

		// 1. Match by exact FixtureID first
		for j, gen := range generated {
			for _, old := range finishedOld {
				if !claimedOld[old.FixtureID] && gen.FixtureID == old.FixtureID {
					claimedGenerated[j] = true
					claimedOld[old.FixtureID] = true
					break
				}
			}
		}

		isLeagueComp := func(comp string) bool {
			c := strings.ToLower(strings.TrimSpace(comp))
			return c == "" || c == "super-league" || c == "super_league" || c == "super league" || c == "league"
		}

		// 2. For unmapped finished fixtures, match by (Competition, HomeClubID, AwayClubID)
		// sequentially in chronological matchweek order
		for _, old := range finishedOld {
			if claimedOld[old.FixtureID] {
				continue
			}
			for j, gen := range generated {
				if claimedGenerated[j] {
					continue
				}
				if gen.HomeID == old.HomeID && gen.AwayID == old.AwayID &&
					(gen.Competition == old.Competition || (isLeagueComp(gen.Competition) && isLeagueComp(old.Competition))) {
					claimedGenerated[j] = true
					claimedOld[old.FixtureID] = true
					remapping[old.FixtureID] = gen.FixtureID
					break
				}
			}
		}

		oldMap := make(map[string]Fixture, len(finishedOld))
		for _, f := range finishedOld {
			oldMap[f.FixtureID] = f
		}
		reverseRemap := make(map[string]string)
		for oldID, genID := range remapping {
			reverseRemap[genID] = oldID
		}

		merged := make([]Fixture, 0, len(generated))
		for _, gen := range generated {
			if old, ok := oldMap[gen.FixtureID]; ok && claimedOld[gen.FixtureID] && remapping[gen.FixtureID] == "" {
				merged = append(merged, old)
			} else if oldID, ok := reverseRemap[gen.FixtureID]; ok {
				old := oldMap[oldID]
				f := gen
				f.Status = old.Status
				f.HomeGoals = old.HomeGoals
				f.AwayGoals = old.AwayGoals
				f.Report = old.Report
				f.DecidedBy = old.DecidedBy
				f.Penalties = old.Penalties
				f.Method = old.Method
				merged = append(merged, f)
			} else {
				merged = append(merged, gen)
			}
		}

		tm.Fixtures = merged
		tm.MaxMatchweeks = LeagueRounds
		if tm.CurrentMatchweek > LeagueRounds+1 {
			tm.CurrentMatchweek = LeagueRounds + 1
		}

		// Remap InboxItem.FixtureID where applicable
		if len(remapping) > 0 {
			for i := range tm.Inbox {
				if newID, ok := remapping[tm.Inbox[i].FixtureID]; ok {
					tm.Inbox[i].FixtureID = newID
				}
			}
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
	tm.ReputationAppliedSeason = ""
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
