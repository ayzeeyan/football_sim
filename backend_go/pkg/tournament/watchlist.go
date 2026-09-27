package tournament

import (
	"sort"

	"football_sim/pkg/models"
)

// WatchlistEntity is one watchable kind. The watchlist is an observational
// tool only: it filters what the viewer is notified about and never grants
// control of any AI entity.
type WatchlistEntity string

const (
	WatchClub        WatchlistEntity = "club"
	WatchPlayer      WatchlistEntity = "player"
	WatchCompetition WatchlistEntity = "competition"
)

// WatchlistState is the persisted multi-entity watchlist. IDs are stored
// sorted and deduplicated so saves are byte-stable across runs.
type WatchlistState struct {
	Clubs        []string `json:"clubs,omitempty"`
	Players      []string `json:"players,omitempty"`
	Competitions []string `json:"competitions,omitempty"`
}

func (tm *TournamentManager) watchlistValid(entity WatchlistEntity, id string) bool {
	switch entity {
	case WatchClub:
		_, ok := tm.Clubs[id]
		return ok
	case WatchPlayer:
		return tm.findPlayerByIDUnlocked(id) != nil
	case WatchCompetition:
		if tm.World == nil {
			return false
		}
		_, ok := tm.World.Competitions[id]
		return ok
	default:
		return false
	}
}

func (tm *TournamentManager) findPlayerByIDUnlocked(playerID string) *models.Player {
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
	return nil
}

func watchlistCopy(in WatchlistState) WatchlistState {
	return WatchlistState{
		Clubs:        append([]string(nil), in.Clubs...),
		Players:      append([]string(nil), in.Players...),
		Competitions: append([]string(nil), in.Competitions...),
	}
}

// Watchlist returns a copy of the current watchlist.
func (tm *TournamentManager) Watchlist() WatchlistState {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	out := WatchlistState{
		Clubs:        append([]string(nil), tm.Watch.Clubs...),
		Players:      append([]string(nil), tm.Watch.Players...),
		Competitions: append([]string(nil), tm.Watch.Competitions...),
	}
	return out
}

// ToggleWatchlist adds or removes one entity. Returns the new state and
// whether the entity is now watched. Unknown entities are rejected.
func (tm *TournamentManager) ToggleWatchlist(entity WatchlistEntity, id string) (WatchlistState, bool, bool) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if !tm.watchlistValid(entity, id) {
		return watchlistCopy(tm.Watch), false, false
	}
	var list *[]string
	switch entity {
	case WatchClub:
		list = &tm.Watch.Clubs
	case WatchPlayer:
		list = &tm.Watch.Players
	case WatchCompetition:
		list = &tm.Watch.Competitions
	default:
		return tm.Watch, false, false
	}
	idx := sort.SearchStrings(*list, id)
	watched := idx < len(*list) && (*list)[idx] == id
	if watched {
		*list = append((*list)[:idx], (*list)[idx+1:]...)
	} else {
		*list = append(*list, id)
		sort.Strings(*list)
	}
	return watchlistCopy(tm.Watch), !watched, true
}

// IsWatched reports whether an entity id is on the watchlist.
func (tm *TournamentManager) IsWatched(entity WatchlistEntity, id string) bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	var list []string
	switch entity {
	case WatchClub:
		list = tm.Watch.Clubs
	case WatchPlayer:
		list = tm.Watch.Players
	case WatchCompetition:
		list = tm.Watch.Competitions
	default:
		return false
	}
	idx := sort.SearchStrings(list, id)
	return idx < len(list) && list[idx] == id
}
