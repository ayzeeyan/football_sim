package matchengine

import (
	"fmt"
	"math"
	"math/rand"
	"strings"

	"football_sim/pkg/managers"
	"football_sim/pkg/matchreport"
	"football_sim/pkg/models"
)

// Coordinate represents a 2D position on the pitch [0.0, 1.0].
type Coordinate struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// LivePlayerRadar represents one of the 22 on-pitch dots.
type LivePlayerRadar struct {
	PlayerID          string  `json:"player_id"`
	FullName          string  `json:"full_name"`
	Position          string  `json:"position"` // natural position
	NaturalPosition   string  `json:"natural_position"`
	TacticalSlot      string  `json:"tactical_slot"`
	PositionFit       string  `json:"position_fit"`
	Category          string  `json:"category"`
	OVR               int     `json:"ovr"`
	X                 float64 `json:"x"`
	Y                 float64 `json:"y"`
	IsWonderkid       bool    `json:"is_wonderkid"`
	UniverseWonderkid bool    `json:"universe_wonderkid"`
	Number            int     `json:"number"`
	BaseX             float64 `json:"-"`
	BaseY             float64 `json:"-"`
}

// CommentaryItem represents a timestamped event on the commentary ticker.
type CommentaryItem struct {
	Minute      int    `json:"minute"`
	Text        string `json:"text"`
	Category    string `json:"category"` // KICKOFF, CHANCE, SAVE, GOAL, FOUL, WONDERKID, FULLTIME, TACTIC
	IsWonderkid bool   `json:"is_wonderkid"`
	Timestamp   string `json:"timestamp"`
}

// PassTrailItem is the most recent ball movement segment the pitch canvas
// draws as a pass/shot line. From is the previous ball target, To is the new
// target, IsShot marks goal-bound efforts, Color carries the possessing side.
type PassTrailItem struct {
	From   [2]float64 `json:"from"`
	To     [2]float64 `json:"to"`
	IsShot bool       `json:"is_shot"`
	Color  [3]uint8   `json:"color"`
}

// PlannedSub is a scheduled live substitution (Python: planned_subs rows).
type PlannedSub struct {
	Minute int            `json:"minute"`
	Out    *models.Player `json:"-"`
	In     *models.Player `json:"-"`
	Side   string         `json:"side"`
	Done   bool           `json:"done"`
}

// TacticalShift records an AI manager game-state adaptation
// (Python: latest_tactical_shift dict).
type TacticalShift struct {
	Minute    int    `json:"minute"`
	Team      string `json:"team"`
	ClubShort string `json:"club_short"`
	Manager   string `json:"manager"`
	Stance    string `json:"stance"`
	Label     string `json:"label"`
	Text      string `json:"text"`
}

