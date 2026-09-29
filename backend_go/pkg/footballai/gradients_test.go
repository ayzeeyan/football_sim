package footballai

import (
	"math"
	"math/rand"
	"testing"
)

// Each primitive's backward is checked below against central finite
// differences on tiny deterministic tensors, per the development order.

func TestLinearForward(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	l := NewLinear("t", 3, 2, rng)
	x := []float32{1, -2, 0.5}
	y := l.Forward([][]float32{x})[0]
	if len(y) != 2 {
		t.Fatalf("output width %d", len(y))
	}
	for o := 0; o < 2; o++ {
		want := l.b[o]
		for j := 0; j < 3; j++ {
			want += l.w[o*3+j] * x[j]
		}
		if math.Abs(float64(y[o]-want)) > 1e-6 {
			t.Errorf("y[%d] = %g, want %g", o, y[o], want)
		}
	}
}

func TestLinearBackwardNumerical(t *testing.T) {
	rng := rand.New(rand.NewSource(11))
	l := NewLinear("t", 5, 4, rng)
	x := [][]float32{{0.3, -1.2, 0.7, 2.2, -0.4}}
	g := [][]float32{{0.5, -1.0, 0.25, 2.0}}
	eps := 1e-3

	loss := func() float32 {
		l.Reset()
		y := l.Forward(x)[0]
		s := float32(0)
		for j := range y {
			s += y[j] * g[0][j]
		}
		return s
	}
	// Analytic.
	l.Reset()
	l.Forward(x)
	gradX := l.Backward(g)

	check := func(p *Param) {
		for i := range p.Data {
			if i%13 != 0 {
				continue // spot-check for speed
			}
			orig := p.Data[i]
			p.Data[i] = orig + float32(eps)
			lp := loss()
			p.Data[i] = orig - float32(eps)
			lm := loss()
			p.Data[i] = orig
			fd := float64((lp - lm) / (2 * float32(eps)))
			if math.Abs(fd-float64(p.Grad[i])) > 2e-2*(1+math.Abs(fd)) {
				t.Fatalf("%s[%d]: analytic %g, fd %g", p.Name, i, p.Grad[i], fd)
			}
		}
	}
	check(l.pw)
	check(l.pb)

	// Input gradient.
	for j := range x[0] {
		orig := x[0][j]
		x[0][j] = orig + float32(eps)
		lp := loss()
		x[0][j] = orig - float32(eps)
		lm := loss()
		x[0][j] = orig
		fd := float64((lp - lm) / (2 * float32(eps)))
		if math.Abs(fd-float64(gradX[0][j])) > 2e-2*(1+math.Abs(fd)) {
			t.Fatalf("gradX[%d]: analytic %g, fd %g", j, gradX[0][j], fd)
		}
	}
}

func TestSiLUGradient(t *testing.T) {
	s := &SiLU{}
	for _, xv := range []float32{-2, -0.5, 0, 0.7, 3} {
		x := [][]float32{{xv}}
		g := [][]float32{{1.7}}
		s.Reset()
		s.Forward(x)
		gx := s.Backward(g)[0][0]
		eps := 1e-3
		fd := float64((silu(xv+float32(eps))-silu(xv-float32(eps)))/(2*float32(eps))) * 1.7
		if math.Abs(fd-float64(gx)) > 1e-3*(1+math.Abs(fd)) {
			t.Errorf("silu grad at %g: analytic %g, fd %g", xv, gx, fd)
		}
	}
}

func TestSoftmaxGradient(t *testing.T) {
	s := &Softmax{}
	logits := []float32{0.2, -1.1, 0.9, 0.05}
	g := []float32{0.4, -0.3, 1.2, -0.8}
	s.Reset()
	s.Forward([][]float32{logits})
	gl := s.Backward([][]float32{g})[0]
	eps := 1e-3
	loss := func(shift []float32) float32 {
		p := softmaxRow(shift)
		s := float32(0)
		for i := range p {
			s += p[i] * g[i]
		}
		return s
	}
	for j := range logits {
		plus := make([]float32, len(logits))
		minus := make([]float32, len(logits))
		copy(plus, logits)
		copy(minus, logits)
		plus[j] += float32(eps)
		minus[j] -= float32(eps)
		fd := float64((loss(plus) - loss(minus)) / (2 * float32(eps)))
		if math.Abs(fd-float64(gl[j])) > 1e-2*(1+math.Abs(fd)) {
			t.Errorf("softmax grad[%d]: analytic %g, fd %g", j, gl[j], fd)
		}
	}
}

