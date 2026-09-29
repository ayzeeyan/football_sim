package training

import (
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"football_sim/pkg/footballai"
	"football_sim/pkg/footballai/weights"
)

// Options configures one offline training run. Every knob is explicit; no
// value is hidden inside the model.
type Options struct {
	DataDir        string
	OutPath        string
	CheckpointPath string
	ResumePath     string

	Epochs       int
	BatchSize    int
	LearningRate float64
	WeightDecay  float64
	GradClip     float64
	BalanceCoef  float32
	Seed         int64

	// Tasks filters dataset tasks by dataset name (e.g. "match_prediction").
	// Empty means all tasks. "router" is always included when present in the
	// data unless explicitly excluded with a leading '!'.
	Tasks []string

	// TaskLossWeights scales head losses per dataset task name.
	TaskLossWeights map[string]float64

	// LabelWeights scales per-sample loss by label source.
	LabelWeights map[footballai.LabelSource]float64

	EarlyStopPatience int

	ModelVersion uint32

	Log io.Writer
}

// DefaultLabelWeights are the recommended initial relative weights. Real
// outcomes outweigh the bootstrap teacher; the bootstrap data exists to
// warm-start, not to dominate.
func DefaultLabelWeights() map[footballai.LabelSource]float64 {
	return map[footballai.LabelSource]float64{
		footballai.LabelBootstrapTeacher: 0.30,
		footballai.LabelSimulation:       0.80,
		footballai.LabelHistorical:       1.00,
		footballai.LabelHumanCurated:     1.00,
	}
}

// RouterStats summarizes routing behavior over an evaluation pass.
type RouterStats struct {
	Top1Frac  [footballai.NumExperts]float64
	MeanProb  [footballai.NumExperts]float64
	PairFrac  map[string]float64
	TeacherCE float64
	Top1Acc   float64
	PerTask   map[string][footballai.NumExperts]float64

	n            int
	teacherTotal int
}

// EpochStats is one epoch's training summary.
type EpochStats struct {
	Epoch       int
	TrainLoss   float64
	ValLoss     float64
	LR          float64
	Router      RouterStats
	TaskLoss    map[string]float64 // validation loss per task
	TaskMetrics map[string]map[string]float64
}

// Result summarizes a completed training run.
type Result struct {
	BestEpoch   int
	BestValLoss float64
	History     []EpochStats
	TestMetrics map[string]map[string]float64
	TestLoss    float64
	Model       *footballai.Model
	ModelHash   string
	ParamCount  uint64
	Baselines   map[string]map[string]float64
}