// LiveMatchEngine simulates the real-time match state machine.
type LiveMatchEngine struct {
	HomeClub        *models.Club
	AwayClub        *models.Club
	HomeManager     *managers.ManagerProfile
	AwayManager     *managers.ManagerProfile
	State           string  // NOT_STARTED, PLAYING, PAUSED, GOAL_PAUSE, FULL_TIME
	CurrentMinute   float64 `json:"current_minute"`
	halfTimeReached bool
	Speed           int `json:"speed"` // 1x, 2x, 5x, 999
	HomeScore       int `json:"home_score"`
	AwayScore       int `json:"away_score"`

	HomeShots   int     `json:"home_shots"`
	AwayShots   int     `json:"away_shots"`
	HomeShotsOn int     `json:"home_shots_on_target"`
	AwayShotsOn int     `json:"away_shots_on_target"`
	HomeXG      float64 `json:"home_xg"`
	AwayXG      float64 `json:"away_xg"`

	Phase          string `json:"phase"`           // BUILDUP, MIDFIELD, ATTACKING, SHOT, CELEBRATION
	PossessionTeam string `json:"possession_team"` // home, away
	ActiveThird    string `json:"active_third"`    // DEFENSIVE, MIDFIELD, ATTACKING
	PhaseTimer     float64

	BallPos    Coordinate `json:"ball_pos"`
	BallTarget Coordinate `json:"ball_target"`
	BallSpeed  float64    `json:"ball_speed"`
	// Live tick presentation state (F1): momentum in [-1,1] (home-positive),
	// most recent pass/shot segment, and shot elevation. Updated by the phase
	// machine in live_sim.go; surfaced by the WS tick builder.
	PossessionMomentum float64        `json:"possession_momentum"`
	PassTrail          *PassTrailItem `json:"pass_trail"`
	BallHeight         float64        `json:"ball_height"`
	BallIsShot         bool           `json:"ball_is_shot"`

	HomeStance          string         `json:"home_stance"` // NORMAL, OVERLOAD, PARK_BUS
	AwayStance          string         `json:"away_stance"`
	LatestTacticalShift *TacticalShift `json:"latest_tactical_shift,omitempty"`

	HomePlayers   []LivePlayerRadar `json:"home_players"`
	AwayPlayers   []LivePlayerRadar `json:"away_players"`
	HomeFormation string            `json:"home_formation"`
	AwayFormation string            `json:"away_formation"`

	Commentary []CommentaryItem             `json:"commentary"`
	Events     []matchreport.MatchEventItem `json:"events"`

	// Headless simulation state (Python: MatchEngine sim fields).
	HomeStarters        []*models.Player
	AwayStarters        []*models.Player
	HomeKickoffXI       []*models.Player
	AwayKickoffXI       []*models.Player
	HomeSlots           []models.StartingSlot
	AwaySlots           []models.StartingSlot
	HomeKickoffSlots    []models.StartingSlot
	AwayKickoffSlots    []models.StartingSlot
	HomeBench           []*models.Player
	AwayBench           []*models.Player
	PlannedSubs         []PlannedSub
	Bookings            map[string]int
	InstanceID          int
	HomeCorners         int `json:"home_corners"`
	AwayCorners         int `json:"away_corners"`
	HomePossessionTicks int
	AwayPossessionTicks int
	// Territory-weighted possession (F7): each tick scores by pitch third
	// (see thirdWeight). Seeded 1.0/1.0 with the tick counters; the weighted
	// share engages once the sim records territory (total above baseline).
	HomePossessionWeighted float64
	AwayPossessionWeighted float64
	SubstitutionsMade      map[string]int
	LiveShots              []matchreport.ShotMapItem
	LiveTouches            map[string][][2]float64
	Banner                 string
	BannerTimer            float64
	IsHighHeatDerby        bool
	IsRecognizedDerby      bool `json:"-"`
	Competition            string
	Matchweek              int
	Stage                  string
	Weather                string `json:"-"`

	RNG *rand.Rand
}

// positionHomeCoords maps a player position string to default (x, y) pitch
// coordinates for the home team (attacking left-to-right, 0.0–1.0).
var positionHomeCoords = map[string][2]float64{
	"GK":  {0.06, 0.50},
	"LB":  {0.20, 0.16},
	"LWB": {0.24, 0.14},
	"CB":  {0.18, 0.50},
	"RB":  {0.20, 0.84},
	"RWB": {0.24, 0.86},
	"CDM": {0.32, 0.50},
	"CM":  {0.40, 0.50},
	"CAM": {0.52, 0.50},
	"LM":  {0.40, 0.16},
	"RM":  {0.40, 0.84},
	"LW":  {0.58, 0.18},
	"RW":  {0.58, 0.82},
	"ST":  {0.62, 0.50},
	"CF":  {0.60, 0.50},
}

// positionAwayCoords maps position string to (x, y) for the away team
// (attacking right-to-left, mirrored from home).
var positionAwayCoords = map[string][2]float64{
	"GK":  {0.94, 0.50},
	"RB":  {0.80, 0.16},
	"RWB": {0.76, 0.14},
	"CB":  {0.82, 0.50},
	"LB":  {0.80, 0.84},
	"LWB": {0.76, 0.86},
	"CDM": {0.68, 0.50},
	"CM":  {0.60, 0.50},
	"CAM": {0.48, 0.50},
	"RM":  {0.60, 0.16},
	"LM":  {0.60, 0.84},
	"RW":  {0.42, 0.18},
	"LW":  {0.42, 0.82},
	"ST":  {0.38, 0.50},
	"CF":  {0.40, 0.50},
}

