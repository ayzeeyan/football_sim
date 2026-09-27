// Package medical models injury risk, severity, and rehabilitation. All
// functions are pure: risk scoring reads only its input struct, and severity
// selection consumes the caller's RNG stream so match resolution stays
// deterministic under the universe seed.
package medical

import (
	"fmt"
	"math/rand"
	"sort"
)

// Severity tiers. The distribution intentionally keeps the historical shape
// of the legacy roll (most injuries are minor, serious ones are rare) while
// adding the moderate tier the old two-list model could not express.
const (
	SeverityMinor    = "minor"
	SeverityModerate = "moderate"
	SeveritySerious  = "serious"
)

// RiskInput is the per-player, per-match injury-risk read.
type RiskInput struct {
	Age           int
	Fitness       int     // 0-100
	MinutesPlayed int     // minutes in the current match
	MatchDensity  float64 // matches per week over the season so far
	IsWonderkid   bool
	HighPress     bool // club plays a high-press style
}

// RiskFactor is one contribution to the risk multiplier, surfaced so the UI
// can explain a medical assessment.
type RiskFactor struct {
	Label string  `json:"label"`
	Mult  float64 `json:"mult"`
}

// RiskMultiplier returns a multiplier around 1.0 for the base injury chance,
// plus the itemised factors behind it. Pure and deterministic.
func RiskMultiplier(in RiskInput) (float64, []RiskFactor) {
	mult := 1.0
	var factors []RiskFactor
	add := func(label string, m float64) {
		mult *= m
		factors = append(factors, RiskFactor{Label: label, Mult: round2(m)})
	}

	// Fitness: a fatigued body is more vulnerable; a fresh one resists.
	switch {
	case in.Fitness > 0 && in.Fitness < 60:
		add("fitness below 60", 1.5)
	case in.Fitness < 75:
		add("elevated fatigue", 1.2)
	case in.Fitness >= 90:
		add("excellent fitness", 0.85)
	}

	// Age: developing bodies and veteran bodies both carry extra risk.
	switch {
	case in.Age > 0 && in.Age <= 19:
		add("young, still-developing body", 1.25)
	case in.Age >= 33:
		add("veteran age", 1.3)
	case in.Age >= 30:
		add("advancing age", 1.1)
	}

	// Load: full matches accumulate risk.
	if in.MinutesPlayed >= 80 {
		add("full-match load", 1.15)
	}

	// Match density: more matches per week than the calendar allows for
	// recovery raises risk sharply.
	switch {
	case in.MatchDensity > 1.05:
		add("congested schedule", 1.35)
	case in.MatchDensity > 0.9:
		add("busy schedule", 1.15)
	}

	// Style: high pressing costs bodies.
	if in.HighPress {
		add("high-press style", 1.25)
	}

	// Franchise protection: the narrative wonderkids keep their legacy
	// shielding.
	if in.IsWonderkid {
		add("franchise wonderkid", 0.65)
	}

	if len(factors) == 0 {
		factors = append(factors, RiskFactor{Label: "no elevated factors", Mult: 1.0})
	}
	return round2(mult), factors
}

// RiskScore converts the multiplier to a 0-100 assessment for display.
func RiskScore(in RiskInput) int {
	mult, _ := RiskMultiplier(in)
	score := int(mult*50 + 0.5)
	if score < 1 {
		return 1
	}
	if score > 100 {
		return 100
	}
	return score
}

type severitySpec struct {
	name   string
	kinds  []string
	minOut int
	maxOut int // inclusive
	weight float64
}

var severityTable = []severitySpec{
	{SeverityMinor, []string{"knock", "hamstring strain", "ankle sprain", "thigh strain", "calf issue"}, 1, 3, 0.80},
	{SeverityModerate, []string{"hamstring tear", "ankle ligament damage", "stress fracture", "knee sprain"}, 4, 10, 0.17},
	{SeveritySerious, []string{"ACL tear", "meniscus tear", "ruptured cruciate ligament"}, 15, 25, 0.03},
}

// Injury is one rolled injury: tier, kind, and the matches expected out.
type Injury struct {
	Severity   string `json:"severity"`
	Kind       string `json:"kind"`
	MatchesOut int    `json:"matches_out"`
}

// RollInjury draws one injury from the severity distribution using the
// caller's stream. Deterministic for a given RNG state.
func RollInjury(rng *rand.Rand) Injury {
	draw := rng.Float64()
	cumulative := 0.0
	for _, spec := range severityTable {
		cumulative += spec.weight
		if draw < cumulative {
			out := spec.minOut
			if spec.maxOut > spec.minOut {
				out += rng.Intn(spec.maxOut - spec.minOut + 1)
			}
			return Injury{
				Severity:   spec.name,
				Kind:       spec.kinds[rng.Intn(len(spec.kinds))],
				MatchesOut: out,
			}
		}
	}
	// Floating-point shortfall: fall back to the first tier.
	spec := severityTable[0]
	return Injury{Severity: spec.name, Kind: spec.kinds[0], MatchesOut: spec.minOut}
}

