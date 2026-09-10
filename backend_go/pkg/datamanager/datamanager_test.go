package datamanager

import (
	"math/rand"
	"os"
	"strings"
	"testing"

	"football_sim/pkg/growth"
	"football_sim/pkg/models"
)

func newTestGrowthEngine() *growth.GrowthEngine {
	return growth.NewGrowthEngine(42)
}

func TestLoadDataset_Fidelity(t *testing.T) {
	ge := newTestGrowthEngine()
	dm := NewDataManager("dataset.json", ge)

	if err := dm.LoadDataset(); err != nil {
		t.Fatalf("LoadDataset failed: %v", err)
	}

	// 1. Ingest all 96 clubs
	if len(dm.Clubs) != 96 {
		t.Errorf("expected 96 clubs in map, got %d", len(dm.Clubs))
	}
	if len(dm.ClubsList) != 96 {
		t.Errorf("expected 96 clubs in list, got %d", len(dm.ClubsList))
	}

	// 2. Ingest every listed player (this snapshot is 2,401; older files were 2,294).
	totalPlayers := 0
	for _, club := range dm.ClubsList {
		totalPlayers += len(club.Squad)
	}
	wantPlayers := dm.Metadata.Totals["players"]
	if wantPlayers == 0 {
		wantPlayers = totalPlayers
	}
	if totalPlayers != wantPlayers {
		t.Errorf("expected %d total squad players, got %d", wantPlayers, totalPlayers)
	}

	// 3. League counts match official 2026-27 composition
	expectedLeagues := map[string]int{
		"Premier League": 20,
		"La Liga":        20,
		"Serie A":        20,
		"Bundesliga":     18,
		"Ligue 1":        18,
	}
	for league, expectedCount := range expectedLeagues {
		actualClubs := dm.Leagues[league]
		if len(actualClubs) != expectedCount {
			t.Errorf("expected %d clubs in %s, got %d", expectedCount, league, len(actualClubs))
		}
	}

	// 4. Metadata fidelity
	if dm.Metadata.Season != "2026-27" {
		t.Errorf("expected season 2026-27, got %q", dm.Metadata.Season)
	}
	if dm.Metadata.Totals["clubs"] != 96 || dm.Metadata.Totals["players"] != totalPlayers {
		t.Errorf("metadata totals mismatch: %+v (squad sum %d)", dm.Metadata.Totals, totalPlayers)
	}
	if dm.Metadata.Totals["wonderkids"] != 12 {
		t.Errorf("metadata wonderkids = %d; want 12", dm.Metadata.Totals["wonderkids"])
	}

	// 5. Season ledger reset: all player match/career stats must be 0 for fresh save kickoff
	for _, club := range dm.ClubsList {
		for _, p := range club.Squad {
			if p.CareerGoals != 0 || p.CareerAssists != 0 || p.CareerApps != 0 ||
				p.BestGoals != 0 || p.BestAssists != 0 || p.BestSeason != "" ||
				p.OwnGoals != 0 || p.SuspendedMatches != 0 || p.InjuredMatches != 0 ||
				p.Goals != 0 || p.Assists != 0 || p.Appearances != 0 {
				t.Fatalf("player %s in %s did not have stats reset to 0: %+v", p.FullName, club.ClubID, p)
			}
		}
	}
}

