package tournament

import (
	"sort"

	"football_sim/pkg/models"
)

// AssignSquadRolesUnlocked stamps each player with a playing-time expectation
// based on OVR rank, age, and wonderkid status. Deterministic for a given squad.
func (tm *TournamentManager) AssignSquadRolesUnlocked() {
	for _, club := range tm.ClubsList {
		assignClubSquadRoles(club)
	}
}

func assignClubSquadRoles(club *models.Club) {
	if club == nil || len(club.Squad) == 0 {
		return
	}
	squad := append([]*models.Player(nil), club.Squad...)
	sort.SliceStable(squad, func(i, j int) bool {
		if squad[i].OVR != squad[j].OVR {
			return squad[i].OVR > squad[j].OVR
		}
		if squad[i].Age != squad[j].Age {
			return squad[i].Age < squad[j].Age
		}
		return squad[i].PlayerID < squad[j].PlayerID
	})
	for i, p := range squad {
		p.SquadRole = roleForRank(p, i, len(squad))
		if p.Morale == 0 {
			p.Morale = 70
		}
		if p.Fitness == 0 {
			p.Fitness = 80
		}
		if p.Sharpness == 0 {
			p.Sharpness = 65
		}
		p.ClampDynamics()
	}
}

func roleForRank(p *models.Player, rank, n int) string {
	if (p.UniverseWonderkid || p.Age <= 18) && rank >= 2 {
		return models.RoleProspect
	}
	switch {
	case rank <= 1:
		return models.RoleCrucial
	case rank <= 5:
		return models.RoleImportant
	case rank <= 11:
		return models.RoleRotation
	case rank <= n-4:
		return models.RoleSquad
	default:
		if p.Age <= 21 {
			return models.RoleProspect
		}
		return models.RoleSquad
	}
}
