package matchreport

import (
	"fmt"
	"math"
	"math/rand"
	"sort"

	"football_sim/pkg/growth"
	"football_sim/pkg/models"
)

// Simulation helpers ported from match_report.py: minute stoppage text,
// referees, Poisson/weighted sampling, set-piece taker selection, card and
// substitution planning, team-stat construction, and report assembly.

// --------------------------------------------------------------------------
// RNG primitives
// --------------------------------------------------------------------------

// ensureRNG substitutes a fresh source when no RNG is supplied.
func ensureRNG(rng *rand.Rand) *rand.Rand {
	if rng == nil {
		return rand.New(rand.NewSource(rand.Int63()))
	}
	return rng
}

// Poisson draws a Poisson-distributed count via Knuth's algorithm.
// All simulation lambdas are small (<= ~13), so this is exact and fast.
func Poisson(rng *rand.Rand, lambda float64) int {
	rng = ensureRNG(rng)
	if lambda <= 0 {
		return 0
	}
	l := math.Exp(-lambda)
	k := 0
	p := 1.0
	for p > l {
		k++
		p *= rng.Float64()
	}
	return k - 1
}

// weightedIndex picks an index proportionally to non-negative weights.
// Falls back to uniform choice when all weights are zero.
func weightedIndex(rng *rand.Rand, weights []float64) int {
	rng = ensureRNG(rng)
	var total float64
	for _, w := range weights {
		total += w
	}
	if total <= 0 || len(weights) == 0 {
		if len(weights) == 0 {
			return -1
		}
		return rng.Intn(len(weights))
	}
	r := rng.Float64() * total
	var cum float64
	for i, w := range weights {
		cum += w
		if r <= cum {
			return i
		}
	}
	return len(weights) - 1
}

// WeightedChoice picks an index proportionally to weights (exported for the
// live engine; falls back to uniform choice on all-zero weights).
func WeightedChoice(rng *rand.Rand, weights []float64) int {
	return weightedIndex(rng, weights)
}

// SampleMinutes draws n unique sorted minutes from 1..90 (Python:
// sorted(random.sample(range(1, 91), k=min(n, 90)))).
func SampleMinutes(rng *rand.Rand, n int) []int {
	rng = ensureRNG(rng)
	if n <= 0 {
		return nil
	}
	if n > 90 {
		n = 90
	}
	perm := rng.Perm(90)
	out := make([]int, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, perm[i]+1)
	}
	sort.Ints(out)
	return out
}

// gammaSample draws from Gamma(alpha, 1) via Marsaglia-Tsang (alpha >= 1)
// with the alpha<1 boost transformation.
func gammaSample(rng *rand.Rand, alpha float64) float64 {
	if alpha < 1 {
		return gammaSample(rng, alpha+1) * math.Pow(rng.Float64(), 1.0/alpha)
	}
	d := alpha - 1.0/3.0
	c := 1.0 / math.Sqrt(9.0*d)
	for {
		x := rng.NormFloat64()
		v := 1.0 + c*x
		if v <= 0 {
			continue
		}
		v = v * v * v
		u := rng.Float64()
		if u < 1.0-0.0331*(x*x)*(x*x) {
			return d * v
		}
		if math.Log(u) < 0.5*x*x+d*(1.0-v+math.Log(v)) {
			return d * v
		}
	}
}

// Beta draws a Beta(a, b) variate (Python: np.random.beta).
func Beta(rng *rand.Rand, a, b float64) float64 {
	rng = ensureRNG(rng)
	g1 := gammaSample(rng, a)
	g2 := gammaSample(rng, b)
	return g1 / (g1 + g2)
}

// --------------------------------------------------------------------------
// Minutes and referees
// --------------------------------------------------------------------------

// MinuteDisplay renders stoppage-aware minute text (Python: minute_display).
func MinuteDisplay(minute int, rng *rand.Rand) string {
	rng = ensureRNG(rng)
	switch {
	case minute > 120:
		return "120+1'"
	case minute > 90:
		return intSuffix(minute)
	case minute >= 44 && minute <= 45:
		return "45+" + intSuffix(1+rng.Intn(4))
	case minute >= 89:
		return "90+" + intSuffix(1+rng.Intn(5))
	default:
		return intSuffix(minute)
	}
}

