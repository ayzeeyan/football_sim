package persistence

import (
	"encoding/json"
	"testing"
)

func TestProfileHeatmapAndShotMapInternals(t *testing.T) {
	raw := readExistingSaveBytes(t)
	var full map[string]json.RawMessage
	if err := json.Unmarshal(raw, &full); err != nil {
		t.Fatal(err)
	}
	var world map[string]json.RawMessage
	if err := json.Unmarshal(full["world"], &world); err != nil {
		t.Fatal(err)
	}
	var fixtures []map[string]json.RawMessage
	if err := json.Unmarshal(world["fixtures"], &fixtures); err != nil {
		t.Fatal(err)
	}
	heatAgg := make(map[string]int)
	shotAgg := make(map[string]int)
	var homePts, awayPts int
	for _, f := range fixtures {
		r, ok := f["report"]
		if !ok {
			continue
		}
		var rep map[string]json.RawMessage
		if err := json.Unmarshal(r, &rep); err != nil {
			t.Fatal(err)
		}
		var heat map[string]json.RawMessage
		if err := json.Unmarshal(rep["heatmap"], &heat); err != nil {
			t.Fatal(err)
		}
		for k, v := range heat {
			heatAgg[k] += len(v)
		}
		var pts [2][][]float64
		for i, key := range []string{"home_points", "away_points"} {
			rawPts, ok := heat[key]
			if !ok || len(rawPts) == 0 {
				continue
			}
			if err := json.Unmarshal(rawPts, &pts[i]); err != nil {
				t.Fatal(err)
			}
		}
		homePts += len(pts[0])
		awayPts += len(pts[1])
		var shots map[string]json.RawMessage
		if err := json.Unmarshal(rep["shot_map"], &shots); err != nil {
			t.Fatal(err)
		}
		for k, v := range shots {
			shotAgg[k] += len(v)
		}
	}
	t.Logf("heatmap points: home=%d away=%d", homePts, awayPts)
	reportFieldProfile(t, "heatmap", heatAgg)
	reportFieldProfile(t, "shot_map", shotAgg)
}