// RehabStage is one phase of a rehabilitation plan.
type RehabStage struct {
	Phase  string `json:"phase"`
	Detail string `json:"detail"`
}

// RehabPlan is the expected rehabilitation roadmap for one injury.
type RehabPlan struct {
	Kind       string       `json:"kind"`
	Severity   string       `json:"severity"`
	MatchesOut int          `json:"matches_out"`
	Stages     []RehabStage `json:"stages"`
}

// RehabPlanFor builds the roadmap for a rolled injury.
func RehabPlanFor(inj Injury) RehabPlan {
	plan := RehabPlan{Kind: inj.Kind, Severity: inj.Severity, MatchesOut: inj.MatchesOut}
	switch inj.Severity {
	case SeveritySerious:
		plan.Stages = []RehabStage{
			{Phase: "Acute care", Detail: "Post-injury stabilisation and imaging; weeks 1-3"},
			{Phase: "Rehab strength block", Detail: "Rebuild the affected joint and surrounding musculature"},
			{Phase: "Reconditioning", Detail: "Return to full running and contact work"},
			{Phase: "Squad reintegration", Detail: "Reserved minutes before a full return"},
		}
	case SeverityModerate:
		plan.Stages = []RehabStage{
			{Phase: "Acute care", Detail: "Initial rest and mobility; the first week"},
			{Phase: "Rehab strength block", Detail: "Progressive loading of the affected area"},
			{Phase: "Squad reintegration", Detail: "Full training, then match minutes"},
		}
	default:
		plan.Stages = []RehabStage{
			{Phase: "Acute care", Detail: "Short rest and treatment; days, not weeks"},
			{Phase: "Squad reintegration", Detail: "Back in contention as symptoms clear"},
		}
	}
	return plan
}

// Assessment is the per-player medical read served to the UI.
type Assessment struct {
	RiskScore int          `json:"risk_score"`
	Factors   []RiskFactor `json:"factors"`
}

// Assess builds the display assessment for one player read.
func Assess(in RiskInput) Assessment {
	_, factors := RiskMultiplier(in)
	return Assessment{RiskScore: RiskScore(in), Factors: factors}
}

// HistorySummary aggregates one club's injury history for the medical view.
type HistorySummary struct {
	Season        string         `json:"season"`
	TotalInjuries int            `json:"total_injuries"`
	BySeverity    map[string]int `json:"by_severity"`
	MatchesLost   int            `json:"matches_lost"`
}

// SummarizeHistory aggregates injury records for one season. Records must
// arrive in a deterministic order from the caller.
func SummarizeHistory(season string, records []Record) HistorySummary {
	summary := HistorySummary{Season: season, BySeverity: map[string]int{}}
	for _, rec := range records {
		if rec.Season != season {
			continue
		}
		summary.TotalInjuries++
		summary.BySeverity[rec.Severity]++
		summary.MatchesLost += rec.MatchesOut
	}
	return summary
}

// Record is one historical injury entry persisted on a player.
type Record struct {
	Season     string `json:"season"`
	Matchweek  int    `json:"matchweek"`
	Kind       string `json:"kind"`
	Severity   string `json:"severity"`
	MatchesOut int    `json:"matches_out"`
	FixtureID  string `json:"fixture_id,omitempty"`
}

// MaxHistoryPerPlayer bounds the persisted history; the latest entries win.
const MaxHistoryPerPlayer = 20

// AppendHistory appends one record, keeping the newest MaxHistoryPerPlayer
// entries.
func AppendHistory(history []Record, rec Record) []Record {
	history = append(history, rec)
	if len(history) > MaxHistoryPerPlayer {
		history = history[len(history)-MaxHistoryPerPlayer:]
	}
	return history
}

// SortFactors orders a factor list deterministically (largest multiplier
// first, then label) so UI rendering never depends on build order.
func SortFactors(factors []RiskFactor) {
	sort.SliceStable(factors, func(i, j int) bool {
		if factors[i].Mult != factors[j].Mult {
			return factors[i].Mult > factors[j].Mult
		}
		return factors[i].Label < factors[j].Label
	})
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

// DescribeInjury renders a one-line human summary for news and the UI.
func DescribeInjury(inj Injury) string {
	switch inj.Severity {
	case SeveritySerious:
		return fmt.Sprintf("severe %s, ruled out for %d matches", inj.Kind, inj.MatchesOut)
	case SeverityModerate:
		return fmt.Sprintf("%s, expected out for %d matches", inj.Kind, inj.MatchesOut)
	default:
		unit := "match"
		if inj.MatchesOut > 1 {
			unit = "matches"
		}
		return fmt.Sprintf("%s, out %d %s", inj.Kind, inj.MatchesOut, unit)
	}
}
