package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

// Tier B manager career (B5): the viewer accepts a dugout, the world sees
// the viewer as the club's manager, and resigning appoints a successor.
func TestViewerManagerEndpoints(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	if payload := postNewCareer(t, ts.URL, false, nil); payload["status"] != "success" {
		t.Fatalf("fresh world career payload: %v", payload)
	}

	// The career payload lists every club's dugout.
	resp, err := http.Get(ts.URL + "/api/career/manager")
	if err != nil {
		t.Fatalf("GET career manager: %v", err)
	}
	var career struct {
		Manager *struct {
			Name    string `json:"name"`
			ClubID  string `json:"club_id"`
			History []struct {
				ClubID  string `json:"club_id"`
				Outcome string `json:"outcome"`
			} `json:"history"`
		} `json:"manager"`
		Job   map[string]interface{} `json:"job"`
		Clubs []struct {
			ClubID      string `json:"club_id"`
			ManagerName string `json:"manager_name"`
		} `json:"clubs"`
	}
	decodeJSONBody(t, resp, &career)
	if career.Manager != nil {
		t.Fatalf("fresh world must have no viewer manager: %+v", career.Manager)
	}
	if len(career.Clubs) == 0 {
		t.Fatal("career payload must list clubs")
	}
	clubID := career.Clubs[0].ClubID
	previousManager := career.Clubs[0].ManagerName

	// Accepting an unknown club is a clean 404.
	raw, _ := json.Marshal(map[string]interface{}{"club_id": "NO-SUCH-CLUB"})
	resp, err = http.Post(ts.URL+"/api/career/manager/job", "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("POST job unknown club: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown club job status=%d want 404", resp.StatusCode)
	}

	// A missing club_id is refused.
	resp, err = http.Post(ts.URL+"/api/career/manager/job", "application/json", bytes.NewReader([]byte("{}")))
	if err != nil {
		t.Fatalf("POST job empty: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("empty job request status=%d want 400", resp.StatusCode)
	}

	// Accept the job.
	raw, _ = json.Marshal(map[string]interface{}{"club_id": clubID, "name": "Test Gaffer"})
	resp, err = http.Post(ts.URL+"/api/career/manager/job", "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("POST job: %v", err)
	}
	var accepted struct {
		Status  string `json:"status"`
		Manager *struct {
			Name   string `json:"name"`
			ClubID string `json:"club_id"`
		} `json:"manager"`
		Job struct {
			Name        string `json:"name"`
			JobSecurity string `json:"job_security"`
		} `json:"job"`
	}
	decodeJSONBody(t, resp, &accepted)
	if accepted.Status != "success" || accepted.Manager == nil {
		t.Fatalf("accept job failed: %d %+v", resp.StatusCode, accepted)
	}
	if accepted.Manager.Name != "Test Gaffer" || accepted.Manager.ClubID != clubID {
		t.Fatalf("unexpected career after accept: %+v", accepted.Manager)
	}
	if accepted.Job.Name != "Test Gaffer" {
		t.Fatalf("club manager is not the viewer: %+v", accepted.Job)
	}

	// The career payload now shows the active job with security.
	resp, err = http.Get(ts.URL + "/api/career/manager")
	if err != nil {
		t.Fatalf("GET career manager after accept: %v", err)
	}
	decodeJSONBody(t, resp, &career)
	if career.Manager == nil || career.Manager.ClubID != clubID || len(career.Manager.History) != 1 {
		t.Fatalf("career payload lost the job: %+v", career.Manager)
	}
	if career.Manager.History[0].Outcome != "active" {
		t.Fatalf("job record outcome=%q want active", career.Manager.History[0].Outcome)
	}
	if career.Job == nil || career.Job["name"] != "Test Gaffer" {
		t.Fatalf("job security payload missing: %+v", career.Job)
	}

	// The previous manager's record is preserved in the new profile's history.
	srv.worldMu.RLock()
	mgr := srv.TournamentManager.Managers[clubID]
	srv.worldMu.RUnlock()
	if mgr == nil {
		t.Fatal("no manager installed for the club")
	}
	foundPrevious := false
	for _, entry := range mgr.History {
		if entry.ManagerName == previousManager {
			foundPrevious = true
			break
		}
	}
	if !foundPrevious {
		t.Fatalf("previous manager %q not preserved in history: %+v", previousManager, mgr.History)
	}

	// Resigning appoints a deterministic successor.
	resp, err = http.Post(ts.URL+"/api/career/manager/resign", "application/json", bytes.NewReader(nil))
	if err != nil {
		t.Fatalf("POST resign: %v", err)
	}
	var resigned struct {
		Status  string `json:"status"`
		Manager *struct {
			ClubID string `json:"club_id"`
		} `json:"manager"`
	}
	decodeJSONBody(t, resp, &resigned)
	if resigned.Status != "success" || resigned.Manager == nil || resigned.Manager.ClubID != "" {
		t.Fatalf("resign failed: %d %+v", resp.StatusCode, resigned)
	}
	srv.worldMu.RLock()
	successor := srv.TournamentManager.Managers[clubID]
	srv.worldMu.RUnlock()
	if successor == nil || successor.Name == "Test Gaffer" {
		t.Fatalf("resign did not appoint a successor: %+v", successor)
	}

	// Resigning with no job is refused.
	resp, err = http.Post(ts.URL+"/api/career/manager/resign", "application/json", bytes.NewReader(nil))
	if err != nil {
		t.Fatalf("POST resign twice: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("resign without job status=%d want 400", resp.StatusCode)
	}
}
