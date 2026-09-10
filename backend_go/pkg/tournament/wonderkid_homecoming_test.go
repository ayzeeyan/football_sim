package tournament

import (
	"testing"

	"football_sim/pkg/models"
	"football_sim/pkg/transfers"
)

func TestWonderkidHomecomingOnResetNewSeason(t *testing.T) {
	tm, ge := loadTestUniverse(t)
	te := transfers.NewTransferEngine(tm.ClubsList, tm.Managers, 99)
	tm.TransferEngine = te
	tm.GrowthEngine = ge

	// Locate a canonical wonderkid and a regular player
	var wk *models.Player
	var parentClub, loanClub *models.Club
	for _, club := range tm.ClubsList {
		for _, p := range club.Squad {
			if p.UniverseWonderkid && wk == nil {
				wk = p
				parentClub = club
				break
			}
		}
	}
	if wk == nil || parentClub == nil {
		t.Fatal("no canonical wonderkid found in test universe")
	}

	for _, club := range tm.ClubsList {
		if club.ClubID != parentClub.ClubID {
			loanClub = club
			break
		}
	}

	// Move wonderkid from parentClub to loanClub
	te.TriggerSpecificBid(loanClub.ClubID, parentClub.ClubID, wk.PlayerID)
	// Execute the transfer directly
	var newParentSquad []*models.Player
	for _, p := range parentClub.Squad {
		if p.PlayerID != wk.PlayerID {
			newParentSquad = append(newParentSquad, p)
		}
	}
	parentClub.Squad = newParentSquad
	wk.ClubID = loanClub.ClubID
	loanClub.Squad = append(loanClub.Squad, wk)

	// Also perform a permanent transfer for a non-wonderkid player
	var permPlayer *models.Player
	for _, p := range loanClub.Squad {
		if !p.UniverseWonderkid && permPlayer == nil {
			permPlayer = p
			break
		}
	}
	if permPlayer != nil {
		permPlayer.OriginalClubID = parentClub.ClubID
		permPlayer.ClubID = parentClub.ClubID
		parentClub.Squad = append(parentClub.Squad, permPlayer)
		// Remove from loanClub
		var newLoanSquad []*models.Player
		for _, p := range loanClub.Squad {
			if p.PlayerID != permPlayer.PlayerID {
				newLoanSquad = append(newLoanSquad, p)
			}
		}
		loanClub.Squad = newLoanSquad
	}

	// Fast-forward or trigger ResetNewSeason
	tm.ResetNewSeason()

	// Verify wonderkid returned to parentClub
	foundInParent := false
	for _, p := range parentClub.Squad {
		if p.PlayerID == wk.PlayerID {
			foundInParent = true
			break
		}
	}
	if !foundInParent {
		t.Errorf("Wonderkid %s did not return to parent club %s after ResetNewSeason", wk.FullName, parentClub.ClubID)
	}

	foundInLoan := false
	for _, p := range loanClub.Squad {
		if p.PlayerID == wk.PlayerID {
			foundInLoan = true
			break
		}
	}
	if foundInLoan {
		t.Errorf("Wonderkid %s is still in loan club %s after ResetNewSeason", wk.FullName, loanClub.ClubID)
	}

	// Verify permanent player did NOT return
	if permPlayer != nil {
		foundPermInParent := false
		for _, p := range parentClub.Squad {
			if p.PlayerID == permPlayer.PlayerID {
				foundPermInParent = true
				break
			}
		}
		if !foundPermInParent {
			t.Errorf("Permanent transfer %s should remain at %s, but was not found", permPlayer.FullName, parentClub.ClubID)
		}
	}

	// Verify zero duplicates across all squads
	seen := make(map[string]string)
	for _, c := range tm.ClubsList {
		for _, p := range c.Squad {
			if prevClub, ok := seen[p.PlayerID]; ok {
				t.Fatalf("Duplicate player %s found in %s and %s", p.FullName, prevClub, c.ClubID)
			}
			seen[p.PlayerID] = c.ClubID
		}
	}
}
