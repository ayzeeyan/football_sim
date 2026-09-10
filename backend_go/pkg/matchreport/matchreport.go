package matchreport

import (
	"fmt"
	"math"
	"math/rand"
	"sort"

	"football_sim/pkg/models"
)

type MiniPlayer struct {
	PlayerID string `json:"player_id"`
	FullName string `json:"full_name"`
	Position string `json:"position"`
	OVR      int    `json:"ovr"`
	Age      int    `json:"age"`
	Category string `json:"category"`
	IsWK     bool   `json:"is_wk"`
}

func ToMiniPlayer(p *models.Player) MiniPlayer {
	if p == nil {
		return MiniPlayer{}
	}
	return MiniPlayer{
		PlayerID: p.PlayerID,
		FullName: p.FullName,
		Position: p.Position,
		OVR:      p.OVR,
		Age:      p.Age,
		Category: p.Category,
		IsWK:     p.UniverseWonderkid,
	}
}

type MatchEventItem struct {
	Seq         int         `json:"seq"`
	Minute      int         `json:"minute"`
	Type        string      `json:"type"` // goal, penalty, corner_goal, free_kick_goal, own_goal, penalty_miss, yellow, red, sub, var_review
	Side        string      `json:"side"` // home, away
	Beneficiary string      `json:"beneficiary,omitempty"`
	Scorer      *MiniPlayer `json:"scorer,omitempty"`
	Assister    *MiniPlayer `json:"assister,omitempty"`
	Player      *MiniPlayer `json:"player,omitempty"`     // booked player
	PlayerOut   *MiniPlayer `json:"player_out,omitempty"` // subbed out
	PlayerIn    *MiniPlayer `json:"player_in,omitempty"`  // subbed in
	Display     string      `json:"display"`
	Disallowed  bool        `json:"disallowed,omitempty"`
	Outcome     string      `json:"outcome,omitempty"`  // for var_review: goal_stands, goal_disallowed
	Reason      string      `json:"reason,omitempty"`   // for var_review: offside, handball, check complete
	Decision    string      `json:"decision,omitempty"` // mirrors outcome
	SentOff     bool        `json:"sent_off,omitempty"` // red-card dismissal flag
	HomeScore   int         `json:"home_score"`
	AwayScore   int         `json:"away_score"`
	Detail      string      `json:"detail,omitempty"`
	Period      string      `json:"period,omitempty"` // et for extra-time goals
}

type MatchPlayerRow struct {
	PlayerID     string   `json:"player_id"`
	FullName     string   `json:"full_name"`
	Position     string   `json:"position"`
	OVR          int      `json:"ovr"`
	Age          int      `json:"age"`
	Category     string   `json:"category"`
	IsWK         bool     `json:"is_wk"`
	Rating       *float64 `json:"rating"`
	MatchGoals   int      `json:"match_goals"`
	MatchAssists int      `json:"match_assists"`
	MatchOG      int      `json:"match_og"`
	MatchPenMiss int      `json:"match_pen_miss"`
	Card         *string  `json:"card"`
	Minutes      int      `json:"minutes"`
	OnMinute     *int     `json:"on_minute"`
	OffMinute    *int     `json:"off_minute"`
	Starter      bool     `json:"starter"`
	Played       bool     `json:"played"`
	Side         string   `json:"side,omitempty"` // set on MOTM rows only
}

type TeamStats struct {
	Possession   int     `json:"possession"`
	Shots        int     `json:"shots"`
	ShotsOn      int     `json:"shots_on_target"`
	XG           float64 `json:"xg"`
	Passes       int     `json:"passes"`
	PassAccuracy int     `json:"pass_accuracy"`
	Corners      int     `json:"corners"`
	Fouls        int     `json:"fouls"`
	YellowCards  int     `json:"yellow_cards"`
	RedCards     int     `json:"red_cards"`
	Saves        int     `json:"saves"`
}

type MatchStats struct {
	Home TeamStats `json:"home"`
	Away TeamStats `json:"away"`
}

