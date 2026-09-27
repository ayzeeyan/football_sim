package persistence

import (
	"encoding/json"
	"testing"
)

func TestProfileShotItemFields(t *testing.T) {
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
	n := 0
	for _, f := range fixtures {
		r, ok := f["report"]
		if !ok {
			continue
		}
		var rep map[string]json.RawMessage
		if err := json.Unmarshal(r, &rep); err != nil {
			t.Fatal(err)
		}
		var shots map[string]json.RawMessage
		if err := json.Unmarshal(rep["shot_map"], &shots); err != nil {
			t.Fatal(err)
		}
		rawItems, ok := shots["shots"]
		if !ok || len(rawItems) == 0 {
			continue
		}
		var items []map[string]json.RawMessage
		if err := json.Unmarshal(rawItems, &items); err != nil {
			t.Fatal(err)
		}
		n += len(items)
		for _, it := range items {
			for k, v := range it {
				agg[k] += len(v)
			}
		}
	}
	t.Logf("total shots: %d", n)
	reportFieldProfile(t, "shots[]", agg)
}
