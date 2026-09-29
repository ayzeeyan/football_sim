// Package training implements the offline FootballMoE trainer: streaming
// JSONL dataset loading, AdamW, curriculum phases, validation metrics, router
// statistics, and checkpointing. Nothing in this package is linked into the
// game server's inference path; weights only change via cmd/train.
package training

import (
	"math"
	"strings"

	"football_sim/pkg/footballai"
	"football_sim/pkg/footballai/weights"
)

// AdamWOptions configures the optimizer.
type AdamWOptions struct {
	LearningRate float64
	Beta1        float64
	Beta2        float64
	Eps          float64
	WeightDecay  float64
}

// AdamW is a decoupled-weight-decay Adam optimizer over named parameters.
type AdamW struct {
	opts AdamWOptions
	t    int
	m    map[string][]float32
	v    map[string][]float32
}

// NewAdamW creates an optimizer for the given parameters.
func NewAdamW(params []*footballai.Param, opts AdamWOptions) *AdamW {
	if opts.Beta1 == 0 {
		opts.Beta1 = 0.9
	}
	if opts.Beta2 == 0 {
		opts.Beta2 = 0.999
	}
	if opts.Eps == 0 {
		opts.Eps = 1e-8
	}
	a := &AdamW{opts: opts, m: map[string][]float32{}, v: map[string][]float32{}}
	for _, p := range params {
		a.m[p.Name] = make([]float32, len(p.Data))
		a.v[p.Name] = make([]float32, len(p.Data))
	}
	return a
}

// Step applies one optimizer update using the accumulated gradients.
func (a *AdamW) Step(params []*footballai.Param) error {
	a.t++
	bc1 := math.Pow(a.opts.Beta1, float64(a.t))
	bc2 := math.Pow(a.opts.Beta2, float64(a.t))
	lr := a.opts.LearningRate
	for _, p := range params {
		m := a.m[p.Name]
		v := a.v[p.Name]
		for i := range p.Data {
			g := float64(p.Grad[i])
			if math.IsNaN(g) || math.IsInf(g, 0) {
				return &NaNError{Param: p.Name, Index: i, Value: g}
			}
			// Decoupled weight decay (excludes biases and norms by suffix).
			if a.opts.WeightDecay > 0 && !isBiasLike(p.Name) {
				p.Data[i] = float32(float64(p.Data[i]) * (1 - lr*a.opts.WeightDecay))
			}
			mi := a.opts.Beta1*float64(m[i]) + (1-a.opts.Beta1)*g
			vi := a.opts.Beta2*float64(v[i]) + (1-a.opts.Beta2)*g*g
			m[i] = float32(mi)
			v[i] = float32(vi)
			mHat := mi / (1 - bc1)
			vHat := vi / (1 - bc2)
			p.Data[i] -= float32(lr * mHat / (math.Sqrt(vHat) + a.opts.Eps))
		}
	}
	return nil
}

// NaNError reports a non-finite gradient encountered during optimization.
type NaNError struct {
	Param string
	Index int
	Value float64
}

func (e *NaNError) Error() string {
	return "training: non-finite gradient in " + e.Param + " (gradient training aborted)"
}

func isBiasLike(name string) bool {
	return strings.HasSuffix(name, ".bias") ||
		strings.Contains(name, "_rms") ||
		strings.HasPrefix(name, "embedding.")
}

// ClipGlobalNorm rescales all gradients so their global L2 norm is at most
// maxNorm, and returns the pre-clip norm.
func ClipGlobalNorm(params []*footballai.Param, maxNorm float64) float64 {
	total := 0.0
	for _, p := range params {
		for _, g := range p.Grad {
			total += float64(g) * float64(g)
		}
	}
	norm := math.Sqrt(total)
	if norm <= maxNorm || norm == 0 {
		return norm
	}
	scale := maxNorm / norm
	for _, p := range params {
		for i := range p.Grad {
			p.Grad[i] = float32(float64(p.Grad[i]) * scale)
		}
	}
	return norm
}

// exportMoments returns a copy of the optimizer state for checkpointing.
func (a *AdamW) exportMoments() map[string]weights.Moment {
	out := make(map[string]weights.Moment, len(a.m))
	for name, m := range a.m {
		mc := make([]float32, len(m))
		copy(mc, m)
		vc := make([]float32, len(a.v[name]))
		copy(vc, a.v[name])
		out[name] = weights.Moment{M: mc, V: vc}
	}
	return out
}
