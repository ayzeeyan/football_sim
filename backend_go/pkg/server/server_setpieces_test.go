package server

import (
	"encoding/json"
	"net/http"
	"testing"
)

// The set-piece endpoint serves the probable XI's briefing: every pick names
// a real squad player and carries a reason.
func TestSetPiecesEndpointServesBriefing(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer ts.Close()
	defer srv.Stop()

	if payload := postNewCareer(t, ts.URL, false, nil); payload["status"] != "success" {
		t.Fatalf("fresh world career payload: %v", payload)
	}

	clubs := []struct {
		ClubID string `json:"club_id"`
	}{}
	resp, err := http.Get(ts.URL + "/api/clubs")
	if err != nil {
		t.Fatalf("GET /api/clubs: %v", err)
	}
	decodeJSONBody(t, resp, &clubs)
	if len(clubs) == 0 {
		t.Fatal("no clubs in the world")
	}
	clubID := clubs[0].ClubID

	resp, err = http.Get(ts.URL + "/api/clubs/" + clubID + "/set-pieces")
	if err != nil {
		t.Fatalf("GET set-pieces: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("set-pieces status=%d", resp.StatusCode)
	}
	var body struct {
		ClubID    string `json:"club_id"`
		SetPieces struct {
			PenaltyTaker  *setPiecePickJSON `json:"penalty_taker"`
			FreeKickTaker *setPiecePickJSON `json:"free_kick_taker"`
			AerialTarget  *setPiecePickJSON `json:"aerial_target"`
			CornerTaker   *setPiecePickJSON `json:"corner_taker"`
		} `json:"set_pieces"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode set-pieces: %v", err)
	}
	if body.ClubID != clubID {
		t.Fatalf("set-pieces club_id=%q want %q", body.ClubID, clubID)
	}
	if body.SetPieces.PenaltyTaker == nil || body.SetPieces.PenaltyTaker.PlayerID == "" {
		t.Fatalf("penalty taker missing: %+v", body.SetPieces)
	}
	if body.SetPieces.AerialTarget == nil || body.SetPieces.CornerTaker == nil {
		t.Fatalf("aerial/corner picks missing: %+v", body.SetPieces)
	}
	if body.SetPieces.CornerTaker.PlayerID == body.SetPieces.AerialTarget.PlayerID {
		t.Fatal("corner taker must differ from the aerial target")
	}
	for _, pick := range []*setPiecePickJSON{body.SetPieces.PenaltyTaker, body.SetPieces.FreeKickTaker, body.SetPieces.AerialTarget, body.SetPieces.CornerTaker} {
		if pick == nil {
			continue
		}
		if pick.Reason == "" || pick.FullName == "" {
			t.Fatalf("pick %s must carry a name and reason: %+v", pick.PlayerID, pick)
		}
	}

	// Unknown club is a clean 404.
	resp, err = http.Get(ts.URL + "/api/clubs/NO-SUCH-CLUB/set-pieces")
	if err != nil {
		t.Fatalf("GET unknown club set-pieces: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown club status=%d want 404", resp.StatusCode)
	}
}

type setPiecePickJSON struct {
	PlayerID   string  `json:"player_id"`
	FullName   string  `json:"full_name"`
	Position   string  `json:"position"`
	OVR        int     `json:"ovr"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason"`
}
