package tournament

import (
	"sort"
	"testing"
)

// After a full season, European knockouts must be home-and-away ties (two
// fixtures per tie) with real aggregate winners, except the one-off final.
// Domestic cups stay single-leg.
func TestWorldEuropeanKnockoutsAreTwoLeggedWithRealWinners(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	batch := tm.SimulateBatchWeeks(38)
	if batch.Status != "success" || !batch.SeasonFinished {
		t.Fatalf("season did not finish: %+v", batch)
	}
	for _, id := range []string{"champions-league", "europa-league", "conference-league"} {
		comp := tm.World.Competitions[id]
		if comp == nil || comp.ChampionID == "" {
			t.Fatalf("%s has no champion", id)
		}
		if len(comp.Rounds) == 0 {
			t.Fatalf("%s has no rounds", id)
		}
		// The post-league-phase playoff must not collide with the round of 16:
		// playoff legs open the knockout calendar (weeks 28/29), final closes it.
		if first := comp.Rounds[0]; first.Stage != "Knockout play-off" {
			t.Fatalf("%s first knockout round stage=%q want Knockout play-off", id, first.Stage)
		} else {
			weeks := map[int]bool{}
			for _, fid := range first.FixtureIDs {
				f := tm.worldFixtureUnlocked(fid)
				if f == nil {
					t.Fatalf("%s missing fixture %s", id, fid)
				}
				weeks[f.Matchweek] = true
			}
			if !weeks[28] || !weeks[29] || len(weeks) != 2 {
				t.Fatalf("%s playoff weeks=%v want exactly 28+29", id, weeks)
			}
		}
		for ri, round := range comp.Rounds {
			last := ri == len(comp.Rounds)-1
			if last {
				if round.Stage != "Final" {
					t.Fatalf("%s last round stage=%q", id, round.Stage)
				}
				if len(round.FixtureIDs) != 1 || len(round.TieIDs) != 1 {
					t.Fatalf("%s final fixtures=%d ties=%d, want 1/1", id, len(round.FixtureIDs), len(round.TieIDs))
				}
				continue
			}
			if len(round.TieIDs) == 0 {
				t.Fatalf("%s round %d (%s) missing tie IDs", id, ri, round.Stage)
			}
			if len(round.FixtureIDs) != 2*len(round.TieIDs) {
				t.Fatalf("%s round %d (%s) fixtures=%d ties=%d, want 2:1", id, ri, round.Stage, len(round.FixtureIDs), len(round.TieIDs))
			}
			// Winners must equal the true aggregate per tie, in sorted tie order.
			byTie := map[string][]string{}
			for _, fid := range round.FixtureIDs {
				f := tm.worldFixtureUnlocked(fid)
				if f == nil {
					t.Fatalf("%s missing fixture %s", id, fid)
				}
				if f.Leg != 1 && f.Leg != 2 {
					t.Fatalf("%s fixture %s has leg %d", id, fid, f.Leg)
				}
				byTie[f.TieID] = append(byTie[f.TieID], fid)
			}
			tieOrder := append([]string(nil), round.TieIDs...)
			sort.Strings(tieOrder)
			if len(round.WinnerIDs) != len(tieOrder) {
				t.Fatalf("%s round %d winners=%d ties=%d", id, ri, len(round.WinnerIDs), len(tieOrder))
			}
			for i, tie := range tieOrder {
				got := tm.winnerIDForWorldTieUnlocked(comp, tie, byTie[tie])
				if got == "" {
					t.Fatalf("%s round %d tie %s has no winner", id, ri, tie)
				}
				if got != round.WinnerIDs[i] {
					t.Fatalf("%s round %d tie %s winner=%s want %s (manufactured?)", id, ri, tie, round.WinnerIDs[i], got)
				}
			}
		}
	}
	for _, id := range []string{"fa-cup", "efl-cup", "copa-del-rey", "dfb-pokal", "coppa-italia", "coupe-de-france"} {
		comp := tm.World.Competitions[id]
		if comp == nil || comp.ChampionID == "" {
			t.Fatalf("%s has no champion", id)
		}
		for ri, round := range comp.Rounds {
			if len(round.FixtureIDs) != len(round.TieIDs) {
				t.Fatalf("%s round %d should stay single-leg: fixtures=%d ties=%d", id, ri, len(round.FixtureIDs), len(round.TieIDs))
			}
		}
	}
}

func TestWorldTieLevelOnAggregate(t *testing.T) {
	mkLeg := func(homeID, awayID string, hg, ag int) *Fixture {
		return &Fixture{HomeID: homeID, AwayID: awayID, HomeGoals: &hg, AwayGoals: &ag}
	}
	// Leg 1: A 1-1 B at A. Leg 2 at B 1-1 -> aggregate 2-2, level.
	if !worldTieLevelOnAggregate(mkLeg("A", "B", 1, 1), "B", "A", 1, 1) {
		t.Fatal("2-2 aggregate should be level")
	}
	// Leg 2 at B 2-0 -> B totals 3, A totals 1: not level.
	if worldTieLevelOnAggregate(mkLeg("A", "B", 1, 1), "B", "A", 2, 0) {
		t.Fatal("3-1 aggregate should not be level")
	}
	// Non-swapped venues still sum per club: leg1 A 2-0 B, leg2 A 0-1 B -> 2-1.
	if worldTieLevelOnAggregate(mkLeg("A", "B", 2, 0), "A", "B", 0, 1) {
		t.Fatal("2-1 aggregate should not be level")
	}
	// Missing leg 1 falls back to the leg-2 scoreline.
	if !worldTieLevelOnAggregate(nil, "B", "A", 2, 2) {
		t.Fatal("nil leg 1 with 2-2 leg 2 should be level")
	}
}

