package persistence

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// Legacy single-file saves (inline clubs map, no club_index) must keep
// loading after the sharded layout became the writer.
func TestLoadLegacySingleFileCareer(t *testing.T) {
	_, ge, tm, te := setupTestWorld(t)
	built := BuildSnapshot(tm, ge, te)

	legacy, err := json.MarshalIndent(built, "", "  ")
	if err != nil {
		t.Fatalf("marshal legacy snapshot: %v", err)
	}
	tempDir := t.TempDir()
	legacyPath := filepath.Join(tempDir, "legacy_career.json")
	if err := os.WriteFile(legacyPath, legacy, 0644); err != nil {
		t.Fatalf("write legacy file: %v", err)
	}

	loaded, err := LoadCareer(legacyPath)
	if err != nil {
		t.Fatalf("LoadCareer(legacy) failed: %v", err)
	}
	if len(loaded.Clubs) != len(built.Clubs) {
		t.Fatalf("legacy load clubs=%d, want %d", len(loaded.Clubs), len(built.Clubs))
	}
	if loaded.SeasonName != built.SeasonName || loaded.CurrentMatchweek != built.CurrentMatchweek {
		t.Fatal("legacy load metadata mismatch")
	}
}

// Sharded saves must skip byte-identical club files on rewrite.
func TestShardedSaveSkipsCleanClubs(t *testing.T) {
	_, ge, tm, te := setupTestWorld(t)
	tempDir := t.TempDir()
	savePath := filepath.Join(tempDir, "career.json")

	if _, err := SaveCareer(tm, ge, te, savePath); err != nil {
		t.Fatalf("SaveCareer failed: %v", err)
	}
	first := map[string][]byte{}
	entries, err := os.ReadDir(filepath.Join(tempDir, "clubs"))
	if err != nil {
		t.Fatalf("read clubs dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no club sidecars written")
	}
	for _, e := range entries {
		raw, err := os.ReadFile(filepath.Join(tempDir, "clubs", e.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		first[e.Name()] = raw
	}

	// Save the untouched world again: every sidecar must be byte-identical
	// (and therefore not rewritten).
	if _, err := SaveCareer(tm, ge, te, savePath); err != nil {
		t.Fatalf("second SaveCareer failed: %v", err)
	}
	for name, raw := range first {
		again, err := os.ReadFile(filepath.Join(tempDir, "clubs", name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if string(again) != string(raw) {
			t.Fatalf("clean club file %s changed on rewrite", name)
		}
	}

	loaded, err := LoadCareer(savePath)
	if err != nil {
		t.Fatalf("LoadCareer(sharded) failed: %v", err)
	}
	if len(loaded.Clubs) != len(tm.Clubs) {
		t.Fatalf("sharded load clubs=%d, want %d", len(loaded.Clubs), len(tm.Clubs))
	}
}
