package tournament

import (
	"sort"

	"football_sim/pkg/models"
)

// advancedPlayerRow is one player's season aggregate for the statistics
// centre. Goals, assists, appearances, and minutes come from the
// authoritative player ledgers; shot-level xG is aggregated from the shot
// maps that survive the report retention window.
type advancedPlayerRow struct {
	PlayerID    string  `json:"player_id"`
	FullName    string  `json:"full_name"`
	Position    string  `json:"position"`
	Category    string  `json:"category"`
	ClubID      string  `json:"club_id"`
	ClubShort   string  `json:"club_short"`
	OVR         int     `json:"ovr"`
	Age         int     `json:"age"`
	Appearances int     `json:"appearances"`
	Minutes     int     `json:"minutes"`
	Goals       int     `json:"goals"`
	Assists     int     `json:"assists"`
	AvgRating   float64 `json:"avg_rating"`
	Shots       int     `json:"shots"`
	XG          float64 `json:"xg"`
	// Zone buckets from available shot maps (0-1 pitch coordinates).
	ShotsBox     int     `json:"shots_box"`
	ShotsOutside int     `json:"shots_outside"`
	XGBox        float64 `json:"xg_box"`
	XGOutside    float64 `json:"xg_outside"`
	Percentile   int     `json:"percentile"`
}

// advancedClubRow is one club's season trend from finished-fixture team
// stats (which survive report compaction) and heatmap zone splits.
type advancedClubRow struct {
	ClubID        string  `json:"club_id"`
	ClubName      string  `json:"club_name"`
	ShortName     string  `json:"short_name"`
	Matches       int     `json:"matches"`
	AvgPossession float64 `json:"avg_possession"`
	AvgPassAcc    float64 `json:"avg_pass_accuracy"`
	AvgShots      float64 `json:"avg_shots"`
	AvgXG         float64 `json:"avg_xg"`
	AvgXGAgainst  float64 `json:"avg_xg_against"`
	// Territory: share of time in each third (heatmap zone splits).
	TerritoryDef float64 `json:"territory_defensive"`
	TerritoryMid float64 `json:"territory_midfield"`
	TerritoryAtt float64 `json:"territory_attacking"`
}

