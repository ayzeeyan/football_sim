package models

import (
	"sort"
	"strings"
)

const (
	Formation433       = "4-3-3"
	Formation433Attack = "4-3-3 Attack"
	Formation4231      = "4-2-3-1"
	Formation442       = "4-4-2"
)

// PositionFit describes how naturally a player covers one match-specific
// tactical slot. Position remains the player's natural position; it is never
// overwritten by a match assignment.
type PositionFit string

const (
	PositionFitNatural    PositionFit = "Natural"
	PositionFitGood       PositionFit = "Good"
	PositionFitAcceptable PositionFit = "Acceptable"
	PositionFitEmergency  PositionFit = "Emergency"
)

// FormationDefinition owns the eleven unique slots used by a shape. Callers
// receive copies through FormationSlots so this canonical table cannot be
// mutated by a renderer or simulation path.
type FormationDefinition struct {
	Name  string
	Slots []string
}

var formationDefinitions = map[string]FormationDefinition{
	Formation433: {
		Name:  Formation433,
		Slots: []string{"GK", "LB", "LCB", "RCB", "RB", "LCM", "CM", "RCM", "LW", "ST", "RW"},
	},
	Formation433Attack: {
		Name:  Formation433Attack,
		Slots: []string{"GK", "LB", "LCB", "RCB", "RB", "LCM", "RCM", "CAM", "LW", "ST", "RW"},
	},
	Formation4231: {
		Name:  Formation4231,
		Slots: []string{"GK", "LB", "LCB", "RCB", "RB", "LDM", "RDM", "LW", "CAM", "RW", "ST"},
	},
	Formation442: {
		Name:  Formation442,
		Slots: []string{"GK", "LB", "LCB", "RCB", "RB", "LM", "LCM", "RCM", "RM", "LST", "RST"},
	},
}

// NormalizeFormation accepts persisted/display aliases and returns one of the
// canonical formations. Blank and unknown legacy values use the historical
// 4-3-3 shape.
func NormalizeFormation(formation string) string {
	switch strings.ToLower(strings.TrimSpace(formation)) {
	case "4-2-3-1", "4231":
		return Formation4231
	case "4-4-2", "442":
		return Formation442
	case "4-3-3 attack", "4-3-3-attack", "433 attack", "433a":
		return Formation433Attack
	default:
		return Formation433
	}
}

// FormationSlots returns the formation-owned slots in stable build-up order.
func FormationSlots(formation string) []string {
	def := formationDefinitions[NormalizeFormation(formation)]
	return append([]string(nil), def.Slots...)
}

// FormationForStyle gives every manager philosophy an explicit shape without
// adding mutable lineup state to an existing career save. The mapping is
// deterministic, so a legacy manager lacking any future formation field still
// produces the same XI after reload.
func FormationForStyle(style string) string {
	switch strings.ToLower(strings.TrimSpace(style)) {
	case "free_flowing":
		return Formation433Attack
	case "possession":
		return Formation4231
	case "low_block", "counter":
		return Formation442
	default:
		return Formation433
	}
}

// StartingSlot is one explicit, non-overlapping match role. Player.Position is
// the natural position; Slot is the assigned tactical position for this match.
type StartingSlot struct {
	Slot            string      `json:"tactical_slot"`
	NaturalPosition string      `json:"natural_position"`
	PositionFit     PositionFit `json:"position_fit"`
	*Player
}

// PlayersFromStartingSlots retains the formation order while exposing the
// legacy []*Player view used by match simulation calculations.
func PlayersFromStartingSlots(slots []StartingSlot) []*Player {
	players := make([]*Player, 0, len(slots))
	for _, assignment := range slots {
		if assignment.Player != nil {
			players = append(players, assignment.Player)
		}
	}
	return players
}

func normalizeTacticalPosition(position string) string {
	return strings.ToUpper(strings.TrimSpace(position))
}

