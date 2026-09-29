package footballai

import "sort"

// Network is the complete FootballMoE graph. It is deterministic: given the
// same weights and the same requests it produces the same outputs, with no
// hidden inference-time randomness.
type Network struct {
	cfg     ModelConfig
	specs   map[TaskType]TaskSpec
	norm    NormMeta
	encoder *StateEncoder
	mgrEnc  *ManagerEncoder
	taskEmb *Embedding
	router  *Router
	experts []*Expert
	heads   map[TaskType]*TaskHead
}

// taskRow maps a task to a stable task-embedding row. Runtime tasks use
// their ID; the training-only router task gets the final row.
func taskRow(t TaskType) int {
	if t == TaskRouter {
		return numRuntimeTasks
	}
	return int(t)
}

// numTaskEmbRows is the task embedding table size.
const numTaskEmbRows = numRuntimeTasks + 1

// NewNetwork builds an untrained network with deterministic initialization.
func NewNetwork(cfg ModelConfig, seed int64) (*Network, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if seed == 0 {
		seed = 1
	}
	rng := newDeterministicRand(seed)
	n := &Network{
		cfg:     cfg,
		specs:   TaskSpecs(),
		norm:    DefaultNormMeta(),
		encoder: NewStateEncoder(cfg, rng),
		mgrEnc:  NewManagerEncoder(cfg, rng),
		taskEmb: NewEmbedding("embedding.task", numTaskEmbRows, cfg.TaskEmbedWidth),
		router:  NewRouter(cfg, rng),
	}
	// Small random task embeddings break symmetry without letting the task
	// ID dominate routing; context features still drive expert selection.
	for i := range n.taskEmb.table {
		n.taskEmb.table[i] = float32(rng.NormFloat64() * 0.02)
	}
	for e := 0; e < NumExperts; e++ {
		n.experts = append(n.experts, NewExpert(cfg, ExpertID(e), rng))
	}
	n.heads = map[TaskType]*TaskHead{}
	for _, t := range HeadedTasks() {
		spec := n.specs[t]
		n.heads[t] = NewTaskHead(cfg, t, spec.OutputWidth(), rng)
	}
	return n, nil
}

// Config returns the network's architecture configuration.
func (n *Network) Config() ModelConfig { return n.cfg }

// NormMeta returns the normalization metadata (also serialized in .fmoe).
func (n *Network) NormMeta() NormMeta { return n.norm }

// SetModelVersion overrides the model version (e.g. when a fine-tuning run
// resumes an older checkpoint but exports a new behavior-affecting model).
func (n *Network) SetModelVersion(v uint32) {
	n.cfg.ModelVersion = v
}

// SetNormMeta installs normalization metadata computed from training data.
func (n *Network) SetNormMeta(meta NormMeta) error {
	if len(meta.Slots) != InputWidth {
		return errConfig("normalization metadata covers %d slots, expected %d", len(meta.Slots), InputWidth)
	}
	if meta.SchemaVersion != NormSchemaVersion {
		return errConfig("normalization schema v%d unsupported", meta.SchemaVersion)
	}
	n.norm = meta
	return nil
}

// Params returns all trainable parameters in stable, name-sorted order.
func (n *Network) Params() []*Param {
	ps := []*Param{}
	ps = append(ps, n.encoder.Params()...)
	ps = append(ps, n.mgrEnc.Params()...)
	ps = append(ps, n.taskEmb.Params()...)
	ps = append(ps, n.router.Params()...)
	for _, e := range n.experts {
		ps = append(ps, e.Params()...)
	}
	// Heads in stable task order.
	tasks := make([]TaskType, 0, len(n.heads))
	for t := range n.heads {
		tasks = append(tasks, t)
	}
	sort.Slice(tasks, func(i, j int) bool { return tasks[i] < tasks[j] })
	for _, t := range tasks {
		ps = append(ps, n.heads[t].Params()...)
	}
	sort.Slice(ps, func(i, j int) bool { return ps[i].Name < ps[j].Name })
	return ps
}

// ParameterCount returns the total number of scalar parameters.
func (n *Network) ParameterCount() uint64 {
	total := uint64(0)
	for _, p := range n.Params() {
		total += uint64(len(p.Data))
	}
	return total
}

