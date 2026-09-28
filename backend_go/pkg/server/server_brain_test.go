package server

import (
	"net/http"
	"testing"
)

// The brain endpoint exposes the learned state observationally.
func TestBrainEndpoint(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	if payload := postNewCareer(t, ts.URL, false, nil); payload["status"] != "success" {
		t.Fatalf("fresh world career payload: %v", payload)
	}

	resp, err := http.Get(ts.URL + "/api/brain")
	if err != nil {
		t.Fatalf("GET brain: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("brain status=%d", resp.StatusCode)
	}
	var body struct {
		Brain *struct {
			Weights []struct {
				Feature string  `json:"feature"`
				Weight  float64 `json:"weight"`
			} `json:"weights"`
			Bias    float64 `json:"bias"`
			Samples int     `json:"samples"`
			Edges   []struct {
				HomeStyle string  `json:"home_style"`
				FixedEdge float64 `json:"fixed_edge"`
				Edge      float64 `json:"edge"`
			} `json:"tactical_edges"`
		} `json:"brain"`
	}
	decodeJSONBody(t, resp, &body)
	if body.Brain == nil {
		t.Fatal("fresh world has no brain")
	}
	if len(body.Brain.Weights) != 8 {
		t.Fatalf("weights=%d want 8", len(body.Brain.Weights))
	}
	if body.Brain.Samples < 1000 {
		t.Fatalf("base samples=%d, want the pre-trained base", body.Brain.Samples)
	}
	if len(body.Brain.Edges) == 0 {
		t.Fatal("no tactical edges exposed")
	}
}
