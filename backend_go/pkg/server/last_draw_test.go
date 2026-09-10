package server

import (
	"net/http"
	"testing"
)

// Second new careers pre-fill the last draw: homes, shuffle flag, from_last.
func TestDefaultHomesRemembersLastDraw(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	first := postNewCareer(t, ts.URL, true, nil)
	firstHomes, _ := first["homes"].(map[string]interface{})
	if len(firstHomes) == 0 {
		t.Fatal("new career should report homes")
	}

	resp, err := http.Get(ts.URL + "/api/career/default-homes")
	if err != nil {
		t.Fatalf("GET default-homes: %v", err)
	}
	var payload map[string]interface{}
	decodeJSONBody(t, resp, &payload)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("default-homes status=%d", resp.StatusCode)
	}
	if payload["from_last"] != true {
		t.Fatalf("default-homes should flag from_last after a career, got %v", payload)
	}
	if payload["shuffle"] != true {
		t.Fatalf("default-homes should remember shuffle=true, got %v", payload)
	}
	gotHomes, _ := payload["homes"].(map[string]interface{})
	if len(gotHomes) != len(firstHomes) {
		t.Fatalf("default-homes should pre-fill last homes (%d), got %d", len(firstHomes), len(gotHomes))
	}
	for k, v := range firstHomes {
		if gotHomes[k] != v {
			t.Fatalf("home mismatch for %s: %v vs %v", k, v, gotHomes[k])
		}
	}

	// Starting unshuffled resets the memory to the default draw, unshuffled.
	second := postNewCareer(t, ts.URL, false, nil)
	_ = second
	resp2, err := http.Get(ts.URL + "/api/career/default-homes")
	if err != nil {
		t.Fatalf("GET default-homes: %v", err)
	}
	var payload2 map[string]interface{}
	decodeJSONBody(t, resp2, &payload2)
	if payload2["shuffle"] != false && payload2["shuffle"] != nil {
		t.Fatalf("default-homes should remember shuffle=false, got %v", payload2["shuffle"])
	}
}
