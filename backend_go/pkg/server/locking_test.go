package server

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"football_sim/pkg/persistence"

	"github.com/gorilla/websocket"
)

func errString(op string, status int) error {
	return fmt.Errorf("%s: unexpected status %d", op, status)
}

func TestBroadcastDoesNotHoldWSMutexDuringWrite(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	srv.wsWrite = func(conn *websocket.Conn, payload interface{}) error {
		once.Do(func() { close(started) })
		<-release
		return nil
	}

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/match"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect websocket: %v", err)
	}
	defer conn.Close()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("ticker did not attempt a websocket write")
	}

	statsDone := make(chan error, 1)
	go func() {
		resp, err := http.Get(ts.URL + "/api/stats")
		if err != nil {
			statsDone <- err
			return
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			statsDone <- errString("stats status", resp.StatusCode)
			return
		}
		statsDone <- nil
	}()

	select {
	case err := <-statsDone:
		if err != nil {
			t.Fatalf("GET /api/stats failed while a tick write was blocked: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("GET /api/stats deadlocked behind a blocked websocket write")
	}

	unreg := make(chan struct{})
	go func() {
		_ = conn.Close()
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			if srv.wsClientCount() == 0 {
				close(unreg)
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()

	select {
	case <-unreg:
	case <-time.After(2 * time.Second):
		close(release)
		t.Fatal("websocket unregister deadlocked behind a blocked broadcast write")
	}
	close(release)
}

func TestStaleCareerSnapshotDoesNotOverwriteNewerSave(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	srv.worldMu.Lock()
	weekBefore := srv.TournamentManager.CurrentMatchweek
	stale, staleGen := srv.takeCareerSnapshotLocked()
	srv.worldMu.Unlock()

	resp, err := http.Post(ts.URL+"/api/fixtures/simulate-remaining", "application/json", nil)
	if err != nil {
		t.Fatalf("simulate remaining failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("simulate remaining status=%d", resp.StatusCode)
	}

	srv.commitCareerSnapshot(stale, staleGen)

	saved, err := persistence.LoadCareer(srv.savePath)
	if err != nil {
		t.Fatalf("load career: %v", err)
	}
	if saved.CurrentMatchweek == weekBefore {
		t.Fatalf("stale snapshot overwrote the newer career: matchweek=%d", saved.CurrentMatchweek)
	}
	if saved.CurrentMatchweek != srv.TournamentManager.CurrentMatchweek {
		t.Fatalf("saved matchweek=%d, live=%d", saved.CurrentMatchweek, srv.TournamentManager.CurrentMatchweek)
	}
}

func TestStatsAndLiveCommandsDoNotDeadlock(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/match"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect websocket: %v", err)
	}
	defer conn.Close()

	done := make(chan error, 1)
	go func() {
		for i := 0; i < 40; i++ {
			resp, err := http.Get(ts.URL + "/api/stats")
			if err != nil {
				done <- err
				return
			}
			resp.Body.Close()
			if err := conn.WriteJSON(map[string]interface{}{"action": "pause"}); err != nil {
				done <- err
				return
			}
		}
		done <- nil
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("concurrent stats/live commands failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("concurrent stats and live commands deadlocked")
	}
}
