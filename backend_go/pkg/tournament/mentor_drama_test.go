package tournament

import (
	"testing"

	"football_sim/pkg/datamanager"
	"football_sim/pkg/growth"
	"football_sim/pkg/matchreport"
	"football_sim/pkg/models"
)

func testManagerForDrama(t *testing.T) *TournamentManager {
	t.Helper()
	ge := growth.NewGrowthEngine(21)
	dm := datamanager.NewDataManager("../../dataset.json", ge)
	if dm == nil {
		t.Skip("dataset unavailable")
	}
	elite := dm.GetEliteClubs()
	if len(elite) != 12 {
		t.Skip("elite clubs unavailable")
	}
	return NewTournamentManager(elite, ge, 21)
}

func dramaKid(t *testing.T, tm *TournamentManager) (*models.Player, *models.Club) {
	t.Helper()
	for _, c := range tm.ClubsList {
		for _, p := range c.Squad {
			if p.UniverseWonderkid && p.MentorID != "" && p.MentorName != "" {
				return p, c
			}
		}
	}
	t.Skip("no mentored prodigy")
	return nil, nil
}

func inboxCount(tm *TournamentManager) int { return len(tm.Inbox) }

func TestMentorLeavesInWindow(t *testing.T) {
	tm := testManagerForDrama(t)
	kid, club := dramaKid(t, tm)
	mentorID, mentorName := kid.MentorID, kid.MentorName
	tm.NoteMentorDeparture(mentorID, mentorName, club.ClubID, "Galacticos", 34)
	if len(tm.Inbox) == 0 {
		t.Fatal("mentor departure should write an inbox letter")
	}
	top := tm.Inbox[0]
	if top.Category != "transfer" || top.PlayerID != kid.PlayerID {
		t.Fatalf("departure letter should link the kid via transfer, got %+v", top)
	}
	n := inboxCount(tm)
	tm.NoteMentorDeparture(mentorID, mentorName, club.ClubID, "Galacticos", 34)
	if inboxCount(tm) != n {
		t.Fatal("mentor departure must fire once per season")
	}
}

func TestFallingOutAfterRed(t *testing.T) {
	tm := testManagerForDrama(t)
	kid, club := dramaKid(t, tm)
	mini := matchreport.ToMiniPlayer(kid)
	report := &matchreport.MatchReport{
		Events: []matchreport.MatchEventItem{
			{Type: "red", Side: "home", Player: &mini},
		},
	}
	home, away := club, tm.ClubsList[0]
	if away == club {
		away = tm.ClubsList[1]
	}
	tm.mentorDramaForReport(home, away, report, 9, "FX")
	if len(tm.Inbox) == 0 {
		t.Fatal("prodigy red should write a falling-out letter")
	}
	if got := tm.Inbox[0].PlayerID; got != kid.PlayerID {
		t.Fatalf("feud letter should link the kid, got %q", got)
	}
	n := inboxCount(tm)
	tm.mentorDramaForReport(home, away, report, 10, "FX2")
	if inboxCount(tm) != n {
		t.Fatal("falling-out must fire once per season")
	}
}

func TestStartedAheadOfMeWeek(t *testing.T) {
	tm := testManagerForDrama(t)
	kid, club := dramaKid(t, tm)
	// Mentor starts; kid gets no minutes.
	var rows []matchreport.MatchPlayerRow
	for _, p := range club.Squad {
		if p.PlayerID == kid.MentorID {
			rows = append(rows, matchreport.MatchPlayerRow{PlayerID: p.PlayerID, Starter: true, Played: true, Minutes: 90})
		}
	}
	if len(rows) == 0 {
		t.Skip("mentor not in squad")
	}
	report := &matchreport.MatchReport{HomeXI: rows}
	home, away := club, tm.ClubsList[0]
	if away == club {
		away = tm.ClubsList[1]
	}
	// Kid must be available for the fixture context; force enrolled-clear.
	kid.SuspendedMatches, kid.InjuredMatches = 0, 0
	tm.mentorDramaForReport(home, away, report, 9, "MW9-"+club.ClubID)
	found := false
	for _, item := range tm.Inbox {
		if item.PlayerID == kid.PlayerID {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("benched kid with starting mentor should get an ahead-of-me letter")
	}
	n := inboxCount(tm)
	tm.mentorDramaForReport(home, away, report, 10, "MW10-"+club.ClubID)
	if inboxCount(tm) != n {
		t.Fatal("ahead-of-me must fire once per season")
	}
}

func TestPairingStillWorks(t *testing.T) {
	tm := testManagerForDrama(t)
	PairSeniorMentors(tm.ClubsList, tm.GrowthEngine)
	for _, c := range tm.ClubsList {
		for _, p := range c.Squad {
			if p.UniverseWonderkid && p.MentorID == "" {
				t.Fatalf("%s lost his mentor pairing", p.FullName)
			}
		}
	}
}
