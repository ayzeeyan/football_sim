package tournament

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"testing"
)

type wonderkidAuditRow struct {
	PlayerID string
	Age int
	OVR int
	Potential int
	HeightCM float64
	WeightKG float64
	ClubID string
}

func captureWonderkidAudit(tm *TournamentManager) []wonderkidAuditRow {
	rows := make([]wonderkidAuditRow, 0, 12)
	for _, club := range tm.ClubsList {
		for _, p := range club.Squad {
			if p == nil || !p.UniverseWonderkid {
				continue
			}
			row := wonderkidAuditRow{PlayerID: p.PlayerID, Age: p.Age, OVR: p.OVR, ClubID: p.ClubID}
			if tm.GrowthEngine != nil {
				if bio := tm.GrowthEngine.Biometrics[p.PlayerID]; bio != nil {
					row.Potential = bio.Potential
					row.HeightCM = math.Round(bio.CurrentHeightCM*100) / 100
					row.WeightKG = math.Round(bio.CurrentWeightKG*100) / 100
				}
			}
			rows = append(rows, row)
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].PlayerID < rows[j].PlayerID })
	return rows
}

func auditFiveSeasons(t *testing.T, seed int64) map[int][]wonderkidAuditRow {
	t.Helper()
	tm := deterministicUniverse(t, seed)
	out := map[int][]wonderkidAuditRow{}
	previous := captureWonderkidAudit(tm)
	if len(previous) != 12 {
		t.Fatalf("initial canonical wonderkids=%d want 12", len(previous))
	}

	for season := 1; season <= 5; season++ {
		batch := tm.SimulateBatchWeeks(LeagueRounds)
		if batch.Status != "success" || !batch.SeasonFinished {
			t.Fatalf("season %d simulation failed: %+v", season, batch)
		}
		end := captureWonderkidAudit(tm)
		if len(end) != 12 {
			t.Fatalf("season %d canonical wonderkids=%d want 12", season, len(end))
		}
		for i, row := range end {
			if row.OVR < 0 || row.OVR > 100 {
				t.Fatalf("season %d %s OVR=%d outside 0..100", season, row.PlayerID, row.OVR)
			}
			if row.Potential < 90 || row.Potential > 100 {
				t.Fatalf("season %d %s potential=%d is implausible for canonical wonderkid", season, row.PlayerID, row.Potential)
			}
			if row.HeightCM <= 0 || row.WeightKG <= 0 || math.IsNaN(row.HeightCM) || math.IsNaN(row.WeightKG) {
				t.Fatalf("season %d %s invalid biometrics h=%.2f w=%.2f", season, row.PlayerID, row.HeightCM, row.WeightKG)
			}
			if i < len(previous) && previous[i].PlayerID == row.PlayerID {
				if row.OVR < previous[i].OVR {
					t.Fatalf("season %d %s regressed from OVR %d to %d while still a developing teenager", season, row.PlayerID, previous[i].OVR, row.OVR)
				}
				if row.HeightCM+0.01 < previous[i].HeightCM {
					t.Fatalf("season %d %s height regressed from %.2f to %.2f", season, row.PlayerID, previous[i].HeightCM, row.HeightCM)
				}
			}
		}
		if season == 1 || season == 3 || season == 5 {
			out[season] = append([]wonderkidAuditRow(nil), end...)
		}

		reset := tm.ResetNewSeason()
		if reset["status"] != "success" {
			t.Fatalf("season %d reset failed: %#v", season, reset)
		}
		if err := tm.ValidateWorldState(); err != nil {
			t.Fatalf("season %d post-reset world invalid: %v", season, err)
		}
		previous = captureWonderkidAudit(tm)
	}
	return out
}

func TestWonderkidGrowthAuditDeterministicAtOneThreeFiveSeasons(t *testing.T) {
	const seed int64 = 56001337
	a := auditFiveSeasons(t, seed)
	b := auditFiveSeasons(t, seed)
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("same-seed 1/3/5-season growth audit diverged\nA=%s\nB=%s", fmt.Sprint(a), fmt.Sprint(b))
	}
	for _, season := range []int{1, 3, 5} {
		if len(a[season]) != 12 {
			t.Fatalf("season %d report has %d rows want 12", season, len(a[season]))
		}
	}
}