func TestDedupePlayers_Invariant(t *testing.T) {
	ge := newTestGrowthEngine()
	dm := NewDataManager("dataset.json", ge)
	if err := dm.LoadDataset(); err != nil {
		t.Fatalf("LoadDataset failed: %v", err)
	}

	// Collapse any name collisions in this snapshot, then inject known extras.
	_ = dm.DedupePlayers()
	if leftover := dm.DedupePlayers(); leftover != 0 {
		t.Errorf("second dedupe should be idle, got %d", leftover)
	}

	// 1. Inject duplicate for a player in PREFERRED_HOMES
	// "Martin Zubimendi" has preferred home "EPL-ARS"
	chelsea := dm.Clubs["EPL-CHE"]
	arsenal := dm.Clubs["EPL-ARS"]
	if chelsea == nil || arsenal == nil {
		t.Fatal("chelsea or arsenal not found")
	}

	dupZubiChelsea := &models.Player{
		PlayerID:       "P_DUP_ZUBI_CHE",
		FullName:       "Martin Zubimendi",
		Position:       "CDM",
		OVR:            84,
		Appearances:    10,
		Goals:          2,
		Assists:        3,
		CareerGoals:    15,
		CareerAssists:  12,
		CareerApps:     80,
		ClubID:         chelsea.ClubID,
		OriginalClubID: chelsea.ClubID,
	}
	chelsea.Squad = append(chelsea.Squad, dupZubiChelsea)
	chelsea.SquadSize = len(chelsea.Squad)

	dupZubiArsenal := &models.Player{
		PlayerID:       "P_DUP_ZUBI_ARS",
		FullName:       "Martin Zubimendi",
		Position:       "CDM",
		OVR:            84,
		Appearances:    3,
		Goals:          1,
		Assists:        1,
		CareerGoals:    4,
		CareerAssists:  5,
		CareerApps:     25,
		ClubID:         arsenal.ClubID,
		OriginalClubID: arsenal.ClubID,
	}
	arsenal.Squad = append(arsenal.Squad, dupZubiArsenal)
	arsenal.SquadSize = len(arsenal.Squad)

	// 2. Inject duplicate across Elite and Non-Elite clubs (not in PREFERRED_HOMES)
	fulham := dm.Clubs["EPL-FUL"]
	if fulham == nil {
		t.Fatal("fulham not found")
	}
	dupElite := &models.Player{
		PlayerID:       "P_DUP_ELITE",
		FullName:       "Elite Contested Player",
		Position:       "CB",
		OVR:            80,
		Appearances:    5,
		ClubID:         arsenal.ClubID,
		OriginalClubID: arsenal.ClubID,
	}
	dupNonElite := &models.Player{
		PlayerID:       "P_DUP_NONELITE",
		FullName:       "Elite Contested Player",
		Position:       "CB",
		OVR:            80,
		Appearances:    2,
		ClubID:         fulham.ClubID,
		OriginalClubID: fulham.ClubID,
	}
	arsenal.Squad = append(arsenal.Squad, dupElite)
	arsenal.SquadSize = len(arsenal.Squad)
	fulham.Squad = append(fulham.Squad, dupNonElite)
	fulham.SquadSize = len(fulham.Squad)

	// 3. Inject duplicate between two non-elite clubs (higher OVR/apps should win)
	crystalPalace := dm.Clubs["EPL-CRY"]
	if crystalPalace == nil {
		t.Fatal("crystal palace not found")
	}
	dupLow := &models.Player{
		PlayerID:       "P_DUP_LOW",
		FullName:       "NonElite Split Player",
		Position:       "ST",
		OVR:            74,
		Appearances:    2,
		ClubID:         fulham.ClubID,
		OriginalClubID: fulham.ClubID,
	}
	dupHigh := &models.Player{
		PlayerID:       "P_DUP_HIGH",
		FullName:       "NonElite Split Player",
		Position:       "ST",
		OVR:            78,
		Appearances:    8,
		Goals:          5,
		ClubID:         crystalPalace.ClubID,
		OriginalClubID: crystalPalace.ClubID,
	}
	fulham.Squad = append(fulham.Squad, dupLow)
	fulham.SquadSize = len(fulham.Squad)
	crystalPalace.Squad = append(crystalPalace.Squad, dupHigh)
	crystalPalace.SquadSize = len(crystalPalace.Squad)

	// 4. Inject intra-squad duplicate (same squad twice)
	dupIntra := &models.Player{
		PlayerID:       "P_DUP_INTRA",
		FullName:       "Intra Squad Duplicate",
		Position:       "LW",
		OVR:            76,
		Appearances:    1,
		ClubID:         arsenal.ClubID,
		OriginalClubID: arsenal.ClubID,
	}
	dupIntra2 := &models.Player{
		PlayerID:       "P_DUP_INTRA2",
		FullName:       "Intra Squad Duplicate",
		Position:       "LW",
		OVR:            77,
		Appearances:    4,
		Goals:          1,
		ClubID:         arsenal.ClubID,
		OriginalClubID: arsenal.ClubID,
	}
	arsenal.Squad = append(arsenal.Squad, dupIntra, dupIntra2)
	arsenal.SquadSize = len(arsenal.Squad)

	// Execute deduplication
	removed := dm.DedupePlayers()
	if removed != 4 {
		t.Errorf("expected 4 duplicates removed, got %d", removed)
	}

	// Verify Zubimendi kept in Arsenal, removed from Chelsea
	var chelseaHasZubi, arsenalHasZubi bool
	var arszubi *models.Player
	for _, p := range chelsea.Squad {
		if strings.EqualFold(p.FullName, "Martin Zubimendi") {
			chelseaHasZubi = true
		}
	}
	for _, p := range arsenal.Squad {
		if strings.EqualFold(p.FullName, "Martin Zubimendi") {
			arsenalHasZubi = true
			arszubi = p
		}
	}
	if chelseaHasZubi {
		t.Error("Martin Zubimendi was not removed from Chelsea squad")
	}
	if !arsenalHasZubi {
		t.Error("Martin Zubimendi was not kept in Arsenal squad")
	}
	if arszubi != nil && (arszubi.CareerGoals < 15 || arszubi.CareerApps < 80) {
		t.Errorf("Martin Zubimendi stats were not max-merged properly: %+v", arszubi)
	}

	// Verify Elite Contested Player kept in Arsenal, removed from Fulham
	var fulhamHasElite, arsenalHasElite bool
	for _, p := range fulham.Squad {
		if strings.EqualFold(p.FullName, "Elite Contested Player") {
			fulhamHasElite = true
		}
	}
	for _, p := range arsenal.Squad {
		if strings.EqualFold(p.FullName, "Elite Contested Player") {
			arsenalHasElite = true
		}
	}
	if fulhamHasElite || !arsenalHasElite {
		t.Errorf("elite priority failed: fulhamHas=%v, arsenalHas=%v", fulhamHasElite, arsenalHasElite)
	}

	// Verify NonElite Split Player kept in Crystal Palace (higher OVR/apps)
	var cpHas, fulhamHas bool
	for _, p := range crystalPalace.Squad {
		if strings.EqualFold(p.FullName, "NonElite Split Player") {
			cpHas = true
		}
	}
	for _, p := range fulham.Squad {
		if strings.EqualFold(p.FullName, "NonElite Split Player") {
			fulhamHas = true
		}
	}
	if !cpHas || fulhamHas {
		t.Errorf("highest OVR fallback failed: cpHas=%v, fulhamHas=%v", cpHas, fulhamHas)
	}

	// Verify intra-squad duplicate merged into single copy
	intraCount := 0
	for _, p := range arsenal.Squad {
		if strings.EqualFold(p.FullName, "Intra Squad Duplicate") {
			intraCount++
		}
	}
	if intraCount != 1 {
		t.Errorf("expected exactly 1 copy of Intra Squad Duplicate, got %d", intraCount)
	}

	// Invariant verification: exactly 0 duplicates anywhere
	seen := make(map[string]string)
	for _, c := range dm.ClubsList {
		if len(c.Squad) != c.SquadSize {
			t.Errorf("club %s SquadSize (%d) != len(Squad) (%d)", c.ClubID, c.SquadSize, len(c.Squad))
		}
		for _, p := range c.Squad {
			norm := strings.ToLower(strings.TrimSpace(p.FullName))
			if prevClub, exists := seen[norm]; exists {
				t.Fatalf("invariant violated: duplicate player %q found in %s and %s", p.FullName, prevClub, c.ClubID)
			}
			seen[norm] = c.ClubID
		}
	}
}

