package tournament

import (
	"testing"

	"football_sim/pkg/models"
)

// buildUnrestTestClub returns a club with a controlled season state: the
// star has barely featured while the squad has played.
func buildUnrestTestClub() *models.Club {
	club := &models.Club{ClubID: "EPL-TST", ClubName: "Test FC", Played: 20}
	star := &models.Player{PlayerID: "P-STAR", FullName: "Marco Star", OVR: 84, Age: 29, Appearances: 3}
	fringe := &models.Player{PlayerID: "P-FRN", FullName: "Young Fringe", OVR: 66, Age: 21, Appearances: 1}
	regular := &models.Player{PlayerID: "P-REG", FullName: "Regular Starter", OVR: 78, Age: 26, Appearances: 18}
	for i := 0; i < 14; i++ {
		club.Squad = append(club.Squad, &models.Player{
			PlayerID: string(rune('A'+i)) + "-FILL", FullName: "Filler", OVR: 60 + i, Age: 25, Appearances: 10,
		})
	}
	club.Squad = append(club.Squad, star, fringe, regular)
	return club
}

// A peak-age star starved of minutes wants to leave; a regular starter and
// a low-rated youngster stay content.
func TestUnrestStarvedStarWantsToLeave(t *testing.T) {
	club := buildUnrestTestClub()
	rows := playerUnrestForClub(club)
	if len(rows) == 0 {
		t.Fatal("expected unrest rows for the starved star")
	}
	byID := map[string]PlayerUnrest{}
	for _, row := range rows {
		byID[row.PlayerID] = row
	}
	star, ok := byID["P-STAR"]
	if !ok {
		t.Fatal("the starved star has no unrest row")
	}
	if star.Level != models.UnrestWantsToLeave {
		t.Fatalf("star level=%q want wants_to_leave", star.Level)
	}
	if star.SquadRank != 1 {
		t.Fatalf("star squad rank=%d want 1", star.SquadRank)
	}
	if _, ok := byID["P-REG"]; ok {
		t.Fatal("a regular starter must not be unrestful")
	}
	if _, ok := byID["P-FRN"]; ok {
		t.Fatal("a low-rated youngster has no case")
	}
}

// Too early in the season, nobody has a case.
func TestUnrestNeedsASeasonSample(t *testing.T) {
	club := buildUnrestTestClub()
	club.Played = 4
	if rows := playerUnrestForClub(club); len(rows) != 0 {
		t.Fatalf("early season produced %d rows, want 0", len(rows))
	}
}

// The weekly tick charges morale and publishes exactly one story for the
// loudest case.
func TestUnrestWeeklyTickChargesMoraleAndPublishes(t *testing.T) {
	tm := testManagerForWatch(t)
	club := buildUnrestTestClub()
	tm.Clubs["EPL-TST"] = club
	tm.ClubsList = append(tm.ClubsList, club)
	star := club.Squad[len(club.Squad)-3]
	star.Morale = 70
	before := star.Morale

	tm.mu.Lock()
	tm.evaluatePlayerUnrestUnlocked(20)
	tm.mu.Unlock()

	if star.Morale != before-4 {
		t.Fatalf("star morale=%d want %d", star.Morale, before-4)
	}
	stories := 0
	for _, item := range tm.Inbox {
		if item.PlayerID == "P-STAR" {
			stories++
		}
	}
	if stories != 1 {
		t.Fatalf("published %d stories for the star, want 1", stories)
	}
}

// The assessment is deterministic: the same facts produce the same rows.
func TestUnrestIsDeterministic(t *testing.T) {
	club := buildUnrestTestClub()
	a := playerUnrestForClub(club)
	b := playerUnrestForClub(club)
	if len(a) != len(b) {
		t.Fatalf("row counts differ: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("row %d differs: %+v vs %+v", i, a[i], b[i])
		}
	}
}
