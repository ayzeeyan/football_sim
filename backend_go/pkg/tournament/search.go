package tournament

import (
	"sort"
	"strings"

	"football_sim/pkg/models"
)

const defaultSearchLimit = 8
const directoryPlayerLimit = 40

// SearchWorld finds clubs, players and competitions for the global inspector.
// An empty query returns a compact directory (top-rated players, strongest
// clubs, every competition) so the Players page has a real starting list.
func (tm *TournamentManager) SearchWorld(query string, limit int) map[string]interface{} {
	if tm == nil {
		return map[string]interface{}{
			"query": query, "clubs": []map[string]interface{}{},
			"players": []map[string]interface{}{}, "competitions": []map[string]interface{}{},
		}
	}
	q := strings.ToLower(strings.TrimSpace(query))
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	if q == "" {
		return tm.worldDirectoryUnlocked(limit)
	}

	type clubHit struct {
		club  *models.Club
		score int
	}
	type playerHit struct {
		player *models.Player
		club   *models.Club
		score  int
	}
	var clubs []clubHit
	var players []playerHit
	for _, club := range tm.ClubsList {
		if club == nil {
			continue
		}
		if score := searchScore(q, club.ClubName, club.ShortName, club.ClubID, club.League); score > 0 {
			clubs = append(clubs, clubHit{club: club, score: score})
		}
		for _, p := range club.Squad {
			if p == nil {
				continue
			}
			if score := searchScore(q, p.FullName, p.PlayerID, p.Position); score > 0 {
				players = append(players, playerHit{player: p, club: club, score: score})
			}
		}
	}
	sort.SliceStable(clubs, func(i, j int) bool {
		if clubs[i].score != clubs[j].score {
			return clubs[i].score > clubs[j].score
		}
		if clubs[i].club.OverallTeamRating != clubs[j].club.OverallTeamRating {
			return clubs[i].club.OverallTeamRating > clubs[j].club.OverallTeamRating
		}
		return clubs[i].club.ClubID < clubs[j].club.ClubID
	})
	sort.SliceStable(players, func(i, j int) bool {
		if players[i].score != players[j].score {
			return players[i].score > players[j].score
		}
		if players[i].player.OVR != players[j].player.OVR {
			return players[i].player.OVR > players[j].player.OVR
		}
		if players[i].player.FullName != players[j].player.FullName {
			return players[i].player.FullName < players[j].player.FullName
		}
		return players[i].player.PlayerID < players[j].player.PlayerID
	})
	if len(clubs) > limit {
		clubs = clubs[:limit]
	}
	if len(players) > limit {
		players = players[:limit]
	}

	clubOut := make([]map[string]interface{}, 0, len(clubs))
	for _, hit := range clubs {
		clubOut = append(clubOut, compactSearchClub(hit.club))
	}
	playerOut := make([]map[string]interface{}, 0, len(players))
	for _, hit := range players {
		playerOut = append(playerOut, compactSearchPlayer(hit.player, hit.club))
	}
	return map[string]interface{}{
		"query":        query,
		"clubs":        clubOut,
		"players":      playerOut,
		"competitions": tm.searchCompetitionsUnlocked(q, limit),
	}
}

