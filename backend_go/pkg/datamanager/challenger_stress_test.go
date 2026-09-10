package datamanager

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"football_sim/pkg/growth"
	"football_sim/pkg/models"
)

// ============================================================================
// CHALLENGER TEST SUITE 1: DUPLICATE INJECTION STRESS
// ============================================================================

func TestChallenger_DuplicateInjection_MultiClubAndIntraClub(t *testing.T) {
	ge := growth.NewGrowthEngine(101)
	dm := NewDataManager("dataset.json", ge)
	if err := dm.LoadDataset(); err != nil {
		t.Fatalf("LoadDataset failed: %v", err)
	}

	_ = dm.DedupePlayers()
	if leftover := dm.DedupePlayers(); leftover != 0 {
		t.Errorf("second baseline dedupe should be idle, got %d", leftover)
	}

	// 1. Multi-way duplicate across 4 clubs (1 elite, 3 non-elite)
	// Player: "Alexandre Cross" (not in PREFERRED_HOMES)
	ars := dm.Clubs["EPL-ARS"] // elite
	ful := dm.Clubs["EPL-FUL"] // non-elite
	cry := dm.Clubs["EPL-CRY"] // non-elite
	get := dm.Clubs["LAL-GET"] // non-elite
	if ars == nil || ful == nil || cry == nil || get == nil {
		t.Fatal("target clubs not found")
	}

	pArs := &models.Player{PlayerID: "P_AC_ARS", FullName: "Alexandre Cross", Position: "CM", OVR: 75, Appearances: 2, Goals: 1, Assists: 1, ClubID: "EPL-ARS"}
	pFul := &models.Player{PlayerID: "P_AC_FUL", FullName: "Alexandre Cross", Position: "CM", OVR: 76, Appearances: 10, Goals: 5, Assists: 4, CareerGoals: 20, ClubID: "EPL-FUL"}
	pCry := &models.Player{PlayerID: "P_AC_CRY", FullName: "Alexandre Cross", Position: "CM", OVR: 74, Appearances: 5, Goals: 2, Assists: 2, ClubID: "EPL-CRY"}
	pGet := &models.Player{PlayerID: "P_AC_GET", FullName: "Alexandre Cross", Position: "CM", OVR: 73, Appearances: 1, Goals: 0, Assists: 0, ClubID: "LAL-GET"}

	ars.Squad = append(ars.Squad, pArs)
	ful.Squad = append(ful.Squad, pFul)
	cry.Squad = append(cry.Squad, pCry)
	get.Squad = append(get.Squad, pGet)
	ars.SquadSize = len(ars.Squad)
	ful.SquadSize = len(ful.Squad)
	cry.SquadSize = len(cry.Squad)
	get.SquadSize = len(get.Squad)

	// 2. Intra-club duplicate (3 copies inside Crystal Palace)
	pIntra1 := &models.Player{PlayerID: "P_INTRA_1", FullName: "Intra Triple Player", Position: "CB", OVR: 72, Appearances: 5, Goals: 1, ClubID: "EPL-CRY"}
	pIntra2 := &models.Player{PlayerID: "P_INTRA_2", FullName: "Intra Triple Player", Position: "CB", OVR: 74, Appearances: 8, Goals: 2, ClubID: "EPL-CRY"}
	pIntra3 := &models.Player{PlayerID: "P_INTRA_3", FullName: "Intra Triple Player", Position: "CB", OVR: 71, Appearances: 3, Goals: 0, ClubID: "EPL-CRY"}
	cry.Squad = append(cry.Squad, pIntra1, pIntra2, pIntra3)
	cry.SquadSize = len(cry.Squad)

	// 3. Preferred Homes duplicate across 5 clubs
	// "Cole Palmer" has preferred home "EPL-CHE"
	che := dm.Clubs["EPL-CHE"]
	liv := dm.Clubs["EPL-LIV"]
	mci := dm.Clubs["EPL-MCI"]
	bar := dm.Clubs["LAL-BAR"]
	om := dm.Clubs["FL1-OM"]
	if che == nil || liv == nil || mci == nil || bar == nil || om == nil {
		t.Fatal("clubs for preferred home test not found")
	}

	pChe := &models.Player{PlayerID: "P_CP_CHE", FullName: "Cole Palmer", Position: "CAM", OVR: 86, Appearances: 12, Goals: 10, Assists: 8, CareerGoals: 30, ClubID: "EPL-CHE"}
	pLiv := &models.Player{PlayerID: "P_CP_LIV", FullName: "Cole Palmer", Position: "CAM", OVR: 86, Appearances: 5, Goals: 3, Assists: 2, CareerGoals: 40, ClubID: "EPL-LIV"}
	pMci := &models.Player{PlayerID: "P_CP_MCI", FullName: "Cole Palmer", Position: "CAM", OVR: 85, Appearances: 15, Goals: 8, Assists: 12, ClubID: "EPL-MCI"}
	pBar := &models.Player{PlayerID: "P_CP_BAR", FullName: "Cole Palmer", Position: "CAM", OVR: 86, Appearances: 1, Goals: 1, Assists: 0, ClubID: "LAL-BAR"}
	pOM := &models.Player{PlayerID: "P_CP_OM", FullName: "Cole Palmer", Position: "CAM", OVR: 84, Appearances: 0, Goals: 0, Assists: 0, ClubID: "FL1-OM"}

	che.Squad = append(che.Squad, pChe)
	liv.Squad = append(liv.Squad, pLiv)
	mci.Squad = append(mci.Squad, pMci)
	bar.Squad = append(bar.Squad, pBar)
	om.Squad = append(om.Squad, pOM)
	che.SquadSize = len(che.Squad)
	liv.SquadSize = len(liv.Squad)
	mci.SquadSize = len(mci.Squad)
	bar.SquadSize = len(bar.Squad)
	om.SquadSize = len(om.Squad)

	// 4. Duplicate Wonderkid injection across 3 clubs
	// "Venjamin Valerio" belongs to LAL-BAR
	rma := dm.Clubs["LAL-RMA"]
	juv := dm.Clubs["SEA-INT"]
	if rma == nil || juv == nil {
		t.Fatal("rma or juv not found")
	}
	pWkBar := &models.Player{PlayerID: "WK_Venjamin_Valerio", FullName: "Venjamin Valerio", Position: "ST", OVR: 78, Age: 14, UniverseWonderkid: true, ClubID: "LAL-BAR"}
	pWkRma := &models.Player{PlayerID: "P_DUP_VV_RMA", FullName: "Venjamin Valerio", Position: "ST", OVR: 76, Age: 15, ClubID: "LAL-RMA"}
	pWkJuv := &models.Player{PlayerID: "P_DUP_VV_JUV", FullName: "Venjamin Valerio", Position: "ST", OVR: 77, Age: 14, ClubID: "SEA-INT"}
	bar.Squad = append(bar.Squad, pWkBar)
	rma.Squad = append(rma.Squad, pWkRma)
	juv.Squad = append(juv.Squad, pWkJuv)
	bar.SquadSize = len(bar.Squad)
	rma.SquadSize = len(rma.Squad)
	juv.SquadSize = len(juv.Squad)

	// Total injected duplicates:
	// Alexandre Cross: 4 copies -> 3 should be removed
	// Intra Triple Player: 3 copies -> 2 should be removed
	// Cole Palmer: 5 copies -> 4 should be removed
	// Venjamin Valerio: 3 copies -> 2 should be removed
	// Total to remove = 3 + 2 + 4 + 2 = 11 duplicates

	removed := dm.DedupePlayers()
	if removed != 13 {
		t.Errorf("expected 13 duplicates removed (including existing dataset copies of Cole Palmer and Venjamin Valerio), got %d", removed)
	}

	// Assertions for Alexandre Cross:
	// Arsenal is the sole elite club among the 4, so Arsenal must keep him
	var arsHasAC, fulHasAC, cryHasAC, getHasAC bool
	var keptAC *models.Player
	for _, p := range ars.Squad {
		if strings.EqualFold(p.FullName, "Alexandre Cross") {
			arsHasAC = true
			keptAC = p
		}
	}
	for _, p := range ful.Squad {
		if strings.EqualFold(p.FullName, "Alexandre Cross") {
			fulHasAC = true
		}
	}
	for _, p := range cry.Squad {
		if strings.EqualFold(p.FullName, "Alexandre Cross") {
			cryHasAC = true
		}
	}
	for _, p := range get.Squad {
		if strings.EqualFold(p.FullName, "Alexandre Cross") {
			getHasAC = true
		}
	}

	if !arsHasAC {
		t.Error("Alexandre Cross was not retained in Arsenal (elite priority)")
	}
	if fulHasAC || cryHasAC || getHasAC {
		t.Errorf("Alexandre Cross duplicate survived: ful=%v, cry=%v, get=%v", fulHasAC, cryHasAC, getHasAC)
	}
	if keptAC != nil {
		if keptAC.CareerGoals != 20 {
			t.Errorf("Alexandre Cross CareerGoals not max-merged: got %d, expected 20", keptAC.CareerGoals)
		}
		if keptAC.Appearances != 10 {
			t.Errorf("Alexandre Cross Appearances not max-merged: got %d, expected 10", keptAC.Appearances)
		}
		if keptAC.ClubID != "EPL-ARS" {
			t.Errorf("Alexandre Cross ClubID = %q, expected 'EPL-ARS'", keptAC.ClubID)
		}
	}

	// Assertions for Intra Triple Player:
	intraCount := 0
	for _, p := range cry.Squad {
		if strings.EqualFold(p.FullName, "Intra Triple Player") {
			intraCount++
		}
	}
	if intraCount != 1 {
		t.Errorf("expected exactly 1 copy of Intra Triple Player in Crystal Palace, got %d", intraCount)
	}

	// Assertions for Cole Palmer:
	// Must be kept at EPL-CHE (preferred home), absent from LIV, MCI, BAR, OM
	var cheHasCP, livHasCP, mciHasCP, barHasCP, omHasCP bool
	var keptCP *models.Player
	for _, p := range che.Squad {
		if strings.EqualFold(p.FullName, "Cole Palmer") {
			cheHasCP = true
			keptCP = p
		}
	}
	for _, p := range liv.Squad {
		if strings.EqualFold(p.FullName, "Cole Palmer") {
			livHasCP = true
		}
	}
	for _, p := range mci.Squad {
		if strings.EqualFold(p.FullName, "Cole Palmer") {
			mciHasCP = true
		}
	}
	for _, p := range bar.Squad {
		if strings.EqualFold(p.FullName, "Cole Palmer") {
			barHasCP = true
		}
	}
	for _, p := range om.Squad {
		if strings.EqualFold(p.FullName, "Cole Palmer") {
			omHasCP = true
		}
	}

	if !cheHasCP {
		t.Error("Cole Palmer not kept in Chelsea (preferred home)")
	}
	if livHasCP || mciHasCP || barHasCP || omHasCP {
		t.Errorf("Cole Palmer duplicate survived: liv=%v, mci=%v, bar=%v, om=%v", livHasCP, mciHasCP, barHasCP, omHasCP)
	}
	if keptCP != nil {
		if keptCP.CareerGoals != 40 {
			t.Errorf("Cole Palmer CareerGoals not max-merged: got %d, expected 40", keptCP.CareerGoals)
		}
		if keptCP.Assists != 12 {
			t.Errorf("Cole Palmer Assists not max-merged: got %d, expected 12", keptCP.Assists)
		}
	}

	// Assertions for Venjamin Valerio:
	// Must be kept at LAL-BAR, absent from RMA, JUV
	var barHasVV, rmaHasVV, juvHasVV bool
	var keptVV *models.Player
	for _, p := range bar.Squad {
		if strings.EqualFold(p.FullName, "Venjamin Valerio") {
			barHasVV = true
			keptVV = p
		}
	}
	for _, p := range rma.Squad {
		if strings.EqualFold(p.FullName, "Venjamin Valerio") {
			rmaHasVV = true
		}
	}
	for _, p := range juv.Squad {
		if strings.EqualFold(p.FullName, "Venjamin Valerio") {
			juvHasVV = true
		}
	}

	if !barHasVV {
		t.Error("Venjamin Valerio not retained in Barcelona")
	}
	if rmaHasVV || juvHasVV {
		t.Errorf("Venjamin Valerio duplicate survived: rma=%v, juv=%v", rmaHasVV, juvHasVV)
	}
	if keptVV != nil {
		if keptVV.Age != 14 || keptVV.Education != "middle_school" || !keptVV.UniverseWonderkid {
			t.Errorf("Venjamin Valerio attributes corrupted after dedupe: %+v", keptVV)
		}
	}

	// Final check: exactly 0 duplicates across all 96 clubs
	nameSeen := make(map[string]string)
	for _, club := range dm.ClubsList {
		if club.SquadSize != len(club.Squad) {
			t.Errorf("club %s SquadSize %d != len(Squad) %d", club.ClubID, club.SquadSize, len(club.Squad))
		}
		for _, p := range club.Squad {
			norm := strings.ToLower(strings.TrimSpace(p.FullName))
			if prevClub, exists := nameSeen[norm]; exists {
				t.Fatalf("DUPLICATE VIOLATION: player %q found in %s and %s", p.FullName, prevClub, club.ClubID)
			}
			nameSeen[norm] = club.ClubID
		}
	}
}