func TestInitializeEliteProdigies_CanonicalWonderkids(t *testing.T) {
	ge := newTestGrowthEngine()
	dm := NewDataManager("dataset.json", ge)
	if err := dm.LoadDataset(); err != nil {
		t.Fatalf("LoadDataset failed: %v", err)
	}
	dm.DedupePlayers()
	if err := dm.InitializeEliteProdigies(); err != nil {
		t.Fatalf("InitializeEliteProdigies failed: %v", err)
	}

	// Exactly 12 wonderkids
	if len(dm.Wonderkids) != 12 {
		t.Fatalf("expected 12 wonderkids, got %d", len(dm.Wonderkids))
	}

	expectedPotentials := map[string]int{
		"Venjamin Valerio":         96,
		"Maverick Cantalejo":       95,
		"Yeshua Emmanuel Gocotano": 93,
		"Izyan Levin Bantol":       95,
		"James Bernard Rizon":      94,
		"Reid Randell Libatan":     95,
		"Ashle Zylle Baguio":       95,
		"Cliergy Jave Lanticse":    94,
		"Ezail Zamora":             96,
		"Earl Josh Hernando":       94,
		"Rich Lorenz Suico":        94,
		"Jhed Anthony Guinita":     94,
	}

	for _, w := range dm.Wonderkids {
		// Age 14 invariant
		if w.Age != 14 {
			t.Errorf("wonderkid %s has age %d, expected 14", w.FullName, w.Age)
		}

		// Middle school invariant
		if w.Education != "middle_school" || w.EducationPending {
			t.Errorf("wonderkid %s education not middle_school or pending true: %s, %v", w.FullName, w.Education, w.EducationPending)
		}

		// Category FWD invariant (including CAMs)
		if w.Category != "FWD" {
			t.Errorf("wonderkid %s category %q, expected 'FWD'", w.FullName, w.Category)
		}

		// Stable WK_ ID invariant
		if !strings.HasPrefix(w.PlayerID, "WK_") {
			t.Errorf("wonderkid %s ID %q does not start with WK_", w.FullName, w.PlayerID)
		}
		expectedID := ProdigyStableID(w.FullName)
		if w.PlayerID != expectedID {
			t.Errorf("wonderkid %s ID %q != expected %q", w.FullName, w.PlayerID, expectedID)
		}

		// Flag invariant
		if !w.UniverseWonderkid {
			t.Errorf("wonderkid %s UniverseWonderkid is false", w.FullName)
		}

		// Potential invariant: [93, 96], never 99
		expPot, ok := expectedPotentials[w.FullName]
		if !ok {
			t.Fatalf("unknown wonderkid %s", w.FullName)
		}
		bio := ge.Biometrics[w.PlayerID]
		if bio == nil {
			t.Fatalf("wonderkid %s not found in GrowthEngine Biometrics", w.FullName)
		}
		if bio.Potential != expPot {
			t.Errorf("wonderkid %s potential %d != expected %d", w.FullName, bio.Potential, expPot)
		}
		if bio.Potential < 93 || bio.Potential > 96 {
			t.Errorf("wonderkid %s potential %d out of bounds [93, 96]", w.FullName, bio.Potential)
		}
		if bio.Potential == 99 {
			t.Errorf("wonderkid %s potential is 99 (forbidden!)", w.FullName)
		}

		// Composure and OVR calculated from GrowthEngine
		if w.Composure == 0 {
			t.Errorf("wonderkid %s composure is 0", w.FullName)
		}
		if w.OVR < 70 || w.OVR > 85 {
			t.Errorf("wonderkid %s OVR %d unexpected", w.FullName, w.OVR)
		}

		// Valuation clamped
		if w.MarketValueEUR < 300_000 || w.MarketValueEUR > 500_000_000 {
			t.Errorf("wonderkid %s MarketValueEUR %d out of clamped corridor", w.FullName, w.MarketValueEUR)
		}
	}
}

