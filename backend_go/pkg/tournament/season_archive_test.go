package tournament

import (
	"encoding/json"
	"testing"
)

// At season end every finished fixture's full report is replaced by its
// archival summary, so the offseason never carries the season's match
// payloads through the transfer window.
func TestSeasonEndArchivesAllReports(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	batch := tm.SimulateBatchWeeks(tm.MaxMatchweeks)
	if batch.Status != "success" || !batch.SeasonFinished {
		t.Fatalf("season did not finish: %+v", batch)
	}
	if tm.SeasonPhase != "transfer_window" {
		t.Fatalf("phase=%q want transfer_window", tm.SeasonPhase)
	}
	check := func(list []Fixture, label string) {
		full, summarized, finished := 0, 0, 0
		for i := range list {
			f := &list[i]
			if f.Status != "finished" {
				continue
			}
			finished++
			if f.Report != nil {
				full++
			}
			if f.ReportSummary != nil {
				summarized++
			}
		}
		if full != 0 {
			t.Fatalf("%s: %d finished fixtures still carry full reports", label, full)
		}
		if summarized != finished {
			t.Fatalf("%s: %d/%d finished fixtures have summaries", label, summarized, finished)
		}
		t.Logf("%s: %d finished fixtures, all summarized", label, finished)
	}
	check(tm.Fixtures, "domestic")
	check(tm.UCLFixtures, "ucl")
	check(tm.SuperCupFixtures, "super cup")
	if tm.World != nil {
		check(tm.World.Fixtures, "world")
	}
}

// The wire shape keeps the summary and never a nil-slice null: a serialized
// finished fixture carries report == nil and report_summary != nil.
func TestArchivedFixtureWireShape(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	batch := tm.SimulateBatchWeeks(tm.MaxMatchweeks)
	if batch.Status != "success" || !batch.SeasonFinished {
		t.Fatalf("season did not finish: %+v", batch)
	}
	var archived *Fixture
	for i := range tm.Fixtures {
		f := &tm.Fixtures[i]
		if f.Status == "finished" && f.ReportSummary != nil {
			archived = f
			break
		}
	}
	if archived == nil {
		t.Fatal("no archived fixture found after season end")
	}
	raw, err := json.Marshal(archived)
	if err != nil {
		t.Fatalf("marshal archived fixture: %v", err)
	}
	var wire map[string]json.RawMessage
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatalf("unmarshal archived fixture: %v", err)
	}
	if _, ok := wire["report"]; ok {
		t.Fatal("archived fixture must not carry a report key")
	}
	if string(wire["report_summary"]) == "null" {
		t.Fatal("archived fixture summary must never be null")
	}
}

// The in-season retention windows are tight: beyond the archive window a
// finished fixture keeps at most a compacted report, and beyond the summary
// window only the summary.
func TestInSeasonRetentionWindows(t *testing.T) {
	if reportArchiveKeepWeeks != 2 || reportSummaryKeepWeeks != 6 {
		t.Fatalf("retention windows: archive=%d summary=%d, want 2/6", reportArchiveKeepWeeks, reportSummaryKeepWeeks)
	}
	tm, _, _ := loadEuropeanWorldForTest(t)
	// Simulate well past the summary window.
	batch := tm.SimulateBatchWeeks(reportSummaryKeepWeeks + 3)
	if batch.Status != "success" {
		t.Fatalf("simulation failed: %+v", batch)
	}
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	full, compactedOrSummary, summaryOnly := 0, 0, 0
	for i := range tm.Fixtures {
		f := &tm.Fixtures[i]
		if f.Status != "finished" {
			continue
		}
		switch {
		case f.Report != nil && f.Matchweek >= tm.CurrentMatchweek-reportArchiveKeepWeeks:
			full++
		case f.Report != nil:
			compactedOrSummary++
		case f.ReportSummary != nil:
			summaryOnly++
		}
	}
	if full == 0 || summaryOnly == 0 {
		t.Fatalf("retention tiers empty: full=%d compacted=%d summary=%d", full, compactedOrSummary, summaryOnly)
	}
	t.Logf("retention tiers: full=%d compacted=%d summary=%d", full, compactedOrSummary, summaryOnly)
}