func intSuffix(m int) string {
	return fmt.Sprintf("%d'", m)
}

// Referees lists the 12 canonical officials (Python: REFEREES).
var Referees = []string{
	"Michael Oliver", "Anthony Taylor", "Daniele Orsato", "Istvan Kovacs",
	"Felix Zwayer", "Clement Turpin", "Szymon Marciniak", "Jesus Gil Manzano",
	"Danny Makkelie", "Francois Letexier", "Slavko Vincic", "Ivan Barton",
}

// RefereePersonalities maps officials to strict/lenient/balanced.
var RefereePersonalities = map[string]string{
	"Michael Oliver": "strict", "Anthony Taylor": "strict",
	"Jesus Gil Manzano": "strict", "Felix Zwayer": "strict",
	"Daniele Orsato": "lenient", "Clement Turpin": "lenient",
	"Francois Letexier": "lenient", "Danny Makkelie": "lenient",
	"Szymon Marciniak": "balanced", "Istvan Kovacs": "balanced",
	"Slavko Vincic": "balanced", "Ivan Barton": "balanced",
}

// RefereePersonality resolves an official's style, defaulting to balanced.
func RefereePersonality(name string) string {
	if p, ok := RefereePersonalities[name]; ok {
		return p
	}
	return "balanced"
}

// PickRefereeName resolves the officiating referee name. A known official
// name is honored; a personality hint draws a random official of that style;
// anything else draws from the full roster.
func PickRefereeName(rng *rand.Rand, hint string) string {
	rng = ensureRNG(rng)
	if _, ok := RefereePersonalities[hint]; ok {
		return hint
	}
	if hint == "strict" || hint == "lenient" || hint == "balanced" {
		var pool []string
		for _, name := range Referees {
			if RefereePersonalities[name] == hint {
				pool = append(pool, name)
			}
		}
		return pool[rng.Intn(len(pool))]
	}
	return Referees[rng.Intn(len(Referees))]
}

// RefCardMult maps a referee personality to its card-rate multiplier.
func RefCardMult(personality string) float64 {
	switch personality {
	case "strict":
		return 1.20
	case "lenient":
		return 0.80
	default:
		return 1.0
	}
}

// --------------------------------------------------------------------------
// Pick helpers (scorer, assister, set pieces, bookings, subs)
// --------------------------------------------------------------------------

// PickScorer weights FWD/MID candidates by squared OVR with wonderkid,
// mentor, personality, composure, and contextual big-game boosts. The
// optional bigGame argument keeps existing callers source-compatible while
// making the default context neutral.
func PickScorer(xi []*models.Player, minute int, isClutch bool, rng *rand.Rand, bigGame ...bool) *models.Player {
	rng = ensureRNG(rng)
	inBigGame := len(bigGame) > 0 && bigGame[0]
	if len(xi) == 0 {
		return nil
	}
	cands := make([]*models.Player, 0, len(xi))
	for _, p := range xi {
		if p.Category == "FWD" || p.Category == "MID" {
			cands = append(cands, p)
		}
	}
	if len(cands) == 0 {
		cands = xi
	}
	weights := make([]float64, len(cands))
	for i, p := range cands {
		base := math.Pow(float64(p.OVR)/75.0, 2)
		if p.Category == "FWD" {
			base *= 3.0
		} else {
			base *= 1.2
		}
		if p.UniverseWonderkid {
			mult := 1.0
			if p.MentorName != "" {
				mult += 0.08
			}
			if p.Personality == "big_game_performer" && inBigGame {
				mult += 0.18
			} else if p.Personality == "flamboyant_star" {
				mult += 0.12
			}
			if p.Composure > 75 {
				mult += math.Min(0.12, float64(p.Composure-75)*0.004)
			}
			base *= mult
		}
		weights[i] = base
	}
	return cands[weightedIndex(rng, weights)]
}

// PickAssister returns a weighted teammate (72% of goals assisted, GK eligible).
func PickAssister(xi []*models.Player, scorer *models.Player, rng *rand.Rand) *models.Player {
	rng = ensureRNG(rng)
	if rng.Float64() >= 0.72 || len(xi) < 2 {
		return nil
	}
	cands := make([]*models.Player, 0, len(xi))
	for _, p := range xi {
		if p != scorer {
			cands = append(cands, p)
		}
	}
	// len(xi) >= 2 guarantees at least one candidate here.
	weights := make([]float64, len(cands))
	for i, p := range cands {
		mult := 1.5
		if p.Category == "MID" {
			mult = 2.5
		}
		weights[i] = float64(p.OVR) / 75.0 * mult
	}
	return cands[weightedIndex(rng, weights)]
}

