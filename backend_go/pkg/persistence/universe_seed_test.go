package persistence

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUniverseSeedPersistsAcrossReload(t *testing.T) {
	savePath := filepath.Join(t.TempDir(), "career.json")
	first, err := LoadOrCreateUniverseSeed(savePath)
	if err != nil {
		t.Fatalf("create seed: %v", err)
	}
	if first == 0 {
		t.Fatal("generated zero universe seed")
	}
	second, err := LoadOrCreateUniverseSeed(savePath)
	if err != nil {
		t.Fatalf("reload seed: %v", err)
	}
	if first != second {
		t.Fatalf("persisted universe seed changed: %d -> %d", first, second)
	}
}

func TestLegacyCareerWithoutSeedUsesStableCompatibilitySeed(t *testing.T) {
	savePath := filepath.Join(t.TempDir(), "career.json")
	if err := os.WriteFile(savePath, []byte("{}"), 0644); err != nil {
		t.Fatalf("write legacy save: %v", err)
	}
	seed, err := LoadOrCreateUniverseSeed(savePath)
	if err != nil {
		t.Fatalf("legacy seed: %v", err)
	}
	if seed != legacyUniverseSeed {
		t.Fatalf("legacy seed=%d, want %d", seed, legacyUniverseSeed)
	}
	if _, err := os.Stat(UniverseSeedPath(savePath)); err != nil {
		t.Fatalf("legacy compatibility seed was not persisted: %v", err)
	}
}

func TestMalformedUniverseSeedIsRejected(t *testing.T) {
	savePath := filepath.Join(t.TempDir(), "career.json")
	if err := os.WriteFile(UniverseSeedPath(savePath), []byte("not-a-seed\n"), 0644); err != nil {
		t.Fatalf("write malformed seed: %v", err)
	}
	if _, err := LoadOrCreateUniverseSeed(savePath); err == nil {
		t.Fatal("expected malformed universe seed to be rejected")
	}
}
