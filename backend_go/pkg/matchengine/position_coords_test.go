package matchengine

import (
	"testing"

	"football_sim/pkg/managers"
	"football_sim/pkg/models"
)

func TestPositionDrivenCoordinates(t *testing.T) {
	// Create a club with distinct tactical positions:
	// GK, LB, CB, CB, RB, CDM, CM, CAM, LW, ST, RW
	customPositions := []string{"GK", "LB", "CB", "CB", "RB", "CDM", "CM", "CAM", "LW", "ST", "RW"}
	customCategories := []string{"GK", "DEF", "DEF", "DEF", "DEF", "MID", "MID", "MID", "FWD", "FWD", "FWD"}

	squad := make([]*models.Player, 11)
	for i := 0; i < 11; i++ {
		squad[i] = &models.Player{
			PlayerID: models.FormatCurrency(int64(i + 100)),
			FullName: "Player " + customPositions[i],
			Position: customPositions[i],
			Category: customCategories[i],
			OVR:      80,
		}
	}

	homeClub := &models.Club{
		ClubID:      "TEST-HOME",
		ClubName:    "Home FC",
		ShortName:   "HME",
		HomeStadium: "Home Arena",
		Squad:       squad,
	}
	awayClub := &models.Club{
		ClubID:      "TEST-AWAY",
		ClubName:    "Away FC",
		ShortName:   "AWY",
		HomeStadium: "Away Arena",
		Squad:       squad,
	}
	mgr := &managers.ManagerProfile{Name: "Boss", Style: "possession"}

	engine := NewLiveMatchEngine(homeClub, awayClub, mgr, mgr, 12345)

	if len(engine.HomePlayers) != 11 {
		t.Fatalf("expected 11 home players, got %d", len(engine.HomePlayers))
	}

	// Verify coordinates for key positions on home side:
	// 1. CAM should be in central attacking midfield: X > CDM and X > CM, but X < ST; Y near center 0.50
	var cam, cdm, cm, st, lb, rb, lw, rw *LivePlayerRadar
	for i := range engine.HomePlayers {
		p := &engine.HomePlayers[i]
		switch p.Position {
		case "CAM":
			cam = p
		case "CDM":
			cdm = p
		case "CM":
			cm = p
		case "ST":
			st = p
		case "LB":
			lb = p
		case "RB":
			rb = p
		case "LW":
			lw = p
		case "RW":
			rw = p
		}
	}

	if cam == nil || cdm == nil || st == nil || cm == nil {
		t.Fatal("missing key positions in engine.HomePlayers")
	}

	// CDM must be deeper than CAM (lower X for home team attacking left-to-right)
	if cdm.X >= cam.X {
		t.Errorf("expected CDM (X=%f) to be deeper than CAM (X=%f)", cdm.X, cam.X)
	}

	// CAM must be behind ST
	if cam.X >= st.X {
		t.Errorf("expected CAM (X=%f) to be behind ST (X=%f)", cam.X, st.X)
	}

	// CAM must be centrally positioned (Y near 0.50)
	if cam.Y < 0.40 || cam.Y > 0.60 {
		t.Errorf("expected CAM to be centrally positioned, got Y=%f", cam.Y)
	}

	// Flanks must be spaced wide (not clustered on left wing)
	if lb != nil && rb != nil {
		if lb.Y >= 0.30 || rb.Y <= 0.70 {
			t.Errorf("expected LB (Y=%f) on left and RB (Y=%f) on right flank", lb.Y, rb.Y)
		}
	}
	if lw != nil && rw != nil {
		if lw.Y >= 0.30 || rw.Y <= 0.70 {
			t.Errorf("expected LW (Y=%f) on left and RW (Y=%f) on right wing", lw.Y, rw.Y)
		}
	}

	// Now run multiple ticks and ensure positions don't snap back to left wing
	engine.State = "PLAYING"
	for tick := 0; tick < 100; tick++ {
		engine.Tick(1.0)
	}

	// Check CAM is still centrally attacking
	for i := range engine.HomePlayers {
		p := &engine.HomePlayers[i]
		if p.Position == "CAM" {
			if p.Y < 0.35 || p.Y > 0.65 {
				t.Errorf("after simulation ticks, CAM drifted off center: Y=%f", p.Y)
			}
			if p.X < 0.40 || p.X > 0.60 {
				t.Errorf("after simulation ticks, CAM drifted from attacking midfield: X=%f", p.X)
			}
		}
	}
}

func TestDuplicatePositionStaggering(t *testing.T) {
	// 3 CBs and 3 CMs
	players := []*models.Player{
		{PlayerID: "1", FullName: "GK", Position: "GK", Category: "GK"},
		{PlayerID: "2", FullName: "CB1", Position: "CB", Category: "DEF"},
		{PlayerID: "3", FullName: "CB2", Position: "CB", Category: "DEF"},
		{PlayerID: "4", FullName: "CB3", Position: "CB", Category: "DEF"},
		{PlayerID: "5", FullName: "CM1", Position: "CM", Category: "MID"},
		{PlayerID: "6", FullName: "CM2", Position: "CM", Category: "MID"},
		{PlayerID: "7", FullName: "CM3", Position: "CM", Category: "MID"},
		{PlayerID: "8", FullName: "LW", Position: "LW", Category: "FWD"},
		{PlayerID: "9", FullName: "RW", Position: "RW", Category: "FWD"},
		{PlayerID: "10", FullName: "ST1", Position: "ST", Category: "FWD"},
		{PlayerID: "11", FullName: "ST2", Position: "ST", Category: "FWD"},
	}

	radar := radarPlayersPositional(players, positionHomeCoords)
	if len(radar) != 11 {
		t.Fatalf("expected 11 radar players, got %d", len(radar))
	}

	// Check that 3 CBs have distinct Y coordinates (staggered)
	var cbYs []float64
	for _, r := range radar {
		if r.Position == "CB" {
			cbYs = append(cbYs, r.Y)
		}
	}
	if len(cbYs) != 3 {
		t.Fatalf("expected 3 CBs, got %d", len(cbYs))
	}
	if cbYs[0] == cbYs[1] || cbYs[1] == cbYs[2] || cbYs[0] == cbYs[2] {
		t.Errorf("CBs are overlapping: %v", cbYs)
	}

	// Check that 2 STs have distinct Y coordinates
	var stYs []float64
	for _, r := range radar {
		if r.Position == "ST" {
			stYs = append(stYs, r.Y)
		}
	}
	if len(stYs) != 2 {
		t.Fatalf("expected 2 STs, got %d", len(stYs))
	}
	if stYs[0] == stYs[1] {
		t.Errorf("STs are overlapping: %v", stYs)
	}
}
