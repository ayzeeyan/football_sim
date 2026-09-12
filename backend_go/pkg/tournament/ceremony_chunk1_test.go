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

func TestTeamOfTheSeasonRigid1433And11UniquePlayers(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	awards := tm.GetSeasonAwards()
	tots, ok := awards["team_of_the_season"].(map[string]interface{})
	if !ok || tots == nil {
		t.Fatalf("missing team_of_the_season in GetSeasonAwards(): %#v", awards["team_of_the_season"])
	}

	formation, _ := tots["formation"].(string)
	if formation != "4-3-3" {
		t.Fatalf("expected formation 4-3-3, got %q", formation)
	}

	slots := []string{"gk", "lb", "cb1", "cb2", "rb", "mid1", "mid2", "mid3", "fwd1", "fwd2", "fwd3"}
	usedIDs := make(map[string]bool)
	for _, slot := range slots {
		card, ok := tots[slot].(map[string]interface{})
		if !ok || card == nil {
			t.Fatalf("slot %q missing or nil", slot)
		}
		pid, _ := card["player_id"].(string)
		if pid == "" {
			t.Fatalf("slot %q missing player_id", slot)
		}
		if usedIDs[pid] {
			t.Fatalf("duplicate player %q in TOTS slot %q", pid, slot)
		}
		usedIDs[pid] = true
	}

	xi, ok := tots["xi"].([]map[string]interface{})
	if !ok || len(xi) != 11 {
		t.Fatalf("expected 11 cards in xi, got %d", len(xi))
	}
	if len(usedIDs) != 11 {
		t.Fatalf("expected 11 unique players across TOTS, got %d", len(usedIDs))
	}
}

func TestManagerOfTheYearDeterministicScoringAndAccolade(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	moty1 := tm.managerOfTheYearUnlocked()
	moty2 := tm.managerOfTheYearUnlocked()

	if moty1 == nil || moty2 == nil {
		t.Fatal("managerOfTheYearUnlocked() returned nil")
	}
	if moty1["club_id"] != moty2["club_id"] || moty1["name"] != moty2["name"] {
		t.Fatalf("moty non-deterministic: %+v vs %+v", moty1, moty2)
	}

	pts, _ := moty1["pts"].(int)
	outperformed, _ := moty1["outperformed_places"].(int)
	trophies, _ := moty1["trophies_won"].(int)
	score, _ := moty1["score"].(float64)
	expectedScore := round2(float64(outperformed)*10.0 + float64(pts)*0.5 + float64(trophies)*25.0)
	if score != expectedScore {
		t.Fatalf("moty score mismatch: got %v, expected %v", score, expectedScore)
	}

	accolade, _ := moty1["accolade"].(string)
	if accolade == "" {
		t.Fatal("expected non-empty accolade for manager of the year")
	}
}

func TestAwardsCeremonyReadyGuards(t *testing.T) {
	tm, _ := loadTestUniverse(t)
	tm.CurrentMatchweek = 1
	if tm.AwardsCeremonyReady() {
		t.Fatal("ceremony should NOT be ready at matchweek 1")
	}

	tm.CurrentMatchweek = tm.MaxMatchweeks + 1
	if !tm.AwardsCeremonyReady() {
		t.Fatal("ceremony SHOULD be ready when matchweek > max_matchweeks")
	}

	tm.ResetNewSeason()
	if tm.AwardsCeremonyReady() {
		t.Fatal("ceremony should NOT be ready after ResetNewSeason")
	}
}
