package persistence

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"football_sim/pkg/matchreport"
)

// The retention policy must keep the career save bounded: full reports for
// the recent window, archival summaries beyond it, and a load that round-trips.
func TestReportRetentionKeepsSaveBounded(t *testing.T) {
	_, ge, tm, te := setupTestWorld(t)

	// Simulate enough weeks that some reports age past the summary window.
	for mw := 1; mw <= 12; mw++ {
		if res := tm.SimulateMatchweek(mw); res["status"] != "success" {
			t.Fatalf("matchweek %d failed: %v", mw, res)
		}
	}

	snap := BuildSnapshot(tm, ge, te)
	if err := ValidateCareerSnapshot(snap); err != nil {
		t.Fatalf("snapshot with archived summaries failed validation: %v", err)
	}

	summarized, full := 0, 0
	var summarizedBytes, fullBytes int
	for i := range snap.Fixtures {
		f := &snap.Fixtures[i]
		if f.Status != "finished" {
			continue
		}
		if f.ReportSummary != nil {
			if f.Report != nil {
				t.Fatalf("fixture %s carries both a report and a summary", f.FixtureID)
			}
			summarized++
			raw, _ := json.Marshal(f)
			summarizedBytes = len(raw)
		} else if f.Report != nil {
			full++
			raw, _ := json.Marshal(f)
			fullBytes = len(raw)
		}
	}
	if summarized == 0 {
		t.Fatal("after 12 matchweeks at least one fixture must be archived to a summary")
	}
	if full == 0 {
		t.Fatal("recent fixtures must keep full reports")
	}
	// Size drill: an archived fixture must be a small fraction of a
	// report-carrying one. Guard against a 10x floor so empty reports
	// cannot trivially pass.
	if summarizedBytes > fullBytes/4 {
		t.Fatalf("archived fixture (%d bytes) is not compact enough vs full report (%d bytes)", summarizedBytes, fullBytes)
	}
	t.Logf("archived fixture: %d bytes; report fixture: %d bytes; archived=%d full=%d",
		summarizedBytes, fullBytes, summarized, full)
}

func TestArchivedSummaryRoundTrip(t *testing.T) {
	_, ge, tm, te := setupTestWorld(t)
	for mw := 1; mw <= 12; mw++ {
		if res := tm.SimulateMatchweek(mw); res["status"] != "success" {
			t.Fatalf("matchweek %d failed: %v", mw, res)
		}
	}

	tempDir := t.TempDir()
	savePath := filepath.Join(tempDir, "retention_career.json")
	if _, err := SaveCareer(tm, ge, te, savePath); err != nil {
		t.Fatalf("SaveCareer failed: %v", err)
	}

	snap, err := LoadCareer(savePath)
	if err != nil {
		t.Fatalf("LoadCareer failed: %v", err)
	}
	if snap.Version != SaveVersion {
		t.Fatalf("snapshot version %d, want %d", snap.Version, SaveVersion)
	}
	var archived int
	for i := range snap.Fixtures {
		if snap.Fixtures[i].ReportSummary != nil {
			archived++
		}
	}
	if archived == 0 {
		t.Fatal("saved snapshot must contain archived summaries")
	}

	// Restore onto a fresh world and re-save: summaries must survive.
	_, freshGE, freshTM, freshTE := setupTestWorld(t)
	if err := RestoreCareer(freshTM, freshGE, freshTE, snap); err != nil {
		t.Fatalf("RestoreCareer failed: %v", err)
	}
	roundSnap := BuildSnapshot(freshTM, freshGE, freshTE)
	if err := ValidateCareerSnapshot(roundSnap); err != nil {
		t.Fatalf("round-trip snapshot failed validation: %v", err)
	}
	var roundArchived int
	for i := range roundSnap.Fixtures {
		if roundSnap.Fixtures[i].ReportSummary != nil {
			roundArchived++
		}
	}
	if roundArchived != archived {
		t.Fatalf("archived summaries lost in round trip: %d -> %d", archived, roundArchived)
	}
}

func TestValidationRejectsReportAndSummaryTogether(t *testing.T) {
	_, ge, tm, te := setupTestWorld(t)
	snap := BuildSnapshot(tm, ge, te)
	if len(snap.Fixtures) == 0 {
		t.Fatal("expected fixtures in snapshot")
	}
	// Force the impossible state: a fixture with both payload forms.
	snap.Fixtures[0].Status = "finished"
	snap.Fixtures[0].Report = &matchreport.MatchReport{HomeGoals: 1}
	snap.Fixtures[0].ReportSummary = &matchreport.ReportSummary{HomeGoals: 1}
	if err := ValidateCareerSnapshot(snap); err == nil {
		t.Fatal("validation must reject a fixture carrying both a report and a summary")
	}
}