type ShotShooter struct {
	FullName string `json:"full_name"`
	Position string `json:"position"`
	OVR      int    `json:"ovr"`
	PlayerID string `json:"player_id,omitempty"`
}

type ShotMapItem struct {
	Minute      int         `json:"minute"`
	Team        string      `json:"team"` // home, away
	Shooter     ShotShooter `json:"shooter"`
	X           float64     `json:"x"`
	Y           float64     `json:"y"`
	XG          float64     `json:"xg"`
	Outcome     string      `json:"outcome"` // goal, save, blocked, miss
	IsWonderkid bool        `json:"is_wonderkid"`
}

type XGFlowPoint struct {
	Minute int     `json:"minute"`
	HomeXG float64 `json:"home_xg"`
	AwayXG float64 `json:"away_xg"`
}

type ShotMapData struct {
	Shots       []ShotMapItem `json:"shots"`
	XGFlow      []XGFlowPoint `json:"xg_flow"`
	TotalHomeXG float64       `json:"total_home_xg"`
	TotalAwayXG float64       `json:"total_away_xg"`
}

type ZoneSplit struct {
	Defensive int `json:"defensive"`
	Midfield  int `json:"midfield"`
	Attacking int `json:"attacking"`
	Left      int `json:"left"`
	Center    int `json:"center"`
	Right     int `json:"right"`
}

type TouchHeatmapData struct {
	HomePoints [][]float64 `json:"home_points"`
	AwayPoints [][]float64 `json:"away_points"`
	HomeZones  ZoneSplit   `json:"home_zones"`
	AwayZones  ZoneSplit   `json:"away_zones"`
}

type PressConferenceData struct {
	Headline    string `json:"headline"`
	HomeQuote   string `json:"home_quote"`
	AwayQuote   string `json:"away_quote"`
	HomeManager string `json:"home_manager"`
	AwayManager string `json:"away_manager"`
	Narrative   string `json:"narrative"`
}

type MatchReport struct {
	Method          string              `json:"method"`
	HomeGoals       int                 `json:"home_goals"`
	AwayGoals       int                 `json:"away_goals"`
	Events          []MatchEventItem    `json:"events"`
	HomeXI          []MatchPlayerRow    `json:"home_xi"`
	AwayXI          []MatchPlayerRow    `json:"away_xi"`
	HomeBench       []MatchPlayerRow    `json:"home_bench"`
	AwayBench       []MatchPlayerRow    `json:"away_bench"`
	Stats           MatchStats          `json:"stats"`
	ShotMap         ShotMapData         `json:"shot_map"`
	Heatmap         TouchHeatmapData    `json:"heatmap"`
	PressConference PressConferenceData `json:"press_conference"`
	MOTM            *MatchPlayerRow     `json:"motm,omitempty"`
	HTHome          int                 `json:"ht_home"`
	HTAway          int                 `json:"ht_away"`
	Attendance      int                 `json:"attendance"`
	Referee         string              `json:"referee"`
	Weather         string              `json:"weather"`
	DecidedBy       *string             `json:"decided_by,omitempty"`
	Penalties       interface{}         `json:"penalties,omitempty"`
}

// AppearanceWindow returns minutes played, on-minute, and off-minute for a player.
func AppearanceWindow(pid string, events []MatchEventItem, starter bool, extraTime bool) (int, *int, *int) {
	end := 90
	if extraTime {
		end = 120
	}
	on := 0
	off := end
	played := starter

	for _, e := range events {
		if e.Type == "sub" {
			if e.PlayerIn != nil && e.PlayerIn.PlayerID == pid {
				on = e.Minute
				played = true
			}
			if e.PlayerOut != nil && e.PlayerOut.PlayerID == pid {
				if e.Minute < off {
					off = e.Minute
				}
			}
		} else if e.Type == "red" && e.Player != nil && e.Player.PlayerID == pid {
			if e.Minute < off {
				off = e.Minute
			}
		}
	}

	if !played {
		return 0, nil, nil
	}
	mins := off - on
	if mins < 1 {
		mins = 1
	}
	var onMin *int
	if !starter {
		m := on
		onMin = &m
	}
	var offMin *int
	if off < end {
		m := off
		offMin = &m
	}
	return mins, onMin, offMin
}

