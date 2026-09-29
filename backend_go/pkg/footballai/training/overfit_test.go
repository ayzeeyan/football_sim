package training

import (
	"math"
	"testing"

	"football_sim/pkg/footballai"
)

// TestOverfitTinyBatch trains the full MoE on 32 synthetic examples. Driving
// the loss very low is the fastest detector for broken backpropagation: if
// any layer's gradient is wrong, the loss stalls. A regression task is used
// because probabilistic losses have an irreducible entropy floor.
func TestOverfitTinyBatch(t *testing.T) {
	cfg := footballai.DefaultConfig()
	net, err := footballai.NewNetwork(cfg, 7)
	if err != nil {
		t.Fatal(err)
	}
	if err := net.SetNormMeta(footballai.DefaultNormMeta()); err != nil {
		t.Fatal(err)
	}
	specs := footballai.TaskSpecs()
	spec := specs[footballai.TaskDevelopment]

	// 32 development samples: the season delta is a smooth function of age,
	// fitness, and training quality.
	samples := make([]*Sample, 32)
	for i := range samples {
		age := 17 + i%15
		training := float32(i%5) * 0.2
		slots := make([]float32, footballai.InputWidth)
		slots[footballai.SlotAge] = float32(age)
		slots[footballai.SlotOVR] = 70 + float32(i)*0.5
		slots[footballai.SlotFitness] = 85 + float32(i%7)
		slots[footballai.SlotTrainingQuality] = training
		slots[footballai.SlotPositionOneHot+int(footballai.PosMID)] = 1
		delta := 1.2 - float32(age-17)*0.06 + training
		samples[i] = &Sample{
			Task:   footballai.TaskDevelopment,
			Slots:  slots,
			Target: []float32{delta},
			Weight: 1,
		}
	}

	params := net.Params()
	adam := NewAdamW(params, AdamWOptions{LearningRate: 3e-3, WeightDecay: 0})

	firstLoss := -1.0
	for step := 0; step < 400; step++ {
		var loss float64
		for _, s := range samples {
			reqs := []footballai.Request{{Task: s.Task, Slots: s.Slots}}
			net.Reset()
			fp, err := net.ForwardTrain(reqs)
			if err != nil {
				t.Fatal(err)
			}
			raw := fp.RawOutputs(0)
			l, g := footballai.LossAndGrad(spec.Outputs[0], raw[0], s.Target[0], 1)
			loss += float64(l)
			net.BackwardTrain(fp, footballai.BackwardInput{
				HeadGrad:     [][]float32{{g}},
				SampleWeight: []float32{1},
			})
			ClipGlobalNorm(net.Params(), 1.0)
			if err := adam.Step(net.Params()); err != nil {
				t.Fatal(err)
			}
			net.ZeroGrad()
			net.Reset()
		}
		if step == 0 {
			firstLoss = loss / float64(len(samples))
		}
		if step == 399 {
			final := loss / float64(len(samples))
			t.Logf("overfit: first loss %.4f -> final loss %.4f", firstLoss, final)
			if math.IsNaN(final) {
				t.Fatal("NaN loss during overfit")
			}
			if final > 0.02 {
				t.Fatalf("network failed to overfit 32 examples: %.4f -> %.4f", firstLoss, final)
			}
		}
	}
}
