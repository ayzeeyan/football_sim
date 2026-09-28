package tournament

import (
	"fmt"
	"sort"

	"football_sim/pkg/models"
)

// Tier B national team control (B6): the viewer picks one nation's squad
// for the Nations Cup. Eligibility is unchanged — a player is only callable
// if their original club's country matches the nation — and the AI
// selection remains the default for every other nation.

// NationalSquadSize is the rigid squad contract: every Nations Cup squad,
// AI-selected or viewer-selected, carries exactly this many players.
const NationalSquadSize = 23

// nationalCandidatesByCountryUnlocked collects the eligibility pool per
// country. A player is eligible for the country of their immutable original
// club (falling back to the current club). Caller must hold tm.mu.
func (tm *TournamentManager) nationalCandidatesByCountryUnlocked() map[string][]nationalPlayerCandidate {
	playersByCountry := make(map[string][]nationalPlayerCandidate)
	seen := make(map[string]bool)
	for _, club := range tm.ClubsList {
		if club == nil {
			continue
		}
		for _, player := range club.Squad {
			if player == nil || player.PlayerID == "" || seen[player.PlayerID] {
				continue
			}
			seen[player.PlayerID] = true
			originID := player.OriginalClubID
			if originID == "" {
				originID = player.ClubID
			}
			origin := tm.Clubs[originID]
			if origin == nil {
				origin = club
			}
			if origin.Country == "" {
				continue
			}
			playersByCountry[origin.Country] = append(playersByCountry[origin.Country], nationalPlayerCandidate{player: player})
		}
	}
	if tm.TransferEngine != nil {
		for _, player := range tm.TransferEngine.FreeAgents {
			if player == nil || player.PlayerID == "" || seen[player.PlayerID] {
				continue
			}
			seen[player.PlayerID] = true
			originID := player.OriginalClubID
			if originID == "" {
				originID = player.ClubID
			}
			origin := tm.Clubs[originID]
			if origin != nil && origin.Country != "" {
				playersByCountry[origin.Country] = append(playersByCountry[origin.Country], nationalPlayerCandidate{player: player})
			}
		}
	}
	return playersByCountry
}

// sortNationalCandidates orders a pool deterministically: OVR descending,
// then younger first, then player ID.
func sortNationalCandidates(rows []nationalPlayerCandidate) {
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].player.OVR != rows[j].player.OVR {
			return rows[i].player.OVR > rows[j].player.OVR
		}
		if rows[i].player.Age != rows[j].player.Age {
			return rows[i].player.Age < rows[j].player.Age
		}
		return rows[i].player.PlayerID < rows[j].player.PlayerID
	})
}

// validateNationalSquadSelection checks the rigid squad contract against a
// pool: exactly 23 unique eligible players with at least one goalkeeper.
func validateNationalSquadSelection(ids []string, pool []nationalPlayerCandidate) string {
	if len(ids) != NationalSquadSize {
		return fmt.Sprintf("A Nations Cup squad needs exactly %d players; %d were given.", NationalSquadSize, len(ids))
	}
	eligible := make(map[string]bool, len(pool))
	for _, row := range pool {
		if row.player != nil {
			eligible[row.player.PlayerID] = true
		}
	}
	seen := make(map[string]bool, len(ids))
	goalkeepers := 0
	for _, id := range ids {
		if !eligible[id] {
			return fmt.Sprintf("Player %s is not eligible for this nation.", id)
		}
		if seen[id] {
			return fmt.Sprintf("Player %s was selected twice.", id)
		}
		seen[id] = true
	}
	for _, row := range pool {
		if !seen[row.player.PlayerID] {
			continue
		}
		if models.GetPositionCategory(row.player.Position) == "GK" {
			goalkeepers++
		}
	}
	if goalkeepers == 0 {
		return "A Nations Cup squad needs at least one goalkeeper."
	}
	return ""
}

// applyViewerNationalSquadsUnlocked re-applies persisted viewer selections
// after the competition is rebuilt for a new season. Selections that no
// longer satisfy the contract (retired or transferred-ineligible players)
// fall back to the AI selection. Caller must hold tm.mu.
func (tm *TournamentManager) applyViewerNationalSquadsUnlocked(competition *NationalTeamsCompetition, poolByCountry map[string][]nationalPlayerCandidate) {
	if competition == nil || len(competition.ViewerSquads) == 0 {
		return
	}
	teamIDs := make([]string, 0, len(competition.ViewerSquads))
	for teamID := range competition.ViewerSquads {
		teamIDs = append(teamIDs, teamID)
	}
	sort.Strings(teamIDs)
	for _, teamID := range teamIDs {
		team := competition.Teams[teamID]
		if team == nil {
			delete(competition.ViewerSquads, teamID)
			continue
		}
		ids := competition.ViewerSquads[teamID]
		pool := poolByCountry[team.Country]
		if msg := validateNationalSquadSelection(ids, pool); msg != "" {
			delete(competition.ViewerSquads, teamID)
			continue
		}
		team.PlayerIDs = append([]string(nil), ids...)
		team.Rating = nationalSquadRating(team.PlayerIDs, pool)
	}
}

