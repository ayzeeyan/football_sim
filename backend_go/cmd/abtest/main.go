// Command abtest builds two identical worlds from the same universe seed —
// one with FootballMoE enabled, one with the pure deterministic simulation —
// advances both by the same number of matchweeks, and compares the outcome
// distributions. It is the A/B evidence for whether a learned model may
// replace existing behavior: quality must hold, not just training loss.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"football_sim/pkg/datamanager"
	"football_sim/pkg/footballai"

	// The weights package registers itself as the .fmoe loader for Brain.
	_ "football_sim/pkg/footballai/weights"
	"football_sim/pkg/growth"
	"football_sim/pkg/tournament"
	"football_sim/pkg/transfers"
)

// worldStats aggregates the distributional evidence from one world run.
type worldStats struct {
	Matches      int
	HomeWins     int
	Draws        int
	AwayWins     int
	TotalGoals   int
	Injuries     int
	Transfers    int
	TransferFees int64
	AvgGoals     float64
	HomeWinPct   float64
	DrawPct      float64
	AwayWinPct   float64
	AvgGoalsMin  float64 // lower/upper sanity band: 2.2 - 3.4 goals per match
	AvgGoalsMax  float64
}

func collect(tm *tournament.TournamentManager, te *transfers.TransferEngine) worldStats {
	s := worldStats{
		AvgGoalsMin: 2.2,
		AvgGoalsMax: 3.4,
	}
	for i := range tm.Fixtures {
		f := &tm.Fixtures[i]
		if f.Status != "finished" || f.HomeGoals == nil || f.AwayGoals == nil {
			continue
		}
		h, a := *f.HomeGoals, *f.AwayGoals
		s.Matches++
		s.TotalGoals += h + a
		switch {
		case h > a:
			s.HomeWins++
		case h == a:
			s.Draws++
		default:
			s.AwayWins++
		}
	}
	for _, c := range tm.Clubs {
		for _, p := range c.Squad {
			if p == nil {
				continue
			}
			for _, rec := range p.InjuryHistory {
				if rec.Season == tm.SeasonName {
					s.Injuries++
				}
			}
		}
	}
	if te != nil {
		for _, t := range te.AllTransfers() {
			s.Transfers++
			s.TransferFees += t.FeeEUR
		}
	}
	if s.Matches > 0 {
		s.AvgGoals = float64(s.TotalGoals) / float64(s.Matches)
		s.HomeWinPct = float64(s.HomeWins) / float64(s.Matches) * 100
		s.DrawPct = float64(s.Draws) / float64(s.Matches) * 100
		s.AwayWinPct = float64(s.AwayWins) / float64(s.Matches) * 100
	}
	return s
}

// buildWorld constructs one independent, deterministically seeded universe.
func buildWorld(dataset string, seed int64, brain *footballai.Brain) (
	*tournament.TournamentManager, *transfers.TransferEngine, error) {
	rng := tournament.NewSubsystemRNG(seed)
	ge := growth.NewGrowthEngine(rng.SeedFor("development"))
	dm := datamanager.NewDataManager(dataset, ge)
	dm.SetSeed(rng.SeedFor("datamanager"))
	if len(dm.Clubs) == 0 {
		return nil, nil, fmt.Errorf("no clubs in dataset %s", dataset)
	}
	tm := tournament.NewEuropeanWorldManager(dm.ClubsList, ge, rng.SeedFor("matches"))
	te := transfers.NewTransferEngine(dm.ClubsList, tm.Managers, rng.SeedFor("transfers"))
	tm.TransferEngine = te
	tm.AIBrain = brain
	te.AIBrain = brain
	return tm, te, nil
}

