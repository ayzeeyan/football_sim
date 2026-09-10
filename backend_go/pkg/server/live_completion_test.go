package server

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"football_sim/pkg/matchreport"
	"football_sim/pkg/models"
	"football_sim/pkg/persistence"
	"football_sim/pkg/tournament"

	"github.com/gorilla/websocket"
)

func serverLiveCompletionFixtureWithProdigy(t *testing.T, srv *Server) (tournament.Fixture, *models.Player) {
	t.Helper()
	for _, f := range srv.TournamentManager.GetSlate(srv.TournamentManager.CurrentMatchweek) {
		if f.Competition != "super-league" || f.Status != "scheduled" {
			continue
		}
		for _, clubID := range []string{f.HomeID, f.AwayID} {
			club := srv.TournamentManager.Clubs[clubID]
			if club == nil {
				continue
			}
			for _, p := range club.GetStartingEleven(models.FixtureContext(f.Competition, f.Matchweek)) {
				if p != nil && p.UniverseWonderkid && srv.GrowthEngine.Biometrics[p.PlayerID] != nil {
					return f, p
				}
			}
		}
	}
	t.Fatal("no scheduled Super League fixture with a registered starting prodigy")
	return tournament.Fixture{}, nil
}

func serverLiveCompletionInboxCount(srv *Server, fixtureID string) int {
	count := 0
	for _, item := range srv.TournamentManager.Inbox {
		if item.FixtureID == fixtureID {
			count++
		}
	}
	return count
}

func readLiveCompletionTick(t *testing.T, conn *websocket.Conn) map[string]interface{} {
	t.Helper()
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read live tick: %v", err)
	}
	var tick map[string]interface{}
	if err := json.Unmarshal(msg, &tick); err != nil {
		t.Fatalf("failed to decode live tick: %v", err)
	}
	if _, ok := tick["league_fixture"]; !ok {
		t.Fatalf("live tick omitted league_fixture: %s", string(msg))
	}
	return tick
}

