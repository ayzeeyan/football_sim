package tournament

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
	"sync"

	"football_sim/pkg/growth"
	"football_sim/pkg/managers"
	"football_sim/pkg/matchreport"
	"football_sim/pkg/models"
	"football_sim/pkg/transfers"
)

// TournamentManager coordinates the entire European Super League calendar,
// match simulation, standings, narratives, and multi-decade records.
type TournamentManager struct {
	mu                    sync.RWMutex
	Clubs                 map[string]*models.Club
	ClubsList             []*models.Club
	Managers              map[string]*managers.ManagerProfile
	GrowthEngine          *growth.GrowthEngine
	CurrentMatchweek      int
	MaxMatchweeks         int
	SeasonName            string
	SeasonPhase           string // season, transfer_window
	Fixtures              []Fixture
	UCLFixtures           []Fixture
	SuperCupFixtures      []Fixture
	Inbox                 []InboxItem
	InboxSeq              int
	DerbyHeat             map[string]int
	DerbiesPlayedThisMW   map[string]bool
	MilestonesFired       map[string]map[string]bool
	ManagerConsecutiveHot map[string]int
	SeasonHistory         []map[string]interface{}
	ClubSeasonHistory     map[string][]map[string]interface{}
	TransferEngine        *transfers.TransferEngine // optional; enables market-watch wire
	ProdigyHomes          map[string]string
	LastCareerShuffle     bool
	FavouriteClubID       string
	RNG                   *rand.Rand

	RecentResults       []string
	GrowthNotifications []string
	PlayerOfTheWeek     map[string]interface{}
	MonthlyAwards       []map[string]interface{}
	MatchweekWeather    map[int]string
	MoraleStoryStatus   map[string]string

	UCLGroupA             []*models.Club
	UCLGroupB             []*models.Club
	UCLRecords            map[string]*models.CompetitionRecord
	UCLStage              string
	UCLQuarterFinals      map[string]CupTie
	UCLSemiFinals         map[string]CupTie
	UCLFinal              CupTie
	UCLChampionID         string
	SuperCupPlayIn        map[string]CupTie
	SuperCupQuarterFinals map[string]CupTie
	SuperCupSemiFinals    map[string]CupTie
	SuperCupFinal         CupTie
	SuperCupChampionID    string
	SuperCupStage         string
	SuperCupByes          []*models.Club
}

// NewTournamentManager initializes a complete TournamentManager instance.
func NewTournamentManager(eliteClubs []*models.Club, ge *growth.GrowthEngine, seed int64) *TournamentManager {
	if seed == 0 {
		seed = 20260907
	}
	rng := rand.New(rand.NewSource(seed))

	clubsMap := make(map[string]*models.Club)
	for _, c := range eliteClubs {
		clubsMap[c.ClubID] = c
	}

	mgrs := managers.BuildManagers(eliteClubs)
	fixtures := GenerateLeagueFixtures(eliteClubs, rng)

	heat := make(map[string]int)
	for _, name := range DerbyNames {
		if _, ok := heat[name]; !ok {
			heat[name] = 50
		}
	}

	tm := &TournamentManager{
		Clubs:                 clubsMap,
		ClubsList:             eliteClubs,
		Managers:              mgrs,
		GrowthEngine:          ge,
		CurrentMatchweek:      1,
		MaxMatchweeks:         LeagueRounds,
		SeasonName:            "2026-27",
		SeasonPhase:           "season",
		Fixtures:              fixtures,
		Inbox:                 make([]InboxItem, 0),
		DerbyHeat:             heat,
		DerbiesPlayedThisMW:   map[string]bool{},
		MilestonesFired:       make(map[string]map[string]bool),
		ManagerConsecutiveHot: make(map[string]int),
		SeasonHistory:         make([]map[string]interface{}, 0),
		ClubSeasonHistory:     map[string][]map[string]interface{}{},
		RNG:                   rng,
		MoraleStoryStatus:     map[string]string{},
		MatchweekWeather:      map[int]string{},
		UCLRecords:            map[string]*models.CompetitionRecord{},
		UCLQuarterFinals:      map[string]CupTie{},
		UCLSemiFinals:         map[string]CupTie{},
		SuperCupPlayIn:        map[string]CupTie{},
		SuperCupQuarterFinals: map[string]CupTie{},
		SuperCupSemiFinals:    map[string]CupTie{},
	}
	for _, c := range eliteClubs {
		tm.MoraleStoryStatus[c.ClubID] = "normal"
		tm.ClubSeasonHistory[c.ClubID] = nil
	}
	for mw := 1; mw <= 44; mw++ {
		tm.MatchweekWeather[mw] = WeatherOptions[rng.Intn(len(WeatherOptions))]
	}

	PairSeniorMentors(eliteClubs, ge)
	tm.initCups()
	tm.seedOpeningInbox()
	return tm
}

