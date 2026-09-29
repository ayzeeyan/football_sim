// Command modeltool inspects, verifies, lists, and benchmarks FootballMoE
// model files without touching the game server.
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"time"

	"football_sim/pkg/footballai"
	"football_sim/pkg/footballai/weights"
)

func usage() {
	fmt.Fprintln(os.Stderr, `modeltool — FootballMoE model inspection

Usage:
  modeltool inspect <model.fmoe>
  modeltool verify <model.fmoe>
  modeltool tensors <model.fmoe>
  modeltool benchmark <model.fmoe>`)
	os.Exit(2)
}

func main() {
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() < 2 {
		usage()
	}
	cmd, path := flag.Arg(0), flag.Arg(1)
	switch cmd {
	case "inspect":
		if err := inspect(path); err != nil {
			fail(err)
		}
	case "verify":
		if err := verify(path); err != nil {
			fail(err)
		}
	case "tensors":
		if err := tensors(path); err != nil {
			fail(err)
		}
	case "benchmark":
		if err := benchmark(path); err != nil {
			fail(err)
		}
	default:
		usage()
	}
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "modeltool: %v\n", err)
	os.Exit(1)
}

func inspect(path string) error {
	ins, err := weights.Inspect(path)
	if err != nil {
		return err
	}
	c := ins.Config
	specs := footballai.TaskSpecs()

	fmt.Println("FootballMoE")
	fmt.Println()
	fmt.Printf("Format:            %s\n", weights.Magic)
	fmt.Printf("Format version:    %d\n", ins.FormatVersion)
	fmt.Printf("Model version:     %d\n", ins.ModelVersion)
	fmt.Printf("DType:             FP32\n")
	fmt.Printf("Parameters:        %d\n", ins.ParameterCount)
	fmt.Printf("Tensors:           %d\n", ins.TensorCount)
	fmt.Printf("Checkpoint:        %v\n", ins.Flags&weights.FlagCheckpoint != 0)
	fmt.Println()
	fmt.Printf("Encoder:           width %d, expansion %d, blocks %d\n", c.StateWidth, c.ExpansionWidth, c.EncoderBlocks)
	fmt.Printf("Router:            top-%d over %d experts (hidden %d)\n", c.TopK, c.NumExperts, c.RouterHidden)
	fmt.Println("Experts:           Match, Player, Economy, Club")
	fmt.Printf("  match widths:    %d / %d\n", c.ExpertWidthA, c.ExpertWidthB)
	fmt.Printf("Manager rep:        %d (hidden %d)\n", c.ManagerWidth, c.ManagerHidden)
	fmt.Printf("Task embedding:     %d\n", c.TaskEmbedWidth)
	fmt.Printf("Head hidden:        %d\n", c.HeadHidden)
	fmt.Printf("Feature schema:    v%d (%d slots)\n", c.FeatureSchemaVersion, c.InputWidth)
	fmt.Printf("Norm schema:       v%d\n", c.NormSchemaVersion)
	fmt.Println()
	fmt.Println("Heads:")
	names := make([]string, 0, len(specs))
	for t := range specs {
		names = append(names, footballai.TaskName(t))
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Printf("  %s\n", name)
	}
	fmt.Println()
	fmt.Printf("Model hash:        %s\n", ins.DataHash)
	fmt.Printf("Checksum:          ")
	if ins.ChecksumOK {
		fmt.Println("OK")
	} else {
		fmt.Println("MISMATCH")
		os.Exit(1)
	}
	return nil
}

func verify(path string) error {
	if err := weights.Verify(path); err != nil {
		return err
	}
	// Full load as the strongest check: architecture, tensors, normalization.
	m, err := weights.Load(path)
	if err != nil {
		return err
	}
	fmt.Printf("OK: %s loads cleanly (%d parameters, model version %d, schema v%d)\n",
		path, m.ParameterCount(), m.Config().ModelVersion, m.Config().FeatureSchemaVersion)
	return nil
}

func tensors(path string) error {
	ins, err := weights.Inspect(path)
	if err != nil {
		return err
	}
	fmt.Printf("%-44s %10s %8s\n", "TENSOR", "DIMS", "ELEMENTS")
	for _, t := range ins.Tensors {
		dims := fmt.Sprint(t.Dims)
		fmt.Printf("%-44s %10s %8d\n", t.Name, dims, t.Length)
	}
	return nil
}

func benchmark(path string) error {
	m, err := weights.Load(path)
	if err != nil {
		return err
	}
	req := footballai.MatchRequest{
		Match: footballai.MatchContext{
			HomeRating: 82, AwayRating: 74, HomeForm: 2, AwayForm: -1,
			HomeFitness: 90, AwayFitness: 85, HomeFatigue: 1, AwayFatigue: 3,
			TacticalEdge: 0.2, MatchImportance: 0.6, HomeAbsences: 1, AwayAbsences: 2,
		},
		World:   footballai.WorldContext{MatchImportance: 0.6, PlayerFatigue: 0.3},
		Manager: footballai.ManagerFeatures{Risk: 0.4, Patience: 0.5, Attacking: 0.6, Pressing: 0.5},
	}

	// Warmup.
	for i := 0; i < 50; i++ {
		if _, err := m.PredictMatch(req); err != nil {
			return err
		}
	}

	start := time.Now()
	const n = 1000
	for i := 0; i < n; i++ {
		if _, err := m.PredictMatch(req); err != nil {
			return err
		}
	}
	elapsed := time.Since(start)
	fmt.Printf("Single prediction latency (mean over %d): %v\n", n, elapsed/n)
	fmt.Printf("%d predictions: %v\n", n, elapsed)

	// Batched.
	batch := make([]footballai.Request, 256)
	for i := range batch {
		batch[i] = req.Encode()
	}
	start = time.Now()
	if _, err := m.PredictBatch(batch); err != nil {
		return err
	}
	fmt.Printf("Batch of 256 (one forward): %v\n", time.Since(start))
	return nil
}
