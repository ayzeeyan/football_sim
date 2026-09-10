package tournament

import (
	"testing"

	"football_sim/pkg/models"
	"football_sim/pkg/transfers"
)

func TestWonderkidPermanentTransferSurvivesResetNewSeason(t *testing.T) {
	tm, ge := loadTestUniverse(t)
	te := transfers.NewTransferEngine(tm.ClubsList, tm.Managers, 99)
	tm.TransferEngine = te
	tm.GrowthEngine = ge

	var wk *models.Player
	var originalClub, destination *models.Club
	for _, club := range tm.ClubsList {
		for _, p := range club.Squad {
			if transfers.IsCanonicalWonderkid(p) {
				wk = p
				originalClub = club
				break
			}
		}
		if wk != nil { break }
	}
	if wk == nil || originalClub == nil { t.Fatal("no canonical wonderkid found in test universe") }
	for _, club := range tm.ClubsList {
		if club.ClubID != originalClub.ClubID {
			destination = club
			break
		}
	}
	if destination == nil { t.Fatal("no destination club found") }

	originalID := wk.OriginalClubID
	if originalID == "" { originalID = originalClub.ClubID; wk.OriginalClubID = originalID }
	var remaining []*models.Player
	for _, p := range originalClub.Squad {
		if p.PlayerID != wk.PlayerID { remaining = append(remaining, p) }
	}
	originalClub.Squad = remaining
	wk.ClubID = destination.ClubID
	destination.Squad = append(destination.Squad, wk)
	te.TransferredThisWindow[wk.PlayerID] = true

	res := tm.ResetNewSeason()
	if res["status"] != "success" { t.Fatalf("reset response: %v", res) }
	if wk.ClubID != destination.ClubID { t.Fatalf("wonderkid club reverted from %s to %s", destination.ClubID, wk.ClubID) }
	if wk.OriginalClubID != originalID { t.Fatalf("historical OriginalClubID changed from %s to %s", originalID, wk.OriginalClubID) }

	foundDestination, foundOriginal := false, false
	seen := map[string]string{}
	for _, club := range tm.ClubsList {
		for _, p := range club.Squad {
			if prev, ok := seen[p.PlayerID]; ok { t.Fatalf("duplicate player %s in %s and %s", p.PlayerID, prev, club.ClubID) }
			seen[p.PlayerID] = club.ClubID
			if p.PlayerID == wk.PlayerID {
				if club.ClubID == destination.ClubID { foundDestination = true }
				if club.ClubID == originalClub.ClubID { foundOriginal = true }
			}
		}
	}
	if !foundDestination || foundOriginal { t.Fatalf("permanent wonderkid transfer did not survive rollover: destination=%v original=%v", foundDestination, foundOriginal) }
}