// staggerDuplicates spreads players sharing the same position evenly
// across the Y axis around their base coordinate.
func staggerDuplicates(players []LivePlayerRadar) {
	// Group indices by position
	groups := map[string][]int{}
	for i, p := range players {
		groups[p.Position] = append(groups[p.Position], i)
	}
	for _, indices := range groups {
		n := len(indices)
		if n <= 1 {
			continue
		}
		baseY := players[indices[0]].Y
		spread := 0.22 // total Y spread for duplicate positions
		if n > 3 {
			spread = 0.30
		}
		step := spread / float64(n-1)
		startY := baseY - spread/2
		for k, idx := range indices {
			newY := startY + step*float64(k)
			if newY < 0.08 {
				newY = 0.08
			}
			if newY > 0.92 {
				newY = 0.92
			}
			players[idx].Y = newY
		}
	}
}

// NewLiveMatchEngine creates and initializes a live match engine.
func NewLiveMatchEngine(
	home *models.Club,
	away *models.Club,
	homeMgr *managers.ManagerProfile,
	awayMgr *managers.ManagerProfile,
	seed int64,
) *LiveMatchEngine {
	if seed == 0 {
		seed = 20260907
	}
	engine := &LiveMatchEngine{
		HomeClub:          home,
		AwayClub:          away,
		HomeManager:       homeMgr,
		AwayManager:       awayMgr,
		State:             "NOT_STARTED",
		CurrentMinute:     0.0,
		Speed:             1,
		Phase:             "BUILDUP",
		PossessionTeam:    "home",
		ActiveThird:       "MIDFIELD",
		BallPos:           Coordinate{X: 0.5, Y: 0.5},
		BallTarget:        Coordinate{X: 0.5, Y: 0.5},
		HomeStance:        "NORMAL",
		AwayStance:        "NORMAL",
		Weather:           "clear",
		Bookings:          make(map[string]int),
		SubstitutionsMade: map[string]int{"home": 0, "away": 0},
		LiveTouches:       map[string][][2]float64{"home": {}, "away": {}},
		RNG:               rand.New(rand.NewSource(seed)),
	}

	engine.initPlayers()
	engine.AddCommentary(0, fmt.Sprintf("Welcome to %s! %s face %s in a high-stakes encounter.", home.HomeStadium, home.ClubName, away.ClubName), "KICKOFF", false)
	return engine
}

func managerBias(mgr *managers.ManagerProfile) (string, string) {
	if mgr == nil {
		return "", ""
	}
	return mgr.Style, mgr.Focus
}

func (e *LiveMatchEngine) initPlayers() {
	hs, hf := managerBias(e.HomeManager)
	as, af := managerBias(e.AwayManager)
	e.HomeFormation = models.FormationForStyle(hs)
	e.AwayFormation = models.FormationForStyle(as)
	e.HomeSlots = e.HomeClub.GetStartingElevenSlotsForFormation(e.HomeFormation, hs, hf)
	e.AwaySlots = e.AwayClub.GetStartingElevenSlotsForFormation(e.AwayFormation, as, af)
	e.initPlayersForSlots(e.HomeSlots, e.AwaySlots)
}

// initPlayersForXI is the legacy entry point for tests and older callers. It
// reconstructs explicit slots from formation + natural positions before any
// coordinates are assigned.
func (e *LiveMatchEngine) initPlayersForXI(homeXI, awayXI []*models.Player) {
	if e.HomeFormation == "" {
		hs, _ := managerBias(e.HomeManager)
		e.HomeFormation = models.FormationForStyle(hs)
	}
	if e.AwayFormation == "" {
		as, _ := managerBias(e.AwayManager)
		e.AwayFormation = models.FormationForStyle(as)
	}
	e.HomeSlots = models.AssignPlayersToFormation(homeXI, e.HomeFormation)
	e.AwaySlots = models.AssignPlayersToFormation(awayXI, e.AwayFormation)
	e.initPlayersForSlots(e.HomeSlots, e.AwaySlots)
}

func (e *LiveMatchEngine) initPlayersForSlots(homeSlots, awaySlots []models.StartingSlot) {
	e.HomePlayers = radarPlayersSlotted(homeSlots, positionHomeCoords)
	e.AwayPlayers = radarPlayersSlotted(awaySlots, positionAwayCoords)
}

