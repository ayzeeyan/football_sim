package tournament

import (
	"math/rand"
	"strings"
	"testing"

	"football_sim/pkg/growth"
	"football_sim/pkg/managers"
	"football_sim/pkg/matchreport"
	"football_sim/pkg/models"
	"football_sim/pkg/transfers"
)

// Chunk 3 coverage: post-match application (ledgers, market, prodigy XP,
// injuries) plus backfill for the season-sim surface.

// --------------------------------------------------------------------------
// Helpers
// --------------------------------------------------------------------------

func chunk3mkSquad(prefix string, n int) []*models.Player {
	cats := []string{"GK", "DEF", "DEF", "DEF", "DEF", "MID", "MID", "MID", "FWD", "FWD", "FWD"}
	poss := []string{"GK", "CB", "CB", "CB", "RB", "CM", "CM", "CAM", "LW", "ST", "RW"}
	squad := make([]*models.Player, 0, n)
	for i := 0; i < n; i++ {
		ci := i % 11
		squad = append(squad, &models.Player{
			PlayerID: prefix + string(rune('A'+i%26)) + string(rune('a'+i/26)),
			FullName: prefix + " Player", Position: poss[ci], Category: cats[ci],
			OVR: 72, Age: 25, MarketValueEUR: 10000000, WageEUR: 50000,
			ContractYears: 3, Loyalty: 65, ClubID: prefix,
		})
	}
	return squad
}

func chunk3mkClub(id string) *models.Club {
	return &models.Club{
		ClubID: id, ClubName: id + " FC", ShortName: id,
		HomeStadium: id + " Park", OverallTeamRating: 78, Morale: 70,
		StadiumCapacity: 40000, Squad: chunk3mkSquad(id, 14),
	}
}

// --------------------------------------------------------------------------
// ApplyPlayerMatchStats
// --------------------------------------------------------------------------

func TestChunk3ApplyStats(t *testing.T) {
	home := chunk3mkClub("H")
	away := chunk3mkClub("A")
	hs1, hs2 := home.Squad[8], home.Squad[5] // ST + CM
	as1 := away.Squad[1]                     // DEF

	scMini := matchreport.ToMiniPlayer(hs1)
	asMini := matchreport.ToMiniPlayer(hs2)
	ogMini := matchreport.ToMiniPlayer(as1)
	report := &matchreport.MatchReport{
		Events: []matchreport.MatchEventItem{
			{Minute: 10, Seq: 1, Type: "goal", Side: "home", Scorer: &scMini, Assister: &asMini},
			{Minute: 20, Seq: 2, Type: "goal", Side: "home", Scorer: &scMini, Disallowed: true},
			{Minute: 30, Seq: 3, Type: "own_goal", Side: "away", Beneficiary: "home", Scorer: &ogMini},
			{Minute: 40, Seq: 4, Type: "red", Side: "home", Player: &asMini},
			{Minute: 50, Seq: 5, Type: "goal", Side: "home", Scorer: &matchreport.MiniPlayer{PlayerID: "GHOST"}},
			{Minute: 55, Seq: 6, Type: "goal", Side: "home", Scorer: &scMini, Assister: &matchreport.MiniPlayer{PlayerID: "GHOST2"}},
		},
		HomeXI: []matchreport.MatchPlayerRow{
			chunk3mkRowLocal(hs1.PlayerID, true, 90, 8.0),
			chunk3mkRowLocal(hs2.PlayerID, true, 90, 7.0),
		},
		HomeBench: []matchreport.MatchPlayerRow{
			chunk3mkRowLocal(home.Squad[12].PlayerID, true, 20, 6.5),
			chunk3mkRowLocal(home.Squad[13].PlayerID, false, 0, 0),
			chunk3mkRowLocal("GHOST3", true, 30, 6.5),
		},
		AwayXI: []matchreport.MatchPlayerRow{
			chunk3mkRowLocal(as1.PlayerID, true, 90, 6.0),
		},
		AwayBench: nil,
	}
	// Decay fixtures: unused suspended/injured players.
	home.Squad[0].SuspendedMatches = 2
	home.Squad[1].InjuredMatches, home.Squad[1].Injury = 2, "knock"
	home.Squad[2].InjuredMatches, home.Squad[2].Injury = 1, "knock"
	away.Squad[0].SuspendedMatches = 1

	ApplyPlayerMatchStats(home, away, report)

	if hs1.Goals != 2 { // disallowed + ghost-assist goals excluded appropriately
		t.Errorf("striker goals = %d; want 2", hs1.Goals)
	}
	if hs2.Assists != 1 {
		t.Errorf("mid assists = %d; want 1", hs2.Assists)
	}
	if as1.OwnGoals != 1 {
		t.Errorf("OG not banked: %d", as1.OwnGoals)
	}
	if hs2.SuspendedMatches != 1 {
		t.Errorf("red ban not banked: %d", hs2.SuspendedMatches)
	}
	if hs1.Appearances != 1 || home.Squad[12].Appearances != 1 || home.Squad[13].Appearances != 0 {
		t.Errorf("appearances wrong: %d/%d/%d", hs1.Appearances, home.Squad[12].Appearances, home.Squad[13].Appearances)
	}
	if hs1.ConsecutiveStarts != 1 || home.Squad[12].ConsecutiveStarts != 0 {
		t.Errorf("fatigue tracking wrong: %d/%d", hs1.ConsecutiveStarts, home.Squad[12].ConsecutiveStarts)
	}
	if home.Squad[0].SuspendedMatches != 1 {
		t.Errorf("suspension should decay 2->1, got %d", home.Squad[0].SuspendedMatches)
	}
	if home.Squad[1].InjuredMatches != 1 {
		t.Errorf("injury should decay 2->1, got %d", home.Squad[1].InjuredMatches)
	}
	if home.Squad[2].InjuredMatches != 0 || home.Squad[2].Injury != "" {
		t.Errorf("served injury should clear: %+v", home.Squad[2])
	}
	if away.Squad[0].SuspendedMatches != 0 {
		t.Errorf("away suspension should decay 1->0, got %d", away.Squad[0].SuspendedMatches)
	}

	// Nil guards.
	ApplyPlayerMatchStats(nil, nil, report)
	ApplyPlayerMatchStats(home, away, nil)
}

