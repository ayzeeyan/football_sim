package models

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestClubUpdateResultAndMorale(t *testing.T) {
	club := &Club{
		ClubID:   "TEST",
		ClubName: "Test FC",
		Morale:   70,
	}

	// 1. Single Win
	club.UpdateResult(2, 0)
	if club.Played != 1 || club.Won != 1 || club.Points != 3 || club.GoalDifference != 2 {
		t.Errorf("UpdateResult(2, 0) stats incorrect: %+v", club)
	}
	if club.Morale != 73 {
		t.Errorf("Morale after win = %d; want 73", club.Morale)
	}
	if len(club.Form) != 1 || club.Form[0] != "W" {
		t.Errorf("Form unexpected: %v", club.Form)
	}

	// 2. Win Streak Bonus (+3 standard + 5 streak bonus on 3rd consecutive W)
	club.UpdateResult(1, 0) // Morale: 73 + 3 = 76
	club.UpdateResult(3, 1) // Morale: 76 + 3 + 5 = 84
	if club.Morale != 84 {
		t.Errorf("Morale after 3-win streak = %d; want 84", club.Morale)
	}

	// 3. Draw (-1 morale, floor 30)
	club.UpdateResult(1, 1) // Morale: 84 - 1 = 83
	if club.Drawn != 1 || club.Points != 10 {
		t.Errorf("Stats after draw: D=%d, Pts=%d", club.Drawn, club.Points)
	}
	if club.Morale != 83 {
		t.Errorf("Morale after draw = %d; want 83", club.Morale)
	}

	// 4. Loss Streak Penalty (-3 standard - 5 streak penalty on 3rd consecutive L)
	club.UpdateResult(0, 2) // L (Morale 80)
	club.UpdateResult(0, 1) // L (Morale 77)
	club.UpdateResult(1, 4) // 3rd L (Morale: 77 - 3 - 5 = 69)
	if club.Morale != 69 {
		t.Errorf("Morale after 3-loss streak = %d; want 69", club.Morale)
	}

	// 5. Max form length is 5
	if len(club.Form) != 5 {
		t.Errorf("Form length = %d; want 5", len(club.Form))
	}
}

func TestClubStartingEleven433(t *testing.T) {
	club := &Club{
		ClubID:   "ARS",
		ClubName: "Arsenal",
	}

	// Create a balanced squad with 2 GK, 6 DEF, 5 MID, 5 FWD
	// One FWD is a wonderkid
	for i := 1; i <= 2; i++ {
		club.Squad = append(club.Squad, &Player{
			PlayerID: fmt.Sprintf("GK_%d", i),
			Category: "GK",
			OVR:      75 + i,
		})
	}
	for i := 1; i <= 6; i++ {
		club.Squad = append(club.Squad, &Player{
			PlayerID: fmt.Sprintf("DEF_%d", i),
			Category: "DEF",
			OVR:      74 + i,
		})
	}
	for i := 1; i <= 5; i++ {
		club.Squad = append(club.Squad, &Player{
			PlayerID: fmt.Sprintf("MID_%d", i),
			Category: "MID",
			OVR:      74 + i,
		})
	}
	for i := 1; i <= 4; i++ {
		club.Squad = append(club.Squad, &Player{
			PlayerID: fmt.Sprintf("FWD_%d", i),
			Category: "FWD",
			OVR:      80 + i, // 81, 82, 83, 84
		})
	}
	// Wonderkid with lower initial OVR (78)
	club.Squad = append(club.Squad, &Player{
		PlayerID:          "WK_FWD",
		Category:          "FWD",
		OVR:               78,
		UniverseWonderkid: true,
		ConsecutiveStarts: 0,
	})

	starters := club.GetStartingEleven()
	if len(starters) != 11 {
		t.Fatalf("Starting XI count = %d; want 11", len(starters))
	}

	counts := map[string]int{}
	wkIncluded := false
	for _, p := range starters {
		counts[p.Category]++
		if p.PlayerID == "WK_FWD" {
			wkIncluded = true
		}
	}

	// Confirm 4-3-3 formation
	if counts["GK"] != 1 {
		t.Errorf("GK count = %d; want 1", counts["GK"])
	}
	if counts["DEF"] != 4 {
		t.Errorf("DEF count = %d; want 4", counts["DEF"])
	}
	if counts["MID"] != 3 {
		t.Errorf("MID count = %d; want 3", counts["MID"])
	}
	if counts["FWD"] != 3 {
		t.Errorf("FWD count = %d; want 3", counts["FWD"])
	}

	// Confirm wonderkid prioritization over higher OVR non-wonderkids
	if !wkIncluded {
		t.Errorf("Wonderkid should have been prioritized to start despite lower base OVR")
	}
}