// Train runs the full offline pipeline: load, normalize, train, validate,
// early-stop, evaluate, and export.
func Train(opts Options) (*Result, error) {
	log := opts.Log
	if log == nil {
		log = os.Stdout
	}
	if opts.Epochs <= 0 {
		opts.Epochs = 60
	}
	if opts.BatchSize <= 0 {
		opts.BatchSize = 512
	}
	if opts.LearningRate <= 0 {
		opts.LearningRate = 1e-3
	}
	if opts.GradClip <= 0 {
		opts.GradClip = 1.0
	}
	if opts.Seed == 0 {
		opts.Seed = 42
	}
	if opts.LabelWeights == nil {
		opts.LabelWeights = DefaultLabelWeights()
	}

	fmt.Fprintf(log, "FootballMoE trainer — data %s, seed %d\n", opts.DataDir, opts.Seed)

	// 1. Stream the splits.
	taskFilter := newTaskFilter(opts.Tasks)
	train, err := LoadSplit(opts.DataDir, SplitTrain, taskFilter)
	if err != nil {
		return nil, err
	}
	val, err := LoadSplit(opts.DataDir, SplitVal, taskFilter)
	if err != nil {
		return nil, err
	}
	test, err := LoadSplit(opts.DataDir, SplitTest, taskFilter)
	if err != nil {
		return nil, err
	}
	if len(train) == 0 {
		return nil, fmt.Errorf("training: no training samples in %s", opts.DataDir)
	}
	fmt.Fprintf(log, "Loaded %d train / %d val / %d test samples\n", len(train), len(val), len(test))

	// 2. Compute normalization statistics from the training split only and
	// install them into the network; the same metadata is exported.
	cfg := footballai.DefaultConfig()
	if opts.ModelVersion > 0 {
		cfg.ModelVersion = opts.ModelVersion
	}
	net, err := footballai.NewNetwork(cfg, opts.Seed)
	if err != nil {
		return nil, err
	}
	norm := computeNormMeta(train)
	if err := net.SetNormMeta(norm); err != nil {
		return nil, err
	}

	params := net.Params()
	weightDecay := opts.WeightDecay
	if weightDecay < 0 {
		weightDecay = 1e-4
	}
	adam := NewAdamW(params, AdamWOptions{
		LearningRate: opts.LearningRate,
		WeightDecay:  weightDecay,
	})

	startEpoch := 0
	if opts.ResumePath != "" {
		model, state, err := weights.LoadCheckpoint(opts.ResumePath)
		if err != nil {
			return nil, fmt.Errorf("training: resume: %w", err)
		}
		net = model.Network()
		params = net.Params()
		adam = NewAdamW(params, AdamWOptions{
			LearningRate: float64(state.LearningRate),
			WeightDecay:  opts.WeightDecay,
		})
		for name, m := range state.Moments {
			if mm, ok := adam.m[name]; ok && len(mm) == len(m.M) {
				copy(mm, m.M)
			}
			if vv, ok := adam.v[name]; ok && len(vv) == len(m.V) {
				copy(vv, m.V)
			}
		}
		startEpoch = state.Epoch
		// The checkpoint carries the old run's config; a resumed export is a
		// new behavior-affecting model, so honor the requested version.
		if opts.ModelVersion > 0 {
			net.SetModelVersion(opts.ModelVersion)
		}
		fmt.Fprintf(log, "Resumed from %s at epoch %d\n", opts.ResumePath, startEpoch)
	}

	paramCount := net.ParameterCount()
	fmt.Fprintf(log, "Network: %d parameters (target 250k-400k)\n", paramCount)

	// 3. Training loop.
	res := &Result{ParamCount: paramCount}
	var bestParams map[string][]float32
	bestVal := math.Inf(1)
	bestEpoch := 0
	stale := 0

	for epoch := startEpoch; epoch < opts.Epochs; epoch++ {
		lr := scheduleLR(opts.LearningRate, epoch, opts.Epochs)
		adam.opts.LearningRate = lr

		trainLoss, err := trainEpoch(net, adam, train, opts, lr, log)
		if err != nil {
			return nil, fmt.Errorf("training: epoch %d: %w", epoch+1, err)
		}

		valStats, valLoss, err := evaluate(net, val, opts)
		if err != nil {
			return nil, fmt.Errorf("training: validation epoch %d: %w", epoch+1, err)
		}

		res.History = append(res.History, EpochStats{
			Epoch:       epoch + 1,
			TrainLoss:   trainLoss,
			ValLoss:     valLoss,
			LR:          lr,
			Router:      valStats.Router,
			TaskLoss:    valStats.TaskLoss,
			TaskMetrics: valStats.Metrics,
		})

		fmt.Fprintf(log, "Epoch %d/%d  train %.4f  val %.4f  lr %.5f\n",
			epoch+1, opts.Epochs, trainLoss, valLoss, lr)
		printTaskMetrics(log, valStats.Metrics)
		printRouterStats(log, valStats.Router)

		if valLoss < bestVal {
			bestVal = valLoss
			bestEpoch = epoch + 1
			stale = 0
			bestParams = snapshotParams(net)
		} else {
			stale++
			if opts.EarlyStopPatience > 0 && stale >= opts.EarlyStopPatience {
				fmt.Fprintf(log, "Early stopping at epoch %d (best epoch %d)\n", epoch+1, bestEpoch)
				break
			}
		}

		// Checkpoint every epoch so training is resumable after interruption.
		if opts.CheckpointPath != "" {
			state := &weights.CheckpointState{
				Epoch:        epoch + 1,
				Step:         (epoch + 1) * ((len(train) + opts.BatchSize - 1) / opts.BatchSize),
				LearningRate: float32(lr),
				BestValScore: float32(bestVal),
				RandomSeed:   opts.Seed,
				Metrics:      map[string]float64{"val_loss": valLoss, "train_loss": trainLoss},
				Moments:      adam.exportMoments(),
			}
			if _, err := weights.SaveCheckpoint(opts.CheckpointPath, net, state); err != nil {
				return nil, fmt.Errorf("training: checkpoint: %w", err)
			}
		}
	}

	// 4. Restore the best weights.
	if bestParams != nil {
		if err := net.ApplyParams(bestParams); err != nil {
			return nil, fmt.Errorf("training: restore best: %w", err)
		}
	}

	// 5. Test evaluation and baselines.
	testStats, testLoss, err := evaluate(net, test, opts)
	if err != nil {
		return nil, fmt.Errorf("training: test evaluation: %w", err)
	}
	res.TestMetrics = testStats.Metrics
	res.TestLoss = testLoss
	res.BestEpoch = bestEpoch
	res.BestValLoss = bestVal
	res.Baselines = computeBaselines(train, val, test)

	// 6. Export the immutable deployment model.
	res.Model = footballai.BuildModel(net, "")
	hash, err := weights.Save(opts.OutPath, net)
	if err != nil {
		return nil, fmt.Errorf("training: export: %w", err)
	}
	res.ModelHash = hash
	res.Model = footballai.BuildModel(net, hash)

	fmt.Fprintf(log, "Best epoch %d (val %.4f). Test loss %.4f\n", bestEpoch, bestVal, testLoss)
	fmt.Fprintf(log, "Exported %s (model hash %s)\n", opts.OutPath, hash)
	printBaselines(log, res)
	return res, nil
}