// RateXI computes 5.0–10.0 player ratings based on match events and results.
func RateXI(
	xi []*models.Player,
	events []MatchEventItem,
	side string,
	conceded int,
	won bool,
	starter bool,
	rng *rand.Rand,
) []MatchPlayerRow {
	if rng == nil {
		rng = rand.New(rand.NewSource(rand.Int63()))
	}

	type tally struct {
		g    int
		a    int
		og   int
		pm   int
		card string
	}
	tallies := make(map[string]*tally)

	for _, e := range events {
		if e.Side != side {
			continue
		}
		if (e.Type == "goal" || e.Type == "penalty" || e.Type == "corner_goal" || e.Type == "free_kick_goal") && !e.Disallowed {
			if e.Scorer != nil {
				t, ok := tallies[e.Scorer.PlayerID]
				if !ok {
					t = &tally{}
					tallies[e.Scorer.PlayerID] = t
				}
				t.g++
			}
			if e.Assister != nil {
				t, ok := tallies[e.Assister.PlayerID]
				if !ok {
					t = &tally{}
					tallies[e.Assister.PlayerID] = t
				}
				t.a++
			}
		} else if e.Type == "own_goal" {
			if e.Scorer != nil {
				t, ok := tallies[e.Scorer.PlayerID]
				if !ok {
					t = &tally{}
					tallies[e.Scorer.PlayerID] = t
				}
				t.og++
			}
		} else if e.Type == "penalty_miss" {
			if e.Scorer != nil {
				t, ok := tallies[e.Scorer.PlayerID]
				if !ok {
					t = &tally{}
					tallies[e.Scorer.PlayerID] = t
				}
				t.pm++
			}
		} else if e.Type == "yellow" || e.Type == "red" {
			if e.Player != nil {
				t, ok := tallies[e.Player.PlayerID]
				if !ok {
					t = &tally{}
					tallies[e.Player.PlayerID] = t
				}
				if e.Type == "red" || t.card == "" {
					t.card = e.Type
				}
			}
		}
	}

	extraTime := false
	for _, e := range events {
		if e.Minute > 90 {
			extraTime = true
			break
		}
	}

	rows := make([]MatchPlayerRow, 0, len(xi))
	for _, p := range xi {
		t := tallies[p.PlayerID]
		if t == nil {
			t = &tally{}
		}

		mins, onMin, offMin := AppearanceWindow(p.PlayerID, events, starter, extraTime)
		if mins <= 0 {
			rows = append(rows, MatchPlayerRow{
				PlayerID: p.PlayerID,
				FullName: p.FullName,
				Position: p.Position,
				OVR:      p.OVR,
				Age:      p.Age,
				Category: p.Category,
				IsWK:     p.UniverseWonderkid,
				Rating:   nil,
				Starter:  starter,
				Played:   false,
			})
			continue
		}

		r := 6.4 + float64(t.g)*0.9 + float64(t.a)*0.5 - float64(t.og)*1.2 - float64(t.pm)*0.8
		if t.card == "yellow" {
			r -= 0.4
		} else if t.card == "red" {
			r -= 1.6
		}
		if (p.Category == "GK" || p.Category == "DEF") && conceded == 0 {
			r += 0.5
		}
		if won {
			r += 0.2
		}
		r += (rng.Float64()*0.55 - 0.25)

		if mins < 20 {
			r = 6.15 + float64(t.g)*0.9 + float64(t.a)*0.5 - float64(t.og)*1.0 - float64(t.pm)*0.6
			if t.card == "yellow" {
				r -= 0.3
			} else if t.card == "red" {
				r -= 1.4
			}
			r += (rng.Float64()*0.45 - 0.2)
		}

		// Clamp between 5.0 and 10.0
		r = math.Max(5.0, math.Min(10.0, math.Round(r*10)/10))
		var cardPtr *string
		if t.card != "" {
			c := t.card
			cardPtr = &c
		}

		rows = append(rows, MatchPlayerRow{
			PlayerID:     p.PlayerID,
			FullName:     p.FullName,
			Position:     p.Position,
			OVR:          p.OVR,
			Age:          p.Age,
			Category:     p.Category,
			IsWK:         p.UniverseWonderkid,
			Rating:       &r,
			MatchGoals:   t.g,
			MatchAssists: t.a,
			MatchOG:      t.og,
			MatchPenMiss: t.pm,
			Card:         cardPtr,
			Minutes:      mins,
			OnMinute:     onMin,
			OffMinute:    offMin,
			Starter:      starter,
			Played:       true,
		})
	}
	return rows
}