// syncRadarActor keeps the on-pitch actor identity aligned with a live
// substitution while preserving that slot's current coordinates.
func (e *LiveMatchEngine) syncRadarActor(side, oldID string, replacement *models.Player) {
	if replacement == nil || oldID == "" {
		return
	}
	actors := &e.HomePlayers
	slots := &e.HomeSlots
	if side != "home" {
		actors = &e.AwayPlayers
		slots = &e.AwaySlots
	}
	for i := range *slots {
		assignment := &(*slots)[i]
		if assignment.Player != nil && assignment.PlayerID == oldID {
			assignment.Player = replacement
			assignment.NaturalPosition = strings.ToUpper(strings.TrimSpace(replacement.Position))
			assignment.PositionFit = models.PositionFitForPlayer(replacement, assignment.Slot)
			break
		}
	}
	for i := range *actors {
		actor := &(*actors)[i]
		if actor.PlayerID != oldID {
			continue
		}
		// A substitution replaces the actor in the existing tactical role;
		// it must never teleport to the substitute's natural-position lane.
		x, y := actor.X, actor.Y
		number := actor.Number
		*actor = LivePlayerRadar{
			PlayerID: replacement.PlayerID, FullName: replacement.FullName,
			Position:        replacement.Position,
			NaturalPosition: strings.ToUpper(strings.TrimSpace(replacement.Position)),
			TacticalSlot:    actor.TacticalSlot,
			PositionFit:     string(models.PositionFitForPlayer(replacement, actor.TacticalSlot)),
			Category:        replacement.Category,
			OVR:             replacement.OVR, X: x, Y: y,
			IsWonderkid:       replacement.UniverseWonderkid,
			UniverseWonderkid: replacement.UniverseWonderkid, Number: number,
			BaseX: x,
			BaseY: y,
		}
		return
	}

	// Fallback: rebuild side from scratch
	if side == "home" {
		if len(e.HomeSlots) == 0 {
			e.HomeSlots = models.AssignPlayersToFormation(e.HomeStarters, e.HomeFormation)
		}
		e.HomePlayers = radarPlayersSlotted(e.HomeSlots, positionHomeCoords)
	} else {
		if len(e.AwaySlots) == 0 {
			e.AwaySlots = models.AssignPlayersToFormation(e.AwayStarters, e.AwayFormation)
		}
		e.AwayPlayers = radarPlayersSlotted(e.AwaySlots, positionAwayCoords)
	}
}

// radarPlayersSlotted derives all base coordinates from the assigned tactical
// slot. Natural Position remains display/identity metadata only.
func radarPlayersSlotted(assignments []models.StartingSlot, coordMap map[string][2]float64) []LivePlayerRadar {
	players := make([]LivePlayerRadar, 0, len(assignments))
	for _, assignment := range assignments {
		player := assignment.Player
		if player == nil {
			continue
		}
		slot := strings.ToUpper(strings.TrimSpace(assignment.Slot))
		coord, ok := coordMap[slot]
		if !ok {
			coord = coordMap["CM"]
		}
		players = append(players, LivePlayerRadar{
			PlayerID: player.PlayerID, FullName: player.FullName,
			Position:        player.Position,
			NaturalPosition: strings.ToUpper(strings.TrimSpace(player.Position)),
			TacticalSlot:    slot,
			PositionFit:     string(models.PositionFitForPlayer(player, slot)),
			Category:        player.Category, OVR: player.OVR,
			X: coord[0], Y: coord[1], BaseX: coord[0], BaseY: coord[1],
			IsWonderkid: player.UniverseWonderkid, UniverseWonderkid: player.UniverseWonderkid,
			Number: len(players) + 1,
		})
	}
	return players
}

// radarBaseCoordinate recovers a missing base coordinate from role metadata,
// never from the actor's array index. TacticalSlot is authoritative; natural
// position and category are legacy-only fallbacks.
func radarBaseCoordinate(player LivePlayerRadar, coordMap map[string][2]float64) [2]float64 {
	for _, position := range []string{player.TacticalSlot, player.NaturalPosition, player.Position} {
		if coord, ok := coordMap[strings.ToUpper(strings.TrimSpace(position))]; ok {
			return coord
		}
	}
	switch player.Category {
	case "GK":
		return coordMap["GK"]
	case "DEF":
		return coordMap["CB"]
	case "MID":
		return coordMap["CM"]
	default:
		return coordMap["ST"]
	}
}

