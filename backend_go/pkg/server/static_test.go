package server

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestStaticSPAServesAssetsWithoutEscapingDist(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer srv.Stop()
	defer ts.Close()

	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>career</html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "assets", "app.js"), []byte("window.app=1"), 0o644); err != nil {
		t.Fatal(err)
	}
	srv.staticDir = dir

	js, err := http.Get(ts.URL + "/assets/app.js")
	if err != nil {
		t.Fatalf("GET asset: %v", err)
	}
	body, _ := io.ReadAll(js.Body)
	js.Body.Close()
	if js.StatusCode != http.StatusOK || string(body) != "window.app=1" {
		t.Fatalf("asset status=%d body=%q", js.StatusCode, body)
	}

	home, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	homeBody, _ := io.ReadAll(home.Body)
	home.Body.Close()
	if home.StatusCode != http.StatusOK || string(homeBody) != "<html>career</html>" {
		t.Fatalf("index status=%d body=%q", home.StatusCode, homeBody)
	}
}
