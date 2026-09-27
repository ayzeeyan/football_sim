package persistence

import (
	"encoding/json"
	"testing"
)

func TestProfileReportInternals(t *testing.T) {
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
	agg := make(map[string]int)
	count := 0
	for _, f := range fixtures {
		r, ok := f["report"]
		if !ok {
			continue
		}
		count++
		for key, val := range fieldSizeProfile(t, r, false) {
			agg[key] += val
		}
	}
	t.Logf("aggregated over %d world reports", count)
	reportFieldProfile(t, "report[]", agg)
}
