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
	BoardObjective    string       `json:"board_objective,omitempty"`
	ExpectedFinish    int          `json:"expected_finish,omitempty"`
	// Coefficient is the UEFA-style points total earned from European
	// campaigns (league-phase results plus knockout progress). It seeds
	// Swiss pots and playoff prestige; 0 on fresh worlds (rating order
	// applies until points are earned).
	Coefficient int `json:"coefficient,omitempty"`

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
	var identityEnvelope struct {
		Identity *ClubIdentity `json:"identity"`
	}
	if err := json.Unmarshal(data, &identityEnvelope); err != nil {
		return err
	}

	if c.Morale == 0 {
		c.Morale = 70
	}
	if c.Morale < 0 {
		c.Morale = 0
	} else if c.Morale > 100 {
		c.Morale = 100
	}

	// A missing or null identity block means legacy/static-data input. A present
	// object remains authoritative even when all nine traits are zero.
	identityMissing := identityEnvelope.Identity == nil
	if identityMissing {
		c.Identity = DefaultClubIdentity(c.ClubID, c.OverallTeamRating)
	} else {
		c.Identity.present = true
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
	if c.Finances.TransferBudget > c.Finances.Balance {
		c.Finances.TransferBudget = c.Finances.Balance
	}
	if c.Finances.WageCap < 0 {
		c.Finances.WageCap = 0
	}
	var bill int64
	for _, p := range c.Squad {
		if p != nil {
			bill += p.WageEUR * 52
		}
	}
	if c.Finances.EuropeanRevenue < 0 {
		c.Finances.EuropeanRevenue = 0
	}
	if c.Coefficient < 0 {
		c.Coefficient = 0
	}
	formula := WageCapForIdentity(c.Identity)
	if identityMissing {
		// Static dataset input: establish a cap that covers the real squad.
		c.Finances.WageCap = formula
		if c.Finances.WageCap < bill {
			headroom := bill / 10
			if headroom > 10_000_000 {
				headroom = 10_000_000
			}
			c.Finances.WageCap = bill + headroom
		}
	} else if c.Finances.WageCap <= 0 {
		c.Finances.WageCap = formula
		if c.Finances.WageCap < bill {
			headroom := bill / 10
			if headroom > 10_000_000 {
				headroom = 10_000_000
			}
			c.Finances.WageCap = bill + headroom
		}
	} else if c.Finances.WageCap < formula {
		// Pre-headroom save (running code guarantees cap >= formula): raise
		// to the structural floor, covering the committed bill. Caps at or
		// above formula are never touched here, so legal transients where
		// academy intake or loan returns pushed the bill over the cap do not
		// ratchet — the wage gate handles those.
		c.Finances.WageCap = formula
		if c.Finances.WageCap < bill {
			headroom := bill / 10
			if headroom > 10_000_000 {
				headroom = 10_000_000
			}
			c.Finances.WageCap = bill + headroom
		}
	}
	c.Finances.WageBudget = bill

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

// sortKey computes wonderkid priority and a selection OVR that respects
// fatigue, fitness, sharpness, morale, form, role, and match importance.
func sortKey(p *Player, competition string, matchweek int) (wkPriority int, adjustedOVR int) {
	fatigueDrop := 0
	if p.ConsecutiveStarts >= 3 {
		fatigueDrop = (p.ConsecutiveStarts - 2) * 3
	}
	importance := CompetitionImportance(competition, matchweek)
	if importance <= 50 && p.ConsecutiveStarts >= 2 {
		fatigueDrop += 4
	}
	if importance <= 50 && p.Fitness > 0 && p.Fitness < 65 {
		fatigueDrop += 5
	}
	if importance >= 80 && (p.SquadRole == RoleRotation || p.SquadRole == RoleSquad || p.SquadRole == RoleProspect) {
		fatigueDrop += 3
	}

	wkPriority = 0
	if p.UniverseWonderkid && p.ConsecutiveStarts < 5 {
		wkPriority = 1
	}

	adjustedOVR = p.OVR - fatigueDrop + p.FormModifier()
	if p.Fitness > 0 {
		adjustedOVR += (p.Fitness - 70) / 8
	}
	if p.Sharpness > 0 {
		adjustedOVR += (p.Sharpness - 65) / 10
	}
	if p.Morale > 0 {
		adjustedOVR += (p.Morale - 70) / 15
	}
	if p.SquadRole == RoleCrucial && importance >= 70 {
		adjustedOVR += 2
	}
	return wkPriority, adjustedOVR
}

func applyManagerBias(p *Player, style, focus string) int {
	if p == nil {
		return 0
	}
	bias := 0
	switch strings.ToLower(strings.TrimSpace(focus)) {
	case "youth":
		if p.Age <= 21 {
			bias += 3
		}
	case "stars":
		if p.OVR >= 84 {
			bias += 2
		}
	}
	switch strings.ToLower(strings.TrimSpace(style)) {
	case "high_press", "press":
		if p.Fitness >= 75 {
			bias += 2
		}
		if p.Age >= 32 {
			bias -= 2
		}
	case "possession":
		if p.Category == "MID" {
			bias += 2
		}
	case "low_block", "counter":
		if p.Age >= 28 && p.Category == "DEF" {
			bias += 2
		}
	case "free_flowing":
		if p.UniverseWonderkid {
			bias += 2
		}
	}
	return bias
}

// sortPlayersForXI sorts players according to starting XI priority.
func sortPlayersForXI(players []*Player, competition string, matchweek int, style, focus string) {
	sort.SliceStable(players, func(i, j int) bool {
		wkI, ovrI := sortKey(players[i], competition, matchweek)
		wkJ, ovrJ := sortKey(players[j], competition, matchweek)
		ovrI += applyManagerBias(players[i], style, focus)
		ovrJ += applyManagerBias(players[j], style, focus)

		if wkI != wkJ {
			return wkI > wkJ
		}
		if ovrI != ovrJ {
			return ovrI > ovrJ
		}
		return players[i].PlayerID < players[j].PlayerID
	})
}

// StartingSlot is one non-overlapping tactical place in the default 4-3-3.
// Keeping the slot beside the player lets API consumers render an XI without
// guessing which of two centre-backs or full-backs belongs on each side.
type StartingSlot struct {
	Slot string `json:"slot"`
	*Player
}

// GetStartingEleven selects the best 11 players in a 4-3-3 formation
// (1 GK, 4 DEF, 3 MID, 3 FWD) with wonderkid priority and fatigue rotation.
func (c *Club) GetStartingEleven(fixture ...string) []*Player {
	slots := c.GetStartingElevenSlots(fixture...)
	startingXI := make([]*Player, 0, len(slots))
	for _, slot := range slots {
		if slot.Player != nil {
			startingXI = append(startingXI, slot.Player)
		}
	}
	return startingXI
}

// GetStartingElevenWithBias applies a manager's style and recruitment focus
// on top of fitness, form, and match importance.
func (c *Club) GetStartingElevenWithBias(style, focus string, fixture ...string) []*Player {
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

	comp, week := parseFixtureArgs(fixture)
	sortPlayersForXI(gks, comp, week, style, focus)
	sortPlayersForXI(defs, comp, week, style, focus)
	sortPlayersForXI(mids, comp, week, style, focus)
	sortPlayersForXI(fwds, comp, week, style, focus)

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

		sortPlayersForXI(remain, comp, week, style, focus)
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

// GetStartingElevenSlots returns the selected XI in stable 4-3-3 tactical
// slots. Natural positions are preferred before a category-compatible
// fallback, and each player can only be assigned once.
func (c *Club) GetStartingElevenSlots(fixture ...string) []StartingSlot {
	return c.GetStartingElevenSlotsWithBias("", "", fixture...)
}

// GetStartingElevenSlotsWithBias is the manager-aware equivalent used by
// squad and fixture previews. It retains the same selection priorities as
// GetStartingElevenWithBias while exposing a renderer-safe tactical slot.
func (c *Club) GetStartingElevenSlotsWithBias(style, focus string, fixture ...string) []StartingSlot {
	pool := c.AvailableSquad(fixture...)
	if len(pool) == 0 {
		return nil
	}
	// Preserve the long-standing no-keeper safety rule: malformed short
	// squads open with their first available player rather than silently
	// changing that fallback because the tactical ranking is sorted below.
	firstAvailable := pool[0]
	comp, week := parseFixtureArgs(fixture)
	sortPlayersForXI(pool, comp, week, style, focus)

	type slotRule struct {
		slot      string
		category  string
		positions map[string]bool
	}
	rules := []slotRule{
		{"GK", "GK", map[string]bool{"GK": true}},
		{"LB", "DEF", map[string]bool{"LB": true, "LWB": true}},
		{"LCB", "DEF", map[string]bool{"CB": true}},
		{"RCB", "DEF", map[string]bool{"CB": true}},
		{"RB", "DEF", map[string]bool{"RB": true, "RWB": true}},
		{"LCM", "MID", map[string]bool{"CM": true, "CAM": true, "CDM": true}},
		{"CM", "MID", map[string]bool{"CDM": true, "CM": true, "CAM": true}},
		{"RCM", "MID", map[string]bool{"CM": true, "CAM": true, "CDM": true}},
		{"LW", "FWD", map[string]bool{"LW": true}},
		{"ST", "FWD", map[string]bool{"ST": true, "CF": true}},
		{"RW", "FWD", map[string]bool{"RW": true}},
	}

	used := make(map[string]bool, len(pool))
	pick := func(rule slotRule, naturalOnly bool) *Player {
		for _, p := range pool {
			if p == nil || used[p.PlayerID] {
				continue
			}
			if naturalOnly {
				if rule.positions[strings.ToUpper(strings.TrimSpace(p.Position))] {
					return p
				}
				continue
			}
			if p.Category == rule.category {
				return p
			}
		}
		return nil
	}

	slots := make([]StartingSlot, 0, 11)
	for _, rule := range rules {
		p := pick(rule, true)
		if p == nil {
			p = pick(rule, false)
		}
		if p == nil && rule.slot == "GK" && firstAvailable != nil && !used[firstAvailable.PlayerID] {
			p = firstAvailable
		}
		if p == nil {
			// A short or malformed squad can be missing an entire category.
			// Still give every selected player one unique visible slot.
			for _, fallback := range pool {
				if fallback != nil && !used[fallback.PlayerID] {
					p = fallback
					break
				}
			}
		}
		if p == nil {
			continue
		}
		used[p.PlayerID] = true
		slots = append(slots, StartingSlot{Slot: rule.slot, Player: p})
	}
	return slots
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
		c.Finances.WageBudget = 0
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
	c.RecalculateWageBill()
}

// RecalculateWageBill stores the annual committed wage bill from current contracts.
// WageBudget keeps the bill (legacy name); WageCap is the real constraint
// derived from financial power. The cap never sits below committed wages:
// fresh squads get headroom, and future signings must fit under it.
func (c *Club) RecalculateWageBill() {
	if c == nil {
		return
	}
	var annual int64
	for _, p := range c.Squad {
		if p != nil {
			annual += p.WageEUR * 52
		}
	}
	c.Finances.WageBudget = annual
	formula := WageCapForIdentity(c.Identity)
	// The cap is structural (financial power), not a shadow of the bill:
	// it is fixed at max(formula, first-seen bill + headroom) and never
	// ratchets upward just because the bill grew. Future signings must fit
	// under it via CanAffordWage. Identity upgrades can still raise it.
	if c.Finances.WageCap <= 0 {
		cap := formula
		if cap < annual {
			headroom := annual / 10
			if headroom > 10_000_000 {
				headroom = 10_000_000
			}
			cap = annual + headroom
		}
		c.Finances.WageCap = cap
		return
	}
	if formula > c.Finances.WageCap {
		c.Finances.WageCap = formula
	}
}

// WageBill returns the annual committed wages for the current squad.
func (c *Club) WageBill() int64 {
	if c == nil {
		return 0
	}
	var annual int64
	for _, p := range c.Squad {
		if p != nil {
			annual += p.WageEUR * 52
		}
	}
	return annual
}

// WageCap returns the club's annual wage cap (financial-power derived).
func (c *Club) WageCap() int64 {
	if c == nil {
		return 0
	}
	if c.Finances.WageCap > 0 {
		return c.Finances.WageCap
	}
	return WageCapForIdentity(c.Identity)
}

// CanAffordWage reports whether adding annualWage keeps the bill under the cap.
// Valuation clamps are enforced separately; this never mutates market values.
func (c *Club) CanAffordWage(annualWage int64) bool {
	if c == nil || annualWage < 0 {
		return false
	}
	return c.WageBill()+annualWage <= c.WageCap()
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
