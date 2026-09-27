package tournament

import (
	"sort"

	"football_sim/pkg/models"
)

// EnforceRosterCapsUnlocked redistributes players from clubs above the
// senior-squad ceiling. Canonical wonderkids and captains stay put; the
// lowest-rated remaining players move first to clubs with spare slots.
func (tm *TournamentManager) EnforceRosterCapsUnlocked() {
	if tm == nil {
		return
	}
	for round := 0; round < 8; round++ {
		moved := false
		oversized := make([]*models.Club, 0)
		for _, club := range tm.ClubsList {
			if club != nil && len(club.Squad) > models.MaxSeniorSquadSize {
				oversized = append(oversized, club)
			}
		}
		if len(oversized) == 0 {
			return
		}
		sort.Slice(oversized, func(i, j int) bool { return oversized[i].ClubID < oversized[j].ClubID })
		for _, club := range oversized {
			cands := overflowCandidates(club)
			for _, p := range cands {
				if len(club.Squad) <= models.MaxSeniorSquadSize {
					break
				}
				dest := pickUnderCapacityClub(p, club, tm.ClubsList)
				if dest == nil {
					continue
				}
				if !removePlayerFromClub(club, p) {
					continue
				}
				p.ClubID = dest.ClubID
				dest.Squad = append(dest.Squad, p)
				moved = true
			}
		}
		if !moved {
			break
		}
	}
	for _, club := range tm.ClubsList {
		if club == nil || len(club.Squad) <= models.MaxSeniorSquadSize {
			if club != nil {
				club.SquadSize = len(club.Squad)
				club.RecalculateRatings()
			}
			continue
		}
		cands := overflowCandidates(club)
		need := len(club.Squad) - models.MaxSeniorSquadSize
		if need > len(cands) {
			need = len(cands)
		}
		drop := make(map[string]bool, need)
		for i := 0; i < need; i++ {
			drop[cands[i].PlayerID] = true
		}
		kept := make([]*models.Player, 0, models.MaxSeniorSquadSize)
		for _, p := range club.Squad {
			if p == nil || drop[p.PlayerID] {
				continue
			}
			kept = append(kept, p)
		}
		club.Squad = kept
		club.SquadSize = len(club.Squad)
		club.RecalculateRatings()
	}
}

func overflowCandidates(club *models.Club) []*models.Player {
	if club == nil {
		return nil
	}
	out := make([]*models.Player, 0, len(club.Squad))
	for _, p := range club.Squad {
		if p == nil || p.UniverseWonderkid || models.IsCanonicalWonderkidID(p.PlayerID) {
			continue
		}
		if p.PlayerID == club.CaptainID || p.PlayerID == club.ViceCaptainID {
			continue
		}
		out = append(out, p)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].OVR != out[j].OVR {
			return out[i].OVR < out[j].OVR
		}
		if out[i].Age != out[j].Age {
			return out[i].Age > out[j].Age
		}
		return out[i].PlayerID < out[j].PlayerID
	})
	return out
}

func pickUnderCapacityClub(p *models.Player, from *models.Club, clubs []*models.Club) *models.Club {
	if p == nil || from == nil {
		return nil
	}
	var dests []*models.Club
	for _, club := range clubs {
		if club == nil || club.ClubID == from.ClubID {
			continue
		}
		if len(club.Squad) >= models.MaxSeniorSquadSize {
			continue
		}
		if models.IsCanonicalWonderkidID(p.PlayerID) && !models.IsDesignatedWonderkidClubID(club.ClubID) {
			continue
		}
		dests = append(dests, club)
	}
	sort.SliceStable(dests, func(i, j int) bool {
		sameI := from != nil && dests[i].League == from.League
		sameJ := from != nil && dests[j].League == from.League
		if sameI != sameJ {
			return sameI
		}
		if len(dests[i].Squad) != len(dests[j].Squad) {
			return len(dests[i].Squad) < len(dests[j].Squad)
		}
		return dests[i].ClubID < dests[j].ClubID
	})
	if len(dests) == 0 {
		return nil
	}
	return dests[0]
}

func removePlayerFromClub(club *models.Club, player *models.Player) bool {
	if club == nil || player == nil {
		return false
	}
	kept := make([]*models.Player, 0, len(club.Squad))
	found := false
	for _, p := range club.Squad {
		if p == nil {
			continue
		}
		if p.PlayerID == player.PlayerID {
			found = true
			continue
		}
		kept = append(kept, p)
	}
	if !found {
		return false
	}
	club.Squad = kept
	club.SquadSize = len(kept)
	return true
}
