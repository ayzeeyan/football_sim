package matchreport

import (
	"encoding/json"
	"math/rand"
	"testing"

	"football_sim/pkg/models"
)

func reportTacticalPlayer(id, position string, ovr int) *models.Player {
	return &models.Player{
		PlayerID: id, FullName: id, Position: position,
		Category: models.GetPositionCategory(position), OVR: ovr,
		Fitness: 80, Sharpness: 70, Morale: 70,
	}
}

func report4231Lineup(prefix string) []*models.Player {
	return []*models.Player{
		reportTacticalPlayer(prefix+"-GK", "GK", 80),
		reportTacticalPlayer(prefix+"-LB", "LB", 80),
		reportTacticalPlayer(prefix+"-CB1", "CB", 80),
		reportTacticalPlayer(prefix+"-CB2", "CB", 79),
		reportTacticalPlayer(prefix+"-RB", "RB", 80),
		reportTacticalPlayer(prefix+"-DM1", "CDM", 80),
		reportTacticalPlayer(prefix+"-DM2", "CDM", 79),
		reportTacticalPlayer(prefix+"-LW", "LW", 80),
		reportTacticalPlayer(prefix+"-CAM", "CAM", 84),
		reportTacticalPlayer(prefix+"-RW", "RW", 80),
		reportTacticalPlayer(prefix+"-ST", "ST", 80),
	}
}

func findReportRow(rows []MatchPlayerRow, id string) *MatchPlayerRow {
	for i := range rows {
		if rows[i].PlayerID == id {
			return &rows[i]
		}
	}
	return nil
}

func TestAssembleReportPreservesCantalejoAssignedCAMThroughJSON(t *testing.T) {
	home := report4231Lineup("H")
	home[8].PlayerID = "WK_Maverick_Cantalejo"
	home[8].FullName = "Maverick Cantalejo"
	away := report4231Lineup("A")
	report := AssembleReport(InstantPayload{
		HomeXI: home, AwayXI: away,
		HomeSlots:     models.AssignPlayersToFormation(home, models.Formation4231),
		AwaySlots:     models.AssignPlayersToFormation(away, models.Formation4231),
		HomeFormation: models.Formation4231,
		AwayFormation: models.Formation4231,
	}, "instant", rand.New(rand.NewSource(42)))

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	var restored MatchReport
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal report: %v", err)
	}
	row := findReportRow(restored.HomeXI, "WK_Maverick_Cantalejo")
	if row == nil || row.Position != "CAM" || row.NaturalPosition != "CAM" || row.TacticalSlot != "CAM" || row.PositionFit != string(models.PositionFitNatural) {
		t.Fatalf("Cantalejo report row after round-trip = %+v", row)
	}
	if restored.HomeFormation != models.Formation4231 || restored.AwayFormation != models.Formation4231 {
		t.Fatalf("formations after round-trip = %q/%q", restored.HomeFormation, restored.AwayFormation)
	}
}

func TestBackfillTacticalSlotsMigratesLegacyReportDeterministically(t *testing.T) {
	players := report4231Lineup("H")
	players[8].PlayerID = "WK_Maverick_Cantalejo"
	players[8].FullName = "Maverick Cantalejo"
	rows := RateXI(players, nil, "home", 0, false, true, rand.New(rand.NewSource(2)))
	// Legacy report order is deliberately unrelated to formation order.
	for left, right := 0, len(rows)-1; left < right; left, right = left+1, right-1 {
		rows[left], rows[right] = rows[right], rows[left]
	}
	report := &MatchReport{HomeXI: rows}
	BackfillTacticalSlots(report, models.Formation4231, models.Formation433)
	row := findReportRow(report.HomeXI, "WK_Maverick_Cantalejo")
	if row == nil || row.TacticalSlot != "CAM" || row.NaturalPosition != "CAM" || row.PositionFit != string(models.PositionFitNatural) {
		t.Fatalf("legacy Cantalejo backfill = %+v", row)
	}
	seen := make(map[string]bool, len(report.HomeXI))
	for _, candidate := range report.HomeXI {
		if candidate.TacticalSlot == "" || seen[candidate.TacticalSlot] {
			t.Fatalf("legacy migration produced missing/duplicate slot %q", candidate.TacticalSlot)
		}
		seen[candidate.TacticalSlot] = true
	}
}