// --------------------------------------------------------------------------
// ApplyMarketMovements
// --------------------------------------------------------------------------

func TestChunk3MarketMovements(t *testing.T) {
	home := chunk3mkClub("H")
	star := home.Squad[8]
	star.MarketValueEUR = 10000000
	quiet := home.Squad[5]
	quiet.MarketValueEUR = 8000000
	quiet.Age = 30
	poor := home.Squad[1]
	poor.MarketValueEUR = 5000000
	poor.OVR = 65 // corridor [3.85M, 33M] holds the clamped result
	cheap := home.Squad[2]
	cheap.MarketValueEUR = 400000
	cheap.Age = 45 // veteran anchor 500k keeps the 500k floor observable
	rating := func(v float64) *float64 { return &v }
	card := "red"
	tm := &TournamentManager{Clubs: map[string]*models.Club{"H": home}}
	report := &matchreport.MatchReport{
		HomeXI: []matchreport.MatchPlayerRow{
			{PlayerID: star.PlayerID, MatchGoals: 2, Rating: rating(8.5), Played: true, Minutes: 90},
			{PlayerID: quiet.PlayerID, Rating: rating(6.5), Played: true, Minutes: 90},
			{PlayerID: poor.PlayerID, Rating: rating(5.5), Card: &card, Played: true, Minutes: 90},
			{PlayerID: cheap.PlayerID, MatchGoals: 3, Rating: rating(9.0), Played: true, Minutes: 90},
			{PlayerID: "GHOST", MatchGoals: 2, Rating: rating(9.0), Played: true, Minutes: 90},
		},
		AwayXI: nil,
	}
	star.Age = 20
	tm.ApplyMarketMovements(home, nil, report, 7)
	// star: 1 + .12 + .05 + .02 = 1.19 -> clamp 1.15 -> 11.5M
	if star.MarketValueEUR != 11500000 {
		t.Errorf("star value = %d; want 11500000", star.MarketValueEUR)
	}
	if quiet.MarketValueEUR != 8000000 {
		t.Errorf("quiet veteran must be skipped (mult 1.0), got %d", quiet.MarketValueEUR)
	}
	// poor: 1 - .05 - .05 = 0.90 -> clamp 0.92 -> 4.6M (inside the corridor)
	if poor.MarketValueEUR != 4600000 {
		t.Errorf("poor value = %d; want 4600000", poor.MarketValueEUR)
	}
	if cheap.MarketValueEUR != 500000 {
		t.Errorf("cheap value = %d; want floored 500000", cheap.MarketValueEUR)
	}
	// Nil report / nil clubs are safe.
	tm.ApplyMarketMovements(home, nil, nil, 7)
	tm.ApplyMarketMovements(nil, nil, report, 7)

	// Transfer wire with an attached engine.
	te := &transfers.TransferEngine{}
	tm.TransferEngine = te
	riser := home.Squad[9]
	riser.MarketValueEUR = 20000000
	riser.ClubID = "ZZZ" // unknown home -> seller falls back to the club
	report2 := &matchreport.MatchReport{
		HomeXI: []matchreport.MatchPlayerRow{
			{PlayerID: riser.PlayerID, MatchGoals: 3, Rating: rating(9.0), Played: true, Minutes: 90},
		},
	}
	tm.ApplyMarketMovements(home, nil, report2, 9)
	if len(te.TransferFeed) != 1 {
		t.Fatalf("riser should hit the wire, feed=%d", len(te.TransferFeed))
	}
	item := te.TransferFeed[0]
	if item.Category != "RUMOR" || item.Matchweek != 9 || !strings.Contains(item.Headline, riser.FullName) {
		t.Errorf("feed item wrong: %+v", item)
	}
	// Known club home resolves the seller short name.
	riser.ClubID = "H"
	te.TransferFeed = nil
	tm.ApplyMarketMovements(home, nil, report2, 9)
	if len(te.TransferFeed) != 1 || !strings.HasSuffix(te.TransferFeed[0].Headline, "starring for H.") {
		t.Errorf("seller short name wrong: %+v", te.TransferFeed)
	}
	// Nil-rating rows default to 6.4.
	plain := home.Squad[6]
	plain.MarketValueEUR = 10000000
	report3 := &matchreport.MatchReport{
		HomeXI: []matchreport.MatchPlayerRow{{PlayerID: plain.PlayerID, MatchGoals: 1, Played: true, Minutes: 90}},
	}
	tm.TransferEngine = nil
	tm.ApplyMarketMovements(home, nil, report3, 9)
	if plain.MarketValueEUR != 10600000 { // 1 + .06
		t.Errorf("nil-rating riser = %d; want 10600000", plain.MarketValueEUR)
	}
}

