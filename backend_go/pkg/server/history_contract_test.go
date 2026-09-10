package server

import (
	"net/http"
	"testing"
)

// React contract: History and Standings tabs dereference these payloads
// directly (data.past spread, stats.assisters.length, records keys). A bare
// array or a null slice crashes the tab, so pin the shapes here.
func TestServer_HistoryStatsContracts(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	getObj := func(path string) map[string]interface{} {
		t.Helper()
		resp, err := http.Get(ts.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		var payload map[string]interface{}
		decodeJSONBody(t, resp, &payload)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s status=%d", path, resp.StatusCode)
		}
		return payload
	}

	history := getObj("/api/season/history")
	for _, key := range []string{"season_name", "current_matchweek", "max_matchweeks", "current", "table", "past", "trophy_cabinet", "all_time_records"} {
		if _, ok := history[key]; !ok {
			t.Errorf("/api/season/history missing %q", key)
		}
	}
	if past, ok := history["past"].([]interface{}); !ok || past == nil {
		t.Errorf("/api/season/history past must be an array, got %T", history["past"])
	}
	if table, ok := history["table"].([]interface{}); !ok || len(table) != 12 {
		t.Errorf("/api/season/history table should list 12 clubs, got %v", history["table"])
	}
	records, ok := history["all_time_records"].(map[string]interface{})
	if !ok {
		t.Fatalf("all_time_records missing: %v", history["all_time_records"])
	}
	for _, key := range []string{"top_goalscorers", "top_assisters", "highest_scoring_match", "biggest_margin_victory", "single_match_goals_record", "highest_season_points", "wonderkid_milestones"} {
		if _, ok := records[key]; !ok {
			t.Errorf("all_time_records missing %q", key)
		}
	}

	stats := getObj("/api/season/stats")
	for _, key := range []string{"scorers", "assisters", "player_of_the_week", "monthly_awards", "history"} {
		if _, ok := stats[key]; !ok {
			t.Errorf("/api/season/stats missing %q", key)
		}
	}
	if scorers, ok := stats["scorers"].([]interface{}); !ok || scorers == nil {
		t.Errorf("/api/season/stats scorers must be an array, got %T", stats["scorers"])
	}
	if assisters, ok := stats["assisters"].([]interface{}); !ok || assisters == nil {
		t.Errorf("/api/season/stats assisters must be an array, got %T", stats["assisters"])
	}

	trophies := getObj("/api/trophies")
	cabinet, ok := trophies["cabinet"].([]interface{})
	if !ok || len(cabinet) != 12 {
		t.Errorf("/api/trophies cabinet should list 12 clubs, got %v", trophies["cabinet"])
	}

	rec := getObj("/api/records")
	if _, ok := rec["records"].(map[string]interface{}); !ok {
		t.Errorf("/api/records should wrap a records object, got %v", rec["records"])
	}

	gala := getObj("/api/season/awards/ceremony")
	cats, ok := gala["categories"].([]interface{})
	if !ok || len(cats) != 4 {
		t.Fatalf("/api/season/awards/ceremony needs 4 ranked categories, got %v", gala["categories"])
	}
	for _, c := range cats {
		cm, ok := c.(map[string]interface{})
		if !ok {
			t.Fatal("ceremony category must be an object")
		}
		nominees, ok := cm["nominees"].([]interface{})
		if !ok || len(nominees) == 0 {
			t.Errorf("category %v needs nominees", cm["title"])
		}
		if _, ok := cm["winner"].(map[string]interface{}); !ok {
			t.Errorf("category %v needs a winner", cm["title"])
		}
	}

	// Timeline milestones must be narrative achievements (id/title/unlocked),
	// not raw growth-feed rows, or the lab renders blank locked cards.
	var kidID string
	for _, c := range srv.TournamentManager.ClubsList {
		for _, p := range c.Squad {
			if p.UniverseWonderkid {
				kidID = p.PlayerID
				break
			}
		}
		if kidID != "" {
			break
		}
	}
	if kidID == "" {
		t.Skip("no prodigy")
	}
	resp, err := http.Get(ts.URL + "/api/prodigies/" + kidID + "/timeline")
	if err != nil {
		t.Fatalf("GET timeline: %v", err)
	}
	var timeline map[string]interface{}
	decodeJSONBody(t, resp, &timeline)
	miles, ok := timeline["milestones"].([]interface{})
	if !ok || len(miles) != 8 {
		t.Fatalf("timeline needs 8 narrative milestones, got %v", timeline["milestones"])
	}
	for _, m := range miles {
		mm, ok := m.(map[string]interface{})
		if !ok {
			t.Fatal("timeline milestone must be an object")
		}
		for _, key := range []string{"id", "title", "description", "unlocked", "badge"} {
			if _, ok := mm[key]; !ok {
				t.Errorf("timeline milestone missing %q: %v", key, mm)
			}
		}
	}
}
