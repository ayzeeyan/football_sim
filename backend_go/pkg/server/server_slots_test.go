package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"football_sim/pkg/persistence"
)

type slotPayload struct {
	Status string `json:"status"`
	Slot   struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Season    string `json:"season"`
		Matchweek int    `json:"matchweek"`
	} `json:"slot"`
}

func postSlotAction(t *testing.T, tsURL, path string, form url.Values) (map[string]interface{}, int) {
	t.Helper()
	resp, err := http.Post(tsURL+path, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	defer resp.Body.Close()
	var body map[string]interface{}
	if resp.StatusCode == http.StatusOK {
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
	}
	return body, resp.StatusCode
}

// The slot endpoints archive, rename, duplicate, export, import, and delete
// self-contained career snapshots — and a new career never destroys them.
func TestSaveSlotLifecycle(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	if payload := postNewCareer(t, ts.URL, false, nil); payload["status"] != "success" {
		t.Fatalf("fresh world career payload: %v", payload)
	}

	// Create a slot from the current world.
	created, code := postSlotAction(t, ts.URL, "/api/career/slots", url.Values{"name": {"Title run"}})
	if code != http.StatusOK || created["status"] != "success" {
		t.Fatalf("create slot status=%d body=%v", code, created)
	}
	slotID := created["slot"].(map[string]interface{})["id"].(string)
	if created["slot"].(map[string]interface{})["name"] != "Title run" {
		t.Fatalf("slot name not stored: %v", created)
	}

	// List shows it.
	resp, err := http.Get(ts.URL + "/api/career/slots")
	if err != nil {
		t.Fatalf("GET slots: %v", err)
	}
	var listed struct {
		Slots []slotPayload_Slot `json:"slots"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&listed)
	resp.Body.Close()
	if len(listed.Slots) != 1 || listed.Slots[0].ID != slotID {
		t.Fatalf("list must show the created slot: %+v", listed.Slots)
	}

	// Rename.
	renamed, code := postSlotAction(t, ts.URL, "/api/career/slots/"+slotID+"/rename", url.Values{"name": {"Renamed run"}})
	if code != http.StatusOK || renamed["slot"].(map[string]interface{})["name"] != "Renamed run" {
		t.Fatalf("rename failed: %d %v", code, renamed)
	}

	// Duplicate.
	dup, code := postSlotAction(t, ts.URL, "/api/career/slots/"+slotID+"/duplicate", nil)
	if code != http.StatusOK {
		t.Fatalf("duplicate failed: %d %v", code, dup)
	}
	dupID := dup["slot"].(map[string]interface{})["id"].(string)
	if dupID == slotID {
		t.Fatal("duplicate must have a distinct id")
	}

	// Export: attachment whose body is a valid snapshot.
	resp, err = http.Get(ts.URL + "/api/career/slots/" + slotID + "/export")
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("export status=%d", resp.StatusCode)
	}
	if !strings.Contains(resp.Header.Get("Content-Disposition"), "attachment") {
		t.Fatal("export must be an attachment")
	}
	exported, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("read export: %v", err)
	}
	var snap persistence.CareerSnapshot
	if err := json.Unmarshal(exported, &snap); err != nil {
		t.Fatalf("export is not a career snapshot: %v", err)
	}
	if err := persistence.ValidateCareerSnapshot(&snap); err != nil {
		t.Fatalf("exported snapshot fails validation: %v", err)
	}

	// Import the exported snapshot back as a new slot.
	importResp, err := http.Post(ts.URL+"/api/career/slots/import?name=Reimported", "application/json", bytes.NewReader(exported))
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	var imported map[string]interface{}
	_ = json.NewDecoder(importResp.Body).Decode(&imported)
	importResp.Body.Close()
	if importResp.StatusCode != http.StatusOK || imported["status"] != "success" {
		t.Fatalf("import failed: %d %v", importResp.StatusCode, imported)
	}

	// A new career must not touch the slot archives.
	if payload := postNewCareer(t, ts.URL, false, nil); payload["status"] != "success" {
		t.Fatalf("second career payload: %v", payload)
	}
	resp, err = http.Get(ts.URL + "/api/career/slots")
	if err != nil {
		t.Fatalf("GET slots after new career: %v", err)
	}
	var after struct {
		Slots []slotPayload_Slot `json:"slots"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&after)
	resp.Body.Close()
	if len(after.Slots) != 3 {
		t.Fatalf("slots must survive a new career, got %d: %+v", len(after.Slots), after.Slots)
	}

	// Delete all three; deleting a missing slot is a clean 404.
	for _, s := range after.Slots {
		if _, code := postSlotAction(t, ts.URL, "/api/career/slots/"+s.ID+"/delete", nil); code != http.StatusOK {
			t.Fatalf("delete %s status=%d", s.ID, code)
		}
	}
	if _, code := postSlotAction(t, ts.URL, "/api/career/slots/"+slotID+"/delete", nil); code != http.StatusNotFound {
		t.Fatalf("double delete status=%d want 404", code)
	}

	// Invalid slot ids are rejected before any filesystem use.
	if _, code := postSlotAction(t, ts.URL, "/api/career/slots/..%2F..%2Fetc/rename", url.Values{"name": {"x"}}); code != http.StatusBadRequest {
		t.Fatalf("path traversal status=%d want 400", code)
	}
}

type slotPayload_Slot struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Season    string `json:"season"`
	Matchweek int    `json:"matchweek"`
}

// Import rejects files that are not career snapshots.
func TestImportSlotRejectsGarbage(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	resp, err := http.Post(ts.URL+"/api/career/slots/import", "application/json", strings.NewReader("{not a snapshot"))
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("garbage import status=%d want 400", resp.StatusCode)
	}
}
