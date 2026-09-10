package models

import (
	"encoding/json"
	"math"
	"sort"
	"strconv"
	"strings"
)

// Club represents a football club, its roster, financial/stadium info, and domestic standings.
type Club struct {
	ClubID            string       `json:"club_id"`
	ClubName          string       `json:"club_name"`
	ShortName         string       `json:"short_name"`
	League            string       `json:"league"`
	Country           string       `json:"country"`
	HomeStadium       string       `json:"home_stadium"`
	StadiumCapacity   int          `json:"stadium_capacity"`
	OverallTeamRating int          `json:"overall_team_rating"`
	SquadSize         int          `json:"squad_size"`
	SquadAvgOVR       float64      `json:"squad_avg_ovr"`
	PrimaryColor      [3]uint8     `json:"primary_color"`
	SecondaryColor    [3]uint8     `json:"secondary_color"`
	Morale            int          `json:"morale"`
	Identity          ClubIdentity `json:"identity"`
	Finances          ClubFinances `json:"finances"`

	// Standings & Form
	Played         int      `json:"p"`
	Won            int      `json:"w"`
	Drawn          int      `json:"d"`
	Lost           int      `json:"l"`
	GoalsFor       int      `json:"gf"`
	GoalsAgainst   int      `json:"ga"`
	GoalDifference int      `json:"gd"`
	Points         int      `json:"pts"`
	Form           []string `json:"form"`

	// Squad
	Squad []*Player `json:"squad"`
}

type clubAlias Club

// UnmarshalJSON implements custom JSON decoding with kit colors, morale, squad,
// identity and old-save defaults.
func (c *Club) UnmarshalJSON(data []byte) error {
	var raw clubAlias
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*c = Club(raw)

	if c.Morale == 0 {
		c.Morale = 70
	}
	if c.Morale < 0 {
		c.Morale = 0
	} else if c.Morale > 100 {
		c.Morale = 100
	}

	// A zero identity means the block was absent in a legacy/static-data payload.
	// Capture that fact before clamping so a legitimate persisted zero-valued
	// individual trait remains zero rather than being treated as missing.
	identityMissing := c.Identity.IsZero()
	if identityMissing {
		c.Identity = DefaultClubIdentity(c.ClubID, c.OverallTeamRating)
	} else {
		c.Identity = c.Identity.Clamp()
	}
	if identityMissing && c.Finances == (ClubFinances{}) {
		c.Finances = InitialClubFinances(c.Identity)
	}
	// Corrupt finance values are made safe without refilling a legitimately
	// depleted warchest.
	if c.Finances.TransferBudget < 0 {
		c.Finances.TransferBudget = 0
	}
	if c.Finances.Balance < 0 {
		c.Finances.Balance = 0
	}

	// Kit colors fallback
	if c.PrimaryColor == [3]uint8{0, 0, 0} && c.SecondaryColor == [3]uint8{0, 0, 0} {
		c.PrimaryColor, c.SecondaryColor = KitColorsForClub(c.ShortName)
	}

	if c.Form == nil {
		c.Form = []string{}
	}

	// Link squad players. Current ClubID is authoritative once present;
	// OriginalClubID is historical metadata and never overwrites membership.
	for _, p := range c.Squad {
		if p.ClubID == "" {
			p.ClubID = c.ClubID
		}
		if p.OriginalClubID == "" {
			p.OriginalClubID = c.ClubID
		}
	}

	return nil
}

// UpdateMorale modifies club dressing room morale based on match outcome (W, D, L).
func (c *Club) UpdateMorale(result string) {
	switch result {
	case "W":
		streakBonus := 0
		if len(c.Form) >= 3 {
			last3 := c.Form[len(c.Form)-3:]
			if last3[0] == "W" && last3[1] == "W" && last3[2] == "W" {
				streakBonus = 5
			}
		}
		c.Morale += 3 + streakBonus
		if c.Morale > 100 {
			c.Morale = 100
		}
	case "D":
		c.Morale -= 1
		if c.Morale < 30 {
			c.Morale = 30
		}
	case "L":
		streakPenalty := 0
		if len(c.Form) >= 3 {
			last3 := c.Form[len(c.Form)-3:]
			if last3[0] == "L" && last3[1] == "L" && last3[2] == "L" {
				streakPenalty = 5
			}
		}
		c.Morale -= 3 + streakPenalty
		if c.Morale < 20 {
			c.Morale = 20
		}
	}
}