func TestChallenger_DuplicateInjection_Stress100Duplicates(t *testing.T) {
	ge := growth.NewGrowthEngine(202)
	dm := NewDataManager("dataset.json", ge)
	if err := dm.LoadDataset(); err != nil {
		t.Fatalf("LoadDataset failed: %v", err)
	}
	dm.DedupePlayers()

	// Inject 100 duplicate players across random clubs
	rng := rand.New(rand.NewSource(42))
	injectedDuplicates := 0

	for i := 0; i < 50; i++ {
		name := fmt.Sprintf("Synthetic Injected_%d", i)
		// Pick 3 distinct clubs
		c1 := dm.ClubsList[rng.Intn(len(dm.ClubsList))]
		c2 := dm.ClubsList[rng.Intn(len(dm.ClubsList))]
		for c2 == c1 {
			c2 = dm.ClubsList[rng.Intn(len(dm.ClubsList))]
		}
		c3 := dm.ClubsList[rng.Intn(len(dm.ClubsList))]
		for c3 == c1 || c3 == c2 {
			c3 = dm.ClubsList[rng.Intn(len(dm.ClubsList))]
		}

		p1 := &models.Player{PlayerID: fmt.Sprintf("SYN_%d_1", i), FullName: name, Position: "ST", OVR: 70 + rng.Intn(10), ClubID: c1.ClubID}
		p2 := &models.Player{PlayerID: fmt.Sprintf("SYN_%d_2", i), FullName: name, Position: "ST", OVR: 70 + rng.Intn(10), ClubID: c2.ClubID}
		p3 := &models.Player{PlayerID: fmt.Sprintf("SYN_%d_3", i), FullName: name, Position: "ST", OVR: 70 + rng.Intn(10), ClubID: c3.ClubID}

		c1.Squad = append(c1.Squad, p1)
		c2.Squad = append(c2.Squad, p2)
		c3.Squad = append(c3.Squad, p3)
		c1.SquadSize = len(c1.Squad)
		c2.SquadSize = len(c2.Squad)
		c3.SquadSize = len(c3.Squad)

		// 3 copies = 2 duplicates to remove
		injectedDuplicates += 2
	}

	removed := dm.DedupePlayers()
	if removed != injectedDuplicates {
		t.Errorf("expected %d duplicates removed, got %d", injectedDuplicates, removed)
	}

	// Verify invariant: strictly 0 duplicates across all 96 clubs
	seen := make(map[string]string)
	for _, club := range dm.ClubsList {
		if club.SquadSize != len(club.Squad) {
			t.Errorf("club %s SquadSize %d != len(Squad) %d", club.ClubID, club.SquadSize, len(club.Squad))
		}
		for _, p := range club.Squad {
			norm := strings.ToLower(strings.TrimSpace(p.FullName))
			if prevClub, exists := seen[norm]; exists {
				t.Fatalf("DUPLICATE VIOLATION: player %q found in %s and %s", p.FullName, prevClub, club.ClubID)
			}
			seen[norm] = club.ClubID
		}
	}
}