func TestRelocation_JhedAnthonyGuinita(t *testing.T) {
	ge := newTestGrowthEngine()
	dm := NewDataManager("dataset.json", ge)
	if err := dm.LoadDataset(); err != nil {
		t.Fatalf("LoadDataset failed: %v", err)
	}

	marseille := dm.Clubs["FL1-OM"]
	tottenham := dm.Clubs["EPL-TOT"]
	if marseille == nil || tottenham == nil {
		t.Fatal("FL1-OM or EPL-TOT missing from dataset")
	}

	dm.DedupePlayers()
	if err := dm.InitializeEliteProdigies(); err != nil {
		t.Fatalf("InitializeEliteProdigies failed: %v", err)
	}

	var jhed *models.Player
	for _, p := range tottenham.Squad {
		if strings.EqualFold(p.FullName, "Jhed Anthony Guinita") {
			jhed = p
			break
		}
	}
	if jhed == nil {
		t.Fatal("Jhed Anthony Guinita is not in the EPL-TOT squad")
	}
	if jhed.ClubID != "EPL-TOT" {
		t.Errorf("Jhed Anthony Guinita ClubID = %q, expected 'EPL-TOT'", jhed.ClubID)
	}

	// Check that he is NO LONGER in Marseille
	for _, p := range marseille.Squad {
		if strings.EqualFold(p.FullName, "Jhed Anthony Guinita") {
			t.Errorf("Jhed Anthony Guinita still present in Marseille squad!")
		}
	}

	// Check squad sizes match
	if marseille.SquadSize != len(marseille.Squad) {
		t.Errorf("Marseille SquadSize %d != len(Squad) %d", marseille.SquadSize, len(marseille.Squad))
	}
	if tottenham.SquadSize != len(tottenham.Squad) {
		t.Errorf("Tottenham SquadSize %d != len(Squad) %d", tottenham.SquadSize, len(tottenham.Squad))
	}
}

func TestAdoptU14Prodigies(t *testing.T) {
	ge := newTestGrowthEngine()
	dm := NewDataManager("dataset.json", ge)
	if err := dm.LoadDataset(); err != nil {
		t.Fatalf("LoadDataset failed: %v", err)
	}
	dm.DedupePlayers()
	_ = dm.InitializeEliteProdigies()

	if len(dm.Wonderkids) == 0 {
		t.Fatal("no wonderkids found")
	}

	wk := dm.Wonderkids[0]
	// Artificially change age and education
	wk.Age = 16
	wk.Education = "high_school"
	wk.PlayerID = "TEMPORARY_ID"

	changed := dm.AdoptU14Prodigies()
	if !changed {
		t.Errorf("expected AdoptU14Prodigies to return true after mutation")
	}
	if wk.Age != 14 {
		t.Errorf("expected age 14, got %d", wk.Age)
	}
	if wk.Education != "middle_school" {
		t.Errorf("expected education middle_school, got %q", wk.Education)
	}
	if !strings.HasPrefix(wk.PlayerID, "WK_") {
		t.Errorf("expected ID prefix WK_, got %q", wk.PlayerID)
	}

	// Calling again without changes should return false
	changedAgain := dm.AdoptU14Prodigies()
	if changedAgain {
		t.Errorf("expected AdoptU14Prodigies to return false when no mutation")
	}
}

