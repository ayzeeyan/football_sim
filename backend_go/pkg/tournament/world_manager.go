package tournament

import (
	"fmt"
	"sort"
	"strings"

	"football_sim/pkg/growth"
	"football_sim/pkg/models"
)

// swissPair is an undirected league-phase pairing with its round index.
type swissPair struct {
	a, b  string
	round int
}

// swissDirected is an oriented pairing (home hosts away).
type swissDirected struct {
	home, away string
	round      int
}

// NewEuropeanWorldManager creates the playable Top Five universe. The legacy
// NewTournamentManager is intentionally retained for version-3 saves; new
// careers call this constructor and receive five domestic leagues, national cups
// including the EFL Cup, and three shared European league-phase competitions.
func NewEuropeanWorldManager(clubs []*models.Club, ge *growth.GrowthEngine, seed int64) *TournamentManager {
	tm := NewTournamentManager(clubs, ge, seed)
	if seed == 0 {
		seed = 20260907
	}
	tm.World = &EuropeanWorld{
		Version:      1,
		Seed:         seed,
		Competitions: map[string]*Competition{},
		Fixtures:     []Fixture{},
	}
	tm.MaxMatchweeks = 38
	tm.UCLFixtures = nil
	tm.SuperCupFixtures = nil
	tm.UCLGroupA, tm.UCLGroupB = nil, nil
	tm.UCLRecords = map[string]*models.CompetitionRecord{}
	tm.UCLQuarterFinals, tm.UCLSemiFinals = map[string]CupTie{}, map[string]CupTie{}
	tm.UCLFinal = CupTie{}
	tm.UCLChampionID = ""
	tm.SuperCupPlayIn, tm.SuperCupQuarterFinals, tm.SuperCupSemiFinals = map[string]CupTie{}, map[string]CupTie{}, map[string]CupTie{}
	tm.SuperCupFinal = CupTie{}
	tm.SuperCupChampionID = ""
	tm.SuperCupByes = nil
	tm.SuperCupStage = ""

	leagueClubs := make(map[string][]*models.Club)
	for _, club := range clubs {
		if club != nil && isTopFiveLeague(club.League) {
			leagueClubs[club.League] = append(leagueClubs[club.League], club)
		}
	}
	tm.Fixtures = nil
	for _, def := range domesticLeagueDefinitions {
		members := leagueClubs[def.League]
		if len(members) < 2 {
			continue
		}
		ids := sortedClubIDs(members)
		competition := &Competition{ID: def.ID, Name: def.Name, Country: def.Country, Kind: def.Kind, Prestige: def.Prestige, ParticipantIDs: ids, Stage: "League"}
		tm.World.Competitions[def.ID] = competition
		tm.World.CompetitionOrder = append(tm.World.CompetitionOrder, def.ID)
		fixtures, rounds := GenerateDoubleRoundRobinFixtures(members, def.ID)
		if rounds > tm.MaxMatchweeks {
			tm.MaxMatchweeks = rounds
		}
		tm.Fixtures = append(tm.Fixtures, fixtures...)
	}
	for _, def := range domesticCupDefinitions {
		members := leagueClubs[def.League]
		if len(members) < 2 {
			continue
		}
		competition := &Competition{ID: def.ID, Name: def.Name, Country: def.Country, Kind: def.Kind, Prestige: def.Prestige, ParticipantIDs: sortedClubIDs(members), Stage: "Draw"}
		tm.World.Competitions[def.ID] = competition
		tm.World.CompetitionOrder = append(tm.World.CompetitionOrder, def.ID)
		tm.scheduleWorldCupOpeningRoundUnlocked(competition)
	}
	for _, def := range europeanDefinitions {
		competition := &Competition{ID: def.ID, Name: def.Name, Country: def.Country, Kind: def.Kind, Prestige: def.Prestige, Records: map[string]*models.CompetitionRecord{}, QualificationSources: map[string]string{}, Stage: "League Phase"}
		tm.World.Competitions[def.ID] = competition
		tm.World.CompetitionOrder = append(tm.World.CompetitionOrder, def.ID)
		tm.seedOpeningEuropeanParticipantsUnlocked(competition, leagueClubs)
		tm.scheduleEuropeanLeaguePhaseUnlocked(competition)
	}
	tm.Inbox = nil
	tm.InboxSeq = 0
	tm.AssignSquadRolesUnlocked()
	tm.AssignBoardExpectationsUnlocked()
	tm.ArrangeLoansUnlocked()
	tm.PushInbox("system", tm.SeasonName+" European world opens", "Five domestic leagues, national cups, and three European competitions now share one calendar.", 1, nil, "", "")
	return tm
}

func (tm *TournamentManager) worldCompetitionUnlocked(id string) *Competition {
	if tm == nil || tm.World == nil || tm.World.Competitions == nil {
		return nil
	}
	return tm.World.Competitions[id]
}

func (tm *TournamentManager) isWorldDomesticLeague(compID string) bool {
	comp := tm.worldCompetitionUnlocked(compID)
	return comp != nil && comp.Kind == CompetitionLeague
}

func (tm *TournamentManager) isWorldKnockoutFixture(f *Fixture) bool {
	if f == nil {
		return false
	}
	comp := tm.worldCompetitionUnlocked(f.Competition)
	if comp == nil {
		return false
	}
	return comp.Kind == CompetitionDomestic || (comp.Kind == CompetitionEuropean && f.Stage != "League Phase")
}

