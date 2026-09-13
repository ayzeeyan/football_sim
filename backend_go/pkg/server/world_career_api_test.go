package server

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestFreshCareerExposesWorldCompetitionHub(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	payload := postNewCareer(t, ts.URL, false, nil)
	if payload["status"] != "success" || payload["max_matchweeks"].(float64) != 38 {
		t.Fatalf("fresh world career payload: %v", payload)
	}

	resp, err := http.Get(ts.URL + "/api/clubs")
	if err != nil {
		t.Fatalf("GET /api/clubs: %v", err)
	}
	var clubs []map[string]interface{}
	decodeJSONBody(t, resp, &clubs)
	if resp.StatusCode != http.StatusOK || len(clubs) != 96 {
		t.Fatalf("fresh career clubs=%d status=%d want 96/200", len(clubs), resp.StatusCode)
	}

	resp, err = http.Get(ts.URL + "/api/competitions")
	if err != nil {
		t.Fatalf("GET /api/competitions: %v", err)
	}
	var list struct {
		World        bool `json:"world"`
		Competitions []struct {
			ID           string `json:"id"`
			Name         string `json:"name"`
			Kind         string `json:"kind"`
			Participants int    `json:"participants"`
			Stage        string `json:"stage"`
		} `json:"competitions"`
	}
	decodeJSONBody(t, resp, &list)
	if resp.StatusCode != http.StatusOK || !list.World || len(list.Competitions) != 14 {
		t.Fatalf("competitions hub world=%v n=%d status=%d", list.World, len(list.Competitions), resp.StatusCode)
	}
	byID := map[string]int{}
	for _, c := range list.Competitions {
		byID[c.ID] = c.Participants
	}
	wants := map[string]int{
		"premier-league": 20, "la-liga": 20, "serie-a": 20, "bundesliga": 18, "ligue-1": 18,
		"fa-cup": 20, "efl-cup": 20, "copa-del-rey": 20, "dfb-pokal": 18, "coppa-italia": 20, "coupe-de-france": 18,
		"champions-league": 36, "europa-league": 20, "conference-league": 20,
	}
	for id, n := range wants {
		if byID[id] != n {
			t.Fatalf("%s participants=%d want %d", id, byID[id], n)
		}
	}

	resp, err = http.Get(ts.URL + "/api/competitions/premier-league")
	if err != nil {
		t.Fatalf("GET premier-league: %v", err)
	}
	var premier map[string]interface{}
	decodeJSONBody(t, resp, &premier)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("premier-league status=%d", resp.StatusCode)
	}
	if premier["kind"] != "LEAGUE" || premier["name"] != "Premier League" {
		t.Fatalf("premier-league payload=%v", premier)
	}
	fixtures, _ := premier["fixtures"].([]interface{})
	table, _ := premier["table"].([]interface{})
	if len(fixtures) != 380 || len(table) != 20 {
		t.Fatalf("premier-league fixtures=%d table=%d want 380/20", len(fixtures), len(table))
	}

	resp, err = http.Get(ts.URL + "/api/competitions/champions-league")
	if err != nil {
		t.Fatalf("GET champions-league: %v", err)
	}
	var ucl map[string]interface{}
	decodeJSONBody(t, resp, &ucl)
	if resp.StatusCode != http.StatusOK || ucl["kind"] != "EUROPEAN" {
		t.Fatalf("champions-league status=%d payload=%v", resp.StatusCode, ucl)
	}
	sources, _ := ucl["qualification_sources"].(map[string]interface{})
	participants, _ := ucl["participants"].([]interface{})
	if len(sources) != 36 || len(participants) != 36 {
		t.Fatalf("champions-league sources=%d participants=%d want 36/36", len(sources), len(participants))
	}

	resp, err = http.Get(ts.URL + "/api/competitions/fa-cup")
	if err != nil {
		t.Fatalf("GET fa-cup: %v", err)
	}
	var cup map[string]interface{}
	decodeJSONBody(t, resp, &cup)
	rounds, _ := cup["rounds"].([]interface{})
	if resp.StatusCode != http.StatusOK || cup["kind"] != "DOMESTIC_CUP" || len(rounds) == 0 {
		t.Fatalf("fa-cup status=%d kind=%v rounds=%d", resp.StatusCode, cup["kind"], len(rounds))
	}

	resp, err = http.Get(ts.URL + "/api/super-league")
	if err != nil {
		t.Fatalf("GET /api/super-league: %v", err)
	}
	var legacy map[string]interface{}
	decodeJSONBody(t, resp, &legacy)
	if resp.StatusCode != http.StatusOK || legacy["world"] != true {
		t.Fatalf("legacy super-league compatibility failed: status=%d payload=%v", resp.StatusCode, legacy)
	}

	resp, err = http.Get(ts.URL + "/api/competitions/not-a-competition")
	if err != nil {
		t.Fatalf("GET missing competition: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("missing competition status=%d want 404", resp.StatusCode)
	}

	srv.worldMu.RLock()
	nClubs := len(srv.TournamentManager.ClubsList)
	world := srv.TournamentManager.World != nil
	srv.worldMu.RUnlock()
	if nClubs != 96 || !world {
		t.Fatalf("live world clubs=%d world=%v", nClubs, world)
	}
}

func TestLegacyCareerCompetitionHubIsEmpty(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	resp, err := http.Get(ts.URL + "/api/competitions")
	if err != nil {
		t.Fatalf("GET /api/competitions: %v", err)
	}
	var list struct {
		World        bool          `json:"world"`
		Competitions []interface{} `json:"competitions"`
	}
	decodeJSONBody(t, resp, &list)
	if resp.StatusCode != http.StatusOK || list.World || len(list.Competitions) != 0 {
		t.Fatalf("legacy hub world=%v n=%d status=%d", list.World, len(list.Competitions), resp.StatusCode)
	}

	resp, err = http.Get(ts.URL + "/api/clubs")
	if err != nil {
		t.Fatalf("GET /api/clubs: %v", err)
	}
	var clubs []json.RawMessage
	decodeJSONBody(t, resp, &clubs)
	if len(clubs) != 12 {
		t.Fatalf("legacy test server clubs=%d want 12", len(clubs))
	}
}