// --------------------------------------------------------------------------
// AttributeProdigyPerformance
// --------------------------------------------------------------------------

func TestChunk3ProdigyPerformance(t *testing.T) {
	ge := growth.NewGrowthEngine(717)
	club := chunk3mkClub("WK")
	wk := club.Squad[8]
	wk.UniverseWonderkid = true
	wk.Age = 16
	wk.Personality = "dedicated_pro"
	wk.MentorName = "Vet"
	wk.MentorOVR = 86
	ge.RegisterProdigy(wk.PlayerID, wk.FullName, 16, 178, 70, "FWD", 78, 94, 19)
	bioBefore := ge.Biometrics[wk.PlayerID].AccumulatedXP

	sc := matchreport.ToMiniPlayer(wk)
	as := matchreport.ToMiniPlayer(club.Squad[5])
	report := &matchreport.MatchReport{
		Events: []matchreport.MatchEventItem{
			{Minute: 10, Type: "goal", Side: "home", Scorer: &sc, Assister: &as},
			{Minute: 70, Type: "penalty", Side: "home", Scorer: &sc},
			{Minute: 80, Type: "goal", Side: "away", Scorer: &as},
		},
		HomeXI: []matchreport.MatchPlayerRow{chunk3mkRowLocal(wk.PlayerID, true, 90, 7.5)},
	}
	events := AttributeProdigyPerformance(club, report, "home", ge)
	if ge.Biometrics[wk.PlayerID].AccumulatedXP <= bioBefore {
		t.Errorf("prodigy XP should increase")
	}
	if wk.OVR != ge.CalculateOVR(wk.PlayerID, wk.Category) {
		t.Errorf("prodigy OVR not synced: %d", wk.OVR)
	}
	if wk.Composure != ge.Attributes[wk.PlayerID].Composure {
		t.Errorf("prodigy composure not synced")
	}
	_ = events

	// Guards: nils, no prodigy, missing row, unplayed row, nil rating.
	if got := AttributeProdigyPerformance(nil, report, "home", ge); got != nil {
		t.Errorf("nil club should yield nil")
	}
	if got := AttributeProdigyPerformance(club, nil, "home", ge); got != nil {
		t.Errorf("nil report should yield nil")
	}
	if got := AttributeProdigyPerformance(club, report, "home", nil); got != nil {
		t.Errorf("nil engine should yield nil")
	}
	plain := chunk3mkClub("PL")
	if got := AttributeProdigyPerformance(plain, report, "home", ge); got != nil {
		t.Errorf("prodigy-less club should yield nil")
	}
	awayReport := &matchreport.MatchReport{}
	if got := AttributeProdigyPerformance(club, awayReport, "away", ge); got != nil {
		t.Errorf("missing row should yield nil")
	}
	unplayed := &matchreport.MatchReport{
		HomeXI: []matchreport.MatchPlayerRow{chunk3mkRowLocal(wk.PlayerID, false, 0, 0)},
	}
	if got := AttributeProdigyPerformance(club, unplayed, "home", ge); got != nil {
		t.Errorf("unplayed prodigy should yield nil")
	}
	noRating := &matchreport.MatchReport{
		HomeXI: []matchreport.MatchPlayerRow{{PlayerID: wk.PlayerID, Played: true, Minutes: 90}},
	}
	if got := AttributeProdigyPerformance(club, noRating, "home", ge); got != nil {
		t.Errorf("rating-less row should yield nil")
	}
	// Unregistered prodigy: XP engine misses, OVR falls to the default.
	stranger := chunk3mkClub("ST")
	stranger.Squad[8].UniverseWonderkid = true
	strangerReport := &matchreport.MatchReport{
		HomeXI: []matchreport.MatchPlayerRow{chunk3mkRowLocal(stranger.Squad[8].PlayerID, true, 90, 7.0)},
	}
	if got := AttributeProdigyPerformance(stranger, strangerReport, "home", ge); got != nil {
		t.Errorf("unregistered prodigy should yield nil events, got %v", got)
	}
	if stranger.Squad[8].OVR != 75 {
		t.Errorf("unregistered OVR = %d; want default 75", stranger.Squad[8].OVR)
	}
	// Bare prodigy without mentor/personality completes cleanly.
	bare := chunk3mkClub("BR")
	bare.Squad[8].UniverseWonderkid = true
	ge.RegisterProdigy(bare.Squad[8].PlayerID, bare.Squad[8].FullName, 16, 175, 68, "FWD", 74, 92, 19)
	bareReport := &matchreport.MatchReport{
		HomeXI: []matchreport.MatchPlayerRow{chunk3mkRowLocal(bare.Squad[8].PlayerID, true, 90, 6.8)},
	}
	AttributeProdigyPerformance(bare, bareReport, "home", ge)
}