// aerialScore rates a header target via growth biometrics or OVR fallback.
func aerialScore(p *models.Player, ge *growth.GrowthEngine) float64 {
	if ge != nil {
		attrs := ge.Attributes[p.PlayerID]
		bio := ge.Biometrics[p.PlayerID]
		if attrs != nil && bio != nil {
			return (float64(attrs.AerialReach)*0.5 + float64(attrs.HeadingPower)*0.5) * (bio.CurrentHeightCM / 180.0)
		}
		if attrs != nil {
			return float64(attrs.AerialReach)*0.5 + float64(attrs.HeadingPower)*0.5
		}
	}
	bonus := 1.0
	if p.Category == "DEF" || p.Category == "FWD" {
		bonus = 1.15
	}
	return float64(p.EffectiveOVR()) * bonus
}

// PickAerialTarget selects a corner header target (squared biometric scores).
func PickAerialTarget(xi []*models.Player, ge *growth.GrowthEngine, rng *rand.Rand) *models.Player {
	rng = ensureRNG(rng)
	if len(xi) == 0 {
		return nil
	}
	cands := make([]*models.Player, 0, len(xi))
	for _, p := range xi {
		if p.Category == "DEF" || p.Category == "FWD" || p.Category == "MID" {
			cands = append(cands, p)
		}
	}
	if len(cands) == 0 {
		cands = xi
	}
	weights := make([]float64, len(cands))
	for i, p := range cands {
		s := aerialScore(p, ge)
		weights[i] = math.Max(1.0, s*s)
	}
	return cands[weightedIndex(rng, weights)]
}

// PickCornerTaker selects a corner taker (never the aerial target).
func PickCornerTaker(xi []*models.Player, target *models.Player, rng *rand.Rand) *models.Player {
	rng = ensureRNG(rng)
	cands := make([]*models.Player, 0, len(xi))
	for _, p := range xi {
		if p != target && (p.Category == "MID" || p.Category == "FWD" || p.Category == "DEF") {
			cands = append(cands, p)
		}
	}
	if len(cands) == 0 {
		return nil
	}
	weights := make([]float64, len(cands))
	for i, p := range cands {
		mult := 1.2
		if p.Category == "MID" {
			mult = 2.0
		}
		weights[i] = float64(p.EffectiveOVR()) / 75.0 * mult
	}
	return cands[weightedIndex(rng, weights)]
}

// PickFreeKickTaker returns the best dead-ball specialist (shooting >= 85).
func PickFreeKickTaker(xi []*models.Player, ge *growth.GrowthEngine) *models.Player {
	type qualified struct {
		shooting int
		player   *models.Player
	}
	var list []qualified
	for _, p := range xi {
		sh := 0
		if ge != nil {
			if attrs := ge.Attributes[p.PlayerID]; attrs != nil {
				sh = attrs.Shooting
			}
		}
		if sh == 0 {
			switch p.Category {
			case "FWD":
				sh = p.OVR
			case "MID":
				sh = p.OVR - 5
			default:
				sh = p.OVR - 20
			}
		}
		if sh >= 85 {
			list = append(list, qualified{shooting: sh, player: p})
		}
	}
	if len(list) == 0 {
		return nil
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].shooting > list[j].shooting })
	return list[0].player
}

// PickBooked selects a booking candidate (DEF/MID preferred, uniform).
func PickBooked(xi []*models.Player, rng *rand.Rand) *models.Player {
	rng = ensureRNG(rng)
	if len(xi) == 0 {
		return nil
	}
	cands := make([]*models.Player, 0, len(xi))
	for _, p := range xi {
		if p.Category == "DEF" || p.Category == "MID" {
			cands = append(cands, p)
		}
	}
	if len(cands) == 0 {
		cands = xi
	}
	return cands[rng.Intn(len(cands))]
}

