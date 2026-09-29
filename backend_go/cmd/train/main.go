// Command train runs the offline FootballMoE training pipeline. It is the
// only component in the repository that modifies model weights; the runtime
// never trains.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"football_sim/pkg/footballai"
	"football_sim/pkg/footballai/training"
)

func main() {
	dataFlag := flag.String("data", "", "Training data directory containing the JSONL datasets")
	outFlag := flag.String("out", "", "Output .fmoe deployment model path")
	ckptFlag := flag.String("checkpoint", "", "Optional .fmckpt training checkpoint path")
	resumeFlag := flag.String("resume", "", "Optional .fmckpt to resume from")
	epochsFlag := flag.Int("epochs", 40, "Training epochs")
	batchFlag := flag.Int("batch", 512, "Batch size")
	lrFlag := flag.Float64("lr", 1e-3, "Initial learning rate")
	wdFlag := flag.Float64("weight-decay", 1e-4, "AdamW weight decay")
	clipFlag := flag.Float64("grad-clip", 1.0, "Gradient clipping global norm")
	balanceFlag := flag.Float64("router-balance", 0.01, "Router load-balancing coefficient")
	seedFlag := flag.Int64("seed", 42, "Deterministic training seed")
	tasksFlag := flag.String("tasks", "", "Comma-separated dataset tasks to train (empty = all; curriculum phases)")
	taskWeightsFlag := flag.String("task-weights", "", "Comma-separated task=weight pairs for head loss weighting")
	patienceFlag := flag.Int("patience", 8, "Early-stopping patience in epochs (0 = off)")
	versionFlag := flag.Uint("model-version", 1, "Exported model version")
	flag.Parse()

	if *dataFlag == "" {
		fmt.Fprintln(os.Stderr, "train: -data is required")
		flag.Usage()
		os.Exit(2)
	}
	if *outFlag == "" {
		fmt.Fprintln(os.Stderr, "train: -out is required")
		flag.Usage()
		os.Exit(2)
	}

	opts := training.Options{
		DataDir:           *dataFlag,
		OutPath:           *outFlag,
		CheckpointPath:    *ckptFlag,
		ResumePath:        *resumeFlag,
		Epochs:            *epochsFlag,
		BatchSize:         *batchFlag,
		LearningRate:      *lrFlag,
		WeightDecay:       *wdFlag,
		GradClip:          *clipFlag,
		BalanceCoef:       float32(*balanceFlag),
		Seed:              *seedFlag,
		EarlyStopPatience: *patienceFlag,
		ModelVersion:      uint32(*versionFlag),
		LabelWeights:      training.DefaultLabelWeights(),
	}

	if *tasksFlag != "" {
		for _, t := range strings.Split(*tasksFlag, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				opts.Tasks = append(opts.Tasks, t)
			}
		}
	}
	if *taskWeightsFlag != "" {
		opts.TaskLossWeights = map[string]float64{}
		for _, pair := range strings.Split(*taskWeightsFlag, ",") {
			parts := strings.SplitN(strings.TrimSpace(pair), "=", 2)
			if len(parts) == 2 {
				var w float64
				fmt.Sscanf(parts[1], "%g", &w)
				opts.TaskLossWeights[parts[0]] = w
			}
		}
	}

	fmt.Printf("FootballMoE trainer\n")
	fmt.Printf("  data           %s\n", opts.DataDir)
	fmt.Printf("  out            %s\n", opts.OutPath)
	fmt.Printf("  epochs         %d\n", opts.Epochs)
	fmt.Printf("  batch          %d\n", opts.BatchSize)
	fmt.Printf("  lr             %g\n", opts.LearningRate)
	fmt.Printf("  seed           %d\n", opts.Seed)
	fmt.Printf("  label weights  bootstrap=%.2f simulation=%.2f historical=%.2f human=%.2f\n",
		opts.LabelWeights[footballai.LabelBootstrapTeacher],
		opts.LabelWeights[footballai.LabelSimulation],
		opts.LabelWeights[footballai.LabelHistorical],
		opts.LabelWeights[footballai.LabelHumanCurated])

	res, err := training.Train(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "train: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\nDone. Best epoch %d, val loss %.4f, test loss %.4f, %d parameters, hash %s\n",
		res.BestEpoch, res.BestValLoss, res.TestLoss, res.ParamCount, res.ModelHash)
}