// TestChallenger_DuplicateInjection_IdenticalPointerAttack tests whether DedupePlayers
// handles the case where the EXACT SAME POINTER is appended multiple times to a squad.
func TestChallenger_DuplicateInjection_IdenticalPointerAttack(t *testing.T) {
	ge := growth.NewGrowthEngine(999)
	dm := NewDataManager("dataset.json", ge)
	if err := dm.LoadDataset(); err != nil {
		t.Fatalf("LoadDataset failed: %v", err)
	}

	ars := dm.Clubs["EPL-ARS"]
	if ars == nil {
		t.Fatal("EPL-ARS not found")
	}

	p := &models.Player{
		PlayerID: "P_IDENTICAL_PTR",
		FullName: "Identical Pointer Player",
		Position: "ST",
		OVR:      75,
		ClubID:   "EPL-ARS",
	}

	// Append the EXACT SAME POINTER twice
	ars.Squad = append(ars.Squad, p, p)
	ars.SquadSize = len(ars.Squad)

	removed := dm.DedupePlayers()

	// Count how many copies remain in Arsenal squad
	count := 0
	for _, sqP := range ars.Squad {
		if strings.EqualFold(sqP.FullName, "Identical Pointer Player") {
			count++
		}
	}

	t.Logf("Identical pointer duplicate test: removed=%d, remaining count=%d", removed, count)
	if count != 1 {
		t.Errorf("BUG FOUND: expected exactly 1 copy remaining of Identical Pointer Player, got %d (removed=%d)", count, removed)
	}
}

// TestChallenger_DuplicateInjection_CrossClubSharedPointerAttack tests whether DedupePlayers
// handles the case where the SAME POINTER is shared across two different clubs.
func TestChallenger_DuplicateInjection_CrossClubSharedPointerAttack(t *testing.T) {
	ge := growth.NewGrowthEngine(998)
	dm := NewDataManager("dataset.json", ge)
	if err := dm.LoadDataset(); err != nil {
		t.Fatalf("LoadDataset failed: %v", err)
	}

	ars := dm.Clubs["EPL-ARS"]
	che := dm.Clubs["EPL-CHE"]
	if ars == nil || che == nil {
		t.Fatal("EPL-ARS or EPL-CHE not found")
	}

	sharedP := &models.Player{
		PlayerID: "P_SHARED_PTR",
		FullName: "Cross Club Shared Player",
		Position: "ST",
		OVR:      75,
	}

	// Add the exact same pointer to both Arsenal and Chelsea
	ars.Squad = append(ars.Squad, sharedP)
	che.Squad = append(che.Squad, sharedP)
	ars.SquadSize = len(ars.Squad)
	che.SquadSize = len(che.Squad)

	removed := dm.DedupePlayers()

	// Check if player still exists in both clubs
	arsHas := false
	for _, p := range ars.Squad {
		if strings.EqualFold(p.FullName, "Cross Club Shared Player") {
			arsHas = true
		}
	}
	cheHas := false
	for _, p := range che.Squad {
		if strings.EqualFold(p.FullName, "Cross Club Shared Player") {
			cheHas = true
		}
	}

	t.Logf("Cross-club shared pointer test: removed=%d, arsHas=%v, cheHas=%v", removed, arsHas, cheHas)
	if arsHas && cheHas {
		t.Errorf("BUG FOUND: Cross Club Shared Player still present in both Arsenal and Chelsea! Dedupe failed (removed=%d)", removed)
	}
}

// ============================================================================
// CHALLENGER TEST SUITE 2: WHOLE-DATABASE 96-CLUB INVARIANT CHECK
// ============================================================================

func TestChallenger_WholeDatabaseInvariant_96Clubs(t *testing.T) {
	ge := growth.NewGrowthEngine(303)
	dm := NewDataManager("dataset.json", ge)
	if err := dm.LoadDataset(); err != nil {
		t.Fatalf("LoadDataset failed: %v", err)
	}
	dm.DedupePlayers()
	if err := dm.InitializeEliteProdigies(); err != nil {
		t.Fatalf("InitializeEliteProdigies failed: %v", err)
	}

	// 1. Exactly 96 clubs
	if len(dm.ClubsList) != 96 {
		t.Fatalf("expected 96 clubs in list, got %d", len(dm.ClubsList))
	}
	if len(dm.Clubs) != 96 {
		t.Fatalf("expected 96 clubs in map, got %d", len(dm.Clubs))
	}

	// 2. Verify all clubs have squad size == len(Squad)
	// and verify zero duplicates across all players
	seenNames := make(map[string]string)
	totalPlayers := 0

	for _, club := range dm.ClubsList {
		if club.ClubID == "" {
			t.Errorf("club has empty ClubID")
		}
		if club.SquadSize != len(club.Squad) {
			t.Errorf("club %s SquadSize (%d) != len(Squad) (%d)", club.ClubID, club.SquadSize, len(club.Squad))
		}
		if len(club.Squad) < 11 {
			t.Errorf("club %s cannot field an XI: %d players", club.ClubID, len(club.Squad))
		}

		for _, p := range club.Squad {
			totalPlayers++
			if p.FullName == "" {
				t.Errorf("club %s has player with empty FullName", club.ClubID)
			}
			if p.PlayerID == "" {
				t.Errorf("player %s has empty PlayerID", p.FullName)
			}
			if p.ClubID != club.ClubID {
				t.Errorf("player %s ClubID (%s) != club.ClubID (%s)", p.FullName, p.ClubID, club.ClubID)
			}
			if p.Age < 14 {
				t.Errorf("player %s age %d < 14", p.FullName, p.Age)
			}
			if p.OVR < 40 || p.OVR > 99 {
				t.Errorf("player %s OVR %d out of bounds [40, 99]", p.FullName, p.OVR)
			}

			norm := strings.ToLower(strings.TrimSpace(p.FullName))
			if prevClub, exists := seenNames[norm]; exists {
				t.Fatalf("DUPLICATE VIOLATION in production database: %q exists in both %s and %s", p.FullName, prevClub, club.ClubID)
			}
			seenNames[norm] = club.ClubID
		}
	}

	t.Logf("Whole-database check passed: verified %d unique players across 96 clubs with 0 duplicates.", totalPlayers)
}

// ============================================================================
// CHALLENGER TEST SUITE 3: WONDERKID INVARIANTS
// ============================================================================