// UpdateResult records a completed domestic match, updates table metrics, and triggers morale shifts.
func (c *Club) UpdateResult(gf, ga int) {
	c.Played++
	c.GoalsFor += gf
	c.GoalsAgainst += ga
	c.GoalDifference = c.GoalsFor - c.GoalsAgainst

	if gf > ga {
		c.Won++
		c.Points += 3
		c.Form = append(c.Form, "W")
		c.UpdateMorale("W")
	} else if gf == ga {
		c.Drawn++
		c.Points += 1
		c.Form = append(c.Form, "D")
		c.UpdateMorale("D")
	} else {
		c.Lost++
		c.Form = append(c.Form, "L")
		c.UpdateMorale("L")
	}

	if len(c.Form) > 5 {
		c.Form = c.Form[len(c.Form)-5:]
	}
}

// parseFixtureArgs extracts competition and matchweek from optional string arguments.
func parseFixtureArgs(args []string) (string, int) {
	comp := "super-league"
	week := 1
	if len(args) == 0 || args[0] == "" {
		return comp, week
	}

	parts := strings.Split(args[0], ":")
	if len(parts) > 0 && parts[0] != "" {
		comp = parts[0]
	}
	if len(parts) > 1 {
		if w, err := strconv.Atoi(parts[1]); err == nil {
			week = w
		}
	}
	return comp, week
}

// AvailableSquad returns players eligible to feature in a fixture.
func (c *Club) AvailableSquad(fixture ...string) []*Player {
	comp, week := parseFixtureArgs(fixture)
	var ready []*Player
	for _, p := range c.Squad {
		if p != nil && !p.IsUnavailable(comp, week) {
			ready = append(ready, p)
		}
	}
	return ready
}

// sortKey computes wonderkid priority and fatigue-penalized OVR.
func sortKey(p *Player) (wkPriority int, adjustedOVR int) {
	fatigueDrop := 0
	if p.ConsecutiveStarts >= 3 {
		fatigueDrop = (p.ConsecutiveStarts - 2) * 3
	}

	wkPriority = 0
	if p.UniverseWonderkid && p.ConsecutiveStarts < 5 {
		wkPriority = 1
	}

	adjustedOVR = p.OVR - fatigueDrop
	return wkPriority, adjustedOVR
}

// sortPlayersForXI sorts players according to starting XI priority.
func sortPlayersForXI(players []*Player) {
	sort.SliceStable(players, func(i, j int) bool {
		wkI, ovrI := sortKey(players[i])
		wkJ, ovrJ := sortKey(players[j])

		if wkI != wkJ {
			return wkI > wkJ
		}
		if ovrI != ovrJ {
			return ovrI > ovrJ
		}
		return players[i].PlayerID < players[j].PlayerID
	})
}

// GetStartingEleven selects the best 11 players in a 4-3-3 formation
// (1 GK, 4 DEF, 3 MID, 3 FWD) with wonderkid priority and fatigue rotation.
func (c *Club) GetStartingEleven(fixture ...string) []*Player {
	pool := c.AvailableSquad(fixture...)

	var gks, defs, mids, fwds []*Player
	for _, p := range pool {
		switch p.Category {
		case "GK":
			gks = append(gks, p)
		case "DEF":
			defs = append(defs, p)
		case "MID":
			mids = append(mids, p)
		case "FWD":
			fwds = append(fwds, p)
		default:
			fwds = append(fwds, p)
		}
	}

	sortPlayersForXI(gks)
	sortPlayersForXI(defs)
	sortPlayersForXI(mids)
	sortPlayersForXI(fwds)

	var startingXI []*Player

	// 1 GK
	if len(gks) > 0 {
		startingXI = append(startingXI, gks[0])
	} else if len(pool) > 0 {
		startingXI = append(startingXI, pool[0])
	}

	// 4 DEF
	takeDEF := len(defs)
	if takeDEF > 4 {
		takeDEF = 4
	}
	startingXI = append(startingXI, defs[:takeDEF]...)

	// 3 MID
	takeMID := len(mids)
	if takeMID > 3 {
		takeMID = 3
	}
	startingXI = append(startingXI, mids[:takeMID]...)

	// 3 FWD
	takeFWD := len(fwds)
	if takeFWD > 3 {
		takeFWD = 3
	}
	startingXI = append(startingXI, fwds[:takeFWD]...)

	// Dedupe: the no-GK fallback (pool[0]) may coincide with a positional
	// pick in short squads. A starting XI must never name a player twice.
	seenXI := make(map[*Player]bool, len(startingXI))
	uniqueXI := make([]*Player, 0, len(startingXI))
	for _, p := range startingXI {
		if p == nil || seenXI[p] {
			continue
		}
		seenXI[p] = true
		uniqueXI = append(uniqueXI, p)
	}
	startingXI = uniqueXI

	// Fill to 11 if position counts are insufficient
	if len(startingXI) < 11 && len(pool) > len(startingXI) {
		used := make(map[string]bool)
		for _, p := range startingXI {
			used[p.PlayerID] = true
		}

		var remain []*Player
		for _, p := range pool {
			if !used[p.PlayerID] {
				remain = append(remain, p)
			}
		}

		sortPlayersForXI(remain)
		needed := 11 - len(startingXI)
		if needed > len(remain) {
			needed = len(remain)
		}
		startingXI = append(startingXI, remain[:needed]...)
	}

	// Assembly is capped at exactly 11 (1 GK + 4 DEF + 3 MID + 3 FWD) and the
	// fill above tops up to precisely 11, so no truncation is needed.
	return startingXI
}