// Reset drops all per-batch caches (gradients are preserved).
func (n *Network) Reset() {
	n.encoder.Reset()
	n.mgrEnc.Reset()
	n.taskEmb.Reset()
	n.router.Reset()
	for _, e := range n.experts {
		e.Reset()
	}
	for _, h := range n.heads {
		h.Reset()
	}
}

// ZeroGrad zeroes every parameter gradient.
func (n *Network) ZeroGrad() {
	for _, p := range n.Params() {
		p.ZeroGrad()
	}
}

// ApplyParams overwrites parameters from a name→data map. Used by the model
// loader; unknown names are rejected so stale weights cannot silently
// mis-bind.
func (n *Network) ApplyParams(named map[string][]float32) error {
	ps := n.Params()
	seen := make(map[string]bool, len(ps))
	for _, p := range ps {
		data, ok := named[p.Name]
		if !ok {
			return errConfig("model file is missing tensor %q", p.Name)
		}
		if len(data) != len(p.Data) {
			return errConfig("tensor %q has %d elements, architecture expects %d", p.Name, len(data), len(p.Data))
		}
		copy(p.Data, data)
		seen[p.Name] = true
	}
	for name := range named {
		if !seen[name] {
			return errConfig("model file contains unknown tensor %q", name)
		}
	}
	return nil
}

// ForwardPass holds one batch's activations and routing decisions. It is the
// unit of backward for training.
type ForwardPass struct {
	net         *Network
	reqs        []Request
	normIn      [][]float32
	state       [][]float32
	mgrIn       [][]float32
	embIdx      []int
	emb         [][]float32
	routerIn    [][]float32
	probs       [][]float32
	route       []Top2
	expertRow   [NumExperts][]int // sample indices routed to each expert
	expertPosIn [NumExperts][]int // per sample: position inside its expert subset (-1 if unrouted)
	expertOut   [NumExperts][][]float32
	mixed       [][]float32
	headRows    map[TaskType][]int
	headOut     map[TaskType][][]float32
}

// ForwardTrain runs the full forward pass for training and evaluation. The
// caller keeps the returned pass and hands it to BackwardTrain.
func (n *Network) ForwardTrain(reqs []Request) (*ForwardPass, error) {
	return n.forward(reqs)
}

// BackwardTrain accumulates parameter gradients for a forward pass and
// returns the router balance loss applied.
func (n *Network) BackwardTrain(fp *ForwardPass, in BackwardInput) float32 {
	return n.backward(fp, in)
}

// Request task of sample i.
func (fp *ForwardPass) Task(i int) TaskType { return fp.reqs[i].Task }

// Routing returns the routing decision for sample i.
func (fp *ForwardPass) Routing(i int) Top2 { return fp.route[i] }

// RawOutputs returns the raw (pre-activation) head outputs for sample i, or
// nil when the task has no trained head.
func (fp *ForwardPass) RawOutputs(i int) []float32 {
	rows, ok := fp.headRows[fp.reqs[i].Task]
	if !ok {
		return nil
	}
	for pos, r := range rows {
		if r == i {
			return fp.headOut[fp.reqs[i].Task][pos]
		}
	}
	return nil
}