func (tm *TournamentManager) seedOpeningEuropeanParticipantsUnlocked(comp *Competition, byLeague map[string][]*models.Club) {
	if comp == nil {
		return
	}
	for _, def := range domesticLeagueDefinitions {
		ranked := rankClubsForOpening(byLeague[def.League])
		start, end := europeanOpeningRankRange(comp.ID, len(ranked))
		for idx := start - 1; idx < end && idx < len(ranked); idx++ {
			club := ranked[idx]
			comp.ParticipantIDs = append(comp.ParticipantIDs, club.ClubID)
			comp.QualificationSources[club.ClubID] = fmt.Sprintf("Opening coefficient allocation · %s", def.Name)
		}
	}
}

func (tm *TournamentManager) addWorldFixtureUnlocked(f Fixture) {
	if tm.World == nil {
		return
	}
	tm.World.Fixtures = append(tm.World.Fixtures, f)
}

func (tm *TournamentManager) worldFixtureUnlocked(id string) *Fixture {
	if tm.World == nil {
		return nil
	}
	for i := range tm.World.Fixtures {
		if tm.World.Fixtures[i].FixtureID == id {
			return &tm.World.Fixtures[i]
		}
	}
	return nil
}

func (tm *TournamentManager) newWorldFixtureUnlocked(id string, mw int, compID, stage, homeID, awayID string) Fixture {
	home, away := tm.Clubs[homeID], tm.Clubs[awayID]
	f := tm.blankFixture(id, mw, compID, stage, home, away, 1, id)
	f.TieID = id
	return f
}

func (tm *TournamentManager) newWorldLegFixtureUnlocked(id string, mw int, compID, stage, homeID, awayID, tieID string, leg int) Fixture {
	home, away := tm.Clubs[homeID], tm.Clubs[awayID]
	f := tm.blankFixture(id, mw, compID, stage, home, away, leg, tieID)
	f.TieID = tieID
	f.Leg = leg
	return f
}

// europeanKnockoutLegWeeks maps a knockout stage to its home-and-away weeks.
// Finals are single-leg; every other European knockout round is two-legged.
func europeanKnockoutLegWeeks(stage string) (int, int, bool) {
	switch stage {
	case "Knockout play-off", "Play-off":
		return 28, 29, true
	case "Round of 16":
		return 30, 31, true
	case "Quarter-final":
		return 32, 33, true
	case "Semi-final":
		return 35, 36, true
	case "Final":
		return 38, 38, false
	default:
		return 38, 38, false
	}
}

func isEuropeanTwoLeggedStage(comp *Competition, stage string, entrants int) bool {
	if comp == nil || comp.Kind != CompetitionEuropean {
		return false
	}
	if entrants <= 2 {
		return false
	}
	_, _, two := europeanKnockoutLegWeeks(stage)
	return two
}

func cupRoundWeek(roundIndex int) int {
	return cupRoundWeekFor("", roundIndex)
}

func cupRoundWeekFor(compID string, roundIndex int) int {
	weeks := []int{2, 10, 18, 27, 35}
	if compID == "efl-cup" {
		weeks = []int{5, 13, 20, 28, 36}
	}
	if roundIndex < 0 {
		return weeks[0]
	}
	if roundIndex >= len(weeks) {
		return weeks[len(weeks)-1]
	}
	return weeks[roundIndex]
}

func europeanPhaseWeeks() []int { return []int{3, 6, 9, 12, 15, 18, 23, 26} }

func championsLeagueOpeningSlots(leagueSize int) int {
	if leagueSize >= 20 {
		return 8
	}
	if leagueSize >= 18 {
		return 6
	}
	return 4
}

func europeanOpeningRankRange(compID string, leagueSize int) (int, int) {
	ucl := championsLeagueOpeningSlots(leagueSize)
	switch compID {
	case "champions-league":
		return 1, ucl
	case "europa-league":
		return ucl + 1, ucl + 4
	case "conference-league":
		return ucl + 5, ucl + 8
	default:
		return 1, 4
	}
}

func europeanKnockoutWeek(roundIndex int) int {
	weeks := []int{28, 30, 32, 35, 38}
	if roundIndex < 0 {
		return weeks[0]
	}
	if roundIndex >= len(weeks) {
		return weeks[len(weeks)-1]
	}
	return weeks[roundIndex]
}

func (tm *TournamentManager) scheduleWorldCupOpeningRoundUnlocked(comp *Competition) {
	if comp == nil || len(comp.ParticipantIDs) < 2 {
		return
	}
	ids := shuffledIDs(tm.World.Seed, comp.ID+":opening", comp.ParticipantIDs)
	target := 1
	for target*2 <= len(ids) {
		target *= 2
	}
	playInTeams := 2 * (len(ids) - target)
	if playInTeams < 0 {
		playInTeams = 0
	}
	entrants := append([]string(nil), ids[:playInTeams]...)
	byes := append([]string(nil), ids[playInTeams:]...)
	stage := stageForKnockoutSize(len(ids))
	if playInTeams > 0 {
		stage = "Play-off"
	}
	round := KnockoutRound{Stage: stage, EntrantIDs: entrants, ByeIDs: byes, FixtureIDs: []string{}, TieIDs: []string{}}
	for i := 0; i+1 < len(entrants); i += 2 {
		homeID, awayID := entrants[i], entrants[i+1]
		if (i/2)%2 == 1 {
			homeID, awayID = awayID, homeID
		}
		id := fmt.Sprintf("%s-R%d-%s-%s", comp.ID, 1, homeID, awayID)
		round.TieIDs = append(round.TieIDs, id)
		tm.addWorldFixtureUnlocked(tm.newWorldFixtureUnlocked(id, cupRoundWeekFor(comp.ID, 0), comp.ID, stage, homeID, awayID))
		round.FixtureIDs = append(round.FixtureIDs, id)
	}
	comp.Rounds = append(comp.Rounds, round)
	comp.Stage = stage
}

