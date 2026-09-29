package footballai

import "testing"

// Benchmarks for the spec's required surfaces. Run with:
//
//	go test ./pkg/footballai/... -bench . -benchmem
//
// Do not optimize before measuring; these establish the CPU baseline.

func benchNet(b *testing.B) *Network {
	net, err := NewNetwork(DefaultConfig(), 42)
	if err != nil {
		b.Fatal(err)
	}
	return net
}

func benchMatchReq() MatchRequest {
	return MatchRequest{
		Match: MatchContext{
			HomeRating: 82, AwayRating: 75, HomeForm: 2, AwayForm: -1,
			HomeFitness: 90, AwayFitness: 85, HomeFatigue: 1, AwayFatigue: 3,
			TacticalEdge: 0.2, MatchImportance: 0.6, HomeAbsences: 1, AwayAbsences: 2,
		},
		World:   WorldContext{MatchImportance: 0.6, PlayerFatigue: 0.3},
		Manager: ManagerFeatures{Risk: 0.4, Patience: 0.5, Attacking: 0.6, Pressing: 0.5},
	}
}

func BenchmarkSinglePrediction(b *testing.B) {
	m := BuildModel(benchNet(b), "")
	req := benchMatchReq()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := m.PredictMatch(req); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPredict1000(b *testing.B) {
	m := BuildModel(benchNet(b), "")
	req := benchMatchReq()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 1000; j++ {
			if _, err := m.PredictMatch(req); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkPredictBatch256(b *testing.B) {
	m := BuildModel(benchNet(b), "")
	batch := make([]Request, 256)
	for i := range batch {
		batch[i] = benchMatchReq().Encode()
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := m.PredictBatch(batch); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMixedTaskForwardBackward(b *testing.B) {
	// One training-style batch: forward + backward over a mixed-task batch,
	// the dominant cost of an offline training step.
	net := benchNet(b)
	reqs := make([]Request, 256)
	for i := range reqs {
		if i%2 == 0 {
			reqs[i] = benchMatchReq().Encode()
		} else {
			reqs[i] = InjuryRequest{
				Player: PlayerFeatures{Age: 25, OVR: 78, Fitness: 88, Fatigue: 3, Position: PosMID},
				World:  WorldContext{MatchImportance: 0.4},
			}.Encode()
		}
	}
	grad := make([][]float32, len(reqs))
	for i := range grad {
		grad[i] = []float32{0.1, -0.2, 0.05, 0.01}
	}
	weight := make([]float32, len(reqs))
	for i := range weight {
		weight[i] = 0.3
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		net.Reset()
		fp, err := net.ForwardTrain(reqs)
		if err != nil {
			b.Fatal(err)
		}
		net.BackwardTrain(fp, BackwardInput{HeadGrad: grad, SampleWeight: weight})
		net.ZeroGrad()
	}
}
