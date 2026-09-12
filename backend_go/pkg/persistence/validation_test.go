package persistence

import (
	"path/filepath"
	"testing"

	"football_sim/pkg/models"
	"football_sim/pkg/transfers"
)

func TestValidateCareerSnapshotAllowsLegacyMissingAdditiveFields(t *testing.T) {
	_, ge, tm, te := setupTestWorld(t)
	snap := BuildSnapshot(tm, ge, te)

	// Model an older save that predates recently-added transfer continuity
	// fields. Zero values must be accepted and RestoreCareer should retain the
	// fresh engine's safe defaults.
	snap.Version = 1
	snap.SeasonPhase = ""
	snap.CurrentMatchweek = 0
	snap.MaxMatchweeks = 0
	snap.Transfers.CurrentWeek = 0
	snap.Transfers.CurrentDay = 0
	snap.Transfers.CurrentMatchweek = 0
	snap.Transfers.TransferredThisWindow = nil

	if err := ValidateCareerSnapshot(snap); err != nil {
		t.Fatalf("legacy-compatible snapshot rejected: %v", err)
	}

	_, freshGE, freshTM, freshTE := setupTestWorld(t)
	if err := RestoreCareer(freshTM, freshGE, freshTE, snap); err != nil {
		t.Fatalf("legacy-compatible restore failed: %v", err)
	}
	if freshTM.SeasonPhase != "season" || freshTM.CurrentMatchweek != 1 || freshTE.CurrentWeek != 1 {
		t.Fatalf("safe defaults not preserved: phase=%q mw=%d transfer_week=%d", freshTM.SeasonPhase, freshTM.CurrentMatchweek, freshTE.CurrentWeek)
	}
	freshTM.TransferEngine = freshTE
	if err := freshTM.ValidateWorldState(); err != nil {
		t.Fatalf("legacy-compatible restored world failed validation: %v", err)
	}
}

func TestValidateCareerSnapshotRejectsCriticalCorruption(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*CareerSnapshot)
	}{
		{"unknown_phase", func(s *CareerSnapshot) { s.SeasonPhase = "broken_phase" }},
		{"negative_matchweek", func(s *CareerSnapshot) { s.CurrentMatchweek = -1 }},
		{"impossible_transfer_week", func(s *CareerSnapshot) { s.Transfers.CurrentWeek = 99 }},
		{"negative_player_stats", func(s *CareerSnapshot) {
			for _, club := range s.Clubs {
				if len(club.Squad) > 0 {
					club.Squad[0].Goals = -1
					return
				}
			}
		}},
		{"impossible_fixture", func(s *CareerSnapshot) {
			if len(s.Fixtures) > 0 {
				s.Fixtures[0].AwayID = s.Fixtures[0].HomeID
			}
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, ge, tm, te := setupTestWorld(t)
			snap := BuildSnapshot(tm, ge, te)
			tc.mutate(snap)
			if err := ValidateCareerSnapshot(snap); err == nil {
				t.Fatal("expected corrupted snapshot to be rejected")
			}
		})
	}
}