func TestLiveTickerCommitsFullTimeFixtureOnceAndSavesCareer(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	srv.worldMu.RLock()
	fixture, prodigy := serverLiveCompletionFixtureWithProdigy(t, srv)
	home := srv.TournamentManager.Clubs[fixture.HomeID]
	away := srv.TournamentManager.Clubs[fixture.AwayID]
	homePlayedBefore := home.Played
	awayPlayedBefore := away.Played
	prodigyAppsBefore := prodigy.Appearances
	bioBefore := srv.GrowthEngine.Biometrics[prodigy.PlayerID]
	xpBefore, targetBefore := bioBefore.AccumulatedXP, bioBefore.LevelXPTarget
	inboxBefore := len(srv.TournamentManager.Inbox)
	currentMatchweek := srv.TournamentManager.CurrentMatchweek
	srv.worldMu.RUnlock()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/match"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("failed to set WebSocket deadline: %v", err)
	}

	if err := conn.WriteJSON(map[string]interface{}{
		"action":  "set_clubs",
		"home_id": fixture.HomeID,
		"away_id": fixture.AwayID,
	}); err != nil {
		t.Fatalf("failed to configure live fixture: %v", err)
	}

	for {
		tick := readLiveCompletionTick(t, conn)
		lf, ok := tick["league_fixture"].(map[string]interface{})
		if !ok || lf["id"] != fixture.FixtureID {
			continue
		}
		if lf["status"] != "scheduled" {
			t.Fatalf("configured league fixture was not scheduled before kickoff: %v", lf)
		}
		break
	}

	if err := conn.WriteJSON(map[string]interface{}{"action": "set_speed", "speed": 999}); err != nil {
		t.Fatalf("failed to set live speed: %v", err)
	}
	if err := conn.WriteJSON(map[string]interface{}{"action": "kickoff"}); err != nil {
		t.Fatalf("failed to kickoff live fixture: %v", err)
	}

	var fullTime map[string]interface{}
	for {
		tick := readLiveCompletionTick(t, conn)
		if state, _ := tick["state"].(string); state == "FULL_TIME" {
			fullTime = tick
			break
		}
	}
	lf, ok := fullTime["league_fixture"].(map[string]interface{})
	if !ok || lf["id"] != fixture.FixtureID || lf["status"] != "finished" {
		t.Fatalf("full-time tick did not expose the committed league fixture: %v", fullTime["league_fixture"])
	}

	srv.worldMu.RLock()
	finished := srv.TournamentManager.FindFixture(fixture.FixtureID)
	if finished == nil {
		srv.worldMu.RUnlock()
		t.Fatalf("committed fixture %s disappeared", fixture.FixtureID)
	}
	if finished.Status != "finished" || finished.Method != "live" || finished.Report == nil {
		srv.worldMu.RUnlock()
		t.Fatalf("fixture was not fully committed: status=%s method=%s report=%v", finished.Status, finished.Method, finished.Report != nil)
	}
	if finished.HomeGoals == nil || finished.AwayGoals == nil || *finished.HomeGoals != srv.LiveMatchEngine.HomeScore || *finished.AwayGoals != srv.LiveMatchEngine.AwayScore {
		srv.worldMu.RUnlock()
		t.Fatalf("fixture score does not match live engine: %v-%v vs %d-%d", finished.HomeGoals, finished.AwayGoals, srv.LiveMatchEngine.HomeScore, srv.LiveMatchEngine.AwayScore)
	}
	if len(finished.Report.Events) != len(srv.LiveMatchEngine.Events) || len(finished.Report.HomeXI) == 0 || len(finished.Report.AwayXI) == 0 || len(finished.Report.HomeBench) == 0 || len(finished.Report.AwayBench) == 0 {
		srv.worldMu.RUnlock()
		t.Fatalf("live report omitted engine data: events=%d/%d homeXI=%d awayXI=%d benches=%d/%d", len(finished.Report.Events), len(srv.LiveMatchEngine.Events), len(finished.Report.HomeXI), len(finished.Report.AwayXI), len(finished.Report.HomeBench), len(finished.Report.AwayBench))
	}
	if home.Played != homePlayedBefore+1 || away.Played != awayPlayedBefore+1 {
		srv.worldMu.RUnlock()
		t.Fatalf("standings updated incorrectly: home=%d from %d, away=%d from %d", home.Played, homePlayedBefore, away.Played, awayPlayedBefore)
	}
	prodigyPlayed := false
	for _, row := range append(append([]matchreport.MatchPlayerRow{}, finished.Report.HomeXI...), finished.Report.AwayXI...) {
		if row.PlayerID == prodigy.PlayerID {
			prodigyPlayed = row.Played && row.Minutes > 0
			break
		}
	}
	if !prodigyPlayed || prodigy.Appearances != prodigyAppsBefore+1 {
		srv.worldMu.RUnlock()
		t.Fatalf("participating prodigy application missing: played=%v appearances=%d from %d", prodigyPlayed, prodigy.Appearances, prodigyAppsBefore)
	}
	bioAfter := srv.GrowthEngine.Biometrics[prodigy.PlayerID]
	if bioAfter.AccumulatedXP == xpBefore && bioAfter.LevelXPTarget == targetBefore {
		srv.worldMu.RUnlock()
		t.Fatalf("participating prodigy received no match XP")
	}
	if len(srv.TournamentManager.Inbox) <= inboxBefore || serverLiveCompletionInboxCount(srv, fixture.FixtureID) == 0 {
		srv.worldMu.RUnlock()
		t.Fatalf("live commit did not add fixture inbox items")
	}
	currentAfterCommit := srv.TournamentManager.CurrentMatchweek
	if currentAfterCommit != currentMatchweek {
		srv.worldMu.RUnlock()
		t.Fatalf("live commit unexpectedly rolled over matchweek: %d from %d", currentAfterCommit, currentMatchweek)
	}
	srv.worldMu.RUnlock()

	snapshot, err := persistence.LoadCareer(srv.savePath)
	if err != nil {
		t.Fatalf("live commit did not produce a readable save: %v", err)
	}
	var savedFixture *tournament.Fixture
	for i := range snapshot.Fixtures {
		if snapshot.Fixtures[i].FixtureID == fixture.FixtureID {
			savedFixture = &snapshot.Fixtures[i]
			break
		}
	}
	if savedFixture == nil || savedFixture.Status != "finished" || savedFixture.Method != "live" || savedFixture.Report == nil || snapshot.CurrentMatchweek != currentMatchweek {
		t.Fatalf("saved career does not match live result: fixture=%+v matchweek=%d", savedFixture, snapshot.CurrentMatchweek)
	}

	srv.worldMu.Lock()
	second := srv.TournamentManager.CommitLiveFixture(fixture.HomeID, fixture.AwayID, srv.LiveMatchEngine)
	srv.worldMu.Unlock()
	if second["recorded"] == true {
		t.Fatalf("same live engine was recorded twice: %v", second)
	}
	srv.worldMu.RLock()
	if home.Played != homePlayedBefore+1 || away.Played != awayPlayedBefore+1 || prodigy.Appearances != prodigyAppsBefore+1 || len(srv.TournamentManager.Inbox) <= inboxBefore {
		srv.worldMu.RUnlock()
		t.Fatalf("duplicate live commit changed career state: %v", second)
	}
	srv.worldMu.RUnlock()

	srv.worldMu.Lock()
	slate := srv.TournamentManager.GetSlate(srv.TournamentManager.CurrentMatchweek)
	remainingExpected := 0
	for _, f := range slate {
		if f.Status == "scheduled" {
			remainingExpected++
		}
	}
	remaining := srv.TournamentManager.SimulateRemaining()
	srv.worldMu.Unlock()
	played, ok := remaining["played"].(int)
	if !ok || played != remainingExpected {
		t.Fatalf("remaining slate played=%v, want %d", remaining["played"], remainingExpected)
	}
	if srv.TournamentManager.CurrentMatchweek != currentMatchweek+1 {
		t.Fatalf("remaining slate did not roll over once: current matchweek=%d", srv.TournamentManager.CurrentMatchweek)
	}
	if replayed := srv.TournamentManager.FindFixture(fixture.FixtureID); replayed == nil || replayed.Status != "finished" || replayed.Method != "live" || replayed.HomeGoals == nil || *replayed.HomeGoals != int(fullTime["home_score"].(float64)) {
		t.Fatalf("live fixture changed during remaining simulation: %+v", replayed)
	}
}