// trainEpoch runs one shuffled pass over the training samples.
func trainEpoch(net *footballai.Network, adam *AdamW, samples []*Sample, opts Options, lr float64, log io.Writer) (float64, error) {
	order := rand.New(rand.NewSource(opts.Seed + int64(len(samples))))
	idx := make([]int, len(samples))
	for i := range idx {
		idx[i] = i
	}
	order.Shuffle(len(idx), func(i, j int) { idx[i], idx[j] = idx[j], idx[i] })

	totalLoss := float64(0)
	totalWeight := float64(0)
	specs := footballai.TaskSpecs()
	batches := 0
	for start := 0; start < len(idx); start += opts.BatchSize {
		end := start + opts.BatchSize
		if end > len(idx) {
			end = len(idx)
		}
		batch := make([]*Sample, end-start)
		for i, s := range idx[start:end] {
			batch[i] = samples[s]
		}
		loss, weight, err := trainBatch(net, adam, batch, specs, opts)
		if err != nil {
			return 0, err
		}
		totalLoss += loss
		totalWeight += weight
		batches++
	}
	if totalWeight == 0 {
		return 0, fmt.Errorf("empty weighted batch total")
	}
	return totalLoss / totalWeight, nil
}

// trainBatch performs forward, loss, backward, clip, and optimizer step.
func trainBatch(net *footballai.Network, adam *AdamW, batch []*Sample, specs map[footballai.TaskType]footballai.TaskSpec, opts Options) (float64, float64, error) {
	reqs := make([]footballai.Request, len(batch))
	for i, s := range batch {
		reqs[i] = footballai.Request{Task: s.Task, Slots: s.Slots}
	}
	net.Reset()
	net.ZeroGrad()
	fp, err := net.ForwardTrain(reqs)
	if err != nil {
		return 0, 0, err
	}

	headGrad := make([][]float32, len(batch))
	teacher := make([][]float32, len(batch))
	sampleWeight := make([]float32, len(batch))
	lossSum := float64(0)
	weightSum := float64(0)

	for i, s := range batch {
		w := float32(1)
		if lw, ok := opts.LabelWeights[s.LabelSource]; ok {
			w = float32(lw)
		}
		sampleWeight[i] = w
		if s.IsRouter {
			teacher[i] = s.Teacher
			route := fp.Routing(i)
			lossSum += float64(w) * float64(footballai.KLToTeacher(route.Probs[:], s.Teacher))
			weightSum += float64(w)
			continue
		}
		spec, ok := specs[s.Task]
		if !ok || s.Target == nil {
			continue
		}
		tw := float32(1)
		if opts.TaskLossWeights != nil {
			if v, ok := opts.TaskLossWeights[footballai.TaskName(s.Task)]; ok {
				tw = float32(v)
			}
		}
		raw := fp.RawOutputs(i)
		if raw == nil {
			continue
		}
		grad := make([]float32, len(raw))
		for j, out := range spec.Outputs {
			loss, g := footballai.LossAndGrad(out, raw[j], s.Target[j], w*tw)
			lossSum += float64(loss)
			grad[j] = g
		}
		headGrad[i] = grad
		weightSum += float64(w)
	}

	balanceCoef := opts.BalanceCoef
	if balanceCoef == 0 {
		balanceCoef = 0.01
	}
	// The router balance loss is included in backward but reported
	// separately; it is not part of the comparable task loss.
	net.BackwardTrain(fp, footballai.BackwardInput{
		HeadGrad:      headGrad,
		RouterTeacher: teacher,
		SampleWeight:  sampleWeight,
		BalanceCoef:   balanceCoef,
	})

	if opts.GradClip > 0 {
		ClipGlobalNorm(net.Params(), opts.GradClip)
	}
	if math.IsNaN(lossSum) || math.IsInf(lossSum, 0) {
		return 0, 0, fmt.Errorf("non-finite batch loss (%g); training aborted", lossSum)
	}
	if err := adam.Step(net.Params()); err != nil {
		return 0, 0, err
	}
	net.Reset()
	return lossSum, weightSum, nil
}

