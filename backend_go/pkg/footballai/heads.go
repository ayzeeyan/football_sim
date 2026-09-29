package footballai

// OutputKind selects the activation applied to a head output at inference
// (and the corresponding target transform used during training).
type OutputKind uint8

const (
	// OutReg is an identity output (goals, deltas, ratios).
	OutReg OutputKind = iota
	// OutProb is a probability; the head emits a raw logit and inference
	// applies a sigmoid. The simulation RNG, never the network, resolves
	// occurrence.
	OutProb
	// OutLogMult is a positive multiplier modeled in log space (valuation
	// premium, wage-demand ratio); inference applies exp().
	OutLogMult
)

// LossKind selects the training loss for one output.
type LossKind uint8

const (
	LossHuber LossKind = iota
	LossMSE
	LossBCE
)

// OutputSpec describes one head output dimension.
type OutputSpec struct {
	Name string
	Kind OutputKind
	Loss LossKind
	// HuberDelta is the transition point of the Huber loss (regression only).
	HuberDelta float32
}

// TaskSpec defines a task's head outputs and their losses.
type TaskSpec struct {
	Task    TaskType
	Outputs []OutputSpec
}

// OutputWidth returns the head output vector width for the task.
func (s TaskSpec) OutputWidth() int { return len(s.Outputs) }

// OutputIndex returns the position of the named output, or -1.
func (s TaskSpec) OutputIndex(name string) int {
	for i, o := range s.Outputs {
		if o.Name == name {
			return i
		}
	}
	return -1
}

// TaskSpecs returns the trained task-head definitions for model version 1.
func TaskSpecs() map[TaskType]TaskSpec {
	specs := map[TaskType]TaskSpec{
		TaskMatchPrediction: {
			Task: TaskMatchPrediction,
			Outputs: []OutputSpec{
				{Name: "home_xg", Kind: OutReg, Loss: LossHuber, HuberDelta: 0.5},
				{Name: "away_xg", Kind: OutReg, Loss: LossHuber, HuberDelta: 0.5},
				{Name: "goal_diff", Kind: OutReg, Loss: LossHuber, HuberDelta: 0.75},
				{Name: "uncertainty", Kind: OutReg, Loss: LossMSE},
			},
		},
		TaskInjuryRisk: {
			Task: TaskInjuryRisk,
			Outputs: []OutputSpec{
				{Name: "injury_probability", Kind: OutProb, Loss: LossBCE},
			},
		},
		TaskRotation: {
			Task: TaskRotation,
			Outputs: []OutputSpec{
				{Name: "start_probability", Kind: OutProb, Loss: LossBCE},
				{Name: "rest_value", Kind: OutReg, Loss: LossHuber, HuberDelta: 0.5},
			},
		},
		TaskDevelopment: {
			Task: TaskDevelopment,
			Outputs: []OutputSpec{
				{Name: "ovr_delta_season", Kind: OutReg, Loss: LossHuber, HuberDelta: 0.5},
			},
		},
		TaskDecline: {
			Task: TaskDecline,
			Outputs: []OutputSpec{
				{Name: "ovr_decline_season", Kind: OutReg, Loss: LossHuber, HuberDelta: 0.5},
			},
		},
		TaskValuation: {
			Task: TaskValuation,
			Outputs: []OutputSpec{
				// Predicted in log space: the head learns a premium over the
				// deterministic baseline valuation rather than the money scale.
				{Name: "premium_multiplier", Kind: OutLogMult, Loss: LossHuber, HuberDelta: 0.25},
			},
		},
		TaskNegotiation: {
			Task: TaskNegotiation,
			Outputs: []OutputSpec{
				{Name: "accept_probability", Kind: OutProb, Loss: LossBCE},
				{Name: "counteroffer_ratio_to_ask", Kind: OutReg, Loss: LossHuber, HuberDelta: 0.1},
				{Name: "walk_away_probability", Kind: OutProb, Loss: LossBCE},
			},
		},
		TaskContract: {
			Task: TaskContract,
			Outputs: []OutputSpec{
				{Name: "renew_probability", Kind: OutProb, Loss: LossBCE},
				// Modeled as log(wage demand / baseline wage).
				{Name: "wage_demand_ratio", Kind: OutLogMult, Loss: LossHuber, HuberDelta: 0.25},
			},
		},
		TaskBoardPatience: {
			Task: TaskBoardPatience,
			Outputs: []OutputSpec{
				{Name: "sack_probability", Kind: OutProb, Loss: LossBCE},
			},
		},
	}
	return specs
}

// HeadedTasks lists tasks that have trained heads in model version 1.
func HeadedTasks() []TaskType {
	return []TaskType{
		TaskMatchPrediction,
		TaskInjuryRisk,
		TaskRotation,
		TaskDevelopment,
		TaskDecline,
		TaskValuation,
		TaskNegotiation,
		TaskContract,
		TaskBoardPatience,
	}
}
