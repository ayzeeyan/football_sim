package tournament

import (
	"encoding/json"
	"testing"
)

// TestScoutingShortlistReadOnlyAndDeterministic pins the two contracts of
// the recruitment desk: the world is untouched, and the same inputs always
// produce the same shortlist.
func TestScoutingShortlistReadOnlyAndDeterministic(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	tm.mu.RLock()
	clubID := ""
	for _, id := range sortedClubIDs(tm.ClubsList) {
		clubID = id
		break
	}
	tm.mu.RUnlock()
	if clubID == "" {
		t.Fatal("no club in world")
	}

	before := sandboxSnapshot(t, tm, "")
	first := tm.ScoutingShortlist(clubID, 12)
	second := tm.ScoutingShortlist(clubID, 12)
	after := sandboxSnapshot(t, tm, "")

	if string(before) != string(after) {
		t.Fatal("scouting shortlist mutated the real universe")
	}
	b1, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("marshal first: %v", err)
	}
	b2, err := json.Marshal(second)
	if err != nil {
		t.Fatalf("marshal second: %v", err)
	}
	if string(b1) != string(b2) {
		t.Fatal("scouting shortlist is not deterministic")
	}

	if len(first) == 0 {
		t.Fatal("shortlist must not be empty in a 96-club world")
	}
	if len(first) > 12 {
		t.Fatalf("shortlist must respect the limit, got %d", len(first))
	}
	for i, r := range first {
		if r.ClubID == clubID {
			t.Fatalf("shortlist entry %s belongs to the buying club", r.PlayerID)
		}
		if r.Region == "" || r.League == "" {
			t.Fatalf("shortlist entry %s missing region/league: %+v", r.PlayerID, r)
		}
		if r.PotentialCeiling < r.OVR {
			t.Fatalf("ceiling below current OVR for %s: %+v", r.PlayerID, r)
		}
		if i > 0 && first[i-1].ScoutScore < r.ScoutScore {
			t.Fatal("shortlist must be sorted by scout score")
		}
	}
}

// TestScoutingShortlistUnknownClub verifies the not-found contract.
func TestScoutingShortlistUnknownClub(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	if got := tm.ScoutingShortlist("NO-SUCH-CLUB", 12); got != nil {
		t.Fatalf("unknown club must return nil, got %d entries", len(got))
	}
}
