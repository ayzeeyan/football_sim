package matchreport

import (
	"sort"
	"strings"

	"football_sim/pkg/models"
)

type reportAssignment struct {
	NaturalPosition string
	TacticalSlot    string
	PositionFit     string
}

func normalizedPosition(position string) string {
	return strings.ToUpper(strings.TrimSpace(position))
}

func playerIDs(players []*models.Player) map[string]bool {
	ids := make(map[string]bool, len(players))
	for _, player := range players {
		if player != nil && player.PlayerID != "" {
			ids[player.PlayerID] = true
		}
	}
	return ids
}

// NormalizeLineupSlots validates an explicit assignment against the supplied
// XI. Missing, duplicate, or stale slots trigger a deterministic formation +
// natural-position reconstruction for legacy payloads.
func NormalizeLineupSlots(players []*models.Player, assignments []models.StartingSlot, formation string) []models.StartingSlot {
	formation = models.NormalizeFormation(formation)
	wantPlayers := playerIDs(players)
	validSlots := make(map[string]bool)
	for _, slot := range models.FormationSlots(formation) {
		validSlots[slot] = true
	}
	seenPlayers := make(map[string]bool, len(assignments))
	seenSlots := make(map[string]bool, len(assignments))
	valid := len(assignments) == len(wantPlayers) && len(wantPlayers) > 0
	if valid {
		for _, assignment := range assignments {
			if assignment.Player == nil || !wantPlayers[assignment.PlayerID] || seenPlayers[assignment.PlayerID] ||
				!validSlots[normalizedPosition(assignment.Slot)] || seenSlots[normalizedPosition(assignment.Slot)] {
				valid = false
				break
			}
			seenPlayers[assignment.PlayerID] = true
			seenSlots[normalizedPosition(assignment.Slot)] = true
		}
	}
	if !valid {
		return models.AssignPlayersToFormation(players, formation)
	}

	bySlot := make(map[string]models.StartingSlot, len(assignments))
	for _, assignment := range assignments {
		assignment.Slot = normalizedPosition(assignment.Slot)
		assignment.NaturalPosition = normalizedPosition(assignment.Player.Position)
		assignment.PositionFit = models.PositionFitForPlayer(assignment.Player, assignment.Slot)
		bySlot[assignment.Slot] = assignment
	}
	ordered := make([]models.StartingSlot, 0, len(assignments))
	for _, slot := range models.FormationSlots(formation) {
		if assignment, ok := bySlot[slot]; ok {
			ordered = append(ordered, assignment)
		}
	}
	return ordered
}

func assignmentMap(slots []models.StartingSlot) map[string]reportAssignment {
	result := make(map[string]reportAssignment, len(slots))
	for _, slot := range slots {
		if slot.Player == nil {
			continue
		}
		result[slot.PlayerID] = reportAssignment{
			NaturalPosition: normalizedPosition(slot.Player.Position),
			TacticalSlot:    normalizedPosition(slot.Slot),
			PositionFit:     string(models.PositionFitForPlayer(slot.Player, slot.Slot)),
		}
	}
	return result
}

func indexPlayers(groups ...[]*models.Player) map[string]*models.Player {
	result := make(map[string]*models.Player)
	for _, group := range groups {
		for _, player := range group {
			if player != nil && player.PlayerID != "" {
				result[player.PlayerID] = player
			}
		}
	}
	return result
}

func applyAssignmentToRows(rows []MatchPlayerRow, assignments map[string]reportAssignment) {
	for i := range rows {
		row := &rows[i]
		row.NaturalPosition = normalizedPosition(row.Position)
		if assignment, ok := assignments[row.PlayerID]; ok {
			row.NaturalPosition = assignment.NaturalPosition
			row.TacticalSlot = assignment.TacticalSlot
			row.PositionFit = assignment.PositionFit
		}
	}
}