// forward runs the full graph over a batch of requests (training path; use
// the Model wrappers for inference).
func (n *Network) forward(reqs []Request) (*ForwardPass, error) {
	if len(reqs) == 0 {
		return nil, errConfig("empty batch")
	}
	fp := &ForwardPass{
		net:      n,
		headRows: map[TaskType][]int{},
		headOut:  map[TaskType][][]float32{},
	}
	fp.reqs = reqs

	// 1. Normalize raw slots with the stored metadata.
	fp.normIn = make([][]float32, len(reqs))
	for i, r := range reqs {
		if len(r.Slots) != InputWidth {
			return nil, errConfig("request %d has %d slots, expected %d", i, len(r.Slots), InputWidth)
		}
		fp.normIn[i] = n.norm.ApplyVec(r.Slots)
	}

	// 2. Shared state encoder.
	fp.state = n.encoder.Forward(fp.normIn)

	// 3. Manager characteristics encoder.
	fp.mgrIn = make([][]float32, len(reqs))
	for i := range reqs {
		m := make([]float32, managerWidthFeats)
		copy(m, fp.normIn[i][slotMgrStart:slotMgrStart+managerWidthFeats])
		fp.mgrIn[i] = m
	}
	mgr := n.mgrEnc.Forward(fp.mgrIn)

	// 4. Task embedding.
	fp.embIdx = make([]int, len(reqs))
	for i, r := range reqs {
		if int(r.Task) >= numRuntimeTasks && r.Task != TaskRouter {
			return nil, errConfig("unsupported task %d", r.Task)
		}
		fp.embIdx[i] = taskRow(r.Task)
	}
	fp.emb = n.taskEmb.Forward(fp.embIdx)

	// 5. Router input and probabilities.
	width := n.cfg.RouterInputWidth()
	fp.routerIn = make([][]float32, len(reqs))
	for i := range reqs {
		ri := make([]float32, 0, width)
		ri = append(ri, fp.state[i]...)
		ri = append(ri, mgr[i]...)
		ri = append(ri, fp.emb[i]...)
		fp.routerIn[i] = ri
	}
	fp.probs = n.router.ForwardSoftmax(fp.routerIn)

	// 6. Top-2 selection and expert dispatch.
	fp.route = make([]Top2, len(reqs))
	for i := range reqs {
		fp.route[i] = selectTop2(fp.probs[i], n.cfg.TopK)
	}
	for e := 0; e < NumExperts; e++ {
		fp.expertRow[e] = nil
		fp.expertPosIn[e] = make([]int, len(reqs))
		for i := range fp.expertPosIn[e] {
			fp.expertPosIn[e][i] = -1
		}
	}
	for i := range reqs {
		for _, ex := range fp.route[i].Experts {
			fp.expertPosIn[ex][i] = len(fp.expertRow[ex])
			fp.expertRow[ex] = append(fp.expertRow[ex], i)
		}
	}
	for e := 0; e < NumExperts; e++ {
		if len(fp.expertRow[e]) == 0 {
			fp.expertOut[e] = nil
			continue
		}
		subset := make([][]float32, len(fp.expertRow[e]))
		for j, r := range fp.expertRow[e] {
			subset[j] = fp.state[r]
		}
		fp.expertOut[e] = n.experts[e].Forward(subset)
	}

	// 7. Weighted mixture.
	fp.mixed = make([][]float32, len(reqs))
	for i := range reqs {
		t := fp.route[i]
		a := fp.expertOut[t.Experts[0]][fp.expertPosIn[t.Experts[0]][i]]
		b := fp.expertOut[t.Experts[1]][fp.expertPosIn[t.Experts[1]][i]]
		m := make([]float32, n.cfg.StateWidth)
		for j := 0; j < n.cfg.StateWidth; j++ {
			m[j] = t.Weights[0]*a[j] + t.Weights[1]*b[j]
		}
		fp.mixed[i] = m
	}

	// 8. Task heads over per-task subsets.
	for i, r := range reqs {
		if _, ok := n.heads[r.Task]; ok {
			fp.headRows[r.Task] = append(fp.headRows[r.Task], i)
		}
	}
	for t, rows := range fp.headRows {
		subset := make([][]float32, len(rows))
		for j, r := range rows {
			subset[j] = fp.mixed[r]
		}
		fp.headOut[t] = n.heads[t].Forward(subset)
	}
	return fp, nil
}

// BackwardInput carries the gradients and auxiliary router targets for one
// batch step.
type BackwardInput struct {
	// HeadGrad[i] is the gradient at sample i's raw head output, or nil when
	// the sample contributes no head loss (router samples).
	HeadGrad [][]float32
	// RouterTeacher[i] is the teacher expert distribution for router-task
	// samples, or nil.
	RouterTeacher [][]float32
	// SampleWeight[i] scales per-sample losses (label-source weighting).
	SampleWeight []float32
	// BalanceCoef is the router load-balancing coefficient for this batch.
	BalanceCoef float32
}

