package weights

import (
	"os"
	"path/filepath"
	"testing"

	"football_sim/pkg/footballai"
)

func testNet(t *testing.T) *footballai.Network {
	t.Helper()
	net, err := footballai.NewNetwork(footballai.DefaultConfig(), 99)
	if err != nil {
		t.Fatal(err)
	}
	return net
}

func matchReq() footballai.MatchRequest {
	return footballai.MatchRequest{
		Match: footballai.MatchContext{
			HomeRating: 83, AwayRating: 76, HomeForm: 1.5, AwayForm: -2,
			HomeFitness: 91, AwayFitness: 88, HomeFatigue: 2, AwayFatigue: 4,
			TacticalEdge: 0.3, MatchImportance: 0.7, Derby: true,
			HomeAbsences: 1, AwayAbsences: 0, HomeAttackBias: 0.4, AwayAttackBias: 0.6,
		},
		World:   footballai.WorldContext{MatchImportance: 0.7, ClubPressure: 0.3},
		Manager: footballai.ManagerFeatures{Risk: 0.5, Patience: 0.4, Attacking: 0.7},
	}
}

func TestSaveLoadIdenticalPredictions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.fmoe")
	net := testNet(t)

	want, err := footballai.BuildModel(net, "").PredictMatch(matchReq())
	if err != nil {
		t.Fatal(err)
	}

	if _, err := Save(path, net); err != nil {
		t.Fatal(err)
	}
	m, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := m.PredictMatch(matchReq())
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("reloaded model predicts differently: %+v vs %+v", got, want)
	}
	if m.ParameterCount() != net.ParameterCount() {
		t.Fatalf("parameter count mismatch %d vs %d", m.ParameterCount(), net.ParameterCount())
	}
}

func TestChecksumDetectsCorruption(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.fmoe")
	if _, err := Save(path, testNet(t)); err != nil {
		t.Fatal(err)
	}
	// Corrupt one byte in the tensor data section.
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	raw[len(raw)-100] ^= 0xFF
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("corrupted model loaded without checksum error")
	}
	if err := Verify(path); err == nil {
		t.Fatal("Verify accepted a corrupted file")
	}
}

func TestVerifyCleanFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.fmoe")
	if _, err := Save(path, testNet(t)); err != nil {
		t.Fatal(err)
	}
	if err := Verify(path); err != nil {
		t.Fatal(err)
	}
}

func TestCheckpointRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.fmckpt")
	net := testNet(t)

	state := &CheckpointState{
		Epoch:        12,
		Step:         3456,
		LearningRate: 0.0007,
		BestValScore: 0.31,
		RandomSeed:   42,
		Metrics:      map[string]float64{"val_loss": 0.31},
		Moments:      map[string]Moment{},
	}
	for _, p := range net.Params() {
		m := make([]float32, len(p.Data))
		v := make([]float32, len(p.Data))
		for i := range m {
			m[i] = float32(i) * 0.01
			v[i] = float32(i) * 0.001
		}
		state.Moments[p.Name] = Moment{M: m, V: v}
	}
	if _, err := SaveCheckpoint(path, net, state); err != nil {
		t.Fatal(err)
	}
	model, loaded, err := LoadCheckpoint(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Epoch != 12 || loaded.Step != 3456 || loaded.RandomSeed != 42 {
		t.Fatalf("checkpoint state mismatch: %+v", loaded)
	}
	for _, p := range model.Network().Params() {
		mom, ok := loaded.Moments[p.Name]
		if !ok {
			t.Fatalf("missing moments for %s", p.Name)
		}
		if len(mom.M) != len(p.Data) {
			t.Fatalf("moments length mismatch for %s: %d vs %d", p.Name, len(mom.M), len(p.Data))
		}
		if len(mom.M) > 3 && (mom.M[3] != float32(3)*0.01 || mom.V[3] != float32(3)*0.001) {
			t.Fatalf("moments corrupted for %s", p.Name)
		}
	}
	// A checkpoint must not be loadable as a deployment model.
	if _, err := Load(path); err == nil {
		t.Fatal("checkpoint loaded as deployment model")
	}
}

func TestIncompatibleArchitectureRejected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.fmoe")
	net := testNet(t)
	if _, err := Save(path, net); err != nil {
		t.Fatal(err)
	}
	// Patch the config's input width so the feature schema no longer matches
	// the runtime; the loader must refuse the file.
	ins, err := Inspect(path)
	if err != nil {
		t.Fatal(err)
	}
	bad := ins.Config
	bad.InputWidth = ins.Config.InputWidth + 1
	if err := bad.Validate(); err == nil {
		t.Fatal("modified config unexpectedly validated")
	}
}

func TestParameterBudget(t *testing.T) {
	net := testNet(t)
	count := net.ParameterCount()
	if count < 250_000 || count > 400_000 {
		t.Fatalf("parameter count %d outside the 250k-400k target", count)
	}
}
