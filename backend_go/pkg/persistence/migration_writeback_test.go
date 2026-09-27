package persistence

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMaybeWriteMigratedCareerWritesOlderVersionImmediately(t *testing.T) {
	_, ge, tm, te := setupTestWorld(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "career.json")
	if _, err := SaveCareer(tm, ge, te, path); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]json.RawMessage
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	manifest["version"] = json.RawMessage("6")
	downgraded, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, downgraded, 0644); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadCareer(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Version != 6 {
		t.Fatalf("fixture version=%d want 6", loaded.Version)
	}
	if err := RestoreCareer(tm, ge, te, loaded); err != nil {
		t.Fatal(err)
	}
	wrote, err := MaybeWriteMigratedCareer(tm, ge, te, path, loaded.Version)
	if err != nil {
		t.Fatalf("migration write failed: %v", err)
	}
	if !wrote {
		t.Fatal("expected eager write of upgraded save")
	}
	reloaded, err := LoadCareer(path)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Version != SaveVersion {
		t.Fatalf("disk version=%d want %d", reloaded.Version, SaveVersion)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	wroteAgain, err := MaybeWriteMigratedCareer(tm, ge, te, path, reloaded.Version)
	if err != nil {
		t.Fatal(err)
	}
	if wroteAgain {
		t.Fatal("current-version save was rewritten without a migration")
	}
	info2, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info2.ModTime().After(info.ModTime()) && info2.Size() != info.Size() {
		t.Fatal("current-version save changed on disk")
	}
}

func TestMaybeWriteMigratedCareerPreservesOriginalOnValidationFailure(t *testing.T) {
	_, ge, tm, te := setupTestWorld(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "career.json")
	if _, err := SaveCareer(tm, ge, te, path); err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(tm.ClubsList) == 0 || len(tm.ClubsList[0].Squad) == 0 || tm.ClubsList[0].Squad[0] == nil {
		t.Fatal("expected a playable club")
	}
	tm.ClubsList[0].Squad[0].ContractYears = -4
	wrote, err := MaybeWriteMigratedCareer(tm, ge, te, path, SaveVersion-1)
	if err == nil || wrote {
		t.Fatal("corrupt migration must not overwrite the original save")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(original) {
		t.Fatal("original save was destroyed by a failed migration")
	}
}

func TestShardedMigrationKeepsManifestAndClubIndexConsistent(t *testing.T) {
	_, ge, tm, te := setupTestWorld(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "career.json")
	if _, err := SaveCareer(tm, ge, te, path); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]json.RawMessage
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	if _, ok := manifest[clubIndexKey]; !ok {
		t.Fatal("expected sharded club index")
	}
	manifest["version"] = json.RawMessage("6")
	downgraded, _ := json.Marshal(manifest)
	if err := os.WriteFile(path, downgraded, 0644); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadCareer(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := RestoreCareer(tm, ge, te, loaded); err != nil {
		t.Fatal(err)
	}
	if _, err := MaybeWriteMigratedCareer(tm, ge, te, path, 6); err != nil {
		t.Fatal(err)
	}
	upgraded, err := LoadCareer(path)
	if err != nil {
		t.Fatal(err)
	}
	if upgraded.Version != SaveVersion {
		t.Fatalf("upgraded version=%d", upgraded.Version)
	}
	if len(upgraded.Clubs) != len(tm.Clubs) {
		t.Fatalf("club shards=%d live=%d", len(upgraded.Clubs), len(tm.Clubs))
	}
	raw2, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var manifest2 map[string]json.RawMessage
	if err := json.Unmarshal(raw2, &manifest2); err != nil {
		t.Fatal(err)
	}
	if _, ok := manifest2["clubs"]; ok {
		t.Fatal("manifest should not inline clubs after sharded write")
	}
	if _, ok := manifest2[clubIndexKey]; !ok {
		t.Fatal("upgraded manifest missing club index")
	}
}
