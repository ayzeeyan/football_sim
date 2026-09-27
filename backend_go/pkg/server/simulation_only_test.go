package server

import (
	"net/http"
	"os"
	"testing"
)

// Older engine contract tests still exercise the retired transport. Production
// starts with it disabled; only this package's legacy tests opt in.
func TestMain(m *testing.M) {
	LiveMatchStreaming = true
	os.Exit(m.Run())
}

func TestSimulationOnlyServerRetiresLiveTransport(t *testing.T) {
	LiveMatchStreaming = false
	defer func() { LiveMatchStreaming = true }()
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()
	if srv.LiveMatchEngine != nil || srv.activeTick {
		t.Fatal("simulation-only server started a live engine or ticker")
	}
	resp, err := http.Get(ts.URL + "/ws/match")
	if err != nil {
		t.Fatalf("retired endpoint: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusGone {
		t.Fatalf("retired endpoint status=%d, want 410", resp.StatusCode)
	}
}
