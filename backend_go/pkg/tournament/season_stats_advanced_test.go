package tournament

import (
	"testing"

	"football_sim/pkg/matchreport"
	"football_sim/pkg/models"
)

func TestAdvancedSeasonStatsAggregatesAndRanks(t *testing.T) {
	home := &models.Club{ClubID: "SH", ClubName: "Stats FC", ShortName: "SFC"}
	away := &models.Club{ClubID: "SA", ClubName: "Other FC", ShortName: "OFC"}
	star := &models.Player{
		PlayerID: "STAR", FullName: "Star Man", Position: "ST", Category: "FWD",
		ClubID: "SH", OVR: 88, Age: 26, Appearances: 10, Goals: 12, Assists: 4,
		RecentRatings: []float64{8.0, 8.4},
	}
	squadPlayer := &models.Player{
		PlayerID: "SQUAD", FullName: "Squad Man", Position: "CM", Category: "MID",
		ClubID: "SH", OVR: 74, Age: 29, Appearances: 10, Goals: 1, Assists: 1,
	}
	home.Squad = []*models.Player{star, squadPlayer}
	away.Squad = []*models.Player{{PlayerID: "AWAYP", FullName: "Away Man", Position: "ST", Category: "FWD", ClubID: "SA", OVR: 80, Age: 24, Appearances: 10, Goals: 5, Assists: 2}}

	hg, ag := 3, 1
	tm := &TournamentManager{
		Clubs:     map[string]*models.Club{"SH": home, "SA": away},
		ClubsList: []*models.Club{home, away},
		SeasonName: "2026-27",
		Fixtures: []Fixture{{
			FixtureID: "SX1", Matchweek: 2, Competition: "premier-league",
			HomeID: "SH", AwayID: "SA", Status: "finished", HomeGoals: &hg, AwayGoals: &ag,
			Report: &matchreport.MatchReport{
				Stats: matchreport.MatchStats{
					Home: matchreport.TeamStats{Possession: 60, PassAccuracy: 88, Shots: 15, XG: 2.4},
					Away: matchreport.TeamStats{Possession: 40, PassAccuracy: 78, Shots: 7, XG: 0.9},
				},
				Heatmap: matchreport.TouchHeatmapData{
					HomeZones: matchreport.ZoneSplit{Defensive: 20, Midfield: 50, Attacking: 30},
					AwayZones: matchreport.ZoneSplit{Defensive: 50, Midfield: 30, Attacking: 20},
				},
				ShotMap: matchreport.ShotMapData{
					Shots: []matchreport.ShotMapItem{
						{Shooter: matchreport.ShotShooter{PlayerID: "STAR"}, XG: 0.4, Y: 0.9},
						{Shooter: matchreport.ShotShooter{PlayerID: "STAR"}, XG: 0.1, Y: 0.5},
					},
					TotalHomeXG: 0.5, TotalAwayXG: 0,
				},
			},
		}},
	}

	payload := tm.GetAdvancedSeasonStats()
	players := payload["players"].([]advancedPlayerRow)
	if len(players) != 3 {
		t.Fatalf("expected 3 player rows, got %d", len(players))
	}
	byID := map[string]advancedPlayerRow{}
	for _, row := range players {
		byID[row.PlayerID] = row
	}
	starRow := byID["STAR"]
	if starRow.Goals != 12 || starRow.Assists != 4 {
		t.Fatalf("star ledger wrong: %+v", starRow)
	}
	if starRow.AvgRating != 8.2 {
		t.Fatalf("star avg rating = %v, want 8.2", starRow.AvgRating)
	}
	if starRow.Shots != 2 || starRow.XG != 0.5 {
		t.Fatalf("star shot aggregation wrong: %+v", starRow)
	}
	if starRow.ShotsBox != 1 || starRow.ShotsOutside != 1 || starRow.XGBox != 0.4 || starRow.XGOutside != 0.1 {
		t.Fatalf("star zone split wrong: %+v", starRow)
	}
	// Percentiles: star (16 contributions) > away (7) > squad (2).
	if starRow.Percentile != 100 {
		t.Fatalf("top contributor percentile = %d, want 100", starRow.Percentile)
	}
	if byID["SQUAD"].Percentile != 0 {
		t.Fatalf("bottom contributor percentile = %d, want 0", byID["SQUAD"].Percentile)
	}
	// Rows are sorted by percentile descending.
	if players[0].PlayerID != "STAR" {
		t.Fatalf("rows not sorted by percentile: %+v", players[0])
	}

	clubs := payload["clubs"].([]advancedClubRow)
	if len(clubs) != 2 {
		t.Fatalf("expected 2 club rows, got %d", len(clubs))
	}
	var sfc advancedClubRow
	for _, c := range clubs {
		if c.ClubID == "SH" {
			sfc = c
		}
	}
	if sfc.Matches != 1 || sfc.AvgPossession != 60 || sfc.AvgPassAcc != 88 || sfc.AvgXG != 2.4 || sfc.AvgXGAgainst != 0.9 {
		t.Fatalf("club trend wrong: %+v", sfc)
	}
	if sfc.TerritoryDef != 20 || sfc.TerritoryMid != 50 || sfc.TerritoryAtt != 30 {
		t.Fatalf("territory split wrong: %+v", sfc)
	}
}

func TestAdvancedSeasonStatsIsPureAndDeterministic(t *testing.T) {
	home := &models.Club{ClubID: "SH", ShortName: "SFC", Squad: []*models.Player{{
		PlayerID: "P1", FullName: "A", Position: "ST", Category: "FWD", ClubID: "SH",
		Appearances: 5, Goals: 3,
	}}}
	tm := &TournamentManager{
		Clubs: map[string]*models.Club{"SH": home}, ClubsList: []*models.Club{home},
		SeasonName: "2026-27",
	}
	a := tm.GetAdvancedSeasonStats()
	b := tm.GetAdvancedSeasonStats()
	if len(a["players"].([]advancedPlayerRow)) != len(b["players"].([]advancedPlayerRow)) {
		t.Fatal("repeated calls must return identical row counts")
	}
	ra, rb := a["players"].([]advancedPlayerRow), b["players"].([]advancedPlayerRow)
	for i := range ra {
		if ra[i] != rb[i] {
			t.Fatalf("row %d differs between calls: %+v vs %+v", i, ra[i], rb[i])
		}
	}
}