func (tm *TournamentManager) scheduleEuropeanLeaguePhaseUnlocked(comp *Competition) {
	if comp == nil || len(comp.ParticipantIDs) < 2 {
		return
	}
	// The 36-team Champions League uses a seeded Swiss/coefficient draw:
	// 4 pots of 9, exactly 2 opponents per pot, 8 games with even home/away.
	// Smaller 20-team phases keep the deterministic circle rotation.
	if comp.ID == "champions-league" && len(comp.ParticipantIDs) == 36 {
		tm.scheduleChampionsLeagueSwissUnlocked(comp)
		return
	}
	clubs := clubsForIDs(tm.Clubs, comp.ParticipantIDs)
	tm.scheduleEuropeanCircleUnlocked(comp, clubs)
}

// scheduleEuropeanCircleUnlocked is the deterministic circle rotation used by
// the 20-team phases (and as a never-expected fallback for the Swiss draw).
// Ordering is seeded by the universe seed; pairings never consume match RNG.
func (tm *TournamentManager) scheduleEuropeanCircleUnlocked(comp *Competition, clubs []*models.Club) {
	if comp == nil || len(clubs) < 2 {
		return
	}
	if len(clubs)%2 != 0 {
		return // Top Five configuration intentionally seeds 20 evenly.
	}
	// Deterministic circle rotation seeded by universe seed ordering.
	ordered := append([]*models.Club(nil), clubs...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].ClubID < ordered[j].ClubID })
	seedOrder := shuffledIDs(tm.World.Seed, comp.ID+":circle-order", sortedClubIDs(ordered))
	byID := map[string]*models.Club{}
	for _, c := range ordered {
		byID[c.ClubID] = c
	}
	rotation := make([]*models.Club, 0, len(seedOrder))
	for _, id := range seedOrder {
		if c := byID[id]; c != nil {
			rotation = append(rotation, c)
		}
	}
	for round, mw := range europeanPhaseWeeks() {
		for i := 0; i < len(rotation)/2; i++ {
			a, b := rotation[i], rotation[len(rotation)-1-i]
			home, away := a, b
			if (round+i)%2 != 0 {
				home, away = away, home
			}
			id := fmt.Sprintf("%s-LP%d-%s-%s", comp.ID, round+1, home.ClubID, away.ClubID)
			tm.addWorldFixtureUnlocked(tm.newWorldFixtureUnlocked(id, mw, comp.ID, "League Phase", home.ClubID, away.ClubID))
			comp.LeaguePhaseFixtureIDs = append(comp.LeaguePhaseFixtureIDs, id)
		}
		last := rotation[len(rotation)-1]
		copy(rotation[2:], rotation[1:len(rotation)-1])
		rotation[1] = last
	}
	for _, id := range comp.ParticipantIDs {
		comp.Records[id] = &models.CompetitionRecord{Form: []string{}}
	}
}

