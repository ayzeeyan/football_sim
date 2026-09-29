package footballai

import (
	"math"
	"math/rand"
)

// Param is one named trainable tensor (flat float32 storage) with its
// gradient buffer. Names mirror the .fmoe tensor directory.
type Param struct {
	Name string
	Data []float32
	Grad []float32
}

// ZeroGrad resets the gradient buffer in place.
func (p *Param) ZeroGrad() {
	for i := range p.Grad {
		p.Grad[i] = 0
	}
}

// NumElems returns the element count of the parameter.
func (p *Param) NumElems() int { return len(p.Data) }

// Module is implemented by every neural component so the trainer and the
// serializer can walk parameters uniformly.
type Module interface {
	// Params returns the module's parameters in stable order.
	Params() []*Param
	// Reset clears per-batch caches. Gradients are kept (they accumulate
	// across backward calls within a step and are zeroed by the optimizer).
	Reset()
}

// sigmoid computes the logistic function in float64 for accuracy.
func sigmoid(x float32) float32 {
	return float32(1.0 / (1.0 + math.Exp(-float64(x))))
}

// silu computes x * sigmoid(x).
func silu(x float32) float32 {
	return x * sigmoid(x)
}

// softmaxRow computes a numerically stable softmax over one row in place.
func softmaxRow(x []float32) []float32 {
	out := make([]float32, len(x))
	max := x[0]
	for _, v := range x[1:] {
		if v > max {
			max = v
		}
	}
	sum := 0.0
	for i, v := range x {
		e := math.Exp(float64(v - max))
		out[i] = float32(e)
		sum += e
	}
	inv := 1.0 / sum
	for i := range out {
		out[i] = float32(float64(out[i]) * inv)
	}
	return out
}

// xavierInit fills w with scaled gaussian weights using the given stream.
func xavierInit(w []float32, in, out int, rng *rand.Rand) {
	if rng == nil {
		rng = rand.New(rand.NewSource(1))
	}
	std := math.Sqrt(2.0 / float64(in+out))
	for i := range w {
		w[i] = float32(rng.NormFloat64() * std)
	}
}

// zeros allocates a zeroed float32 slice.
func zeros(n int) []float32 { return make([]float32, n) }

// batchZeros allocates a batch of zeroed vectors.
func batchZeros(n, width int) [][]float32 {
	b := make([][]float32, n)
	for i := range b {
		b[i] = zeros(width)
	}
	return b
}