// ApplyTacticalAssignments stamps kickoff roles on starters and propagates the
// outgoing player's role to each substitute. A substitution changes the actor,
// not the formation coordinate, so out-of-position use remains explicit.
func ApplyTacticalAssignments(starters, bench []MatchPlayerRow, slots []models.StartingSlot, benchPlayers []*models.Player, events []MatchEventItem, side string) {
	assignments := assignmentMap(slots)
	allPlayers := indexPlayers(models.PlayersFromStartingSlots(slots), benchPlayers)
	orderedEvents := append([]MatchEventItem(nil), events...)
	sort.SliceStable(orderedEvents, func(i, j int) bool {
		if orderedEvents[i].Minute != orderedEvents[j].Minute {
			return orderedEvents[i].Minute < orderedEvents[j].Minute
		}
		return orderedEvents[i].Seq < orderedEvents[j].Seq
	})
	for _, event := range orderedEvents {
		if event.Side != side || event.Type != "sub" || event.PlayerOut == nil || event.PlayerIn == nil {
			continue
		}
		outAssignment, ok := assignments[event.PlayerOut.PlayerID]
		if !ok {
			continue
		}
		inPlayer := allPlayers[event.PlayerIn.PlayerID]
		fit := models.PositionFitEmergency
		natural := normalizedPosition(event.PlayerIn.Position)
		if inPlayer != nil {
			fit = models.PositionFitForPlayer(inPlayer, outAssignment.TacticalSlot)
			natural = normalizedPosition(inPlayer.Position)
		}
		assignments[event.PlayerIn.PlayerID] = reportAssignment{
			NaturalPosition: natural,
			TacticalSlot:    outAssignment.TacticalSlot,
			PositionFit:     string(fit),
		}
	}
	applyAssignmentToRows(starters, assignments)
	applyAssignmentToRows(bench, assignments)
}

func rowsAsPlayers(rows []MatchPlayerRow) []*models.Player {
	players := make([]*models.Player, 0, len(rows))
	for _, row := range rows {
		players = append(players, &models.Player{
			PlayerID: row.PlayerID,
			FullName: row.FullName,
			Position: row.Position,
			Category: row.Category,
			OVR:      row.OVR,
			Age:      row.Age,
		})
	}
	return players
}

// LineupSlotsFromRows restores explicit report assignments to live player
// pointers. It is used when a knockout slate is reassembled after extra time.
func LineupSlotsFromRows(players []*models.Player, rows []MatchPlayerRow, formation string) []models.StartingSlot {
	byID := indexPlayers(players)
	assignments := make([]models.StartingSlot, 0, len(rows))
	for _, row := range rows {
		player := byID[row.PlayerID]
		if player == nil || row.TacticalSlot == "" {
			return models.AssignPlayersToFormation(players, formation)
		}
		assignments = append(assignments, models.StartingSlot{Slot: row.TacticalSlot, Player: player})
	}
	return NormalizeLineupSlots(players, assignments, formation)
}

func backfillSide(rows, bench []MatchPlayerRow, events []MatchEventItem, side, formation string) ([]MatchPlayerRow, []MatchPlayerRow) {
	players := rowsAsPlayers(rows)
	slots := LineupSlotsFromRows(players, rows, formation)
	benchPlayers := rowsAsPlayers(bench)
	ApplyTacticalAssignments(rows, bench, slots, benchPlayers, events, side)
	return rows, bench
}

// BackfillTacticalSlots migrates additive formation metadata in memory. Old
// reports remain readable, and the corrected fields are written naturally on
// the next ordinary career snapshot without replacing the user's save.
func BackfillTacticalSlots(report *MatchReport, homeFormation, awayFormation string) {
	if report == nil {
		return
	}
	if report.HomeFormation == "" {
		report.HomeFormation = models.NormalizeFormation(homeFormation)
	} else {
		report.HomeFormation = models.NormalizeFormation(report.HomeFormation)
	}
	if report.AwayFormation == "" {
		report.AwayFormation = models.NormalizeFormation(awayFormation)
	} else {
		report.AwayFormation = models.NormalizeFormation(report.AwayFormation)
	}
	report.HomeXI, report.HomeBench = backfillSide(report.HomeXI, report.HomeBench, report.Events, "home", report.HomeFormation)
	report.AwayXI, report.AwayBench = backfillSide(report.AwayXI, report.AwayBench, report.Events, "away", report.AwayFormation)
	if report.MOTM == nil {
		return
	}
	groups := [][]MatchPlayerRow{report.HomeXI, report.HomeBench, report.AwayXI, report.AwayBench}
	for _, group := range groups {
		for _, row := range group {
			if row.PlayerID == report.MOTM.PlayerID {
				report.MOTM.NaturalPosition = row.NaturalPosition
				report.MOTM.TacticalSlot = row.TacticalSlot
				report.MOTM.PositionFit = row.PositionFit
				return
			}
		}
	}
}