func TestRMSNormGradient(t *testing.T) {
	n := NewRMSNorm("t", 6, 1e-6)
	for i := range n.gamma {
		n.gamma[i] = 0.5 + float32(i)*0.2
	}
	x := []float32{0.8, -1.4, 2.1, -0.3, 0.55, 1.9}
	g := []float32{0.3, -0.7, 1.1, 0.2, -0.45, 0.9}
	eps := 2e-3

	loss := func() float32 {
		n.Reset()
		y := n.Forward([][]float32{x})[0]
		s := float32(0)
		for i := range y {
			s += y[i] * g[i]
		}
		return s
	}
	n.Reset()
	n.Forward([][]float32{x})
	gx := n.Backward([][]float32{g})[0]

	for i := range n.gamma {
		orig := n.gamma[i]
		n.gamma[i] = orig + float32(eps)
		lp := loss()
		n.gamma[i] = orig - float32(eps)
		lm := loss()
		n.gamma[i] = orig
		fd := float64((lp - lm) / (2 * float32(eps)))
		if math.Abs(fd-float64(n.gg[i])) > 2e-2*(1+math.Abs(fd)) {
			t.Errorf("gamma[%d]: analytic %g, fd %g", i, n.gg[i], fd)
		}
	}
	for i := range x {
		orig := x[i]
		x[i] = orig + float32(eps)
		lp := loss()
		x[i] = orig - float32(eps)
		lm := loss()
		x[i] = orig
		fd := float64((lp - lm) / (2 * float32(eps)))
		if math.Abs(fd-float64(gx[i])) > 2e-2*(1+math.Abs(fd)) {
			t.Errorf("gradX[%d]: analytic %g, fd %g", i, gx[i], fd)
		}
	}
}

func TestResidualBlockGradient(t *testing.T) {
	rng := rand.New(rand.NewSource(23))
	b := NewResidualBlock("t", 4, 7, rng)
	x := [][]float32{{0.4, -1.1, 0.9, 2.0}}
	g := [][]float32{{0.7, -0.2, 1.3, -0.5}}
	eps := 1e-3

	loss := func() float32 {
		b.Reset()
		y := b.Forward(x)[0]
		s := float32(0)
		for i := range y {
			s += y[i] * g[0][i]
		}
		return s
	}
	b.Reset()
	b.Forward(x)
	gx := b.Backward(g)[0]

	for _, p := range b.Params() {
		for i := range p.Data {
			if i%11 != 0 {
				continue
			}
			orig := p.Data[i]
			p.Data[i] = orig + float32(eps)
			lp := loss()
			p.Data[i] = orig - float32(eps)
			lm := loss()
			p.Data[i] = orig
			fd := float64((lp - lm) / (2 * float32(eps)))
			if math.Abs(fd-float64(p.Grad[i])) > 5e-3*(1+math.Abs(fd)) {
				t.Fatalf("%s[%d]: analytic %g, fd %g", p.Name, i, p.Grad[i], fd)
			}
		}
	}
	for i := range x[0] {
		orig := x[0][i]
		x[0][i] = orig + float32(eps)
		lp := loss()
		x[0][i] = orig - float32(eps)
		lm := loss()
		x[0][i] = orig
		fd := float64((lp - lm) / (2 * float32(eps)))
		if math.Abs(fd-float64(gx[i])) > 5e-3*(1+math.Abs(fd)) {
			t.Fatalf("gradX[%d]: analytic %g, fd %g", i, gx[i], fd)
		}
	}
}

