package matchreport

import "testing"

func TestDetectStoryFactsComebackAndLateWinner(t *testing.T) {
	scorer := MiniPlayer{PlayerID: "P1", FullName: "Late Hero"}
	away := MiniPlayer{PlayerID: "P2", FullName: "Opener"}
	events := []MatchEventItem{
		{Minute: 12, Type: "goal", Side: "away", Scorer: &away, PlayerID: "P2", PlayerName: "Opener"},
		{Minute: 18, Type: "goal", Side: "away", Scorer: &away, PlayerID: "P2", PlayerName: "Opener"},
		{Minute: 61, Type: "goal", Side: "home", Scorer: &scorer, PlayerID: "P1", PlayerName: "Late Hero"},
		{Minute: 70, Type: "goal", Side: "home", Scorer: &scorer, PlayerID: "P1", PlayerName: "Late Hero"},
		{Minute: 88, Type: "goal", Side: "home", Scorer: &scorer, PlayerID: "P1", PlayerName: "Late Hero"},
		{Minute: 40, Type: "red", Side: "away", Player: &away, PlayerID: "P2", PlayerName: "Opener"},
	}

	facts := DetectStoryFacts(events, 3, 2, nil)
	kinds := map[string]int{}
	for _, f := range facts {
		kinds[f.Kind]++
	}
	for _, want := range []string{"opening_goal", "equalizer", "late_winner", "two_goal_comeback", "hat_trick", "red_card"} {
		if kinds[want] == 0 {
			t.Fatalf("missing story fact %s in %+v", want, facts)
		}
	}
}

func TestDetectStoryFactsPenaltyAndCleanSheet(t *testing.T) {
	scorer := MiniPlayer{PlayerID: "ST", FullName: "Striker"}
	events := []MatchEventItem{
		{Minute: 22, Type: "penalty", Side: "home", Scorer: &scorer, PlayerID: "ST", PlayerName: "Striker"},
	}
	facts := DetectStoryFacts(events, 1, 0, nil)
	kinds := map[string]bool{}
	for _, f := range facts {
		kinds[f.Kind] = true
	}
	if !kinds["penalty"] || !kinds["opening_goal"] || !kinds["clean_sheet"] {
		t.Fatalf("facts=%+v", facts)
	}
}

func TestDetectStoryFactsMultiGoalLead(t *testing.T) {
	scorer := MiniPlayer{PlayerID: "ST", FullName: "Striker"}
	events := []MatchEventItem{
		{Minute: 10, Type: "goal", Side: "home", Scorer: &scorer, PlayerID: "ST", PlayerName: "Striker"},
		{Minute: 20, Type: "goal", Side: "home", Scorer: &scorer, PlayerID: "ST", PlayerName: "Striker"},
		{Minute: 70, Type: "goal", Side: "home", Scorer: &scorer, PlayerID: "ST", PlayerName: "Striker"},
	}
	facts := DetectStoryFacts(events, 3, 0, nil)
	kinds := map[string]bool{}
	for _, f := range facts {
		kinds[f.Kind] = true
	}
	if !kinds["multi_goal_lead"] || !kinds["hat_trick"] || !kinds["clean_sheet"] {
		t.Fatalf("facts=%+v", facts)
	}
}
