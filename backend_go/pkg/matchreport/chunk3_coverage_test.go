package matchreport

import (
	"math/rand"
	"testing"

	"football_sim/pkg/models"
)

// Chunk 3 coverage: on-field replay, live shot/touch passthroughs.

func chunk3XI(prefix string, n int) []*models.Player {
	cats := []string{"GK", "DEF", "DEF", "DEF", "DEF", "MID", "MID", "MID", "FWD", "FWD", "FWD"}
	poss := []string{"GK", "CB", "CB", "CB", "RB", "CM", "CM", "CAM", "LW", "ST", "RW"}
	xi := make([]*models.Player, 0, n)
	for i := 0; i < n && i < 11; i++ {
		xi = append(xi, &models.Player{
			PlayerID: prefix + string(rune('A'+i)), FullName: prefix + " Player " + string(rune('A'+i)),
			Position: poss[i], Category: cats[i], OVR: 75 + i%5, Age: 25,
		})
	}
	return xi
}

func TestChunk3OnFieldPlayers(t *testing.T) {
	homeXI := chunk3XI("H", 11)
	homeBench := []*models.Player{
		{PlayerID: "HB1", FullName: "Home Sub 1", Position: "CM", Category: "MID", OVR: 74, Age: 22},
	}
	awayXI := chunk3XI("A", 11)
	payload := InstantPayload{
		HomeXI: homeXI, HomeBench: homeBench, AwayXI: awayXI,
		Events: []MatchEventItem{
			{Minute: 60, Seq: 1, Type: "sub", Side: "home",
				PlayerOut: &MiniPlayer{PlayerID: "HC"}, PlayerIn: &MiniPlayer{PlayerID: "HB1"}},
			{Minute: 70, Seq: 2, Type: "red", Side: "home", Player: &MiniPlayer{PlayerID: "HB"}},
			{Minute: 75, Seq: 3, Type: "sub", Side: "away",
				PlayerOut: &MiniPlayer{PlayerID: "AX"}, PlayerIn: &MiniPlayer{PlayerID: "AY"}},
			{Minute: 80, Seq: 4, Type: "goal", Side: "home"},
			{Minute: 65, Seq: 6, Type: "yellow", Side: "home", Player: &MiniPlayer{PlayerID: "HA"}},
			{Minute: 65, Seq: 5, Type: "yellow", Side: "away", Player: &MiniPlayer{PlayerID: "AA"}},
		},
	}

	// Full time: sub applied, dismissal removed.
	on := OnFieldPlayers(payload, "home", 90)
	if len(on) != 10 {
		t.Fatalf("home field = %d; want 10 (11 -1 sub +1 -1 red)", len(on))
	}
	ids := map[string]bool{}
	for _, p := range on {
		ids[p.PlayerID] = true
	}
	if !ids["HB1"] || ids["HC"] || ids["HB"] {
		t.Errorf("home field wrong: %v", ids)
	}
	// At half time nobody had left yet.
	early := OnFieldPlayers(payload, "home", 45)
	if len(early) != 11 {
		t.Errorf("early home field = %d; want 11", len(early))
	}
	// Away side replays its own events only.
	aw := OnFieldPlayers(payload, "away", 90)
	if len(aw) != 11 {
		t.Errorf("away field = %d; want 11 (unknown sub IDs ignored)", len(aw))
	}
	// Malformed sub/red events without players are skipped safely.
	broken := payload
	broken.Events = []MatchEventItem{
		{Minute: 60, Seq: 1, Type: "sub", Side: "home"},
		{Minute: 70, Seq: 2, Type: "red", Side: "home"},
	}
	if got := OnFieldPlayers(broken, "home", 90); len(got) != 11 {
		t.Errorf("broken events should leave XI intact, got %d", len(got))
	}
	// Unknown side falls through to the away XI (mirroring Python's if/else).
	if got := OnFieldPlayers(payload, "third", 90); len(got) != 11 {
		t.Errorf("unknown side should mirror away XI, got %d", len(got))
	}
}

func TestChunk3LiveShotPassthrough(t *testing.T) {
	home, away := &models.Club{ClubID: "H", ShortName: "HOM", OverallTeamRating: 84},
		&models.Club{ClubID: "A", ShortName: "AWY", OverallTeamRating: 82}
	live := []ShotMapItem{
		{Minute: 23, Team: "home", Shooter: ShotShooter{FullName: "Live Shooter", Position: "ST", OVR: 85}, X: 0.9, Y: 0.5, XG: 0.44, Outcome: "goal", IsWonderkid: true},
		{Minute: 67, Team: "away", Shooter: ShotShooter{FullName: "Other", Position: "CM", OVR: 80}, X: 0.1, Y: 0.5, XG: 0.2, Outcome: "miss"},
	}
	rng := rand.New(rand.NewSource(3))
	sm := GenerateShotMap(home, away, nil, 0, 0, 0, 0, rng, live)
	if len(sm.Shots) != 2 || sm.Shots[0].Minute != 23 || sm.Shots[1].Minute != 67 {
		t.Fatalf("live shots should pass through sorted: %+v", sm.Shots)
	}
	if sm.TotalHomeXG != 0.44 || sm.TotalAwayXG != 0.2 {
		t.Errorf("passthrough xG wrong: %v/%v", sm.TotalHomeXG, sm.TotalAwayXG)
	}
	if tail := sm.XGFlow[len(sm.XGFlow)-1]; tail.Minute != 90 {
		t.Errorf("passthrough flow tail = %d; want 90", tail.Minute)
	}
	if !sm.Shots[0].IsWonderkid {
		t.Errorf("wonderkid flag should survive passthrough")
	}
}

func TestChunk3LiveTouchPassthrough(t *testing.T) {
	home, away := &models.Club{ClubID: "H", ShortName: "HOM"}, &models.Club{ClubID: "A", ShortName: "AWY"}
	var touchesH, touchesA [][2]float64
	for i := 0; i < 10; i++ {
		touchesH = append(touchesH, [2]float64{0.7, 0.4})
		touchesA = append(touchesA, [2]float64{0.3, 0.6})
	}
	rng := rand.New(rand.NewSource(5))
	hm := GenerateTouchHeatmap(home, away, 50, rng, map[string][][2]float64{"home": touchesH, "away": touchesA})
	if len(hm.HomePoints) != 10 || len(hm.AwayPoints) != 10 {
		t.Fatalf("live touches should pass through: %d/%d", len(hm.HomePoints), len(hm.AwayPoints))
	}
	if hm.HomeZones.Attacking != 100 || hm.AwayZones.Defensive != 100 {
		t.Errorf("passthrough zones wrong: %+v / %+v", hm.HomeZones, hm.AwayZones)
	}
	// Fewer than 8 home samples falls back to synthesis.
	short := map[string][][2]float64{"home": touchesH[:3], "away": touchesA}
	hm2 := GenerateTouchHeatmap(home, away, 50, rng, short)
	if len(hm2.HomePoints) != 60 {
		t.Errorf("short sample should synthesize 60/60, got %d", len(hm2.HomePoints))
	}
	// 150-per-side cap.
	var big [][2]float64
	for i := 0; i < 200; i++ {
		big = append(big, [2]float64{0.5, 0.5})
	}
	hm3 := GenerateTouchHeatmap(home, away, 50, rng, map[string][][2]float64{"home": big, "away": big})
	if len(hm3.HomePoints) != 150 || len(hm3.AwayPoints) != 150 {
		t.Errorf("passthrough should cap at 150: %d/%d", len(hm3.HomePoints), len(hm3.AwayPoints))
	}
}