// GetAdvancedSeasonStats feeds the statistics centre: per-player season
// aggregates with percentile ranks, per-club team trends, and shot-zone
// splits. It is a pure read of resolved state — no simulation, no
// randomness — so repeated calls return identical payloads.
func (tm *TournamentManager) GetAdvancedSeasonStats() map[string]interface{} {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	// 1. Per-player rows from the authoritative ledgers.
	rows := make([]advancedPlayerRow, 0, 256)
	ratingSum := map[string]float64{}
	ratingN := map[string]int{}
	for _, club := range tm.ClubsList {
		if club == nil {
			continue
		}
		for _, p := range club.Squad {
			if p == nil || p.Appearances == 0 {
				continue
			}
			short := club.ShortName
			rows = append(rows, advancedPlayerRow{
				PlayerID: p.PlayerID, FullName: p.FullName, Position: p.Position,
				Category: p.Category, ClubID: club.ClubID, ClubShort: short,
				OVR: p.OVR, Age: p.Age,
				Appearances: p.Appearances, Minutes: seasonMinutesLedger(p),
				Goals: p.Goals, Assists: p.Assists,
			})
			if len(p.RecentRatings) > 0 {
				sum := 0.0
				for _, r := range p.RecentRatings {
					sum += r
				}
				ratingSum[p.PlayerID] = sum
				ratingN[p.PlayerID] = len(p.RecentRatings)
			}
		}
	}

	// 2. Shot-zone splits from the shot maps inside the retention window.
	shotXG := map[string]*advancedPlayerRow{}
	scanShots := func(fixtures []Fixture) {
		for i := range fixtures {
			f := &fixtures[i]
			if f.Status != "finished" || f.Report == nil {
				continue
			}
			for _, shot := range f.Report.ShotMap.Shots {
				pid := shot.Shooter.PlayerID
				if pid == "" {
					continue
				}
				row := shotXG[pid]
				if row == nil {
					row = &advancedPlayerRow{}
					shotXG[pid] = row
				}
				row.Shots++
				row.XG += shot.XG
				// Pitch coordinates: y >= 0.75 is inside the box area for
				// this dataset's shot map; everything else is outside.
				if shot.Y >= 0.75 {
					row.ShotsBox++
					row.XGBox += shot.XG
				} else {
					row.ShotsOutside++
					row.XGOutside += shot.XG
				}
			}
		}
	}
	scanShots(tm.Fixtures)
	scanShots(tm.UCLFixtures)
	scanShots(tm.SuperCupFixtures)
	if tm.World != nil {
		scanShots(tm.World.Fixtures)
	}
	for i := range rows {
		if s := shotXG[rows[i].PlayerID]; s != nil {
			rows[i].Shots = s.Shots
			rows[i].XG = round2(s.XG)
			rows[i].ShotsBox = s.ShotsBox
			rows[i].ShotsOutside = s.ShotsOutside
			rows[i].XGBox = round2(s.XGBox)
			rows[i].XGOutside = round2(s.XGOutside)
		}
		if n := ratingN[rows[i].PlayerID]; n > 0 {
			rows[i].AvgRating = round2(ratingSum[rows[i].PlayerID] / float64(n))
		}
	}

	// 3. Percentile ranks on goal contributions (goals + assists),
	// computed against players with at least one appearance.
	contribution := func(r advancedPlayerRow) int { return r.Goals + r.Assists }
	sorted := append([]advancedPlayerRow{}, rows...)
	sort.Slice(sorted, func(i, j int) bool { return contribution(sorted[i]) > contribution(sorted[j]) })
	percentile := make(map[string]int, len(rows))
	for i, r := range sorted {
		if len(sorted) == 1 {
			percentile[r.PlayerID] = 100
			continue
		}
		// Rank position as a percentile of the population; ties share the
		// higher percentile of the tied group.
		rank := i
		for rank > 0 && contribution(sorted[rank-1]) == contribution(r) {
			rank--
		}
		percentile[r.PlayerID] = int(float64(len(sorted)-1-rank) / float64(len(sorted)-1) * 100)
	}
	for i := range rows {
		rows[i].Percentile = percentile[rows[i].PlayerID]
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Percentile != rows[j].Percentile {
			return rows[i].Percentile > rows[j].Percentile
		}
		if contribution(rows[i]) != contribution(rows[j]) {
			return contribution(rows[i]) > contribution(rows[j])
		}
		return rows[i].PlayerID < rows[j].PlayerID
	})

	// 4. Per-club trends from finished-fixture team stats and heatmaps.
	clubAgg := map[string]*advancedClubRow{}
	territory := map[string][3]int{} // defensive, midfield, attacking sums
	scanTeams := func(fixtures []Fixture) {
		for i := range fixtures {
			f := &fixtures[i]
			if f.Status != "finished" || f.Report == nil {
				continue
			}
			report := f.Report
			home := tm.clubTrendRow(clubAgg, f.HomeID)
			away := tm.clubTrendRow(clubAgg, f.AwayID)
			home.Matches++
			away.Matches++
			home.AvgPossession += float64(report.Stats.Home.Possession)
			away.AvgPossession += float64(report.Stats.Away.Possession)
			home.AvgPassAcc += float64(report.Stats.Home.PassAccuracy)
			away.AvgPassAcc += float64(report.Stats.Away.PassAccuracy)
			home.AvgShots += float64(report.Stats.Home.Shots)
			away.AvgShots += float64(report.Stats.Away.Shots)
			home.AvgXG += report.Stats.Home.XG
			away.AvgXG += report.Stats.Away.XG
			home.AvgXGAgainst += report.Stats.Away.XG
			away.AvgXGAgainst += report.Stats.Home.XG
			th := territory[f.HomeID]
			th[0] += report.Heatmap.HomeZones.Defensive
			th[1] += report.Heatmap.HomeZones.Midfield
			th[2] += report.Heatmap.HomeZones.Attacking
			territory[f.HomeID] = th
			ta := territory[f.AwayID]
			ta[0] += report.Heatmap.AwayZones.Defensive
			ta[1] += report.Heatmap.AwayZones.Midfield
			ta[2] += report.Heatmap.AwayZones.Attacking
			territory[f.AwayID] = ta
		}
	}
	scanTeams(tm.Fixtures)
	scanTeams(tm.UCLFixtures)
	scanTeams(tm.SuperCupFixtures)
	if tm.World != nil {
		scanTeams(tm.World.Fixtures)
	}

	clubRows := make([]advancedClubRow, 0, len(clubAgg))
	for id, row := range clubAgg {
		if club := tm.Clubs[id]; club != nil {
			row.ClubName = club.ClubName
			row.ShortName = club.ShortName
		} else {
			row.ClubName, row.ShortName = id, id
		}
		if row.Matches > 0 {
			m := float64(row.Matches)
			row.AvgPossession = round2(row.AvgPossession / m)
			row.AvgPassAcc = round2(row.AvgPassAcc / m)
			row.AvgShots = round2(row.AvgShots / m)
			row.AvgXG = round2(row.AvgXG / m)
			row.AvgXGAgainst = round2(row.AvgXGAgainst / m)
			if t := territory[id]; t[0]+t[1]+t[2] > 0 {
				total := float64(t[0] + t[1] + t[2])
				row.TerritoryDef = round2(float64(t[0]) / total * 100)
				row.TerritoryMid = round2(float64(t[1]) / total * 100)
				row.TerritoryAtt = round2(float64(t[2]) / total * 100)
			}
		}
		clubRows = append(clubRows, *row)
	}
	sort.Slice(clubRows, func(i, j int) bool {
		if clubRows[i].AvgXG != clubRows[j].AvgXG {
			return clubRows[i].AvgXG > clubRows[j].AvgXG
		}
		return clubRows[i].ClubID < clubRows[j].ClubID
	})

	return map[string]interface{}{
		"season_name": tm.SeasonName,
		"players":     rows,
		"clubs":       clubRows,
		// Shot-level detail exists only inside the report retention window.
		"shot_detail_note": "Shot-zone splits cover fixtures inside the report retention window; goals, assists, and minutes are full-season ledgers.",
	}
}

// seasonMinutesLedger sums the per-competition minute ledger.
func seasonMinutesLedger(p *models.Player) int {
	total := 0
	for _, row := range p.CompetitionStats {
		if row != nil {
			total += row.Minutes
		}
	}
	return total
}

func (tm *TournamentManager) clubTrendRow(agg map[string]*advancedClubRow, clubID string) *advancedClubRow {
	row := agg[clubID]
	if row == nil {
		row = &advancedClubRow{ClubID: clubID}
		agg[clubID] = row
	}
	return row
}
