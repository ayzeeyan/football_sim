package footballai

// ModelConfig fully describes a FootballMoE network's architecture. It is
// stored inside the .fmoe file so incompatible weights are rejected at load
// time instead of silently misbehaving.
type ModelConfig struct {
	// Hidden width of the shared state encoder.
	StateWidth int `json:"state_width"`
	// Expansion width of the encoder residual blocks.
	ExpansionWidth int `json:"expansion_width"`
	// Number of encoder residual blocks.
	EncoderBlocks int `json:"encoder_blocks"`

	// Expert hidden widths: each expert has two residual sub-blocks with
	// these inner widths (see expert.go).
	ExpertWidthA int `json:"expert_width_a"`
	ExpertWidthB int `json:"expert_width_b"`

	// Manager characteristic encoder output width.
	ManagerWidth int `json:"manager_width"`
	// Manager encoder hidden width.
	ManagerHidden int `json:"manager_hidden"`
	// Learned task embedding width.
	TaskEmbedWidth int `json:"task_embed_width"`

	// Router hidden width.
	RouterHidden int `json:"router_hidden"`
	// Number of experts selected per request (top-K routing).
	TopK int `json:"top_k"`
	// Number of experts in the pool.
	NumExperts int `json:"num_experts"`

	// Task head hidden width.
	HeadHidden int `json:"head_hidden"`

	// Input slot layout width (feature schema).
	InputWidth int `json:"input_width"`
	// Version of the feature slot layout the weights were trained against.
	FeatureSchemaVersion uint16 `json:"feature_schema_version"`
	// Version of the normalization metadata schema.
	NormSchemaVersion uint16 `json:"norm_schema_version"`
	// Model version, incremented when behavior-affecting training changes.
	ModelVersion uint32 `json:"model_version"`

	// Names of the slot groups in order; informational but validated.
	SlotGroups []SlotGroup `json:"slot_groups,omitempty"`
}

// SlotGroup describes one contiguous block of input slots.
type SlotGroup struct {
	Name  string `json:"name"`
	Start int    `json:"start"`
	Width int    `json:"width"`
}

// FeatureSchemaVersion is the current feature slot layout version. Bump when
// slot order, meaning, or transforms change.
const FeatureSchemaVersion uint16 = 1

// NormSchemaVersion is the current normalization metadata version.
const NormSchemaVersion uint16 = 1

// DefaultConfig returns the recommended FootballMoE architecture:
// ~360k FP32 parameters, CPU-friendly, deterministic.
func DefaultConfig() ModelConfig {
	c := ModelConfig{
		StateWidth:           96,
		ExpansionWidth:       160,
		EncoderBlocks:        3,
		ExpertWidthA:         144,
		ExpertWidthB:         128,
		ManagerWidth:         24,
		ManagerHidden:        32,
		TaskEmbedWidth:       16,
		RouterHidden:         48,
		TopK:                 2,
		NumExperts:           NumExperts,
		HeadHidden:           48,
		InputWidth:           InputWidth,
		FeatureSchemaVersion: FeatureSchemaVersion,
		NormSchemaVersion:    NormSchemaVersion,
		ModelVersion:         1,
	}
	c.SlotGroups = SlotLayout()
	return c
}

// Validate checks internal consistency of a config, whether from code or from
// a model file.
func (c ModelConfig) Validate() error {
	if c.StateWidth <= 0 || c.ExpansionWidth <= 0 || c.EncoderBlocks <= 0 {
		return errConfig("state encoder dimensions must be positive")
	}
	if c.ExpertWidthA <= 0 || c.ExpertWidthB <= 0 {
		return errConfig("expert widths must be positive")
	}
	if c.ManagerWidth <= 0 || c.ManagerHidden <= 0 {
		return errConfig("manager encoder dimensions must be positive")
	}
	if c.TaskEmbedWidth <= 0 {
		return errConfig("task embedding width must be positive")
	}
	if c.RouterHidden <= 0 {
		return errConfig("router hidden width must be positive")
	}
	if c.TopK < 1 || c.TopK > c.NumExperts {
		return errConfig("top-k routing must select between 1 and num_experts")
	}
	if c.TopK != 2 {
		// The sparse routing kernel implements exactly top-2 renormalized
		// mixture; refuse other values rather than silently mis-mixing.
		return errConfig("top-k routing in this model format supports exactly 2 experts")
	}
	if c.NumExperts != NumExperts {
		return errConfig("expected %d experts, got %d", NumExperts, c.NumExperts)
	}
	if c.HeadHidden <= 0 {
		return errConfig("head hidden width must be positive")
	}
	if c.InputWidth != InputWidth {
		return errConfig("feature schema mismatch: weights expect %d input slots, runtime provides %d (schema v%d)", c.InputWidth, InputWidth, c.FeatureSchemaVersion)
	}
	if c.FeatureSchemaVersion != FeatureSchemaVersion {
		return errConfig("feature schema version mismatch: weights expect v%d, runtime uses v%d", c.FeatureSchemaVersion, FeatureSchemaVersion)
	}
	if c.NormSchemaVersion != NormSchemaVersion {
		return errConfig("normalization schema version mismatch: weights expect v%d, runtime uses v%d", c.NormSchemaVersion, NormSchemaVersion)
	}
	return nil
}

// RouterInputWidth is the width of the router input vector:
// state + manager representation + task embedding.
func (c ModelConfig) RouterInputWidth() int {
	return c.StateWidth + c.ManagerWidth + c.TaskEmbedWidth
}
