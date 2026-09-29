package weights

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"football_sim/pkg/footballai"
)

// TestBrainLoadAndRecorderRoundTrip exercises the runtime surface the game
// server uses: brain construction from a .fmoe file, nil-safety, and the
// observational recorder's pending-observation lifecycle.
func TestBrainLoadAndRecorderRoundTrip(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "m.fmoe")
	recordPath := filepath.Join(dir, "outcomes.jsonl")
	net := testNet(t)
	if _, err := Save(modelPath, net); err != nil {
		t.Fatal(err)
	}

	// Disabled brain: no model path.
	idle, err := footballai.NewBrain(footballai.AIConfig{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if idle.Enabled() || idle.Model() != nil || idle.Recorder() != nil {
		t.Fatal("empty config must produce a disabled brain")
	}
	// Nil brain is valid and fully disabled.
	var nilBrain *footballai.Brain
	if nilBrain.Enabled() || nilBrain.Model() != nil || nilBrain.Recorder() != nil {
		t.Fatal("nil brain must be nil-safe")
	}

	brain, err := footballai.NewBrain(footballai.AIConfig{
		ModelPath:      modelPath,
		RecordOutcomes: true,
		RecordPath:     recordPath,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !brain.Enabled() {
		t.Fatal("brain should be enabled with a valid model")
	}
	info := brain.Info()
	if info.ModelHash == "" || info.ModelVersion != net.Config().ModelVersion {
		t.Fatalf("bad model info: %+v", info)
	}

	// Recorder lifecycle: observe pre-match state, complete later.
	rec := brain.Recorder()
	if rec == nil {
		t.Fatal("recorder missing")
	}
	rec.SetClock(7, 1, 12)
	req := footballai.MatchRequest{
		Match: footballai.MatchContext{HomeRating: 80, AwayRating: 74, MatchImportance: 0.5},
	}
	id := rec.Observe(footballai.TaskMatchPrediction, req.Encode())
	if id == 0 {
		t.Fatal("observation ID must be nonzero when recording")
	}
	// A second observation that never completes stays pending harmlessly.
	_ = rec.Observe(footballai.TaskInjuryRisk, footballai.InjuryRequest{
		Player: footballai.PlayerFeatures{Age: 28, OVR: 80, Position: footballai.PosMID},
	}.Encode())
	rec.Complete(id, map[string]float64{"home_goals": 2, "away_goals": 1, "goal_diff": 1})
	// Duplicate completion is a no-op.
	rec.Complete(id, map[string]float64{"home_goals": 9})
	if err := rec.Close(); err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(recordPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	rows := 0
	for sc.Scan() {
		var row struct {
			LabelSource string             `json:"label_source"`
			Task        string             `json:"task"`
			SimID       uint64             `json:"sim_id"`
			Season      int                `json:"season"`
			Week        int                `json:"week"`
			Features    map[string]float64 `json:"features"`
			Target      map[string]float64 `json:"target"`
		}
		if err := json.Unmarshal(sc.Bytes(), &row); err != nil {
			t.Fatal(err)
		}
		if row.LabelSource != "simulation_outcome" {
			t.Fatalf("label source %q", row.LabelSource)
		}
		if row.Task != "match_prediction" {
			t.Fatalf("task %q", row.Task)
		}
		if row.SimID != 7 || row.Season != 1 || row.Week != 12 {
			t.Fatalf("clock not stamped: %+v", row)
		}
		if row.Features["match.home_rating"] != 80 {
			t.Fatalf("features not slotted: %v", row.Features)
		}
		if row.Target["home_goals"] != 2 || row.Target["goal_diff"] != 1 {
			t.Fatalf("targets wrong: %v", row.Target)
		}
		rows++
	}
	if rows != 1 {
		t.Fatalf("wrote %d rows, want exactly the completed one", rows)
	}

	// Model hash mismatch policy: a pinned save with a different hash must
	// not silently match.
	other := info
	other.ModelHash = "different"
	if other.Matches(brain.Info()) {
		t.Fatal("mismatched hashes must not match")
	}
	if !info.Matches(brain.Info()) {
		t.Fatal("identical hashes must match")
	}
}