func chunk3mkRowLocal(pid string, played bool, minutes int, rating float64) matchreport.MatchPlayerRow {
	row := matchreport.MatchPlayerRow{PlayerID: pid, Played: played, Minutes: minutes}
	if played && minutes > 0 {
		r := rating
		row.Rating = &r
	}
	return row
}

// --------------------------------------------------------------------------
// MaybeInjure
// --------------------------------------------------------------------------

func chunk3injuryTM(seed int64, managers map[string]*managers.ManagerProfile) *TournamentManager {
	return &TournamentManager{
		Managers: managers, RNG: rand.New(rand.NewSource(seed)), SeasonName: "2026-27",
		MilestonesFired: map[string]map[string]bool{},
	}
}

func chunk3injuryReport(home, away *models.Club) *matchreport.MatchReport {
	mkRows := func(club *models.Club) []matchreport.MatchPlayerRow {
		rows := make([]matchreport.MatchPlayerRow, 0, 11)
		for _, p := range club.Squad[:11] {
			rows = append(rows, chunk3mkRowLocal(p.PlayerID, true, 90, 6.8))
		}
		return rows
	}
	return &matchreport.MatchReport{HomeXI: mkRows(home), AwayXI: mkRows(away)}
}

func TestChunk3Injuries(t *testing.T) {
	var sawSerious, sawMinor, sawSingular, sawPlural bool
	for seed := int64(0); seed < 60 && !(sawSerious && sawMinor && sawSingular && sawPlural); seed++ {
		home, away := chunk3mkClub("H"), chunk3mkClub("A")
		tm := chunk3injuryTM(seed, nil)
		hBefore, aBefore := countInjured(home), countInjured(away)
		tm.MaybeInjure(home, away, chunk3injuryReport(home, away), 5, "F1")
		if dh := countInjured(home) - hBefore; dh > 1 {
			t.Fatalf("seed %d: %d home casualties, max 1", seed, dh)
		}
		if da := countInjured(away) - aBefore; da > 1 {
			t.Fatalf("seed %d: %d away casualties, max 1", seed, da)
		}
		for _, item := range tm.Inbox {
			if item.Category != "injury" {
				continue
			}
			if strings.HasPrefix(item.Headline, "CRUSHING BLOW") {
				sawSerious = true
				if item.FixtureID != "F1" || item.Matchweek != 5 {
					t.Errorf("serious inbox wiring wrong: %+v", item)
				}
			} else {
				sawMinor = true
				if strings.Contains(item.Headline, " out 1 match") {
					sawSingular = true
				}
				if strings.Contains(item.Headline, " out 2 matches") || strings.Contains(item.Headline, " out 3 matches") {
					sawPlural = true
				}
			}
		}
		// Bounds on every casualty.
		for _, club := range []*models.Club{home, away} {
			for _, p := range club.Squad {
				if p.InjuredMatches < 0 || p.InjuredMatches > 25 {
					t.Errorf("injury span out of bounds: %+v", p)
				}
				if p.InjuredMatches > 3 && !isSeriousKind(p.Injury) {
					t.Errorf("long layoff without serious kind: %+v", p)
				}
			}
		}
	}
	for name, seen := range map[string]bool{"serious": sawSerious, "minor": sawMinor, "singular": sawSingular, "plural": sawPlural} {
		if !seen {
			t.Errorf("injury family %q never observed in 60 fixtures", name)
		}
	}

	// Full treatment room: no new casualties.
	home, away := chunk3mkClub("H"), chunk3mkClub("A")
	for i := 0; i < 3; i++ {
		home.Squad[i].InjuredMatches, home.Squad[i].Injury = 2, "knock"
	}
	tm := chunk3injuryTM(7, nil)
	before := len(tm.Inbox)
	tm.MaybeInjure(home, away, chunk3injuryReport(home, away), 5, "F1")
	if len(tm.Inbox) != before {
		t.Errorf("full treatment room should block new injuries")
	}

	// Lone keeper is protected and short shifts are safe (both structural).
	for seed := int64(0); seed < 30; seed++ {
		h := chunk3mkClub("H2")
		solo := []*models.Player{}
		seenGK := false
		for _, p := range h.Squad {
			if p.Category == "GK" {
				if seenGK {
					continue
				}
				seenGK = true
			}
			solo = append(solo, p)
		}
		h.Squad = solo
		a := chunk3mkClub("A2")
		tm2 := chunk3injuryTM(100+seed, nil)
		rep := chunk3injuryReport(h, a)
		shortID := h.Squad[len(h.Squad)-1].PlayerID
		rep.HomeBench = []matchreport.MatchPlayerRow{chunk3mkRowLocal(shortID, true, 20, 6.5)}
		tm2.MaybeInjure(h, a, rep, 5, "F1")
		if h.Squad[0].InjuredMatches != 0 {
			t.Errorf("seed %d: lone keeper must be protected", seed)
		}
		for _, p := range h.Squad {
			if p.PlayerID == shortID && p.InjuredMatches != 0 {
				t.Errorf("seed %d: 20-minute shift must be safe", seed)
			}
		}
	}

	// Suspended and injured players are never re-injured.
	h3, a3 := chunk3mkClub("H3"), chunk3mkClub("A3")
	h3.Squad[5].SuspendedMatches = 2
	h3.Squad[6].InjuredMatches = 2
	h3.Squad[6].Injury = "knock"
	for seed := int64(0); seed < 20; seed++ {
		tm3 := chunk3injuryTM(300+seed, nil)
		tm3.MaybeInjure(h3, a3, chunk3injuryReport(h3, a3), 5, "F1")
		if h3.Squad[5].InjuredMatches != 0 || h3.Squad[6].InjuredMatches != 2 {
			t.Errorf("unavailable players must be skipped")
		}
	}

	// High-press dugouts (direct + canonical styles) run the hotter table.
	for _, style := range []string{"high_press", "press"} {
		hit := false
		for seed := int64(0); seed < 30 && !hit; seed++ {
			h, a := chunk3mkClub("HP"), chunk3mkClub("HQ")
			tm4 := chunk3injuryTM(500+seed, map[string]*managers.ManagerProfile{
				"HP": {ClubID: "HP", Style: style},
			})
			tm4.MaybeInjure(h, a, chunk3injuryReport(h, a), 5, "F1")
			hit = countInjured(h)+countInjured(a) > 0
		}
		if !hit {
			t.Errorf("high-press style %q produced no injuries in 30 fixtures", style)
		}
	}

	// Nil guards.
	tm5 := chunk3injuryTM(9, nil)
	h5, a5 := chunk3mkClub("H5"), chunk3mkClub("A5")
	tm5.MaybeInjure(h5, a5, nil, 5, "F1")
	tm5.MaybeInjure(nil, a5, chunk3injuryReport(h5, a5), 5, "F1")
	(&TournamentManager{}).MaybeInjure(h5, a5, chunk3injuryReport(h5, a5), 5, "F1")
}