func TestClubFatigueAndRotation(t *testing.T) {
	club := &Club{
		ClubID: "FATIGUE_TEST",
	}

	// 2 GKs: GK1 has OVR 82 with 4 consecutive starts (drop = (4-2)*3 = 6 -> adj 76)
	// GK2 has OVR 80 with 0 starts (adj 80)
	gk1 := &Player{PlayerID: "GK1", Category: "GK", OVR: 82, ConsecutiveStarts: 4}
	gk2 := &Player{PlayerID: "GK2", Category: "GK", OVR: 80, ConsecutiveStarts: 0}
	club.Squad = append(club.Squad, gk1, gk2)

	// Add minimum outfield players
	for i := 1; i <= 4; i++ {
		club.Squad = append(club.Squad, &Player{PlayerID: fmt.Sprintf("D%d", i), Category: "DEF", OVR: 75})
	}
	for i := 1; i <= 3; i++ {
		club.Squad = append(club.Squad, &Player{PlayerID: fmt.Sprintf("M%d", i), Category: "MID", OVR: 75})
	}
	for i := 1; i <= 3; i++ {
		club.Squad = append(club.Squad, &Player{PlayerID: fmt.Sprintf("F%d", i), Category: "FWD", OVR: 75})
	}

	starters := club.GetStartingEleven()
	var startingGK *Player
	for _, p := range starters {
		if p.Category == "GK" {
			startingGK = p
			break
		}
	}
	if startingGK == nil || startingGK.PlayerID != "GK2" {
		t.Errorf("Fatigued GK1 (eff 76) should rotate out for fresh GK2 (80); started: %v", startingGK)
	}

	// Wonderkid with extreme consecutive starts (>= 5) loses priority
	wkFWD := &Player{
		PlayerID:          "WK_TIRED",
		Category:          "FWD",
		OVR:               70,
		UniverseWonderkid: true,
		ConsecutiveStarts: 5,
	}
	freshFWD := &Player{
		PlayerID:          "FRESH_FWD",
		Category:          "FWD",
		OVR:               85,
		UniverseWonderkid: false,
		ConsecutiveStarts: 0,
	}
	wkPriority, _ := sortKey(wkFWD)
	if wkPriority != 0 {
		t.Errorf("Wonderkid with 5+ consecutive starts should not have priority; got %d", wkPriority)
	}
	_ = freshFWD
}

func TestClubGetBench(t *testing.T) {
	club := &Club{ClubID: "BENCH_TEST"}
	for i := 1; i <= 20; i++ {
		club.Squad = append(club.Squad, &Player{
			PlayerID: fmt.Sprintf("P_%02d", i),
			Position: "CM",
			Category: "MID",
			OVR:      60 + i,
		})
	}

	bench := club.GetBench(nil, 7)
	if len(bench) != 7 {
		t.Fatalf("Bench size = %d; want 7", len(bench))
	}

	// Starters and bench should not overlap
	starters := club.GetStartingEleven()
	starterMap := make(map[string]bool)
	for _, s := range starters {
		starterMap[s.PlayerID] = true
	}
	for _, b := range bench {
		if starterMap[b.PlayerID] {
			t.Errorf("Player %s appears in both starting XI and bench", b.PlayerID)
		}
	}
}

