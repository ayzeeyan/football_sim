package matchengine

import (
	"testing"

	"football_sim/pkg/matchreport"
)

func TestLivePenaltySkipsRedCardedPrimaryTaker(t *testing.T) {
	e := chunk3LiveEngine(44)
	e.PossessionTeam = "home"
	primary := matchreport.DesignatedPenaltyTaker(e.HomeKickoffXI, nil)
	if primary == nil {
		t.Fatal("no designated taker")
	}
	e.Bookings[primary.PlayerID] = 2
	before := len(e.Events)
	e.ResolvePenalty(e.HomeClub, e.HomeStarters)
	if len(e.Events) <= before {
		t.Fatal("penalty produced no event")
	}
	last := e.Events[len(e.Events)-1]
	if last.Type != "penalty" && last.Type != "penalty_miss" {
		t.Fatalf("unexpected event %s", last.Type)
	}
	if last.PlayerID == primary.PlayerID {
		t.Fatalf("red-carded primary taker %s still took the penalty", primary.PlayerID)
	}
	onPitch := map[string]bool{}
	for _, p := range e.OnPitch(e.HomeStarters) {
		onPitch[p.PlayerID] = true
	}
	if !onPitch[last.PlayerID] {
		t.Fatalf("penalty taker %s was not on the pitch", last.PlayerID)
	}
}

func TestLivePenaltySkipsSubstitutedPrimaryTaker(t *testing.T) {
	e := chunk3LiveEngine(45)
	e.PossessionTeam = "home"
	primary := matchreport.DesignatedPenaltyTaker(e.HomeKickoffXI, nil)
	if primary == nil || len(e.HomeBench) == 0 {
		t.Fatal("need a designated taker and a bench")
	}
	sub := e.HomeBench[0]
	replaced := false
	for i, p := range e.HomeStarters {
		if p.PlayerID == primary.PlayerID {
			e.HomeStarters[i] = sub
			replaced = true
			break
		}
	}
	if !replaced {
		t.Fatal("could not substitute primary taker")
	}
	before := len(e.Events)
	e.ResolvePenalty(e.HomeClub, e.HomeStarters)
	if len(e.Events) <= before {
		t.Fatal("penalty produced no event")
	}
	last := e.Events[len(e.Events)-1]
	if last.PlayerID == primary.PlayerID {
		t.Fatalf("substituted primary taker %s still took the penalty", primary.PlayerID)
	}
}