// radarPlayersPositional assigns pitch coordinates based on each player's
// natural position string, then staggers any duplicates.
func radarPlayersPositional(xi []*models.Player, coordMap map[string][2]float64) []LivePlayerRadar {
	players := make([]LivePlayerRadar, 0, len(xi))
	for _, p := range xi {
		if p == nil {
			continue
		}
		pos := strings.ToUpper(strings.TrimSpace(p.Position))
		coord, ok := coordMap[pos]
		if !ok {
			// Fallback: use category-based default
			switch p.Category {
			case "GK":
				coord = coordMap["GK"]
			case "DEF":
				coord = coordMap["CB"]
			case "MID":
				coord = coordMap["CM"]
			default:
				coord = coordMap["ST"]
			}
		}
		players = append(players, LivePlayerRadar{
			PlayerID:          p.PlayerID,
			FullName:          p.FullName,
			Position:          p.Position,
			NaturalPosition:   pos,
			TacticalSlot:      pos,
			PositionFit:       string(models.PositionFitNatural),
			Category:          p.Category,
			OVR:               p.OVR,
			X:                 coord[0],
			Y:                 coord[1],
			IsWonderkid:       p.UniverseWonderkid,
			UniverseWonderkid: p.UniverseWonderkid,
			Number:            len(players) + 1,
			BaseX:             coord[0],
			BaseY:             coord[1],
		})
	}
	staggerDuplicates(players)
	for i := range players {
		players[i].BaseX = players[i].X
		players[i].BaseY = players[i].Y
	}
	return players
}

// AddCommentary records a commentary item.
func (e *LiveMatchEngine) AddCommentary(minute int, text string, category string, isWK bool) {
	e.Commentary = append(e.Commentary, CommentaryItem{
		Minute:      minute,
		Text:        text,
		Category:    category,
		IsWonderkid: isWK,
		Timestamp:   fmt.Sprintf("%d'", minute),
	})
}

// CheckTacticalAdaptations checks and triggers in-match stance shifts (OVERLOAD / PARK_BUS).
func (e *LiveMatchEngine) CheckTacticalAdaptations(minute int) {
	if minute >= 75 && minute <= 88 {
		// Home trailing by 1
		if e.HomeScore < e.AwayScore && e.HomeStance == "NORMAL" {
			e.HomeStance = "OVERLOAD"
			e.LatestTacticalShift = &TacticalShift{Minute: minute, Team: "home", ClubShort: e.HomeClub.ShortName, Stance: "OVERLOAD", Text: fmt.Sprintf("%s shifts to an all-out 3-striker OVERLOAD!", e.HomeClub.ShortName)}
			e.AddCommentary(minute, e.LatestTacticalShift.Text, "TACTIC", false)
		}
		// Away trailing by 1
		if e.AwayScore < e.HomeScore && e.AwayStance == "NORMAL" {
			e.AwayStance = "OVERLOAD"
			e.LatestTacticalShift = &TacticalShift{Minute: minute, Team: "away", ClubShort: e.AwayClub.ShortName, Stance: "OVERLOAD", Text: fmt.Sprintf("%s commits numbers forward into a high-risk OVERLOAD!", e.AwayClub.ShortName)}
			e.AddCommentary(minute, e.LatestTacticalShift.Text, "TACTIC", false)
		}
	}

	if minute >= 78 {
		// Home protecting a 1-goal lead
		if e.HomeScore == e.AwayScore+1 && e.HomeStance == "NORMAL" {
			e.HomeStance = "PARK_BUS"
			e.LatestTacticalShift = &TacticalShift{Minute: minute, Team: "home", ClubShort: e.HomeClub.ShortName, Stance: "PARK_BUS", Text: fmt.Sprintf("%s adopts a deep 5-man low-block to park the bus!", e.HomeClub.ShortName)}
			e.AddCommentary(minute, e.LatestTacticalShift.Text, "TACTIC", false)
		}
		// Away protecting a 1-goal lead
		if e.AwayScore == e.HomeScore+1 && e.AwayStance == "NORMAL" {
			e.AwayStance = "PARK_BUS"
			e.LatestTacticalShift = &TacticalShift{Minute: minute, Team: "away", ClubShort: e.AwayClub.ShortName, Stance: "PARK_BUS", Text: fmt.Sprintf("%s drops into a compact low block to protect the lead!", e.AwayClub.ShortName)}
			e.AddCommentary(minute, e.LatestTacticalShift.Text, "TACTIC", false)
		}
	}
}