func TestLiveTickerKeepsExactFixtureIdentityWhenLastLeagueResultRollsOver(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	srv.worldMu.Lock()
	fixture, _ := serverLiveCompletionFixtureWithProdigy(t, srv)
	for _, other := range srv.TournamentManager.GetSlate(srv.TournamentManager.CurrentMatchweek) {
		if other.Competition != "super-league" || other.Status != "scheduled" || other.FixtureID == fixture.FixtureID {
			continue
		}
		result := srv.TournamentManager.SimulateFixture(other.FixtureID)
		if result["status"] != "success" {
			srv.worldMu.Unlock()
			t.Fatalf("failed to simulate preceding league fixture %s: %v", other.FixtureID, result)
		}
	}
	if srv.TournamentManager.CurrentMatchweek != fixture.Matchweek {
		srv.worldMu.Unlock()
		t.Fatalf("preceding fixtures changed matchweek before live kickoff: current=%d target=%d", srv.TournamentManager.CurrentMatchweek, fixture.Matchweek)
	}
	srv.worldMu.Unlock()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/match"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("failed to set WebSocket deadline: %v", err)
	}
	if err := conn.WriteJSON(map[string]interface{}{"action": "set_clubs", "home_id": fixture.HomeID, "away_id": fixture.AwayID}); err != nil {
		t.Fatalf("failed to configure live fixture: %v", err)
	}
	for {
		tick := readLiveCompletionTick(t, conn)
		lf, ok := tick["league_fixture"].(map[string]interface{})
		if ok && lf["id"] == fixture.FixtureID {
			if lf["status"] != "scheduled" {
				t.Fatalf("selected fixture was not scheduled before kickoff: %v", lf)
			}
			break
		}
	}
	if err := conn.WriteJSON(map[string]interface{}{"action": "set_speed", "speed": 999}); err != nil {
		t.Fatalf("failed to set live speed: %v", err)
	}
	if err := conn.WriteJSON(map[string]interface{}{"action": "kickoff"}); err != nil {
		t.Fatalf("failed to kickoff live fixture: %v", err)
	}

	var fullTime map[string]interface{}
	for {
		tick := readLiveCompletionTick(t, conn)
		if state, _ := tick["state"].(string); state == "FULL_TIME" {
			fullTime = tick
			break
		}
	}
	lf, ok := fullTime["league_fixture"].(map[string]interface{})
	if !ok || lf["id"] != fixture.FixtureID || lf["status"] != "finished" {
		t.Fatalf("rollover full-time tick lost the selected fixture: %v", fullTime["league_fixture"])
	}
	srv.worldMu.RLock()
	current := srv.TournamentManager.CurrentMatchweek
	finished := srv.TournamentManager.FindFixture(fixture.FixtureID)
	srv.worldMu.RUnlock()
	if current != fixture.Matchweek+1 || finished == nil || finished.Status != "finished" || finished.Method != "live" {
		t.Fatalf("live commit did not finish and roll over once: current=%d fixture=%+v", current, finished)
	}
}

