package models

import "testing"

func TestGetPositionCategoryWideMidfielders(t *testing.T) {
	tests := map[string]string{
		"LM":  "MID",
		"RM":  "MID",
		" lm ": "MID",
		"rm":  "MID",
		"LW":  "FWD",
		"RW":  "FWD",
		"CAM": "FWD",
	}

	for position, want := range tests {
		if got := GetPositionCategory(position); got != want {
			t.Fatalf("GetPositionCategory(%q) = %q, want %q", position, got, want)
		}
	}
}

func TestRecalculateRatingsClearsEmptySquadMetadata(t *testing.T) {
	club := &Club{
		SquadSize:         24,
		SquadAvgOVR:       81.7,
		OverallTeamRating: 84,
		Squad:             nil,
	}

	club.RecalculateRatings()

	if club.SquadSize != 0 {
		t.Fatalf("SquadSize = %d, want 0", club.SquadSize)
	}
	if club.SquadAvgOVR != 0 {
		t.Fatalf("SquadAvgOVR = %v, want 0", club.SquadAvgOVR)
	}
	if club.OverallTeamRating != 0 {
		t.Fatalf("OverallTeamRating = %d, want 0", club.OverallTeamRating)
	}
}
