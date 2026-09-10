package server

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// blockingRW stalls inside Write so the test can probe whether worldMu is
// still held while the handler encodes.
type blockingRW struct {
	header  http.Header
	entered chan struct{}
	release chan struct{}
	once    sync.Once
	wrote   int
}

func (b *blockingRW) Header() http.Header { return b.header }
func (b *blockingRW) WriteHeader(int)     {}
func (b *blockingRW) Write(p []byte) (int, error) {
	b.once.Do(func() { close(b.entered) })
	<-b.release
	b.wrote += len(p)
	return len(p), nil
}

// F2: fat GETs must snapshot under worldMu and encode after release, so the
// 60 FPS ticker (worldMu.Lock every 16ms) is never stalled by fat JSON.
func TestFatGETEncodesOutsideWorldMu(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	for _, target := range []string{"/api/fixtures", "/api/prodigies"} {
		rw := &blockingRW{header: http.Header{}, entered: make(chan struct{}), release: make(chan struct{})}
		done := make(chan struct{})
		req := httptest.NewRequest(http.MethodGet, target, nil)
		go func() {
			srv.ServeHTTP(rw, req)
			close(done)
		}()

		select {
		case <-rw.entered:
		case <-time.After(5 * time.Second):
			t.Fatalf("GET %s never reached encode", target)
		}

		// Handler is blocked inside encode: worldMu must be acquirable.
		locked := make(chan struct{})
		go func() {
			srv.worldMu.Lock()
			close(locked)
			srv.worldMu.Unlock()
		}()
		select {
		case <-locked:
		case <-time.After(2 * time.Second):
			close(rw.release)
			<-done
			t.Fatalf("GET %s holds worldMu during encode", target)
		}

		close(rw.release)
		<-done
		if rw.wrote == 0 {
			t.Fatalf("GET %s wrote an empty body", target)
		}
	}
}
