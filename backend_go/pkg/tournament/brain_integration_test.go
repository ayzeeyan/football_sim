package tournament

import (
	"encoding/json"
	"testing"

	"football_sim/pkg/brain"
)

// The brain post-trains on real matches: after simulated weeks the sample
// count grows and the learned edge blends toward the brain's own opinion.
func TestBrainPostTrainsDuringSeason(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	if tm.Brain == nil {
		t.Fatal("world has no brain")
	}
	baseSamples := tm.Brain.Samples
	batch := tm.SimulateBatchWeeks(10)
	if batch.Status != "success" {
		t.Fatalf("simulation failed: %+v", batch)
	}
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	if tm.Brain.Samples <= baseSamples {
		t.Fatalf("brain did not train: samples %d -> %d", baseSamples, tm.Brain.Samples)
	}
	if !tm.Brain.Valid() {
		t.Fatal("brain went invalid during the season")
	}
}

// Determinism: identical seeds produce byte-identical brains — the online
// post-training is a pure function of the deterministic match history.
func TestBrainIsDeterministicAcrossSeeds(t *testing.T) {
	run := func() *brain.Model {
		tm, _, _ := loadEuropeanWorldForTest(t)
		batch := tm.SimulateBatchWeeks(12)
		if batch.Status != "success" {
			t.Fatalf("simulation failed: %+v", batch)
		}
		return tm.BrainState()
	}
	a := run()
	b := run()
	if a == nil || b == nil {
		t.Fatal("brain missing")
	}
	rawA, _ := json.Marshal(a)
	rawB, _ := json.Marshal(b)
	if string(rawA) != string(rawB) {
		t.Fatalf("identical seeds produced different brains:\n%s\nvs\n%s", rawA, rawB)
	}
}

// The learned edge starts at the fixed table and blends toward the brain's
// own prediction as samples accumulate.
func TestBrainEdgeBlendsFromFixedToLearned(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	// A fresh brain with zero samples keeps the fixed table exactly.
	tm.mu.Lock()
	tm.Brain = &brain.Model{}
	tm.mu.Unlock()
	fixed := tm.BrainTacticEdge("high_press", "possession")
	if fixed <= 0 {
		t.Fatalf("zero-sample edge=%f, want the fixed table value", fixed)
	}
	// A heavily trained brain owns the edge: the blend weight saturates.
	tm.mu.Lock()
	tm.Brain = &brain.Model{Samples: brainEdgeBlendSamples * 2}
	tm.Brain.Weights[brain.FeatureHomeBeats] = 0.2
	tm.mu.Unlock()
	learned := tm.BrainTacticEdge("high_press", "possession")
	if learned < 0.19 || learned > 0.21 {
		t.Fatalf("saturated edge=%f, want the learned 0.2", learned)
	}
}