func countInjured(club *models.Club) int {
	n := 0
	for _, p := range club.Squad {
		if p.InjuredMatches > 0 {
			n++
		}
	}
	return n
}

func isSeriousKind(kind string) bool {
	return kind == "ACL tear" || kind == "meniscus tear" || kind == "ruptured cruciate ligament"
}

// --------------------------------------------------------------------------
// ApplyMatchReport orchestration
// --------------------------------------------------------------------------

func TestChunk3ApplyMatchReport(t *testing.T) {
	ge := growth.NewGrowthEngine(727)
	home, away := chunk3mkClub("H"), chunk3mkClub("A")
	wk := home.Squad[8]
	wk.UniverseWonderkid = true
	wk.Age = 16
	ge.RegisterProdigy(wk.PlayerID, wk.FullName, 16, 178, 70, "FWD", 78, 94, 19)
	sc := matchreport.ToMiniPlayer(wk)
	tm := &TournamentManager{
		Clubs:        map[string]*models.Club{"H": home, "A": away},
		GrowthEngine: ge, RNG: rand.New(rand.NewSource(3)),
		SeasonName: "2026-27", MilestonesFired: map[string]map[string]bool{},
		TransferEngine: &transfers.TransferEngine{},
	}
	report := &matchreport.MatchReport{
		Events: []matchreport.MatchEventItem{
			{Minute: 10, Type: "goal", Side: "home", Scorer: &sc},
		},
		HomeXI: []matchreport.MatchPlayerRow{chunk3mkRowLocal(wk.PlayerID, true, 90, 7.5)},
		AwayXI: []matchreport.MatchPlayerRow{chunk3mkRowLocal(away.Squad[8].PlayerID, true, 90, 6.5)},
	}
	growthEvents := tm.ApplyMatchReport(home, away, report, 4, "FX1")
	if wk.Goals != 1 || wk.Appearances != 1 {
		t.Errorf("ledgers not applied: %+v", wk)
	}
	if len(growthEvents) > 8 {
		t.Errorf("growth wire should stay compact: %d", len(growthEvents))
	}
	// Prodigy-less orchestration stays quiet.
	plain := chunk3mkClub("P")
	plainReport := &matchreport.MatchReport{
		HomeXI: []matchreport.MatchPlayerRow{chunk3mkRowLocal(plain.Squad[8].PlayerID, true, 90, 7.0)},
	}
	if got := tm.ApplyMatchReport(plain, away, plainReport, 4, "FX2"); len(got) != 0 {
		t.Errorf("prodigy-less report should yield no growth events: %v", got)
	}
}