// SetNationalSquad installs a viewer-selected squad for one nation.
func (tm *TournamentManager) SetNationalSquad(teamID string, playerIDs []string) (string, map[string]interface{}) {
	if tm == nil {
		return "No world is loaded.", nil
	}
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if tm.World == nil || tm.World.NationalTeams == nil {
		return "The Nations Cup has not been drawn yet.", nil
	}
	competition := tm.World.NationalTeams
	team := competition.Teams[teamID]
	if team == nil {
		return "National team not found.", nil
	}
	poolByCountry := tm.nationalCandidatesByCountryUnlocked()
	pool := poolByCountry[team.Country]
	if msg := validateNationalSquadSelection(playerIDs, pool); msg != "" {
		return msg, nil
	}
	team.PlayerIDs = append([]string(nil), playerIDs...)
	team.Rating = nationalSquadRating(team.PlayerIDs, pool)
	if competition.ViewerSquads == nil {
		competition.ViewerSquads = map[string][]string{}
	}
	competition.ViewerSquads[teamID] = append([]string(nil), playerIDs...)

	tm.PushInbox(
		MsgCategoryCup,
		fmt.Sprintf("Nations Cup squad announced: %s", team.Name),
		fmt.Sprintf("%s name a %d-player squad for the Nations Cup. %d of the call-ups come from the viewer's list.", team.Name, NationalSquadSize, len(playerIDs)),
		tm.CurrentMatchweek,
		[]string{}, "", "",
	)
	return "", tm.compactNationalTeamUnlocked(team, true)
}

// ClearNationalSquad returns one nation to the AI selection.
func (tm *TournamentManager) ClearNationalSquad(teamID string) (string, map[string]interface{}) {
	if tm == nil {
		return "No world is loaded.", nil
	}
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if tm.World == nil || tm.World.NationalTeams == nil {
		return "The Nations Cup has not been drawn yet.", nil
	}
	competition := tm.World.NationalTeams
	team := competition.Teams[teamID]
	if team == nil {
		return "National team not found.", nil
	}
	if len(competition.ViewerSquads[teamID]) == 0 {
		return "The squad is already AI-selected.", nil
	}
	delete(competition.ViewerSquads, teamID)
	pool := tm.nationalCandidatesByCountryUnlocked()[team.Country]
	team.PlayerIDs = selectNationalSquad(pool)
	team.Rating = nationalSquadRating(team.PlayerIDs, pool)

	tm.PushInbox(
		MsgCategoryCup,
		fmt.Sprintf("Nations Cup squad announced: %s", team.Name),
		fmt.Sprintf("%s revert to the staff's automatic selection for the Nations Cup.", team.Name),
		tm.CurrentMatchweek,
		[]string{}, "", "",
	)
	return "", tm.compactNationalTeamUnlocked(team, true)
}

// NationalSquadSelection returns one nation's current squad, the eligible
// player pool, and whether the squad is viewer-selected.
func (tm *TournamentManager) NationalSquadSelection(teamID string) map[string]interface{} {
	if tm == nil {
		return nil
	}
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	if tm.World == nil || tm.World.NationalTeams == nil {
		return nil
	}
	competition := tm.World.NationalTeams
	team := competition.Teams[teamID]
	if team == nil {
		return nil
	}
	pool := append([]nationalPlayerCandidate(nil), tm.nationalCandidatesByCountryUnlocked()[team.Country]...)
	sortNationalCandidates(pool)
	eligible := make([]map[string]interface{}, 0, len(pool))
	selected := make(map[string]bool, len(team.PlayerIDs))
	for _, id := range team.PlayerIDs {
		selected[id] = true
	}
	for _, row := range pool {
		if row.player == nil {
			continue
		}
		clubName := ""
		if club := tm.Clubs[row.player.ClubID]; club != nil {
			clubName = club.ClubName
		}
		eligible = append(eligible, map[string]interface{}{
			"player_id": row.player.PlayerID, "full_name": row.player.FullName,
			"position": row.player.Position, "category": row.player.Category,
			"ovr": row.player.OVR, "age": row.player.Age,
			"club_id": row.player.ClubID, "club_name": clubName,
			"selected": selected[row.player.PlayerID],
		})
	}
	return map[string]interface{}{
		"team":            tm.compactNationalTeamUnlocked(team, true),
		"eligible_pool":   eligible,
		"viewer_selected": len(competition.ViewerSquads[teamID]) > 0,
		"squad_size":      NationalSquadSize,
	}
}
