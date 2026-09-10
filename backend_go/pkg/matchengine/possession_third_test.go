package matchengine

import "testing"

// F7: possession is territory, not tick-count. A side pinned in their own
// box must not read 50/50 just because raw ticks are even.
func TestPossessionWeightedByThird(t *testing.T) {
	e := chunk3LiveEngine(777)
	// Keep the phase machine quiet: tiny steps never reach the phase timer,
	// so BallTarget stays where the test pins it and possession never flips.
	for i := 0; i < 100; i++ {
		e.PossessionTeam = "home"
		e.BallTarget.X = 0.2 // pinned in the home box: sterile
		e.Update(0.001)
	}
	for i := 0; i < 100; i++ {
		e.PossessionTeam = "away"
		e.BallTarget.X = 0.5 // midfield: full value
		e.Update(0.001)
	}

	if e.HomePossessionTicks != e.AwayPossessionTicks {
		t.Fatalf("test setup should hold even ticks, got %d/%d",
			e.HomePossessionTicks, e.AwayPossessionTicks)
	}
	got := e.HomePossessionPct()
	if got >= 50 {
		t.Fatalf("pinned home side reads %d%%; want clearly below 50", got)
	}
	if got != 100-e.AwayPossessionPct() {
		t.Fatal("away share does not mirror home share")
	}

	// Front-foot possession up the other end flips the tilt.
	e2 := chunk3LiveEngine(778)
	for i := 0; i < 100; i++ {
		e2.PossessionTeam = "home"
		e2.BallTarget.X = 0.8 // camped in the attacking third
		e2.Update(0.001)
	}
	for i := 0; i < 100; i++ {
		e2.PossessionTeam = "away"
		e2.BallTarget.X = 0.5
		e2.Update(0.001)
	}
	if got := e2.HomePossessionPct(); got <= 50 {
		t.Fatalf("attacking home side reads %d%%; want clearly above 50", got)
	}
}
