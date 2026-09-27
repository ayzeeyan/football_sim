package persistence

import (
	"encoding/json"
	"testing"
)

func fieldSizeProfile(t *testing.T, raw json.RawMessage, ofArray bool) map[string]int {
	t.Helper()
	if ofArray {
		var items []map[string]json.RawMessage
		if err := json.Unmarshal(raw, &items); err != nil {
			t.Fatalf("decode array: %v", err)
		}
		out := make(map[string]int)
		for _, item := range items {
			for key, val := range item {
				out[key] += len(val)
			}
		}
		return out
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		t.Fatalf("decode object: %v", err)
	}
	out := make(map[string]int, len(obj))
	for key, val := range obj {
		out[key] = len(val)
	}
	return out
}

func reportFieldProfile(t *testing.T, label string, profile map[string]int) {
	t.Helper()
	t.Logf("%s field byte contributions:", label)
	for key, size := range profile {
		if size > 1024 {
			t.Logf("  %-28s %12d", key, size)
		}
	}
	t.Logf("  %-28s %12d", "TOTAL", sumOfMaps(profile))
}

func TestProfileCurrentFixturesArray(t *testing.T) {
	raw := readExistingSaveBytes(t)
	var full map[string]json.RawMessage
	if err := json.Unmarshal(raw, &full); err != nil {
		t.Fatal(err)
	}
	var fixtures []map[string]json.RawMessage
	if err := json.Unmarshal(full["fixtures"], &fixtures); err != nil {
		t.Fatal(err)
	}
	t.Logf("fixtures count: %d", len(fixtures))
	reportFieldProfile(t, "fixtures[]", fieldSizeProfile(t, full["fixtures"], true))
	var withReport int
	var reportBytes int
	for _, f := range fixtures {
		if r, ok := f["report"]; ok {
			withReport++
			reportBytes += len(r)
		}
	}
	t.Logf("fixtures with report: %d, report bytes total: %d", withReport, reportBytes)
}

func TestProfileCurrentWorldObject(t *testing.T) {
	raw := readExistingSaveBytes(t)
	var full map[string]json.RawMessage
	if err := json.Unmarshal(raw, &full); err != nil {
		t.Fatal(err)
	}
	var world map[string]json.RawMessage
	if err := json.Unmarshal(full["world"], &world); err != nil {
		t.Fatal(err)
	}
	reportFieldProfile(t, "world", fieldSizeProfile(t, full["world"], false))
	var fixtures []map[string]json.RawMessage
	if err := json.Unmarshal(world["fixtures"], &fixtures); err != nil {
		t.Fatal(err)
	}
	t.Logf("world.fixtures count: %d", len(fixtures))
	reportFieldProfile(t, "world.fixtures[]", fieldSizeProfile(t, world["fixtures"], true))
	var withReport int
	var reportBytes int
	for _, f := range fixtures {
		if r, ok := f["report"]; ok {
			withReport++
			reportBytes += len(r)
		}
	}
	t.Logf("world.fixtures with report: %d, report bytes total: %d", withReport, reportBytes)
}
