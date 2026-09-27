package tournament

import (
	"testing"
)

func TestSearchWorldFindsClubsPlayersAndCompetitions(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	res := tm.SearchWorld("madrid", 8)
	clubs, _ := res["clubs"].([]map[string]interface{})
	if len(clubs) == 0 {
		t.Fatal("expected Real Madrid in club search")
	}
	foundMadrid := false
	for _, club := range clubs {
		if club["club_id"] == "LAL-RMA" || club["short_name"] == "RMA" {
			foundMadrid = true
		}
	}
	if !foundMadrid {
		t.Fatalf("madrid search missed Real Madrid: %v", clubs)
	}

	players := tm.SearchWorld("guinita", 8)
	hits, _ := players["players"].([]map[string]interface{})
	if len(hits) == 0 {
		t.Fatal("expected wonderkid name search to return Jhed Anthony Guinita")
	}

	comps := tm.SearchWorld("champions", 8)
	compHits, _ := comps["competitions"].([]map[string]interface{})
	if len(compHits) == 0 {
		t.Fatal("expected Champions League in competition search")
	}
}

func TestSearchWorldDirectoryIsDeterministicAndFavouriteIndependent(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	a := tm.SearchWorld("", 12)
	if !tm.SetFavouriteClubID("EPL-ARS") {
		t.Fatal("could not set viewing preference")
	}
	b := tm.SearchWorld("", 12)
	playersA, _ := a["players"].([]map[string]interface{})
	playersB, _ := b["players"].([]map[string]interface{})
	if len(playersA) == 0 || len(playersA) != len(playersB) {
		t.Fatalf("directory size drifted with favourite club: %d vs %d", len(playersA), len(playersB))
	}
	if playersA[0]["player_id"] != playersB[0]["player_id"] {
		t.Fatalf("directory order changed with favourite club: %v vs %v", playersA[0]["player_id"], playersB[0]["player_id"])
	}
	first := playersA[0]
	second := playersA[1]
	ovr0, _ := first["ovr"].(int)
	ovr1, _ := second["ovr"].(int)
	if ovr0 < ovr1 {
		t.Fatalf("directory is not OVR-desc: %v then %v", first, second)
	}
}