func TestProdigyHomes_ApplyAndDescribe(t *testing.T) {
	ge := newTestGrowthEngine()
	dm := NewDataManager("dataset.json", ge)
	if err := dm.LoadDataset(); err != nil {
		t.Fatalf("LoadDataset failed: %v", err)
	}
	dm.DedupePlayers()
	_ = dm.InitializeEliteProdigies()

	// 1. Describe prodigy draw
	rows := dm.DescribeProdigyDraw(nil)
	if len(rows) != 12 {
		t.Fatalf("expected 12 rows in DescribeProdigyDraw, got %d", len(rows))
	}
	for _, r := range rows {
		if r.FullName == "" || r.ClubID == "" || r.ClubName == "" {
			t.Errorf("incomplete row in DescribeProdigyDraw: %+v", r)
		}
	}

	// 2. Shuffle prodigy homes and apply
	shuffled := ShuffleProdigyHomes(rand.New(rand.NewSource(123)))
	if len(shuffled) != 12 {
		t.Fatalf("expected 12 shuffled homes, got %d", len(shuffled))
	}
	dm.ApplyProdigyHomes(shuffled)

	// Verify each wonderkid moved to destination club at index 0
	for name, cid := range shuffled {
		club := dm.Clubs[cid]
		if club == nil {
			t.Fatalf("club %s not found", cid)
		}
		if len(club.Squad) == 0 || club.Squad[0].FullName != name {
			t.Errorf("wonderkid %s not at index 0 of club %s", name, cid)
		}
		if club.Squad[0].ClubID != cid {
			t.Errorf("wonderkid %s ClubID %s != %s", name, club.Squad[0].ClubID, cid)
		}
	}

	// 3. Fallback on invalid mapping (e.g. fewer than 12 clubs or non-elite club)
	invalidMapping := map[string]string{
		"Venjamin Valerio": "NON_EXISTENT_CLUB",
	}
	dm.ApplyProdigyHomes(invalidMapping)
	// Should fallback to default prodigy homes without crashing
	defaultHomes := DefaultProdigyHomes()
	for name, cid := range defaultHomes {
		club := dm.Clubs[cid]
		if club == nil || len(club.Squad) == 0 || club.Squad[0].FullName != name {
			t.Errorf("fallback failed for %s -> %s", name, cid)
		}
	}
}

func TestGetEliteClubs(t *testing.T) {
	ge := newTestGrowthEngine()
	dm := NewDataManager("dataset.json", ge)
	if err := dm.LoadDataset(); err != nil {
		t.Fatalf("LoadDataset failed: %v", err)
	}

	elite := dm.GetEliteClubs()
	if len(elite) != 12 {
		t.Fatalf("expected 12 elite clubs, got %d", len(elite))
	}
	seen := make(map[string]bool)
	for _, c := range elite {
		if seen[c.ClubID] {
			t.Errorf("duplicate elite club %s", c.ClubID)
		}
		seen[c.ClubID] = true
	}
}

func TestNewDatasetCareerBoot(t *testing.T) {
	ge := newTestGrowthEngine()
	dm := NewDataManager("dataset.json", ge)
	if len(dm.Clubs) != 96 {
		t.Fatalf("boot clubs = %d; want 96", len(dm.Clubs))
	}
	if got := dm.GetEliteClubs(); len(got) != 12 {
		t.Fatalf("elite clubs = %d; want 12", len(got))
	}
	if len(dm.Wonderkids) != 12 {
		t.Fatalf("boot wonderkids = %d; want 12", len(dm.Wonderkids))
	}
	for _, cfg := range EliteProdigyConfigs {
		if dm.Clubs[cfg.ClubID] == nil {
			t.Errorf("elite club %s missing from dataset", cfg.ClubID)
		}
		found := false
		for _, wk := range dm.Wonderkids {
			if !strings.EqualFold(wk.FullName, cfg.FullName) {
				continue
			}
			found = true
			if wk.Age != 14 || wk.Education != "middle_school" || !strings.HasPrefix(wk.PlayerID, "WK_") {
				t.Errorf("%s not snapped to career prodigy: age=%d edu=%q id=%q", cfg.FullName, wk.Age, wk.Education, wk.PlayerID)
			}
			if wk.ClubID != cfg.ClubID && dm.ProdigyHomes[cfg.FullName] != wk.ClubID {
				t.Errorf("%s club = %q; want default %q", cfg.FullName, wk.ClubID, cfg.ClubID)
			}
		}
		if !found {
			t.Errorf("canonical prodigy %s missing after boot", cfg.FullName)
		}
	}
}

func TestMarketBaseline_Snapping(t *testing.T) {
	ge := newTestGrowthEngine()
	dm := NewDataManager("dataset.json", ge)
	if err := dm.LoadDataset(); err != nil {
		t.Fatalf("LoadDataset failed: %v", err)
	}
	dm.DedupePlayers()
	_ = dm.InitializeEliteProdigies()
	dm.ResetMarketToBaseline()

	for _, c := range dm.ClubsList {
		for _, p := range c.Squad {
			if p.MarketValueEUR < 300_000 {
				t.Errorf("player %s market value %d below absolute min 300k", p.FullName, p.MarketValueEUR)
			}
			if p.MarketValueEUR > 500_000_000 {
				t.Errorf("player %s market value %d above absolute max 500M", p.FullName, p.MarketValueEUR)
			}
		}
	}
}

