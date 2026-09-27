package server

import (
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
)

// documentedRoutes extracts "METHOD /path" pairs from README.md's API list.
// List items may share one method across several backticked paths
// ("- `GET /api/clubs`, `/api/clubs/{club_id}/squad`, ..."), so the method
// carries forward within a line. Query strings are stripped.
func documentedRoutes(t *testing.T) map[string]bool {
	t.Helper()
	raw, err := os.ReadFile(filepathJoin("..", "..", "..", "README.md"))
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	tokenPattern := regexp.MustCompile("`([^`]+)`")
	methodPattern := regexp.MustCompile("^(GET|POST|PUT|DELETE)$")
	out := make(map[string]bool)
	for _, line := range strings.Split(string(raw), "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "- `") {
			continue
		}
		method := ""
		for _, match := range tokenPattern.FindAllStringSubmatch(line, -1) {
			token := strings.TrimSpace(match[1])
			parts := strings.SplitN(token, " ", 2)
			if len(parts) == 2 && methodPattern.MatchString(parts[0]) {
				method = parts[0]
				token = strings.TrimSpace(parts[1])
			}
			if !strings.HasPrefix(token, "/api/") {
				continue
			}
			token = strings.SplitN(token, "?", 2)[0]
			if method == "" {
				t.Fatalf("README documents %q without an HTTP method", token)
			}
			out[method+" "+token] = true
		}
	}
	return out
}

func filepathJoin(parts ...string) string {
	return strings.Join(parts, string(os.PathSeparator))
}

// Every registered route must be documented in README.md, and every
// documented route must exist in the registry.
func TestRouteRegistryMatchesReadmeDocs(t *testing.T) {
	srv := &Server{}
	registered := make(map[string]bool)
	for _, route := range srv.apiRoutes() {
		key := route.Method + " " + route.Path
		if registered[key] {
			t.Errorf("route %s registered twice", key)
		}
		registered[key] = true
	}

	documented := documentedRoutes(t)

	for _, key := range sortedRouteKeys(srv.apiRoutes()) {
		if !documented[key] {
			t.Errorf("registered route %s is not documented in README.md", key)
		}
	}
	for key := range documented {
		if !registered[key] {
			t.Errorf("README.md documents %s but it is not registered", key)
		}
	}
}

func TestOpenAPISpecCoversEveryRoute(t *testing.T) {
	srv := &Server{}
	routes := srv.apiRoutes()
	spec := BuildOpenAPISpec(routes)
	paths, _ := spec["paths"].(map[string]interface{})
	if len(paths) == 0 {
		t.Fatal("OpenAPI spec has no paths")
	}
	if spec["openapi"] != "3.1.0" {
		t.Fatalf("spec version %v, want 3.1.0", spec["openapi"])
	}
	for _, route := range routes {
		entry, ok := paths[route.Path].(map[string]interface{})
		if !ok {
			t.Errorf("spec missing path %s", route.Path)
			continue
		}
		op, ok := entry[strings.ToLower(route.Method)].(map[string]interface{})
		if !ok {
			t.Errorf("spec missing %s operation for %s", route.Method, route.Path)
			continue
		}
		if op["summary"] == "" {
			t.Errorf("spec operation for %s has no summary", route.Path)
		}
	}
	// Path parameters must be declared for parameterized routes.
	for _, name := range openAPIPathParams("/api/clubs/{club_id}/squad") {
		if name != "club_id" {
			t.Errorf("unexpected path param %q", name)
		}
	}
}

func TestOpenAPISpecEndpointServed(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	resp, err := http.Get(ts.URL + "/api/openapi.json")
	if err != nil {
		t.Fatalf("GET /api/openapi.json: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/openapi.json status=%d", resp.StatusCode)
	}
	var spec map[string]interface{}
	decodeJSONBody(t, resp, &spec)
	if spec["openapi"] != "3.1.0" {
		t.Fatalf("served spec version %v, want 3.1.0", spec["openapi"])
	}
	paths, _ := spec["paths"].(map[string]interface{})
	if _, ok := paths["/api/health"]; !ok {
		t.Fatal("served spec missing /api/health")
	}
}