// GenerateShotMap reconstructs full shot map with xG for the match.
func GenerateShotMap(
	homeClub *models.Club,
	awayClub *models.Club,
	events []MatchEventItem,
	homeShotsTotal int,
	awayShotsTotal int,
	homeShotsOn int,
	awayShotsOn int,
	rng *rand.Rand,
	liveShots []ShotMapItem,
) ShotMapData {
	if rng == nil {
		rng = rand.New(rand.NewSource(rand.Int63()))
	}
	var shots []ShotMapItem

	if len(liveShots) > 0 {
		shots = append([]ShotMapItem(nil), liveShots...)
	} else {
		shots = reconstructShots(homeClub, awayClub, events, homeShotsTotal, awayShotsTotal, homeShotsOn, awayShotsOn, rng)
	}

	sort.Slice(shots, func(i, j int) bool {
		return shots[i].Minute < shots[j].Minute
	})

	flow := []XGFlowPoint{{Minute: 0, HomeXG: 0.0, AwayXG: 0.0}}
	var cumH, cumA float64
	for _, s := range shots {
		if s.Team == "home" {
			cumH = math.Round((cumH+s.XG)*100) / 100
		} else {
			cumA = math.Round((cumA+s.XG)*100) / 100
		}
		flow = append(flow, XGFlowPoint{
			Minute: s.Minute,
			HomeXG: cumH,
			AwayXG: cumA,
		})
	}
	// Terminal pad so charts always run to the final whistle.
	lastMin := 90
	for _, s := range shots {
		if s.Minute > lastMin {
			lastMin = s.Minute
		}
	}
	if len(flow) == 0 || flow[len(flow)-1].Minute < lastMin {
		flow = append(flow, XGFlowPoint{Minute: lastMin, HomeXG: cumH, AwayXG: cumA})
	}

	return ShotMapData{
		Shots:       shots,
		XGFlow:      flow,
		TotalHomeXG: cumH,
		TotalAwayXG: cumA,
	}
}