func TestTop2Selection(t *testing.T) {
	// Spec example: Match .12 / Player .56 / Economy .27 / Club .05
	// selects Player+Economy renormalized to .56/.83 and .27/.83.
	top := selectTop2([]float32{0.12, 0.56, 0.27, 0.05}, 2)
	if top.Experts[0] != ExpertPlayer || top.Experts[1] != ExpertEconomy {
		t.Fatalf("selected %v %v, want Player Economy", top.Experts[0], top.Experts[1])
	}
	wantA := 0.56 / 0.83
	wantB := 0.27 / 0.83
	if math.Abs(float64(top.Weights[0]-float32(wantA))) > 1e-5 ||
		math.Abs(float64(top.Weights[1]-float32(wantB))) > 1e-5 {
		t.Fatalf("weights %v, want %v %v", top.Weights, wantA, wantB)
	}
	if top.Weights[0]+top.Weights[1] > 1.0001 || top.Weights[0]+top.Weights[1] < 0.9999 {
		t.Fatalf("weights do not renormalize: %v", top.Weights)
	}
}

func TestTop2TieStability(t *testing.T) {
	top := selectTop2([]float32{0.25, 0.25, 0.25, 0.25}, 2)
	if top.Weights[0] != 0.5 || top.Weights[1] != 0.5 {
		t.Fatalf("uniform tie weights %v", top.Weights)
	}
}

func TestLossGradients(t *testing.T) {
	// Huber transitions to linear outside delta.
	l1, g1 := huber(2.0, 0.0, 1.0, 1.0)
	if math.Abs(float64(l1-1.5)) > 1e-6 || math.Abs(float64(g1-1.0)) > 1e-6 {
		t.Errorf("huber far: loss %g grad %g", l1, g1)
	}
	l2, g2 := huber(0.4, 0.0, 1.0, 1.0)
	if math.Abs(float64(l2-0.08)) > 1e-6 || math.Abs(float64(g2-0.4)) > 1e-6 {
		t.Errorf("huber near: loss %g grad %g", l2, g2)
	}
	// BCE-with-sigmoid gradient is p - t.
	spec := OutputSpec{Name: "p", Kind: OutProb, Loss: LossBCE}
	for _, tc := range []struct{ raw, target float32 }{
		{2.0, 1.0}, {-2.0, 0.0}, {0.0, 0.5}, {0.3, 0.1},
	} {
		_, grad := LossAndGrad(spec, tc.raw, tc.target, 1.0)
		p := sigmoid(tc.raw)
		want := p - tc.target
		if math.Abs(float64(grad-want)) > 1e-5 {
			t.Errorf("bce grad raw=%v t=%v: %g, want %g", tc.raw, tc.target, grad, want)
		}
	}
}

func TestEmbeddingGradient(t *testing.T) {
	e := NewEmbedding("t", 3, 4)
	idx := []int{0, 2, 2, 1}
	e.Forward(idx)
	grad := [][]float32{
		{1, 0, 0, 0}, {0, 0, 1, 0}, {2, 0, 0, 0}, {0, 1, 0, 0},
	}
	e.Backward(grad)
	// Row 2 receives grad[1] + grad[2].
	want := []float32{2, 0, 1, 0}
	for j := range want {
		if e.grad[2*4+j] != want[j] {
			t.Fatalf("row2 grad[%d] = %g, want %g", j, e.grad[2*4+j], want[j])
		}
	}
	if e.grad[0] != 1 {
		t.Errorf("row0 grad %v", e.grad[0:4])
	}
	if e.grad[1*4+1] != 1 {
		t.Errorf("row1 grad wrong")
	}
}

func TestRMSNormUnitScale(t *testing.T) {
	n := NewRMSNorm("t", 4, 1e-6)
	y := n.Forward([][]float32{{1, 1, 1, 1}})[0]
	for _, v := range y {
		if math.Abs(float64(v-1)) > 1e-4 {
			t.Fatalf("unit vector scaled to %g, want 1", v)
		}
	}
}
