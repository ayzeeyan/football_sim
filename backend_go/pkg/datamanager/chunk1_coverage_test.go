package datamanager

import (
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"football_sim/pkg/growth"
	"football_sim/pkg/models"
)

// Supplemental Chunk 1 coverage: DataManager edge paths, youth-intake name
// exhaustion, prodigy-home fallbacks, and dedupe preference branches.

func chunk1CovGE() *growth.GrowthEngine {
	return growth.NewGrowthEngine(4242)
}

func chunk1CovFreshDM(t *testing.T) *DataManager {
	t.Helper()
	dm := NewDataManager("dataset.json", chunk1CovGE())
	if err := dm.LoadDataset(); err != nil {
		t.Fatalf("LoadDataset failed: %v", err)
	}
	return dm
}

func TestChunk1CovRNGAndConstructors(t *testing.T) {
	dm := NewDataManager("", nil) // no path, nil engine
	if dm.GrowthEngine == nil {
		t.Errorf("nil engine should be replaced with a default")
	}
	dm.SetRNG(nil) // no-op, must not panic
	dm.SetRNG(rand.New(rand.NewSource(9)))
	dm.SetSeed(123)
	if dm.rng == nil {
		t.Errorf("SetSeed should install an RNG")
	}
	before := len(dm.ClubsList)
	missing := NewDataManager("definitely_missing_xyz.json", chunk1CovGE())
	if len(missing.ClubsList) != 0 {
		t.Errorf("missing dataset should yield no clubs, got %d", before)
	}
	_ = before
	if a, b := minInt(1, 2), minInt(3, 2); a != 1 || b != 2 {
		t.Errorf("minInt wrong: %d %d", a, b)
	}
	if a, b := maxInt(1, 2), maxInt(3, 2); a != 2 || b != 3 {
		t.Errorf("maxInt wrong: %d %d", a, b)
	}
}

func TestChunk1CovResolveDatasetPath(t *testing.T) {
	abs, err := filepath.Abs(filepath.Join("..", "..", "..", "dataset.json"))
	if err != nil {
		t.Fatal(err)
	}
	if got := resolveDatasetPath(abs); got != abs {
		t.Errorf("direct hit should echo path, got %q", got)
	}
	// From pkg/datamanager, "dataset.json" resolves via parent fallback.
	if got := resolveDatasetPath("dataset.json"); !strings.HasSuffix(got, "dataset.json") {
		t.Errorf("fallback resolution wrong: %q", got)
	} else if _, err := os.Stat(got); err != nil {
		t.Errorf("resolved path does not exist: %q", got)
	}
	if got := resolveDatasetPath("definitely_missing_xyz.json"); got != "definitely_missing_xyz.json" {
		t.Errorf("miss should echo input, got %q", got)
	}
}