// scheduleChampionsLeagueSwissUnlocked builds a true Swiss/coefficient league
// phase for 36 teams: 4 pots of 9 by opening strength, exactly 2 opponents
// from each pot, 8 games per club with 4 home / 4 away, across the 8 phase
// weeks. Pairing uses exact pot-pair blocks (no naive circle): intra-pot
// 9-cycles plus double shifted matchings across pot pairs, with same-league
// clashes minimized over seeded variants. Weeks are perfect matchings, so no
// club ever doubles up in a week. All iteration is over sorted slices.
func (tm *TournamentManager) scheduleChampionsLeagueSwissUnlocked(comp *Competition) {
	clubs := clubsForIDs(tm.Clubs, comp.ParticipantIDs)
	sort.SliceStable(clubs, func(i, j int) bool { return clubs[i].ClubID < clubs[j].ClubID })
	ranked := rankClubsForSwiss(clubs)
	pots := make([][]*models.Club, 4)
	for i, c := range ranked {
		p := i / 9
		if p > 3 {
			p = 3
		}
		pots[p] = append(pots[p], c)
	}
	// Snapshot pots for post-hoc verification and hub display.
	comp.Pots = make([][]string, 4)
	for p := range pots {
		ids := make([]string, 0, len(pots[p]))
		for _, c := range pots[p] {
			ids = append(ids, c.ClubID)
		}
		sort.Strings(ids)
		comp.Pots[p] = ids
	}
	leagueOf := map[string]string{}
	for _, c := range clubs {
		leagueOf[c.ClubID] = c.League
	}
	weeks := europeanPhaseWeeks()
	// Exact 2-per-pot draw via pot-pair blocks, then weekly matchings.
	// Every club meets exactly 2 clubs from each pot (8 games); weeks are
	// perfect matchings so no club ever doubles up in a week.
	pairs := swissBlockPairs(tm.World.Seed, comp.ID, pots, leagueOf)
	weekPairs, ok := swissSplitWeeks(tm.World.Seed, comp.ID, pairs)
	if !ok {
		// Not expected for the block-built draw; fall back to a seeded
		// circle so scheduling stays total (tests assert ok, so this path
		// is loud in CI rather than silent in production).
		tm.scheduleEuropeanCircleUnlocked(comp, clubs)
		return
	}
	var edges []swissPair
	for w, wk := range weekPairs {
		for _, pr := range wk {
			edges = append(edges, swissPair{a: pr[0], b: pr[1], round: w})
		}
	}
	// Balanced orientation via Eulerian trails: every club has degree 8
	// (even), so orienting each closed trail consistently yields exactly
	// 4 home / 4 away per club. Deterministic via sorted adjacency.
	homeCount := map[string]int{}
	awayCount := map[string]int{}
	directedEdges := orientSwissBalanced(edges)
	for _, d := range directedEdges {
		homeCount[d.home]++
		awayCount[d.away]++
	}
	// Iterative repair: flip the deterministically smallest edge whose flip
	// strictly reduces total |homes-4| imbalance. Eulerian 8-regular graphs
	// always admit a 4H/4A orientation; this converges to it.
	for iter := 0; iter < 500; iter++ {
		bestIdx := -1
		bestKey := ""
		for i, d := range directedEdges {
			if homeCount[d.home] <= 4 && homeCount[d.away] >= 4 {
				continue
			}
			// Flipping helps iff home is over-homed and away is under-homed.
			if homeCount[d.home] > 4 && homeCount[d.away] < 4 {
				key := d.home + "-" + d.away + "-" + d.away + "-" + d.home
				// Canonical key for deterministic choice.
				a, b := d.home, d.away
				if a > b {
					a, b = b, a
				}
				key = a + "-" + b
				if bestIdx == -1 || key < bestKey {
					bestIdx = i
					bestKey = key
				}
			}
		}
		if bestIdx == -1 {
			break
		}
		d := directedEdges[bestIdx]
		directedEdges[bestIdx] = swissDirected{home: d.away, away: d.home, round: d.round}
		homeCount[d.home]--
		awayCount[d.away]--
		homeCount[d.away]++
		awayCount[d.home]++
	}
	// Week assignment is inherent from the per-round perfect matchings above:
	// each round already has every club exactly once, so reuse it directly.
	sort.SliceStable(directedEdges, func(i, j int) bool {
		if directedEdges[i].round != directedEdges[j].round {
			return directedEdges[i].round < directedEdges[j].round
		}
		ai := directedEdges[i].home + "-" + directedEdges[i].away
		aj := directedEdges[j].home + "-" + directedEdges[j].away
		return ai < aj
	})
	for _, d := range directedEdges {
		mw := weeks[d.round]
		lp := d.round + 1
		id := fmt.Sprintf("%s-LP%d-%s-%s", comp.ID, lp, d.home, d.away)
		base := id
		suffix := 1
		for tm.worldFixtureUnlocked(id) != nil {
			suffix++
			id = fmt.Sprintf("%s-%d", base, suffix)
		}
		tm.addWorldFixtureUnlocked(tm.newWorldFixtureUnlocked(id, mw, comp.ID, "League Phase", d.home, d.away))
		comp.LeaguePhaseFixtureIDs = append(comp.LeaguePhaseFixtureIDs, id)
	}
	// Sort fixture IDs for stable persistence.
	sort.Strings(comp.LeaguePhaseFixtureIDs)
	for _, id := range comp.ParticipantIDs {
		comp.Records[id] = &models.CompetitionRecord{Form: []string{}}
	}
}

// swissBlockPairs builds the exact 144-pair Champions League league-phase
// draw: every club meets exactly 2 clubs from each of the 4 pots (8 games).
// Pot-pair block edge counts are forced by the quotas: 9 edges inside each
// pot (a seeded 9-cycle, degree 2) and 18 edges across each pot pair (a
// seeded double shifted matching, degree 2 per side). Distinct blocks span
// disjoint pot pairs, so no rematch can occur across blocks. Same-league
// pairings are minimized by trying seeded variants per block. Everything is
// derived from the universe seed; iteration is over sorted slices only.
func swissBlockPairs(seed int64, compID string, pots [][]*models.Club, leagueOf map[string]string) [][2]string {
	canon := func(a, b string) string {
		if a > b {
			a, b = b, a
		}
		return a + "\x00" + b
	}
	idsOf := func(cs []*models.Club) []string {
		out := make([]string, 0, len(cs))
		for _, c := range cs {
			if c != nil && c.ClubID != "" {
				out = append(out, c.ClubID)
			}
		}
		sort.Strings(out)
		return out
	}
	sameLeague := func(a, b string) bool {
		la, lb := leagueOf[a], leagueOf[b]
		return la != "" && la == lb
	}
	var pairs [][2]string
	// Intra-pot 9-cycles.
	for p := 0; p < 4; p++ {
		members := idsOf(pots[p])
		if len(members) != 9 {
			continue
		}
		var best []string
		bestBad := -1
		bestJoined := ""
		for v := 0; v < 3; v++ {
			order := shuffledIDs(seed, fmt.Sprintf("%s:swiss-pot%d:cycle-%d", compID, p, v), members)
			bad := 0
			for i := range order {
				if sameLeague(order[i], order[(i+1)%len(order)]) {
					bad++
				}
			}
			joined := strings.Join(order, "|")
			if bestBad < 0 || bad < bestBad || (bad == bestBad && joined < bestJoined) {
				bestBad, bestJoined, best = bad, joined, order
			}
		}
		for i := range best {
			pairs = append(pairs, [2]string{best[i], best[(i+1)%len(best)]})
		}
	}
	// Inter-pot double shifted matchings.
	for p := 0; p < 4; p++ {
		for q := p + 1; q < 4; q++ {
			aIDs, bIDs := idsOf(pots[p]), idsOf(pots[q])
			if len(aIDs) != 9 || len(bIDs) != 9 {
				continue
			}
			var best [][2]string
			bestBad := -1
			bestKey := ""
			for v := 0; v < 8; v++ {
				A := shuffledIDs(seed, fmt.Sprintf("%s:swiss-%d-%d:A-%d", compID, p, q, v), aIDs)
				B := shuffledIDs(seed, fmt.Sprintf("%s:swiss-%d-%d:B-%d", compID, p, q, v), bIDs)
				shift := v + 1
				cur := make([][2]string, 0, 18)
				keys := make([]string, 0, 18)
				bad := 0
				for i := range A {
					for _, j := range []int{i, (i + shift) % len(B)} {
						cur = append(cur, [2]string{A[i], B[j]})
						keys = append(keys, canon(A[i], B[j]))
						if sameLeague(A[i], B[j]) {
							bad++
						}
					}
				}
				sort.Strings(keys)
				ck := strings.Join(keys, "|")
				if bestBad < 0 || bad < bestBad || (bad == bestBad && ck < bestKey) {
					bestBad, bestKey, best = bad, ck, cur
				}
			}
			pairs = append(pairs, best...)
		}
	}
	return pairs
}

