package footballai

import "sync"

// Model is the immutable, loaded FootballMoE used by the simulator. It is
// safe for concurrent reads after Load. It exposes strongly typed prediction
// methods; callers never touch neural-network internals.
type Model struct {
	net           *Network
	cfg           ModelConfig
	modelHash     string
	paramCount    uint64
	featureSchema uint16

	// mu serializes inference: forward passes reuse per-batch caches on the
	// shared network, so concurrent predictions must not interleave.
	mu sync.Mutex
}

// BuildModel wraps an initialized network as a Model (used by training and
// the loader).
func BuildModel(net *Network, hash string) *Model {
	return &Model{
		net:           net,
		cfg:           net.Config(),
		modelHash:     hash,
		paramCount:    net.ParameterCount(),
		featureSchema: net.Config().FeatureSchemaVersion,
	}
}

// Config returns the loaded architecture configuration.
func (m *Model) Config() ModelConfig { return m.cfg }

// ParameterCount returns the number of scalar parameters.
func (m *Model) ParameterCount() uint64 { return m.paramCount }

// Hash returns the model hash (SHA-256 over the weight data section), used
// for career-save compatibility checks.
func (m *Model) Hash() string { return m.modelHash }

// Network exposes the underlying network for the training toolchain only.
// Runtime code should use the Predict methods.
func (m *Model) Network() *Network { return m.net }

// ---------------------------------------------------------------------------
// Typed predictions.
// ---------------------------------------------------------------------------

// MatchPrediction is the expected-goal view of an upcoming fixture. The
// simulation RNG still resolves the actual scoreline.
type MatchPrediction struct {
	HomeXG      float32
	AwayXG      float32
	GoalDiff    float32
	Uncertainty float32
}

// InjuryPrediction is a calibrated injury probability for one player.
type InjuryPrediction struct {
	Probability float32
}

// RotationPrediction scores starting versus resting one player. Existing
// lineup logic still enforces availability, position, and formation rules.
type RotationPrediction struct {
	StartProbability float32
	RestValue        float32
}

// DevelopmentPrediction is the expected season OVR delta.
type DevelopmentPrediction struct {
	OVRDeltaSeason float32
}

// DeclinePrediction is the expected season OVR decline (positive value).
type DeclinePrediction struct {
	OVRDeclineSeason float32
}

// ValuationPrediction is a premium multiplier over the deterministic
// baseline valuation.
type ValuationPrediction struct {
	PremiumMultiplier float32
}

// NegotiationPrediction describes the seller's response to one bid. The
// existing transfer FSM still advances negotiation states.
type NegotiationPrediction struct {
	AcceptProbability      float32
	CounterofferRatioToAsk float32
	WalkAwayProbability    float32
}

// ContractPrediction describes a contract renewal outlook.
type ContractPrediction struct {
	RenewProbability float32
	WageDemandRatio  float32
}

// BoardPatiencePrediction is the manager's sack probability.
type BoardPatiencePrediction struct {
	SackProbability float32
}

// Prediction is a generic single-result prediction with routing
// diagnostics attached.
type Prediction struct {
	Task    TaskType
	Outputs []float32 // activated values, aligned with the task spec
	Routing RoutingInfo
}

// RoutingInfo exposes raw router probabilities and selected Top-2 experts
// for debugging. Routing values are diagnostics, not causal explanations.
type RoutingInfo struct {
	Probs   [NumExperts]float32
	Top2    [2]ExpertID
	Weights [2]float32
}

func (m *Model) predictOne(req Request) (*Prediction, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	fp, err := m.net.forward([]Request{req})
	if err != nil {
		return nil, err
	}
	defer m.net.Reset()
	spec, ok := m.net.specs[req.Task]
	if !ok {
		return nil, errConfig("task %s has no trained head in model version %d", TaskName(req.Task), m.cfg.ModelVersion)
	}
	raw := fp.RawOutputs(0)
	if raw == nil {
		return nil, errConfig("task %s produced no head output", TaskName(req.Task))
	}
	out := make([]float32, len(raw))
	for i, o := range spec.Outputs {
		out[i] = Activate(o.Kind, raw[i])
	}
	r := fp.Routing(0)
	return &Prediction{
		Task:    req.Task,
		Outputs: out,
		Routing: RoutingInfo{Probs: r.Probs, Top2: r.Experts, Weights: r.Weights},
	}, nil
}

// Predict runs any typed request and returns the generic prediction with
// routing diagnostics.
func (m *Model) Predict(req interface {
	Encode() Request
	Task() TaskType
}) (*Prediction, error) {
	return m.predictOne(req.Encode())
}