func TestChunk1CovLoadDatasetErrors(t *testing.T) {
	dm := NewDataManager("", chunk1CovGE())
	dm.JSONPath = "definitely_missing_xyz.json"
	if err := dm.LoadDataset(); err == nil {
		t.Errorf("expected missing-file error")
	}
	bad, err := os.CreateTemp("", "bad-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(bad.Name())
	if _, err := bad.WriteString("{oops"); err != nil {
		t.Fatal(err)
	}
	bad.Close()
	dm.JSONPath = bad.Name()
	if err := dm.LoadDataset(); err == nil {
		t.Errorf("expected parse error for malformed JSON")
	}
	// Empty season defaults to 2026-27.
	minimal, err := os.CreateTemp("", "minimal-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(minimal.Name())
	if _, err := minimal.WriteString(`{"clubs": []}`); err != nil {
		t.Fatal(err)
	}
	minimal.Close()
	dm.JSONPath = minimal.Name()
	if err := dm.LoadDataset(); err != nil {
		t.Fatalf("minimal load failed: %v", err)
	}
	if dm.Metadata.Season != "2026-27" {
		t.Errorf("default season = %q; want 2026-27", dm.Metadata.Season)
	}
	if len(dm.ClubsList) != 0 {
		t.Errorf("minimal clubs = %d; want 0", len(dm.ClubsList))
	}
}

func TestChunk1CovShuffleProdigyHomes(t *testing.T) {
	for _, r := range []*rand.Rand{nil, rand.New(rand.NewSource(5))} {
		homes := ShuffleProdigyHomes(r)
		if len(homes) != len(EliteProdigyConfigs) {
			t.Fatalf("shuffled homes = %d; want %d", len(homes), len(EliteProdigyConfigs))
		}
		seenClubs := map[string]bool{}
		for _, cid := range homes {
			if seenClubs[cid] {
				t.Errorf("club %s assigned twice", cid)
			}
			seenClubs[cid] = true
		}
	}
}

func TestChunk1CovDedupePreferences(t *testing.T) {
	dm := chunk1CovFreshDM(t)

	// PreferredHomes hit: duplicate Cole Palmer outside Chelsea stays at Chelsea.
	che := dm.Clubs["EPL-CHE"]
	var palmer *models.Player
	for _, p := range che.Squad {
		if strings.EqualFold(p.FullName, "Cole Palmer") {
			palmer = p
			break
		}
	}
	if palmer == nil {
		t.Skip("Cole Palmer not at Chelsea in this dataset snapshot")
	}
	other := dm.Clubs["EPL-ARS"]
	dup := *palmer
	dup.PlayerID = "DUP_PALMER"
	dup.OVR = palmer.OVR + 5 // even stronger copy elsewhere
	other.Squad = append(other.Squad, &dup)
	other.SquadSize = len(other.Squad)
	dm.DedupePlayers()
	stillAtChe := false
	for _, p := range che.Squad {
		if strings.EqualFold(p.FullName, "Cole Palmer") {
			stillAtChe = true
		}
	}
	for _, p := range other.Squad {
		if strings.EqualFold(p.FullName, "Cole Palmer") {
			t.Errorf("preferred-home duplicate should be purged from %s", other.ClubID)
		}
	}
	if !stillAtChe {
		t.Errorf("Cole Palmer should remain at Chelsea (preferred home)")
	}

	// Best-OVR fallback: duplicate a non-preferred name across two non-elite clubs.
	dm2 := chunk1CovFreshDM(t)
	var victim *models.Player
	var srcClub *models.Club
	for _, c := range dm2.ClubsList {
		if c.ClubID == "EPL-CHE" || c.ClubID == "EPL-ARS" {
			continue
		}
		if len(c.Squad) > 0 {
			srcClub = c
			victim = c.Squad[0]
			break
		}
	}
	if victim == nil {
		t.Skip("no victim found")
	}
	var dstClub *models.Club
	for _, c := range dm2.ClubsList {
		if c != srcClub && c.ClubID != "EPL-CHE" {
			dstClub = c
			break
		}
	}
	strong := *victim
	strong.PlayerID = "DUP_STRONG"
	strong.OVR = 95
	dstClub.Squad = append(dstClub.Squad, &strong)
	dstClub.SquadSize = len(dstClub.Squad)
	dm2.DedupePlayers()
	count := 0
	for _, c := range dm2.ClubsList {
		for _, p := range c.Squad {
			if strings.EqualFold(p.FullName, victim.FullName) {
				count++
			}
		}
	}
	if count != 1 {
		t.Errorf("dedupe should leave exactly 1 copy, left %d", count)
	}
}

func TestChunk1CovInitializeEliteProdigiesEdges(t *testing.T) {
	// Bogus home club -> skipped (continue branch).
	dm := chunk1CovFreshDM(t)
	dm.ProdigyHomes = DefaultProdigyHomes()
	dm.ProdigyHomes["Venjamin Valerio"] = "NOPE-NOPE"
	if err := dm.InitializeEliteProdigies(); err != nil {
		t.Fatalf("init failed: %v", err)
	}
	if len(dm.Wonderkids) != len(EliteProdigyConfigs)-1 {
		t.Errorf("bogus home should skip one prodigy, got %d wonderkids", len(dm.Wonderkids))
	}

	// Nil growth engine path: fields still stamped, no registration.
	dm2 := chunk1CovFreshDM(t)
	dm2.GrowthEngine = nil
	if err := dm2.InitializeEliteProdigies(); err != nil {
		t.Fatalf("nil-engine init failed: %v", err)
	}
	if len(dm2.Wonderkids) != len(EliteProdigyConfigs) {
		t.Errorf("nil-engine wonderkids = %d; want %d", len(dm2.Wonderkids), len(EliteProdigyConfigs))
	}
	for _, p := range dm2.Wonderkids {
		if p.Age != 14 || p.Education != "middle_school" || !p.UniverseWonderkid {
			t.Errorf("nil-engine prodigy misconfigured: %+v", p)
		}
	}
}

func TestChunk1CovAdoptU14Edges(t *testing.T) {
	dm := chunk1CovFreshDM(t)
	if err := dm.InitializeEliteProdigies(); err != nil {
		t.Fatal(err)
	}
	// Unknown name in Wonderkids is skipped; clean state reports no change.
	dm.Wonderkids = append(dm.Wonderkids, &models.Player{FullName: "Nobody Here", Age: 20})
	if dm.AdoptU14Prodigies() {
		t.Errorf("clean prodigies should report no change")
	}
	// Tampered prodigy is restored (nil engine variant too).
	dm.Wonderkids[0].Age = 16
	dm.Wonderkids[0].Education = "high_school"
	dm.Wonderkids[0].PlayerID = "WRONG_ID"
	if !dm.AdoptU14Prodigies() {
		t.Errorf("tampered prodigy should report change")
	}
	if dm.Wonderkids[0].Age != 14 || dm.Wonderkids[0].PlayerID != ProdigyStableID(dm.Wonderkids[0].FullName) {
		t.Errorf("prodigy not restored: %+v", dm.Wonderkids[0])
	}
	dm.GrowthEngine = nil
	dm.Wonderkids[1].Age = 17
	if !dm.AdoptU14Prodigies() {
		t.Errorf("nil-engine adopt should still report change")
	}
	if dm.Wonderkids[1].Age != 14 {
		t.Errorf("nil-engine adopt should still reset age")
	}
}

func TestChunk1CovApplyAndDescribeHomes(t *testing.T) {
	dm := chunk1CovFreshDM(t)
	if err := dm.InitializeEliteProdigies(); err != nil {
		t.Fatal(err)
	}
	// Invalid homes (all to one club) fall back to defaults.
	bogus := map[string]string{}
	for _, cfg := range EliteProdigyConfigs {
		bogus[cfg.FullName] = "EPL-ARS"
	}
	dm.ApplyProdigyHomes(bogus)
	if len(dm.Wonderkids) != len(EliteProdigyConfigs) {
		t.Errorf("fallback homes wonderkids = %d", len(dm.Wonderkids))
	}
	// Missing prodigy in squads exercises the p == nil skip.
	dm2 := chunk1CovFreshDM(t)
	if err := dm2.InitializeEliteProdigies(); err != nil {
		t.Fatal(err)
	}
	lost := EliteProdigyConfigs[0].FullName
	for _, c := range dm2.ClubsList {
		kept := c.Squad[:0:0]
		kept = kept[:0]
		for _, p := range c.Squad {
			if !strings.EqualFold(p.FullName, lost) {
				kept = append(kept, p)
			}
		}
		c.Squad = kept
		c.SquadSize = len(kept)
	}
	dm2.ApplyProdigyHomes(DefaultProdigyHomes())
	if len(dm2.Wonderkids) != len(EliteProdigyConfigs)-1 {
		t.Errorf("lost prodigy should be skipped, got %d", len(dm2.Wonderkids))
	}

	// Describe fallbacks.
	rows := dm.DescribeProdigyDraw(nil)
	if len(rows) != len(EliteProdigyConfigs) {
		t.Errorf("describe rows = %d", len(rows))
	}
	empty := chunk1CovFreshDM(t)
	empty.ProdigyHomes = map[string]string{}
	rows2 := empty.DescribeProdigyDraw(nil)
	if len(rows2) != len(EliteProdigyConfigs) {
		t.Errorf("default describe rows = %d", len(rows2))
	}
	partial := map[string]string{EliteProdigyConfigs[0].FullName: "NOPE-CLUB"}
	rows3 := empty.DescribeProdigyDraw(partial)
	if rows3[0].ClubName != "NOPE-CLUB" || rows3[0].ShortName != "NOPE-CLUB" {
		t.Errorf("unknown club should echo CID: %+v", rows3[0])
	}
}

func TestChunk1CovGetEliteClubsMissing(t *testing.T) {
	dm := chunk1CovFreshDM(t)
	delete(dm.Clubs, "LAL-BAR")
	if got := dm.GetEliteClubs(); len(got) != len(EliteProdigyConfigs)-1 {
		t.Errorf("elite clubs with one missing = %d", len(got))
	}
}

func TestChunk1CovYouthIntakeBranches(t *testing.T) {
	dm := chunk1CovFreshDM(t)
	if err := dm.InitializeEliteProdigies(); err != nil {
		t.Fatal(err)
	}
	// Unknown club errors.
	if _, err := dm.RunYouthIntakeWithCount("NOPE-CLUB", 2); err == nil {
		t.Errorf("expected unknown-club error")
	}
	if _, err := dm.RunYouthIntake("NOPE-CLUB"); err == nil {
		t.Errorf("expected unknown-club error (RunYouthIntake)")
	}
	// Explicit count on one club.
	signed, err := dm.RunYouthIntakeWithCount("EPL-ARS", 2)
	if err != nil || len(signed) != 2 {
		t.Errorf("explicit intake = %d, err=%v; want 2", len(signed), err)
	}
	// Full-squad club is skipped.
	full := dm.Clubs["EPL-ARS"]
	for len(full.Squad) < 34 {
		full.Squad = append(full.Squad, &models.Player{
			PlayerID: "FILLER_" + string(rune(len(full.Squad))),
			FullName: "Filler Player",
			Position: "CM",
			Category: "MID",
			OVR:      65,
			Age:      24,
		})
	}
	full.SquadSize = len(full.Squad)
	before := len(full.Squad)
	out, err := dm.RunYouthIntakeWithCount("EPL-ARS", 3)
	if err != nil || len(out) != 0 || len(full.Squad) != before {
		t.Errorf("capped club should be skipped: out=%d err=%v size %d->%d", len(out), err, before, len(full.Squad))
	}
	// Package-level wrapper with nil rng + nil global entry.
	c, _ := dm.RunYouthIntakeWithCount("EPL-CHE", 1)
	clubs := []*models.Club{dm.Clubs["EPL-CHE"]}
	_ = c
	count := 1
	wrapped, err := RunYouthIntakeClubs(clubs, &count, dm.GrowthEngine, nil)
	if err != nil || len(wrapped) == 0 {
		t.Errorf("wrapper intake failed: %v %d", err, len(wrapped))
	}
	withNil, err := RunYouthIntakeClubs(clubs, nil, nil, rand.New(rand.NewSource(3)))
	if err != nil || len(withNil) == 0 {
		t.Errorf("nil-ge intake failed: %v", err)
	}
	_ = runYouthIntakeNilGlobal(t)
	// Empty ClubsList falls back to targetClubs.
	bare := &DataManager{Clubs: map[string]*models.Club{"SOLO": {
		ClubID: "SOLO", ClubName: "Solo FC", Squad: []*models.Player{},
	}}, GrowthEngine: chunk1CovGE(), rng: rand.New(rand.NewSource(4))}
	bare.ClubsList = nil
	got, err := bare.RunYouthIntakeWithCount("SOLO", 2)
	if err != nil || len(got) != 2 {
		t.Errorf("bare-DM intake = %d, err=%v; want 2", len(got), err)
	}
	// Golden-generation sweep: both golden and standard regens observable.
	seenGolden, seenStandard := false, false
	for seed := int64(1); seed <= 60; seed++ {
		g := chunk1CovFreshDM(t)
		g.SetSeed(seed)
		s, err := g.RunYouthIntakeWithCount("EPL-TOT", 4)
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range s {
			if p.OVR >= 72 {
				seenGolden = true
			} else {
				seenStandard = true
			}
		}
	}
	if !seenGolden || !seenStandard {
		t.Errorf("should observe golden (%v) and standard (%v) regens", seenGolden, seenStandard)
	}
}

func runYouthIntakeNilGlobal(t *testing.T) error {
	t.Helper()
	dm := chunk1CovFreshDM(t)
	target := []*models.Club{dm.Clubs["EPL-LIV"]}
	global := []*models.Club{nil, dm.Clubs["EPL-LIV"]}
	count := 1
	_, err := runYouthIntakeInternal(target, global, &count, dm.GrowthEngine, rand.New(rand.NewSource(8)))
	return err
}

func TestChunk1CovYouthIntakeNameExhaustion(t *testing.T) {
	// Saturate the global name pool so the 40-attempt loop fails and the
	// numeric-suffix fallback engages.
	big := &models.Club{ClubID: "POOL", ClubName: "Pool FC"}
	for _, f := range AcademyFirst {
		for _, l := range AcademyLast {
			big.Squad = append(big.Squad, &models.Player{FullName: f + " " + l})
		}
	}
	fresh := &models.Club{ClubID: "FRESH", ClubName: "Fresh FC"}
	count := 1
	signed, err := runYouthIntakeInternal(
		[]*models.Club{fresh},
		[]*models.Club{big, fresh},
		&count, nil, rand.New(rand.NewSource(11)),
	)
	if err != nil {
		t.Fatalf("exhaustion intake failed: %v", err)
	}
	if len(signed) != 1 {
		t.Fatalf("exhaustion intake = %d; want 1", len(signed))
	}
	parts := strings.Split(signed[0].FullName, " ")
	if len(parts) != 3 {
		t.Errorf("exhausted name should carry numeric suffix, got %q", signed[0].FullName)
	}
}

// chunk1CovInit relocates/copies a wonderkid so tests can build duplicate scenarios.
func chunk1CovInit(t *testing.T) *DataManager {
	t.Helper()
	dm := chunk1CovFreshDM(t)
	if err := dm.InitializeEliteProdigies(); err != nil {
		t.Fatalf("init failed: %v", err)
	}
	return dm
}

func chunk1CovCountName(dm *DataManager, name string) (int, string) {
	count := 0
	clubID := ""
	for _, c := range dm.ClubsList {
		for _, p := range c.Squad {
			if strings.EqualFold(p.FullName, name) {
				count++
				clubID = c.ClubID
			}
		}
	}
	return count, clubID
}

func TestChunk1CovDedupeWonderkidNoWKIDs(t *testing.T) {
	// No WK_-prefixed copies, but UniverseWonderkid set: the flag fallback
	// selects the canonical pool (covers the wkCopies flag branch).
	dm := chunk1CovInit(t)
	var canon *models.Player
	for _, p := range dm.Clubs["LAL-BAR"].Squad {
		if strings.EqualFold(p.FullName, "Venjamin Valerio") {
			canon = p
		}
	}
	if canon == nil {
		t.Fatal("Venjamin Valerio not at LAL-BAR")
	}
	canon.PlayerID = "VENJ_PLAIN"
	dup := *canon
	dup.PlayerID = "VENJ_DUP"
	liv := dm.Clubs["EPL-LIV"]
	liv.Squad = append(liv.Squad, &dup)
	liv.SquadSize = len(liv.Squad)
	dm.DedupePlayers()
	if n, at := chunk1CovCountName(dm, "Venjamin Valerio"); n != 1 || at != "LAL-BAR" {
		t.Errorf("flag-based dedupe left %d copies at %q; want 1 at LAL-BAR", n, at)
	}
}

func TestChunk1CovDedupeWonderkidNoFlagsAtHome(t *testing.T) {
	// No WK_ IDs and no wonderkid flags: searchPool falls back to all copies
	// and the home-club copy is kept.
	dm := chunk1CovInit(t)
	var canon *models.Player
	for _, p := range dm.Clubs["SEA-NAP"].Squad {
		if strings.EqualFold(p.FullName, "Ezail Zamora") {
			canon = p
		}
	}
	if canon == nil {
		t.Fatal("Ezail Zamora not at SEA-NAP")
	}
	canon.PlayerID = "EZ_PLAIN"
	canon.UniverseWonderkid = false
	dup := *canon
	dup.PlayerID = "EZ_DUP"
	mci := dm.Clubs["EPL-MCI"]
	mci.Squad = append(mci.Squad, &dup)
	mci.SquadSize = len(mci.Squad)
	dm.DedupePlayers()
	if n, at := chunk1CovCountName(dm, "Ezail Zamora"); n != 1 || at != "SEA-NAP" {
		t.Errorf("unflagged dedupe left %d copies at %q; want 1 at SEA-NAP", n, at)
	}
}

func TestChunk1CovDedupeWonderkidAwayCopies(t *testing.T) {
	// Duplicate WK_-prefixed copies with NO copy at the home club: the first
	// WK copy is kept (covers the wkCopies[0] fallback).
	dm := chunk1CovInit(t)
	bar := dm.Clubs["LAL-BAR"]
	kept := bar.Squad[:0]
	for _, p := range bar.Squad {
		if !strings.EqualFold(p.FullName, "Venjamin Valerio") {
			kept = append(kept, p)
		}
	}
	bar.Squad = kept
	bar.SquadSize = len(kept)
	for _, cid := range []string{"EPL-LIV", "EPL-MCI"} {
		c := dm.Clubs[cid]
		c.Squad = append(c.Squad, &models.Player{
			PlayerID: "WK_Venj_" + cid, FullName: "Venjamin Valerio",
			Position: "ST", Category: "FWD", OVR: 78, Age: 20,
			UniverseWonderkid: true, ClubID: cid,
		})
		c.SquadSize = len(c.Squad)
	}
	dm.DedupePlayers()
	if n, _ := chunk1CovCountName(dm, "Venjamin Valerio"); n != 1 {
		t.Errorf("away WK copies should collapse to 1, got %d", n)
	}

	// Same shape with plain IDs and no flags: copies[0] fallback.
	dm2 := chunk1CovInit(t)
	nap := dm2.Clubs["SEA-NAP"]
	kept2 := nap.Squad[:0]
	for _, p := range nap.Squad {
		if !strings.EqualFold(p.FullName, "Ezail Zamora") {
			kept2 = append(kept2, p)
		}
	}
	nap.Squad = kept2
	nap.SquadSize = len(kept2)
	for _, cid := range []string{"EPL-LIV", "EPL-MCI"} {
		c := dm2.Clubs[cid]
		c.Squad = append(c.Squad, &models.Player{
			PlayerID: "EZ_" + cid, FullName: "Ezail Zamora",
			Position: "ST", Category: "FWD", OVR: 77, Age: 22, ClubID: cid,
		})
		c.SquadSize = len(c.Squad)
	}
	dm2.DedupePlayers()
	if n, _ := chunk1CovCountName(dm2, "Ezail Zamora"); n != 1 {
		t.Errorf("away plain copies should collapse to 1, got %d", n)
	}
}

func TestChunk1CovInitEmptyAndPartialHomes(t *testing.T) {
	// Empty homes map falls back to defaults (12 wonderkids).
	dm := chunk1CovFreshDM(t)
	dm.ProdigyHomes = map[string]string{}
	if err := dm.InitializeEliteProdigies(); err != nil {
		t.Fatalf("empty-homes init failed: %v", err)
	}
	if len(dm.Wonderkids) != len(EliteProdigyConfigs) {
		t.Errorf("empty-homes wonderkids = %d; want %d", len(dm.Wonderkids), len(EliteProdigyConfigs))
	}
	// One name missing from homes falls back to its canonical club.
	dm2 := chunk1CovFreshDM(t)
	dm2.ProdigyHomes = DefaultProdigyHomes()
	delete(dm2.ProdigyHomes, "Ezail Zamora")
	if err := dm2.InitializeEliteProdigies(); err != nil {
		t.Fatalf("partial-homes init failed: %v", err)
	}
	if _, at := chunk1CovCountName(dm2, "Ezail Zamora"); at != "SEA-NAP" {
		t.Errorf("missing home should fall back to SEA-NAP, got %q", at)
	}
}

func TestChunk1CovApplyHomesMissingClubAndFakeNames(t *testing.T) {
	// Deleted elite club: destination lookup misses and the prodigy is skipped.
	dm := chunk1CovInit(t)
	delete(dm.Clubs, "LAL-BAR")
	dm.ApplyProdigyHomes(DefaultProdigyHomes())
	if len(dm.Wonderkids) != len(EliteProdigyConfigs)-1 {
		t.Errorf("missing-club homes wonderkids = %d; want %d", len(dm.Wonderkids), len(EliteProdigyConfigs)-1)
	}

	// Twelve bogus names mapped to the twelve elite clubs: the map survives
	// validation (12 unique clubs), and the per-config fallback engages.
	dm2 := chunk1CovInit(t)
	elite := []string{}
	for _, cfg := range EliteProdigyConfigs {
		elite = append(elite, cfg.ClubID)
	}
	fake := map[string]string{}
	for i, cid := range elite {
		fake["Fake Prodigy "+string(rune('A'+i))] = cid
	}
	dm2.ApplyProdigyHomes(fake)
	if len(dm2.Wonderkids) != len(EliteProdigyConfigs) {
		t.Errorf("fake-name homes wonderkids = %d; want %d", len(dm2.Wonderkids), len(EliteProdigyConfigs))
	}
	for _, cfg := range EliteProdigyConfigs {
		if got := dm2.ProdigyHomes[cfg.FullName]; got != cfg.ClubID {
			t.Errorf("fallback home for %s = %q; want %q", cfg.FullName, got, cfg.ClubID)
		}
	}
}
