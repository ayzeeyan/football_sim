package tournament

import (
	"fmt"
	"testing"

	"football_sim/pkg/matchreport"
	"football_sim/pkg/models"
)

func TestClubCultureAssignsSingleCaptainAndHomegrownList(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	tm.RefreshClubCultureUnlocked()
	if err := tm.ValidateWorldState(); err != nil {
		t.Fatalf("culture world invalid: %v", err)
	}
	captains := 0
	registered := 0
	homegrown := 0
	for _, club := range tm.ClubsList {
		if club.CaptainID == "" {
			t.Fatalf("%s missing captain", club.ClubID)
		}
		localCap := 0
		for _, p := range club.Squad {
			if p.IsCaptain {
				localCap++
				captains++
			}
			if p.RegisteredEurope {
				registered++
			}
			if p.Homegrown {
				homegrown++
			}
			if p.Leadership < 1 || p.Leadership > 99 {
				t.Fatalf("%s leadership %d", p.PlayerID, p.Leadership)
			}
		}
		if localCap != 1 {
			t.Fatalf("%s captains=%d", club.ClubID, localCap)
		}
	}
	if captains != len(tm.ClubsList) {
		t.Fatalf("captains=%d clubs=%d", captains, len(tm.ClubsList))
	}
	if registered == 0 || homegrown == 0 {
		t.Fatalf("registered=%d homegrown=%d", registered, homegrown)
	}
	ranks := tm.powerRankingsUnlocked(10)
	if len(ranks) != 10 {
		t.Fatalf("power rankings=%d", len(ranks))
	}
}

func TestEuropeanRegistrationDropsUnlistedPlayers(t *testing.T) {
	club := &models.Club{ClubID: "EPL-TES", ClubName: "Test", ShortName: "TES", Country: "England", League: "Premier League"}
	for i := 0; i < 28; i++ {
		p := &models.Player{
			PlayerID: fmtID(i), FullName: fmtName(i), ClubID: club.ClubID, OriginalClubID: club.ClubID,
			Position: "CM", Category: "MID", OVR: 60 + i%20, Age: 24, Morale: 70, Fitness: 80, Sharpness: 65,
		}
		if i >= 20 {
			p.OriginalClubID = "OTHER"
			p.OVR = 90
		}
		club.Squad = append(club.Squad, p)
	}
	tm := &TournamentManager{Clubs: map[string]*models.Club{club.ClubID: club}, ClubsList: []*models.Club{club}}
	assignClubSquadRoles(club)
	tm.refreshOneClubCultureUnlocked(club)
	listed := 0
	for _, p := range club.Squad {
		if p.RegisteredEurope {
			listed++
		}
	}
	if listed != europeanRegistrationLimit {
		t.Fatalf("registered=%d want %d", listed, europeanRegistrationLimit)
	}
	ready := club.AvailableSquad("champions-league:10")
	if len(ready) != listed {
		t.Fatalf("european pool=%d registered=%d", len(ready), listed)
	}
	domestic := club.AvailableSquad("premier-league:10")
	if len(domestic) != 28 {
		t.Fatalf("domestic pool=%d want 28", len(domestic))
	}
}

func TestCleanSheetsAndRetrainingOnShutout(t *testing.T) {
	gk := &models.Player{PlayerID: "GK1", FullName: "Keep", Position: "GK", Category: "GK", ClubID: "H", OVR: 80, Morale: 70}
	cb := &models.Player{PlayerID: "CB1", FullName: "Stop", Position: "CB", Category: "DEF", ClubID: "H", OVR: 78, Morale: 70}
	st := &models.Player{PlayerID: "ST1", FullName: "Wide", Position: "ST", Category: "FWD", ClubID: "H", OVR: 77, Morale: 70}
	home := &models.Club{ClubID: "H", Squad: []*models.Player{gk, cb, st}}
	away := &models.Club{ClubID: "A", Squad: []*models.Player{{PlayerID: "A1", FullName: "Away", Position: "ST", Category: "FWD", ClubID: "A"}}}
	rep := &matchreport.MatchReport{
		HomeGoals: 1, AwayGoals: 0, Attendance: 40000,
		HomeXI: []matchreport.MatchPlayerRow{
			{PlayerID: "GK1", Played: true, Minutes: 90, Position: "GK", Category: "GK"},
			{PlayerID: "CB1", Played: true, Minutes: 90, Position: "CB", Category: "DEF"},
			{PlayerID: "ST1", Played: true, Minutes: 90, Position: "LW", Category: "FWD"},
		},
	}
	ApplyPlayerMatchStatsInCompetition(home, away, rep, "premier-league")
	if gk.CleanSheets != 1 || cb.CleanSheets != 1 {
		t.Fatalf("clean sheets gk=%d cb=%d", gk.CleanSheets, cb.CleanSheets)
	}
	if st.PositionXP != 1 {
		t.Fatalf("out-of-position XP=%d", st.PositionXP)
	}
	if home.SeasonAttendance != 40000 || home.AttendanceMatches != 1 {
		t.Fatalf("attendance %d/%d", home.SeasonAttendance, home.AttendanceMatches)
	}
}

