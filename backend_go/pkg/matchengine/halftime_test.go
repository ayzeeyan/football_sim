package matchengine

import "testing"

func TestHalfTimePauseAndInstantBypass(t *testing.T) {
	e := chunk3LiveEngine(2026)
	e.CurrentMinute = 44.9
	e.Update(1)
	if e.State != "HALF_TIME" || e.CurrentMinute != 45 {
		t.Fatalf("did not pause at half: %s %v", e.State, e.CurrentMinute)
	}
	for i := 0; i < 10; i++ {
		e.Update(1)
		e.Kickoff()
		e.TogglePause()
	}
	e.SetSpeed(999)
	e.Update(1)
	if e.State != "HALF_TIME" || e.CurrentMinute != 45 {
		t.Fatal("pause bypassed")
	}
	if !e.ResumeHalfTime("home", "", "", "") {
		t.Fatal("resume rejected")
	}
	e.Update(1)
	if e.State != "FULL_TIME" {
		t.Fatal("instant after resume did not finish")
	}
	e.ResetMatch()
	e.SetSpeed(1)
	e.Kickoff()
	e.CurrentMinute = 44.9
	e.Update(1)
	if e.State != "HALF_TIME" {
		t.Fatal("reset failed to restore half time")
	}
	e.ResetMatch()
	e.Kickoff()
	e.SetSpeed(999)
	e.Update(1)
	if e.State != "FULL_TIME" {
		t.Fatal("999 must bypass half time")
	}
}

func TestHalfTimeChoicesAndSecondHalf(t *testing.T) {
	for _, side := range []string{"home", "away"} {
		for _, stance := range []string{"OVERLOAD", "PARK_BUS"} {
			t.Run(side+stance, func(t *testing.T) {
				e := chunk3LiveEngine(2026)
				out, in := e.starters(side)[5], e.bench(side)[0]
				// A 45' automatic sub must not run ahead of the user's choice.
				e.PlannedSubs = []PlannedSub{{Minute: 45, Side: side, Out: out, In: in}}
				e.CurrentMinute = 44.9
				e.Update(1)
				if e.SubstitutionsMade[side] != 0 {
					t.Fatal("automatic sub ran before dugout")
				}
				if !e.ResumeHalfTime(side, stance, out.PlayerID, in.PlayerID) {
					t.Fatal("valid choice rejected")
				}
				if e.ResumeHalfTime(side, stance, out.PlayerID, in.PlayerID) {
					t.Fatal("duplicate accepted")
				}
				e.Update(1)
				if e.State != "PLAYING" || e.CurrentMinute <= 45 || e.stance(side) != stance || e.starters(side)[5] != in || e.SubstitutionsMade[side] != 1 {
					t.Fatal("second half lost choice")
				}
				if !e.plannedInUse(in.PlayerID) {
					t.Fatal("AI could reuse substitute")
				}
				for i := 0; i < 500 && e.State != "FULL_TIME"; i++ {
					e.Update(1)
				}
				if e.State != "FULL_TIME" {
					t.Fatal("second half did not finish")
				}
				seen := map[string]bool{}
				for _, p := range e.starters(side) {
					if seen[p.PlayerID] {
						t.Fatal("duplicate XI player")
					}
					seen[p.PlayerID] = true
				}
			})
		}
	}
}

func TestHalfTimeInvalidChoiceIsAtomic(t *testing.T) {
	for _, scenario := range []string{"side", "stance", "missing", "not_active", "not_bench", "sent_off", "used", "limit"} {
		t.Run(scenario, func(t *testing.T) {
			e := chunk3LiveEngine(2026)
			e.CurrentMinute = 44.9
			e.Update(1)
			side, stance := "home", "OVERLOAD"
			out, in := e.HomeStarters[5].PlayerID, e.HomeBench[0].PlayerID
			switch scenario {
			case "side":
				side = "invalid"
			case "stance":
				stance = "invalid"
			case "missing":
				in = ""
			case "not_active":
				out = "invalid"
			case "not_bench":
				in = e.HomeStarters[6].PlayerID
			case "sent_off":
				e.Bookings[out] = 2
			case "used":
				e.ExecuteDirectSub("home", e.HomeStarters[6], e.HomeBench[0], 40, "test")
				e.ExecuteDirectSub("home", e.HomeBench[0], e.HomeBench[1], 42, "test")
			case "limit":
				e.SubstitutionsMade["home"] = 5
			}
			before := len(e.Events)
			if e.ResumeHalfTime(side, stance, out, in) || e.State != "HALF_TIME" || e.HomeStance != "NORMAL" || len(e.Events) != before {
				t.Fatal("invalid choice changed match")
			}
		})
	}
}
