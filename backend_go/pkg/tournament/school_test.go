package tournament

import (
	"testing"

	"football_sim/pkg/datamanager"
	"football_sim/pkg/growth"
	"football_sim/pkg/models"
)

func testManagerForSchool(t *testing.T) *TournamentManager {
	t.Helper()
	ge := growth.NewGrowthEngine(11)
	dm := datamanager.NewDataManager("../../dataset.json", ge)
	if dm == nil {
		t.Skip("dataset unavailable")
	}
	elite := dm.GetEliteClubs()
	if len(elite) != 12 {
		t.Skip("elite clubs unavailable")
	}
	return NewTournamentManager(elite, ge, 11)
}

func enrolledKid(tm *TournamentManager) *models.Player {
	for _, c := range tm.ClubsList {
		for _, p := range c.Squad {
			if p.UniverseWonderkid && (p.Education == "middle_school" || p.Education == "high_school") {
				return p
			}
		}
	}
	return nil
}

func TestSchoolTrackExamBenching(t *testing.T) {
	tm := testManagerForSchool(t)
	kid := enrolledKid(tm)
	if kid == nil {
		t.Skip("no enrolled prodigy")
	}
	kid.SchoolTrack = models.SchoolTrackStay
	if !kid.SchoolConflict("super-league", 12) {
		t.Fatal("stay-in-school kid should sit exam week 12")
	}
	kid.SchoolTrack = models.SchoolTrackFootballFirst
	if kid.SchoolConflict("super-league", 12) {
		t.Fatal("football-first kid should play exam week 12")
	}
	kid.SchoolTrack = models.SchoolTrackClubForced
	if kid.SchoolConflict("super-league", 12) {
		t.Fatal("club-forced kid should play exam week 12")
	}
	// Cup-night sit for middle school is unchanged by the track.
	kid.SchoolTrack = models.SchoolTrackFootballFirst
	kid.Education = "middle_school"
	if !kid.SchoolConflict("ucl", 20) {
		t.Fatal("middle-school cup-night sit must remain regardless of track")
	}
	kid.SchoolTrack = models.SchoolTrackStay
	if !kid.SchoolConflict("ucl", 20) {
		t.Fatal("middle-school cup-night sit must remain")
	}
}

func TestSchoolLettersFireOnlyExamTermWeeks(t *testing.T) {
	tm := testManagerForSchool(t)
	for _, mw := range []int{1, 5, 11, 14, 23, 26, 33} {
		if got := tm.schoolTrackLetters(mw); len(got) != 0 {
			t.Fatalf("week %d must produce no school letters, got %d", mw, len(got))
		}
	}
	seenSenders := map[string]bool{}
	for _, mw := range []int{12, 13, 24, 25} {
		letters := tm.schoolTrackLetters(mw)
		if len(letters) == 0 {
			t.Fatalf("week %d must produce school/mentor/board letters", mw)
		}
		for _, l := range letters {
			if l.Matchweek != mw {
				t.Fatalf("letter %+v stamped wrong week", l)
			}
			switch {
			case len(l.Headline) >= 13 && l.Headline[:13] == "School letter":
				seenSenders["school"] = true
				if l.PlayerID == "" {
					t.Fatal("school letter must link the kid")
				}
			case len(l.Headline) >= 12 && l.Headline[:12] == "Mentor note:":
				seenSenders["mentor"] = true
			case len(l.Headline) >= 13 && l.Headline[:13] == "Board letter:":
				seenSenders["board"] = true
			default:
				t.Fatalf("unexpected letter headline %q", l.Headline)
			}
		}
	}
	for _, s := range []string{"school", "board"} {
		if !seenSenders[s] {
			t.Fatalf("missing %s letters across exam-term weeks", s)
		}
	}
}

func TestBenchedKidScoresNothingOnExamWeek(t *testing.T) {
	tm := testManagerForSchool(t)
	tm.CurrentMatchweek = 12
	// Force every prodigy onto the stay track so exam benching bites.
	for _, c := range tm.ClubsList {
		for _, p := range c.Squad {
			if p.UniverseWonderkid {
				p.SchoolTrack = models.SchoolTrackStay
				if p.Education == "dropout" || p.Education == "graduated" || p.Education == "none" {
					p.Education = "high_school"
				}
			}
		}
	}
	var target *Fixture
	for i := range tm.Fixtures {
		if tm.Fixtures[i].Matchweek == 12 && tm.Fixtures[i].Status == "scheduled" {
			target = &tm.Fixtures[i]
			break
		}
	}
	if target == nil {
		t.Skip("no MW12 league fixture")
	}
	home, away := tm.Clubs[target.HomeID], tm.Clubs[target.AwayID]
	var benched *models.Player
	for _, p := range append(append([]*models.Player{}, home.Squad...), away.Squad...) {
		if p.UniverseWonderkid && p.SchoolConflict("super-league", 12) {
			benched = p
			break
		}
	}
	if benched == nil {
		t.Skip("no benched kid on this fixture")
	}
	res := tm.SimulateFixture(target.FixtureID)
	if res["status"] != "success" {
		t.Fatalf("sim failed: %v", res)
	}
	f := tm.findFixtureUnlocked(target.FixtureID)
	if f == nil || f.Report == nil {
		t.Fatal("missing report")
	}
	for _, row := range append(f.Report.HomeXI, f.Report.AwayXI...) {
		if row.PlayerID == benched.PlayerID {
			t.Fatalf("benched kid %s appeared in exam-week XI", benched.FullName)
		}
	}
	for _, e := range f.Report.Events {
		if e.Scorer != nil && e.Scorer.PlayerID == benched.PlayerID {
			t.Fatalf("benched kid %s credited with exam-week goal", benched.FullName)
		}
		if e.Assister != nil && e.Assister.PlayerID == benched.PlayerID {
			t.Fatalf("benched kid %s credited with exam-week assist", benched.FullName)
		}
	}
}
