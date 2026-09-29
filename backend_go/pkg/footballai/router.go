package footballai

import "math/rand"

// Router scores each expert from [state | manager | task embedding] and
// selects the top-K experts. Selection is hard; the renormalized top-K
// probabilities remain differentiable so task losses shape routing.
type Router struct {
	fc1  *Linear
	act  *SiLU
	fc2  *Linear
	soft *Softmax
}

// NewRouter builds the router trunk.
func NewRouter(cfg ModelConfig, rng *rand.Rand) *Router {
	return &Router{
		fc1:  NewLinear("router.fc1", cfg.RouterInputWidth(), cfg.RouterHidden, rng),
		act:  &SiLU{},
		fc2:  NewLinear("router.fc2", cfg.RouterHidden, cfg.NumExperts, rng),
		soft: &Softmax{},
	}
}

// Forward computes expert logits for a batch.
func (r *Router) Forward(batch [][]float32) [][]float32 {
	return r.fc2.Forward(r.act.Forward(r.fc1.Forward(batch)))
}

// Backward returns the gradient with respect to the router input, given
// gradients with respect to the expert probabilities.
func (r *Router) Backward(gradProbs [][]float32) [][]float32 {
	gLogits := r.soft.Backward(gradProbs)
	return r.fc1.Backward(r.act.Backward(r.fc2.Backward(gLogits)))
}

// ForwardSoftmax runs the trunk and the softmax, caching probabilities for
// backward.
func (r *Router) ForwardSoftmax(batch [][]float32) [][]float32 {
	return r.soft.Forward(r.Forward(batch))
}

// Params returns the router's parameters.
func (r *Router) Params() []*Param { return append(r.fc1.Params(), r.fc2.Params()...) }

// Reset drops caches recursively.
func (r *Router) Reset() {
	r.fc1.Reset()
	r.act.Reset()
	r.fc2.Reset()
	r.soft.Reset()
}

// Top2 holds one request's routing decision.
type Top2 struct {
	// Experts in descending-probability order.
	Experts [2]ExpertID
	// Renormalized mixture weights aligned with Experts.
	Weights [2]float32
	// Raw softmax probabilities over all experts.
	Probs [NumExperts]float32
}

// selectTop2 computes the routing decision for one probability row.
func selectTop2(probs []float32, k int) Top2 {
	var t Top2
	copy(t.Probs[:], probs[:NumExperts])
	// Stable argpartition over 4 experts.
	first, second := 0, 1
	if probs[1] > probs[0] {
		first, second = 1, 0
	}
	for e := 2; e < NumExperts; e++ {
		if probs[e] > probs[second] {
			if probs[e] > probs[first] {
				second = first
				first = e
			} else {
				second = e
			}
		}
	}
	t.Experts = [2]ExpertID{ExpertID(first), ExpertID(second)}
	sum := probs[first] + probs[second]
	if sum <= 0 {
		t.Weights = [2]float32{0.5, 0.5}
	} else {
		t.Weights = [2]float32{probs[first] / sum, probs[second] / sum}
	}
	return t
}

// Expert is one semantic expert network: two pre-norm residual blocks with
// configurable inner widths.
type Expert struct {
	id     ExpertID
	name   string
	block1 *ResidualBlock
	block2 *ResidualBlock
}

// NewExpert builds an expert for the given domain.
func NewExpert(cfg ModelConfig, id ExpertID, rng *rand.Rand) *Expert {
	name := "expert." + expertDirName(id)
	return &Expert{
		id:     id,
		name:   name,
		block1: NewResidualBlock(name+".block0", cfg.StateWidth, cfg.ExpertWidthA, rng),
		block2: NewResidualBlock(name+".block1", cfg.StateWidth, cfg.ExpertWidthB, rng),
	}
}

// Forward runs the expert over the routed subset of shared states.
func (e *Expert) Forward(batch [][]float32) [][]float32 {
	return e.block2.Forward(e.block1.Forward(batch))
}

// Backward returns the gradient with respect to the expert input.
func (e *Expert) Backward(grad [][]float32) [][]float32 {
	return e.block1.Backward(e.block2.Backward(grad))
}

// Params returns the expert's parameters.
func (e *Expert) Params() []*Param {
	return append(e.block1.Params(), e.block2.Params()...)
}

// Reset drops caches recursively.
func (e *Expert) Reset() {
	e.block1.Reset()
	e.block2.Reset()
}

// expertDirName maps an expert to its stable tensor-name component.
func expertDirName(id ExpertID) string {
	switch id {
	case ExpertMatch:
		return "match"
	case ExpertPlayer:
		return "player"
	case ExpertEconomy:
		return "economy"
	case ExpertClub:
		return "club"
	}
	return "unknown"
}

// TaskHead is a small two-layer head mapping the mixed expert state to one
// task's raw output vector.
type TaskHead struct {
	task TaskType
	fc1  *Linear
	act  *SiLU
	fc2  *Linear
}

// NewTaskHead builds a head with the configured hidden width.
func NewTaskHead(cfg ModelConfig, task TaskType, outWidth int, rng *rand.Rand) *TaskHead {
	name := "head." + headDirName(task)
	return &TaskHead{
		task: task,
		fc1:  NewLinear(name+".fc1", cfg.StateWidth, cfg.HeadHidden, rng),
		act:  &SiLU{},
		fc2:  NewLinear(name+".fc2", cfg.HeadHidden, outWidth, rng),
	}
}

// Forward maps mixed states to raw outputs.
func (h *TaskHead) Forward(batch [][]float32) [][]float32 {
	return h.fc2.Forward(h.act.Forward(h.fc1.Forward(batch)))
}

// Backward returns the gradient with respect to the head input.
func (h *TaskHead) Backward(grad [][]float32) [][]float32 {
	return h.fc1.Backward(h.act.Backward(h.fc2.Backward(grad)))
}

// Params returns the head's parameters.
func (h *TaskHead) Params() []*Param { return append(h.fc1.Params(), h.fc2.Params()...) }

// Reset drops caches recursively.
func (h *TaskHead) Reset() {
	h.fc1.Reset()
	h.act.Reset()
	h.fc2.Reset()
}

// headDirName maps a task to its stable tensor-name component.
func headDirName(t TaskType) string {
	switch t {
	case TaskMatchPrediction:
		return "match"
	case TaskLineupSelection:
		return "lineup"
	case TaskRotation:
		return "rotation"
	case TaskInjuryRisk:
		return "injury"
	case TaskDevelopment:
		return "development"
	case TaskDecline:
		return "decline"
	case TaskValuation:
		return "valuation"
	case TaskTransferBid:
		return "transfer_bid"
	case TaskNegotiation:
		return "negotiation"
	case TaskContract:
		return "contract"
	case TaskBoardPatience:
		return "board_patience"
	case TaskManagerHiring:
		return "manager_hiring"
	case TaskSetPiece:
		return "set_piece"
	case TaskCrowd:
		return "crowd"
	}
	return "unknown"
}