// swissSplitWeeks assigns unique pairs to 8 weekly perfect matchings (18
// pairs each, every club exactly once per week) by sequential MRV extraction
// with deterministic seeded retries and single-week rewinds. Returns ok=false
// if no decomposition is found; the block-built draw is dense and symmetric,
// so failure is not expected (tests pin success across seeds).
func swissSplitWeeks(seed int64, compID string, pairs [][2]string) ([][][2]string, bool) {
	const weeks, perWeek = 8, 18
	verts := make([]string, 0, 36)
	seen := map[string]bool{}
	for _, pr := range pairs {
		for _, v := range pr {
			if !seen[v] {
				seen[v] = true
				verts = append(verts, v)
			}
		}
	}
	sort.Strings(verts)
	if len(verts) != 36 || len(pairs) != weeks*perWeek {
		return nil, false
	}
	alive := make([]bool, len(pairs))
	for i := range alive {
		alive[i] = true
	}
	other := func(ei int, v string) string {
		if pairs[ei][0] == v {
			return pairs[ei][1]
		}
		return pairs[ei][0]
	}
	const stepCap = 200000
	// extract finds one perfect matching over alive edges.
	extract := func(week, attempt, rewinds int) ([]int, bool) {
		prefOrder := shuffledIDs(seed, fmt.Sprintf("%s:swiss-week-%d:%d:%d", compID, week, attempt, rewinds), verts)
		rank := make(map[string]int, len(verts))
		for i, v := range prefOrder {
			rank[v] = i
		}
		adj := make(map[string][]int, len(verts))
		for i := range pairs {
			if !alive[i] {
				continue
			}
			adj[pairs[i][0]] = append(adj[pairs[i][0]], i)
			adj[pairs[i][1]] = append(adj[pairs[i][1]], i)
		}
		matched := make(map[string]bool, len(verts))
		var chosen []int
		steps := 0
		var dfs func() bool
		dfs = func() bool {
			steps++
			if steps > stepCap {
				return false
			}
			// MRV: unmatched club with fewest available partners.
			u := ""
			var uCands []int
			haveU := false
			for _, v := range verts {
				if matched[v] {
					continue
				}
				var cands []int
				for _, ei := range adj[v] {
					if w := other(ei, v); !matched[w] {
						cands = append(cands, ei)
					}
				}
				sort.SliceStable(cands, func(i, j int) bool {
					ni, nj := other(cands[i], v), other(cands[j], v)
					if rank[ni] != rank[nj] {
						return rank[ni] < rank[nj]
					}
					if ni != nj {
						return ni < nj
					}
					return cands[i] < cands[j]
				})
				if !haveU || len(cands) < len(uCands) ||
					(len(cands) == len(uCands) && (rank[v] < rank[u] || (rank[v] == rank[u] && v < u))) {
					u, uCands, haveU = v, cands, true
				}
			}
			if !haveU {
				return true
			}
			if len(uCands) == 0 {
				return false
			}
			for _, ei := range uCands {
				w := other(ei, u)
				matched[u] = true
				matched[w] = true
				chosen = append(chosen, ei)
				if dfs() {
					return true
				}
				matched[u] = false
				matched[w] = false
				chosen = chosen[:len(chosen)-1]
			}
			return false
		}
		if !dfs() {
			return nil, false
		}
		return append([]int(nil), chosen...), true
	}
	committed := make([][]int, 0, weeks)
	attempt := make([]int, weeks)
	rewinds := 0
	w := 0
	for w < weeks {
		got, ok := extract(w, attempt[w], rewinds)
		if ok && len(got) == perWeek {
			committed = append(committed, got)
			for _, ei := range got {
				alive[ei] = false
			}
			w++
			continue
		}
		if w == 0 || rewinds >= 24 {
			return nil, false
		}
		for _, ei := range committed[w-1] {
			alive[ei] = true
		}
		committed = committed[:w-1]
		attempt[w-1]++
		attempt[w] = 0
		w--
		rewinds++
	}
	out := make([][][2]string, 0, weeks)
	for _, wk := range committed {
		one := make([][2]string, 0, perWeek)
		for _, ei := range wk {
			one = append(one, pairs[ei])
		}
		sort.SliceStable(one, func(i, j int) bool {
			ai, bi := one[i][0], one[i][1]
			if ai > bi {
				ai, bi = bi, ai
			}
			aj, bj := one[j][0], one[j][1]
			if aj > bj {
				aj, bj = bj, aj
			}
			if ai != aj {
				return ai < aj
			}
			return bi < bj
		})
		out = append(out, one)
	}
	return out, true
}