// scheduleLR applies a cosine decay to 10% of the base rate.
func scheduleLR(base float64, epoch, totalEpochs int) float64 {
	if totalEpochs <= 1 {
		return base
	}
	t := float64(epoch) / float64(totalEpochs-1)
	return base * (0.1 + 0.45*(math.Cos(t*math.Pi)+1))
}

// evalStats aggregates an evaluation pass.
type evalStats struct {
	Loss     float64
	TaskLoss map[string]float64
	Metrics  map[string]map[string]float64
	Router   RouterStats
}

// evaluate runs the network over a split without training.
func evaluate(net *footballai.Network, samples []*Sample, opts Options) (*evalStats, float64, error) {
	if len(samples) == 0 {
		return &evalStats{TaskLoss: map[string]float64{}, Metrics: map[string]map[string]float64{}, Router: emptyRouterStats()}, 0, nil
	}
	specs := footballai.TaskSpecs()
	st := &evalStats{
		TaskLoss: map[string]float64{},
		Metrics:  map[string]map[string]float64{},
	}
	taskCounts := map[string]float64{}
	router := emptyRouterStats()

	taskMetric := func(task string) map[string]float64 {
		m, ok := st.Metrics[task]
		if !ok {
			m = map[string]float64{}
			st.Metrics[task] = m
		}
		return m
	}

	batchSize := 1024
	for start := 0; start < len(samples); start += batchSize {
		end := start + batchSize
		if end > len(samples) {
			end = len(samples)
		}
		batch := samples[start:end]
		reqs := make([]footballai.Request, len(batch))
		for i, s := range batch {
			reqs[i] = footballai.Request{Task: s.Task, Slots: s.Slots}
		}
		net.Reset()
		fp, err := net.ForwardTrain(reqs)
		if err != nil {
			return nil, 0, err
		}
		for i, s := range batch {
			route := fp.Routing(i)
			accrueRouter(&router, s, route)
			if s.IsRouter {
				ce := footballai.KLToTeacher(route.Probs[:], s.Teacher)
				st.Loss += float64(ce)
				st.TaskLoss["router"] += float64(ce)
				taskCounts["router"]++
				continue
			}
			spec, ok := specs[s.Task]
			if !ok || s.Target == nil {
				continue
			}
			raw := fp.RawOutputs(i)
			if raw == nil {
				continue
			}
			name := footballai.TaskName(s.Task)
			loss := float64(0)
			for j, out := range spec.Outputs {
				l, _ := footballai.LossAndGrad(out, raw[j], s.Target[j], 1)
				loss += float64(l)
			}
			st.Loss += loss
			st.TaskLoss[name] += loss
			taskCounts[name]++
			collectTaskMetrics(taskMetric(name), s, spec, raw)
		}
		net.Reset()
	}

	for name, v := range st.TaskLoss {
		if c := taskCounts[name]; c > 0 {
			st.TaskLoss[name] = v / c
		}
	}
	loss := st.Loss
	if n := float64(len(samples)); n > 0 {
		loss /= n
	}
	finalizeRouter(&router, len(samples))
	st.Router = router
	return st, loss, nil
}

