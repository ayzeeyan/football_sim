package tournament

import (
	"fmt"
	"sort"
	"strings"

	"football_sim/pkg/matchreport"
	"football_sim/pkg/models"
)

// EuropeanNight is the cup-night badge React reads on MatchCard.
func (tm *TournamentManager) EuropeanNight(f *Fixture) map[string]interface{} {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.europeanNightUnlocked(f)
}

func (tm *TournamentManager) europeanNightUnlocked(f *Fixture) map[string]interface{} {
	if f == nil || (f.Competition != "ucl" && f.Competition != "super-cup") {
		return nil
	}
	home := tm.Clubs[f.HomeID]
	away := tm.Clubs[f.AwayID]
	badge := "Super Cup"
	if f.Competition == "ucl" {
		badge = "Champions Cup"
	}
	var status interface{}
	if f.Competition == "ucl" && strings.HasPrefix(f.Stage, "Group") && home != nil {
		if st := tm.uclGroupStatus(home.ClubID); st != "" {
			status = st
		}
	}
	var leg interface{}
	if f.Leg > 0 {
		leg = f.Leg
	}
	return map[string]interface{}{
		"kind":         f.Competition,
		"badge":        badge,
		"story":        tm.kickoffNoteUnlocked(f, home, away),
		"group_status": status,
		"aggregate":    tm.leg2AggregateLabelUnlocked(f),
		"leg1_label":   tm.leg2FirstLegLabelUnlocked(f),
		"leg":          leg,
	}
}

// leg2FirstLegLabelUnlocked names the finished first leg for a Champions Cup
// leg-2 card ("1st leg: RMA 2–0 ARS"). Super Cup ties are one-night only.
func (tm *TournamentManager) leg2FirstLegLabelUnlocked(f *Fixture) interface{} {
	if f == nil || f.Competition != "ucl" || f.Leg != 2 {
		return nil
	}
	leg1 := tm.uclLeg(f.TieID, 1)
	if leg1 == nil || leg1.Status != "finished" || leg1.HomeGoals == nil || leg1.AwayGoals == nil {
		return nil
	}
	homeShort, awayShort := leg1.HomeID, leg1.AwayID
	if home := tm.Clubs[leg1.HomeID]; home != nil {
		homeShort = home.ShortName
	}
	if away := tm.Clubs[leg1.AwayID]; away != nil {
		awayShort = away.ShortName
	}
	return fmt.Sprintf("1st leg: %s %d–%d %s", homeShort, *leg1.HomeGoals, *leg1.AwayGoals, awayShort)
}

func (tm *TournamentManager) leg2AggregateLabelUnlocked(f *Fixture) interface{} {
	if f == nil || f.Competition != "ucl" || f.Leg != 2 {
		return nil
	}
	leg1 := tm.uclLeg(f.TieID, 1)
	if leg1 == nil || leg1.Status != "finished" || leg1.HomeGoals == nil || leg1.AwayGoals == nil {
		return nil
	}
	var ours, theirs int
	switch f.HomeID {
	case leg1.AwayID:
		ours, theirs = *leg1.AwayGoals, *leg1.HomeGoals
	case leg1.HomeID:
		ours, theirs = *leg1.HomeGoals, *leg1.AwayGoals
	default:
		return nil
	}
	if ours < theirs {
		return fmt.Sprintf("%d–%d down", theirs, ours)
	}
	if ours > theirs {
		return fmt.Sprintf("%d–%d up", ours, theirs)
	}
	return fmt.Sprintf("%d–%d level", ours, theirs)
}

// FixtureHeadToHead is the last four finished meetings for a slate card.
func (tm *TournamentManager) FixtureHeadToHead(f *Fixture) []map[string]interface{} {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.fixtureHeadToHeadUnlocked(f)
}

