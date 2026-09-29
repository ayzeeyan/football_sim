package footballai

import (
	"fmt"
	"math/rand"
)

// ResidualBlock is a pre-norm feed-forward block:
// out = x + fc2(SiLU(fc1(RMSNorm(x)))).
type ResidualBlock struct {
	name string
	norm *RMSNorm
	fc1  *Linear
	act  *SiLU
	fc2  *Linear

	cacheIn [][]float32
}

// NewResidualBlock creates a block mapping size -> inner -> size.
func NewResidualBlock(name string, size, inner int, rng *rand.Rand) *ResidualBlock {
	return &ResidualBlock{
		name: name,
		norm: NewRMSNorm(name+".rms", size, 1e-6),
		fc1:  NewLinear(name+".fc1", size, inner, rng),
		act:  &SiLU{},
		fc2:  NewLinear(name+".fc2", inner, size, rng),
	}
}

// Forward runs the block over a batch.
func (b *ResidualBlock) Forward(batch [][]float32) [][]float32 {
	b.cacheIn = batch
	h := b.norm.Forward(batch)
	h = b.fc1.Forward(h)
	h = b.act.Forward(h)
	h = b.fc2.Forward(h)
	out := make([][]float32, len(batch))
	for i, x := range batch {
		y := make([]float32, len(x))
		for j := range x {
			y[j] = x[j] + h[i][j]
		}
		out[i] = y
	}
	return out
}

// Backward returns the gradient with respect to the block input.
func (b *ResidualBlock) Backward(gradOut [][]float32) [][]float32 {
	gH := b.fc2.Backward(gradOut)
	gH = b.act.Backward(gH)
	gH = b.fc1.Backward(gH)
	gX := b.norm.Backward(gH)
	out := make([][]float32, len(gradOut))
	for i, g := range gradOut {
		gi := make([]float32, len(g))
		for j := range g {
			gi[j] = g[j] + gX[i][j]
		}
		out[i] = gi
	}
	return out
}

// Params returns the block's parameters.
func (b *ResidualBlock) Params() []*Param {
	return append(b.norm.Params(), append(b.fc1.Params(), b.fc2.Params()...)...)
}

// Reset drops caches recursively.
func (b *ResidualBlock) Reset() {
	b.cacheIn = nil
	b.norm.Reset()
	b.fc1.Reset()
	b.act.Reset()
	b.fc2.Reset()
}

// StateEncoder is the shared trunk: Linear -> width, N pre-norm residual
// blocks, final RMSNorm.
type StateEncoder struct {
	input  *Linear
	blocks []*ResidualBlock
	final  *RMSNorm
}

// NewStateEncoder builds the shared state encoder.
func NewStateEncoder(cfg ModelConfig, rng *rand.Rand) *StateEncoder {
	e := &StateEncoder{
		input: NewLinear("encoder.input", cfg.InputWidth, cfg.StateWidth, rng),
		final: NewRMSNorm("encoder.final_rms", cfg.StateWidth, 1e-6),
	}
	for i := 0; i < cfg.EncoderBlocks; i++ {
		e.blocks = append(e.blocks,
			NewResidualBlock(sprintfName("encoder.block%d", i), cfg.StateWidth, cfg.ExpansionWidth, rng))
	}
	return e
}

// Forward encodes normalized input slots into shared state.
func (e *StateEncoder) Forward(batch [][]float32) [][]float32 {
	h := e.input.Forward(batch)
	for _, b := range e.blocks {
		h = b.Forward(h)
	}
	return e.final.Forward(h)
}

// Backward returns the gradient with respect to the encoder input.
func (e *StateEncoder) Backward(grad [][]float32) [][]float32 {
	g := e.final.Backward(grad)
	for i := len(e.blocks) - 1; i >= 0; i-- {
		g = e.blocks[i].Backward(g)
	}
	return e.input.Backward(g)
}

// Params returns the encoder's parameters.
func (e *StateEncoder) Params() []*Param {
	ps := e.input.Params()
	for _, b := range e.blocks {
		ps = append(ps, b.Params()...)
	}
	return append(ps, e.final.Params()...)
}

// Reset drops caches recursively.
func (e *StateEncoder) Reset() {
	e.input.Reset()
	for _, b := range e.blocks {
		b.Reset()
	}
	e.final.Reset()
}

// ManagerEncoder projects manager characteristics into a dense
// representation. Managers are defined by traits, never by identity.
type ManagerEncoder struct {
	fc1 *Linear
	act *SiLU
	fc2 *Linear
}

// NewManagerEncoder builds the characteristic encoder.
func NewManagerEncoder(cfg ModelConfig, rng *rand.Rand) *ManagerEncoder {
	return &ManagerEncoder{
		fc1: NewLinear("manager.fc1", managerWidthFeats, cfg.ManagerHidden, rng),
		act: &SiLU{},
		fc2: NewLinear("manager.fc2", cfg.ManagerHidden, cfg.ManagerWidth, rng),
	}
}

// Forward encodes normalized manager slots.
func (m *ManagerEncoder) Forward(batch [][]float32) [][]float32 {
	return m.fc2.Forward(m.act.Forward(m.fc1.Forward(batch)))
}

// Backward returns the gradient with respect to the manager slots.
func (m *ManagerEncoder) Backward(grad [][]float32) [][]float32 {
	return m.fc1.Backward(m.act.Backward(m.fc2.Backward(grad)))
}

// Params returns the encoder's parameters.
func (m *ManagerEncoder) Params() []*Param {
	return append(m.fc1.Params(), m.fc2.Params()...)
}

// Reset drops caches recursively.
func (m *ManagerEncoder) Reset() {
	m.fc1.Reset()
	m.act.Reset()
	m.fc2.Reset()
}

// sprintfName keeps block naming terse.
func sprintfName(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}
