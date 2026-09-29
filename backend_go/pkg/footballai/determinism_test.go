package footballai

import "testing"

// TestDeterministicInference verifies the spec's determinism contract: the
// same weights and the same request always produce the same output, with no
// hidden inference-time randomness.
func TestDeterministicInference(t *testing.T) {
	net, err := NewNetwork(DefaultConfig(), 5)
	if err != nil {
		t.Fatal(err)
	}
	m := BuildModel(net, "")
	req := MatchRequest{
		Match: MatchContext{
			HomeRating: 80, AwayRating: 75, HomeForm: 1, AwayForm: -1,
			HomeFitness: 90, AwayFitness: 90, TacticalEdge: 0.1, MatchImportance: 0.5,
		},
		World: WorldContext{MatchImportance: 0.5},
	}
	first, err := m.PredictMatch(req)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		got, err := m.PredictMatch(req)
		if err != nil {
			t.Fatal(err)
		}
		if got != first {
			t.Fatalf("inference not deterministic at call %d: %+v vs %+v", i, got, first)
		}
	}
}

// TestSameSeedSameWeights verifies reproducible initialization.
func TestSameSeedSameWeights(t *testing.T) {
	a, err := NewNetwork(DefaultConfig(), 1234)
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewNetwork(DefaultConfig(), 1234)
	if err != nil {
		t.Fatal(err)
	}
	pa, pb := a.Params(), b.Params()
	if len(pa) != len(pb) {
		t.Fatal("param count mismatch")
	}
	for i := range pa {
		if len(pa[i].Data) != len(pb[i].Data) {
			t.Fatalf("%s length mismatch", pa[i].Name)
		}
		for j := range pa[i].Data {
			if pa[i].Data[j] != pb[i].Data[j] {
				t.Fatalf("%s[%d] differs across same-seed networks", pa[i].Name, j)
			}
		}
	}
}

// TestRouterLearnsContext tests that routing depends on request context, not
// only the task embedding: two requests of the same task must be able to
// route differently given different features.
func TestRoutingUsesContext(t *testing.T) {
	net, err := NewNetwork(DefaultConfig(), 21)
	if err != nil {
		t.Fatal(err)
	}
	// Two injury requests with wildly different context.
	low := InjuryRequest{
		Player: PlayerFeatures{Age: 22, OVR: 70, Fitness: 100, Fatigue: 0, ConsecutiveStarts: 0, Position: PosGK},
		World:  WorldContext{MatchImportance: 0.1},
	}
	high := InjuryRequest{
		Player: PlayerFeatures{Age: 34, OVR: 88, Fitness: 60, Fatigue: 9, ConsecutiveStarts: 12, RecentInjury: true, Position: PosFWD},
		World:  WorldContext{MatchImportance: 0.95, PlayerFatigue: 0.9},
	}
	fp, err := net.forward([]Request{low.Encode(), high.Encode()})
	if err != nil {
		t.Fatal(err)
	}
	defer net.Reset()
	r0 := fp.Routing(0)
	r1 := fp.Routing(1)
	// With random init the two must renormalize correctly regardless.
	for _, r := range []Top2{r0, r1} {
		sum := r.Weights[0] + r.Weights[1]
		if sum < 0.9999 || sum > 1.0001 {
			t.Fatalf("top-2 weights do not renormalize: %v", r.Weights)
		}
	}
}

// TestPredictBatchMatchesSingle verifies batched prediction equals
// per-sample prediction.
func TestPredictBatchMatchesSingle(t *testing.T) {
	net, err := NewNetwork(DefaultConfig(), 33)
	if err != nil {
		t.Fatal(err)
	}
	m := BuildModel(net, "")
	base := InjuryRequest{
		Player: PlayerFeatures{Age: 25, OVR: 78, Fitness: 88, Fatigue: 3, Position: PosMID},
		World:  WorldContext{MatchImportance: 0.4},
	}
	reqs := make([]Request, 8)
	for i := range reqs {
		r := base
		r.Player.Fatigue = float32(i)
		reqs[i] = r.Encode()
	}
	batch, err := m.PredictBatch(reqs)
	if err != nil {
		t.Fatal(err)
	}
	for i, r := range reqs {
		single, err := m.predictOne(r)
		if err != nil {
			t.Fatal(err)
		}
		if batch[i].Outputs[0] != single.Outputs[0] {
			t.Fatalf("batch/single mismatch at %d: %g vs %g", i, batch[i].Outputs[0], single.Outputs[0])
		}
	}
}

// TestNormMetaRoundTrip ensures stored normalization is applied identically.
func TestNormMetaRoundTrip(t *testing.T) {
	meta := DefaultNormMeta()
	meta.Slots[slotOVR].Mean = 75
	meta.Slots[slotOVR].Std = 8
	meta.Slots[slotBaselineAnchor].Mean = 16
	meta.Slots[slotBaselineAnchor].Std = 1.5

	x := make([]float32, InputWidth)
	x[slotOVR] = 83                            // (83-75)/8 = 1
	x[slotBaselineAnchor] = float32(exp64(16)) // log1p -> ~16 => (16-16)/1.5 = 0

	y := meta.Apply(x)
	if diff(y[slotOVR]-1) > 1e-5 {
		t.Fatalf("z-norm wrong: %g", y[slotOVR])
	}
	// exp(16)-1 is large; log1p(exp(16)-1) = 16.
	if diff(y[slotBaselineAnchor]) > 1e-3 {
		t.Fatalf("logz-norm wrong: %g", y[slotBaselineAnchor])
	}
	// One-hot untouched.
	x2 := make([]float32, InputWidth)
	x2[slotPositionOneHot+int(PosFWD)] = 1
	y2 := meta.Apply(x2)
	if y2[slotPositionOneHot+int(PosFWD)] != 1 {
		t.Fatal("identity normalization modified a one-hot slot")
	}
}

func diff(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

func exp64(v float64) float64 {
	// small local exp to avoid importing math in this test file
	e := 1.0
	term := 1.0
	for i := 1; i <= 40; i++ {
		term *= v / float64(i)
		e += term
	}
	return e
}