// reconstructShots rebuilds goal events plus filler shots from totals.
func reconstructShots(
	homeClub *models.Club,
	awayClub *models.Club,
	events []MatchEventItem,
	homeShotsTotal int,
	awayShotsTotal int,
	homeShotsOn int,
	awayShotsOn int,
	rng *rand.Rand,
) []ShotMapItem {
	var shots []ShotMapItem

	for _, e := range events {
		if (e.Type == "goal" || e.Type == "penalty" || e.Type == "corner_goal" || e.Type == "free_kick_goal") && !e.Disallowed {
			isHome := e.Side == "home"
			// Fixed near-post target with stoppage-agnostic placement;
			// only the xG value differs by chance type.
			var x, y, xgVal float64
			if isHome {
				x = 0.93
			} else {
				x = 0.07
			}
			y = 0.44 + rng.Float64()*0.12

			if e.Type == "penalty" {
				xgVal = 0.76
			} else {
				xgVal = 0.32 + rng.Float64()*0.30
			}

			shooter := ShotShooter{
				FullName: "Striker",
				Position: "FWD",
				OVR:      80,
			}
			isWk := false
			if e.Scorer != nil {
				shooter.FullName = e.Scorer.FullName
				shooter.Position = e.Scorer.Position
				shooter.OVR = e.Scorer.OVR
				shooter.PlayerID = e.Scorer.PlayerID
				isWk = e.Scorer.IsWK
			}

			shots = append(shots, ShotMapItem{
				Minute:      e.Minute,
				Team:        e.Side,
				Shooter:     shooter,
				X:           math.Round(x*100) / 100,
				Y:           math.Round(y*100) / 100,
				XG:          math.Round(xgVal*100) / 100,
				Outcome:     "goal",
				IsWonderkid: isWk,
			})
		}
	}

	// Home remaining shots
	homeGoalsCount := 0
	for _, s := range shots {
		if s.Team == "home" && s.Outcome == "goal" {
			homeGoalsCount++
		}
	}
	homeRem := homeShotsTotal - homeGoalsCount
	if homeRem < 0 {
		homeRem = 0
	}
	for i := 0; i < homeRem; i++ {
		m := 4 + rng.Intn(87)
		isOn := rng.Float64() < (float64(homeShotsOn) / math.Max(1.0, float64(homeShotsTotal)))
		var xgVal float64
		if isOn {
			xgVal = 0.14 + rng.Float64()*0.22
		} else {
			xgVal = 0.04 + rng.Float64()*0.12
		}
		dist := 0.74 + rng.Float64()*0.20
		outcome := "miss"
		if isOn {
			outcome = "save"
		}
		shots = append(shots, ShotMapItem{
			Minute: m,
			Team:   "home",
			Shooter: ShotShooter{
				FullName: fmt.Sprintf("%s Player", homeClub.ShortName),
				Position: "FWD",
				OVR:      homeClub.OverallTeamRating,
			},
			X:           math.Round(dist*100) / 100,
			Y:           math.Round((0.28+rng.Float64()*0.44)*100) / 100,
			XG:          math.Round(xgVal*100) / 100,
			Outcome:     outcome,
			IsWonderkid: false,
		})
	}

	// Away remaining shots
	awayGoalsCount := 0
	for _, s := range shots {
		if s.Team == "away" && s.Outcome == "goal" {
			awayGoalsCount++
		}
	}
	awayRem := awayShotsTotal - awayGoalsCount
	if awayRem < 0 {
		awayRem = 0
	}
	for i := 0; i < awayRem; i++ {
		m := 4 + rng.Intn(87)
		isOn := rng.Float64() < (float64(awayShotsOn) / math.Max(1.0, float64(awayShotsTotal)))
		var xgVal float64
		if isOn {
			xgVal = 0.14 + rng.Float64()*0.22
		} else {
			xgVal = 0.04 + rng.Float64()*0.12
		}
		dist := 0.06 + rng.Float64()*0.20
		outcome := "miss"
		if isOn {
			outcome = "save"
		}
		shots = append(shots, ShotMapItem{
			Minute: m,
			Team:   "away",
			Shooter: ShotShooter{
				FullName: fmt.Sprintf("%s Player", awayClub.ShortName),
				Position: "FWD",
				OVR:      awayClub.OverallTeamRating,
			},
			X:           math.Round(dist*100) / 100,
			Y:           math.Round((0.28+rng.Float64()*0.44)*100) / 100,
			XG:          math.Round(xgVal*100) / 100,
			Outcome:     outcome,
			IsWonderkid: false,
		})
	}

	return shots
}

