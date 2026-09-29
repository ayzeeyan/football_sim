package footballai

import (
	"math"
	"math/rand"
)

// Linear is a fully connected layer: y = Wx + b, with W stored row-major
// (out x in). Supports batched forward/backward; gradients accumulate across
// backward calls until ZeroGrad.
type Linear struct {
	name   string
	in     int
	out    int
	w      []float32
	b      []float32
	gw     []float32
	gb     []float32
	lastIn [][]float32

	pw *Param
	pb *Param
}

// NewLinear creates a layer with Xavier-initialized weights and zero bias.
func NewLinear(name string, in, out int, rng *rand.Rand) *Linear {
	l := &Linear{name: name, in: in, out: out}
	l.w = zeros(in * out)
	l.b = zeros(out)
	xavierInit(l.w, in, out, rng)
	l.gw = zeros(in * out)
	l.gb = zeros(out)
	l.pw = &Param{Name: name + ".weight", Data: l.w, Grad: l.gw}
	l.pb = &Param{Name: name + ".bias", Data: l.b, Grad: l.gb}
	return l
}

// In returns the input width.
func (l *Linear) In() int { return l.in }

// Out returns the output width.
func (l *Linear) Out() int { return l.out }

// Forward computes the batched affine transform, caching inputs.
func (l *Linear) Forward(batch [][]float32) [][]float32 {
	out := make([][]float32, len(batch))
	l.lastIn = batch
	for i, x := range batch {
		y := zeros(l.out)
		for o := 0; o < l.out; o++ {
			row := l.w[o*l.in : (o+1)*l.in]
			sum := l.b[o]
			for j, xv := range x {
				sum += row[j] * xv
			}
			y[o] = sum
		}
		out[i] = y
	}
	return out
}

// Backward accumulates weight gradients and returns the input gradient.
func (l *Linear) Backward(gradOut [][]float32) [][]float32 {
	gradIn := make([][]float32, len(l.lastIn))
	for i, x := range l.lastIn {
		g := gradOut[i]
		gi := zeros(l.in)
		for o := 0; o < l.out; o++ {
			go_ := g[o]
			if go_ == 0 {
				continue
			}
			row := l.gw[o*l.in : (o+1)*l.in]
			xr := l.w[o*l.in : (o+1)*l.in]
			for j := 0; j < l.in; j++ {
				row[j] += go_ * x[j]
				gi[j] += go_ * xr[j]
			}
			l.gb[o] += go_
		}
		gradIn[i] = gi
	}
	return gradIn
}

// Params returns the layer's parameters.
func (l *Linear) Params() []*Param { return []*Param{l.pw, l.pb} }

// Reset drops the input cache.
func (l *Linear) Reset() { l.lastIn = nil }

// SiLU is a batched SiLU activation with cached sigmoid values.
type SiLU struct {
	cache [][]float32 // sigmoid(x) per element
}

// Forward applies x * sigmoid(x) elementwise.
func (s *SiLU) Forward(batch [][]float32) [][]float32 {
	out := make([][]float32, len(batch))
	s.cache = make([][]float32, len(batch))
	for i, x := range batch {
		y := zeros(len(x))
		c := make([]float32, len(x))
		copy(c, x)
		for j, v := range x {
			y[j] = v * sigmoid(v)
		}
		out[i] = y
		s.cache[i] = c
	}
	return out
}

// Backward returns the input gradient given the output gradient.
// d silu/dx = sigmoid(x) + x * sigmoid(x) * (1 - sigmoid(x)).
func (s *SiLU) Backward(gradOut [][]float32) [][]float32 {
	out := make([][]float32, len(gradOut))
	for i, g := range gradOut {
		x := s.cache[i]
		gi := make([]float32, len(g))
		for j, gv := range g {
			sig := sigmoid(x[j])
			gi[j] = gv * (sig + x[j]*sig*(1-sig))
		}
		out[i] = gi
	}
	return out
}

// Params returns no parameters.
func (s *SiLU) Params() []*Param { return nil }

// Reset drops the activation cache.
func (s *SiLU) Reset() { s.cache = nil }

// Softmax is a batched, parameterless softmax.
type Softmax struct {
	cache [][]float32
}

// Forward computes row-wise softmax.
func (s *Softmax) Forward(batch [][]float32) [][]float32 {
	out := make([][]float32, len(batch))
	s.cache = make([][]float32, len(batch))
	for i, x := range batch {
		p := softmaxRow(x)
		out[i] = p
		s.cache[i] = p
	}
	return out
}