func TestChallenger_Wonderkids_InvariantsAndPotentialBounds(t *testing.T) {
	ge := growth.NewGrowthEngine(404)
	dm := NewDataManager("dataset.json", ge)
	if err := dm.LoadDataset(); err != nil {
		t.Fatalf("LoadDataset failed: %v", err)
	}
	dm.DedupePlayers()
	if err := dm.InitializeEliteProdigies(); err != nil {
		t.Fatalf("InitializeEliteProdigies failed: %v", err)
	}

	if len(dm.Wonderkids) != 12 {
		t.Fatalf("expected 12 wonderkids, got %d", len(dm.Wonderkids))
	}

	expectedConfigs := make(map[string]EliteProdigyConfig)
	for _, cfg := range EliteProdigyConfigs {
		expectedConfigs[cfg.FullName] = cfg
	}

	for _, wk := range dm.Wonderkids {
		cfg, exists := expectedConfigs[wk.FullName]
		if !exists {
			t.Fatalf("unexpected wonderkid %q", wk.FullName)
		}

		// 1. Age must be 14
		if wk.Age != 14 {
			t.Errorf("wonderkid %s Age = %d, expected 14", wk.FullName, wk.Age)
		}

		// 2. Education must be "middle_school" and EducationPending == false
		if wk.Education != "middle_school" {
			t.Errorf("wonderkid %s Education = %q, expected 'middle_school'", wk.FullName, wk.Education)
		}
		if wk.EducationPending {
			t.Errorf("wonderkid %s EducationPending is true, expected false", wk.FullName)
		}

		// 3. Category must be "FWD" (all franchise wonderkids are attackers)
		if wk.Category != "FWD" {
			t.Errorf("wonderkid %s Category = %q, expected 'FWD'", wk.FullName, wk.Category)
		}

		// 4. Stable PlayerID starting with "WK_"
		expectedID := "WK_" + strings.ReplaceAll(wk.FullName, " ", "_")
		if wk.PlayerID != expectedID {
			t.Errorf("wonderkid %s PlayerID = %q, expected %q", wk.FullName, wk.PlayerID, expectedID)
		}

		// 5. UniverseWonderkid flag must be true
		if !wk.UniverseWonderkid {
			t.Errorf("wonderkid %s UniverseWonderkid is false", wk.FullName)
		}

		// 6. Potential strictly in [93, 96] and NEVER 99
		bio := ge.Biometrics[wk.PlayerID]
		if bio == nil {
			t.Fatalf("wonderkid %s not found in GrowthEngine biometrics", wk.FullName)
		}
		if bio.Potential < 93 || bio.Potential > 96 {
			t.Errorf("wonderkid %s potential %d out of bounds [93, 96]", wk.FullName, bio.Potential)
		}
		if bio.Potential == 99 {
			t.Fatalf("CRITICAL INVARIANT VIOLATION: wonderkid %s potential is 99!", wk.FullName)
		}
		if bio.Potential != cfg.Potential {
			t.Errorf("wonderkid %s potential %d != config potential %d", wk.FullName, bio.Potential, cfg.Potential)
		}

		// 7. Wonderkid must be present in home club squad
		homeClub := dm.Clubs[cfg.ClubID]
		if homeClub == nil {
			t.Fatalf("home club %s not found for %s", cfg.ClubID, wk.FullName)
		}
		foundInHome := false
		for _, p := range homeClub.Squad {
			if p.PlayerID == wk.PlayerID && strings.EqualFold(p.FullName, wk.FullName) {
				foundInHome = true
				break
			}
		}
		if !foundInHome {
			t.Errorf("wonderkid %s is not present in %s squad", wk.FullName, cfg.ClubID)
		}

		// 8. Wonderkid must be absent from all other 95 clubs
		for _, otherClub := range dm.ClubsList {
			if otherClub.ClubID == cfg.ClubID {
				continue
			}
			for _, p := range otherClub.Squad {
				if strings.EqualFold(p.FullName, wk.FullName) || p.PlayerID == wk.PlayerID {
					t.Errorf("wonderkid %s leaked into other club %s", wk.FullName, otherClub.ClubID)
				}
			}
		}

		// 9. Valuation clamped
		if wk.MarketValueEUR < 300_000 || wk.MarketValueEUR > 500_000_000 {
			t.Errorf("wonderkid %s market value %d out of clamped bounds", wk.FullName, wk.MarketValueEUR)
		}
	}
}

// ============================================================================
// CHALLENGER TEST SUITE 4: RELOCATION CHECK
// ============================================================================

func TestChallenger_Relocation_GuinitaStrictVerification(t *testing.T) {
	ge := growth.NewGrowthEngine(505)
	dm := NewDataManager("dataset.json", ge)
	if err := dm.LoadDataset(); err != nil {
		t.Fatalf("LoadDataset failed: %v", err)
	}

	marseille := dm.Clubs["FL1-OM"]
	tottenham := dm.Clubs["EPL-TOT"]
	if marseille == nil || tottenham == nil {
		t.Fatal("FL1-OM or EPL-TOT missing")
	}

	// 2. Perform dedupe and elite wonderkids initialization
	dm.DedupePlayers()
	if err := dm.InitializeEliteProdigies(); err != nil {
		t.Fatalf("InitializeEliteProdigies failed: %v", err)
	}

	// 3. Verify Jhed Anthony Guinita is now at Tottenham Hotspur (EPL-TOT)
	tottenhamHasGuinita := false
	var jhedInTottenham *models.Player
	for _, p := range tottenham.Squad {
		if strings.EqualFold(p.FullName, "Jhed Anthony Guinita") {
			tottenhamHasGuinita = true
			jhedInTottenham = p
			break
		}
	}
	if !tottenhamHasGuinita || jhedInTottenham == nil {
		t.Fatal("Jhed Anthony Guinita was not found in Tottenham Hotspur squad")
	}
	if jhedInTottenham.ClubID != "EPL-TOT" {
		t.Errorf("Jhed Anthony Guinita ClubID = %q, expected 'EPL-TOT'", jhedInTottenham.ClubID)
	}
	if jhedInTottenham.PlayerID != "WK_Jhed_Anthony_Guinita" {
		t.Errorf("Jhed Anthony Guinita PlayerID = %q, expected 'WK_Jhed_Anthony_Guinita'", jhedInTottenham.PlayerID)
	}
	if jhedInTottenham.Age != 14 {
		t.Errorf("Jhed Anthony Guinita Age = %d, expected 14", jhedInTottenham.Age)
	}

	// 4. Verify Jhed Anthony Guinita is ABSENT from Marseille (FL1-OM)
	for _, p := range marseille.Squad {
		if strings.EqualFold(p.FullName, "Jhed Anthony Guinita") || p.PlayerID == "WK_Jhed_Anthony_Guinita" {
			t.Fatalf("Jhed Anthony Guinita STILL PRESENT in Marseille squad!")
		}
	}

	// 5. Verify Jhed Anthony Guinita is absent from all other 94 clubs
	for _, c := range dm.ClubsList {
		if c.ClubID == "EPL-TOT" {
			continue
		}
		for _, p := range c.Squad {
			if strings.EqualFold(p.FullName, "Jhed Anthony Guinita") || p.PlayerID == "WK_Jhed_Anthony_Guinita" {
				t.Fatalf("Jhed Anthony Guinita present in club %s!", c.ClubID)
			}
		}
	}

	// 6. Verify SquadSize integrity on both clubs
	if marseille.SquadSize != len(marseille.Squad) {
		t.Errorf("Marseille SquadSize %d != len(Squad) %d", marseille.SquadSize, len(marseille.Squad))
	}
	if tottenham.SquadSize != len(tottenham.Squad) {
		t.Errorf("Tottenham SquadSize %d != len(Squad) %d", tottenham.SquadSize, len(tottenham.Squad))
	}
}

// ============================================================================
// CHALLENGER TEST SUITE 5: YOUTH INTAKE STRESS & SQUAD CAP 34
// ============================================================================

