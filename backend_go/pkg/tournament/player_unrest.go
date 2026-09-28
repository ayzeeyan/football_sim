package tournament

import (
	"fmt"
	"sort"

	"football_sim/pkg/models"
)

// Player unrest, tournament layer: the weekly morale cost, the inbox story,
// and the observational wire views. The pure assessment lives in
// pkg/models (models.AssessUnrest) so the transfer market can read the same
// derived state without an import cycle. Nothing here is persisted.

// PlayerUnrest is the wire view of one player's unrest.
type PlayerUnrest struct {
	PlayerID        string            `json:"player_id"`
	FullName        string            `json:"full_name"`
	ClubID          string            `json:"club_id"`
	ClubName        string            `json:"club_name"`
	Level           models.UnrestLevel `json:"level"`
	Reason          string            `json:"reason"`
	SquadRank       int               `json:"squad_rank"`
	AppearanceShare float64           `json:"appearance_share"`
}

// playerUnrestForClub computes the unrest rows for one club's squad.
func playerUnrestForClub(club *models.Club) []PlayerUnrest {
	if club == nil || club.Played < 6 {
		return nil
	}
	rank := models.SquadRankByRating(club)
	var rows []PlayerUnrest
	for _, p := range club.Squad {
		if p == nil {
			continue
		}
		assessment := models.AssessUnrest(club, p, rank[p.PlayerID])
		if assessment == nil {
			continue
		}
		rows = append(rows, PlayerUnrest{
			PlayerID: p.PlayerID, FullName: p.FullName,
			ClubID: club.ClubID, ClubName: club.ClubName,
			Level: assessment.Level, Reason: assessment.Reason,
			SquadRank: assessment.SquadRank, AppearanceShare: assessment.AppearanceShare,
		})
	}
	// Stable order: most severe first, then by rating rank.
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Level != rows[j].Level {
			return rows[i].Level == models.UnrestWantsToLeave
		}
		return rows[i].SquadRank < rows[j].SquadRank
	})
	return rows
}

// PlayerUnrestForClub returns the derived unrest rows for one club.
func (tm *TournamentManager) PlayerUnrestForClub(clubID string) []PlayerUnrest {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return playerUnrestForClub(tm.Clubs[clubID])
}

// PlayerUnrestLevel reports one player's derived unrest at their club.
func (tm *TournamentManager) PlayerUnrestLevel(playerID string) (models.UnrestLevel, string) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	for _, club := range tm.ClubsList {
		if club == nil {
			continue
		}
		for _, row := range playerUnrestForClub(club) {
			if row.PlayerID == playerID {
				return row.Level, row.Reason
			}
		}
	}
	return models.UnrestContent, ""
}

// evaluatePlayerUnrestUnlocked runs in the weekly tick: it applies the
// morale cost of being starved and publishes the inbox story when a
// top-quality player's patience runs out. Caller must hold tm.mu.
func (tm *TournamentManager) evaluatePlayerUnrestUnlocked(completedMW int) {
	published := 0
	for _, club := range tm.ClubsList {
		for _, row := range playerUnrestForClub(club) {
			player := tm.playerByIDUnlocked(row.PlayerID)
			if player == nil {
				continue
			}
			// The morale cost: wanting out grinds a player down.
			drop := 2
			if row.Level == models.UnrestWantsToLeave {
				drop = 4
			}
			player.Morale -= drop
			if player.Morale < 20 {
				player.Morale = 20
			}
			// Publish at most one story per week: the loudest case.
			if row.Level == models.UnrestWantsToLeave && published == 0 {
				tm.PushInbox(
					MsgCategoryTransfer,
					fmt.Sprintf("%s unhappy with role at %s", row.FullName, row.ClubName),
					row.Reason+" Agents are already working the phones.",
					completedMW,
					[]string{row.ClubID}, row.PlayerID, "",
				)
				published++
			}
		}
	}
}

// playerByIDUnlocked finds a live player anywhere in the world.
func (tm *TournamentManager) playerByIDUnlocked(playerID string) *models.Player {
	for _, club := range tm.ClubsList {
		if club == nil {
			continue
		}
		for _, p := range club.Squad {
			if p != nil && p.PlayerID == playerID {
				return p
			}
		}
	}
	if tm.TransferEngine != nil {
		for _, p := range tm.TransferEngine.FreeAgents {
			if p != nil && p.PlayerID == playerID {
				return p
			}
		}
	}
	return nil
}