// backward accumulates parameter gradients for the batch. Returns the router
// balance loss actually applied, for logging.
func (n *Network) backward(fp *ForwardPass, in BackwardInput) float32 {
	B := len(fp.reqs)
	width := n.cfg.StateWidth

	// 1. Head backward: grad wrt mixed states.
	gradMixed := batchZeros(B, width)
	for t, rows := range fp.headRows {
		gradOut := make([][]float32, len(rows))
		any := false
		for j, r := range rows {
			var g []float32
			if r < len(in.HeadGrad) {
				g = in.HeadGrad[r]
			}
			if g == nil {
				gradOut[j] = zeros(len(n.specs[t].Outputs))
				continue
			}
			any = true
			gradOut[j] = g
		}
		if !any {
			continue
		}
		gradM := n.heads[t].Backward(gradOut)
		for j, r := range rows {
			for k := 0; k < width; k++ {
				gradMixed[r][k] += gradM[j][k]
			}
		}
	}

	// 2. Router probability gradients: mixture weights, teacher KL, balance.
	gProbs := make([][]float32, B)
	for i := range gProbs {
		gProbs[i] = zeros(NumExperts)
	}
	for i := range fp.reqs {
		t := fp.route[i]
		a, b := t.Experts[0], t.Experts[1]
		pa, pb := t.Probs[a], t.Probs[b]
		S := pa + pb
		outA := fp.expertOut[a][fp.expertPosIn[a][i]]
		outB := fp.expertOut[b][fp.expertPosIn[b][i]]
		Ga, Gb := float32(0), float32(0)
		for k := 0; k < width; k++ {
			gm := gradMixed[i][k]
			Ga += gm * outA[k]
			Gb += gm * outB[k]
		}
		if S > lossEps {
			gProbs[i][a] += pb * (Ga - Gb) / (S * S)
			gProbs[i][b] += pa * (Gb - Ga) / (S * S)
		}
		var teacher []float32
		if i < len(in.RouterTeacher) {
			teacher = in.RouterTeacher[i]
		}
		if teacher != nil {
			w := float32(1)
			if i < len(in.SampleWeight) {
				w = in.SampleWeight[i]
			}
			for k := 0; k < NumExperts; k++ {
				gProbs[i][k] += -w * teacher[k] / clampFloat(fp.probs[i][k], lossEps, 1)
			}
		}
	}

	// 3. Load-balancing auxiliary loss (switch-style): prevents pathological
	// expert collapse without forcing uniform utilization.
	balanceLoss := float32(0)
	if in.BalanceCoef > 0 {
		count := make([]float32, NumExperts)
		meanP := make([]float32, NumExperts)
		for i := range fp.reqs {
			argmax := 0
			for k := 1; k < NumExperts; k++ {
				if fp.probs[i][k] > fp.probs[i][argmax] {
					argmax = k
				}
			}
			count[argmax]++
			for k := 0; k < NumExperts; k++ {
				meanP[k] += fp.probs[i][k]
			}
		}
		invB := 1.0 / float32(B)
		for k := 0; k < NumExperts; k++ {
			f := count[k] * invB
			p := meanP[k] * invB
			balanceLoss += NumExperts * f * p
		}
		balanceLoss *= in.BalanceCoef
		for i := range fp.reqs {
			for k := 0; k < NumExperts; k++ {
				gProbs[i][k] += in.BalanceCoef * NumExperts * (count[k] * invB) * invB
			}
		}
	}

	// 4. Router backward, split into state / manager / embedding gradients.
	gradRouterIn := n.router.Backward(gProbs)
	gradState := batchZeros(B, width)
	gradMgr := make([][]float32, B)
	gradEmb := make([][]float32, B)
	for i := range fp.reqs {
		ri := gradRouterIn[i]
		for k := 0; k < width; k++ {
			gradState[i][k] += ri[k]
		}
		gradMgr[i] = ri[width : width+n.cfg.ManagerWidth]
		gradEmb[i] = ri[width+n.cfg.ManagerWidth:]
	}

	// 5. Expert backward over the routed subsets (scaled by mixture weight).
	for e := 0; e < NumExperts; e++ {
		rows := fp.expertRow[e]
		if len(rows) == 0 {
			continue
		}
		gradOut := make([][]float32, len(rows))
		for j, r := range rows {
			t := fp.route[r]
			var w float32
			if t.Experts[0] == ExpertID(e) {
				w = t.Weights[0]
			} else {
				w = t.Weights[1]
			}
			g := make([]float32, width)
			for k := 0; k < width; k++ {
				g[k] = gradMixed[r][k] * w
			}
			gradOut[j] = g
		}
		gradIn := n.experts[e].Backward(gradOut)
		for j, r := range rows {
			for k := 0; k < width; k++ {
				gradState[r][k] += gradIn[j][k]
			}
		}
	}

	// 6. Trunk backward. Input-slot gradients are discarded: raw features and
	// normalization statistics are not trainable.
	_ = n.encoder.Backward(gradState)
	_ = n.mgrEnc.Backward(gradMgr)
	_ = n.taskEmb.Backward(gradEmb)
	return balanceLoss
}