func (tm *TournamentManager) worldDirectoryUnlocked(limit int) map[string]interface{} {
	playerLimit := directoryPlayerLimit
	if limit > defaultSearchLimit {
		playerLimit = limit
	}
	type ratedClub struct {
		club *models.Club
	}
	type ratedPlayer struct {
		player *models.Player
		club   *models.Club
	}
	clubs := make([]ratedClub, 0, len(tm.ClubsList))
	players := make([]ratedPlayer, 0, 256)
	for _, club := range tm.ClubsList {
		if club == nil {
			continue
		}
		clubs = append(clubs, ratedClub{club: club})
		for _, p := range club.Squad {
			if p == nil {
				continue
			}
			players = append(players, ratedPlayer{player: p, club: club})
		}
	}
	sort.SliceStable(clubs, func(i, j int) bool {
		if clubs[i].club.OverallTeamRating != clubs[j].club.OverallTeamRating {
			return clubs[i].club.OverallTeamRating > clubs[j].club.OverallTeamRating
		}
		if clubs[i].club.Points != clubs[j].club.Points {
			return clubs[i].club.Points > clubs[j].club.Points
		}
		return clubs[i].club.ClubID < clubs[j].club.ClubID
	})
	sort.SliceStable(players, func(i, j int) bool {
		if players[i].player.OVR != players[j].player.OVR {
			return players[i].player.OVR > players[j].player.OVR
		}
		if players[i].player.FullName != players[j].player.FullName {
			return players[i].player.FullName < players[j].player.FullName
		}
		return players[i].player.PlayerID < players[j].player.PlayerID
	})
	if len(clubs) > 12 {
		clubs = clubs[:12]
	}
	if len(players) > playerLimit {
		players = players[:playerLimit]
	}
	clubOut := make([]map[string]interface{}, 0, len(clubs))
	for _, hit := range clubs {
		clubOut = append(clubOut, compactSearchClub(hit.club))
	}
	playerOut := make([]map[string]interface{}, 0, len(players))
	for _, hit := range players {
		playerOut = append(playerOut, compactSearchPlayer(hit.player, hit.club))
	}
	return map[string]interface{}{
		"query":        "",
		"clubs":        clubOut,
		"players":      playerOut,
		"competitions": tm.searchCompetitionsUnlocked("", 32),
	}
}

func (tm *TournamentManager) searchCompetitionsUnlocked(q string, limit int) []map[string]interface{} {
	out := []map[string]interface{}{}
	if tm.World == nil {
		return out
	}
	ids := append([]string(nil), tm.World.CompetitionOrder...)
	sort.Strings(ids)
	for _, id := range ids {
		comp := tm.World.Competitions[id]
		if comp == nil {
			continue
		}
		if q != "" && searchScore(q, comp.Name, comp.ID, comp.Country, string(comp.Kind)) == 0 {
			continue
		}
		out = append(out, map[string]interface{}{
			"id":           comp.ID,
			"name":         comp.Name,
			"kind":         comp.Kind,
			"country":      comp.Country,
			"stage":        comp.Stage,
			"participants": len(comp.ParticipantIDs),
		})
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func compactSearchClub(club *models.Club) map[string]interface{} {
	if club == nil {
		return nil
	}
	return map[string]interface{}{
		"club_id": club.ClubID, "club_name": club.ClubName, "short_name": club.ShortName,
		"league": club.League, "ovr": club.OverallTeamRating, "pts": club.Points,
		"primary_color": club.PrimaryColor,
	}
}

func compactSearchPlayer(p *models.Player, club *models.Club) map[string]interface{} {
	if p == nil {
		return nil
	}
	entry := map[string]interface{}{
		"player_id": p.PlayerID, "full_name": p.FullName, "position": p.Position,
		"ovr": p.OVR, "age": p.Age, "category": p.Category, "club_id": p.ClubID,
		"goals": p.Goals, "assists": p.Assists, "is_wonderkid": p.UniverseWonderkid,
	}
	if club != nil {
		entry["club_name"] = club.ClubName
		entry["club_short"] = club.ShortName
		entry["primary_color"] = club.PrimaryColor
	}
	return entry
}

func searchScore(q string, parts ...string) int {
	if q == "" {
		return 0
	}
	best := 0
	for _, part := range parts {
		s := strings.ToLower(strings.TrimSpace(part))
		if s == "" {
			continue
		}
		if s == q {
			if best < 100 {
				best = 100
			}
			continue
		}
		if strings.HasPrefix(s, q) {
			if best < 80 {
				best = 80
			}
			continue
		}
		for _, word := range strings.Fields(s) {
			if strings.HasPrefix(word, q) && best < 70 {
				best = 70
			}
		}
		if strings.Contains(s, q) && best < 40 {
			best = 40
		}
	}
	return best
}