// collectTaskMetrics computes task-appropriate metrics for one sample.
func collectTaskMetrics(m map[string]float64, s *Sample, spec footballai.TaskSpec, raw []float32) {
	add := func(k string, v float64) { m[k+"_sum"] += v; m[k+"_n"]++ }
	switch s.Task {
	case footballai.TaskMatchPrediction:
		home := footballai.Activate(spec.Outputs[0].Kind, raw[0])
		away := footballai.Activate(spec.Outputs[1].Kind, raw[1])
		gd := footballai.Activate(spec.Outputs[2].Kind, raw[2])
		add("mae_xg", (math.Abs(float64(home-s.Target[0]))+math.Abs(float64(away-s.Target[1])))/2)
		add("mae_goal_diff", math.Abs(float64(gd-s.AuxGoalDiff)))
		// Result calibration: predicted sign of goal difference vs actual.
		pred := int8(0)
		if gd > 0.25 {
			pred = 1
		} else if gd < -0.25 {
			pred = -1
		}
		if pred == s.AuxResult {
			m["result_correct"]++
		}
		m["result_total"]++
	case footballai.TaskInjuryRisk:
		p := footballai.Activate(spec.Outputs[0].Kind, raw[0])
		if s.AuxInjury >= 0 {
			d := float64(p) - float64(s.AuxInjury)
			add("brier", d*d)
		}
		add("mae_prob", math.Abs(float64(p-s.Target[0])))
	case footballai.TaskRotation:
		p := footballai.Activate(spec.Outputs[0].Kind, raw[0])
		add("mae_start_prob", math.Abs(float64(p-s.Target[0])))
		add("mae_rest", math.Abs(float64(raw[1]-s.Target[1])))
	case footballai.TaskDevelopment, footballai.TaskDecline:
		add("mae_delta", math.Abs(float64(raw[0]-s.Target[0])))
	case footballai.TaskValuation:
		premium := footballai.Activate(spec.Outputs[0].Kind, raw[0])
		target := float32(math.Exp(float64(s.Target[0])))
		if target > 0 {
			add("mape_premium", math.Abs(float64(premium-target))/float64(target))
		}
		add("log_error", math.Abs(float64(raw[0]-s.Target[0])))
	case footballai.TaskNegotiation:
		acc := footballai.Activate(spec.Outputs[0].Kind, raw[0])
		d := float64(acc - s.Target[0])
		add("brier_accept", d*d)
		add("mae_counter", math.Abs(float64(raw[1]-s.Target[1])))
	case footballai.TaskContract:
		renew := footballai.Activate(spec.Outputs[0].Kind, raw[0])
		d := float64(renew - s.Target[0])
		add("brier_renew", d*d)
		add("log_error_wage", math.Abs(float64(raw[1]-s.Target[1])))
	case footballai.TaskBoardPatience:
		sack := footballai.Activate(spec.Outputs[0].Kind, raw[0])
		d := float64(sack - s.Target[0])
		add("brier_sack", d*d)
	}
}

// summarizeMetrics converts accumulated sums into per-task averages.
func summarizeMetrics(m map[string]float64) map[string]float64 {
	out := map[string]float64{}
	for k, v := range m {
		if !strings.HasSuffix(k, "_sum") {
			continue
		}
		base := strings.TrimSuffix(k, "_sum")
		if n := m[base+"_n"]; n > 0 {
			out[base] = v / n
		}
	}
	if t := m["result_total"]; t > 0 {
		out["result_accuracy"] = m["result_correct"] / t
	}
	return out
}

func emptyRouterStats() RouterStats {
	return RouterStats{
		PairFrac: map[string]float64{},
		PerTask:  map[string][footballai.NumExperts]float64{},
	}
}

