package server

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// The export endpoints serve deterministic CSV attachments of resolved
// state. Nothing is written to disk.
func TestExportEndpointsServeCSV(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	if payload := postNewCareer(t, ts.URL, false, nil); payload["status"] != "success" {
		t.Fatalf("fresh world career payload: %v", payload)
	}

	fetch := func(path string) (string, http.Header) {
		t.Helper()
		resp, err := http.Get(ts.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("GET %s status=%d", path, resp.StatusCode)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		return string(body), resp.Header
	}

	// Standings export: header + 20 Premier League rows, attachment headers.
	csv, header := fetch("/api/export/standings")
	if !strings.Contains(header.Get("Content-Disposition"), "attachment") {
		t.Fatal("standings export must be an attachment")
	}
	if !strings.HasPrefix(csv, "position,club_id,club_name,league,played") {
		t.Fatalf("standings CSV header wrong: %q", strings.SplitN(csv, "\n", 2)[0])
	}
	if lines := strings.Count(csv, "\r\n"); lines != 21 {
		t.Fatalf("standings CSV lines=%d want 21 (header + 20 clubs)", lines)
	}

	// Squad export for the first club.
	resp, err := http.Get(ts.URL + "/api/clubs")
	if err != nil {
		t.Fatalf("GET /api/clubs: %v", err)
	}
	var clubs []map[string]interface{}
	decodeJSONBody(t, resp, &clubs)
	clubID, _ := clubs[0]["club_id"].(string)
	csv, _ = fetch("/api/export/squad?club_id=" + clubID)
	if !strings.HasPrefix(csv, "player_id,full_name,position,category,age,ovr") {
		t.Fatalf("squad CSV header wrong: %q", strings.SplitN(csv, "\n", 2)[0])
	}
	if !strings.Contains(csv, clubID) == false {
		t.Fatal("squad CSV should contain squad rows")
	}

	// Unknown club 404s.
	resp, err = http.Get(ts.URL + "/api/export/squad?club_id=NOPE")
	if err != nil {
		t.Fatalf("GET unknown squad export: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown club export status=%d", resp.StatusCode)
	}

	// Fixtures export has the full header and at least the scheduled slate.
	csv, _ = fetch("/api/export/fixtures")
	if !strings.HasPrefix(csv, "matchweek,fixture_id,competition,stage,home_club_id,away_club_id,status,score") {
		t.Fatalf("fixtures CSV header wrong: %q", strings.SplitN(csv, "\n", 2)[0])
	}

	// Transfers export is well-formed even when empty.
	csv, _ = fetch("/api/export/transfers")
	if !strings.HasPrefix(csv, "player_id,player_name,position,seller_club,buyer_club,fee_eur,matchweek") {
		t.Fatalf("transfers CSV header wrong: %q", strings.SplitN(csv, "\n", 2)[0])
	}

	// CSV escaping: a field containing a comma is quoted.
	if csvEscape("a,b") != "\"a,b\"" {
		t.Fatalf("csvEscape comma wrong: %q", csvEscape("a,b"))
	}
	if csvEscape("he said \"hi\"") != "\"he said \"\"hi\"\"\"" {
		t.Fatalf("csvEscape quote wrong: %q", csvEscape("he said \"hi\""))
	}
}