func TestClubJSONUnmarshal(t *testing.T) {
	rawJSON := `{
		"club_id": "CHE",
		"club_name": "Chelsea",
		"short_name": "CHE",
		"league": "Premier League"
	}`

	var c Club
	if err := json.Unmarshal([]byte(rawJSON), &c); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if c.Morale != 70 {
		t.Errorf("Morale default = %d; want 70", c.Morale)
	}
	// Kit color fallback
	if c.PrimaryColor != [3]uint8{3, 70, 148} {
		t.Errorf("Primary kit color for CHE = %v; want [3, 70, 148]", c.PrimaryColor)
	}
}

func TestClubToStandingsRowAndFixtures(t *testing.T) {
	c := &Club{
		ClubID:            "ARS",
		ClubName:          "Arsenal",
		ShortName:         "ARS",
		OverallTeamRating: 84,
	}
	c.UpdateResult(3, 1)

	row := c.ToStandingsRow(1)
	if row.Position != 1 || row.ClubID != "ARS" || row.Points != 3 || row.GoalsFor != 3 || row.GoalsAgainst != 1 {
		t.Errorf("ToStandingsRow output incorrect: %+v", row)
	}

	// Test AvailableSquad with specific fixture args
	wk := &Player{
		PlayerID:          "WK_01",
		FullName:          "Prodigy",
		Category:          "FWD",
		UniverseWonderkid: true,
		Education:         "middle_school",
	}
	c.Squad = []*Player{wk}

	// Regular fixture: wonderkid available
	squadRegular := c.AvailableSquad("super-league:5")
	if len(squadRegular) != 1 {
		t.Errorf("Wonderkid should be in available squad for week 5")
	}

	// Exam week fixture: wonderkid is unavailable and an empty eligible pool stays empty.
	squadExam := c.AvailableSquad("super-league:12")
	if len(squadExam) != 0 {
		t.Errorf("AvailableSquad should exclude unavailable players; got %d", len(squadExam))
	}
}

func TestClubFixtureAvailabilityFiltersXIAndBench(t *testing.T) {
	examKid := &Player{
		PlayerID:          "EXAM_GK",
		FullName:          "Exam Keeper",
		Category:          "GK",
		OVR:               90,
		Age:               16,
		Education:         "high_school",
		UniverseWonderkid: true,
	}
	eligible := &Player{PlayerID: "ELIGIBLE", FullName: "Eligible Defender", Category: "DEF", OVR: 70}
	club := &Club{ClubID: "FIXTURE", Squad: []*Player{examKid, eligible}}

	regular := club.GetStartingEleven("super-league:11")
	if len(regular) != 2 || regular[0].PlayerID != examKid.PlayerID {
		t.Fatalf("regular XI should include the school player: %+v", regular)
	}

	available := club.AvailableSquad("super-league:12")
	if len(available) != 1 || available[0].PlayerID != eligible.PlayerID {
		t.Fatalf("exam availability should retain only the eligible player: %+v", available)
	}
	examXI := club.GetStartingEleven("super-league:12")
	if len(examXI) != 1 || examXI[0].PlayerID != eligible.PlayerID {
		t.Fatalf("no-GK fallback must use the filtered pool: %+v", examXI)
	}
	if bench := club.GetBench(examXI, 7, "super-league:12"); len(bench) != 0 {
		t.Fatalf("exam bench should not reintroduce unavailable players: %+v", bench)
	}

	allUnavailable := &Club{ClubID: "EMPTY_FIXTURE", Squad: []*Player{examKid}}
	if got := allUnavailable.AvailableSquad("super-league:12"); len(got) != 0 {
		t.Fatalf("all-unavailable squad should have empty availability: %+v", got)
	}
	if got := allUnavailable.GetStartingEleven("super-league:12"); len(got) != 0 {
		t.Fatalf("all-unavailable squad should have empty XI: %+v", got)
	}
	if got := allUnavailable.GetBench(nil, 7, "super-league:12"); len(got) != 0 {
		t.Fatalf("all-unavailable squad should have empty bench: %+v", got)
	}
}