func (tm *TournamentManager) fixtureHeadToHeadUnlocked(current *Fixture) []map[string]interface{} {
	rows := make([]map[string]interface{}, 0)
	if current == nil || current.HomeID == "" || current.AwayID == "" {
		return rows
	}
	pair := func(homeID, awayID string) bool {
		return (homeID == current.HomeID && awayID == current.AwayID) || (homeID == current.AwayID && awayID == current.HomeID)
	}
	scan := func(pool []Fixture) {
		for i := range pool {
			fx := &pool[i]
			if fx.FixtureID == current.FixtureID || fx.Status != "finished" {
				continue
			}
			if !pair(fx.HomeID, fx.AwayID) {
				continue
			}
			rows = append(rows, map[string]interface{}{
				"id":          fx.FixtureID,
				"matchweek":   fx.Matchweek,
				"competition": fx.Competition,
				"home_id":     fx.HomeID,
				"away_id":     fx.AwayID,
				"home_goals":  fx.HomeGoals,
				"away_goals":  fx.AwayGoals,
				"stage":       fx.Stage,
			})
		}
	}
	scan(tm.Fixtures)
	scan(tm.UCLFixtures)
	sort.Slice(rows, func(i, j int) bool {
		mi, _ := rows[i]["matchweek"].(int)
		mj, _ := rows[j]["matchweek"].(int)
		if mi != mj {
			return mi > mj
		}
		idi, _ := rows[i]["id"].(string)
		idj, _ := rows[j]["id"].(string)
		return idi > idj
	})
	if len(rows) > 4 {
		rows = rows[:4]
	}
	return rows
}

// PlayerMatchLog is the recent rated appearances on the player sheet.
func (tm *TournamentManager) PlayerMatchLog(playerID string, limit int) []map[string]interface{} {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.playerMatchLogUnlocked(playerID, limit)
}

func (tm *TournamentManager) playerMatchLogUnlocked(playerID string, limit int) []map[string]interface{} {
	rows := make([]map[string]interface{}, 0)
	if playerID == "" {
		return rows
	}
	scan := func(pool []Fixture) {
		for i := range pool {
			f := &pool[i]
			if f.Status != "finished" || f.Report == nil || f.HomeGoals == nil || f.AwayGoals == nil {
				continue
			}
			home := tm.Clubs[f.HomeID]
			away := tm.Clubs[f.AwayID]
			if home == nil || away == nil {
				continue
			}
			motmID := ""
			if f.Report.MOTM != nil {
				motmID = f.Report.MOTM.PlayerID
			}
			hg, ag := *f.HomeGoals, *f.AwayGoals
			appendSide := func(side string, opp *models.Club, pack []matchreport.MatchPlayerRow) {
				for _, row := range pack {
					if row.PlayerID != playerID || !row.Played || row.Minutes <= 0 {
						continue
					}
					gf, ga := hg, ag
					if side != "home" {
						gf, ga = ag, hg
					}
					result := "D"
					if gf > ga {
						result = "W"
					} else if gf < ga {
						result = "L"
					}
					var rating interface{}
					if row.Rating != nil {
						rating = *row.Rating
					}
					rows = append(rows, map[string]interface{}{
						"fixture_id":  f.FixtureID,
						"matchweek":   f.Matchweek,
						"competition": f.Competition,
						"stage":       f.Stage,
						"opponent":    opp.ShortName,
						"opponent_id": opp.ClubID,
						"home":        side == "home",
						"rating":      rating,
						"goals":       row.MatchGoals,
						"assists":     row.MatchAssists,
						"minutes":     row.Minutes,
						"motm":        motmID == playerID,
						"result":      result,
						"score":       fmt.Sprintf("%d–%d", hg, ag),
					})
				}
			}
			homePack := append([]matchreport.MatchPlayerRow{}, f.Report.HomeXI...)
			homePack = append(homePack, f.Report.HomeBench...)
			awayPack := append([]matchreport.MatchPlayerRow{}, f.Report.AwayXI...)
			awayPack = append(awayPack, f.Report.AwayBench...)
			appendSide("home", away, homePack)
			appendSide("away", home, awayPack)
		}
	}
	scan(tm.Fixtures)
	scan(tm.UCLFixtures)
	scan(tm.SuperCupFixtures)
	sort.Slice(rows, func(i, j int) bool {
		mi, _ := rows[i]["matchweek"].(int)
		mj, _ := rows[j]["matchweek"].(int)
		if mi != mj {
			return mi > mj
		}
		idi, _ := rows[i]["fixture_id"].(string)
		idj, _ := rows[j]["fixture_id"].(string)
		return idi > idj
	})
	if limit > 0 && len(rows) > limit {
		rows = rows[:limit]
	}
	return rows
}