func TestYouthIntake_CapAndGeneration(t *testing.T) {
	ge := newTestGrowthEngine()
	dm := NewDataManager("dataset.json", ge)
	if err := dm.LoadDataset(); err != nil {
		t.Fatalf("LoadDataset failed: %v", err)
	}
	dm.SetSeed(999)

	arsenal := dm.Clubs["EPL-ARS"]
	if arsenal == nil {
		t.Fatal("EPL-ARS not found")
	}

	initialSize := len(arsenal.Squad)
	// 1. Single club intake
	graduates, err := dm.RunYouthIntake("EPL-ARS")
	if err != nil {
		t.Fatalf("RunYouthIntake failed: %v", err)
	}
	if len(graduates) < 2 || len(graduates) > 4 {
		t.Errorf("expected 2-4 graduates, got %d", len(graduates))
	}
	if len(arsenal.Squad) != initialSize+len(graduates) {
		t.Errorf("squad size mismatch: expected %d, got %d", initialSize+len(graduates), len(arsenal.Squad))
	}

	for _, grad := range graduates {
		if !strings.HasPrefix(grad.PlayerID, "AC_EPL-ARS_") {
			t.Errorf("graduate ID %q does not match expected prefix", grad.PlayerID)
		}
		if grad.Age < 16 || grad.Age > 18 {
			t.Errorf("graduate age %d out of range [16, 18]", grad.Age)
		}
		if grad.MarketValueEUR < 300_000 || grad.MarketValueEUR > 500_000_000 {
			t.Errorf("graduate market value %d out of clamped corridor", grad.MarketValueEUR)
		}
		bio := ge.Biometrics[grad.PlayerID]
		if bio == nil {
			t.Errorf("graduate %s not enrolled in GrowthEngine", grad.PlayerID)
		}
	}

	// 2. Squad Cap 34 enforcement: pad arsenal to 34 players
	for len(arsenal.Squad) < 34 {
		arsenal.Squad = append(arsenal.Squad, &models.Player{
			PlayerID:       "PAD_PLAYER",
			FullName:       "Pad Player",
			Position:       "CM",
			OVR:            70,
			Age:            25,
			ClubID:         "EPL-ARS",
			OriginalClubID: "EPL-ARS",
		})
	}
	arsenal.SquadSize = len(arsenal.Squad)

	gradsAtCap, err := dm.RunYouthIntake("EPL-ARS")
	if err != nil {
		t.Fatalf("intake at cap failed: %v", err)
	}
	if len(gradsAtCap) != 0 {
		t.Errorf("expected 0 graduates when squad cap 34 is reached, got %d", len(gradsAtCap))
	}
	if len(arsenal.Squad) != 34 {
		t.Errorf("squad size exceeded 34: got %d", len(arsenal.Squad))
	}

	// 3. Squad cap partial enforcement: pad to 33, request 3
	arsenal.Squad = arsenal.Squad[:33]
	arsenal.SquadSize = len(arsenal.Squad)
	gradsPartial, err := dm.RunYouthIntakeWithCount("EPL-ARS", 3)
	if err != nil {
		t.Fatalf("partial intake failed: %v", err)
	}
	if len(gradsPartial) != 1 {
		t.Errorf("expected exactly 1 graduate to reach cap 34, got %d", len(gradsPartial))
	}
	if len(arsenal.Squad) != 34 {
		t.Errorf("squad size should be exactly 34, got %d", len(arsenal.Squad))
	}

	// 4. Unknown club error
	_, errUnknown := dm.RunYouthIntake("INVALID_CLUB_ID")
	if errUnknown == nil {
		t.Error("expected error for invalid club ID, got nil")
	}

	// 5. Global intake for all clubs ("all")
	allGrads, errAll := dm.RunYouthIntake("all")
	if errAll != nil {
		t.Fatalf("RunYouthIntake('all') failed: %v", errAll)
	}
	if len(allGrads) == 0 {
		t.Errorf("expected global graduates, got 0")
	}
	// Verify no club exceeds 34
	for _, c := range dm.ClubsList {
		if len(c.Squad) > 34 {
			t.Errorf("club %s squad size %d exceeds cap 34", c.ClubID, len(c.Squad))
		}
	}
}

