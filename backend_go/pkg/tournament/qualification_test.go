package tournament

import "testing"

func qualificationCounts(q map[string]map[string]string) map[string]int {
	out := map[string]int{}
	seen := map[string]string{}
	for _, id := range europeanCompOrder {
		out[id] = len(q[id])
		for clubID := range q[id] {
			if prev, ok := seen[clubID]; ok {
				out["duplicate:"+clubID]++
				out["conflict:"+prev+"/"+id]++
			}
			seen[clubID] = id
		}
	}
	return out
}

func assertEuropeanField(t *testing.T, q map[string]map[string]string) {
	t.Helper()
	counts := qualificationCounts(q)
	if counts[compChampionsLeague] != 36 || counts[compEuropaLeague] != 20 || counts[compConferenceLeague] != 20 {
		t.Fatalf("participant counts=%v want 36/20/20", counts)
	}
	for key, n := range counts {
		if n == 0 {
			continue
		}
		if len(key) >= 9 && key[:9] == "duplicate" {
			t.Fatalf("duplicate club in European fields: %s", key)
		}
		if len(key) >= 8 && key[:8] == "conflict" {
			t.Fatalf("club registered in two European competitions: %s", key)
		}
	}
}

func TestEuropeanQualificationMatrix(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	laLiga := tm.worldLeagueStandingsUnlocked("la-liga")
	pl := tm.worldLeagueStandingsUnlocked("premier-league")
	if len(laLiga) < 18 || len(pl) < 18 {
		t.Fatal("domestic tables too small for access-list tests")
	}

	uclWinnerDomestic := laLiga[0].ClubID
	uclWinnerUEL := laLiga[8].ClubID
	uclWinnerOutside := laLiga[len(laLiga)-1].ClubID
	uelWinner := pl[8].ClubID
	ueclWinner := pl[12].ClubID

	t.Run("A_titleholder_also_domestic_ucl", func(t *testing.T) {
		tm.World.Competitions[compChampionsLeague].ChampionID = uclWinnerDomestic
		tm.World.Competitions[compEuropaLeague].ChampionID = ""
		tm.World.Competitions[compConferenceLeague].ChampionID = ""
		q := tm.nextEuropeanQualificationUnlocked()
		assertEuropeanField(t, q)
		if q[compChampionsLeague][uclWinnerDomestic] == "" {
			t.Fatal("domestic UCL winner missing from Champions League")
		}
		if q[compEuropaLeague][uclWinnerDomestic] != "" || q[compConferenceLeague][uclWinnerDomestic] != "" {
			t.Fatal("titleholder duplicated into a lower competition")
		}
	})

	t.Run("B_titleholder_outside_europe", func(t *testing.T) {
		tm.World.Competitions[compChampionsLeague].ChampionID = uclWinnerOutside
		tm.World.Competitions[compEuropaLeague].ChampionID = ""
		tm.World.Competitions[compConferenceLeague].ChampionID = ""
		q := tm.nextEuropeanQualificationUnlocked()
		assertEuropeanField(t, q)
		if q[compChampionsLeague][uclWinnerOutside] == "" {
			t.Fatalf("outside-europe UCL winner %s missing from Champions League", uclWinnerOutside)
		}
		if q[compEuropaLeague][uclWinnerOutside] != "" || q[compConferenceLeague][uclWinnerOutside] != "" {
			t.Fatal("outside-europe titleholder remained in a lower competition")
		}
	})

	t.Run("C_titleholder_domestic_uel", func(t *testing.T) {
		tm.World.Competitions[compChampionsLeague].ChampionID = uclWinnerUEL
		tm.World.Competitions[compEuropaLeague].ChampionID = ""
		tm.World.Competitions[compConferenceLeague].ChampionID = ""
		q := tm.nextEuropeanQualificationUnlocked()
		assertEuropeanField(t, q)
		if q[compChampionsLeague][uclWinnerUEL] == "" {
			t.Fatal("UEL-place UCL winner was not promoted")
		}
		if q[compEuropaLeague][uclWinnerUEL] != "" {
			t.Fatal("promoted titleholder remained in Europa League")
		}
	})

	t.Run("D_uel_winner_earns_ucl", func(t *testing.T) {
		tm.World.Competitions[compChampionsLeague].ChampionID = ""
		tm.World.Competitions[compEuropaLeague].ChampionID = uelWinner
		tm.World.Competitions[compConferenceLeague].ChampionID = ""
		q := tm.nextEuropeanQualificationUnlocked()
		assertEuropeanField(t, q)
		if q[compChampionsLeague][uelWinner] == "" {
			t.Fatal("Europa League winner did not receive a Champions League place")
		}
		if q[compEuropaLeague][uelWinner] != "" {
			t.Fatal("Europa League winner remained in Europa League")
		}
	})

	t.Run("E_two_titleholders", func(t *testing.T) {
		tm.World.Competitions[compChampionsLeague].ChampionID = uclWinnerOutside
		tm.World.Competitions[compEuropaLeague].ChampionID = uelWinner
		tm.World.Competitions[compConferenceLeague].ChampionID = ueclWinner
		q := tm.nextEuropeanQualificationUnlocked()
		assertEuropeanField(t, q)
		if q[compChampionsLeague][uclWinnerOutside] == "" || q[compChampionsLeague][uelWinner] == "" {
			t.Fatal("two titleholder entitlements were not both placed in Champions League")
		}
		if q[compEuropaLeague][ueclWinner] == "" && q[compChampionsLeague][ueclWinner] == "" {
			t.Fatal("Conference League winner received no European place")
		}
	})

	t.Run("H_rebuild_preserves_unique_participants", func(t *testing.T) {
		tm.World.Competitions[compChampionsLeague].ChampionID = uclWinnerOutside
		q := tm.nextEuropeanQualificationUnlocked()
		assertEuropeanField(t, q)
		tm.rebuildEuropeanWorldCalendarUnlocked(q)
		for _, id := range europeanCompOrder {
			comp := tm.World.Competitions[id]
			want := europeanTargetSize(id)
			if comp == nil || len(comp.ParticipantIDs) != want || len(comp.QualificationSources) != want {
				t.Fatalf("%s rebuilt participants=%d sources=%d want %d", id, len(comp.ParticipantIDs), len(comp.QualificationSources), want)
			}
			seen := map[string]bool{}
			for _, clubID := range comp.ParticipantIDs {
				if seen[clubID] {
					t.Fatalf("rebuilt %s duplicated %s", id, clubID)
				}
				seen[clubID] = true
				if tm.Clubs[clubID] == nil {
					t.Fatalf("rebuilt %s missing club %s", id, clubID)
				}
			}
		}
	})
}