func TestCareerRoundTripPreservesLogicalContinuityAndValidates(t *testing.T) {
	_, ge, tm, te := setupTestWorld(t)
	tm.TransferEngine = te

	for mw := 1; mw <= 3; mw++ {
		res := tm.SimulateMatchweek(mw)
		if res["status"] != "success" {
			t.Fatalf("simulate matchweek %d failed: %v", mw, res)
		}
	}
	te.CurrentWeek = 4
	te.CurrentDay = 22
	te.CurrentMatchweek = tm.CurrentMatchweek
	te.TransferredThisWindow[tm.ClubsList[0].Squad[0].PlayerID] = true
	managerName := tm.Managers[tm.ClubsList[0].ClubID].Name
	managerHistoryLen := len(tm.ManagerHistory)
	seasonHistoryLen := len(tm.SeasonHistory)

	path := filepath.Join(t.TempDir(), "roundtrip.json")
	if _, err := SaveCareer(tm, ge, te, path); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	loaded, err := LoadCareer(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if err := ValidateCareerSnapshot(loaded); err != nil {
		t.Fatalf("saved snapshot failed validation: %v", err)
	}

	_, restoredGE, restoredTM, restoredTE := setupTestWorld(t)
	if err := RestoreCareer(restoredTM, restoredGE, restoredTE, loaded); err != nil {
		t.Fatalf("restore failed: %v", err)
	}
	restoredTM.TransferEngine = restoredTE
	if err := restoredTM.ValidateWorldState(); err != nil {
		t.Fatalf("restored world failed validation: %v", err)
	}

	if restoredTM.SeasonName != tm.SeasonName || restoredTM.SeasonPhase != tm.SeasonPhase || restoredTM.CurrentMatchweek != tm.CurrentMatchweek {
		t.Fatalf("season continuity mismatch: got %s/%s/MW%d want %s/%s/MW%d", restoredTM.SeasonName, restoredTM.SeasonPhase, restoredTM.CurrentMatchweek, tm.SeasonName, tm.SeasonPhase, tm.CurrentMatchweek)
	}
	if restoredTE.CurrentWeek != 4 || restoredTE.CurrentDay != 22 || restoredTE.CurrentMatchweek != tm.CurrentMatchweek {
		t.Fatalf("transfer continuity mismatch: week=%d day=%d mw=%d", restoredTE.CurrentWeek, restoredTE.CurrentDay, restoredTE.CurrentMatchweek)
	}
	if len(restoredTM.ManagerHistory) != managerHistoryLen || len(restoredTM.SeasonHistory) != seasonHistoryLen {
		t.Fatalf("history continuity mismatch: manager=%d/%d season=%d/%d", len(restoredTM.ManagerHistory), managerHistoryLen, len(restoredTM.SeasonHistory), seasonHistoryLen)
	}
	if got := restoredTM.Managers[restoredTM.ClubsList[0].ClubID].Name; got != managerName {
		t.Fatalf("manager continuity mismatch: got %q want %q", got, managerName)
	}
	if !restoredTE.TransferredThisWindow[tm.ClubsList[0].Squad[0].PlayerID] {
		t.Fatal("transferred-this-window continuity was lost")
	}
}

func TestValidateCareerSnapshotExact12Clubs(t *testing.T) {
	_, ge, tm, te := setupTestWorld(t)
	snap := BuildSnapshot(tm, ge, te)

	for k := range snap.Clubs {
		delete(snap.Clubs, k)
		break
	}
	if err := ValidateCareerSnapshot(snap); err == nil {
		t.Fatal("expected validation error for 11 clubs")
	}
}

func TestValidateCareerSnapshotWonderkidPotentialBounds(t *testing.T) {
	_, ge, tm, te := setupTestWorld(t)
	snap := BuildSnapshot(tm, ge, te)

	for pid, bio := range snap.Growth.Biometrics {
		if models.IsCanonicalWonderkidID(pid) {
			bio.Potential = 99
			if err := ValidateCareerSnapshot(snap); err == nil {
				t.Fatal("expected validation error for wonderkid potential 99")
			}
			bio.Potential = 92
			if err := ValidateCareerSnapshot(snap); err == nil {
				t.Fatal("expected validation error for wonderkid potential 92")
			}
			bio.Potential = 95
			if err := ValidateCareerSnapshot(snap); err != nil {
				t.Fatalf("unexpected error for wonderkid potential 95: %v", err)
			}
			break
		}
	}
}

func TestValidateCareerSnapshotAllowsExpiredNegotiationsAndHistoricalBuyers(t *testing.T) {
	_, ge, tm, te := setupTestWorld(t)
	snap := BuildSnapshot(tm, ge, te)

	snap.Transfers.Completed = append(snap.Transfers.Completed, transfers.CompletedTransfer{
		PlayerID: "P_FOREIGN_1",
		FeeEUR:   15000000,
		SellerID: "BUN-BVB",
		BuyerID:  tm.ClubsList[0].ClubID,
	})

	snap.Transfers.ActiveNegotiations = append(snap.Transfers.ActiveNegotiations,
		&transfers.TransferNegotiation{
			NegotiationID: "NEG_EXP_1",
			StageName:     "expired",
			CurrentBid:    5000000,
			AskingPrice:   10000000,
		},
		&transfers.TransferNegotiation{
			NegotiationID: "NEG_REJ_1",
			StageName:     "rejected",
			CurrentBid:    3000000,
			AskingPrice:   8000000,
		},
	)

	if err := ValidateCareerSnapshot(snap); err != nil {
		t.Fatalf("expected valid snapshot with historical buyer and expired negotiations, got: %v", err)
	}
}
