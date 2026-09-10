package matchengine

import "testing"

func radarByID(actors []LivePlayerRadar, id string) *LivePlayerRadar {
	for i := range actors {
		if actors[i].PlayerID == id {
			return &actors[i]
		}
	}
	return nil
}

// F6: the universe-wonderkid radar glow must follow the kid across
// substitutions, on or off, instead of sticking to the old slot.
func TestWonderkidGlowFollowsSubstitution(t *testing.T) {
	e := chunk3LiveEngine(90909)
	if len(e.HomeBench) < 2 {
		t.Fatal("expected a home bench for the sub test")
	}

	// Sub the wonderkid ON via the direct (AI dugout) path.
	out := e.HomeStarters[5]
	kid := e.HomeBench[0]
	plain := e.HomeBench[1]
	kid.UniverseWonderkid = true
	plain.UniverseWonderkid = false
	e.ExecuteDirectSub("home", out, kid, 60, "test")
	actor := radarByID(e.HomePlayers, kid.PlayerID)
	if actor == nil {
		t.Fatal("subbed-on wonderkid missing from radar")
	}
	if !actor.UniverseWonderkid || !actor.IsWonderkid {
		t.Fatalf("glow did not follow the kid on: %+v", actor)
	}

	// Sub the wonderkid back OFF: the glow must leave with him.
	e.ExecuteDirectSub("home", kid, plain, 70, "test")
	if again := radarByID(e.HomePlayers, kid.PlayerID); again != nil {
		t.Fatalf("subbed-off wonderkid still on radar: %+v", again)
	}
	replacement := radarByID(e.HomePlayers, plain.PlayerID)
	if replacement == nil {
		t.Fatal("subbed-on replacement missing from radar")
	}
	if replacement.UniverseWonderkid || replacement.IsWonderkid {
		t.Fatalf("glow stuck on the old slot: %+v", replacement)
	}

	// Planned-sub path keeps identity on the new actor too.
	e2 := chunk3LiveEngine(91919)
	idx := -1
	for i, sub := range e2.PlannedSubs {
		if sub.Side == "home" && !sub.Done && sub.Out != nil && sub.In != nil {
			idx = i
			break
		}
	}
	if idx < 0 {
		t.Fatal("expected a planned home sub")
	}
	e2.PlannedSubs[idx].In.UniverseWonderkid = true
	e2.PlannedSubs[idx].Out.UniverseWonderkid = false
	inID := e2.PlannedSubs[idx].In.PlayerID
	e2.ExecuteSub(idx)
	actor2 := radarByID(e2.HomePlayers, inID)
	if actor2 == nil {
		t.Fatal("planned-sub wonderkid missing from radar")
	}
	if !actor2.UniverseWonderkid || !actor2.IsWonderkid {
		t.Fatalf("glow did not follow the kid on planned sub: %+v", actor2)
	}
}
