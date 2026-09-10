package tournament

import (
	"fmt"

	"football_sim/pkg/matchreport"
	"football_sim/pkg/models"
)

// CupTie is a two-legged (UCL KO) or single-leg (Super Cup) knockout pairing.
type CupTie struct {
	HomeID    string `json:"home_id"`
	AwayID    string `json:"away_id"`
	WinnerID  string `json:"winner_id,omitempty"`
	Leg1      []int  `json:"leg1,omitempty"`
	Leg2      []int  `json:"leg2,omitempty"`
	DecidedBy string `json:"decided_by,omitempty"`
	Penalties []int  `json:"penalties,omitempty"`
}

func emptyTie(home, away *models.Club) CupTie {
	t := CupTie{}
	if home != nil {
		t.HomeID = home.ClubID
	}
	if away != nil {
		t.AwayID = away.ClubID
	}
	return t
}

func makeGroupRounds(teams []*models.Club) [][][2]*models.Club {
	n := len(teams)
	if n < 2 {
		return nil
	}
	cur := append([]*models.Club(nil), teams...)
	var rounds [][][2]*models.Club
	for r := 0; r < n-1; r++ {
		var fx [][2]*models.Club
		for i := 0; i < n/2; i++ {
			fx = append(fx, [2]*models.Club{cur[i], cur[n-1-i]})
		}
		rounds = append(rounds, fx)
		rotated := make([]*models.Club, n)
		rotated[0] = cur[0]
		rotated[1] = cur[n-1]
		copy(rotated[2:], cur[1:n-1])
		cur = rotated
	}
	return rounds
}

func (tm *TournamentManager) initCups() {
	n := len(tm.ClubsList)
	if n < 2 {
		return
	}
	split := n / 2
	tm.UCLGroupA = append([]*models.Club(nil), tm.ClubsList[:split]...)
	tm.UCLGroupB = append([]*models.Club(nil), tm.ClubsList[split:]...)
	tm.UCLStage = "GROUP_STAGE"
	tm.UCLQuarterFinals = map[string]CupTie{}
	tm.UCLSemiFinals = map[string]CupTie{}
	tm.UCLFinal = CupTie{}
	tm.UCLChampionID = ""
	tm.initUCLRecords()
	tm.buildUCLGroupFixtures()
	ordered := append([]*models.Club(nil), tm.ClubsList...)
	// Seed Super Cup by current team rating, matching a fresh career draw.
	sortClubsByRating(ordered)
	tm.drawSuperCup(ordered)
}

func (tm *TournamentManager) initUCLRecords() {
	tm.UCLRecords = make(map[string]*models.CompetitionRecord, len(tm.ClubsList))
	for _, c := range tm.ClubsList {
		tm.UCLRecords[c.ClubID] = &models.CompetitionRecord{Form: []string{}}
	}
}

func (tm *TournamentManager) buildUCLGroupFixtures() {
	tm.UCLFixtures = nil
	roundsA := makeGroupRounds(tm.UCLGroupA)
	roundsB := makeGroupRounds(tm.UCLGroupB)
	for rIdx, mw := range UCLGroupWeeks {
		if rIdx >= len(roundsA) || rIdx >= len(roundsB) {
			break
		}
		for _, pair := range roundsA[rIdx] {
			tm.UCLFixtures = append(tm.UCLFixtures, tm.blankFixture(
				fmt.Sprintf("UCL-MW%d-%s-%s", mw, pair[0].ClubID, pair[1].ClubID),
				mw, "ucl", "Group A", pair[0], pair[1], 0, "",
			))
		}
		for _, pair := range roundsB[rIdx] {
			tm.UCLFixtures = append(tm.UCLFixtures, tm.blankFixture(
				fmt.Sprintf("UCL-MW%d-%s-%s", mw, pair[0].ClubID, pair[1].ClubID),
				mw, "ucl", "Group B", pair[0], pair[1], 0, "",
			))
		}
	}
}

func (tm *TournamentManager) blankFixture(id string, mw int, comp, stage string, home, away *models.Club, leg int, tieID string) Fixture {
	derby := ""
	if home != nil && away != nil {
		derby = GetDerbyName(home.ClubID, away.ClubID)
	}
	heat := 0
	if derby != "" {
		heat = tm.derbyHeatUnlocked(derby)
	}
	homeID, awayID := "", ""
	if home != nil {
		homeID = home.ClubID
	}
	if away != nil {
		awayID = away.ClubID
	}
	return Fixture{
		FixtureID:       id,
		Matchweek:       mw,
		Competition:     comp,
		Stage:           stage,
		HomeID:          homeID,
		AwayID:          awayID,
		Home:            home,
		Away:            away,
		Status:          "scheduled",
		Weather:         tm.weatherUnlocked(mw),
		DerbyName:       derby,
		DerbyHeat:       heat,
		IsHighHeatDerby: derby != "" && heat > 70,
		Leg:             leg,
		TieID:           tieID,
		Referee:         matchreport.PickRefereeName(tm.RNG, "balanced"),
	}
}

func (tm *TournamentManager) makeUCLLeg(tieID, stage string, leg, matchweek int, home, away *models.Club) {
	fid := fmt.Sprintf("UCL-MW%d-%s-L%d", matchweek, tieID, leg)
	tm.UCLFixtures = append(tm.UCLFixtures, tm.blankFixture(fid, matchweek, "ucl", stage, home, away, leg, tieID))
}