func TestYouthIntake_GoldenGeneration(t *testing.T) {
	ge := newTestGrowthEngine()
	// Seed 42 gives golden generation hits
	rng := rand.New(rand.NewSource(42))
	testClub := &models.Club{
		ClubID:    "TEST-CLUB",
		ClubName:  "Test FC",
		SquadSize: 20,
		Squad:     make([]*models.Player, 20),
	}
	for i := range testClub.Squad {
		testClub.Squad[i] = &models.Player{
			PlayerID: "P",
			FullName: "Existing Player",
		}
	}

	foundGolden := false
	for attempt := 0; attempt < 50; attempt++ {
		c := &models.Club{
			ClubID:    "TEST-CLUB",
			SquadSize: 20,
			Squad:     make([]*models.Player, 20),
		}
		count := 3
		grads, err := RunYouthIntakeClubs([]*models.Club{c}, &count, ge, rng)
		if err != nil {
			t.Fatalf("RunYouthIntakeClubs failed: %v", err)
		}
		for _, g := range grads {
			bio := ge.Biometrics[g.PlayerID]
			if bio != nil && bio.Potential >= 90 {
				foundGolden = true
				if g.OVR < 72 || g.OVR > 78 {
					t.Errorf("golden generation OVR %d out of expected range [72, 78]", g.OVR)
				}
				if bio.Potential < 90 || bio.Potential > 95 {
					t.Errorf("golden generation potential %d out of expected range [90, 95]", bio.Potential)
				}
				break
			}
		}
		if foundGolden {
			break
		}
	}
	if !foundGolden {
		t.Errorf("did not find golden generation after 50 attempts")
	}
}

func TestYouthIntake_NameCollisionHandling(t *testing.T) {
	ge := newTestGrowthEngine()

	// Pre-fill another club with all possible (first, last) combinations to saturate taken set
	otherClub := &models.Club{
		ClubID:    "SATURATED-CLUB",
		ClubName:  "Saturated FC",
		SquadSize: 34,
		Squad:     make([]*models.Player, 0, len(AcademyFirst)*len(AcademyLast)),
	}
	for _, first := range AcademyFirst {
		for _, last := range AcademyLast {
			otherClub.Squad = append(otherClub.Squad, &models.Player{
				PlayerID: "EXISTING",
				FullName: first + " " + last,
			})
		}
	}
	otherClub.SquadSize = len(otherClub.Squad)

	testClub := &models.Club{
		ClubID:    "COLLISION-CLUB",
		ClubName:  "Collision FC",
		SquadSize: 0,
		Squad:     make([]*models.Player, 0),
	}

	rng := rand.New(rand.NewSource(1))
	count := 2
	grads, err := RunYouthIntakeClubs([]*models.Club{otherClub, testClub}, &count, ge, rng)
	if err != nil {
		t.Fatalf("intake with collisions failed: %v", err)
	}
	if len(grads) != 2 {
		t.Fatalf("expected 2 grads, got %d", len(grads))
	}
	for _, g := range grads {
		if g.FullName == "" {
			t.Errorf("empty graduate full name")
		}
		// Since all base names are taken, name must have numeric suffix (10-99)
		parts := strings.Fields(g.FullName)
		if len(parts) < 3 {
			t.Errorf("expected numeric suffix appended after collision: %q", g.FullName)
		}
	}
}

func TestLoadDataset_MissingFile(t *testing.T) {
	dm := NewDataManager("non_existent_file_path.json", nil)
	err := dm.LoadDataset()
	if err == nil {
		t.Error("expected error for non existent file path, got nil")
	}
	if !os.IsNotExist(err) && !strings.Contains(err.Error(), "missing") {
		t.Errorf("expected descriptive error mentioning missing file, got: %v", err)
	}
}