func TestChallenger_YouthIntake_BoundarySquadCaps(t *testing.T) {
	ge := growth.NewGrowthEngine(606)
	dm := NewDataManager("dataset.json", ge)
	if err := dm.LoadDataset(); err != nil {
		t.Fatalf("LoadDataset failed: %v", err)
	}
	dm.SetSeed(12345)

	targetClub := dm.Clubs["EPL-ARS"]
	if targetClub == nil {
		t.Fatal("EPL-ARS not found")
	}

	// Test boundary matrix: {initialSize, requestedCount, expectedAdmitted, expectedFinalSize}
	testCases := []struct {
		initialSize      int
		requestedCount   int
		expectedAdmitted int
		expectedFinal    int
	}{
		{initialSize: 30, requestedCount: 4, expectedAdmitted: 4, expectedFinal: 34},
		{initialSize: 31, requestedCount: 4, expectedAdmitted: 3, expectedFinal: 34},
		{initialSize: 32, requestedCount: 4, expectedAdmitted: 2, expectedFinal: 34},
		{initialSize: 33, requestedCount: 4, expectedAdmitted: 1, expectedFinal: 34},
		{initialSize: 34, requestedCount: 4, expectedAdmitted: 0, expectedFinal: 34},
		{initialSize: 35, requestedCount: 4, expectedAdmitted: 0, expectedFinal: 35}, // Over-cap: must not grow
		{initialSize: 40, requestedCount: 4, expectedAdmitted: 0, expectedFinal: 40}, // Severely over-cap: must not grow
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("SquadSize_%d_Request_%d", tc.initialSize, tc.requestedCount), func(t *testing.T) {
			// Construct club with exact squad size
			testSquad := make([]*models.Player, tc.initialSize)
			for i := 0; i < tc.initialSize; i++ {
				testSquad[i] = &models.Player{
					PlayerID: fmt.Sprintf("PAD_%d", i),
					FullName: fmt.Sprintf("Pad Player %d", i),
					Position: "CM",
					OVR:      70,
					ClubID:   targetClub.ClubID,
				}
			}
			targetClub.Squad = testSquad
			targetClub.SquadSize = len(targetClub.Squad)

			grads, err := dm.RunYouthIntakeWithCount("EPL-ARS", tc.requestedCount)
			if err != nil {
				t.Fatalf("RunYouthIntakeWithCount failed: %v", err)
			}

			if len(grads) != tc.expectedAdmitted {
				t.Errorf("admitted %d grads, expected %d", len(grads), tc.expectedAdmitted)
			}
			if len(targetClub.Squad) != tc.expectedFinal {
				t.Errorf("final squad size %d, expected %d", len(targetClub.Squad), tc.expectedFinal)
			}
			if targetClub.SquadSize != len(targetClub.Squad) {
				t.Errorf("SquadSize %d != len(Squad) %d", targetClub.SquadSize, len(targetClub.Squad))
			}
		})
	}
}

func TestChallenger_YouthIntake_MultiRoundStress(t *testing.T) {
	ge := growth.NewGrowthEngine(707)
	dm := NewDataManager("dataset.json", ge)
	if err := dm.LoadDataset(); err != nil {
		t.Fatalf("LoadDataset failed: %v", err)
	}
	dm.DedupePlayers()
	dm.InitializeEliteProdigies()
	dm.SetSeed(9999)

	// Run 10 consecutive intake seasons across ALL 96 clubs
	// Invariant: no club must EVER exceed 34 players at any round
	const seasons = 10
	totalAdmittedAcrossSeasons := 0

	for season := 1; season <= seasons; season++ {
		grads, err := dm.RunYouthIntake("all")
		if err != nil {
			t.Fatalf("season %d RunYouthIntake('all') failed: %v", season, err)
		}
		totalAdmittedAcrossSeasons += len(grads)

		// Assert invariant across all 96 clubs
		for _, c := range dm.ClubsList {
			if len(c.Squad) > 34 {
				t.Fatalf("INVARIANT VIOLATION: club %s squad size %d exceeds cap 34 in season %d", c.ClubID, len(c.Squad), season)
			}
			if c.SquadSize != len(c.Squad) {
				t.Errorf("club %s SquadSize %d != len(Squad) %d in season %d", c.ClubID, c.SquadSize, len(c.Squad), season)
			}
		}
	}

	t.Logf("Multi-round stress passed: 10 global intake rounds admitted %d total grads; squad cap 34 held across all 96 clubs.", totalAdmittedAcrossSeasons)

	// Verify all clubs are capped at 34 (since each club started with ~24 players and 10 rounds gives 20-40 graduates)
	for _, c := range dm.ClubsList {
		if len(c.Squad) != 34 {
			t.Errorf("club %s squad size %d != 34 after 10 intake seasons", c.ClubID, len(c.Squad))
		}
	}

	// Verify that running another intake round at full capacity admits exactly 0 players
	zeroGrads, err := dm.RunYouthIntake("all")
	if err != nil {
		t.Fatalf("RunYouthIntake on saturated clubs failed: %v", err)
	}
	if len(zeroGrads) != 0 {
		t.Errorf("expected 0 grads admitted on fully saturated clubs, got %d", len(zeroGrads))
	}
}

func TestChallenger_YouthIntake_GraduatesAttributeValidity(t *testing.T) {
	ge := growth.NewGrowthEngine(808)
	dm := NewDataManager("dataset.json", ge)
	if err := dm.LoadDataset(); err != nil {
		t.Fatalf("LoadDataset failed: %v", err)
	}
	dm.SetSeed(54321)

	// Generate graduates across several clubs and inspect attributes
	grads, err := dm.RunYouthIntakeWithCount("EPL-ARS", 4)
	if err != nil {
		t.Fatalf("RunYouthIntakeWithCount failed: %v", err)
	}

	validPersonalities := map[string]bool{
		"dedicated_pro":      true,
		"flamboyant_star":    true,
		"academic_dual":      true,
		"big_game_performer": true,
	}

	for _, g := range grads {
		if !strings.HasPrefix(g.PlayerID, "AC_EPL-ARS_") {
			t.Errorf("PlayerID %q does not match AC_EPL-ARS_ prefix", g.PlayerID)
		}
		if g.Age < 16 || g.Age > 18 {
			t.Errorf("graduate age %d out of range [16, 18]", g.Age)
		}
		if !validPersonalities[g.Personality] {
			t.Errorf("invalid personality %q", g.Personality)
		}
		if g.MarketValueEUR < 300_000 || g.MarketValueEUR > 500_000_000 {
			t.Errorf("market value %d out of bounds [300k, 500M]", g.MarketValueEUR)
		}
		if g.ClubID != "EPL-ARS" || g.OriginalClubID != "EPL-ARS" {
			t.Errorf("club ID assignment mismatch: ClubID=%s, OriginalClubID=%s", g.ClubID, g.OriginalClubID)
		}
		// Check GrowthEngine enrollment
		bio := ge.Biometrics[g.PlayerID]
		if bio == nil {
			t.Errorf("graduate %s not registered in GrowthEngine", g.PlayerID)
		} else {
			if bio.Potential < 75 || bio.Potential > 95 {
				t.Errorf("graduate potential %d out of expected range [75, 95]", bio.Potential)
			}
		}
	}
}