// Tick advances the match state by dt seconds. The real phase machine lives
// in Update; Tick then overlays radar wobble so the pitch stays alive.
func (e *LiveMatchEngine) Tick(dt float64) {
	e.Update(dt)
	if e.State != "PLAYING" && e.State != "GOAL_PAUSE" {
		return
	}

	intMin := int(e.CurrentMinute)
	e.CheckTacticalAdaptations(intMin)

	// Update subtle radar wobble (ball position is owned by Update/AdvancePhase).
	// Sent-off players have left the shape: freeze them so they do not keep
	// jogging with OVERLOAD / PARK_BUS.
	for i := range e.HomePlayers {
		if e.Bookings[e.HomePlayers[i].PlayerID] >= 2 {
			continue
		}
		baseX := e.HomePlayers[i].BaseX
		baseY := e.HomePlayers[i].BaseY
		if baseX == 0 && baseY == 0 {
			base := radarBaseCoordinate(e.HomePlayers[i], positionHomeCoords)
			baseX, baseY = base[0], base[1]
		}
		offset := 0.0
		if e.HomeStance == "OVERLOAD" && e.HomePlayers[i].Category != "GK" {
			offset = 0.08 // push forward
		} else if e.HomeStance == "PARK_BUS" && e.HomePlayers[i].Category != "GK" {
			offset = -0.06 // drop back
		}
		wobbleX := math.Sin(e.CurrentMinute*2.0+float64(i)) * 0.015
		wobbleY := math.Cos(e.CurrentMinute*1.8+float64(i)) * 0.015
		e.HomePlayers[i].X = math.Max(0.04, math.Min(0.96, baseX+offset+wobbleX))
		e.HomePlayers[i].Y = math.Max(0.06, math.Min(0.94, baseY+wobbleY))
	}

	for i := range e.AwayPlayers {
		if e.Bookings[e.AwayPlayers[i].PlayerID] >= 2 {
			continue
		}
		baseX := e.AwayPlayers[i].BaseX
		baseY := e.AwayPlayers[i].BaseY
		if baseX == 0 && baseY == 0 {
			base := radarBaseCoordinate(e.AwayPlayers[i], positionAwayCoords)
			baseX, baseY = base[0], base[1]
		}
		offset := 0.0
		if e.AwayStance == "OVERLOAD" && e.AwayPlayers[i].Category != "GK" {
			offset = -0.08 // push forward (attacks towards left)
		} else if e.AwayStance == "PARK_BUS" && e.AwayPlayers[i].Category != "GK" {
			offset = 0.06 // drop back
		}
		wobbleX := math.Sin(e.CurrentMinute*2.0+float64(i)) * 0.015
		wobbleY := math.Cos(e.CurrentMinute*1.8+float64(i)) * 0.015
		e.AwayPlayers[i].X = math.Max(0.04, math.Min(0.96, baseX+offset+wobbleX))
		e.AwayPlayers[i].Y = math.Max(0.06, math.Min(0.94, baseY+wobbleY))
	}

	// Idle possession drift so the radar doesn't freeze between phases.
	if e.PossessionTeam == "home" {
		targetX := 0.50 + math.Sin(e.CurrentMinute*0.5)*0.25
		targetY := 0.50 + math.Cos(e.CurrentMinute*0.7)*0.25
		e.BallPos.X = math.Round(targetX*100) / 100
		e.BallPos.Y = math.Round(targetY*100) / 100
	} else {
		targetX := 0.50 - math.Sin(e.CurrentMinute*0.5)*0.25
		targetY := 0.50 - math.Cos(e.CurrentMinute*0.7)*0.25
		e.BallPos.X = math.Round(targetX*100) / 100
		e.BallPos.Y = math.Round(targetY*100) / 100
	}
}

