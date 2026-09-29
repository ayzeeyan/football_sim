package footballai

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
)

// AIConfig gates every FootballMoE feature. All flags default to false: the
// existing deterministic simulation is unchanged until a model has been
// validated against it, and careers stay reproducible.
type AIConfig struct {
	// ModelPath points at the immutable .fmoe deployment model. Empty
	// disables the brain entirely.
	ModelPath string

	// Feature flags for A/B rollout against existing heuristics.
	UseMatchModel     bool
	UseInjuryModel    bool
	UseRotationModel  bool
	UseValuationModel bool

	// RecordOutcomes enables the observational training-data recorder.
	// Recording never alters simulation results and never trains anything;
	// recorded examples feed future offline training runs only.
	RecordOutcomes bool
	RecordPath     string
}

// AIModelInfo is persisted in career saves so a career always remembers
// which model version produced its world. Loading a different model into an
// old career must be an explicit, logged decision — never silent.
type AIModelInfo struct {
	FormatVersion uint16 `json:"format_version"`
	ModelVersion  uint32 `json:"model_version"`
	ModelHash     string `json:"model_hash"`
}

// Matches returns true when the saved info agrees with the given live info.
func (a *AIModelInfo) Matches(other AIModelInfo) bool {
	if a == nil {
		return false
	}
	return *a == other
}

// Brain is the runtime wrapper the game server talks to. It owns the loaded
// immutable model and the optional outcome recorder. A nil Brain is valid
// and disabled; every method is nil-safe.
type Brain struct {
	cfg      AIConfig
	model    *Model
	recorder *Recorder
	info     AIModelInfo
}

// NewBrain loads the configured model and prepares the recorder. Missing or
// incompatible models disable the brain with an error rather than degrading
// the simulation silently.
func NewBrain(cfg AIConfig, logger *log.Logger) (*Brain, error) {
	b := &Brain{cfg: cfg}
	if cfg.ModelPath == "" {
		return b, nil
	}
	if loadModelFile == nil {
		return nil, fmt.Errorf("footballai: no .fmoe loader registered (import pkg/footballai/weights)")
	}
	model, err := loadModelFile(cfg.ModelPath)
	if err != nil {
		return nil, fmt.Errorf("footballai: load %s: %w", cfg.ModelPath, err)
	}
	b.model = model
	b.info = AIModelInfo{
		FormatVersion: model.Config().FeatureSchemaVersion, // format/schema compatibility marker
		ModelVersion:  model.Config().ModelVersion,
		ModelHash:     model.Hash(),
	}
	if logger != nil {
		logger.Printf("[FootballMoE] Loaded %s: model v%d, %d parameters, hash %s",
			cfg.ModelPath, b.info.ModelVersion, model.ParameterCount(), b.info.ModelHash)
	}
	if cfg.RecordOutcomes {
		rec, err := NewRecorder(cfg.RecordPath)
		if err != nil {
			return nil, err
		}
		b.recorder = rec
		if logger != nil {
			logger.Printf("[FootballMoE] Outcome recorder writing to %s", cfg.RecordPath)
		}
	}
	return b, nil
}

// ModelLoader loads a .fmoe file into a Model. The weights package (which
// imports this one) registers itself at init time; binaries that never import
// the weights package simply have no loader and the brain stays disabled.
type ModelLoader func(path string) (*Model, error)

var (
	loadModelFile     ModelLoader
	modelLoaderNotice sync.Once
)

// RegisterModelLoader installs the .fmoe loader. Calling it twice replaces
// the loader; the weights package calls it once from init.
func RegisterModelLoader(l ModelLoader) {
	modelLoaderNotice.Do(func() {})
	loadModelFile = l
}

// NewBrainWithModel builds a brain around an already-loaded model. The
// identity comes from the model itself; the recorder is optional. Used by
// embedders and tests that hold a Model without a .fmoe path.
func NewBrainWithModel(model *Model, cfg AIConfig, recorder *Recorder) *Brain {
	b := &Brain{cfg: cfg, model: model, recorder: recorder}
	if model != nil {
		b.info = AIModelInfo{
			FormatVersion: model.Config().FeatureSchemaVersion,
			ModelVersion:  model.Config().ModelVersion,
			ModelHash:     model.Hash(),
		}
	}
	return b
}

// Enabled reports whether a model is loaded.
func (b *Brain) Enabled() bool { return b != nil && b.model != nil }

// Model returns the loaded model, or nil when disabled.
func (b *Brain) Model() *Model {
	if b == nil {
		return nil
	}
	return b.model
}

// Info returns the persisted identity of the loaded model.
func (b *Brain) Info() AIModelInfo {
	if b == nil {
		return AIModelInfo{}
	}
	return b.info
}

