package server

import (
	"os"
	"path/filepath"
	"testing"

	"football_sim/pkg/persistence"
)

// clubFileBytes snapshots every club sidecar's bytes for later comparison.
func clubFileBytes(t *testing.T, savePath string) map[string][]byte {
	t.Helper()
	out := map[string][]byte{}
	dir := filepath.Join(filepath.Dir(savePath), "clubs")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read clubs dir: %v", err)
	}
	for _, e := range entries {
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("read club file %s: %v", e.Name(), err)
		}
		out[e.Name()] = raw
	}
	return out
}

// F5: full time of one live match must not rewrite untouched squad files.
func TestLiveFullTimeRewritesOnlyPlayedClubs(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	// Baseline sharded save.
	srv.worldMu.Lock()
	snap, gen := srv.takeCareerSnapshotLocked()
	srv.worldMu.Unlock()
	srv.commitCareerSnapshot(snap, gen)
	before := clubFileBytes(t, srv.savePath)
	if len(before) != 12 {
		t.Fatalf("expected 12 club files, got %d", len(before))
	}

	// Play one live match to full time through the real commit path.
	srv.worldMu.Lock()
	mw := srv.TournamentManager.CurrentMatchweek
	var targetID, homeID, awayID string
	for _, f := range srv.TournamentManager.GetSlate(mw) {
		if f.Status == "scheduled" {
			targetID, homeID, awayID = f.FixtureID, f.HomeID, f.AwayID
			break
		}
	}
	if targetID == "" {
		srv.worldMu.Unlock()
		t.Fatal("no scheduled fixture for the live match")
	}
	home := srv.TournamentManager.Clubs[homeID]
	away := srv.TournamentManager.Clubs[awayID]
	srv.LiveMatchEngine.SetClubs(home, away,
		srv.TournamentManager.Managers[homeID], srv.TournamentManager.Managers[awayID])
	srv.liveFixtureID = targetID
	srv.LiveMatchEngine.Kickoff()
	srv.LiveMatchEngine.SetSpeed(999)
	srv.LiveMatchEngine.Tick(1.0)
	if srv.LiveMatchEngine.State != "FULL_TIME" {
		srv.worldMu.Unlock()
		t.Fatalf("engine did not reach full time: %s", srv.LiveMatchEngine.State)
	}
	result := srv.TournamentManager.CommitLiveFixtureByID(targetID, srv.LiveMatchEngine)
	if recorded, _ := result["recorded"].(bool); !recorded {
		srv.worldMu.Unlock()
		t.Fatalf("live fixture was not recorded: %v", result)
	}
	snap2, gen2 := srv.takeCareerSnapshotLocked()
	srv.worldMu.Unlock()
	srv.commitCareerSnapshot(snap2, gen2)

	after := clubFileBytes(t, srv.savePath)
	changed := []string{}
	for name, rawBefore := range before {
		rawAfter, ok := after[name]
		if !ok {
			t.Fatalf("club file %s disappeared after save", name)
		}
		if string(rawAfter) != string(rawBefore) {
			changed = append(changed, name)
		}
	}
	if len(after) != len(before) {
		t.Fatalf("club file count changed: %d -> %d", len(before), len(after))
	}
	// Only the two participating squads may change.
	if len(changed) != 2 {
		t.Fatalf("expected exactly 2 rewritten club files, got %d: %v", len(changed), changed)
	}

	// The sharded save must round-trip with the finished result intact.
	saved, err := persistence.LoadCareer(srv.savePath)
	if err != nil {
		t.Fatalf("load sharded career: %v", err)
	}
	if len(saved.Clubs) != 12 {
		t.Fatalf("loaded %d clubs, want 12", len(saved.Clubs))
	}
	var finished *struct {
		Home, Away int
	}
	for _, f := range saved.Fixtures {
		if f.FixtureID == targetID {
			if f.Status != "finished" || f.HomeGoals == nil || f.AwayGoals == nil {
				t.Fatalf("finished fixture did not persist: %+v", f)
			}
			finished = &struct {
				Home, Away int
			}{*f.HomeGoals, *f.AwayGoals}
		}
	}
	if finished == nil {
		t.Fatal("live fixture missing from reloaded career")
	}
}
