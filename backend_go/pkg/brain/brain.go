package brain

import (
	"math"
)

// The manager brain: a tiny online-learned linear model.
//
// Lifecycle: a small pre-trained base (Base(), fitted offline by
// cmd/pretrain and committed as the shipped artifact) plus continuous
// post-training — every finished matchweek the model takes one SGD step
// per match, so the world's brain keeps learning for the life of the
// career. The whole model is a fixed-size array of eight weights plus a
// bias and two counters: roughly one hundred bytes, no growing state.
//
// Determinism: training is a pure function of the match history. Rows are
// presented in stable fixture order with fixed hyperparameters and no
// randomness, so identical seeds produce byte-identical weights.

// NumFeatures is the fixed width of the model's input.
const NumFeatures = 8

// Feature indices. The feature vector is built by MatchFeatures (features.go)
// and every consumer must use the same order.
const (
	FeatureHomeAdvantage = 0
	FeatureRatingDiff    = 1
	FeatureFormDiff      = 2
	FeatureHomeBeats     = 3
	FeatureAwayBeats     = 4
	FeaturePressDiff     = 5
	FeatureRiskDiff      = 6
	FeatureMoraleDiff    = 7
)

// Model is the learned brain. Fixed-size by design: no slices, no maps,
// no hidden allocations.
type Model struct {
	Weights [NumFeatures]float64 `json:"weights"`
	Bias    float64              `json:"bias"`
	Samples int                  `json:"samples"`
	LossEMA float64              `json:"loss_ema"`
}

// learningRate is the fixed SGD step. Small enough that the brain evolves
// over weeks, not within one.
const learningRate = 0.02

// l2Decay is the fixed weight decay applied per step: the brain slowly
// forgets, so recent football matters more than ancient football.
const l2Decay = 0.0001

// lossEMADecay smooths the reported training loss.
const lossEMADecay = 0.01

// targetClamp bounds the regression target: a 4-goal win is not
// structurally different from a 7-goal win.
const targetClamp = 3.0

// New returns a fresh brain starting from the pre-trained base.
func New() *Model {
	m := Base()
	return m
}

// Predict returns the model's expected goal difference for a feature row
// (home perspective). This is the brain thinking: no tables, no constants —
// its own weights.
func (m *Model) Predict(features [NumFeatures]float64) float64 {
	out := m.Bias
	for i := 0; i < NumFeatures; i++ {
		out += m.Weights[i] * features[i]
	}
	return out
}

// Train takes one deterministic SGD step toward the observed outcome and
// returns the squared error of the pre-update prediction.
func (m *Model) Train(features [NumFeatures]float64, target float64) float64 {
	if target > targetClamp {
		target = targetClamp
	}
	if target < -targetClamp {
		target = -targetClamp
	}
	prediction := m.Predict(features)
	err := target - prediction
	loss := err * err
	for i := 0; i < NumFeatures; i++ {
		// Decay first, then step: the brain slowly forgets old football.
		m.Weights[i] -= l2Decay * m.Weights[i]
		m.Weights[i] += learningRate * err * features[i]
	}
	m.Bias += learningRate * err
	m.Samples++
	if m.Samples == 1 {
		m.LossEMA = loss
	} else {
		m.LossEMA += lossEMADecay * (loss - m.LossEMA)
	}
	return loss
}

// PredictStyleEdge returns the style-attributable component of the
// prediction only: the two style-matchup weights, without the home
// advantage bias. This is what the engine consumes as the tactical
// edge for a philosophical pairing.
func (m *Model) PredictStyleEdge(features [NumFeatures]float64) float64 {
	return m.Weights[FeatureHomeBeats]*features[FeatureHomeBeats] + m.Weights[FeatureAwayBeats]*features[FeatureAwayBeats]
}

// Clone returns an independent copy.
func (m *Model) Clone() *Model {
	cp := *m
	return &cp
}

// Valid reports whether the model's state is finite and sane.
func (m *Model) Valid() bool {
	if math.IsNaN(m.Bias) || math.IsInf(m.Bias, 0) || math.IsNaN(m.LossEMA) || math.IsInf(m.LossEMA, 0) {
		return false
	}
	if m.Samples < 0 {
		return false
	}
	for _, w := range m.Weights {
		if math.IsNaN(w) || math.IsInf(w, 0) || w > 1e3 || w < -1e3 {
			return false
		}
	}
	return true
}

// FeatureLabels names each weight for the observational wire surface.
func FeatureLabels() [NumFeatures]string {
	return [NumFeatures]string{
		"home_advantage",
		"rating_diff",
		"form_diff",
		"home_style_beats",
		"away_style_beats",
		"press_intensity_diff",
		"risk_appetite_diff",
		"morale_diff",
	}
}