// Kickoff starts or resumes the live match (server control path: also
// restarts after full time).
func (e *LiveMatchEngine) Kickoff() {
	if e.State == "HALF_TIME" {
		return
	}
	if e.State == "FULL_TIME" {
		e.Reset()
	}
	e.State = "PLAYING"
}

// StartKickoff blows the opening whistle (Python: start_kickoff): only
// NOT_STARTED/PAUSED matches start playing, with an underway call at 0'.
func (e *LiveMatchEngine) StartKickoff() {
	if e.State != "NOT_STARTED" && e.State != "PAUSED" {
		return
	}
	e.State = "PLAYING"
	if e.CurrentMinute == 0 {
		e.AddCommentary(0, "The referee blows the whistle and we are underway!", "KICKOFF", false)
	}
}

// TogglePause flips between PLAYING and PAUSED.
func (e *LiveMatchEngine) TogglePause() {
	if e.State == "PLAYING" {
		e.State = "PAUSED"
	} else if e.State == "PAUSED" {
		e.State = "PLAYING"
	}
}

// Reset restores initial pre-match state (server control path).
func (e *LiveMatchEngine) Reset() {
	e.ResetMatch()
}

// ResetMatch performs the full Python reset_match: scores, counters,
// phase machine, commentary, events, bookings, stances, subs, and roster
// snapshots are all rebuilt.
func (e *LiveMatchEngine) ResetMatch() {
	e.State = "NOT_STARTED"
	e.CurrentMinute = 0.0
	e.halfTimeReached = false
	e.HomeScore = 0
	e.AwayScore = 0
	e.HomeShots = 0
	e.AwayShots = 0
	e.HomeShotsOn = 0
	e.AwayShotsOn = 0
	e.HomeCorners = 0
	e.AwayCorners = 0
	e.HomePossessionTicks = 1
	e.AwayPossessionTicks = 1
	e.HomePossessionWeighted = 1.0
	e.AwayPossessionWeighted = 1.0
	e.Phase = "BUILDUP"
	e.PossessionTeam = "home"
	e.ActiveThird = "MIDFIELD"
	e.PhaseTimer = 0.0
	e.BallPos = Coordinate{X: 0.5, Y: 0.5}
	e.BallTarget = Coordinate{X: 0.5, Y: 0.5}
	e.PossessionMomentum = 0.0
	e.PassTrail = nil
	e.BallHeight = 0.0
	e.BallIsShot = false
	e.Banner = ""
	e.BannerTimer = 0.0
	e.HomeStance = "NORMAL"
	e.AwayStance = "NORMAL"
	e.LatestTacticalShift = nil
	e.Commentary = make([]CommentaryItem, 0)
	e.Events = make([]matchreport.MatchEventItem, 0)
	if e.Bookings == nil {
		e.Bookings = make(map[string]int)
	} else {
		for k := range e.Bookings {
			delete(e.Bookings, k)
		}
	}
	e.InstanceID++
	e.LiveShots = nil
	if e.LiveTouches == nil {
		e.LiveTouches = map[string][][2]float64{"home": {}, "away": {}}
	} else {
		e.LiveTouches["home"] = e.LiveTouches["home"][:0]
		e.LiveTouches["away"] = e.LiveTouches["away"][:0]
	}
	if e.SubstitutionsMade == nil {
		e.SubstitutionsMade = map[string]int{"home": 0, "away": 0}
	} else {
		e.SubstitutionsMade["home"] = 0
		e.SubstitutionsMade["away"] = 0
	}

	if e.HomeClub == nil || e.AwayClub == nil {
		return
	}
	fx := models.FixtureContext(e.Competition, e.Matchweek)
	hs, hf := managerBias(e.HomeManager)
	as, af := managerBias(e.AwayManager)
	e.HomeFormation = models.FormationForStyle(hs)
	e.AwayFormation = models.FormationForStyle(as)
	e.HomeSlots = e.HomeClub.GetStartingElevenSlotsForFormation(e.HomeFormation, hs, hf, fx)
	e.AwaySlots = e.AwayClub.GetStartingElevenSlotsForFormation(e.AwayFormation, as, af, fx)
	e.HomeStarters = models.PlayersFromStartingSlots(e.HomeSlots)
	e.AwayStarters = models.PlayersFromStartingSlots(e.AwaySlots)
	e.HomeKickoffXI = append([]*models.Player(nil), e.HomeStarters...)
	e.AwayKickoffXI = append([]*models.Player(nil), e.AwayStarters...)
	e.HomeKickoffSlots = append([]models.StartingSlot(nil), e.HomeSlots...)
	e.AwayKickoffSlots = append([]models.StartingSlot(nil), e.AwaySlots...)
	e.HomeBench = e.HomeClub.GetBench(e.HomeKickoffXI, 7, fx)
	e.AwayBench = e.AwayClub.GetBench(e.AwayKickoffXI, 7, fx)
	e.PlannedSubs = nil
	for _, sub := range matchreport.PlanSubstitutions(e.HomeKickoffXI, e.HomeBench, 5, e.RNG) {
		e.PlannedSubs = append(e.PlannedSubs, PlannedSub{Minute: sub.Minute, Out: sub.Out, In: sub.In, Side: "home"})
	}
	for _, sub := range matchreport.PlanSubstitutions(e.AwayKickoffXI, e.AwayBench, 5, e.RNG) {
		e.PlannedSubs = append(e.PlannedSubs, PlannedSub{Minute: sub.Minute, Out: sub.Out, In: sub.In, Side: "away"})
	}
	e.initPlayersForSlots(e.HomeSlots, e.AwaySlots)
	e.AddCommentary(0, fmt.Sprintf("Welcome to %s! %s take on %s.", e.HomeClub.HomeStadium, e.HomeClub.ClubName, e.AwayClub.ClubName), "KICKOFF", false)
}

