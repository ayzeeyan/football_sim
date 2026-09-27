package matchreport

import "testing"

func TestNormalizeEventFactsClockAndAssist(t *testing.T) {
	scorer := MiniPlayer{PlayerID: "P1", FullName: "Serge Gnabry"}
	assist := MiniPlayer{PlayerID: "P2", FullName: "Harry Kane"}
	events := []MatchEventItem{
		{
			Minute:   43,
			Type:     "goal",
			Side:     "home",
			Scorer:   &scorer,
			Assister: &assist,
			Display:  "Serge Gnabry 43' (Assist: Harry Kane)",
		},
		{
			Minute:  92,
			Type:    "goal",
			Side:    "away",
			Scorer:  &MiniPlayer{PlayerID: "P3", FullName: "Karim Adeyemi"},
			Display: "Karim Adeyemi 90+2'",
		},
		{
			Minute:  74,
			Type:    "yellow",
			Side:    "home",
			Player:  &MiniPlayer{PlayerID: "P4", FullName: "Joshua Kimmich"},
			Display: "Yellow Card: Joshua Kimmich 74'",
		},
	}

	NormalizeEventFacts(events)

	if events[0].Display != "43'" {
		t.Fatalf("goal display = %q", events[0].Display)
	}
	if events[0].PlayerID != "P1" || events[0].PlayerName != "Serge Gnabry" {
		t.Fatalf("scorer identity = %+v", events[0])
	}
	if events[0].AssistPlayerID != "P2" || events[0].AssistPlayerName != "Harry Kane" {
		t.Fatalf("assist identity = %+v", events[0])
	}
	if events[1].Display != "90+2'" {
		t.Fatalf("stoppage display = %q", events[1].Display)
	}
	if events[2].Display != "74'" || events[2].PlayerName != "Joshua Kimmich" {
		t.Fatalf("card identity = %+v", events[2])
	}

	NormalizeEventFacts(events)
	if events[0].Display != "43'" || events[0].AssistPlayerName != "Harry Kane" {
		t.Fatal("normalize is not idempotent")
	}
}

func TestAssignEventClubsFillsMissingSide(t *testing.T) {
	events := []MatchEventItem{
		{Minute: 12, Type: "goal", Side: "home", PlayerName: "Harry Kane"},
		{Minute: 67, Type: "goal", Side: "away", PlayerName: "Adeyemi", ClubID: "BUN-DOR", ClubName: "Borussia Dortmund"},
	}
	AssignEventClubs(events, "BUN-BAY", "Bayern Munich", "BUN-DOR", "Borussia Dortmund")
	if events[0].ClubID != "BUN-BAY" || events[0].ClubName != "Bayern Munich" {
		t.Fatalf("home club = %+v", events[0])
	}
	if events[1].ClubID != "BUN-DOR" || events[1].ClubName != "Borussia Dortmund" {
		t.Fatalf("existing club should stay: %+v", events[1])
	}
}

func TestClockOnlyDisplayExtractsStamp(t *testing.T) {
	if got := ClockOnlyDisplay(51, "Christos Tzolis 51'"); got != "51'" {
		t.Fatalf("got %q", got)
	}
	if got := ClockOnlyDisplay(67, "Substitution 67' (A ➜ B)"); got != "67'" {
		t.Fatalf("got %q", got)
	}
	if got := ClockOnlyDisplay(16, ""); got != "16'" {
		t.Fatalf("got %q", got)
	}
}