func TestChallenger_InitializeEliteProdigies_Idempotency(t *testing.T) {
	ge := growth.NewGrowthEngine(909)
	dm := NewDataManager("dataset.json", ge)
	if err := dm.LoadDataset(); err != nil {
		t.Fatalf("LoadDataset failed: %v", err)
	}
	dm.DedupePlayers()

	// Call InitializeEliteProdigies 3 times consecutively
	for i := 0; i < 3; i++ {
		if err := dm.InitializeEliteProdigies(); err != nil {
			t.Fatalf("InitializeEliteProdigies iteration %d failed: %v", i, err)
		}
	}

	// Must still have exactly 12 wonderkids
	if len(dm.Wonderkids) != 12 {
		t.Errorf("expected 12 wonderkids after multiple initializations, got %d", len(dm.Wonderkids))
	}

	// Check that none of the 12 wonderkids are duplicated in their home club squads
	for _, wk := range dm.Wonderkids {
		homeClub := dm.Clubs[wk.ClubID]
		count := 0
		for _, p := range homeClub.Squad {
			if strings.EqualFold(p.FullName, wk.FullName) {
				count++
			}
		}
		if count != 1 {
			t.Errorf("wonderkid %s appears %d times in %s after multiple initializations", wk.FullName, count, wk.ClubID)
		}
	}
}

func TestChallenger_InitializeEliteProdigies_MissingWonderkidDirectInstantiation(t *testing.T) {
	ge := growth.NewGrowthEngine(910)
	// Create an empty DataManager without loading dataset.json
	dm := &DataManager{
		Clubs:        make(map[string]*models.Club),
		ClubsList:    make([]*models.Club, 0),
		Leagues:      make(map[string][]*models.Club),
		Wonderkids:   make([]*models.Player, 0),
		ProdigyHomes: DefaultProdigyHomes(),
		GrowthEngine: ge,
	}

	// Create minimal clubs for the 12 elite clubs
	for _, cfg := range EliteProdigyConfigs {
		c := &models.Club{
			ClubID:    cfg.ClubID,
			ClubName:  cfg.ClubID + " FC",
			Squad:     make([]*models.Player, 0),
			SquadSize: 0,
		}
		dm.Clubs[cfg.ClubID] = c
		dm.ClubsList = append(dm.ClubsList, c)
	}

	// Initialize without any players pre-existing in the clubs
	err := dm.InitializeEliteProdigies()
	if err != nil {
		t.Fatalf("InitializeEliteProdigies failed on empty clubs: %v", err)
	}

	if len(dm.Wonderkids) != 12 {
		t.Errorf("expected 12 wonderkids directly instantiated, got %d", len(dm.Wonderkids))
	}

	for _, wk := range dm.Wonderkids {
		if wk.Age != 14 {
			t.Errorf("directly instantiated wonderkid %s has age %d, expected 14", wk.FullName, wk.Age)
		}
		if !strings.HasPrefix(wk.PlayerID, "WK_") {
			t.Errorf("directly instantiated wonderkid %s has ID %q without WK_ prefix", wk.FullName, wk.PlayerID)
		}
		if wk.Education != "middle_school" {
			t.Errorf("directly instantiated wonderkid %s has education %q, expected 'middle_school'", wk.FullName, wk.Education)
		}
		homeClub := dm.Clubs[wk.ClubID]
		if len(homeClub.Squad) != 1 || homeClub.Squad[0] != wk {
			t.Errorf("wonderkid %s not present in home club %s squad", wk.FullName, wk.ClubID)
		}
	}
}

func TestChallenger_MarketValue_BaselineSnappingFullDB(t *testing.T) {
	ge := growth.NewGrowthEngine(911)
	dm := NewDataManager("dataset.json", ge)
	if err := dm.LoadDataset(); err != nil {
		t.Fatalf("LoadDataset failed: %v", err)
	}
	dm.DedupePlayers()
	dm.InitializeEliteProdigies()
	dm.ResetMarketToBaseline()

	for _, c := range dm.ClubsList {
		for _, p := range c.Squad {
			if p.MarketValueEUR < 300_000 {
				t.Fatalf("player %s in %s market value €%d below €300,000 floor", p.FullName, c.ClubID, p.MarketValueEUR)
			}
			if p.MarketValueEUR > 500_000_000 {
				t.Fatalf("player %s in %s market value €%d above €500,000,000 ceiling", p.FullName, c.ClubID, p.MarketValueEUR)
			}
		}
	}
}

// ============================================================================
// CHALLENGER ROUND 2: ADVERSARIAL STRESS SUITES
// ============================================================================

// TestChallenger_R2_MultiClubPointerMesh_AdversarialStress tests injecting the
// exact same struct pointer across 5 different clubs with multiple intra-club
// repetitions (12 instances total). Verifies exact removal count, single canonical
// retention, squad size synchronization, and idempotency.
func TestChallenger_R2_MultiClubPointerMesh_AdversarialStress(t *testing.T) {
	ge := growth.NewGrowthEngine(2026)
	dm := NewDataManager("dataset.json", ge)
	if err := dm.LoadDataset(); err != nil {
		t.Fatalf("LoadDataset failed: %v", err)
	}

	ars := dm.Clubs["EPL-ARS"] // Elite
	che := dm.Clubs["EPL-CHE"] // Elite
	ful := dm.Clubs["EPL-FUL"] // Non-elite
	bar := dm.Clubs["LAL-BAR"] // Elite
	om := dm.Clubs["FL1-OM"]   // Non-elite
	if ars == nil || che == nil || ful == nil || bar == nil || om == nil {
		t.Fatal("target clubs for multi-club pointer mesh not found")
	}
	_ = dm.DedupePlayers()

	initialTotalPlayers := 0
	for _, c := range dm.ClubsList {
		initialTotalPlayers += len(c.Squad)
	}

	sharedP := &models.Player{
		PlayerID:    "P_SHARED_MESH",
		FullName:    "Mesh Aliased Player",
		Position:    "CAM",
		OVR:         83,
		Appearances: 14,
		Goals:       7,
	}

	// Inject 12 instances across 5 clubs
	// EPL-ARS: 3 copies
	ars.Squad = append(ars.Squad, sharedP, sharedP, sharedP)
	ars.SquadSize = len(ars.Squad)
	// EPL-CHE: 2 copies
	che.Squad = append(che.Squad, sharedP, sharedP)
	che.SquadSize = len(che.Squad)
	// EPL-FUL: 4 copies
	ful.Squad = append(ful.Squad, sharedP, sharedP, sharedP, sharedP)
	ful.SquadSize = len(ful.Squad)
	// LAL-BAR: 1 copy
	bar.Squad = append(bar.Squad, sharedP)
	bar.SquadSize = len(bar.Squad)
	// FL1-OM: 2 copies
	om.Squad = append(om.Squad, sharedP, sharedP)
	om.SquadSize = len(om.Squad)

	removed := dm.DedupePlayers()
	if removed != 11 {
		t.Fatalf("expected exactly 11 duplicates removed for 12 injected copies, got %d", removed)
	}

	// Count occurrences across all 96 clubs
	totalRemaining := 0
	var foundClubID string
	for _, c := range dm.ClubsList {
		if c.SquadSize != len(c.Squad) {
			t.Errorf("club %s SquadSize %d != len(Squad) %d", c.ClubID, c.SquadSize, len(c.Squad))
		}
		for _, p := range c.Squad {
			if strings.EqualFold(p.FullName, "Mesh Aliased Player") {
				totalRemaining++
				foundClubID = c.ClubID
			}
		}
	}

	if totalRemaining != 1 {
		t.Fatalf("expected exactly 1 copy of Mesh Aliased Player remaining across whole DB, got %d", totalRemaining)
	}

	// Verify retained player is in an elite club
	eliteClubs := map[string]bool{"EPL-ARS": true, "EPL-CHE": true, "LAL-BAR": true}
	if !eliteClubs[foundClubID] {
		t.Errorf("expected retained player to be in an elite club (ARS, CHE, or BAR), got %s", foundClubID)
	}

	// Verify idempotency: second deduplication must remove 0 and leave DB untouched
	secondRemoved := dm.DedupePlayers()
	if secondRemoved != 0 {
		t.Errorf("second DedupePlayers run expected 0 removed, got %d", secondRemoved)
	}

	finalTotalPlayers := 0
	for _, c := range dm.ClubsList {
		finalTotalPlayers += len(c.Squad)
	}
	if finalTotalPlayers != initialTotalPlayers+1 {
		t.Errorf("expected final player count to be baseline+1 (%d), got %d", initialTotalPlayers+1, finalTotalPlayers)
	}
}

