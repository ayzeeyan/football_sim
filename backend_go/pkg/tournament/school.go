package tournament

import (
	"fmt"

	"football_sim/pkg/models"
)

// School letter weeks (Feature 4): exam-term correspondence from the school,
// the mentor, and the board. These fire only when one of these matchweeks
// completes — never on other weeks.
var schoolLetterWeeks = map[int]bool{12: true, 13: true, 24: true, 25: true}

// isSchoolLetterWeek reports whether exam letters fire for a completed week.
func isSchoolLetterWeek(completedMW int) bool {
	return schoolLetterWeeks[completedMW]
}

// schoolTrackLetters builds the exam-term inbox letters for one completed
// matchweek. Returns nil outside weeks 12–13 and 24–25. One school letter
// per benched enrolled prodigy, one mentor letter per benched mentored
// prodigy, and a single board letter per week. A breakout kid (10+ apps) is
// moved to the club-forced track by the board letter.
func (tm *TournamentManager) schoolTrackLetters(completedMW int) []InboxItem {
	if !isSchoolLetterWeek(completedMW) {
		return nil
	}
	type kid struct {
		player *models.Player
		clubID string
	}
	var benched []kid
	for _, club := range tm.ClubsList {
		for _, p := range club.Squad {
			if p == nil || !p.UniverseWonderkid {
				continue
			}
			if p.SchoolConflict("super-league", completedMW) {
				benched = append(benched, kid{p, club.ClubID})
			}
		}
	}
	if len(benched) == 0 {
		return nil
	}
	var out []InboxItem
	mk := func(category, headline, body string, clubIDs []string, playerID string) {
		tm.InboxSeq++
		id := fmt.Sprintf("IN_%s_%d_%d", tm.SeasonName, completedMW, tm.InboxSeq)
		out = append(out, NewInboxItem(id, category, headline, body, completedMW, tm.SeasonName, clubIDs, playerID, ""))
	}
	for _, k := range benched {
		p := k.player
		mk("youth",
			fmt.Sprintf("School letter: %s sits out MW %d", p.FullName, completedMW),
			fmt.Sprintf("Exams first. %s stays in school on the %s track and is benched for matchweek %d.", p.FullName, p.SchoolTrackLabel(), completedMW),
			[]string{k.clubID}, p.PlayerID)
		if p.MentorName != "" {
			mk("wonderkid",
				fmt.Sprintf("Mentor note: %s on %s's exam benching", p.MentorName, p.FullName),
				fmt.Sprintf("%s backs the books-first call for %s: head down, exams, then back in the XI.", p.MentorName, p.FullName),
				[]string{k.clubID}, p.PlayerID)
		}
	}
	// Board letter: one per week. Breakout kids are forced onto the
	// club-forced track so exams never bench them again.
	var forced []string
	for _, k := range benched {
		if k.player.Appearances+k.player.CareerApps >= 10 && k.player.SchoolTrack != models.SchoolTrackClubForced {
			k.player.SchoolTrack = models.SchoolTrackClubForced
			forced = append(forced, k.player.FullName)
		}
	}
	body := fmt.Sprintf("Exam-term policy for matchweek %d: enrolled prodigies sit unless football-first or club-forced.", completedMW)
	if len(forced) > 0 {
		body += fmt.Sprintf(" After breakout seasons, the board forces football full-time: %s.", joinNames(forced))
	}
	mk("system",
		fmt.Sprintf("Board letter: exam-term policy, MW %d", completedMW),
		body, nil, "")
	return out
}

func joinNames(names []string) string {
	out := ""
	for i, n := range names {
		if i > 0 {
			out += ", "
		}
		out += n
	}
	return out
}