func TestLiveFixtureSelectionClearsOnResetAndExhibitionReselection(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	current := srv.TournamentManager.CurrentMatchweek
	slate := srv.TournamentManager.GetSlate(current)
	var first, second tournament.Fixture
	for _, f := range slate {
		if f.Competition != "super-league" || f.Status != "scheduled" {
			continue
		}
		if first.FixtureID == "" {
			first = f
		} else if f.FixtureID != first.FixtureID {
			second = f
			break
		}
	}
	if first.FixtureID == "" || second.FixtureID == "" {
		t.Fatal("need two scheduled league fixtures for selection lifecycle test")
	}
	scheduledPairs := map[string]bool{}
	for _, f := range slate {
		if f.Status == "scheduled" {
			scheduledPairs[f.HomeID+"/"+f.AwayID] = true
		}
	}
	var exhibitionHome, exhibitionAway string
	for _, home := range srv.TournamentManager.ClubsList {
		for _, away := range srv.TournamentManager.ClubsList {
			if home.ClubID != away.ClubID && !scheduledPairs[home.ClubID+"/"+away.ClubID] {
				exhibitionHome, exhibitionAway = home.ClubID, away.ClubID
				break
			}
		}
		if exhibitionHome != "" {
			break
		}
	}
	if exhibitionHome == "" {
		t.Fatal("could not find an exhibition pairing outside the current slate")
	}

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/match"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("failed to set WebSocket deadline: %v", err)
	}
	if err := conn.WriteJSON(map[string]interface{}{"action": "set_clubs", "home_id": first.HomeID, "away_id": first.AwayID}); err != nil {
		t.Fatalf("failed to select first fixture: %v", err)
	}
	for {
		tick := readLiveCompletionTick(t, conn)
		lf, ok := tick["league_fixture"].(map[string]interface{})
		if ok && lf["id"] == first.FixtureID {
			break
		}
	}
	if err := conn.WriteJSON(map[string]interface{}{"action": "reset"}); err != nil {
		t.Fatalf("failed to reset live session: %v", err)
	}
	for {
		tick := readLiveCompletionTick(t, conn)
		if state, _ := tick["state"].(string); state == "NOT_STARTED" && tick["league_fixture"] == nil {
			break
		}
	}
	if err := conn.WriteJSON(map[string]interface{}{"action": "set_clubs", "home_id": second.HomeID, "away_id": second.AwayID}); err != nil {
		t.Fatalf("failed to reselect second fixture: %v", err)
	}
	for {
		tick := readLiveCompletionTick(t, conn)
		lf, ok := tick["league_fixture"].(map[string]interface{})
		if ok && lf["id"] == second.FixtureID {
			break
		}
	}
	if err := conn.WriteJSON(map[string]interface{}{"action": "set_clubs", "home_id": exhibitionHome, "away_id": exhibitionAway}); err != nil {
		t.Fatalf("failed to select exhibition pairing: %v", err)
	}
	for {
		tick := readLiveCompletionTick(t, conn)
		if tick["league_fixture"] == nil {
			break
		}
	}
}