var primaryPositionFits = map[string]map[string]PositionFit{
	"GK": {
		"GK": PositionFitNatural,
	},
	"LB": {
		"LB": PositionFitNatural, "LWB": PositionFitGood,
		"LCB": PositionFitAcceptable, "CB": PositionFitAcceptable,
	},
	"LCB": {
		"LCB": PositionFitNatural, "CB": PositionFitNatural, "RCB": PositionFitGood,
		"LB": PositionFitEmergency, "LWB": PositionFitEmergency, "RB": PositionFitEmergency,
	},
	"RCB": {
		"RCB": PositionFitNatural, "CB": PositionFitNatural, "LCB": PositionFitGood,
		"RB": PositionFitEmergency, "RWB": PositionFitEmergency, "LB": PositionFitEmergency,
	},
	"RB": {
		"RB": PositionFitNatural, "RWB": PositionFitGood,
		"RCB": PositionFitAcceptable, "CB": PositionFitAcceptable,
	},
	"LDM": {
		"LDM": PositionFitNatural, "CDM": PositionFitNatural, "DM": PositionFitNatural,
		"CM": PositionFitAcceptable, "LCM": PositionFitAcceptable, "RCM": PositionFitAcceptable,
	},
	"CDM": {
		"CDM": PositionFitNatural, "DM": PositionFitNatural, "LDM": PositionFitGood, "RDM": PositionFitGood,
		"CM": PositionFitAcceptable, "LCM": PositionFitAcceptable, "RCM": PositionFitAcceptable,
	},
	"RDM": {
		"RDM": PositionFitNatural, "CDM": PositionFitNatural, "DM": PositionFitNatural,
		"CM": PositionFitAcceptable, "LCM": PositionFitAcceptable, "RCM": PositionFitAcceptable,
	},
	"LM": {
		"LM": PositionFitNatural, "LW": PositionFitGood, "LWB": PositionFitGood,
		"LCM": PositionFitAcceptable, "CM": PositionFitAcceptable, "CAM": PositionFitAcceptable,
		"RM": PositionFitEmergency, "RW": PositionFitEmergency,
	},
	"LCM": {
		"LCM": PositionFitNatural, "CM": PositionFitNatural, "RCM": PositionFitGood, "LM": PositionFitGood,
		"CDM": PositionFitAcceptable, "DM": PositionFitAcceptable, "CAM": PositionFitAcceptable, "RM": PositionFitAcceptable,
	},
	"CM": {
		"CM": PositionFitNatural, "LCM": PositionFitGood, "RCM": PositionFitGood,
		"CDM": PositionFitGood, "DM": PositionFitGood, "CAM": PositionFitAcceptable, "LM": PositionFitAcceptable, "RM": PositionFitAcceptable,
	},
	"RCM": {
		"RCM": PositionFitNatural, "CM": PositionFitNatural, "LCM": PositionFitGood, "RM": PositionFitGood,
		"CDM": PositionFitAcceptable, "DM": PositionFitAcceptable, "CAM": PositionFitAcceptable, "LM": PositionFitAcceptable,
	},
	"RM": {
		"RM": PositionFitNatural, "RW": PositionFitGood, "RWB": PositionFitGood,
		"RCM": PositionFitAcceptable, "CM": PositionFitAcceptable, "CAM": PositionFitAcceptable,
		"LM": PositionFitEmergency, "LW": PositionFitEmergency,
	},
	"LAM": {
		"LAM": PositionFitNatural, "CAM": PositionFitGood, "AM": PositionFitGood, "LW": PositionFitGood,
		"LM": PositionFitAcceptable, "CM": PositionFitAcceptable, "RAM": PositionFitEmergency,
	},
	"CAM": {
		"CAM": PositionFitNatural, "AM": PositionFitNatural, "LAM": PositionFitGood, "RAM": PositionFitGood,
		"CM": PositionFitAcceptable, "LCM": PositionFitAcceptable, "RCM": PositionFitAcceptable, "CF": PositionFitAcceptable,
	},
	"RAM": {
		"RAM": PositionFitNatural, "CAM": PositionFitGood, "AM": PositionFitGood, "RW": PositionFitGood,
		"RM": PositionFitAcceptable, "CM": PositionFitAcceptable, "LAM": PositionFitEmergency,
	},
	"LW": {
		"LW": PositionFitNatural, "LM": PositionFitGood, "LF": PositionFitGood,
		"RW": PositionFitAcceptable, "RM": PositionFitAcceptable,
	},
	"LF": {
		"LF": PositionFitNatural, "LW": PositionFitGood, "CF": PositionFitGood, "ST": PositionFitAcceptable,
		"CAM": PositionFitAcceptable, "RF": PositionFitEmergency,
	},
	"CF": {
		"CF": PositionFitNatural, "ST": PositionFitGood, "CAM": PositionFitAcceptable,
		"LF": PositionFitAcceptable, "RF": PositionFitAcceptable, "LW": PositionFitEmergency, "RW": PositionFitEmergency,
	},
	"RF": {
		"RF": PositionFitNatural, "RW": PositionFitGood, "CF": PositionFitGood, "ST": PositionFitAcceptable,
		"CAM": PositionFitAcceptable, "LF": PositionFitEmergency,
	},
	"RW": {
		"RW": PositionFitNatural, "RM": PositionFitGood, "RF": PositionFitGood,
		"LW": PositionFitAcceptable, "LM": PositionFitAcceptable,
	},
	"ST": {
		"ST": PositionFitNatural, "CF": PositionFitNatural, "LF": PositionFitAcceptable, "RF": PositionFitAcceptable,
		"LW": PositionFitEmergency, "RW": PositionFitEmergency,
	},
	"LST": {
		"LST": PositionFitNatural, "ST": PositionFitNatural, "CF": PositionFitNatural, "LF": PositionFitGood,
		"RF": PositionFitAcceptable, "LW": PositionFitAcceptable, "RW": PositionFitEmergency,
	},
	"RST": {
		"RST": PositionFitNatural, "ST": PositionFitNatural, "CF": PositionFitNatural, "RF": PositionFitGood,
		"LF": PositionFitAcceptable, "RW": PositionFitAcceptable, "LW": PositionFitEmergency,
	},
}

