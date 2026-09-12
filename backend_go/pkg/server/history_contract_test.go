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
		if err != nil { t.Fatalf("GET %s: %v", path, err) }
		var payload map[string]interface{}
		decodeJSONBody(t, resp, &payload)
		if resp.StatusCode != http.StatusOK { t.Fatalf("%s status=%d", path, resp.StatusCode) }
		return payload
	}

	history := getObj("/api/season/history")
	for _, key := range []string{"season_name", "current_matchweek", "max_matchweeks", "current", "table", "past", "trophy_cabinet", "all_time_records"} {
		if _, ok := history[key]; !ok { t.Errorf("/api/season/history missing %q", key) }
	}
	if past, ok := history["past"].([]interface{}); !ok || past == nil { t.Errorf("/api/season/history past must be an array, got %T", history["past"]) }
	if table, ok := history["table"].([]interface{}); !ok || len(table) != 12 { t.Errorf("/api/season/history table should list 12 clubs, got %v", history["table"]) }
	records, ok := history["all_time_records"].(map[string]interface{})
	if !ok { t.Fatalf("all_time_records missing: %v", history["all_time_records"]) }
	for _, key := range []string{"top_goalscorers", "top_assisters", "highest_scoring_match", "biggest_margin_victory", "single_match_goals_record", "highest_season_points", "wonderkid_milestones"} {
		if _, ok := records[key]; !ok { t.Errorf("all_time_records missing %q", key) }
	}

	stats := getObj("/api/season/stats")
	for _, key := range []string{"scorers", "assisters", "player_of_the_week", "monthly_awards", "history"} {
		if _, ok := stats[key]; !ok { t.Errorf("/api/season/stats missing %q", key) }
	}
	if scorers, ok := stats["scorers"].([]interface{}); !ok || scorers == nil { t.Errorf("/api/season/stats scorers must be an array, got %T", stats["scorers"]) }
	if assisters, ok := stats["assisters"].([]interface{}); !ok || assisters == nil { t.Errorf("/api/season/stats assisters must be an array, got %T", stats["assisters"]) }

	trophies := getObj("/api/trophies")
	cabinet, ok := trophies["cabinet"].([]interface{})
	if !ok || len(cabinet) != 12 { t.Errorf("/api/trophies cabinet should list 12 clubs, got %v", trophies["cabinet"]) }
	rec := getObj("/api/records")
	if _, ok := rec["records"].(map[string]interface{}); !ok { t.Errorf("/api/records should wrap a records object, got %v", rec["records"]) }

	// Mid-season ceremony request must return HTTP 409 Conflict
	respCeremonyMid, err := http.Get(ts.URL + "/api/season/awards/ceremony")
	if err != nil { t.Fatalf("GET ceremony mid-season: %v", err) }
	if respCeremonyMid.StatusCode != http.StatusConflict {
		t.Fatalf("expected HTTP 409 Conflict mid-season for /api/season/awards/ceremony, got %d", respCeremonyMid.StatusCode)
	}
	var errBody map[string]interface{}
	decodeJSONBody(t, respCeremonyMid, &errBody)
	if _, ok := errBody["error"]; !ok {
		t.Fatalf("expected error key in 409 body, got %#v", errBody)
	}

	// Advance season to completion so ceremony is ready
	srv.worldMu.Lock()
	srv.TournamentManager.CurrentMatchweek = srv.TournamentManager.MaxMatchweeks + 1
	srv.worldMu.Unlock()

	gala := getObj("/api/season/awards/ceremony")
	cats, ok := gala["categories"].([]interface{})
	if !ok || len(cats) != 5 { t.Fatalf("/api/season/awards/ceremony needs 5 ranked categories, got %v", gala["categories"]) }
	for _, c := range cats {
		cm, ok := c.(map[string]interface{})
		if !ok { t.Fatal("ceremony category must be an object") }
		nominees, ok := cm["nominees"].([]interface{})
		if !ok || len(nominees) == 0 { t.Errorf("category %v needs nominees", cm["title"]) }
		winner, ok := cm["winner"].(map[string]interface{})
		if !ok { t.Errorf("category %v needs a winner", cm["title"]); continue }
		winnerID, ok := cm["winner_id"].(string)
		if !ok || winnerID == "" { t.Errorf("category %v needs explicit winner_id", cm["title"]); continue }
		if winner["player_id"] != winnerID { t.Errorf("category %v winner object/id mismatch: %v vs %v", cm["title"], winner["player_id"], winnerID) }
	}

	// Verify team_of_the_season and manager_of_the_year in gala
	tots, ok := gala["team_of_the_season"].(map[string]interface{})
	if !ok || tots == nil {
		t.Fatalf("gala missing team_of_the_season: %#v", gala["team_of_the_season"])
	}
	xi, ok := tots["xi"].([]interface{})
	if !ok || len(xi) != 11 {
		t.Fatalf("tots xi needs 11 players, got %v", tots["xi"])
	}
	moty, ok := gala["manager_of_the_year"].(map[string]interface{})
	if !ok || moty == nil {
		t.Fatalf("gala missing manager_of_the_year: %#v", gala["manager_of_the_year"])
	}
	if moty["name"] == "" || moty["club_id"] == "" {
		t.Fatalf("invalid moty: %#v", moty)
	}

	// Post-rollover ceremony request must return HTTP 409 Conflict
	srv.worldMu.Lock()
	srv.TournamentManager.ResetNewSeason()
	srv.worldMu.Unlock()

	respCeremonyPost, err := http.Get(ts.URL + "/api/season/awards/ceremony")
	if err != nil { t.Fatalf("GET ceremony post-rollover: %v", err) }
	if respCeremonyPost.StatusCode != http.StatusConflict {
		t.Fatalf("expected HTTP 409 Conflict post-rollover for /api/season/awards/ceremony, got %d", respCeremonyPost.StatusCode)
	}

	var kidID string
	for _, c := range srv.TournamentManager.ClubsList {
		for _, p := range c.Squad { if p.UniverseWonderkid { kidID = p.PlayerID; break } }
		if kidID != "" { break }
	}
	if kidID == "" { t.Skip("no prodigy") }
	resp, err := http.Get(ts.URL + "/api/prodigies/" + kidID + "/timeline")
	if err != nil { t.Fatalf("GET timeline: %v", err) }
	var timeline map[string]interface{}
	decodeJSONBody(t, resp, &timeline)
	miles, ok := timeline["milestones"].([]interface{})
	if !ok || len(miles) != 8 { t.Fatalf("timeline needs 8 narrative milestones, got %v", timeline["milestones"]) }
	for _, m := range miles {
		mm, ok := m.(map[string]interface{})
		if !ok { t.Fatal("timeline milestone must be an object") }
		for _, key := range []string{"id", "title", "description", "unlocked", "badge"} { if _, ok := mm[key]; !ok { t.Errorf("timeline milestone missing %q: %v", key, mm) } }
	}
}