func TestLiveTickerRetriesTransientExactFixtureCommitFailure(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	fixture, _ := serverLiveCompletionFixtureWithProdigy(t, srv)
	home := srv.TournamentManager.Clubs[fixture.HomeID]
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/match"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()
	if err := conn.SetReadDeadline(time.Now().Add(8 * time.Second)); err != nil {
		t.Fatalf("failed to set WebSocket deadline: %v", err)
	}
	if err := conn.WriteJSON(map[string]interface{}{"action": "set_clubs", "home_id": fixture.HomeID, "away_id": fixture.AwayID}); err != nil {
		t.Fatalf("failed to configure live fixture: %v", err)
	}
	for {
		tick := readLiveCompletionTick(t, conn)
		lf, ok := tick["league_fixture"].(map[string]interface{})
		if ok && lf["id"] == fixture.FixtureID {
			break
		}
	}

	srv.worldMu.Lock()
	srv.TournamentManager.Clubs[fixture.HomeID] = nil
	srv.worldMu.Unlock()
	restored := false
	defer func() {
		if !restored {
			srv.worldMu.Lock()
			srv.TournamentManager.Clubs[fixture.HomeID] = home
			srv.worldMu.Unlock()
		}
	}()
	if err := conn.WriteJSON(map[string]interface{}{"action": "set_speed", "speed": 999}); err != nil {
		t.Fatalf("failed to set live speed: %v", err)
	}
	if err := conn.WriteJSON(map[string]interface{}{"action": "kickoff"}); err != nil {
		t.Fatalf("failed to kickoff live fixture: %v", err)
	}

	instance := 0
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		srv.worldMu.RLock()
		state := srv.LiveMatchEngine.State
		instance = srv.LiveMatchEngine.InstanceID
		handled := srv.lastCommittedLiveInstance
		srv.worldMu.RUnlock()
		if state == "FULL_TIME" {
			if handled == instance {
				t.Fatalf("retryable exact-fixture failure was marked handled: instance=%d", instance)
			}
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if instance == 0 {
		t.Fatal("live engine did not reach full time while inducing transient failure")
	}
	srv.worldMu.Lock()
	srv.TournamentManager.Clubs[fixture.HomeID] = home
	restored = true
	srv.worldMu.Unlock()

	for {
		tick := readLiveCompletionTick(t, conn)
		lf, ok := tick["league_fixture"].(map[string]interface{})
		if ok && lf["id"] == fixture.FixtureID && lf["status"] == "finished" {
			break
		}
	}
	finished := srv.TournamentManager.FindFixture(fixture.FixtureID)
	if finished == nil || finished.Status != "finished" || finished.Method != "live" {
		t.Fatalf("transient exact-fixture failure did not retry: %+v", finished)
	}
}
