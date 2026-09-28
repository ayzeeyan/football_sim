package brain

import (
	"math"
	"testing"
)

// The brain learns: training on a clear pattern moves the corresponding
// weight in the right direction, and the prediction follows.
func TestBrainLearnsFromOutcomes(t *testing.T) {
	m := &Model{Bias: 0}
	var f [NumFeatures]float64
	f[FeatureRatingDiff] = 1.0
	// The feature always predicts a two-goal home win.
	for i := 0; i < 500; i++ {
		m.Train(f, 2.0)
	}
	if m.Weights[FeatureRatingDiff] <= 0.5 {
		t.Fatalf("rating weight=%f, want it to grow clearly toward the target (L2 decay caps it just under 1.0)", m.Weights[FeatureRatingDiff])
	}
	if math.Abs(m.Predict(f)-2.0) > 0.25 {
		t.Fatalf("prediction=%f want close to 2.0", m.Predict(f))
	}
	if m.Samples != 500 {
		t.Fatalf("samples=%d want 500", m.Samples)
	}
}

// Training is deterministic: identical rows in identical order produce
// byte-identical weights.
func TestTrainingIsDeterministic(t *testing.T) {
	run := func() (Model, float64) {
		m := &Model{Bias: 0.1}
		var loss float64
		for i := 0; i < 100; i++ {
			var f [NumFeatures]float64
			f[FeatureRatingDiff] = float64(i%5) / 5.0
			f[FeatureFormDiff] = float64(i%3) / 3.0
			f[FeatureHomeBeats] = float64(i % 2)
			loss = m.Train(f, float64(i%7-3))
		}
		return *m, loss
	}
	a, la := run()
	b, lb := run()
	if a != b || la != lb {
		t.Fatal("identical training runs produced different models")
	}
}

// The base artifact is valid and non-trivial: a career starts from the
// pre-trained weights, not from zero.
func TestBaseIsPreTrained(t *testing.T) {
	base := Base()
	if !base.Valid() {
		t.Fatal("committed base is invalid")
	}
	if base.Samples < 1000 {
		t.Fatalf("base samples=%d, want a real pre-training run (>1000)", base.Samples)
	}
	// The fitted rating weight must point in the obvious direction.
	if base.Weights[FeatureRatingDiff] <= 0 {
		t.Fatalf("base rating weight=%f, want positive", base.Weights[FeatureRatingDiff])
	}
	if base.Weights[FeatureFormDiff] <= 0 {
		t.Fatalf("base form weight=%f, want positive", base.Weights[FeatureFormDiff])
	}
}

// The model stays valid under sustained training and stays tiny: the whole
// brain is a fixed-size struct.
func TestModelStaysValidAndTiny(t *testing.T) {
	m := New()
	for i := 0; i < 10000; i++ {
		var f [NumFeatures]float64
		for j := 0; j < NumFeatures; j++ {
			f[j] = float64((i*j)%7) / 7.0
		}
		m.Train(f, float64(i%9-4))
	}
	if !m.Valid() {
		t.Fatal("model went invalid under sustained training")
	}
	if math.IsNaN(m.Predict([NumFeatures]float64{1, 1, 1, 1, 1, 1, 1, 1})) {
		t.Fatal("prediction went NaN")
	}
}