// orientSwissBalanced orients an even-degree undirected edge set so every
// vertex has equal in/out degree (4H/4A for degree 8). It walks deterministic
// Hierholzer Eulerian trails over sorted adjacency: each closed trail enters
// and leaves every vertex equally often, so orienting edges along the walk
// balances homes and aways exactly. Pure function of the edge set.
func orientSwissBalanced(edges []swissPair) []swissDirected {
	elist := make([]swissPair, 0, len(edges))
	elist = append(elist, edges...)
	// other returns the endpoint of edge i that is not v.
	other := func(i int, v string) string {
		if elist[i].a == v {
			return elist[i].b
		}
		return elist[i].a
	}
	// Sorted adjacency: edge indices by neighbour ID, then index.
	adj := map[string][]int{}
	for i, e := range elist {
		adj[e.a] = append(adj[e.a], i)
		adj[e.b] = append(adj[e.b], i)
	}
	for node, lst := range adj {
		sort.SliceStable(lst, func(i, j int) bool {
			ni, nj := other(lst[i], node), other(lst[j], node)
			if ni != nj {
				return ni < nj
			}
			return lst[i] < lst[j]
		})
		adj[node] = lst
	}
	used := make([]bool, len(elist))
	out := make([]swissDirected, 0, len(elist))
	// Deterministic node order for component starts.
	nodes := make([]string, 0, len(adj))
	for n := range adj {
		nodes = append(nodes, n)
	}
	sort.Strings(nodes)
	ptr := map[string]int{}
	// nextUnused scans components in node order for remaining edges.
	nextUnused := func() (string, bool) {
		for _, n := range nodes {
			for ptr[n] < len(adj[n]) && used[adj[n][ptr[n]]] {
				ptr[n]++
			}
			if ptr[n] < len(adj[n]) {
				return n, true
			}
		}
		return "", false
	}
	for {
		start, ok := nextUnused()
		if !ok {
			break
		}
		// One maximal trail from start; even degrees close it at start.
		vStack := []string{start}
		for len(vStack) > 0 {
			v := vStack[len(vStack)-1]
			for ptr[v] < len(adj[v]) && used[adj[v][ptr[v]]] {
				ptr[v]++
			}
			if ptr[v] >= len(adj[v]) {
				vStack = vStack[:len(vStack)-1]
				continue
			}
			ei := adj[v][ptr[v]]
			ptr[v]++
			if used[ei] {
				continue
			}
			used[ei] = true
			next := other(ei, v)
			vStack = append(vStack, next)
			out = append(out, swissDirected{home: v, away: next, round: elist[ei].round})
		}
	}
	// Safety net (unreachable for connected even-degree input): orient leftovers.
	for i, e := range elist {
		if !used[i] {
			h, a := e.a, e.b
			if h > a {
				h, a = a, h
			}
			out = append(out, swissDirected{home: h, away: a, round: e.round})
		}
	}
	return out
}

func winnerIDForFixture(f *Fixture) string {
	if f == nil || f.HomeGoals == nil || f.AwayGoals == nil {
		return ""
	}
	if *f.HomeGoals > *f.AwayGoals {
		return f.HomeID
	}
	if *f.AwayGoals > *f.HomeGoals {
		return f.AwayID
	}
	if len(f.Penalties) >= 2 && f.Penalties[1] > f.Penalties[0] {
		return f.AwayID
	}
	return f.HomeID
}