func accrueRouter(r *RouterStats, s *Sample, route footballai.Top2) {
	top1 := route.Experts[0]
	r.Top1Frac[top1]++
	for k := 0; k < footballai.NumExperts; k++ {
		r.MeanProb[k] += float64(route.Probs[k])
	}
	a, b := route.Experts[0], route.Experts[1]
	if b < a {
		a, b = b, a
	}
	r.PairFrac[fmt.Sprintf("%s+%s", footballai.ExpertName(a), footballai.ExpertName(b))]++
	var taskProbs [footballai.NumExperts]float64
	p := r.PerTask[footballai.TaskName(s.Task)]
	for k := range taskProbs {
		taskProbs[k] = p[k] + float64(route.Probs[k])
	}
	r.PerTask[footballai.TaskName(s.Task)] = taskProbs
	r.n++
	if s.IsRouter {
		teacherArg := 0
		for k := 1; k < footballai.NumExperts; k++ {
			if s.Teacher[k] > s.Teacher[teacherArg] {
				teacherArg = k
			}
		}
		if teacherArg == int(argmaxProbs(route.Probs[:])) {
			r.Top1Acc++
		}
		r.TeacherCE += float64(footballai.KLToTeacher(route.Probs[:], s.Teacher))
		r.teacherTotal++
	}
}

func argmaxProbs(p []float32) int32 {
	best := int32(0)
	for k := int32(1); k < footballai.NumExperts; k++ {
		if p[k] > p[best] {
			best = k
		}
	}
	return best
}

func finalizeRouter(r *RouterStats, n int) {
	if n == 0 {
		return
	}
	for k := 0; k < footballai.NumExperts; k++ {
		r.Top1Frac[k] /= float64(n)
		r.MeanProb[k] /= float64(n)
	}
	for k := range r.PairFrac {
		r.PairFrac[k] /= float64(n)
	}
	if r.teacherTotal > 0 {
		r.TeacherCE /= float64(r.teacherTotal)
		r.Top1Acc /= float64(r.teacherTotal)
	}
}

func printTaskMetrics(log io.Writer, m map[string]map[string]float64) {
	names := make([]string, 0, len(m))
	for k := range m {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, name := range names {
		mm := summarizeMetrics(m[name])
		keys := make([]string, 0, len(mm))
		for k := range mm {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(log, "  %-18s %-16s %.4f\n", name, k, mm[k])
		}
	}
}

func printRouterStats(log io.Writer, r RouterStats) {
	fmt.Fprintf(log, "  Router top-1:")
	for k := 0; k < footballai.NumExperts; k++ {
		fmt.Fprintf(log, " %s %.1f%%", footballai.ExpertName(footballai.ExpertID(k)), r.Top1Frac[k]*100)
	}
	fmt.Fprintln(log)
	fmt.Fprintf(log, "  Router pairs:")
	for pair, f := range r.PairFrac {
		fmt.Fprintf(log, " %s %.1f%%", pair, f*100)
	}
	fmt.Fprintln(log)
}

func printBaselines(log io.Writer, res *Result) {
	fmt.Fprintln(log, "Baselines (test split):")
	for task, m := range res.Baselines {
		fmt.Fprintf(log, "  %-18s", task)
		for k, v := range m {
			fmt.Fprintf(log, " %s=%.4f", k, v)
		}
		fmt.Fprintln(log)
	}
}

func snapshotParams(net *footballai.Network) map[string][]float32 {
	snap := map[string][]float32{}
	for _, p := range net.Params() {
		c := make([]float32, len(p.Data))
		copy(c, p.Data)
		snap[p.Name] = c
	}
	return snap
}

// LoadSplit streams all dataset files and returns the samples of one split.
func LoadSplit(dataDir, split string, filter func(task string) bool) ([]*Sample, error) {
	var out []*Sample
	for _, file := range DatasetFiles {
		path := filepath.Join(dataDir, file)
		if _, err := os.Stat(path); err != nil {
			return nil, fmt.Errorf("training: missing dataset %s: %w", path, err)
		}
		r, err := OpenDataset(path)
		if err != nil {
			return nil, err
		}
		for {
			s, err := r.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				r.Close()
				return nil, err
			}
			if s.Split != split {
				continue
			}
			name := footballai.TaskName(s.Task)
			if s.IsRouter {
				name = "router"
			}
			if !filter(name) {
				continue
			}
			out = append(out, s)
		}
		if err := r.Close(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// newTaskFilter returns a predicate over dataset task names.
func newTaskFilter(tasks []string) func(string) bool {
	if len(tasks) == 0 {
		return func(string) bool { return true }
	}
	set := map[string]bool{}
	for _, t := range tasks {
		set[t] = true
	}
	return func(name string) bool { return set[name] }
}
