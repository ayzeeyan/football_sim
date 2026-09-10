package matchengine

import (
	"math"
	"testing"
)

func TestPacket4LiveBigGameBonusUsesExplicitFixtureContext(t *testing.T) {
	configure := func(e *LiveMatchEngine) {
		for _, p := range e.HomeStarters {
			if p.Category == "FWD" {
				p.UniverseWonderkid = true
				p.Personality = "big_game_performer"
				p.MentorName = ""
				p.Composure = 75
			}
		}
		e.HomeScore, e.AwayScore = 2, 2
		e.CurrentMinute = 75
	}

	foundDerby := false
	foundUCL := false
	for seed := int64(1); seed <= 500 && (!foundDerby || !foundUCL); seed++ {
		normal := chunk3LiveEngine(seed)
		derby := chunk3LiveEngine(seed)
		ucl := chunk3LiveEngine(seed)
		configure(normal)
		configure(derby)
		configure(ucl)
		derby.Competition = "super-league"
		derby.IsRecognizedDerby = true
		ucl.Competition = "ucl"

		normal.ResolveShot(normal.HomeClub, normal.AwayClub, normal.HomeStarters, normal.AwayStarters)
		derby.ResolveShot(derby.HomeClub, derby.AwayClub, derby.HomeStarters, derby.AwayStarters)
		ucl.ResolveShot(ucl.HomeClub, ucl.AwayClub, ucl.HomeStarters, ucl.AwayStarters)
		if len(normal.LiveShots) != 1 || len(derby.LiveShots) != 1 || len(ucl.LiveShots) != 1 {
			continue
		}

		derbyDelta := derby.LiveShots[0].XG - normal.LiveShots[0].XG
		if math.Abs(derbyDelta-0.03) < 0.001 {
			foundDerby = true
		}
		uclDelta := ucl.LiveShots[0].XG - normal.LiveShots[0].XG
		if math.Abs(uclDelta-0.03) < 0.001 {
			foundUCL = true
		}
	}
	if !foundDerby || !foundUCL {
		t.Fatalf("live big-game conversion bonus missing: derby=%v ucl=%v", foundDerby, foundUCL)
	}
}

func TestPacket4LiveCloseLateWithoutContextHasNoBigGameBonus(t *testing.T) {
	normal := chunk3LiveEngine(991)
	contextual := chunk3LiveEngine(991)
	for _, e := range []*LiveMatchEngine{normal, contextual} {
		for _, p := range e.HomeStarters {
			if p.Category == "FWD" {
				p.UniverseWonderkid = true
				p.Personality = "big_game_performer"
				p.MentorName = ""
				p.Composure = 75
			}
		}
		e.HomeScore, e.AwayScore = 2, 2
		e.CurrentMinute = 75
	}
	contextual.Competition = "super-cup"

	normal.ResolveShot(normal.HomeClub, normal.AwayClub, normal.HomeStarters, normal.AwayStarters)
	contextual.ResolveShot(contextual.HomeClub, contextual.AwayClub, contextual.HomeStarters, contextual.AwayStarters)
	if len(normal.LiveShots) != len(contextual.LiveShots) {
		t.Fatalf("same seeded close/late shots diverged: %d != %d", len(normal.LiveShots), len(contextual.LiveShots))
	}
	if len(normal.LiveShots) == 1 && math.Abs(normal.LiveShots[0].XG-contextual.LiveShots[0].XG) > 0.001 {
		t.Fatalf("non-derby super-cup inherited close/late bonus: normal=%v super_cup=%v", normal.LiveShots[0].XG, contextual.LiveShots[0].XG)
	}
}
