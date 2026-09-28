// Command pretrain fits the world brain's base weights offline.
//
// This is the "small pre-training" stage of the brain's lifecycle: it
// builds a fresh world from the dataset, simulates seasons with the brain
// active (self-play), lets the weekly post-training loop learn from every
// match, and writes the final weights into pkg/brain/base.go as the
// committed artifact. Runtime careers then start from this base and keep
// post-training continuously.
//
// Usage: go run ./cmd/pretrain -dataset dataset.json -seasons 8
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"football_sim/pkg/brain"
	"football_sim/pkg/datamanager"
	"football_sim/pkg/growth"
	"football_sim/pkg/tournament"
	"football_sim/pkg/transfers"
)

func main() {
	datasetFlag := flag.String("dataset", "dataset.json", "Path to dataset.json")
	seasonsFlag := flag.Int("seasons", 8, "Seasons of self-play used to fit the base")
	flag.Parse()

	ge := growth.NewGrowthEngine(8181)
	dm := datamanager.NewDataManager(*datasetFlag, ge)
	if len(dm.ClubsList) != 96 {
		log.Fatalf("dataset clubs=%d want 96", len(dm.ClubsList))
	}
	tm := tournament.NewEuropeanWorldManager(dm.ClubsList, ge, 8181)
	te := transfers.NewTransferEngine(dm.ClubsList, tm.Managers, 9191)
	tm.TransferEngine = te

	for season := 1; season <= *seasonsFlag; season++ {
		batch := tm.SimulateBatchWeeks(tm.MaxMatchweeks)
		if batch.Status != "success" {
			log.Fatalf("season %d failed: %+v", season, batch)
		}
		te.BeginOffSeasonWindow()
		for i := 0; i < transfers.TransferWindowWeeks; i++ {
			te.AdvanceOpenWindow()
		}
		if res := tm.FinalizeSeasonTransition(); res["status"] != "success" {
			log.Fatalf("season %d transition failed: %v", season, res)
		}
	}

	state := tm.BrainState()
	if state == nil {
		log.Fatal("brain never initialised")
	}
	if !state.Valid() {
		log.Fatal("trained brain is invalid")
	}

	var sb strings.Builder
	sb.WriteString("package brain\n\n")
	sb.WriteString("// Base is the pre-trained starting brain, fitted offline by cmd/pretrain\n")
	sb.WriteString("// from simulated seasons and committed as the shipped artifact. Runtime\n")
	sb.WriteString("// post-training continues from here: the brain never stops learning, but\n")
	sb.WriteString("// every career starts from the same good base instead of from zero.\n")
	sb.WriteString("//\n")
	sb.WriteString(fmt.Sprintf("// Fitted from %d seasons of self-play (%d training rows).\n", *seasonsFlag, state.Samples))
	sb.WriteString("func Base() *Model {\n\treturn &Model{\n\t\tWeights: [NumFeatures]float64{\n")
	for _, w := range state.Weights {
		sb.WriteString(fmt.Sprintf("\t\t\t%.6f,\n", w))
	}
	sb.WriteString("\t\t},\n")
	sb.WriteString(fmt.Sprintf("\t\tBias:    %.6f,\n", state.Bias))
	sb.WriteString(fmt.Sprintf("\t\tSamples: %d,\n", state.Samples))
	sb.WriteString(fmt.Sprintf("\t\tLossEMA: %.6f,\n", state.LossEMA))
	sb.WriteString("\t}\n}\n")

	outPath := filepath.Join("pkg", "brain", "base.go")
	if err := os.WriteFile(outPath, []byte(sb.String()), 0o644); err != nil {
		log.Fatalf("write base: %v", err)
	}
	fmt.Printf("wrote %s: %d samples, loss EMA %.4f\n", outPath, state.Samples, state.LossEMA)
	for i, label := range brain.FeatureLabels() {
		fmt.Printf("  %-22s %+.4f\n", label, state.Weights[i])
	}
}