func TestDedupePlayers_ComprehensivePointerAliasingAndMixedInstances(t *testing.T) {
	ge := newTestGrowthEngine()
	dm := NewDataManager("dataset.json", ge)
	if err := dm.LoadDataset(); err != nil {
		t.Fatalf("LoadDataset failed: %v", err)
	}

	ars := dm.Clubs["EPL-ARS"]
	che := dm.Clubs["EPL-CHE"]
	liv := dm.Clubs["EPL-LIV"]
	tot := dm.Clubs["EPL-TOT"]
	if ars == nil || che == nil || liv == nil || tot == nil {
		t.Fatal("required test clubs not found")
	}

	// 1. Intra-club multi-pointer aliasing: append same pointer 3 times to Chelsea
	ptrIntra := &models.Player{
		PlayerID:    "P_INTRA_TEST",
		FullName:    "Intra Aliased Pointer",
		Position:    "CB",
		OVR:         76,
		ClubID:      "EPL-CHE",
		Appearances: 12,
		Goals:       2,
	}
	che.Squad = append(che.Squad, ptrIntra, ptrIntra, ptrIntra)
	che.SquadSize = len(che.Squad)

	// 2. Cross-club pointer sharing across Arsenal and Liverpool
	ptrShared := &models.Player{
		PlayerID:    "P_SHARED_TEST",
		FullName:    "Cross Club Shared Pointer",
		Position:    "RW",
		OVR:         79,
		Appearances: 15,
		CareerGoals: 25,
	}
	ars.Squad = append(ars.Squad, ptrShared)
	liv.Squad = append(liv.Squad, ptrShared)
	ars.SquadSize = len(ars.Squad)
	liv.SquadSize = len(liv.Squad)

	// 3. Mixed: distinct struct with identical name in Tottenham with higher appearances
	distinctCopy := &models.Player{
		PlayerID:    "P_DISTINCT_TEST",
		FullName:    "Cross Club Shared Pointer",
		Position:    "RW",
		OVR:         77,
		Appearances: 22,
		CareerGoals: 10,
		BestGoals:   5,
	}
	tot.Squad = append(tot.Squad, distinctCopy)
	tot.SquadSize = len(tot.Squad)

	removed := dm.DedupePlayers()
	if removed < 4 {
		t.Errorf("expected at least 4 duplicates removed (2 intra + 2 cross-club), got %d", removed)
	}

	// Verify Chelsea has strictly 1 copy of Intra Aliased Pointer
	cheCount := 0
	for _, p := range che.Squad {
		if strings.EqualFold(p.FullName, "Intra Aliased Pointer") {
			cheCount++
		}
	}
	if cheCount != 1 {
		t.Errorf("expected exactly 1 copy of Intra Aliased Pointer in Chelsea, got %d", cheCount)
	}

	// Verify Cross Club Shared Pointer exists in strictly 1 club
	totalSharedCount := 0
	var canonicalShared *models.Player
	for _, c := range dm.ClubsList {
		for _, p := range c.Squad {
			if strings.EqualFold(p.FullName, "Cross Club Shared Pointer") {
				totalSharedCount++
				canonicalShared = p
			}
		}
		if c.SquadSize != len(c.Squad) {
			t.Errorf("club %s SquadSize %d != len(Squad) %d", c.ClubID, c.SquadSize, len(c.Squad))
		}
	}
	if totalSharedCount != 1 {
		t.Errorf("expected exactly 1 copy of Cross Club Shared Pointer in entire DB, got %d", totalSharedCount)
	}
	if canonicalShared != nil {
		if canonicalShared.Appearances < 22 {
			t.Errorf("expected Appearances max-merged to at least 22, got %d", canonicalShared.Appearances)
		}
		if canonicalShared.CareerGoals < 25 {
			t.Errorf("expected CareerGoals max-merged to at least 25, got %d", canonicalShared.CareerGoals)
		}
		if canonicalShared.BestGoals < 5 {
			t.Errorf("expected BestGoals max-merged to at least 5, got %d", canonicalShared.BestGoals)
		}
	}
}

func TestInitializeEliteProdigies_PointerAliasingCleanup(t *testing.T) {
	ge := newTestGrowthEngine()
	dm := NewDataManager("dataset.json", ge)
	if err := dm.LoadDataset(); err != nil {
		t.Fatalf("LoadDataset failed: %v", err)
	}

	bar := dm.Clubs["LAL-BAR"]
	rma := dm.Clubs["LAL-RMA"]
	if bar == nil || rma == nil {
		t.Fatal("Barcelona or Real Madrid not found")
	}

	// Venjamin Valerio is Barcelona's franchise prodigy
	// Inject duplicate pointers into Real Madrid AND Barcelona
	valerioPtr := &models.Player{
		PlayerID: "WK_VENJAMIN_VALERIO",
		FullName: "Venjamin Valerio",
		Position: "ST",
		OVR:      70,
	}
	bar.Squad = append(bar.Squad, valerioPtr, valerioPtr)
	rma.Squad = append(rma.Squad, valerioPtr)
	bar.SquadSize = len(bar.Squad)
	rma.SquadSize = len(rma.Squad)

	if err := dm.InitializeEliteProdigies(); err != nil {
		t.Fatalf("InitializeEliteProdigies failed: %v", err)
	}

	// Venjamin Valerio must exist strictly ONCE in Barcelona and ZERO times in Real Madrid
	barCount := 0
	for _, p := range bar.Squad {
		if strings.EqualFold(p.FullName, "Venjamin Valerio") {
			barCount++
		}
	}
	rmaCount := 0
	for _, p := range rma.Squad {
		if strings.EqualFold(p.FullName, "Venjamin Valerio") {
			rmaCount++
		}
	}

	if barCount != 1 {
		t.Errorf("expected exactly 1 Venjamin Valerio in Barcelona, got %d", barCount)
	}
	if rmaCount != 0 {
		t.Errorf("expected 0 Venjamin Valerio in Real Madrid, got %d", rmaCount)
	}
	if bar.SquadSize != len(bar.Squad) {
		t.Errorf("Barcelona SquadSize mismatch: %d vs %d", bar.SquadSize, len(bar.Squad))
	}
	if rma.SquadSize != len(rma.Squad) {
		t.Errorf("Real Madrid SquadSize mismatch: %d vs %d", rma.SquadSize, len(rma.Squad))
	}
}
