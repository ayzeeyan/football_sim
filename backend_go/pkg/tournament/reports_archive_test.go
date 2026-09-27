package tournament

import (
	"testing"

	"football_sim/pkg/matchreport"
)

func TestCompactAgedReportsKeepsRecentDetail(t *testing.T) {
	tm := &TournamentManager{
		CurrentMatchweek: 10,
		Fixtures: []Fixture{
			{
				Matchweek: 1,
				Status:    "finished",
				Report: &matchreport.MatchReport{
					HomeGoals: 2,
					Heatmap:   matchreport.TouchHeatmapData{HomePoints: [][]float64{{0.2, 0.3}}, HomeZones: matchreport.ZoneSplit{Midfield: 100}},
					ShotMap:   matchreport.ShotMapData{XGFlow: []matchreport.XGFlowPoint{{Minute: 12}}, TotalHomeXG: 1.1},
				},
			},
			{
				Matchweek: 9,
				Status:    "finished",
				Report: &matchreport.MatchReport{
					Heatmap: matchreport.TouchHeatmapData{HomePoints: [][]float64{{0.5, 0.5}}},
					ShotMap: matchreport.ShotMapData{XGFlow: []matchreport.XGFlowPoint{{Minute: 40}}},
				},
			},
		},
	}
	tm.compactAgedReportsUnlocked()
	if tm.Fixtures[0].Report.Heatmap.HomePoints != nil || tm.Fixtures[0].Report.ShotMap.XGFlow != nil || tm.Fixtures[0].Report.ShotMap.Shots != nil {
		t.Fatal("finished reports older than three matchweeks should drop visual payload")
	}
	if tm.Fixtures[0].Report.HomeGoals != 2 || tm.Fixtures[0].Report.Heatmap.HomeZones.Midfield != 100 {
		t.Fatal("authoritative match facts must survive archive compaction")
	}
	if tm.Fixtures[1].Report.Heatmap.HomePoints == nil || tm.Fixtures[1].Report.ShotMap.XGFlow == nil {
		t.Fatal("reports from the last three matchweeks must stay detailed")
	}
}
