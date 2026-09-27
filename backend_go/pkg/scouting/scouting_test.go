package scouting

import (
	"reflect"
	"testing"

	"football_sim/pkg/models"
)

func club(id, name, short, league, country string) *models.Club {
	return &models.Club{ClubID: id, ClubName: name, ShortName: short, League: league, Country: country}
}

func player(id, pos string, age, ovr int) *models.Player {
	return &models.Player{PlayerID: id, FullName: id, Position: pos, Category: "MID", Age: age, OVR: ovr, ClubID: "C-OTHER"}
}

func TestProjectedCeilingIsClampedAndMonotonicInAge(t *testing.T) {
	if got := ProjectedCeiling(17, 95); got != 99 {
		t.Fatalf("ceiling must clamp at 99, got %d", got)
	}
	if got := ProjectedCeiling(35, 88); got != 88 {
		t.Fatalf("veterans are at their peak, got %d", got)
	}
	younger := ProjectedCeiling(19, 70)
	older := ProjectedCeiling(26, 70)
	if younger <= older {
		t.Fatalf("younger players must project more headroom: %d vs %d", younger, older)
	}
}

func TestGenerateReportUsesTrackedPotentialOverProjection(t *testing.T) {
	p := player("P-1", "MID", 17, 72)
	c := club("C-OTHER", "Other FC", "OFC", "La Liga", "Spain")
	tracked := GenerateReport(p, c, 99, true)
	if tracked.PotentialCeiling != 99 || tracked.CeilingDelta != 27 {
		t.Fatalf("tracked potential must win: %+v", tracked)
	}
	if tracked.Region != "Spain" {
		t.Fatalf("region must come from the club country, got %q", tracked.Region)
	}
	projected := GenerateReport(p, c, 0, false)
	if projected.PotentialCeiling != ProjectedCeiling(17, 72) {
		t.Fatalf("untracked potential must project, got %d", projected.PotentialCeiling)
	}
}

func TestConsistencyRewardsTightRatingBands(t *testing.T) {
	steady := &models.Player{RecentRatings: []float64{7.0, 7.1, 7.0, 7.1}}
	volatile := &models.Player{RecentRatings: []float64{9.5, 4.0, 9.0, 3.5}}
	cs, cv := consistencyFromRatings(steady), consistencyFromRatings(volatile)
	if cs <= cv {
		t.Fatalf("steady ratings must outscore volatile ones: %d vs %d", cs, cv)
	}
	if cs < 80 {
		t.Fatalf("tight band should be highly consistent, got %d", cs)
	}
}

func TestRiskAccumulatesConcreteFactors(t *testing.T) {
	risky := &models.Player{
		Age: 17, ContractYears: 1, TransferRequested: true,
		Loyalty: 20, InjuredMatches: 4, Appearances: 2, RecentRatings: []float64{9.0, 3.0, 9.0, 3.0},
	}
	r := GenerateReport(risky, club("C-OTHER", "Other FC", "OFC", "Serie A", "Italy"), 0, false)
	if r.Risk < 70 {
		t.Fatalf("kitchen-sink risk profile must score high, got %d", r.Risk)
	}
	if len(r.RiskFactors) < 4 {
		t.Fatalf("risk factors must be itemised, got %v", r.RiskFactors)
	}
	safe := &models.Player{Age: 26, ContractYears: 3, Loyalty: 80, Appearances: 30, RecentRatings: []float64{7, 7, 7}}
	sr := GenerateReport(safe, club("C-OTHER", "Other FC", "OFC", "Serie A", "Italy"), 0, false)
	if sr.Risk >= 70 {
		t.Fatalf("stable profile must score low risk, got %d", sr.Risk)
	}
}

func TestShortlistIsDeterministicSortedAndExcludesOwnClub(t *testing.T) {
	own := club("C-MINE", "My FC", "MFC", "Premier League", "England")
	other := club("C-OTHER", "Other FC", "OFC", "La Liga", "Spain")
	candidates := []Candidate{
		{Player: &models.Player{PlayerID: "P-B", Age: 20, OVR: 78, ClubID: "C-OTHER"}, Club: other},
		{Player: &models.Player{PlayerID: "P-A", Age: 20, OVR: 78, ClubID: "C-OTHER"}, Club: other},
		{Player: &models.Player{PlayerID: "P-OWN", Age: 20, OVR: 90, ClubID: "C-MINE"}, Club: own},
		{Player: &models.Player{PlayerID: "P-C", Age: 30, OVR: 84, ClubID: "C-OTHER"}, Club: other},
	}
	first := Shortlist("C-MINE", candidates, 2)
	second := Shortlist("C-MINE", []Candidate{candidates[2], candidates[0], candidates[3], candidates[1]}, 2)
	if !reflect.DeepEqual(first, second) {
		t.Fatal("shortlist must not depend on candidate input order")
	}
	if len(first) != 2 {
		t.Fatalf("limit must be respected, got %d entries", len(first))
	}
	for _, r := range first {
		if r.ClubID == "C-MINE" {
			t.Fatalf("shortlist must exclude the buying club, got %s", r.PlayerID)
		}
	}
	for i := 1; i < len(first); i++ {
		if first[i-1].ScoutScore < first[i].ScoutScore {
			t.Fatal("shortlist must be sorted by scout score descending")
		}
	}
	// Equal-score players break ties by player ID.
	if first[0].ScoutScore == first[1].ScoutScore && first[0].PlayerID > first[1].PlayerID {
		t.Fatalf("ties must break by player ID, got %s before %s", first[0].PlayerID, first[1].PlayerID)
	}
}

func TestShortlistDefaultsAndCapsLimit(t *testing.T) {
	candidates := []Candidate{}
	for i := 0; i < 30; i++ {
		candidates = append(candidates, Candidate{
			Player: &models.Player{PlayerID: string(rune('A' + i)), Age: 22, OVR: 70, ClubID: "C-OTHER"},
			Club:   club("C-OTHER", "Other FC", "OFC", "Ligue 1", "France"),
		})
	}
	if got := len(Shortlist("C-MINE", candidates, 0)); got != 12 {
		t.Fatalf("zero limit must default to 12, got %d", got)
	}
	if got := len(Shortlist("C-MINE", candidates, 500)); got != 30 {
		t.Fatalf("limit above the pool must return the pool, got %d", got)
	}
}
