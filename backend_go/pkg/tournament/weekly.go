package tournament

import (
	"fmt"

	"football_sim/pkg/growth"
	"football_sim/pkg/matchreport"
	"football_sim/pkg/models"
)

func (tm *TournamentManager) runWeeklyTicks(completedMW int) {
	if tm.GrowthEngine != nil { tm.GrowthEngine.ReplenishTrainingEnergy() }
	focuses := []string{"hypertrophy", "technical", "tactical"}
	focus := focuses[(completedMW-1)%3]
	var growthEvents []string
	for _, club := range tm.ClubsList {
		var prodigy *models.Player
		for _, p := range club.Squad {
			if p.UniverseWonderkid { prodigy = p; break }
		}
		if prodigy == nil || tm.GrowthEngine == nil { continue }
		growthEvents = append(growthEvents, tm.GrowthEngine.SimulatePubertyCycle(prodigy.PlayerID, completedMW)...)
		if prodigy.MentorName != "" && prodigy.MentorOVR > 0 {
			mName, mOVR, pers := prodigy.MentorName, prodigy.MentorOVR, prodigy.Personality
			growthEvents = append(growthEvents, tm.GrowthEngine.ApplyMentorshipTick(prodigy.PlayerID, prodigy.FullName, growth.MentorshipOptions{MentorName: &mName, MentorOVR: &mOVR, Personality: &pers})...)
		}
		if tm.GrowthEngine.RollAutonomousTraining(0.22) {
			res, err := tm.GrowthEngine.RunTrainingCycle(prodigy.PlayerID, focus, false)
			if err == nil && res["status"] != "error" { growthEvents = append(growthEvents, fmt.Sprintf("%s staff ran %s training for %s.", club.ShortName, focus, prodigy.FullName)) }
		}
		prodigy.OVR = tm.GrowthEngine.CalculateOVR(prodigy.PlayerID, prodigy.Category)
		if attrs, ok := tm.GrowthEngine.Attributes[prodigy.PlayerID]; ok && attrs != nil { prodigy.Composure = attrs.Composure }
	}
	if len(growthEvents) > 0 {
		tm.GrowthNotifications = append(growthEvents, tm.GrowthNotifications...)
		if len(tm.GrowthNotifications) > 8 { tm.GrowthNotifications = tm.GrowthNotifications[:8] }
	}
	if tm.TransferEngine != nil { tm.TransferEngine.RevertToBaselines() }
	tm.pickPlayerOfTheWeek(completedMW)
	tm.maybeCrownMonth(completedMW)
	tm.evaluateManagerTenureWithPatience(completedMW)
	tm.decayDerbyHeat(completedMW)
	tm.maybeExamWeekInbox(completedMW)
	for _, n := range tm.schoolTrackLetters(completedMW) { tm.Inbox = append([]InboxItem{n}, tm.Inbox...) }
	if completedMW == 24 {
		tm.PushInbox("nxgn", "NXGN 2027: The 50 Best Wonderkids in World Football Ranked", "Goal's annual NXGN rankings are officially out! The 12 Franchise Prodigies headline the global elite.", completedMW, nil, "", "")
	}
	for _, n := range CheckWonderkidMilestones(completedMW, tm.SeasonName, tm.ClubsList, tm.MilestonesFired, tm.RNG) { tm.Inbox = append([]InboxItem{n}, tm.Inbox...) }
	tm.DerbiesPlayedThisMW = map[string]bool{}
}

func (tm *TournamentManager) pickPlayerOfTheWeek(mw int) {
	type cand struct { rating float64; goals, ovr int; row map[string]interface{} }
	var best *cand
	consider := func(f *Fixture) {
		if f.Status != "finished" || f.Report == nil || f.Report.MOTM == nil { return }
		motm := f.Report.MOTM
		rating := 0.0
		if motm.Rating != nil { rating = *motm.Rating }
		club := tm.Clubs[f.HomeID]
		if motm.Side == "away" { club = tm.Clubs[f.AwayID] }
		clubName, clubShort := "", ""
		if club != nil { clubName, clubShort = club.ClubName, club.ShortName }
		c := cand{rating: rating, goals: motm.MatchGoals, ovr: motm.OVR, row: map[string]interface{}{
			"player_id": motm.PlayerID, "full_name": motm.FullName, "position": motm.Position, "ovr": motm.OVR,
			"rating": rating, "goals": motm.MatchGoals, "assists": motm.MatchAssists, "club_name": clubName,
			"club_short": clubShort, "is_wonderkid": motm.IsWK, "matchweek": mw,
		}}
		if best == nil || c.rating > best.rating || (c.rating == best.rating && c.goals > best.goals) || (c.rating == best.rating && c.goals == best.goals && c.ovr > best.ovr) { cp := c; best = &cp }
	}
	for i := range tm.Fixtures { if tm.Fixtures[i].Matchweek == mw { consider(&tm.Fixtures[i]) } }
	for i := range tm.UCLFixtures { if tm.UCLFixtures[i].Matchweek == mw { consider(&tm.UCLFixtures[i]) } }
	for i := range tm.SuperCupFixtures { if tm.SuperCupFixtures[i].Matchweek == mw { consider(&tm.SuperCupFixtures[i]) } }
	if best != nil {
		tm.PlayerOfTheWeek = best.row
		tm.PushInbox("honour", fmt.Sprintf("Player of the week: %s", best.row["full_name"]), fmt.Sprintf("%s posted a %.1f against the week's best night.", best.row["full_name"], best.rating), mw, nil, fmt.Sprintf("%v", best.row["player_id"]), "")
	}
}

