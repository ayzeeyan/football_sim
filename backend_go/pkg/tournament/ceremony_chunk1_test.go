package tournament

import "testing"

func TestAwardsCeremonyWinnerIndependentFromDisplayOrder(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	ceremony := tm.GetAwardsCeremony()
	categories, ok := ceremony["categories"].([]map[string]interface{})
	if !ok || len(categories) == 0 {
		t.Fatalf("missing ceremony categories: %#v", ceremony["categories"])
	}

	foundNonFirstWinner := false
	for _, category := range categories {
		winnerID, _ := category["winner_id"].(string)
		nominees, _ := category["nominees"].([]map[string]interface{})
		if winnerID == "" || len(nominees) == 0 {
			continue
		}
		found := false
		for i, nominee := range nominees {
			if nominee["player_id"] == winnerID {
				found = true
				if i != 0 {
					foundNonFirstWinner = true
				}
				break
			}
		}
		if !found {
			t.Fatalf("category %v winner %q is not in nominee set", category["key"], winnerID)
		}
	}
	if !foundNonFirstWinner {
		t.Fatal("deterministic ceremony presentation still placed every winner first")
	}
}

func TestAwardsCeremonyDeterministicAndGoldenBoyU21(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	first := tm.GetAwardsCeremony()
	second := tm.GetAwardsCeremony()
	firstCats, _ := first["categories"].([]map[string]interface{})
	secondCats, _ := second["categories"].([]map[string]interface{})
	if len(firstCats) != len(secondCats) {
		t.Fatalf("category count changed %d -> %d", len(firstCats), len(secondCats))
	}
	for i := range firstCats {
		if firstCats[i]["winner_id"] != secondCats[i]["winner_id"] {
			t.Fatalf("winner changed for %v", firstCats[i]["key"])
		}
		if firstCats[i]["key"] != "golden_boy" {
			continue
		}
		nominees, _ := firstCats[i]["nominees"].([]map[string]interface{})
		for _, nominee := range nominees {
			age, _ := nominee["age"].(int)
			isWK, _ := nominee["is_wonderkid"].(bool)
			if !isWK || age > 21 {
				t.Fatalf("invalid Golden Boy nominee: %#v", nominee)
			}
		}
	}
}
