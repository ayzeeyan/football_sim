package matchengine

import "testing"

// F1: momentum must leave zero once the sim runs, a pass segment must be
// published, and shots must elevate the ball and flag it goal-bound.
func TestLiveTickFieldsPopulate(t *testing.T) {
	e := chunk3LiveEngine(4242)

	// Run the sim: movements publish trails and ease momentum off zero.
	for i := 0; i < 600 && e.PassTrail == nil; i++ {
		e.Tick(0.016)
	}
	if e.PassTrail == nil {
		t.Fatal("pass_trail never published during a live run")
	}
	if e.PossessionMomentum == 0 {
		t.Fatal("possession_momentum stuck at 0 during a live run")
	}

	// Force a shot flight and check ball presentation state.
	team := e.PossessionTeam
	e.ResolveShot(e.HomeClub, e.AwayClub, e.HomeStarters, e.AwayStarters)
	if !e.BallIsShot {
		t.Fatal("ball is_shot not set after ResolveShot")
	}
	if e.BallHeight <= 0 {
		t.Fatal("ball height not elevated after ResolveShot")
	}
	if e.PassTrail == nil || !e.PassTrail.IsShot {
		t.Fatal("pass_trail should mark the shot segment")
	}
	_ = team

	// Elevation decays so the flag does not stick forever.
	for i := 0; i < 5000 && e.BallIsShot; i++ {
		e.Tick(0.016)
	}
	if e.BallIsShot {
		t.Fatal("ball is_shot never cleared after shot flight decayed")
	}
}