func (tm *TournamentManager) seedOpeningInbox() {
	for _, i := range tm.Inbox {
		if i.Category == "system" && i.SeasonName == tm.SeasonName {
			return
		}
	}
	tm.PushInbox(
		"system",
		tm.SeasonName+" Super League opens",
		"Twelve clubs, 33 matchweeks. Champions Cup and Super Cup share the slate. Watch the kids grow.",
		1,
		nil, "", "",
	)
}

// PushInbox appends a news wire item.
func (tm *TournamentManager) PushInbox(
	category string,
	headline string,
	body string,
	matchweek int,
	clubIDs []string,
	playerID string,
	fixtureID string,
) {
	for _, existing := range tm.Inbox {
		if existing.Headline == headline && existing.SeasonName == tm.SeasonName && existing.Matchweek == matchweek && existing.FixtureID == fixtureID {
			return
		}
	}
	tm.InboxSeq++
	id := fmt.Sprintf("IN_%s_%d_%d", tm.SeasonName, matchweek, tm.InboxSeq)
	item := NewInboxItem(id, category, headline, body, matchweek, tm.SeasonName, clubIDs, playerID, fixtureID)
	tm.Inbox = append([]InboxItem{item}, tm.Inbox...)
	if len(tm.Inbox) > 180 {
		tm.Inbox = tm.Inbox[:180]
	}
}

// GetStandings returns the sorted league table.
func (tm *TournamentManager) GetStandings() []*models.Club {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.standingsUnlocked()
}

// KickoffNote is the pre-match line React shows above the probable XIs.
func (tm *TournamentManager) KickoffNote(f *Fixture, home, away *models.Club) string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.kickoffNoteUnlocked(f, home, away)
}