func (tm *TournamentManager) makeSCFixture(tieID, stage string, matchweek int, home, away *models.Club) {
	fid := fmt.Sprintf("SC-MW%d-%s", matchweek, tieID)
	tm.SuperCupFixtures = append(tm.SuperCupFixtures, tm.blankFixture(fid, matchweek, "super-cup", stage, home, away, 1, tieID))
}

func (tm *TournamentManager) drawSuperCup(ordered []*models.Club) {
	seeds := make([]*models.Club, 0, 12)
	seen := map[string]bool{}
	for _, c := range ordered {
		if c == nil || seen[c.ClubID] {
			continue
		}
		seen[c.ClubID] = true
		seeds = append(seeds, c)
		if len(seeds) == 12 {
			break
		}
	}
	for _, c := range tm.ClubsList {
		if len(seeds) >= 12 {
			break
		}
		if !seen[c.ClubID] {
			seeds = append(seeds, c)
		}
	}
	if len(seeds) < 8 {
		return
	}
	tm.SuperCupFixtures = nil
	tm.SuperCupPlayIn = map[string]CupTie{}
	tm.SuperCupQuarterFinals = map[string]CupTie{}
	tm.SuperCupSemiFinals = map[string]CupTie{}
	tm.SuperCupFinal = CupTie{}
	tm.SuperCupChampionID = ""
	tm.SuperCupByes = seeds[:4]
	tm.SuperCupStage = "PLAY_IN"
	pairs := [][2]int{{5, 12}, {6, 11}, {7, 10}, {8, 9}}
	week := SuperCupWeeks["play_in"]
	for i, pair := range pairs {
		hi, lo := pair[0]-1, pair[1]-1
		if hi >= len(seeds) || lo >= len(seeds) {
			continue
		}
		key := fmt.Sprintf("sc_pi_%d", i+1)
		tm.SuperCupPlayIn[key] = emptyTie(seeds[hi], seeds[lo])
		tm.makeSCFixture(key, "Play-in", week, seeds[hi], seeds[lo])
	}
}

func sortClubsByRating(clubs []*models.Club) {
	for i := 0; i < len(clubs); i++ {
		for j := i + 1; j < len(clubs); j++ {
			if clubs[j].OverallTeamRating > clubs[i].OverallTeamRating {
				clubs[i], clubs[j] = clubs[j], clubs[i]
			}
		}
	}
}

func (tm *TournamentManager) reseedUCLGroups(ordered []*models.Club) {
	var a, b []*models.Club
	for i, club := range ordered {
		if i%4 == 0 || i%4 == 3 {
			a = append(a, club)
		} else {
			b = append(b, club)
		}
	}
	if len(a) < 6 {
		a = append(a, ordered[len(a)+len(b):]...)
	}
	if len(a) > 6 {
		a = a[:6]
	}
	if len(b) > 6 {
		b = b[:6]
	}
	if len(b) == 0 && len(ordered) >= 12 {
		b = ordered[6:12]
	}
	tm.UCLGroupA = a
	tm.UCLGroupB = b
}

func (tm *TournamentManager) uclStandingsUnlocked() (groupA, groupB []*models.Club) {
	sortUCL := func(clubs []*models.Club) []*models.Club {
		out := append([]*models.Club(nil), clubs...)
		for i := 0; i < len(out); i++ {
			for j := i + 1; j < len(out); j++ {
				ri := tm.UCLRecords[out[i].ClubID]
				rj := tm.UCLRecords[out[j].ClubID]
				if ri == nil {
					ri = &models.CompetitionRecord{}
				}
				if rj == nil {
					rj = &models.CompetitionRecord{}
				}
				if rj.Points != ri.Points {
					if rj.Points > ri.Points {
						out[i], out[j] = out[j], out[i]
					}
					continue
				}
				if rj.GoalDifference != ri.GoalDifference {
					if rj.GoalDifference > ri.GoalDifference {
						out[i], out[j] = out[j], out[i]
					}
					continue
				}
				if rj.GoalsFor != ri.GoalsFor {
					if rj.GoalsFor > ri.GoalsFor {
						out[i], out[j] = out[j], out[i]
					}
					continue
				}
				if out[j].OverallTeamRating > out[i].OverallTeamRating {
					out[i], out[j] = out[j], out[i]
				}
			}
		}
		return out
	}
	return sortUCL(tm.UCLGroupA), sortUCL(tm.UCLGroupB)
}

// UCLGroupStatus reports qualified/eliminated/must_win/live for a club.
func (tm *TournamentManager) UCLGroupStatus(clubID string) string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.uclGroupStatus(clubID)
}

func (tm *TournamentManager) uclGroupStatus(clubID string) string {
	if tm.UCLChampionID != "" {
		if tm.UCLChampionID == clubID {
			return "qualified"
		}
		return "eliminated"
	}
	a, b := tm.uclStandingsUnlocked()
	in := func(list []*models.Club) int {
		for i, c := range list {
			if c.ClubID == clubID {
				return i
			}
		}
		return -1
	}
	idx := in(a)
	group := a
	if idx < 0 {
		idx = in(b)
		group = b
	}
	if idx < 0 {
		return ""
	}
	if tm.UCLStage != "GROUP_STAGE" {
		if idx < 4 {
			return "qualified"
		}
		return "eliminated"
	}
	played := 0
	if rec := tm.UCLRecords[clubID]; rec != nil {
		played = rec.Played
	}
	if played >= 5 {
		if idx < 4 {
			return "qualified"
		}
		return "eliminated"
	}
	if idx >= 4 && len(group) > 4 {
		return "must_win"
	}
	return "live"
}