// SetSpeed updates simulation speed (1x, 2x, 5x, 10x, etc.).
func (e *LiveMatchEngine) SetSpeed(speed int) {
	if speed <= 0 {
		speed = 1
	}
	e.Speed = speed
}

// SetClubs configures the active matchup and re-initializes rosters.
func (e *LiveMatchEngine) SetClubs(home, away *models.Club, homeMgr, awayMgr *managers.ManagerProfile) {
	e.HomeClub = home
	e.AwayClub = away
	e.HomeManager = homeMgr
	e.AwayManager = awayMgr
	e.ClearFixtureContext()
	e.Reset()
}

// SetFixtureContext stamps competition + matchweek so school/exam sit-outs apply.
// The optional flags carry recognized rivalry and existing high-heat climate
// context independently. Omitting either flag clears its prior value.
func (e *LiveMatchEngine) SetFixtureContext(competition string, matchweek int, context ...bool) {
	e.Competition = competition
	e.Matchweek = matchweek
	e.IsRecognizedDerby = len(context) > 0 && context[0]
	e.IsHighHeatDerby = len(context) > 1 && context[1]
	e.Weather = "clear"
	e.ResetMatch()
}

// ClearFixtureContext removes competition and recognized-derby context while
// leaving the current match state intact for callers that reset separately.
func (e *LiveMatchEngine) ClearFixtureContext() {
	e.Competition = ""
	e.Matchweek = 0
	e.Stage = ""
	e.IsRecognizedDerby = false
	e.IsHighHeatDerby = false
	e.Weather = "clear"
}

// SetFixtureWeather applies the selected fixture's weather to the live
// simulation context. Empty weather uses the canonical dry-weather default.
func (e *LiveMatchEngine) SetFixtureWeather(weather string) {
	if weather == "" {
		weather = "clear"
	}
	e.Weather = weather
}

// SetFixtureStage stamps the cup stage (Play-in, Final, ...) so stage-gated
// personality lines can fire. Empty clears it.
func (e *LiveMatchEngine) SetFixtureStage(stage string) {
	e.Stage = stage
}

func bigGameContext(competition string, recognizedDerby bool) bool {
	return recognizedDerby || strings.EqualFold(strings.TrimSpace(competition), "ucl")
}
