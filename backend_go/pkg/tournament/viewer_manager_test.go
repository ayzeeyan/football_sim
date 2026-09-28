package tournament

import (
	"sort"
	"testing"
)

func testManagerForViewerCareer(t *testing.T) *TournamentManager {
	t.Helper()
	tm := testManagerForWatch(t)
	return tm
}

// sortedClubIDs returns deterministic club IDs for the test world.
func sortedViewerClubIDs(tm *TournamentManager) []string {
	ids := make([]string, 0, len(tm.Clubs))
	for id := range tm.Clubs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// Accepting a job installs the viewer, preserves the previous manager's
// record, and leaves a paper trail in the inbox.
func TestAcceptViewerJobInstallsViewer(t *testing.T) {
	tm := testManagerForViewerCareer(t)
	clubID := sortedViewerClubIDs(tm)[0]
	previous := tm.Managers[clubID].Name

	vm, msg := tm.AcceptViewerJob(clubID, "")
	if msg != "" || vm == nil {
		t.Fatalf("AcceptViewerJob failed: %v", msg)
	}
	// Wire contract: array-typed fields must never be nil (a nil slice
	// marshals as null and crashes the frontend's .length access).
	if vm.Trophies == nil || vm.History == nil {
		t.Fatalf("trophies/history must be non-nil arrays: %+v", vm)
	}
	if got := tm.GetViewerManager(); got == nil || got.Trophies == nil || got.History == nil {
		t.Fatalf("GetViewerManager must return non-nil arrays: %+v", got)
	}
	if vm.Name != ViewerManagerName {
		t.Fatalf("default name=%q want %q", vm.Name, ViewerManagerName)
	}
	mgr := tm.Managers[clubID]
	if mgr == nil || mgr.Name != ViewerManagerName {
		t.Fatalf("viewer not installed: %+v", mgr)
	}
	found := false
	for _, entry := range mgr.History {
		if entry.ManagerName == previous {
			found = true
		}
	}
	if !found {
		t.Fatalf("previous manager %q not preserved: %+v", previous, mgr.History)
	}
	if len(tm.Inbox) == 0 {
		t.Fatal("appointment must reach the inbox")
	}

	// Moving clubs closes the first stint and opens a second.
	other := sortedViewerClubIDs(tm)[1]
	vm, msg = tm.AcceptViewerJob(other, "")
	if msg != "" {
		t.Fatalf("second AcceptViewerJob failed: %v", msg)
	}
	if vm.ClubID != other || len(vm.History) != 2 {
		t.Fatalf("unexpected career after move: %+v", vm)
	}
	if vm.History[0].Outcome != "resigned" || vm.History[0].EndedMatchweek == 0 {
		t.Fatalf("first stint not closed: %+v", vm.History[0])
	}
	if vm.History[1].Outcome != "active" || vm.History[1].EndedMatchweek != 0 {
		t.Fatalf("second stint not active: %+v", vm.History[1])
	}
}

// A sacking closes the stint as sacked and increments the ledger.
func TestRecordViewerSackingClosesStint(t *testing.T) {
	tm := testManagerForViewerCareer(t)
	clubID := sortedViewerClubIDs(tm)[0]
	if _, msg := tm.AcceptViewerJob(clubID, ""); msg != "" {
		t.Fatalf("AcceptViewerJob failed: %v", msg)
	}

	tm.RecordViewerSacking(clubID)
	vm := tm.GetViewerManager()
	if vm.Sackings != 1 {
		t.Fatalf("sackings=%d want 1", vm.Sackings)
	}
	if vm.ClubID != "" {
		t.Fatalf("sacking must clear the active job: %q", vm.ClubID)
	}
	if vm.History[0].Outcome != "sacked" {
		t.Fatalf("stint outcome=%q want sacked", vm.History[0].Outcome)
	}

	// A sacking at another club does not touch the ledger.
	tm.RecordViewerSacking(sortedViewerClubIDs(tm)[1])
	if tm.GetViewerManager().Sackings != 1 {
		t.Fatal("sacking at another club must not increment the viewer ledger")
	}
}

// Trophies are only credited while the viewer is in charge of the club.
func TestRecordViewerTrophiesOnlyInCharge(t *testing.T) {
	tm := testManagerForViewerCareer(t)
	ids := sortedViewerClubIDs(tm)
	club := tm.Clubs[ids[0]]

	// No job: nothing is credited.
	tm.RecordViewerTrophies(club, []string{"Super League"})
	if vm := tm.GetViewerManager(); vm != nil && len(vm.Trophies) != 0 {
		t.Fatalf("trophies credited without a job: %+v", vm.Trophies)
	}

	if _, msg := tm.AcceptViewerJob(ids[0], ""); msg != "" {
		t.Fatalf("AcceptViewerJob failed: %v", msg)
	}
	tm.RecordViewerTrophies(club, []string{"Super League", "Domestic Cup"})
	vm := tm.GetViewerManager()
	if len(vm.Trophies) != 2 {
		t.Fatalf("trophies=%d want 2", len(vm.Trophies))
	}

	// Another club's trophies are not credited to the viewer.
	tm.RecordViewerTrophies(tm.Clubs[ids[1]], []string{"Champions League"})
	if len(tm.GetViewerManager().Trophies) != 2 {
		t.Fatal("another club's trophies must not be credited")
	}
}

// Resigning appoints a deterministic successor and keeps the world running.
func TestResignViewerJobAppointsSuccessor(t *testing.T) {
	tm := testManagerForViewerCareer(t)
	clubID := sortedViewerClubIDs(tm)[0]
	if _, msg := tm.AcceptViewerJob(clubID, ""); msg != "" {
		t.Fatalf("AcceptViewerJob failed: %v", msg)
	}

	vm, msg := tm.ResignViewerJob()
	if msg != "" {
		t.Fatalf("ResignViewerJob failed: %v", msg)
	}
	if vm.ClubID != "" || vm.History[0].Outcome != "resigned" {
		t.Fatalf("unexpected career after resign: %+v", vm)
	}
	successor := tm.Managers[clubID]
	if successor == nil || successor.Name == ViewerManagerName {
		t.Fatalf("no successor appointed: %+v", successor)
	}

	// Resigning with no job is a clean refusal.
	if _, msg := tm.ResignViewerJob(); msg == "" {
		t.Fatal("resigning without a job must fail")
	}
}