func (tm *TournamentManager) kickoffNoteUnlocked(f *Fixture, home, away *models.Club) string {
	if f == nil || home == nil || away == nil {
		return ""
	}
	if f.DerbyName != "" {
		return fmt.Sprintf("%s at %s.", f.DerbyName, home.HomeStadium)
	}
	if f.Competition == "super-cup" {
		if f.Stage == "Final" {
			return fmt.Sprintf("Super Cup final night at %s.", home.HomeStadium)
		}
		stage := f.Stage
		if stage == "" {
			stage = "tie"
		}
		return fmt.Sprintf("Super Cup %s. One night — extra time if it is level.", stage)
	}
	if f.Competition == "ucl" {
		if f.Stage == "Final" {
			return fmt.Sprintf("Champions Cup final night at %s.", home.HomeStadium)
		}
		if f.Leg == 2 {
			return fmt.Sprintf("Leg 2 of the %s at %s.", f.Stage, home.HomeStadium)
		}
		if f.Leg == 1 {
			return fmt.Sprintf("Leg 1 of the %s at %s.", f.Stage, home.HomeStadium)
		}
		if strings.HasPrefix(f.Stage, "Group") {
			status := tm.uclGroupStatus(home.ClubID)
			switch status {
			case "must_win":
				return fmt.Sprintf("Must-win at %s.", home.HomeStadium)
			case "qualified":
				return fmt.Sprintf("%s are already through. Champions Cup night at %s.", home.ShortName, home.HomeStadium)
			case "eliminated":
				return fmt.Sprintf("Already out. Pride at %s.", home.HomeStadium)
			}
			return fmt.Sprintf("Champions Cup %s at %s.", f.Stage, home.HomeStadium)
		}
		return fmt.Sprintf("Champions Cup %s at %s.", f.Stage, home.HomeStadium)
	}
	table := tm.standingsUnlocked()
	pos := map[string]int{}
	for i, c := range table {
		pos[c.ClubID] = i + 1
	}
	hp, ap := pos[home.ClubID], pos[away.ClubID]
	if hp > 0 && ap > 0 {
		if hp <= 3 && ap <= 3 && (home.Played > 0 || away.Played > 0) {
			return fmt.Sprintf("A top-of-the-table meeting. %s %dth, %s %dth.", home.ShortName, hp, away.ShortName, ap)
		}
		if min(hp, ap) >= 10 && absInt(hp-ap) <= 2 && (home.Played > 0 || away.Played > 0) {
			return fmt.Sprintf("A scrap at the wrong end. %s %dth, %s %dth.", home.ShortName, hp, away.ShortName, ap)
		}
		if hp == 1 && home.Played > 0 {
			return fmt.Sprintf("The leaders host %s.", away.ClubName)
		}
		if ap == 1 && away.Played > 0 {
			return fmt.Sprintf("%s host the league leaders.", home.ClubName)
		}
	}
	if models.IsExamWeek(f.Matchweek) {
		return fmt.Sprintf("Exam week — enrolled prodigies sit. %s · %s at %s.", LeaguePhase(f.Matchweek), MonthLabel(f.Matchweek), home.HomeStadium)
	}
	return fmt.Sprintf("%s · %s at %s.", LeaguePhase(f.Matchweek), MonthLabel(f.Matchweek), home.HomeStadium)
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// GetMatchweekFixtures returns all fixtures for a specific round.
func (tm *TournamentManager) GetMatchweekFixtures(mw int) []Fixture {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var result []Fixture
	for _, f := range tm.Fixtures {
		if f.Matchweek == mw {
			result = append(result, f)
		}
	}
	return result
}

// SimulateMatchweek simulates the current week's remaining slate then rolls over.
func (tm *TournamentManager) SimulateMatchweek(mw int) map[string]interface{} {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if mw < 1 || mw > tm.MaxMatchweeks {
		return map[string]interface{}{"status": "error", "message": "Invalid matchweek"}
	}
	played := 0
	var ids []string
	collect := func(list []Fixture) {
		for i := range list {
			if list[i].Matchweek == mw && list[i].Status != "finished" {
				ids = append(ids, list[i].FixtureID)
			}
		}
	}
	collect(tm.Fixtures)
	collect(tm.UCLFixtures)
	collect(tm.SuperCupFixtures)
	saved := tm.CurrentMatchweek
	if saved < 1 {
		saved = 1
		tm.CurrentMatchweek = 1
	}
	if mw > saved {
		tm.CurrentMatchweek = mw
	}
	for _, id := range ids {
		res := tm.simulateFixtureUnlocked(id)
		if res["status"] == "success" {
			played++
		}
	}
	if mw > saved && tm.CurrentMatchweek == mw {
		// Targeted future week sim (tests): restore calendar if rollover didn't fire.
		tm.CurrentMatchweek = saved
	}
	return map[string]interface{}{
		"status":          "success",
		"played":          played,
		"simulated_count": played,
		"matchweek":       mw,
		"next_matchweek":  tm.CurrentMatchweek,
		"is_finished":     tm.CurrentMatchweek > tm.MaxMatchweeks,
		"champion":        champIf(tm.CurrentMatchweek > tm.MaxMatchweeks, tm.championNameUnlocked()),
	}
}

// GetHeadToHead computes historical matchup statistics between two clubs.
func (tm *TournamentManager) GetHeadToHead(clubAID, clubBID string) map[string]interface{} {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	clubA := tm.Clubs[clubAID]
	clubB := tm.Clubs[clubBID]
	if clubA == nil || clubB == nil {
		return nil
	}

	var matchesPlayed, winsA, winsB, draws, goalsA, goalsB int
	recentMatches := make([]map[string]interface{}, 0)
	record := func(f Fixture) {
		if f.Status != "finished" || f.HomeGoals == nil || f.AwayGoals == nil {
			return
		}
		pair := (f.HomeID == clubAID && f.AwayID == clubBID) || (f.HomeID == clubBID && f.AwayID == clubAID)
		if !pair {
			return
		}
		matchesPlayed++
		hg, ag := *f.HomeGoals, *f.AwayGoals
		if f.HomeID == clubAID {
			goalsA += hg
			goalsB += ag
		} else {
			goalsA += ag
			goalsB += hg
		}
		winner := "draw"
		if hg > ag {
			winner = f.HomeID
			if f.HomeID == clubAID {
				winsA++
			} else {
				winsB++
			}
		} else if ag > hg {
			winner = f.AwayID
			if f.AwayID == clubAID {
				winsA++
			} else {
				winsB++
			}
		} else {
			draws++
		}
		recentMatches = append(recentMatches, map[string]interface{}{
			"id":          f.FixtureID,
			"fixture_id":  f.FixtureID,
			"matchweek":   f.Matchweek,
			"competition": f.Competition,
			"stage":       f.Stage,
			"home_id":     f.HomeID,
			"away_id":     f.AwayID,
			"home_goals":  hg,
			"away_goals":  ag,
			"winner":      winner,
		})
	}
	for _, f := range tm.Fixtures {
		record(f)
	}
	for _, f := range tm.UCLFixtures {
		record(f)
	}
	for _, f := range tm.SuperCupFixtures {
		record(f)
	}
	sort.Slice(recentMatches, func(i, j int) bool {
		mi, _ := recentMatches[i]["matchweek"].(int)
		mj, _ := recentMatches[j]["matchweek"].(int)
		if mi != mj {
			return mi > mj
		}
		idi, _ := recentMatches[i]["id"].(string)
		idj, _ := recentMatches[j]["id"].(string)
		return idi > idj
	})
	if len(recentMatches) > 10 {
		recentMatches = recentMatches[:10]
	}

	derbyName := GetDerbyName(clubAID, clubBID)
	heat := 50
	if derbyName != "" {
		heat = tm.derbyHeatUnlocked(derbyName)
	}

	return map[string]interface{}{
		"club_a": map[string]interface{}{
			"club_id":       clubA.ClubID,
			"club_name":     clubA.ClubName,
			"short_name":    clubA.ShortName,
			"primary_color": clubA.PrimaryColor,
		},
		"club_b": map[string]interface{}{
			"club_id":       clubB.ClubID,
			"club_name":     clubB.ClubName,
			"short_name":    clubB.ShortName,
			"primary_color": clubB.PrimaryColor,
		},
		"matches_played": matchesPlayed,
		"wins_a":         winsA,
		"wins_b":         winsB,
		"draws":          draws,
		"goals_a":        goalsA,
		"goals_b":        goalsB,
		"derby_name":     derbyName,
		"derby_heat":     heat,
		"recent_matches": recentMatches,
	}
}

// GetTrophyCabinet compiles silverware counts combining historical and career titles.
func (tm *TournamentManager) GetTrophyCabinet() []map[string]interface{} {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	extraSL, extraUCL, extraSC := map[string]int{}, map[string]int{}, map[string]int{}
	for cid, hist := range tm.ClubSeasonHistory {
		for _, row := range hist {
			trophies, _ := row["trophies"].([]string)
			for _, t := range trophies {
				switch t {
				case "Super League Champion":
					extraSL[cid]++
				case "Champions Cup":
					extraUCL[cid]++
				case "Super Cup":
					extraSC[cid]++
				}
			}
		}
	}
	var cabinet []map[string]interface{}
	for _, club := range tm.ClubsList {
		base := HistoricalClubTrophies[club.ClubID]
		cSL, cUCL, cSC := extraSL[club.ClubID], extraUCL[club.ClubID], extraSC[club.ClubID]
		sl := base.SuperLeague + cSL
		ucl := base.UCL + cUCL
		sc := base.SuperCup + cSC
		cabinet = append(cabinet, map[string]interface{}{
			"club_id":            club.ClubID,
			"club_name":          club.ClubName,
			"short_name":         club.ShortName,
			"primary_color":      club.PrimaryColor,
			"total_trophies":     sl + ucl + sc,
			"super_league_count": sl,
			"ucl_count":          ucl,
			"super_cup_count":    sc,
			// Split view for the mixed history card: heritage vs dynasty.
			"hist_super_league":   base.SuperLeague,
			"hist_ucl":            base.UCL,
			"hist_super_cup":      base.SuperCup,
			"hist_total":          base.SuperLeague + base.UCL + base.SuperCup,
			"career_super_league": cSL,
			"career_ucl":          cUCL,
			"career_super_cup":    cSC,
			"career_total":        cSL + cUCL + cSC,
		})
	}
	return cabinet
}

// GetSeasonAwards compiles end-of-season awards including Ballon d'Or and Team of the Season.
func (tm *TournamentManager) GetSeasonAwards() map[string]interface{} {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.seasonAwardsUnlocked()
}

func (tm *TournamentManager) seasonAwardsUnlocked() map[string]interface{} {
	standings := tm.standingsUnlocked()
	var champ, runner *models.Club
	if len(standings) > 0 {
		champ = standings[0]
	}
	if len(standings) > 1 {
		runner = standings[1]
	}

	var allPlayers []*models.Player
	for _, c := range tm.ClubsList {
		allPlayers = append(allPlayers, c.Squad...)
	}

	sort.Slice(allPlayers, func(i, j int) bool {
		scoreI := float64(allPlayers[i].OVR) + float64(allPlayers[i].Goals)*2.5 + float64(allPlayers[i].Assists)*1.5
		scoreJ := float64(allPlayers[j].OVR) + float64(allPlayers[j].Goals)*2.5 + float64(allPlayers[j].Assists)*1.5
		return scoreI > scoreJ
	})

	var ballonDor []map[string]interface{}
	for idx, p := range allPlayers {
		if idx >= 10 {
			break
		}
		clubShort := p.ClubID
		if c := tm.Clubs[p.ClubID]; c != nil {
			clubShort = c.ShortName
		}
		score := p.OVR + p.Goals*2 + p.Assists
		ballonDor = append(ballonDor, map[string]interface{}{
			"rank":         idx + 1,
			"player_id":    p.PlayerID,
			"full_name":    p.FullName,
			"club_short":   clubShort,
			"position":     p.Position,
			"ovr":          p.OVR,
			"goals":        p.Goals,
			"assists":      p.Assists,
			"score":        score,
			"is_wonderkid": p.UniverseWonderkid,
		})
	}

	clubMini := func(c *models.Club) map[string]interface{} {
		if c == nil {
			return nil
		}
		return map[string]interface{}{
			"club_id":    c.ClubID,
			"club_name":  c.ClubName,
			"short_name": c.ShortName,
			"pts":        c.Points,
			"points":     c.Points,
		}
	}
	playerMini := func(p *models.Player, extra map[string]interface{}) map[string]interface{} {
		if p == nil {
			return nil
		}
		m := map[string]interface{}{
			"player_id":    p.PlayerID,
			"full_name":    p.FullName,
			"position":     p.Position,
			"ovr":          p.OVR,
			"goals":        p.Goals,
			"assists":      p.Assists,
			"club_id":      p.ClubID,
			"is_wonderkid": p.UniverseWonderkid,
		}
		if c := tm.Clubs[p.ClubID]; c != nil {
			m["club_name"] = c.ClubName
			m["short_name"] = c.ShortName
			m["club_short"] = c.ShortName
		}
		for k, v := range extra {
			m[k] = v
		}
		return m
	}

	var topScorer, topAssister, goldenBoy, pots *models.Player
	for _, p := range allPlayers {
		if topScorer == nil || p.Goals > topScorer.Goals || (p.Goals == topScorer.Goals && p.OVR > topScorer.OVR) {
			topScorer = p
		}
		if topAssister == nil || p.Assists > topAssister.Assists || (p.Assists == topAssister.Assists && p.OVR > topAssister.OVR) {
			topAssister = p
		}
		score := float64(p.Goals)*3 + float64(p.Assists)*2 + float64(p.Appearances)*0.5 + float64(p.OVR)
		if pots == nil {
			pots = p
		} else {
			best := float64(pots.Goals)*3 + float64(pots.Assists)*2 + float64(pots.Appearances)*0.5 + float64(pots.OVR)
			if score > best {
				pots = p
			}
		}
		if p.UniverseWonderkid {
			gb := p.Goals*3 + p.Assists*2 + p.Appearances + p.OVR
			if goldenBoy == nil {
				goldenBoy = p
			} else {
				best := goldenBoy.Goals*3 + goldenBoy.Assists*2 + goldenBoy.Appearances + goldenBoy.OVR
				if gb > best {
					goldenBoy = p
				}
			}
		}
	}

	var uclChamp, scChamp *models.Club
	if tm.UCLChampionID != "" {
		uclChamp = tm.Clubs[tm.UCLChampionID]
	}
	if tm.SuperCupChampionID != "" {
		scChamp = tm.Clubs[tm.SuperCupChampionID]
	}

	return map[string]interface{}{
		"season_name":            tm.SeasonName,
		"super_league_champion":  clubMini(champ),
		"super_league_runner_up": clubMini(runner),
		"ucl_champion":           clubMini(uclChamp),
		"super_cup_champion":     clubMini(scChamp),
		"top_scorer":             playerMini(topScorer, nil),
		"top_assister":           playerMini(topAssister, nil),
		"golden_boy":             playerMini(goldenBoy, nil),
		"player_of_the_season":   playerMini(pots, nil),
		"ballon_dor":             ballonDor,
		"monthly_awards":         tm.MonthlyAwards,
		"player_of_the_week":     tm.PlayerOfTheWeek,
	}
}

// GetAllTimeRecords compiles records across goals, assists, fixtures, and wonderkids.
func (tm *TournamentManager) GetAllTimeRecords() map[string]interface{} {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	type playerClub struct {
		p *models.Player
		c *models.Club
	}
	var allPlayers []playerClub
	for _, c := range tm.ClubsList {
		for _, p := range c.Squad {
			allPlayers = append(allPlayers, playerClub{p: p, c: c})
		}
	}

	// Top goalscorers
	sort.Slice(allPlayers, func(i, j int) bool {
		goalsI := allPlayers[i].p.Goals + allPlayers[i].p.CareerGoals
		goalsJ := allPlayers[j].p.Goals + allPlayers[j].p.CareerGoals
		if goalsI != goalsJ {
			return goalsI > goalsJ
		}
		return allPlayers[i].p.OVR > allPlayers[j].p.OVR
	})

	var topScorers []map[string]interface{}
	for i := 0; i < 10 && i < len(allPlayers); i++ {
		p := allPlayers[i].p
		c := allPlayers[i].c
		topScorers = append(topScorers, map[string]interface{}{
			"player_id":    p.PlayerID,
			"full_name":    p.FullName,
			"club_name":    c.ClubName,
			"short_name":   c.ShortName,
			"goals":        p.Goals + p.CareerGoals,
			"appearances":  p.Appearances + p.CareerApps,
			"ovr":          p.OVR,
			"is_wonderkid": p.UniverseWonderkid,
		})
	}

	// Top assisters
	sort.Slice(allPlayers, func(i, j int) bool {
		assistsI := allPlayers[i].p.Assists + allPlayers[i].p.CareerAssists
		assistsJ := allPlayers[j].p.Assists + allPlayers[j].p.CareerAssists
		if assistsI != assistsJ {
			return assistsI > assistsJ
		}
		return allPlayers[i].p.OVR > allPlayers[j].p.OVR
	})

	var topAssisters []map[string]interface{}
	for i := 0; i < 10 && i < len(allPlayers); i++ {
		p := allPlayers[i].p
		c := allPlayers[i].c
		topAssisters = append(topAssisters, map[string]interface{}{
			"player_id":    p.PlayerID,
			"full_name":    p.FullName,
			"club_name":    c.ClubName,
			"short_name":   c.ShortName,
			"assists":      p.Assists + p.CareerAssists,
			"appearances":  p.Appearances + p.CareerApps,
			"ovr":          p.OVR,
			"is_wonderkid": p.UniverseWonderkid,
		})
	}

	// Highest scoring match & biggest margin
	var highestScoringMatch map[string]interface{}
	var biggestMargin map[string]interface{}
	maxGoals := -1
	maxMargin := -1

	for _, f := range tm.Fixtures {
		if f.Status != "finished" || f.HomeGoals == nil || f.AwayGoals == nil {
			continue
		}
		hg := *f.HomeGoals
		ag := *f.AwayGoals
		tot := hg + ag
		diff := int(math.Abs(float64(hg - ag)))

		hClub := tm.Clubs[f.HomeID]
		aClub := tm.Clubs[f.AwayID]
		hName := f.HomeID
		if hClub != nil {
			hName = hClub.ShortName
		}
		aName := f.AwayID
		if aClub != nil {
			aName = aClub.ShortName
		}

		if tot > maxGoals {
			maxGoals = tot
			highestScoringMatch = map[string]interface{}{
				"fixture_id":  f.FixtureID,
				"matchweek":   f.Matchweek,
				"score":       fmt.Sprintf("%s %d - %d %s", hName, hg, ag, aName),
				"goals":       tot,
				"total_goals": tot,
			}
		}
		if diff > maxMargin {
			maxMargin = diff
			winner := ""
			if hg > ag {
				winner = hName
			} else if ag > hg {
				winner = aName
			}
			biggestMargin = map[string]interface{}{
				"fixture_id":  f.FixtureID,
				"matchweek":   f.Matchweek,
				"score":       fmt.Sprintf("%s %d - %d %s", hName, hg, ag, aName),
				"margin":      diff,
				"winner":      winner,
				"competition": "super-league",
			}
		}
	}

	// Single-match goals record: best individual haul in any finished report.
	singleBest := map[string]interface{}{"player_name": "—", "club_name": "—", "goals": 0, "fixture": "—"}
	bestHaul := 0
	scanRows := func(f *Fixture, rows []matchreport.MatchPlayerRow) {
		for _, row := range rows {
			if row.MatchGoals <= bestHaul {
				continue
			}
			bestHaul = row.MatchGoals
			name, club := row.FullName, "—"
			for _, cid := range []string{f.HomeID, f.AwayID} {
				if c := tm.Clubs[cid]; c != nil {
					for _, p := range c.Squad {
						if p.PlayerID == row.PlayerID {
							name, club = p.FullName, c.ShortName
						}
					}
				}
			}
			singleBest = map[string]interface{}{
				"player_name": name,
				"club_name":   club,
				"goals":       row.MatchGoals,
				"fixture":     f.FixtureID,
			}
		}
	}
	for i := range tm.Fixtures {
		f := &tm.Fixtures[i]
		if f.Status != "finished" || f.Report == nil {
			continue
		}
		scanRows(f, f.Report.HomeXI)
		scanRows(f, f.Report.AwayXI)
	}

	// Highest OVR Wonderkid
	var highestOVRWK map[string]interface{}
	maxWKOVR := -1
	for _, c := range tm.ClubsList {
		for _, p := range c.Squad {
			if p.UniverseWonderkid && p.OVR > maxWKOVR {
				maxWKOVR = p.OVR
				pot := 95
				if tm.GrowthEngine != nil {
					if bio, ok := tm.GrowthEngine.Biometrics[p.PlayerID]; ok {
						pot = bio.Potential
					}
				}
				highestOVRWK = map[string]interface{}{
					"player_id":  p.PlayerID,
					"full_name":  p.FullName,
					"club_short": c.ShortName,
					"ovr":        p.OVR,
					"potential":  pot,
				}
			}
		}
	}

	var topScorersI, topAssistersI, hsI, bmI, wkI interface{}
	if len(topScorers) > 0 {
		topScorersI = topScorers
	}
	if len(topAssisters) > 0 {
		topAssistersI = topAssisters
	}
	if highestScoringMatch != nil {
		hsI = highestScoringMatch
	}
	if biggestMargin != nil {
		bmI = biggestMargin
	}
	if highestOVRWK != nil {
		wkI = highestOVRWK
	}
	// Highest points in a single season: archived champions first, then the
	// live table. Always populated so the record book never nulls.
	bestPts := map[string]interface{}{"club_name": "—", "season_name": tm.SeasonName, "points": 0}
	bestPtsN := -1
	for _, row := range tm.SeasonHistory {
		champ, _ := row["champion"].(map[string]interface{})
		if champ == nil {
			continue
		}
		pts := intVal(champ["pts"])
		if pts < 0 {
			pts = intVal(champ["points"])
		}
		if pts > bestPtsN {
			bestPtsN = pts
			name, _ := champ["club_name"].(string)
			season, _ := row["season_name"].(string)
			if name == "" {
				name = "—"
			}
			if season == "" {
				season = tm.SeasonName
			}
			bestPts = map[string]interface{}{"club_name": name, "season_name": season, "points": pts}
		}
	}
	if standings := tm.standingsUnlocked(); len(standings) > 0 && standings[0].Points > bestPtsN {
		bestPtsN = standings[0].Points
		bestPts = map[string]interface{}{"club_name": standings[0].ClubName, "season_name": tm.SeasonName, "points": standings[0].Points}
	}
	// Wonderkid milestones in record-book shape.
	wkBest := map[string]interface{}{"name": "—", "ovr": 0, "potential": 0}
	if highestOVRWK != nil {
		name, _ := highestOVRWK["full_name"].(string)
		ovr, _ := highestOVRWK["ovr"].(int)
		pot, _ := highestOVRWK["potential"].(int)
		wkBest = map[string]interface{}{"name": name, "ovr": ovr, "potential": pot}
	}
	topProdigy := map[string]interface{}{"name": "—", "goals": 0}
	prodigyGoals := -1
	for _, c := range tm.ClubsList {
		for _, p := range c.Squad {
			if !p.UniverseWonderkid {
				continue
			}
			if g := p.Goals + p.CareerGoals; g > prodigyGoals {
				prodigyGoals = g
				topProdigy = map[string]interface{}{"name": p.FullName, "goals": g}
			}
		}
	}
	return map[string]interface{}{
		"top_goalscorers":       topScorersI,
		"top_assisters":         topAssistersI,
		"highest_scoring_match": hsI,
		"biggest_margin":        bmI,
		"highest_ovr_wonderkid": wkI,
		// Record-book aliases matching the React contract.
		"biggest_margin_victory":    bmI,
		"single_match_goals_record": singleBest,
		"highest_season_points":     bestPts,
		"wonderkid_milestones": map[string]interface{}{
			"highest_ovr":       wkBest,
			"top_prodigy_goals": topProdigy,
		},
	}
}

// intVal coerces JSON-ish numbers without failing on unexpected shapes.
func intVal(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return -1
	}
}
