package models

import "testing"

func TestCompetitionImportanceOrdersBigNightsAboveEarlyCups(t *testing.T) {
	if CompetitionImportance("fa-cup", 2) >= CompetitionImportance("champions-league", 34) {
		t.Fatal("early FA Cup should be less important than a UCL knockout")
	}
	if CompetitionImportance("premier-league", 10) <= CompetitionImportance("fa-cup", 2) {
		t.Fatal("league nights should outrank early cup ties")
	}
}