// TestChallenger_R2_MixedAliasedAndDistinctWithMaxStatsMerge tests deduplication
// where some copies are pointer-aliased and other copies are distinct structs
// with different stats, ensuring max() aggregation operates properly without corruption.
func TestChallenger_R2_MixedAliasedAndDistinctWithMaxStatsMerge(t *testing.T) {
	ge := growth.NewGrowthEngine(2027)
	dm := NewDataManager("dataset.json", ge)
	if err := dm.LoadDataset(); err != nil {
		t.Fatalf("LoadDataset failed: %v", err)
	}

	mci := dm.Clubs["EPL-MCI"]
	liv := dm.Clubs["EPL-LIV"]
	rma := dm.Clubs["LAL-RMA"]
	if mci == nil || liv == nil || rma == nil {
		t.Fatal("target clubs for mixed aliasing test not found")
	}
	_ = dm.DedupePlayers()

	// Struct A
	ptrA := &models.Player{
		PlayerID:      "P_MAX_A",
		FullName:      "Maximus Merge",
		Position:      "RW",
		OVR:           82,
		Appearances:   15,
		Goals:         10,
		Assists:       5,
		CareerGoals:   25,
		CareerAssists: 10,
		CareerApps:    40,
		BestGoals:     12,
		BestAssists:   6,
	}
	// Struct C (distinct struct, higher OVR and Appearances)
	ptrC := &models.Player{
		PlayerID:      "P_MAX_C",
		FullName:      "Maximus Merge",
		Position:      "RW",
		OVR:           85,
		Appearances:   25,
		Goals:         20,
		Assists:       15,
		CareerGoals:   50,
		CareerAssists: 30,
		CareerApps:    70,
		BestGoals:     22,
		BestAssists:   16,
	}
	// Struct E (distinct struct, different career stats)
	ptrE := &models.Player{
		PlayerID:      "P_MAX_E",
		FullName:      "Maximus Merge",
		Position:      "RW",
		OVR:           80,
		Appearances:   5,
		Goals:         2,
		Assists:       1,
		CareerGoals:   60, // Highest career goals!
		CareerAssists: 5,
		CareerApps:    90, // Highest career apps!
		BestGoals:     30, // Highest best goals!
		BestAssists:   2,
	}

	// EPL-MCI: ptrA twice
	mci.Squad = append(mci.Squad, ptrA, ptrA)
	mci.SquadSize = len(mci.Squad)
	// EPL-LIV: ptrC twice
	liv.Squad = append(liv.Squad, ptrC, ptrC)
	liv.SquadSize = len(liv.Squad)
	// LAL-RMA: ptrE once, and ptrA once
	rma.Squad = append(rma.Squad, ptrE, ptrA)
	rma.SquadSize = len(rma.Squad)

	// Total 6 copies injected: 2 in MCI, 2 in LIV, 2 in RMA
	removed := dm.DedupePlayers()
	if removed != 5 {
		t.Fatalf("expected 5 duplicates removed, got %d", removed)
	}

	// Verify exactly 1 copy remains
	var canonicalPlayer *models.Player
	var canonicalClub string
	for _, c := range dm.ClubsList {
		for _, p := range c.Squad {
			if strings.EqualFold(p.FullName, "Maximus Merge") {
				if canonicalPlayer != nil {
					t.Fatalf("found multiple copies of Maximus Merge in DB!")
				}
				canonicalPlayer = p
				canonicalClub = c.ClubID
			}
		}
	}

	if canonicalPlayer == nil {
		t.Fatal("Maximus Merge completely vanished from DB!")
	}

	// Verify stats were correctly max-merged
	if canonicalPlayer.Appearances != 25 {
		t.Errorf("expected Appearances 25, got %d", canonicalPlayer.Appearances)
	}
	if canonicalPlayer.Goals != 20 {
		t.Errorf("expected Goals 20, got %d", canonicalPlayer.Goals)
	}
	if canonicalPlayer.Assists != 15 {
		t.Errorf("expected Assists 15, got %d", canonicalPlayer.Assists)
	}
	if canonicalPlayer.CareerGoals != 60 {
		t.Errorf("expected CareerGoals 60 (from ptrE), got %d", canonicalPlayer.CareerGoals)
	}
	if canonicalPlayer.CareerApps != 90 {
		t.Errorf("expected CareerApps 90 (from ptrE), got %d", canonicalPlayer.CareerApps)
	}
	if canonicalPlayer.BestGoals != 30 {
		t.Errorf("expected BestGoals 30 (from ptrE), got %d", canonicalPlayer.BestGoals)
	}
	if canonicalPlayer.BestAssists != 16 {
		t.Errorf("expected BestAssists 16 (from ptrC), got %d", canonicalPlayer.BestAssists)
	}

	t.Logf("Maximus Merge canonical club: %s, OVR: %d, Appearances: %d, CareerGoals: %d",
		canonicalClub, canonicalPlayer.OVR, canonicalPlayer.Appearances, canonicalPlayer.CareerGoals)
}

// TestChallenger_R2_WonderkidPointerAliasedCrossClubDuplication tests injecting
// aliased wonderkid pointers into non-canonical clubs and duplicating inside
// canonical club, ensuring DedupePlayers and InitializeEliteProdigies handle it cleanly.
func TestChallenger_R2_WonderkidPointerAliasedCrossClubDuplication(t *testing.T) {
	ge := growth.NewGrowthEngine(2028)
	dm := NewDataManager("dataset.json", ge)
	if err := dm.LoadDataset(); err != nil {
		t.Fatalf("LoadDataset failed: %v", err)
	}

	tot := dm.Clubs["EPL-TOT"]
	om := dm.Clubs["FL1-OM"]
	rma := dm.Clubs["LAL-RMA"]
	if tot == nil || om == nil || rma == nil {
		t.Fatal("target clubs for wonderkid aliasing test not found")
	}
	_ = dm.DedupePlayers()

	// Guinita pointer
	guinitaPtr := &models.Player{
		PlayerID:          "WK_Jhed_Anthony_Guinita",
		FullName:          "Jhed Anthony Guinita",
		Position:          "ST",
		Age:               14,
		OVR:               78,
		UniverseWonderkid: true,
		Education:         "middle_school",
		ClubID:            "EPL-TOT",
	}

	// Inject 2 copies in TOT, 2 copies in OM, 1 copy in RMA (5 total)
	tot.Squad = append(tot.Squad, guinitaPtr, guinitaPtr)
	tot.SquadSize = len(tot.Squad)
	om.Squad = append(om.Squad, guinitaPtr, guinitaPtr)
	om.SquadSize = len(om.Squad)
	rma.Squad = append(rma.Squad, guinitaPtr)
	rma.SquadSize = len(rma.Squad)

	// Total instances = 1 (baseline in FL1-OM) + 2 (in TOT) + 2 (in OM) + 1 (in RMA) = 6 total
	// DedupePlayers should consolidate all 6 into exactly 1 in EPL-TOT, removing 5
	removed := dm.DedupePlayers()
	if removed != 5 {
		t.Fatalf("expected 5 duplicates removed for Guinita (1 baseline + 5 injected - 1 kept), got %d", removed)
	}

	// Ensure TOT has exactly 1 Guinita, OM and RMA have 0
	totCount := 0
	for _, p := range tot.Squad {
		if strings.EqualFold(p.FullName, "Jhed Anthony Guinita") {
			totCount++
		}
	}
	if totCount != 1 {
		t.Errorf("expected exactly 1 Guinita in EPL-TOT, got %d", totCount)
	}

	for _, p := range om.Squad {
		if strings.EqualFold(p.FullName, "Jhed Anthony Guinita") {
			t.Errorf("Guinita still present in FL1-OM!")
		}
	}
	for _, p := range rma.Squad {
		if strings.EqualFold(p.FullName, "Jhed Anthony Guinita") {
			t.Errorf("Guinita still present in LAL-RMA!")
		}
	}

	// Now run InitializeEliteProdigies and verify full setup
	if err := dm.InitializeEliteProdigies(); err != nil {
		t.Fatalf("InitializeEliteProdigies failed: %v", err)
	}

	// Check wonderkids count and properties
	if len(dm.Wonderkids) != 12 {
		t.Fatalf("expected 12 wonderkids, got %d", len(dm.Wonderkids))
	}
}

