package tournament

import (
	"fmt"

	"football_sim/pkg/matchreport"
	"football_sim/pkg/models"
)

// Mentor drama (Feature 6): three inbox-only events, each firing at most
// once per season per prodigy under its trigger. Pairing is untouched —
// PairSeniorMentors still re-pairs afterwards. No social graph.

// dramaFired reports whether a per-season drama flag is already set.
func (tm *TournamentManager) dramaFired(playerID, kind string) bool {
	if tm.MilestonesFired == nil {
		tm.MilestonesFired = map[string]map[string]bool{}
	}
	return tm.MilestonesFired[playerID][kind+":"+tm.SeasonName]
}

func (tm *TournamentManager) markDramaFired(playerID, kind string) {
	if tm.MilestonesFired == nil {
		tm.MilestonesFired = map[string]map[string]bool{}
	}
	if tm.MilestonesFired[playerID] == nil {
		tm.MilestonesFired[playerID] = map[string]bool{}
	}
	tm.MilestonesFired[playerID][kind+":"+tm.SeasonName] = true
}

// NoteMentorDeparture fires when a veteran mentor leaves in the window.
// Call before PairSeniorMentors re-pairs so the old pairing is still visible.
func (tm *TournamentManager) NoteMentorDeparture(departedID, departedName, sellerID, buyerName string, matchweek int) {
	if departedID == "" {
		return
	}
	for _, club := range tm.ClubsList {
		for _, p := range club.Squad {
			if p == nil || !p.UniverseWonderkid || p.MentorID != departedID {
				continue
			}
			if tm.dramaFired(p.PlayerID, "mentor_left") {
				continue
			}
			tm.markDramaFired(p.PlayerID, "mentor_left")
			name := departedName
			if name == "" {
				name = p.MentorName
			}
			tm.PushInbox("transfer",
				fmt.Sprintf("%s leaves %s without his mentor", name, p.FullName),
				fmt.Sprintf("%s is gone to %s. %s wanted one more season learning from him.", name, buyerName, p.FullName),
				matchweek, []string{club.ClubID, sellerID}, p.PlayerID, "")
		}
	}
}

// mentorDramaForReport scans one finished report for the two match-driven
// dramas: a prodigy red (falling-out) and a benched prodigy whose mentor
// started ("he started ahead of me"). Called from ApplyMatchReport, which
// already holds tm.mu.
func (tm *TournamentManager) mentorDramaForReport(homeClub, awayClub *models.Club, report *matchreport.MatchReport, matchweek int, fixtureID string) {
	if report == nil {
		return
	}
	pairs := []struct {
		club *models.Club
		rows []matchreport.MatchPlayerRow
		side string
	}{
		{homeClub, append(append([]matchreport.MatchPlayerRow{}, report.HomeXI...), report.HomeBench...), "home"},
		{awayClub, append(append([]matchreport.MatchPlayerRow{}, report.AwayXI...), report.AwayBench...), "away"},
	}
	// Falling-out after a red.
	for _, e := range report.Events {
		if e.Type != "red" || e.Player == nil || e.Disallowed {
			continue
		}
		for _, pr := range pairs {
			if pr.club == nil {
				continue
			}
			for _, p := range pr.club.Squad {
				if p == nil || !p.UniverseWonderkid || p.PlayerID != e.Player.PlayerID {
					continue
				}
				if p.MentorName == "" || tm.dramaFired(p.PlayerID, "mentor_feud") {
					continue
				}
				tm.markDramaFired(p.PlayerID, "mentor_feud")
				tm.PushInbox("wonderkid",
					fmt.Sprintf("Falling-out: %s sees red, %s unimpressed", p.FullName, p.MentorName),
					fmt.Sprintf("Sent off and straight down the tunnel past %s. The mentor relationship needs mending.", p.MentorName),
					matchweek, []string{pr.club.ClubID}, p.PlayerID, fixtureID)
			}
		}
	}
	// "He started ahead of me" week: available kid unused, mentor started.
	var comp string
	var week int
	if f := tm.findFixtureUnlocked(fixtureID); f != nil {
		comp, week = f.Competition, f.Matchweek
	} else {
		week = matchweek
	}
	for _, pr := range pairs {
		if pr.club == nil {
			continue
		}
		played := map[string]bool{}
		mentorStarted := map[string]bool{}
		for _, row := range pr.rows {
			if row.Played || row.Minutes > 0 {
				played[row.PlayerID] = true
			}
			if row.Starter {
				mentorStarted[row.PlayerID] = true
			}
		}
		for _, p := range pr.club.Squad {
			if p == nil || !p.UniverseWonderkid || p.MentorID == "" {
				continue
			}
			if played[p.PlayerID] || !mentorStarted[p.MentorID] {
				continue
			}
			if comp != "" && p.IsUnavailable(comp, week) {
				continue
			}
			if tm.dramaFired(p.PlayerID, "mentor_ahead") {
				continue
			}
			tm.markDramaFired(p.PlayerID, "mentor_ahead")
			tm.PushInbox("wonderkid",
				fmt.Sprintf("He started ahead of me: %s benched for %s", p.FullName, p.MentorName),
				fmt.Sprintf("%s watched %s start in his place. The kid wants words with the manager.", p.FullName, p.MentorName),
				matchweek, []string{pr.club.ClubID}, p.PlayerID, fixtureID)
		}
	}
}