// Backward converts output-probability gradients into logit gradients:
// dL/dz_j = p_j * (g_j - sum_k p_k * g_k).
func (s *Softmax) Backward(gradOut [][]float32) [][]float32 {
	out := make([][]float32, len(gradOut))
	for i, g := range gradOut {
		p := s.cache[i]
		dot := float32(0)
		for k, gv := range g {
			dot += p[k] * gv
		}
		gi := zeros(len(g))
		for j, gv := range g {
			gi[j] = p[j] * (gv - dot)
		}
		out[i] = gi
	}
	return out
}

// Params returns no parameters.
func (s *Softmax) Params() []*Param { return nil }

// Reset drops the probability cache.
func (s *Softmax) Reset() { s.cache = nil }

// RMSNorm is a batched root-mean-square layer norm with learnable scale:
// y_i = gamma_i * x_i * rsqrt(mean(x^2) + eps).
type RMSNorm struct {
	name  string
	size  int
	eps   float32
	gamma []float32
	gg    []float32

	// caches
	lastIn [][]float32
	rstd   []float32

	pg *Param
}

// NewRMSNorm creates an RMSNorm with unit scale.
func NewRMSNorm(name string, size int, eps float32) *RMSNorm {
	n := &RMSNorm{name: name, size: size, eps: eps}
	n.gamma = ones(size)
	n.gg = zeros(size)
	n.pg = &Param{Name: name + ".weight", Data: n.gamma, Grad: n.gg}
	return n
}

// Forward normalizes the batch.
func (n *RMSNorm) Forward(batch [][]float32) [][]float32 {
	out := make([][]float32, len(batch))
	n.lastIn = batch
	n.rstd = make([]float32, len(batch))
	inv := 1.0 / float64(n.size)
	for i, x := range batch {
		sum := 0.0
		for _, v := range x {
			sum += float64(v) * float64(v)
		}
		r := float32(1.0 / math.Sqrt(sum*inv+float64(n.eps)))
		n.rstd[i] = r
		y := zeros(n.size)
		for j, v := range x {
			y[j] = n.gamma[j] * v * r
		}
		out[i] = y
	}
	return out
}

// Backward accumulates scale gradients and returns the input gradient.
func (n *RMSNorm) Backward(gradOut [][]float32) [][]float32 {
	gradIn := make([][]float32, len(gradOut))
	inv := 1.0 / float64(n.size)
	for i, g := range gradOut {
		x := n.lastIn[i]
		r := n.rstd[i]
		// s = sum_j g_j * gamma_j * x_j
		s := float32(0)
		for j, gv := range g {
			s += gv * n.gamma[j] * x[j]
			n.gg[j] += gv * x[j] * r
		}
		gi := zeros(n.size)
		coef := s * r * r * r * float32(inv)
		for j, gv := range g {
			gi[j] = gv*n.gamma[j]*r - coef*x[j]
		}
		gradIn[i] = gi
	}
	return gradIn
}

// Params returns the scale parameter.
func (n *RMSNorm) Params() []*Param { return []*Param{n.pg} }

// Reset drops caches.
func (n *RMSNorm) Reset() { n.lastIn = nil; n.rstd = nil }

// Embedding is a learnable lookup table over stable category IDs.
type Embedding struct {
	name  string
	rows  int
	width int
	table []float32 // rows x width, row-major
	grad  []float32
	last  []int

	pt *Param
}

// NewEmbedding creates a zero-initialized embedding table. Task and manager
// inputs begin neutral so routing depends on context, not memorized IDs.
func NewEmbedding(name string, rows, width int) *Embedding {
	e := &Embedding{name: name, rows: rows, width: width}
	e.table = zeros(rows * width)
	e.grad = zeros(rows * width)
	e.pt = &Param{Name: name + ".weight", Data: e.table, Grad: e.grad}
	return e
}

// Forward looks up one row per sample index.
func (e *Embedding) Forward(indices []int) [][]float32 {
	out := make([][]float32, len(indices))
	e.last = indices
	for i, idx := range indices {
		out[i] = e.table[idx*e.width : (idx+1)*e.width]
	}
	return out
}

// Backward scatter-adds output gradients into the matching table rows and
// returns nil: embedding inputs are integer IDs with no gradient.
func (e *Embedding) Backward(gradOut [][]float32) [][]float32 {
	for i, idx := range e.last {
		row := e.grad[idx*e.width : (idx+1)*e.width]
		for j, gv := range gradOut[i] {
			row[j] += gv
		}
	}
	return nil
}

// Rows returns the number of embedding rows.
func (e *Embedding) Rows() int { return e.rows }

// Width returns the embedding width.
func (e *Embedding) Width() int { return e.width }

// Params returns the table parameter.
func (e *Embedding) Params() []*Param { return []*Param{e.pt} }

// Reset drops the index cache.
func (e *Embedding) Reset() { e.last = nil }

// ones allocates a float32 slice filled with 1.
func ones(n int) []float32 {
	s := make([]float32, n)
	for i := range s {
		s[i] = 1
	}
	return s
}
