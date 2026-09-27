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
				// Older than the summary window: archived to a summary.
				Matchweek: 1,
				Status:    "finished",
				Report: &matchreport.MatchReport{
					HomeGoals: 2,
					Heatmap:   matchreport.TouchHeatmapData{HomePoints: [][]float64{{0.2, 0.3}}, HomeZones: matchreport.ZoneSplit{Midfield: 100}},
					ShotMap:   matchreport.ShotMapData{XGFlow: []matchreport.XGFlowPoint{{Minute: 12}}, TotalHomeXG: 1.1},
				},
			},
			{
				// Inside the compacted band: visual payload dropped, report kept.
				Matchweek: 4,
				Status:    "finished",
				Report: &matchreport.MatchReport{
					HomeGoals: 1,
					Heatmap:   matchreport.TouchHeatmapData{HomePoints: [][]float64{{0.2, 0.3}}},
					ShotMap:   matchreport.ShotMapData{XGFlow: []matchreport.XGFlowPoint{{Minute: 12}}},
				},
			},
			{
				// Inside the full-detail window: untouched.
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

	// Aged beyond the summary window: full report replaced by the summary.
	if tm.Fixtures[0].Report != nil {
		t.Fatal("reports older than the summary window must be replaced by their summary")
	}
	if tm.Fixtures[0].ReportSummary == nil {
		t.Fatal("archived fixture must carry a report summary")
	}
	if tm.Fixtures[0].ReportSummary.HomeGoals != 2 {
		t.Fatalf("summary must keep the score, got %d", tm.Fixtures[0].ReportSummary.HomeGoals)
	}

	// Compacted band: visual payload dropped, authoritative facts kept.
	if tm.Fixtures[1].Report == nil {
		t.Fatal("fixtures inside the compacted band must keep their report")
	}
	if tm.Fixtures[1].Report.Heatmap.HomePoints != nil || tm.Fixtures[1].Report.ShotMap.XGFlow != nil {
		t.Fatal("finished reports older than three matchweeks should drop visual payload")
	}
	if tm.Fixtures[1].Report.HomeGoals != 1 || tm.Fixtures[1].ReportSummary != nil {
		t.Fatal("compacted fixtures keep the full report and carry no summary")
	}

	// Recent window: fully detailed.
	if tm.Fixtures[2].Report.Heatmap.HomePoints == nil || tm.Fixtures[2].Report.ShotMap.XGFlow == nil {
		t.Fatal("reports from the last three matchweeks must stay detailed")
	}
}

func TestCompactAgedReportsSummaryKeepsIdentifiers(t *testing.T) {
	rating := 8.4
	tm := &TournamentManager{
		CurrentMatchweek: 20,
		Fixtures: []Fixture{
			{
				Matchweek: 1,
				Status:    "finished",
				Report: &matchreport.MatchReport{
					HomeGoals: 3,
					AwayGoals: 1,
					HTHome:    1,
					HTAway:    0,
					Stats: matchreport.MatchStats{
						Home: matchreport.TeamStats{Possession: 58, Shots: 14, XG: 2.1},
						Away: matchreport.TeamStats{Possession: 42, Shots: 6, XG: 0.8},
					},
					Events: []matchreport.MatchEventItem{
						{Minute: 23, Type: "goal", PlayerID: "P1", PlayerName: "Scorer One", ClubID: "C1"},
						{Minute: 55, Type: "own_goal", PlayerID: "P2", PlayerName: "Scorer Two", ClubID: "C2"},
						{Minute: 70, Type: "red", PlayerID: "P3", PlayerName: "Sent Off", ClubID: "C2"},
						{Minute: 80, Type: "goal", Disallowed: true, PlayerID: "P4", PlayerName: "Ghost", ClubID: "C1"},
					},
					MOTM: &matchreport.MatchPlayerRow{PlayerID: "P1", FullName: "Scorer One", Rating: &rating},
				},
			},
		},
	}
	tm.compactAgedReportsUnlocked()
	s := tm.Fixtures[0].ReportSummary
	if s == nil || tm.Fixtures[0].Report != nil {
		t.Fatal("aged fixture must be summarized")
	}
	if s.HomeGoals != 3 || s.AwayGoals != 1 || s.HTHome != 1 || s.HTAway != 0 {
		t.Fatalf("summary score wrong: %+v", s)
	}
	if s.PossessionHome != 58 || s.ShotsHome != 14 || s.XGHome != 2.1 {
		t.Fatalf("summary stats wrong: %+v", s)
	}
	if len(s.Scorers) != 2 {
		t.Fatalf("disallowed goals must be excluded, got %+v", s.Scorers)
	}
	if s.Scorers[0].PlayerID != "P1" || s.Scorers[0].ClubID != "C1" || s.Scorers[0].Type != "goal" {
		t.Fatalf("scorer identifiers must be retained: %+v", s.Scorers[0])
	}
	if s.Scorers[1].Type != "own_goal" || s.Scorers[1].ClubID != "C2" {
		t.Fatalf("own goal attribution wrong: %+v", s.Scorers[1])
	}
	if len(s.RedCards) != 1 || s.RedCards[0].PlayerID != "P3" || s.RedCards[0].ClubID != "C2" {
		t.Fatalf("red card identifiers must be retained: %+v", s.RedCards)
	}
	if s.MOTM == nil || s.MOTM.PlayerID != "P1" || s.MOTM.Rating != "8.4" {
		t.Fatalf("MOTM must be retained: %+v", s.MOTM)
	}
}

func TestCompactAgedReportsIsDeterministic(t *testing.T) {
	build := func() *TournamentManager {
		rating := 7.2
		return &TournamentManager{
			CurrentMatchweek: 30,
			Fixtures: []Fixture{
				{Matchweek: 1, Status: "finished", Report: &matchreport.MatchReport{
					HomeGoals: 1, AwayGoals: 1,
					Events: []matchreport.MatchEventItem{{Minute: 30, Type: "goal", PlayerID: "P1", PlayerName: "A", ClubID: "C1"}},
					MOTM:    &matchreport.MatchPlayerRow{PlayerID: "P1", FullName: "A", Rating: &rating},
				}},
				{Matchweek: 25, Status: "finished", Report: &matchreport.MatchReport{HomeGoals: 0, AwayGoals: 2}},
			},
		}
	}
	a, b := build(), build()
	a.compactAgedReportsUnlocked()
	b.compactAgedReportsUnlocked()
	if a.Fixtures[0].ReportSummary.HomeGoals != b.Fixtures[0].ReportSummary.HomeGoals ||
		len(a.Fixtures[0].ReportSummary.Scorers) != len(b.Fixtures[0].ReportSummary.Scorers) {
		t.Fatal("compaction must be deterministic")
	}
	if a.Fixtures[1].Report == nil || b.Fixtures[1].Report == nil {
		t.Fatal("fixtures inside the full window must keep reports")
	}
}