func TestWorldLegBlockedRequiresFinishedFirstLeg(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	comp := tm.World.Competitions["europa-league"]
	hg, ag := 0, 0
	leg1 := tm.newWorldLegFixtureUnlocked("zz-tie-L1", 30, comp.ID, "Round of 16", "EPL-ARS", "EPL-LIV", "zz-tie", 1)
	leg1.HomeGoals, leg1.AwayGoals = &hg, &ag
	leg2 := tm.newWorldLegFixtureUnlocked("zz-tie-L2", 31, comp.ID, "Round of 16", "EPL-LIV", "EPL-ARS", "zz-tie", 2)
	tm.addWorldFixtureUnlocked(leg1)
	tm.addWorldFixtureUnlocked(leg2)
	if msg := tm.worldLegBlocked(tm.worldFixtureUnlocked("zz-tie-L2")); msg == "" {
		t.Fatal("leg 2 should be blocked while leg 1 is scheduled")
	}
	leg1f := tm.worldFixtureUnlocked("zz-tie-L1")
	leg1f.Status = "finished"
	if msg := tm.worldLegBlocked(tm.worldFixtureUnlocked("zz-tie-L2")); msg != "" {
		t.Fatalf("leg 2 should be open after leg 1 finishes: %s", msg)
	}
	if msg := tm.worldLegBlocked(tm.worldFixtureUnlocked("zz-tie-L1")); msg != "" {
		t.Fatalf("leg 1 should never be blocked: %s", msg)
	}
}

func TestWinnerIDForWorldTieUsesAggregateNotFixture(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	comp := tm.World.Competitions["conference-league"]
	l1h, l1a := 2, 0
	l2h, l2a := 1, 0
	leg1 := tm.newWorldLegFixtureUnlocked("zz-agg-L1", 32, comp.ID, "Quarter-final", "EPL-ARS", "EPL-CHE", "zz-agg", 1)
	leg1.HomeGoals, leg1.AwayGoals = &l1h, &l1a
	leg1.Status = "finished"
	leg2 := tm.newWorldLegFixtureUnlocked("zz-agg-L2", 33, comp.ID, "Quarter-final", "EPL-CHE", "EPL-ARS", "zz-agg", 2)
	leg2.HomeGoals, leg2.AwayGoals = &l2h, &l2a
	leg2.Status = "finished"
	tm.addWorldFixtureUnlocked(leg1)
	tm.addWorldFixtureUnlocked(leg2)
	// Aggregate: ARS 2+0=2, CHE 0+1=1 -> ARS wins despite losing leg 2.
	if got := tm.winnerIDForWorldTieUnlocked(comp, "zz-agg", []string{"zz-agg-L1", "zz-agg-L2"}); got != "EPL-ARS" {
		t.Fatalf("aggregate winner=%q want EPL-ARS", got)
	}
}

// Legacy rounds persisted before TieIDs carry empty TieIDs; advancement must
// still resolve one winner per finished fixture.
func TestAdvanceLegacyRoundWithoutTieIDs(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	ids := []string{}
	for i, c := range tm.ClubsList {
		if i < 4 {
			ids = append(ids, c.ClubID)
		}
	}
	sort.Strings(ids)
	comp := &Competition{ID: "zz-cup", Name: "ZZ Cup", Kind: CompetitionDomestic, Prestige: 50, ParticipantIDs: append([]string(nil), ids...), Stage: "Semi-final"}
	tm.World.Competitions["zz-cup"] = comp
	h1, a1, h2, a2 := 2, 0, 1, 1
	pen := []int{3, 4}
	f1 := tm.newWorldFixtureUnlocked("zz-cup-SF1", 10, comp.ID, "Semi-final", ids[0], ids[1])
	f1.HomeGoals, f1.AwayGoals = &h1, &a1
	f1.Status = "finished"
	f2 := tm.newWorldFixtureUnlocked("zz-cup-SF2", 10, comp.ID, "Semi-final", ids[2], ids[3])
	f2.HomeGoals, f2.AwayGoals = &h2, &a2
	f2.DecidedBy = "penalties"
	f2.Penalties = pen
	f2.Status = "finished"
	tm.addWorldFixtureUnlocked(f1)
	tm.addWorldFixtureUnlocked(f2)
	// No TieIDs: legacy shape.
	comp.Rounds = append(comp.Rounds, KnockoutRound{Stage: "Semi-final", EntrantIDs: append([]string(nil), ids...), FixtureIDs: []string{"zz-cup-SF1", "zz-cup-SF2"}})
	msg := tm.advanceWorldCompetitionUnlocked("zz-cup")
	_ = msg
	last := &comp.Rounds[0]
	if len(last.WinnerIDs) != 2 {
		t.Fatalf("legacy round winners=%d want 2", len(last.WinnerIDs))
	}
	if last.WinnerIDs[0] != ids[0] {
		t.Fatalf("SF1 winner=%s want %s", last.WinnerIDs[0], ids[0])
	}
	if last.WinnerIDs[1] != ids[3] {
		t.Fatalf("SF2 pens winner=%s want %s", last.WinnerIDs[1], ids[3])
	}
	if len(comp.Rounds) != 2 || comp.Rounds[1].Stage != "Final" {
		t.Fatalf("expected a scheduled Final, rounds=%+v", comp.Rounds)
	}
}