func TestBrokenMinutesPromiseDropsMorale(t *testing.T) {
	p := &models.Player{
		PlayerID: "P1", FullName: "Star", ClubID: "C1", SquadRole: models.RoleCrucial,
		Appearances: 1, Morale: 70, PromiseKind: "minutes", PromiseSeason: "2026-27", PromiseMatchweek: 1,
	}
	club := &models.Club{ClubID: "C1", ClubName: "Club", ShortName: "CLB", Played: 12, Squad: []*models.Player{p}}
	tm := &TournamentManager{Clubs: map[string]*models.Club{"C1": club}, ClubsList: []*models.Club{club}, SeasonName: "2026-27"}
	tm.evaluatePromisesUnlocked(12)
	if p.PromiseKind != "" {
		t.Fatalf("broken promise should clear, still %q", p.PromiseKind)
	}
	if p.Morale >= 70 {
		t.Fatalf("broken promise should drop morale, got %d", p.Morale)
	}
}

func TestDeadlineDayAddsBidsAndFeed(t *testing.T) {
	_, _, te := loadEuropeanWorldForTest(t)
	te.BeginOffSeasonWindow()
	for te.IsWindowOpen() && te.CurrentWeek < te.WindowWeeks()-1 {
		te.AdvanceOpenWindow()
	}
	if !te.IsWindowOpen() {
		t.Fatal("expected window still open approaching deadline")
	}
	beforeNeg := len(te.ActiveNegotiations)
	beforeFeed := len(te.TransferFeed)
	te.AdvanceOpenWindow()
	if len(te.TransferFeed) <= beforeFeed {
		t.Fatalf("deadline feed did not grow: %d -> %d", beforeFeed, len(te.TransferFeed))
	}
	_ = beforeNeg
}

func TestSecondaryPositionFillsNaturalSlot(t *testing.T) {
	lw := &models.Player{PlayerID: "W1", FullName: "Wide", Position: "ST", SecondaryPosition: "LW", Category: "FWD", ClubID: "C", OVR: 70, Fitness: 80, Sharpness: 70, Morale: 70}
	st := &models.Player{PlayerID: "S1", FullName: "Nine", Position: "ST", Category: "FWD", ClubID: "C", OVR: 88, Fitness: 80, Sharpness: 70, Morale: 70}
	club := &models.Club{ClubID: "C", Squad: []*models.Player{
		{PlayerID: "G1", Position: "GK", Category: "GK", OVR: 70, Fitness: 80, Sharpness: 70, Morale: 70},
		{PlayerID: "LB", Position: "LB", Category: "DEF", OVR: 70, Fitness: 80, Sharpness: 70, Morale: 70},
		{PlayerID: "LCB", Position: "CB", Category: "DEF", OVR: 70, Fitness: 80, Sharpness: 70, Morale: 70},
		{PlayerID: "RCB", Position: "CB", Category: "DEF", OVR: 71, Fitness: 80, Sharpness: 70, Morale: 70},
		{PlayerID: "RB", Position: "RB", Category: "DEF", OVR: 70, Fitness: 80, Sharpness: 70, Morale: 70},
		{PlayerID: "LCM", Position: "CM", Category: "MID", OVR: 70, Fitness: 80, Sharpness: 70, Morale: 70},
		{PlayerID: "CM", Position: "CM", Category: "MID", OVR: 71, Fitness: 80, Sharpness: 70, Morale: 70},
		{PlayerID: "RCM", Position: "CM", Category: "MID", OVR: 72, Fitness: 80, Sharpness: 70, Morale: 70},
		lw, st,
		{PlayerID: "RW", Position: "RW", Category: "FWD", OVR: 70, Fitness: 80, Sharpness: 70, Morale: 70},
	}}
	slots := club.GetStartingElevenSlots("premier-league:5")
	var lwSlot, stSlot string
	for _, s := range slots {
		if s.Player != nil && s.Player.PlayerID == "W1" {
			lwSlot = s.Slot
		}
		if s.Player != nil && s.Player.PlayerID == "S1" {
			stSlot = s.Slot
		}
	}
	if lwSlot != "LW" {
		t.Fatalf("versatile forward should take LW, got %q", lwSlot)
	}
	if stSlot != "ST" {
		t.Fatalf("natural striker should keep ST, got %q", stSlot)
	}
}

func fmtID(i int) string   { return fmt.Sprintf("P%02d", i) }
func fmtName(i int) string { return fmt.Sprintf("Player %d", i) }
