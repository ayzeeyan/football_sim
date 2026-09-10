package server

import (
	"testing"

	"football_sim/pkg/persistence"
)

func TestDiagnosticsSnapshotTracksLogicalWorldState(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer srv.Stop()
	defer ts.Close()

	before := srv.Diagnostics()
	if before.SaveVersion != persistence.SaveVersion {
		t.Fatalf("save version=%d, want %d", before.SaveVersion, persistence.SaveVersion)
	}
	if before.TotalClubs != 12 || before.TotalPlayers == 0 || before.TotalManagers == 0 {
		t.Fatalf("incomplete diagnostics: %+v", before)
	}
	if before.SeasonPhase != "season" || before.CurrentMatchweek != 1 {
		t.Fatalf("unexpected initial diagnostics: %+v", before)
	}

	postMacro(t, ts.URL, "/api/sim/week")
	after := srv.Diagnostics()
	if after.CurrentMatchweek != 2 {
		t.Fatalf("diagnostics did not observe macro progress: %+v", after)
	}
	if after.CompletedFixtureCount <= before.CompletedFixtureCount || after.FixturesRemaining >= before.FixturesRemaining {
		t.Fatalf("fixture diagnostics did not move after simulated week: before=%+v after=%+v", before, after)
	}
}
