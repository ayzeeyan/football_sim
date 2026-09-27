package server

import (
	"encoding/json"
	"net/http"
	"testing"
)

// The what-if endpoint is a read-only sandbox: it must answer for scheduled
// and finished fixtures alike, never change the recorded state, and stay
// deterministic for a given scratch seed.
func TestWhatIfEndpointIsReadOnlySandbox(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	if payload := postNewCareer(t, ts.URL, false, nil); payload["status"] != "success" {
		t.Fatalf("fresh world career payload: %v", payload)
	}

	fixtures := struct {
		Fixtures []struct {
			FixtureID string `json:"fixture_id"`
			Status    string `json:"status"`
		} `json:"fixtures"`
	}{}
	resp, err := http.Get(ts.URL + "/api/fixtures")
	if err != nil {
		t.Fatalf("GET /api/fixtures: %v", err)
	}
	decodeJSONBody(t, resp, &fixtures)
	if len(fixtures.Fixtures) == 0 {
		t.Fatal("no fixtures on the opening slate")
	}
	fid := fixtures.Fixtures[0].FixtureID

	fetchWhatIf := func(url string) (map[string]interface{}, int) {
		t.Helper()
		r, err := http.Get(url)
		if err != nil {
			t.Fatalf("GET whatif: %v", err)
		}
		defer r.Body.Close()
		var body map[string]interface{}
		if r.StatusCode == http.StatusOK {
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode whatif: %v", err)
			}
		}
		return body, r.StatusCode
	}

	before, code := fetchWhatIf(ts.URL + "/api/fixtures/" + fid + "/whatif?seed=777")
	if code != http.StatusOK {
		t.Fatalf("whatif status=%d body=%v", code, before)
	}
	if before["fixture_id"] != fid {
		t.Fatalf("whatif fixture_id=%v want %s", before["fixture_id"], fid)
	}
	hypo, ok := before["hypothetical"].(map[string]interface{})
	if !ok {
		t.Fatalf("hypothetical scoreline missing: %v", before)
	}
	if _, ok := hypo["home_goals"]; !ok {
		t.Fatalf("hypothetical home goals missing: %v", hypo)
	}
	table, ok := before["table"].(map[string]interface{})
	if !ok || table["applicable"] != true {
		t.Fatalf("league fixture must report table movement: %v", before["table"])
	}
	if _, ok := before["actual"]; ok {
		t.Fatal("scheduled fixture must not report an actual result")
	}

	again, _ := fetchWhatIf(ts.URL + "/api/fixtures/" + fid + "/whatif?seed=777")
	b1, _ := json.Marshal(before)
	b2, _ := json.Marshal(again)
	if string(b1) != string(b2) {
		t.Fatal("same scratch seed produced a different what-if answer")
	}

	// Read-only: the fixture payload is byte-identical after the sandbox.
	fixtureBefore := getFixturePayload(t, ts.URL, fid)
	fetchWhatIf(ts.URL + "/api/fixtures/" + fid + "/whatif?seed=1234")
	fixtureAfter := getFixturePayload(t, ts.URL, fid)
	if fixtureBefore != fixtureAfter {
		t.Fatal("what-if endpoint changed the recorded fixture")
	}

	// Unknown fixture id is a clean 404.
	if _, code := fetchWhatIf(ts.URL + "/api/fixtures/nope/whatif"); code != http.StatusNotFound {
		t.Fatalf("unknown fixture status=%d want 404", code)
	}
}

func getFixturePayload(t *testing.T, tsURL, fid string) string {
	t.Helper()
	resp, err := http.Get(tsURL + "/api/fixtures/" + fid)
	if err != nil {
		t.Fatalf("GET fixture: %v", err)
	}
	defer resp.Body.Close()
	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	out, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	return string(out)
}