// PredictBatch runs homogeneous typed requests in one batch (batched expert
// and head evaluation). Requests must share a task.
func (m *Model) PredictBatch(reqs []Request) ([]*Prediction, error) {
	if len(reqs) == 0 {
		return nil, errConfig("empty batch")
	}
	task := reqs[0].Task
	for _, r := range reqs {
		if r.Task != task {
			return nil, errConfig("PredictBatch requires one task; got both %s and %s", TaskName(task), TaskName(r.Task))
		}
	}
	spec, ok := m.net.specs[task]
	if !ok {
		return nil, errConfig("task %s has no trained head", TaskName(task))
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	fp, err := m.net.forward(reqs)
	if err != nil {
		return nil, err
	}
	defer m.net.Reset()
	preds := make([]*Prediction, len(reqs))
	for i := range reqs {
		raw := fp.RawOutputs(i)
		if raw == nil {
			return nil, errConfig("task %s produced no head output for sample %d", TaskName(task), i)
		}
		out := make([]float32, len(raw))
		for j, o := range spec.Outputs {
			out[j] = Activate(o.Kind, raw[j])
		}
		r := fp.Routing(i)
		preds[i] = &Prediction{
			Task:    task,
			Outputs: out,
			Routing: RoutingInfo{Probs: r.Probs, Top2: r.Experts, Weights: r.Weights},
		}
	}
	return preds, nil
}

// PredictMatch returns expected goals for a fixture.
func (m *Model) PredictMatch(req MatchRequest) (MatchPrediction, error) {
	p, err := m.predictOne(req.Encode())
	if err != nil {
		return MatchPrediction{}, err
	}
	return MatchPrediction{
		HomeXG:      p.Outputs[0],
		AwayXG:      p.Outputs[1],
		GoalDiff:    p.Outputs[2],
		Uncertainty: p.Outputs[3],
	}, nil
}

// PredictInjury returns one player's injury probability.
func (m *Model) PredictInjury(req InjuryRequest) (InjuryPrediction, error) {
	p, err := m.predictOne(req.Encode())
	if err != nil {
		return InjuryPrediction{}, err
	}
	return InjuryPrediction{Probability: p.Outputs[0]}, nil
}

// PredictRotation returns start/rest scores for one player.
func (m *Model) PredictRotation(req RotationRequest) (RotationPrediction, error) {
	p, err := m.predictOne(req.Encode())
	if err != nil {
		return RotationPrediction{}, err
	}
	return RotationPrediction{StartProbability: p.Outputs[0], RestValue: p.Outputs[1]}, nil
}

// PredictDevelopment returns the expected season OVR delta.
func (m *Model) PredictDevelopment(req DevelopmentRequest) (DevelopmentPrediction, error) {
	p, err := m.predictOne(req.Encode())
	if err != nil {
		return DevelopmentPrediction{}, err
	}
	return DevelopmentPrediction{OVRDeltaSeason: p.Outputs[0]}, nil
}

// PredictDecline returns the expected season OVR decline.
func (m *Model) PredictDecline(req DeclineRequest) (DeclinePrediction, error) {
	p, err := m.predictOne(req.Encode())
	if err != nil {
		return DeclinePrediction{}, err
	}
	return DeclinePrediction{OVRDeclineSeason: p.Outputs[0]}, nil
}

// PredictValuation returns the market-value premium multiplier over the
// deterministic baseline.
func (m *Model) PredictValuation(req ValuationRequest) (ValuationPrediction, error) {
	p, err := m.predictOne(req.Encode())
	if err != nil {
		return ValuationPrediction{}, err
	}
	return ValuationPrediction{PremiumMultiplier: p.Outputs[0]}, nil
}

// PredictNegotiation returns the response probabilities for one bid.
func (m *Model) PredictNegotiation(req NegotiationRequest) (NegotiationPrediction, error) {
	p, err := m.predictOne(req.Encode())
	if err != nil {
		return NegotiationPrediction{}, err
	}
	return NegotiationPrediction{
		AcceptProbability:      p.Outputs[0],
		CounterofferRatioToAsk: p.Outputs[1],
		WalkAwayProbability:    p.Outputs[2],
	}, nil
}

// PredictContract returns the renewal outlook for one player.
func (m *Model) PredictContract(req ContractRequest) (ContractPrediction, error) {
	p, err := m.predictOne(req.Encode())
	if err != nil {
		return ContractPrediction{}, err
	}
	return ContractPrediction{RenewProbability: p.Outputs[0], WageDemandRatio: p.Outputs[1]}, nil
}

// PredictBoardPatience returns the manager's sack probability.
func (m *Model) PredictBoardPatience(req BoardPatienceRequest) (BoardPatiencePrediction, error) {
	p, err := m.predictOne(req.Encode())
	if err != nil {
		return BoardPatiencePrediction{}, err
	}
	return BoardPatiencePrediction{SackProbability: p.Outputs[0]}, nil
}