func fitRank(fit PositionFit) int {
	switch fit {
	case PositionFitNatural:
		return 4
	case PositionFitGood:
		return 3
	case PositionFitAcceptable:
		return 2
	default:
		return 1
	}
}

func secondaryFit(fit PositionFit) PositionFit {
	switch fit {
	case PositionFitNatural:
		return PositionFitGood
	case PositionFitGood:
		return PositionFitAcceptable
	default:
		return fit
	}
}

// PositionFitForPlayer deterministically grades primary and learned secondary
// positions against a tactical slot. Unknown combinations are legal emergency
// assignments, never silently treated as natural.
func PositionFitForPlayer(player *Player, slot string) PositionFit {
	if player == nil {
		return PositionFitEmergency
	}
	slot = normalizeTacticalPosition(slot)
	lookup := primaryPositionFits[slot]
	primary := PositionFitEmergency
	if fit, ok := lookup[normalizeTacticalPosition(player.Position)]; ok {
		primary = fit
	}
	secondary := PositionFitEmergency
	if player.SecondaryPosition != "" {
		if fit, ok := lookup[normalizeTacticalPosition(player.SecondaryPosition)]; ok {
			secondary = secondaryFit(fit)
		}
	}
	if fitRank(secondary) > fitRank(primary) {
		return secondary
	}
	return primary
}

func slotCategory(slot string) string {
	switch normalizeTacticalPosition(slot) {
	case "GK":
		return "GK"
	case "LB", "LCB", "CB", "RCB", "RB", "LWB", "RWB":
		return "DEF"
	case "LDM", "CDM", "RDM", "DM", "LM", "LCM", "CM", "RCM", "RM":
		return "MID"
	default:
		return "FWD"
	}
}

func positionFitBonus(fit PositionFit) int {
	switch fit {
	case PositionFitNatural:
		return 3000
	case PositionFitGood:
		return 2400
	case PositionFitAcceptable:
		return 1600
	default:
		return 0
	}
}

func lineupAssignmentScore(player *Player, slot, competition string, matchweek int, style, focus string) int {
	if player == nil {
		return -1_000_000
	}
	fit := PositionFitForPlayer(player, slot)
	wk, adjusted := sortKey(player, competition, matchweek)
	adjusted += applyManagerBias(player, style, focus)
	score := positionFitBonus(fit) + adjusted*10 + wk*200 + 1
	if fit == PositionFitEmergency {
		category := player.Category
		if category == "" {
			category = GetPositionCategory(player.Position)
		}
		if category == slotCategory(slot) {
			score += 300
		} else if slot == "GK" {
			// A malformed squad without a goalkeeper still gets a stable,
			// football-sensible emergency preference.
			switch category {
			case "DEF":
				score += 250
			case "MID":
				score += 120
			case "FWD":
				score += 60
			}
		}
	}
	return score
}

func stablePlayerKey(player *Player) string {
	if player == nil {
		return "~nil"
	}
	if player.PlayerID != "" {
		return player.PlayerID
	}
	return strings.ToLower(strings.TrimSpace(player.FullName)) + "|" + normalizeTacticalPosition(player.Position)
}

// minimumCostAssignment is the rectangular Hungarian algorithm. Rows are
// formation slots and columns are players (plus dummy columns for short
// squads). Stable slot order and sorted player IDs make equal-score outcomes
// deterministic without using squad or response-array order as a role.
func minimumCostAssignment(weights [][]int) []int {
	n := len(weights)
	if n == 0 {
		return nil
	}
	m := len(weights[0])
	u := make([]int, n+1)
	v := make([]int, m+1)
	p := make([]int, m+1)
	way := make([]int, m+1)
	const inf = int(^uint(0)>>1) / 4

	for i := 1; i <= n; i++ {
		p[0] = i
		j0 := 0
		minv := make([]int, m+1)
		used := make([]bool, m+1)
		for j := 1; j <= m; j++ {
			minv[j] = inf
		}
		for {
			used[j0] = true
			i0 := p[j0]
			delta, j1 := inf, 0
			for j := 1; j <= m; j++ {
				if used[j] {
					continue
				}
				cost := -weights[i0-1][j-1]
				cur := cost - u[i0] - v[j]
				if cur < minv[j] {
					minv[j] = cur
					way[j] = j0
				}
				if minv[j] < delta || (minv[j] == delta && (j1 == 0 || j < j1)) {
					delta = minv[j]
					j1 = j
				}
			}
			for j := 0; j <= m; j++ {
				if used[j] {
					u[p[j]] += delta
					v[j] -= delta
				} else if j > 0 {
					minv[j] -= delta
				}
			}
			j0 = j1
			if p[j0] == 0 {
				break
			}
		}
		for {
			j1 := way[j0]
			p[j0] = p[j1]
			j0 = j1
			if j0 == 0 {
				break
			}
		}
	}

	assignment := make([]int, n)
	for i := range assignment {
		assignment[i] = -1
	}
	for j := 1; j <= m; j++ {
		if p[j] > 0 && p[j] <= n {
			assignment[p[j]-1] = j - 1
		}
	}
	return assignment
}