func main() {
	datasetFlag := flag.String("dataset", "", "Path to dataset.json")
	modelFlag := flag.String("model", "", "FootballMoE model (.fmoe)")
	weeksFlag := flag.Int("weeks", 6, "Matchweeks to advance per world")
	seedFlag := flag.Int64("seed", 424242, "Universe seed shared by both worlds")
	featuresFlag := flag.String("features", "match,injury", "Feature flags for the AI world")
	flag.Parse()

	dataset := *datasetFlag
	if dataset == "" {
		for _, p := range []string{"dataset.json", "../dataset.json", "../../dataset.json"} {
			if _, err := os.Stat(p); err == nil {
				dataset = p
				break
			}
		}
	}
	if dataset == "" {
		log.Fatal("abtest: -dataset is required")
	}

	// Baseline world: pure deterministic simulation.
	baseTM, baseTE, err := buildWorld(dataset, *seedFlag, nil)
	if err != nil {
		log.Fatal(err)
	}

	// AI world: FootballMoE enabled behind explicit flags.
	var brain *footballai.Brain
	if *modelFlag != "" {
		cfg := footballai.AIConfig{ModelPath: *modelFlag}
		for _, f := range splitCSV(*featuresFlag) {
			switch f {
			case "match":
				cfg.UseMatchModel = true
			case "injury":
				cfg.UseInjuryModel = true
			case "rotation":
				cfg.UseRotationModel = true
			case "valuation":
				cfg.UseValuationModel = true
			}
		}
		brain, err = footballai.NewBrain(cfg, log.Default())
		if err != nil {
			log.Fatal(err)
		}
		if !brain.Enabled() {
			log.Fatal("abtest: model did not enable")
		}
		fmt.Printf("AI world: model v%d, hash %s\n", brain.Info().ModelVersion, brain.Info().ModelHash)
	} else {
		fmt.Println("No -model given: running baseline twice (harness self-check)")
	}
	aiTM, aiTE, err := buildWorld(dataset, *seedFlag, brain)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Advancing %d matchweeks on both worlds (seed %d)...\n", *weeksFlag, *seedFlag)
	for w := 0; w < *weeksFlag; w++ {
		baseTM.SimulateSlateWithDigest()
		aiTM.SimulateSlateWithDigest()
	}

	base := collect(baseTM, baseTE)
	ai := collect(aiTM, aiTE)

	fmt.Println()
	fmt.Printf("%-22s %12s %12s\n", "METRIC", "BASELINE", "AI-ENABLED")
	row := func(name string, b, a float64) {
		fmt.Printf("%-22s %12.2f %12.2f\n", name, b, a)
	}
	row("matches", float64(base.Matches), float64(ai.Matches))
	row("goals/match", base.AvgGoals, ai.AvgGoals)
	row("home win %", base.HomeWinPct, ai.HomeWinPct)
	row("draw %", base.DrawPct, ai.DrawPct)
	row("away win %", base.AwayWinPct, ai.AwayWinPct)
	row("injuries", float64(base.Injuries), float64(ai.Injuries))
	row("transfers", float64(base.Transfers), float64(ai.Transfers))
	fmt.Printf("%-22s %12s %12s\n", "transfer fees (EUR)", fmtMillion(base.TransferFees), fmtMillion(ai.TransferFees))

	// Quality verdict: the AI world must stay inside the same realism bands
	// the baseline simulation itself respects.
	ok := true
	if ai.Matches == 0 {
		fmt.Println("no finished matches to compare")
		os.Exit(1)
	}
	if ai.AvgGoals < ai.AvgGoalsMin || ai.AvgGoals > ai.AvgGoalsMax {
		ok = false
		fmt.Printf("FAIL: goals/match %.2f outside sanity band [%.2f, %.2f]\n", ai.AvgGoals, ai.AvgGoalsMin, ai.AvgGoalsMax)
	}
	// Home advantage must survive: home wins should exceed away wins.
	if ai.HomeWinPct <= ai.AwayWinPct {
		ok = false
		fmt.Printf("FAIL: home advantage inverted (%.1f%% vs %.1f%%)\n", ai.HomeWinPct, ai.AwayWinPct)
	}
	if ok {
		fmt.Println("\nVERDICT: AI-enabled world stays inside the baseline realism bands.")
	} else {
		fmt.Println("\nVERDICT: AI-enabled world drifts outside the realism bands — do not enable these flags.")
		os.Exit(1)
	}
}

func splitCSV(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == ',' {
			if cur != "" {
				out = append(out, cur)
			}
			cur = ""
			continue
		}
		if r != ' ' {
			cur += string(r)
		}
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

func fmtMillion(v int64) string {
	return fmt.Sprintf("%.1fM", float64(v)/1_000_000)
}