// PickOwnGoalCulprit selects the unfortunate defender for an own goal
// (uniform among DEF/GK on the field, falling back to anyone available).
func PickOwnGoalCulprit(defSideLive, fallback []*models.Player, rng *rand.Rand) *models.Player {
	rng = ensureRNG(rng)
	var culprits []*models.Player
	for _, p := range defSideLive {
		if p.Category == "DEF" || p.Category == "GK" {
			culprits = append(culprits, p)
		}
	}
	if len(culprits) == 0 {
		culprits = defSideLive
	}
	if len(culprits) == 0 {
		culprits = fallback
	}
	if len(culprits) == 0 {
		return nil
	}
	return culprits[rng.Intn(len(culprits))]
}

// PlannedSub is a scheduled substitution (Python: plan_substitutions rows).
type PlannedSub struct {
	Minute int
	Out    *models.Player
	In     *models.Player
}

// PlanSubstitutions schedules 3-5 like-for-like changes (HT + 56-87 double-headers).
func PlanSubstitutions(xi []*models.Player, bench []*models.Player, maxN int, rng *rand.Rand) []PlannedSub {
	rng = ensureRNG(rng)
	if len(bench) == 0 || len(xi) == 0 {
		return nil
	}
	if maxN < 3 {
		maxN = 3
	}
	n := minInt(len(bench), 3+rng.Intn(maxN-2))
	var minutes []int
	if rng.Float64() < 0.38 {
		minutes = append(minutes, 45)
		n--
	}
	if n > 0 {
		pool := make([]int, 0, 32)
		for m := 56; m < 88; m++ {
			pool = append(pool, m)
		}
		rng.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
		picked := append([]int(nil), pool[:n]...)
		sort.Ints(picked)
		if len(picked) >= 2 && rng.Float64() < 0.42 {
			picked[1] = picked[0]
		}
		minutes = append(minutes, picked...)
	}
	sort.Ints(minutes)
	// Total minutes can never exceed maxN: at most one HT sub plus n picks
	// where n <= maxN (decremented when the HT sub is taken).

	usedOut := map[string]bool{}
	usedIn := map[string]bool{}
	var planned []PlannedSub
	for _, minute := range minutes {
		var outs []*models.Player
		for _, p := range xi {
			if !usedOut[p.PlayerID] && p.Category != "GK" {
				outs = append(outs, p)
			}
		}
		if len(outs) == 0 {
			continue
		}
		outW := make([]float64, len(outs))
		for i, p := range outs {
			if p.UniverseWonderkid {
				outW[i] = 0.35
			} else {
				outW[i] = 1.25
			}
		}
		outP := outs[weightedIndex(rng, outW)]

		var ins []*models.Player
		for _, p := range bench {
			if !usedIn[p.PlayerID] && p.Category != "GK" {
				ins = append(ins, p)
			}
		}
		if len(ins) == 0 {
			continue
		}
		var poolIn []*models.Player
		for _, p := range ins {
			if p.Category == outP.Category {
				poolIn = append(poolIn, p)
			}
		}
		if len(poolIn) == 0 {
			poolIn = ins
		}
		inW := make([]float64, len(poolIn))
		for i, p := range poolIn {
			if p.UniverseWonderkid {
				inW[i] = 1.7
			} else {
				inW[i] = 1.0
			}
		}
		inP := poolIn[weightedIndex(rng, inW)]
		usedOut[outP.PlayerID] = true
		usedIn[inP.PlayerID] = true
		planned = append(planned, PlannedSub{Minute: minute, Out: outP, In: inP})
	}
	return planned
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// --------------------------------------------------------------------------
// Team stats
// --------------------------------------------------------------------------

// PassesSplit splits total passes by possession share (Python: passes_split).
func PassesSplit(homePoss int, rng *rand.Rand) (int, int) {
	rng = ensureRNG(rng)
	total := 820 + rng.Intn(141)
	h := int(float64(total) * float64(homePoss) / 100.0 * (0.96 + rng.Float64()*0.08))
	a := total - h
	if a < 120 {
		a = 120
	}
	return h, a
}

// PassAccuracy derives completion % from team OVR with rain penalty.
func PassAccuracy(ovr int, weather string, rng *rand.Rand) int {
	rng = ensureRNG(rng)
	acc := 76.0 + float64(ovr-70)*1.1 + (rng.Float64()*5.0 - 2.5)
	if weather == "rain" {
		acc -= 5.0
	}
	return int(math.Max(65, math.Min(94, acc)))
}

// BuildStats constructs both team-stat blocks from events (Python: build_stats).
func BuildStats(
	homeClub *models.Club,
	awayClub *models.Club,
	homeShots int,
	awayShots int,
	homeOn int,
	awayOn int,
	homePoss int,
	homeCorners int,
	awayCorners int,
	events []MatchEventItem,
	weather string,
	rng *rand.Rand,
) MatchStats {
	rng = ensureRNG(rng)
	hPass, aPass := PassesSplit(homePoss, rng)

	sentOff := map[string]bool{}
	for _, e := range events {
		if e.Type == "red" && e.SentOff && e.Player != nil {
			sentOff[e.Player.PlayerID] = true
		}
	}
	countCards := func(side string) (yellows, reds int) {
		for _, e := range events {
			if e.Side != side {
				continue
			}
			switch e.Type {
			case "yellow":
				if e.Player != nil && !sentOff[e.Player.PlayerID] {
					yellows++
				}
			case "red":
				reds++
			}
		}
		return yellows, reds
	}
	hY, hR := countCards("home")
	aY, aR := countCards("away")

	xg := func(shots, onT, goals int) float64 {
		off := shots - onT
		if off < 0 {
			off = 0
		}
		raw := float64(onT)*0.28 + float64(off)*0.07 + float64(goals)*0.12
		raw += rng.Float64()*0.30 - 0.15
		return math.Round(math.Max(0.05, math.Min(4.8, raw))*100) / 100
	}
	countGoals := func(side string) int {
		n := 0
		for _, e := range events {
			switch e.Type {
			case "goal", "penalty", "corner_goal", "free_kick_goal":
				if e.Side == side && !e.Disallowed {
					n++
				}
			case "own_goal":
				if e.Beneficiary == side {
					n++
				}
			}
		}
		return n
	}
	hGoals := countGoals("home")
	aGoals := countGoals("away")

	foulBase := 12
	if weather == "rain" {
		foulBase = 13
	}

	return MatchStats{
		Home: TeamStats{
			Possession:   homePoss,
			Shots:        homeShots,
			ShotsOn:      homeOn,
			XG:           xg(homeShots, homeOn, hGoals),
			Passes:       hPass,
			PassAccuracy: PassAccuracy(homeClub.OverallTeamRating, weather, rng),
			Corners:      homeCorners,
			Fouls:        Poisson(rng, float64(foulBase)),
			YellowCards:  hY,
			RedCards:     hR,
			Saves:        maxInt(0, awayOn-aGoals),
		},
		Away: TeamStats{
			Possession:   100 - homePoss,
			Shots:        awayShots,
			ShotsOn:      awayOn,
			XG:           xg(awayShots, awayOn, aGoals),
			Passes:       aPass,
			PassAccuracy: PassAccuracy(awayClub.OverallTeamRating, weather, rng),
			Corners:      awayCorners,
			Fouls:        Poisson(rng, float64(foulBase)),
			YellowCards:  aY,
			RedCards:     aR,
			Saves:        maxInt(0, homeOn-hGoals),
		},
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// --------------------------------------------------------------------------
// Assembly
// --------------------------------------------------------------------------

// InstantPayload mirrors the Python generate_instant_match payload dict.
type InstantPayload struct {
	HomeGoals  int
	AwayGoals  int
	HomeClubName string
	AwayClubName string
	Events     []MatchEventItem
	HomeXI     []*models.Player
	AwayXI     []*models.Player
	HomeBench  []*models.Player
	AwayBench  []*models.Player
	Stats      MatchStats
	HTHome     int
	HTAway     int
	Attendance int
	Referee    string
	Weather    string
	DecidedBy  *string
	Penalties  interface{}
	ShotMap    ShotMapData
	Heatmap    TouchHeatmapData
	Press      PressConferenceData
}

// AssembleReport merges a payload into a frontend-ready report with rated
// XIs/benches and man-of-the-match (Python: assemble_report).
func AssembleReport(payload InstantPayload, method string, rng *rand.Rand) MatchReport {
	rng = ensureRNG(rng)
	events := append([]MatchEventItem(nil), payload.Events...)
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].Minute == events[j].Minute {
			return events[i].Seq < events[j].Seq
		}
		return events[i].Minute < events[j].Minute
	})

	var homeClubID, awayClubID string
	for _, p := range payload.HomeXI {
		if p != nil && p.ClubID != "" {
			homeClubID = p.ClubID
			break
		}
	}
	for _, p := range payload.AwayXI {
		if p != nil && p.ClubID != "" {
			awayClubID = p.ClubID
			break
		}
	}

	for i := range events {
		ev := &events[i]
		if ev.Side == "home" {
			if ev.ClubID == "" {
				ev.ClubID = homeClubID
			}
			if ev.ClubName == "" {
				ev.ClubName = payload.HomeClubName
			}
		} else if ev.Side == "away" {
			if ev.ClubID == "" {
				ev.ClubID = awayClubID
			}
			if ev.ClubName == "" {
				ev.ClubName = payload.AwayClubName
			}
		}
		if ev.Player != nil {
			if ev.PlayerID == "" {
				ev.PlayerID = ev.Player.PlayerID
			}
			if ev.PlayerName == "" {
				ev.PlayerName = ev.Player.FullName
			}
		} else if ev.Scorer != nil {
			if ev.PlayerID == "" {
				ev.PlayerID = ev.Scorer.PlayerID
			}
			if ev.PlayerName == "" {
				ev.PlayerName = ev.Scorer.FullName
			}
		} else if ev.PlayerIn != nil {
			if ev.PlayerID == "" {
				ev.PlayerID = ev.PlayerIn.PlayerID
			}
			if ev.PlayerName == "" {
				ev.PlayerName = ev.PlayerIn.FullName
			}
		}
	}

	homeWon := payload.HomeGoals > payload.AwayGoals
	awayWon := payload.AwayGoals > payload.HomeGoals
	homeRows := RateXI(payload.HomeXI, events, "home", payload.AwayGoals, homeWon, true, rng)
	awayRows := RateXI(payload.AwayXI, events, "away", payload.HomeGoals, awayWon, true, rng)
	homeBenchRows := RateXI(payload.HomeBench, events, "home", payload.AwayGoals, homeWon, false, rng)
	awayBenchRows := RateXI(payload.AwayBench, events, "away", payload.HomeGoals, awayWon, false, rng)

	var motmRow *MatchPlayerRow
	motmSide := "home"
	consider := func(side string, rows []MatchPlayerRow) {
		for i := range rows {
			row := &rows[i]
			if !row.Played {
				continue
			}
			rating := 0.0
			if row.Rating != nil {
				rating = *row.Rating
			}
			bestRating := -1.0
			bestGoals := -1
			bestOVR := -1
			if motmRow != nil {
				if motmRow.Rating != nil {
					bestRating = *motmRow.Rating
				}
				bestGoals = motmRow.MatchGoals
				bestOVR = motmRow.OVR
			}
			if motmRow == nil ||
				rating > bestRating ||
				(rating == bestRating && row.MatchGoals > bestGoals) ||
				(rating == bestRating && row.MatchGoals == bestGoals && row.OVR > bestOVR) {
				motmRow = row
				motmSide = side
			}
		}
	}
	consider("home", homeRows)
	consider("home", homeBenchRows)
	consider("away", awayRows)
	consider("away", awayBenchRows)

	var motm *MatchPlayerRow
	if motmRow != nil {
		cpy := *motmRow
		cpy.Side = motmSide
		motm = &cpy
	}

	return MatchReport{
		Method:          method,
		HomeGoals:       payload.HomeGoals,
		AwayGoals:       payload.AwayGoals,
		Events:          events,
		HomeXI:          homeRows,
		AwayXI:          awayRows,
		HomeBench:       homeBenchRows,
		AwayBench:       awayBenchRows,
		Stats:           payload.Stats,
		ShotMap:         payload.ShotMap,
		Heatmap:         payload.Heatmap,
		PressConference: payload.Press,
		MOTM:            motm,
		HTHome:          payload.HTHome,
		HTAway:          payload.HTAway,
		Attendance:      payload.Attendance,
		Referee:         payload.Referee,
		Weather:         payload.Weather,
		DecidedBy:       payload.DecidedBy,
		Penalties:       payload.Penalties,
	}
}