func assignPlayersToFormation(players []*Player, formation, competition string, matchweek int, style, focus string) []StartingSlot {
	formation = NormalizeFormation(formation)
	slots := FormationSlots(formation)
	if len(players) == 0 || len(slots) == 0 {
		return nil
	}

	pool := make([]*Player, 0, len(players))
	seen := make(map[string]bool, len(players))
	for _, player := range players {
		if player == nil {
			continue
		}
		key := stablePlayerKey(player)
		if seen[key] {
			continue
		}
		seen[key] = true
		pool = append(pool, player)
	}
	if len(pool) == 0 {
		return nil
	}
	sort.SliceStable(pool, func(i, j int) bool {
		ki, kj := stablePlayerKey(pool[i]), stablePlayerKey(pool[j])
		if ki != kj {
			return ki < kj
		}
		if pool[i].OVR != pool[j].OVR {
			return pool[i].OVR > pool[j].OVR
		}
		return normalizeTacticalPosition(pool[i].Position) < normalizeTacticalPosition(pool[j].Position)
	})

	// Goalkeeper is the only role with no meaningful interchange. Lock the
	// best deterministic keeper (or emergency keeper in a malformed squad)
	// before solving the outfield matching, so the optimizer cannot sacrifice
	// the goalkeeper merely to gain a tiny score elsewhere.
	keeperIndex := 0
	keeperScore := lineupAssignmentScore(pool[0], "GK", competition, matchweek, style, focus)
	for i := 1; i < len(pool); i++ {
		score := lineupAssignmentScore(pool[i], "GK", competition, matchweek, style, focus)
		if score > keeperScore {
			keeperIndex, keeperScore = i, score
		}
	}
	keeper := pool[keeperIndex]
	pool = append(pool[:keeperIndex:keeperIndex], pool[keeperIndex+1:]...)
	outfieldSlots := slots[1:]
	if len(outfieldSlots) == 0 || len(pool) == 0 {
		return []StartingSlot{{
			Slot:            "GK",
			NaturalPosition: normalizeTacticalPosition(keeper.Position),
			PositionFit:     PositionFitForPlayer(keeper, "GK"),
			Player:          keeper,
		}}
	}

	columns := len(pool)
	if columns < len(outfieldSlots) {
		columns = len(outfieldSlots)
	}
	weights := make([][]int, len(outfieldSlots))
	for i, slot := range outfieldSlots {
		weights[i] = make([]int, columns)
		for j := 0; j < columns; j++ {
			if j >= len(pool) {
				weights[i][j] = -1_000_000
				continue
			}
			weights[i][j] = lineupAssignmentScore(pool[j], slot, competition, matchweek, style, focus)
		}
	}

	assigned := minimumCostAssignment(weights)
	result := make([]StartingSlot, 0, minInt(len(slots), len(pool)+1))
	result = append(result, StartingSlot{
		Slot:            "GK",
		NaturalPosition: normalizeTacticalPosition(keeper.Position),
		PositionFit:     PositionFitForPlayer(keeper, "GK"),
		Player:          keeper,
	})
	for i, playerIndex := range assigned {
		if playerIndex < 0 || playerIndex >= len(pool) {
			continue
		}
		player := pool[playerIndex]
		result = append(result, StartingSlot{
			Slot:            outfieldSlots[i],
			NaturalPosition: normalizeTacticalPosition(player.Position),
			PositionFit:     PositionFitForPlayer(player, outfieldSlots[i]),
			Player:          player,
		})
	}
	return result
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// AssignPlayersToFormation reconstructs deterministic tactical roles for an
// already-selected XI, which is used for legacy reports lacking slot fields.
func AssignPlayersToFormation(players []*Player, formation string) []StartingSlot {
	return assignPlayersToFormation(players, formation, "", 1, "", "")
}