// GenerateTouchHeatmap generates 2D territorial density points and zone splits.
// When live touches are supplied with at least 8 home samples, they are
// passed through (capped at 150 per side) instead of synthesizing.
func GenerateTouchHeatmap(homeClub *models.Club, awayClub *models.Club, homePoss int, rng *rand.Rand, liveTouches map[string][][2]float64) TouchHeatmapData {
	if rng == nil {
		rng = rand.New(rand.NewSource(rand.Int63()))
	}
	points := map[string][][]float64{"home": {}, "away": {}}

	if len(liveTouches["home"]) >= 8 {
		home := liveTouches["home"]
		if len(home) > 150 {
			home = home[:150]
		}
		for _, t := range home {
			points["home"] = append(points["home"], []float64{round2clip(t[0]), round2clip(t[1]), 1.0})
		}
		away := liveTouches["away"]
		if len(away) > 150 {
			away = away[:150]
		}
		for _, t := range away {
			points["away"] = append(points["away"], []float64{round2clip(t[0]), round2clip(t[1]), 1.0})
		}
	} else {
		hShare := math.Max(0.25, math.Min(0.75, float64(homePoss)/100.0))
		nHome := int(120 * hShare)
		nAway := 120 - nHome

		// Home team (attacks left to right, 0.0 to 1.0)
		for i := 0; i < nHome; i++ {
			x := Beta(rng, 2.8, 2.2)
			y := math.Max(0.08, math.Min(0.92, 0.5+rng.NormFloat64()*0.22))
			points["home"] = append(points["home"], []float64{math.Round(x*100) / 100, math.Round(y*100) / 100, 1.0})
		}

		// Away team (attacks right to left, 1.0 to 0.0)
		for i := 0; i < nAway; i++ {
			x := 1.0 - Beta(rng, 2.8, 2.2)
			y := math.Max(0.08, math.Min(0.92, 0.5+rng.NormFloat64()*0.22))
			points["away"] = append(points["away"], []float64{math.Round(x*100) / 100, math.Round(y*100) / 100, 1.0})
		}
	}

	calcZones := func(pts [][]float64) ZoneSplit {
		var def, mid, att, left, cen, rt int
		for _, p := range pts {
			if p[0] < 0.35 {
				def++
			} else if p[0] <= 0.65 {
				mid++
			} else {
				att++
			}

			if p[1] < 0.35 {
				left++
			} else if p[1] <= 0.65 {
				cen++
			} else {
				rt++
			}
		}
		tot := float64(len(pts))
		return ZoneSplit{
			Defensive: int(math.Round((float64(def) / tot) * 100)),
			Midfield:  int(math.Round((float64(mid) / tot) * 100)),
			Attacking: int(math.Round((float64(att) / tot) * 100)),
			Left:      int(math.Round((float64(left) / tot) * 100)),
			Center:    int(math.Round((float64(cen) / tot) * 100)),
			Right:     int(math.Round((float64(rt) / tot) * 100)),
		}
	}

	return TouchHeatmapData{
		HomePoints: points["home"],
		AwayPoints: points["away"],
		HomeZones:  calcZones(points["home"]),
		AwayZones:  calcZones(points["away"]),
	}
}

func round2clip(x float64) float64 {
	return math.Round(x*100) / 100
}

// OnFieldPlayers replays substitutions and dismissals to list who is on the
// pitch for a side at a given minute (Python: on_field_players).
func OnFieldPlayers(payload InstantPayload, side string, minute int) []*models.Player {
	var starters, bench []*models.Player
	if side == "home" {
		starters = append([]*models.Player(nil), payload.HomeXI...)
		bench = append([]*models.Player(nil), payload.HomeBench...)
	} else {
		starters = append([]*models.Player(nil), payload.AwayXI...)
		bench = append([]*models.Player(nil), payload.AwayBench...)
	}
	roster := make(map[string]*models.Player, len(starters)+len(bench))
	for _, p := range starters {
		roster[p.PlayerID] = p
	}
	for _, p := range bench {
		roster[p.PlayerID] = p
	}
	fieldIDs := make([]string, 0, len(starters))
	for _, p := range starters {
		fieldIDs = append(fieldIDs, p.PlayerID)
	}
	ordered := append([]MatchEventItem(nil), payload.Events...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Minute == ordered[j].Minute {
			return ordered[i].Seq < ordered[j].Seq
		}
		return ordered[i].Minute < ordered[j].Minute
	})
	for _, e := range ordered {
		if e.Minute > minute {
			break
		}
		if e.Side != side {
			continue
		}
		switch e.Type {
		case "sub":
			if e.PlayerOut == nil || e.PlayerIn == nil {
				continue
			}
			for k, id := range fieldIDs {
				if id == e.PlayerOut.PlayerID {
					fieldIDs[k] = e.PlayerIn.PlayerID
				}
			}
		case "red":
			if e.Player == nil {
				continue
			}
			kept := fieldIDs[:0]
			for _, id := range fieldIDs {
				if id != e.Player.PlayerID {
					kept = append(kept, id)
				}
			}
			fieldIDs = kept
		}
	}
	var on []*models.Player
	for _, id := range fieldIDs {
		if p, ok := roster[id]; ok {
			on = append(on, p)
		}
	}
	return on
}