// Recorder returns the outcome recorder (nil-safe).
func (b *Brain) Recorder() *Recorder {
	if b == nil {
		return nil
	}
	return b.recorder
}

// DisableFeatures turns off every gameplay-affecting feature flag while
// keeping the model loaded for inspection and recording. Used when a career
// save pins a different model than the one on disk.
func (b *Brain) DisableFeatures() {
	if b == nil {
		return
	}
	b.cfg.UseMatchModel = false
	b.cfg.UseInjuryModel = false
	b.cfg.UseRotationModel = false
	b.cfg.UseValuationModel = false
}

// Config returns the active configuration.
func (b *Brain) Config() AIConfig {
	if b == nil {
		return AIConfig{}
	}
	return b.cfg
}

// Close flushes and closes the recorder.
func (b *Brain) Close() {
	if b == nil {
		return
	}
	if b.recorder != nil {
		_ = b.recorder.Close()
	}
}

// ---------------------------------------------------------------------------
// Outcome recorder. Runtime records observations for future offline training;
// it never updates weights and never influences any simulation result.
// ---------------------------------------------------------------------------

// Recorder captures (state, later-outcome) pairs. Targets often become known
// long after the state is observed — e.g. a development delta is only known
// at season end — so observations stay pending until their outcome arrives.
type Recorder struct {
	mu      sync.Mutex
	enabled bool
	file    *os.File
	w       *bufio.Writer
	pending map[uint64]pendingObs
	nextID  uint64
	simID   uint64
	season  int
	week    int
}

type pendingObs struct {
	task   TaskType
	slots  []float32
	season int
	week   int
}

// NewRecorder opens (or appends to) a JSONL outcome file.
func NewRecorder(path string) (*Recorder, error) {
	if path == "" {
		return &Recorder{}, nil
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("footballai: recorder: %w", err)
	}
	return &Recorder{
		enabled: true,
		file:    f,
		w:       bufio.NewWriter(f),
		pending: map[uint64]pendingObs{},
	}, nil
}

// SetClock records the simulation position stamped onto every observation.
func (r *Recorder) SetClock(simID uint64, season, week int) {
	if r == nil || !r.enabled {
		return
	}
	r.mu.Lock()
	r.simID = simID
	r.season = season
	r.week = week
	r.mu.Unlock()
}

// Observe records a pre-decision state and returns its observation ID, or 0
// when recording is disabled. The returned ID completes the observation later.
func (r *Recorder) Observe(task TaskType, req Request) uint64 {
	if r == nil || !r.enabled {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID++
	id := r.nextID
	slots := make([]float32, len(req.Slots))
	copy(slots, req.Slots)
	r.pending[id] = pendingObs{task: task, slots: slots, season: r.season, week: r.week}
	return id
}

// Complete attaches an outcome to a pending observation and writes the
// finished example. Unknown IDs are ignored so late completions after a
// restart are harmless.
func (r *Recorder) Complete(id uint64, target map[string]float64) {
	if r == nil || !r.enabled || id == 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	obs, ok := r.pending[id]
	if !ok {
		return
	}
	delete(r.pending, id)
	row := recorderRow{
		SchemaVersion: 1,
		LabelSource:   LabelSimulation,
		Task:          TaskName(obs.task),
		SimID:         r.simID,
		Season:        obs.season,
		Week:          obs.week,
		Features:      map[string]float64{},
		Target:        target,
	}
	names := SlotNames()
	for i, v := range obs.slots {
		if v != 0 {
			row.Features[names[i]] = float64(v)
		}
	}
	data, err := json.Marshal(row)
	if err != nil {
		return
	}
	_, _ = r.w.Write(append(data, '\n'))
	// Flush immediately: outcomes are low-rate events and recorded data must
	// survive an abrupt shutdown, not just a graceful one.
	_ = r.w.Flush()
}

// recorderRow is the on-disk recording schema. It intentionally mirrors the
// bootstrap JSONL shape so future training runs can mix recorded simulator
// outcomes with the bootstrap pack.
type recorderRow struct {
	SchemaVersion int                `json:"schema_version"`
	LabelSource   LabelSource        `json:"label_source"`
	Task          string             `json:"task"`
	SimID         uint64             `json:"sim_id,omitempty"`
	Season        int                `json:"season,omitempty"`
	Week          int                `json:"week,omitempty"`
	Features      map[string]float64 `json:"features"`
	Target        map[string]float64 `json:"target"`
}

// Close flushes and closes the recorder.
func (r *Recorder) Close() error {
	if r == nil || !r.enabled {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.w.Flush(); err != nil {
		return err
	}
	return r.file.Close()
}
