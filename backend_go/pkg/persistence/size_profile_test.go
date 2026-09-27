package persistence

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func readExistingSaveBytes(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "..", "saves", "career.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("saves/career.json not readable: %v", err)
	}
	return raw
}

func profileTopLevel(t *testing.T, raw []byte) map[string]int {
	t.Helper()
	var full map[string]json.RawMessage
	if err := json.Unmarshal(raw, &full); err != nil {
		t.Fatalf("existing save is not valid JSON: %v", err)
	}
	out := make(map[string]int, len(full))
	for key, val := range full {
		out[key] = len(val)
	}
	return out
}

func sumOfMaps(maps ...map[string]int) int {
	total := 0
	for _, m := range maps {
		for _, v := range m {
			total += v
		}
	}
	return total
}

func reportTopLevelProfile(t *testing.T, label string, profile map[string]int) {
	t.Helper()
	t.Logf("%s top-level JSON byte contributions:", label)
	for key, size := range profile {
		if size > 1024 {
			t.Logf("  %-28s %12d", key, size)
		}
	}
	t.Logf("  %-28s %12d", "TOTAL", sumOfMaps(profile))
}

func dirStats(t *testing.T, dir string) (int, int64) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	var total int64
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			t.Fatalf("stat %s: %v", e.Name(), err)
		}
		total += info.Size()
	}
	return len(entries), total
}

func TestMeasureCurrentSaveTopLevel(t *testing.T) {
	raw := readExistingSaveBytes(t)
	reportTopLevelProfile(t, "current manifest", profileTopLevel(t, raw))
	count, total := dirStats(t, filepath.Join("..", "..", "..", "saves", "clubs"))
	t.Logf("manifest file size: %d", len(raw))
	t.Logf("club sidecars: %d files, %d bytes", count, total)
}

func TestMeasureFreshSaveTopLevel(t *testing.T) {
	_, ge, tm, te := setupTestWorld(t)
	_ = tm.SimulateMatchweek(1)
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "fresh_career.json")
	if _, err := SaveCareer(tm, ge, te, path); err != nil {
		t.Fatalf("SaveCareer failed: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fresh save: %v", err)
	}
	reportTopLevelProfile(t, "fresh save", profileTopLevel(t, raw))
	count, total := dirStats(t, filepath.Join(tempDir, "clubs"))
	t.Logf("fresh manifest file size: %d", len(raw))
	t.Logf("fresh club sidecars: %d files, %d bytes", count, total)
}