// GeneratePressConference builds post-match managerial quotes and narrative headlines.
func GeneratePressConference(
	homeClub *models.Club,
	awayClub *models.Club,
	homeGoals int,
	awayGoals int,
	events []MatchEventItem,
	homeManager string,
	awayManager string,
) PressConferenceData {
	if homeManager == "" {
		homeManager = homeClub.ShortName + " Manager"
	}
	if awayManager == "" {
		awayManager = awayClub.ShortName + " Manager"
	}
	var wkScorer string
	for _, e := range events {
		if (e.Type == "goal" || e.Type == "penalty" || e.Type == "corner_goal" || e.Type == "free_kick_goal") && !e.Disallowed {
			if e.Scorer != nil && e.Scorer.IsWK {
				wkScorer = e.Scorer.FullName
				break
			}
		}
	}

	var headline, homeQuote, awayQuote, narrative string
	if homeGoals > awayGoals {
		headline = fmt.Sprintf("%s overpower %s in dominant display", homeClub.ClubName, awayClub.ShortName)
		homeQuote = "The boys executed our tactical plan with precision today. We controlled the tempo and took our chances ruthlessly."
		if wkScorer != "" {
			homeQuote += fmt.Sprintf(" %s showed maturity beyond his years — a special prodigy.", wkScorer)
		}
		awayQuote = fmt.Sprintf("Credit to %s, they punished our defensive lapses. We will regroup and correct our spacing on the training pitch.", homeClub.ShortName)
		narrative = fmt.Sprintf("%s secured all three points with a focused display at %s, controlling the rhythm from kickoff.", homeClub.ClubName, homeClub.HomeStadium)
	} else if awayGoals > homeGoals {
		headline = fmt.Sprintf("%s claim crucial victory away at %s", awayClub.ClubName, homeClub.HomeStadium)
		awayQuote = "A tremendous collective effort away from home. We absorbed their pressure and struck on the counter when the spaces opened up."
		if wkScorer != "" {
			awayQuote += fmt.Sprintf(" %s's composure on the ball was the turning point tonight.", wkScorer)
		}
		homeQuote = "It's a bitter pill to swallow in front of our home supporters. We lacked sharpness in the final third and got caught in transition."
		narrative = fmt.Sprintf("%s silenced the home crowd at %s with a clinical counter-attacking masterclass.", awayClub.ClubName, homeClub.HomeStadium)
	} else {
		headline = fmt.Sprintf("%s and %s share spoils in tense tactical clash", homeClub.ShortName, awayClub.ShortName)
		homeQuote = "Both teams showed immense tactical discipline. A fair point, though we felt we created enough to nick all three."
		awayQuote = fmt.Sprintf("Coming away from %s with a point is a solid foundation. The team showed fighting character until the final whistle.", homeClub.HomeStadium)
		narrative = fmt.Sprintf("Neither %s nor %s could break the deadlock as both defensive blocks stood resolute.", homeClub.ClubName, awayClub.ClubName)
	}

	return PressConferenceData{
		Headline:    headline,
		HomeQuote:   homeQuote,
		AwayQuote:   awayQuote,
		HomeManager: homeManager,
		AwayManager: awayManager,
		Narrative:   narrative,
	}
}