func (tm *TournamentManager) maybeCrownMonth(endMW int) {
	var bandLo, bandHi int
	var name string
	found := false
	for _, b := range MonthBands {
		if b.Hi == endMW || (endMW == tm.MaxMatchweeks && endMW >= b.Lo && endMW <= b.Hi) { bandLo, bandHi, name, found = b.Lo, b.Hi, b.Name, true; break }
	}
	if !found { return }
	for _, a := range tm.MonthlyAwards { if fmt.Sprintf("%v", a["month"]) == name { return } }
	type slot struct { playerID, fullName, position string; ovr int; ratings []float64; goals int; clubName, clubShort string; isWonderkid bool }
	tally := map[string]*slot{}
	addPack := func(clubID string, rows []matchreport.MatchPlayerRow) {
		club := tm.Clubs[clubID]
		for _, row := range rows {
			if !row.Played || row.Minutes <= 0 || row.Rating == nil { continue }
			s := tally[row.PlayerID]
			if s == nil {
				s = &slot{playerID: row.PlayerID, fullName: row.FullName, position: row.Position, ovr: row.OVR, isWonderkid: row.IsWK}
				if club != nil { s.clubName, s.clubShort = club.ClubName, club.ShortName }
				tally[row.PlayerID] = s
			}
			s.ratings = append(s.ratings, *row.Rating); s.goals += row.MatchGoals
		}
	}
	addRows := func(f *Fixture) {
		if f.Status != "finished" || f.Report == nil { return }
		addPack(f.HomeID, append(append([]matchreport.MatchPlayerRow{}, f.Report.HomeXI...), f.Report.HomeBench...))
		addPack(f.AwayID, append(append([]matchreport.MatchPlayerRow{}, f.Report.AwayXI...), f.Report.AwayBench...))
	}
	for mw := bandLo; mw <= bandHi && mw <= endMW; mw++ {
		for i := range tm.Fixtures { if tm.Fixtures[i].Matchweek == mw { addRows(&tm.Fixtures[i]) } }
		for i := range tm.UCLFixtures { if tm.UCLFixtures[i].Matchweek == mw { addRows(&tm.UCLFixtures[i]) } }
		for i := range tm.SuperCupFixtures { if tm.SuperCupFixtures[i].Matchweek == mw { addRows(&tm.SuperCupFixtures[i]) } }
	}
	if len(tally) == 0 { return }
	var best *slot
	bestAvg := -1.0
	for _, s := range tally {
		avg := 0.0; for _, r := range s.ratings { avg += r }; avg /= float64(len(s.ratings))
		if best == nil || avg > bestAvg || (avg == bestAvg && len(s.ratings) > len(best.ratings)) || (avg == bestAvg && len(s.ratings) == len(best.ratings) && s.playerID < best.playerID) { best, bestAvg = s, avg }
	}
	if best == nil { return }
	avg := float64(int(bestAvg*100+0.5)) / 100
	award := map[string]interface{}{"month": name, "through_mw": endMW, "player_id": best.playerID, "full_name": best.fullName, "position": best.position, "ovr": best.ovr, "rating": avg, "apps": len(best.ratings), "goals": best.goals, "club_name": best.clubName, "club_short": best.clubShort, "is_wonderkid": best.isWonderkid}
	tm.MonthlyAwards = append(tm.MonthlyAwards, award)
	tm.GrowthNotifications = append([]string{fmt.Sprintf("Player of the month · %s: %s (%.2f avg).", name, best.fullName, avg)}, tm.GrowthNotifications...)
	if len(tm.GrowthNotifications) > 8 { tm.GrowthNotifications = tm.GrowthNotifications[:8] }
	tm.PushInbox("honour", fmt.Sprintf("%s player of the month: %s", name, best.fullName), fmt.Sprintf("%.2f average across %d appearances.", avg, len(best.ratings)), endMW, nil, best.playerID, "")
}

func (tm *TournamentManager) decayDerbyHeat(mw int) {
	if tm.DerbyHeat == nil { return }
	for k, v := range tm.DerbyHeat { if v > 50 && !tm.DerbiesPlayedThisMW[k] { tm.DerbyHeat[k] = v - 1 } }
	_ = mw
}

func (tm *TournamentManager) maybeExamWeekInbox(completedMW int) {
	next := completedMW + 1
	if !models.IsExamWeek(next) { return }
	sitting := 0
	for _, c := range tm.ClubsList { for _, p := range c.Squad { if p.UniverseWonderkid && p.SchoolConflict("super-league", next) { sitting++ } } }
	if sitting == 0 { return }
	tm.PushInbox("youth", "Exam week: enrolled prodigies sit", "School comes first while they are enrolled. Cup nights are already off-limits in middle school.", completedMW, nil, "", "")
}
