package server

import (
	"testing"
)

// Pre-match XI shows a one-liner only for prodigies who grew since August.
func TestServer_GrewNoteForGrownKid(t *testing.T) {
	srv, ts := setupTestServer(t)
	defer srv.Stop()
	defer ts.Close()

	// Find a wonderkid and force August-to-now growth on his bio.
	var kidID, clubID string
	for _, c := range srv.TournamentManager.ClubsList {
		for _, p := range c.Squad {
			if p.UniverseWonderkid {
				kidID, clubID = p.PlayerID, c.ClubID
				break
			}
		}
		if kidID != "" {
			break
		}
	}
	if kidID == "" {
		t.Skip("no prodigy")
	}
	bio := srv.GrowthEngine.Biometrics[kidID]
	if bio == nil {
		t.Skip("no biometrics")
	}
	bio.CurrentHeightCM = bio.BaselineHeightCM + 1.4

	// Preview a scheduled fixture involving his club.
	found := false
	for i := range srv.TournamentManager.Fixtures {
		f := &srv.TournamentManager.Fixtures[i]
		if f.Status != "scheduled" {
			continue
		}
		if f.HomeID != clubID && f.AwayID != clubID {
			continue
		}
		home := srv.TournamentManager.Clubs[f.HomeID]
		away := srv.TournamentManager.Clubs[f.AwayID]
		pv := srv.fixturePreview(f, home, away)
		if pv == nil {
			t.Fatal("nil preview")
		}
		for _, key := range []string{"home_xi", "away_xi"} {
			rows, _ := pv[key].([]map[string]interface{})
			for _, row := range rows {
				if row["player_id"] != kidID {
					continue
				}
				found = true
				note, _ := row["grew_note"].(string)
				if note == "" {
					t.Fatalf("grown kid %s should carry a grew_note in the XI", kidID)
				}
			}
		}
		break
	}
	if !found {
		t.Skip("kid not in a scheduled preview XI")
	}

	// Reset growth: the note must disappear.
	bio.CurrentHeightCM = bio.BaselineHeightCM
	for i := range srv.TournamentManager.Fixtures {
		f := &srv.TournamentManager.Fixtures[i]
		if f.Status != "scheduled" {
			continue
		}
		if f.HomeID != clubID && f.AwayID != clubID {
			continue
		}
		home := srv.TournamentManager.Clubs[f.HomeID]
		away := srv.TournamentManager.Clubs[f.AwayID]
		pv := srv.fixturePreview(f, home, away)
		for _, key := range []string{"home_xi", "away_xi"} {
			rows, _ := pv[key].([]map[string]interface{})
			for _, row := range rows {
				if row["player_id"] == kidID {
					if note, _ := row["grew_note"].(string); note != "" {
						t.Fatalf("ungrown kid must not carry a grew_note, got %q", note)
					}
				}
			}
		}
		break
	}
}