func (tm *TournamentManager) advanceWorldCompetitionUnlocked(compID string) string {
	comp := tm.worldCompetitionUnlocked(compID)
	if comp == nil || comp.ChampionID != "" || comp.Kind == CompetitionLeague {
		return ""
	}
	if comp.Kind == CompetitionEuropean && comp.Stage == "League Phase" {
		for _, id := range comp.LeaguePhaseFixtureIDs {
			f := tm.worldFixtureUnlocked(id)
			if f == nil || f.Status != "finished" {
				return ""
			}
		}
		tm.scheduleEuropeanPlayoffUnlocked(comp)
		return comp.Name + " league phase complete."
	}
	if len(comp.Rounds) == 0 {
		return ""
	}
	last := &comp.Rounds[len(comp.Rounds)-1]
	// Expected winners: one per tie when TieIDs exist (1 fixture for
	// single-leg, 2 for two-legged legs), else one per fixture (legacy saves
	// persisted before TieIDs). The old `!= TieIDs && != FixtureIDs` check
	// deadlocked legacy rounds where both WinnerIDs and TieIDs were empty.
	wantWinners := len(last.TieIDs)
	if wantWinners == 0 {
		wantWinners = len(last.FixtureIDs)
	}
	if len(last.WinnerIDs) != wantWinners {
		// Two-legged rounds resolve per tie (aggregate); single-leg per fixture.
		// TieIDs is authoritative when present; legacy saves fall back to fixtures.
		tieOrder := append([]string(nil), last.TieIDs...)
		if len(tieOrder) == 0 {
			tieOrder = append([]string(nil), last.FixtureIDs...)
		}
		sort.Strings(tieOrder)
		// Map tie -> fixtures for deterministic resolution, rebuilt from the
		// fixtures' own TieID fields so single-leg (TieID == FixtureID) and
		// two-legged (two FixtureIDs share one TieID) group correctly.
		byTie := map[string][]string{}
		for _, fid := range last.FixtureIDs {
			tie := fid
			if f := tm.worldFixtureUnlocked(fid); f != nil && f.TieID != "" {
				tie = f.TieID
			}
			byTie[tie] = append(byTie[tie], fid)
		}
		winners := make([]string, 0, len(tieOrder))
		for _, tie := range tieOrder {
			fids := byTie[tie]
			if len(fids) == 0 {
				return ""
			}
			sort.Strings(fids)
			for _, fid := range fids {
				f := tm.worldFixtureUnlocked(fid)
				if f == nil || f.Status != "finished" {
					return ""
				}
			}
			winner := tm.winnerIDForWorldTieUnlocked(comp, tie, fids)
			if winner == "" {
				return ""
			}
			winners = append(winners, winner)
		}
		last.WinnerIDs = winners
		if len(last.TieIDs) == 0 {
			// Legacy single-leg round: keep FixtureIDs length contract.
		}
	}
	next := append([]string(nil), last.ByeIDs...)
	next = append(next, last.WinnerIDs...)
	next = append(next, comp.PendingByeIDs...)
	comp.PendingByeIDs = nil
	if len(next) == 1 {
		comp.ChampionID = next[0]
		comp.Stage = "Complete"
		return fmt.Sprintf("%s won the %s.", tm.Clubs[next[0]].ClubName, comp.Name)
	}
	tm.scheduleWorldKnockoutRoundUnlocked(comp, next)
	return ""
}

func (tm *TournamentManager) scheduleEuropeanPlayoffUnlocked(comp *Competition) {
	ranked := tm.worldEuropeanStandingsUnlocked(comp)
	if len(ranked) >= 24 {
		byes := make([]string, 0, 8)
		for i := 0; i < 8; i++ {
			byes = append(byes, ranked[i].ClubID)
		}
		comp.PendingByeIDs = byes
		entrants := make([]string, 0, 16)
		for i := 0; i < 8; i++ {
			entrants = append(entrants, ranked[8+i].ClubID, ranked[23-i].ClubID)
		}
		tm.scheduleWorldKnockoutRoundUnlocked(comp, entrants)
		comp.Rounds[len(comp.Rounds)-1].Stage = "Knockout play-off"
		comp.Stage = "Knockout play-off"
		return
	}
	if len(ranked) < 12 {
		return
	}
	comp.PendingByeIDs = []string{ranked[0].ClubID, ranked[1].ClubID, ranked[2].ClubID, ranked[3].ClubID}
	entrants := []string{ranked[4].ClubID, ranked[11].ClubID, ranked[5].ClubID, ranked[10].ClubID, ranked[6].ClubID, ranked[9].ClubID, ranked[7].ClubID, ranked[8].ClubID}
	tm.scheduleWorldKnockoutRoundUnlocked(comp, entrants)
	comp.Rounds[len(comp.Rounds)-1].Stage = "Knockout play-off"
	comp.Stage = "Knockout play-off"
}

func (tm *TournamentManager) scheduleWorldKnockoutRoundUnlocked(comp *Competition, entrants []string) {
	if comp == nil || len(entrants) < 2 || len(entrants)%2 != 0 {
		return
	}
	roundIndex := len(comp.Rounds)
	stage := stageForKnockoutSize(len(entrants))
	// Any European round scheduled while byes are still pending is the
	// knockout play-off (16 entrants + 8 byes in the 36-team draw, 8 + 4 in
	// the 20-team draws), regardless of entrant count. Without this, the
	// 16-team playoff mislabels as "Round of 16" and collides with the real
	// round of 16 on weeks 30/31.
	if comp.Kind == CompetitionEuropean && len(comp.PendingByeIDs) > 0 {
		stage = "Knockout play-off"
	}
	twoLegged := isEuropeanTwoLeggedStage(comp, stage, len(entrants))
	week1, week2, _ := europeanKnockoutLegWeeks(stage)
	if !twoLegged {
		week1 = cupRoundWeekFor(comp.ID, roundIndex)
		if comp.Kind == CompetitionEuropean {
			week1 = europeanKnockoutWeek(roundIndex)
			// Final single-leg always closes the calendar.
			if len(entrants) == 2 {
				week1 = 38
			}
		}
		week2 = week1
	}
	round := KnockoutRound{Stage: stage, EntrantIDs: append([]string(nil), entrants...), FixtureIDs: []string{}, TieIDs: []string{}}
	for i := 0; i < len(entrants); i += 2 {
		homeID, awayID := entrants[i], entrants[i+1]
		if (i/2+roundIndex)%2 == 1 {
			homeID, awayID = awayID, homeID
		}
		tieID := fmt.Sprintf("%s-KO%d-%s-%s", comp.ID, roundIndex+1, entrants[i], entrants[i+1])
		// TieID is canonical in entrant order so both legs share one aggregate.
		if !twoLegged {
			id := fmt.Sprintf("%s-KO%d-%s-%s", comp.ID, roundIndex+1, homeID, awayID)
			round.TieIDs = append(round.TieIDs, id)
			tm.addWorldFixtureUnlocked(tm.newWorldFixtureUnlocked(id, week1, comp.ID, stage, homeID, awayID))
			round.FixtureIDs = append(round.FixtureIDs, id)
			continue
		}
		round.TieIDs = append(round.TieIDs, tieID)
		leg1ID := fmt.Sprintf("%s-KO%d-L1-%s-%s", comp.ID, roundIndex+1, homeID, awayID)
		leg2ID := fmt.Sprintf("%s-KO%d-L2-%s-%s", comp.ID, roundIndex+1, awayID, homeID)
		tm.addWorldFixtureUnlocked(tm.newWorldLegFixtureUnlocked(leg1ID, week1, comp.ID, stage, homeID, awayID, tieID, 1))
		tm.addWorldFixtureUnlocked(tm.newWorldLegFixtureUnlocked(leg2ID, week2, comp.ID, stage, awayID, homeID, tieID, 2))
		round.FixtureIDs = append(round.FixtureIDs, leg1ID, leg2ID)
	}
	sort.Strings(round.TieIDs)
	sort.Strings(round.FixtureIDs)
	comp.Rounds = append(comp.Rounds, round)
	comp.Stage = stage
}