// TestChallenger_R2_WhitespaceAndCaseAliasingAttack tests deduplication when
// duplicate instances have leading/trailing whitespace and mixed case names.
func TestChallenger_R2_WhitespaceAndCaseAliasingAttack(t *testing.T) {
	ge := growth.NewGrowthEngine(2029)
	dm := NewDataManager("dataset.json", ge)
	if err := dm.LoadDataset(); err != nil {
		t.Fatalf("LoadDataset failed: %v", err)
	}

	ars := dm.Clubs["EPL-ARS"]
	che := dm.Clubs["EPL-CHE"]
	if ars == nil || che == nil {
		t.Fatal("clubs not found")
	}
	_ = dm.DedupePlayers()

	p1 := &models.Player{
		PlayerID: "P_WS_1",
		FullName: "  Whitespace Striker  ",
		Position: "ST",
		OVR:      77,
	}
	p2 := &models.Player{
		PlayerID: "P_WS_2",
		FullName: "WHITESPACE STRIKER",
		Position: "ST",
		OVR:      79,
	}

	ars.Squad = append(ars.Squad, p1)
	che.Squad = append(che.Squad, p2)
	ars.SquadSize = len(ars.Squad)
	che.SquadSize = len(che.Squad)

	removed := dm.DedupePlayers()
	if removed != 1 {
		t.Fatalf("expected 1 duplicate removed for whitespace/case variance, got %d", removed)
	}

	total := 0
	for _, c := range dm.ClubsList {
		for _, p := range c.Squad {
			if strings.EqualFold(strings.TrimSpace(p.FullName), "Whitespace Striker") {
				total++
			}
		}
	}
	if total != 1 {
		t.Fatalf("expected exactly 1 copy of Whitespace Striker remaining, got %d", total)
	}
}

// TestChallenger_R2_ExhaustiveWholeDatabaseSanity executes the full ingestion,
// deduplication, wonderkid initialization, and market baseline snapping pipeline
// and audits the entire 96-club database for strict invariant compliance.
func TestChallenger_R2_ExhaustiveWholeDatabaseSanity(t *testing.T) {
	ge := growth.NewGrowthEngine(2030)
	dm := NewDataManager("dataset.json", ge)
	if err := dm.LoadDataset(); err != nil {
		t.Fatalf("LoadDataset failed: %v", err)
	}

	removed := dm.DedupePlayers()
	if again := dm.DedupePlayers(); again != 0 {
		t.Errorf("dedupe should be idle on a second pass, first=%d second=%d", removed, again)
	}

	if err := dm.InitializeEliteProdigies(); err != nil {
		t.Fatalf("InitializeEliteProdigies failed: %v", err)
	}

	dm.ResetMarketToBaseline()

	if len(dm.ClubsList) != 96 {
		t.Fatalf("expected 96 clubs, got %d", len(dm.ClubsList))
	}

	seenNames := make(map[string]string) // normName -> ClubID
	totalPlayers := 0

	for _, c := range dm.ClubsList {
		if c.SquadSize != len(c.Squad) {
			t.Errorf("club %s SquadSize %d != len(Squad) %d", c.ClubID, c.SquadSize, len(c.Squad))
		}
		for _, p := range c.Squad {
			totalPlayers++
			if p.ClubID != c.ClubID {
				t.Errorf("player %s has ClubID %q, expected %q", p.FullName, p.ClubID, c.ClubID)
			}
			norm := strings.ToLower(strings.TrimSpace(p.FullName))
			if prevClub, exists := seenNames[norm]; exists {
				t.Errorf("DUPLICATE FOUND: player %q present in %s and %s", p.FullName, prevClub, c.ClubID)
			}
			seenNames[norm] = c.ClubID

			if p.MarketValueEUR < 300_000 || p.MarketValueEUR > 500_000_000 {
				t.Errorf("player %s in %s market value €%d out of bounds [300k, 500M]", p.FullName, c.ClubID, p.MarketValueEUR)
			}
		}
	}

	if totalPlayers != len(seenNames) {
		t.Fatalf("expected unique-name census %d, got %d players", len(seenNames), totalPlayers)
	}

	if len(dm.Wonderkids) != 12 {
		t.Fatalf("expected 12 wonderkids, got %d", len(dm.Wonderkids))
	}

	for _, wk := range dm.Wonderkids {
		if wk.Age != 14 {
			t.Errorf("wonderkid %s age %d != 14", wk.FullName, wk.Age)
		}
		if wk.Education != "middle_school" {
			t.Errorf("wonderkid %s education %q != 'middle_school'", wk.FullName, wk.Education)
		}
		if wk.Category != "FWD" {
			t.Errorf("wonderkid %s category %q != 'FWD'", wk.FullName, wk.Category)
		}
		if !wk.UniverseWonderkid {
			t.Errorf("wonderkid %s UniverseWonderkid is false", wk.FullName)
		}
		if !strings.HasPrefix(wk.PlayerID, "WK_") {
			t.Errorf("wonderkid %s ID %q lacks WK_ prefix", wk.FullName, wk.PlayerID)
		}

		profile := ge.Biometrics[wk.PlayerID]
		if profile == nil {
			t.Errorf("wonderkid %s has no BiometricProfile in GrowthEngine", wk.FullName)
		} else {
			if profile.Potential < 93 || profile.Potential > 96 {
				t.Errorf("wonderkid %s potential %d out of [93, 96] bounds", wk.FullName, profile.Potential)
			}
			if profile.Potential == 99 {
				t.Errorf("wonderkid %s potential is 99 (forbidden ceiling)", wk.FullName)
			}
		}
	}

	// Verify Guinita specifically
	totClub := dm.Clubs["EPL-TOT"]
	omClub := dm.Clubs["FL1-OM"]
	if totClub == nil || omClub == nil {
		t.Fatal("EPL-TOT or FL1-OM not found")
	}

	totHasGuinita := false
	for _, p := range totClub.Squad {
		if strings.EqualFold(p.FullName, "Jhed Anthony Guinita") {
			totHasGuinita = true
			if p.ClubID != "EPL-TOT" {
				t.Errorf("Guinita in EPL-TOT has wrong ClubID %s", p.ClubID)
			}
		}
	}
	if !totHasGuinita {
		t.Errorf("Jhed Anthony Guinita NOT found in EPL-TOT squad")
	}

	for _, p := range omClub.Squad {
		if strings.EqualFold(p.FullName, "Jhed Anthony Guinita") {
			t.Errorf("Jhed Anthony Guinita still present in FL1-OM squad!")
		}
	}

	t.Logf("Exhaustive whole-database sanity passed: 96 clubs, %d players, 0 duplicates, 12 wonderkids valid.", totalPlayers)
}
