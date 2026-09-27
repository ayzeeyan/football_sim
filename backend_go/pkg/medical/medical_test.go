package medical

import (
	"math/rand"
	"testing"
)

func TestRiskMultiplierIsMonotonicInItsFactors(t *testing.T) {
	base := RiskInput{Age: 25, Fitness: 85, MinutesPlayed: 60, MatchDensity: 0.5}

	baseMult, baseFactors := RiskMultiplier(base)
	if len(baseFactors) == 0 {
		t.Fatal("risk factors must always be itemised")
	}

	// Low fitness raises risk.
	lowFitness, _ := RiskMultiplier(RiskInput{Age: 25, Fitness: 50, MinutesPlayed: 60, MatchDensity: 0.5})
	if lowFitness <= baseMult {
		t.Fatalf("low fitness must raise risk: %v vs %v", lowFitness, baseMult)
	}
	// Congested schedule raises risk.
	congested, _ := RiskMultiplier(RiskInput{Age: 25, Fitness: 85, MinutesPlayed: 60, MatchDensity: 1.2})
	if congested <= baseMult {
		t.Fatalf("congested schedule must raise risk: %v vs %v", congested, baseMult)
	}
	// Veteran age raises risk.
	veteran, _ := RiskMultiplier(RiskInput{Age: 34, Fitness: 85, MinutesPlayed: 60, MatchDensity: 0.5})
	if veteran <= baseMult {
		t.Fatalf("veteran age must raise risk: %v vs %v", veteran, baseMult)
	}
	// Full-match load raises risk.
	fullMatch, _ := RiskMultiplier(RiskInput{Age: 25, Fitness: 85, MinutesPlayed: 90, MatchDensity: 0.5})
	if fullMatch <= baseMult {
		t.Fatalf("full-match load must raise risk: %v vs %v", fullMatch, baseMult)
	}
	// Wonderkids keep their legacy shielding.
	shielded, _ := RiskMultiplier(RiskInput{Age: 17, Fitness: 85, MinutesPlayed: 60, MatchDensity: 0.5, IsWonderkid: true})
	unshielded, _ := RiskMultiplier(RiskInput{Age: 17, Fitness: 85, MinutesPlayed: 60, MatchDensity: 0.5})
	if shielded >= unshielded {
		t.Fatalf("wonderkid shielding must lower risk: %v vs %v", shielded, unshielded)
	}
	// Risk score stays in 1..100 and tracks the multiplier.
	if s := RiskScore(base); s < 1 || s > 100 {
		t.Fatalf("risk score out of bounds: %d", s)
	}
	if RiskScore(RiskInput{Age: 34, Fitness: 40, MatchDensity: 1.2, HighPress: true}) <= RiskScore(base) {
		t.Fatal("risky profile must outscore the baseline")
	}
}

func TestRollInjuryDeterministicAndBounded(t *testing.T) {
	first := RollInjury(rand.New(rand.NewSource(42)))
	second := RollInjury(rand.New(rand.NewSource(42)))
	if first != second {
		t.Fatalf("same seed must roll the same injury: %+v vs %+v", first, second)
	}

	// Distribution sanity over a large sample: every tier appears, every
	// roll is within its tier's bounds.
	counts := map[string]int{}
	for i := 0; i < 5000; i++ {
		inj := RollInjury(rand.New(rand.NewSource(int64(i))))
		switch inj.Severity {
		case SeverityMinor:
			if inj.MatchesOut < 1 || inj.MatchesOut > 3 {
				t.Fatalf("minor injury out of bounds: %+v", inj)
			}
		case SeverityModerate:
			if inj.MatchesOut < 4 || inj.MatchesOut > 10 {
				t.Fatalf("moderate injury out of bounds: %+v", inj)
			}
		case SeveritySerious:
			if inj.MatchesOut < 15 || inj.MatchesOut > 25 {
				t.Fatalf("serious injury out of bounds: %+v", inj)
			}
		default:
			t.Fatalf("unknown severity: %+v", inj)
		}
		if inj.Kind == "" {
			t.Fatalf("injury without a kind: %+v", inj)
		}
		counts[inj.Severity]++
	}
	for _, tier := range []string{SeverityMinor, SeverityModerate, SeveritySerious} {
		if counts[tier] == 0 {
			t.Fatalf("tier %q never rolled in 5000 draws", tier)
		}
	}
	if counts[SeverityMinor] < counts[SeverityModerate] || counts[SeverityModerate] < counts[SeveritySerious] {
		t.Fatalf("tier distribution must be minor > moderate > serious: %v", counts)
	}
}

func TestRehabPlanStagesMatchSeverity(t *testing.T) {
	minor := RehabPlanFor(Injury{Severity: SeverityMinor, Kind: "knock", MatchesOut: 2})
	moderate := RehabPlanFor(Injury{Severity: SeverityModerate, Kind: "hamstring tear", MatchesOut: 7})
	serious := RehabPlanFor(Injury{Severity: SeveritySerious, Kind: "ACL tear", MatchesOut: 20})
	if len(minor.Stages) >= len(moderate.Stages) || len(moderate.Stages) >= len(serious.Stages) {
		t.Fatalf("rehab stages must scale with severity: %d / %d / %d", len(minor.Stages), len(moderate.Stages), len(serious.Stages))
	}
	for _, plan := range []RehabPlan{minor, moderate, serious} {
		if plan.Kind == "" || plan.MatchesOut == 0 {
			t.Fatalf("rehab plan missing injury data: %+v", plan)
		}
		for _, stage := range plan.Stages {
			if stage.Phase == "" || stage.Detail == "" {
				t.Fatalf("rehab stage missing content: %+v", stage)
			}
		}
	}
}

func TestHistoryAppendBoundAndSummary(t *testing.T) {
	var history []Record
	for i := 0; i < MaxHistoryPerPlayer+10; i++ {
		history = AppendHistory(history, Record{
			Season: "2026-27", Matchweek: i + 1,
			Kind: "knock", Severity: SeverityMinor, MatchesOut: 1,
		})
	}
	if len(history) != MaxHistoryPerPlayer {
		t.Fatalf("history must cap at %d, got %d", MaxHistoryPerPlayer, len(history))
	}
	// The latest entries survive.
	if history[len(history)-1].Matchweek != MaxHistoryPerPlayer+10 {
		t.Fatalf("latest history entry must survive: %+v", history[len(history)-1])
	}

	summary := SummarizeHistory("2026-27", append(history, Record{
		Season: "2025-26", Matchweek: 2, Kind: "ACL tear", Severity: SeveritySerious, MatchesOut: 20,
	}))
	if summary.TotalInjuries != MaxHistoryPerPlayer {
		t.Fatalf("summary must count only the requested season: %+v", summary)
	}
	if summary.BySeverity[SeverityMinor] != MaxHistoryPerPlayer || summary.MatchesLost != MaxHistoryPerPlayer {
		t.Fatalf("summary aggregation wrong: %+v", summary)
	}
}

func TestSortFactorsAndDescribe(t *testing.T) {
	factors := []RiskFactor{{Label: "b", Mult: 0.85}, {Label: "a", Mult: 1.5}, {Label: "c", Mult: 1.2}}
	SortFactors(factors)
	if factors[0].Label != "a" || factors[1].Label != "c" || factors[2].Label != "b" {
		t.Fatalf("factors must sort by multiplier desc: %+v", factors)
	}
	if DescribeInjury(Injury{Severity: SeverityMinor, Kind: "knock", MatchesOut: 1}) == "" {
		t.Fatal("describe must render a line")
	}
}