// FixtureContext builds the competition:matchweek key used by XI/bench selection
// so school, exam weeks, and cup sit-outs are respected.
func FixtureContext(competition string, matchweek int) string {
	if competition == "" {
		competition = "super-league"
	}
	if matchweek < 1 {
		matchweek = 1
	}
	return competition + ":" + strconv.Itoa(matchweek)
}

// GetBench selects up to n substitutes from available non-starters.
func (c *Club) GetBench(starters []*Player, n int, fixture ...string) []*Player {
	if starters == nil {
		starters = c.GetStartingEleven(fixture...)
	}
	if n <= 0 {
		n = 7
	}

	starterIDs := make(map[string]bool)
	for _, p := range starters {
		if p != nil {
			starterIDs[p.PlayerID] = true
		}
	}

	var pool []*Player
	for _, p := range c.AvailableSquad(fixture...) {
		if p != nil && !starterIDs[p.PlayerID] {
			pool = append(pool, p)
		}
	}

	sort.SliceStable(pool, func(i, j int) bool {
		wkI := 0
		if pool[i].UniverseWonderkid {
			wkI = 1
		}
		wkJ := 0
		if pool[j].UniverseWonderkid {
			wkJ = 1
		}

		if wkI != wkJ {
			return wkI > wkJ
		}
		if pool[i].OVR != pool[j].OVR {
			return pool[i].OVR > pool[j].OVR
		}
		return pool[i].PlayerID < pool[j].PlayerID
	})

	if len(pool) > n {
		return pool[:n]
	}
	return pool
}

// RecalculateRatings updates SquadSize, SquadAvgOVR, and OverallTeamRating from the current squad.
func (c *Club) RecalculateRatings() {
	c.SquadSize = len(c.Squad)
	if c.SquadSize == 0 {
		c.SquadAvgOVR = 0
		c.OverallTeamRating = 0
		return
	}
	total := 0
	for _, p := range c.Squad {
		if p != nil {
			total += p.OVR
		}
	}
	avg := float64(total) / float64(c.SquadSize)
	c.SquadAvgOVR = math.Round(avg*10) / 10

	// Team rating based on the current starting XI.
	starters := c.GetStartingEleven()
	if len(starters) > 0 {
		xiTotal := 0
		for _, s := range starters {
			xiTotal += s.OVR
		}
		c.OverallTeamRating = int(math.Round(float64(xiTotal) / float64(len(starters))))
	} else {
		c.OverallTeamRating = 0
	}
}

// ToStandingsRow converts a Club's current season record to a StandingsRow.
func (c *Club) ToStandingsRow(position int) StandingsRow {
	formCopy := make([]string, len(c.Form))
	copy(formCopy, c.Form)

	return StandingsRow{
		Position:       position,
		ClubID:         c.ClubID,
		ClubName:       c.ClubName,
		ShortName:      c.ShortName,
		Played:         c.Played,
		Won:            c.Won,
		Drawn:          c.Drawn,
		Lost:           c.Lost,
		GoalsFor:       c.GoalsFor,
		GoalsAgainst:   c.GoalsAgainst,
		GoalDifference: c.GoalDifference,
		Points:         c.Points,
		TeamRating:     c.OverallTeamRating,
		Form:           formCopy,
	}
}