// winnerIDForWorldTieUnlocked resolves a tie without manufacturing results:
// single-leg ties use the fixture winner; two-legged ties use the real
// aggregate across both finished legs (extra time/penalties already landed
// on the decider via the knockout decider). No fake winners.
func (tm *TournamentManager) winnerIDForWorldTieUnlocked(comp *Competition, tieID string, fixtureIDs []string) string {
	if len(fixtureIDs) == 0 {
		return ""
	}
	sort.Strings(fixtureIDs)
	if len(fixtureIDs) == 1 {
		return winnerIDForFixture(tm.worldFixtureUnlocked(fixtureIDs[0]))
	}
	// Two-legged aggregate by club ID (venues swap, so sum per club).
	// Clubs are collected in fixture order, never by Go map iteration.
	totals := map[string]int{}
	ids := make([]string, 0, 2)
	seen := map[string]bool{}
	for _, fid := range fixtureIDs {
		f := tm.worldFixtureUnlocked(fid)
		if f == nil || f.HomeGoals == nil || f.AwayGoals == nil {
			return ""
		}
		totals[f.HomeID] += *f.HomeGoals
		totals[f.AwayID] += *f.AwayGoals
		for _, id := range []string{f.HomeID, f.AwayID} {
			if !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}
	if len(ids) != 2 {
		// Unexpected shape: fall back to decider leg winner (never fake).
		last := tm.worldFixtureUnlocked(fixtureIDs[len(fixtureIDs)-1])
		return winnerIDForFixture(last)
	}
	// Deterministic club order for tie-breaks.
	sort.Strings(ids)
	if totals[ids[0]] != totals[ids[1]] {
		if totals[ids[0]] > totals[ids[1]] {
			return ids[0]
		}
		return ids[1]
	}
	// Level on aggregate: the decider leg carries ET/pens.
	for _, fid := range fixtureIDs {
		f := tm.worldFixtureUnlocked(fid)
		if f != nil && f.Leg == 2 {
			return winnerIDForFixture(f)
		}
	}
	last := tm.worldFixtureUnlocked(fixtureIDs[len(fixtureIDs)-1])
	return winnerIDForFixture(last)
}

// worldTieLegsUnlocked returns a tie's legs in leg order (deterministic).
func (tm *TournamentManager) worldTieLegsUnlocked(tieID string) []*Fixture {
	if tieID == "" || tm.World == nil {
		return nil
	}
	var out []*Fixture
	for i := range tm.World.Fixtures {
		f := &tm.World.Fixtures[i]
		if f.TieID == tieID {
			out = append(out, f)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Leg != out[j].Leg {
			return out[i].Leg < out[j].Leg
		}
		return out[i].FixtureID < out[j].FixtureID
	})
	return out
}

// worldLegBlocked prevents playing a second leg before the first is decided.
// A return leg with no scheduled first leg is corrupt data, never an open
// tie: report it rather than letting the decider simulate blind.
func (tm *TournamentManager) worldLegBlocked(f *Fixture) string {
	if f == nil || f.Leg != 2 || f.TieID == "" {
		return ""
	}
	if tm.worldCompetitionUnlocked(f.Competition) == nil {
		return ""
	}
	for _, leg := range tm.worldTieLegsUnlocked(f.TieID) {
		if leg.Leg == 1 {
			if leg.Status != "finished" {
				return "Play the first leg before the return."
			}
			return ""
		}
	}
	return "First leg of this tie is missing from the calendar."
}

func (tm *TournamentManager) worldEuropeanStandingsUnlocked(comp *Competition) []*models.Club {
	clubs := clubsForIDs(tm.Clubs, comp.ParticipantIDs)
	sort.SliceStable(clubs, func(i, j int) bool {
		a, b := comp.Records[clubs[i].ClubID], comp.Records[clubs[j].ClubID]
		if a == nil {
			a = &models.CompetitionRecord{}
		}
		if b == nil {
			b = &models.CompetitionRecord{}
		}
		if a.Points != b.Points {
			return a.Points > b.Points
		}
		if a.GoalDifference != b.GoalDifference {
			return a.GoalDifference > b.GoalDifference
		}
		if a.GoalsFor != b.GoalsFor {
			return a.GoalsFor > b.GoalsFor
		}
		if clubs[i].OverallTeamRating != clubs[j].OverallTeamRating {
			return clubs[i].OverallTeamRating > clubs[j].OverallTeamRating
		}
		return clubs[i].ClubID < clubs[j].ClubID
	})
	return clubs
}
