package models

import (
	"fmt"
	"sort"
)

// Player unrest: quality players starved of playing time want to leave.
//
// The model is derived, not persisted: unrest is a pure function of the
// current-season facts (squad rank by rating, appearances share, age, and
// rating level). It lives in models so both the tournament weekly tick and
// the transfer market can read it without import cycles.

const (
	// unrestMinRating is the floor below which a player has no case:
	// squad players at modest ratings accept rotation.
	unrestMinRating = 70
	// unrestSquadRankCutoff: a player only has a case if they rank among
	// the club's best fourteen players by rating.
	unrestSquadRankCutoff = 14
	// unrestAppearanceShare is the share of the club's played matches
	// below which a top-rated player counts as starved.
	unrestAppearanceShare = 0.35
	// unrestMinPlayed: before this many club matches nobody has a case.
	unrestMinPlayed = 6
)

// Unrest levels, most severe last.
const (
	UnrestContent      = "content"
	UnrestUnsettled    = "unsettled"
	UnrestWantsToLeave = "wants_to_leave"
)

// UnrestAssessment is one player's derived career-state at their club.
type UnrestAssessment struct {
	Level           UnrestLevel
	Reason          string
	SquadRank       int
	AppearanceShare float64
}

// UnrestLevel is the wire type for the assessment level.
type UnrestLevel string

// SquadRankByRating returns the player's 1-based rank in the club squad by
// rating (ties broken by player ID for determinism).
func SquadRankByRating(club *Club) map[string]int {
	ranked := make([]*Player, 0, len(club.Squad))
	for _, p := range club.Squad {
		if p != nil {
			ranked = append(ranked, p)
		}
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].OVR != ranked[j].OVR {
			return ranked[i].OVR > ranked[j].OVR
		}
		return ranked[i].PlayerID < ranked[j].PlayerID
	})
	rank := make(map[string]int, len(ranked))
	for i, p := range ranked {
		rank[p.PlayerID] = i + 1
	}
	return rank
}

// AssessUnrest derives one player's unrest from the current-season facts.
// Pure and deterministic; nil assessment means the player is content.
func AssessUnrest(club *Club, p *Player, squadRank int) *UnrestAssessment {
	if club == nil || p == nil || club.Played < unrestMinPlayed {
		return nil
	}
	if p.OVR < unrestMinRating || p.Age < 23 || squadRank > unrestSquadRankCutoff {
		return nil
	}
	share := float64(p.Appearances) / float64(club.Played)
	if share >= unrestAppearanceShare {
		return nil
	}
	if p.Age >= 27 && p.OVR >= 75 && share < unrestAppearanceShare/2 {
		return &UnrestAssessment{
			Level: UnrestWantsToLeave,
			Reason: fmt.Sprintf("%s is at peak age and ranks #%d by rating at %s, yet has featured in only %d of %d matches. The player's camp wants a move.",
				p.FullName, squadRank, club.ClubName, p.Appearances, club.Played),
			SquadRank:       squadRank,
			AppearanceShare: share,
		}
	}
	return &UnrestAssessment{
		Level: UnrestUnsettled,
		Reason: fmt.Sprintf("%s ranks #%d by rating at %s but has featured in only %d of %d matches.",
			p.FullName, squadRank, club.ClubName, p.Appearances, club.Played),
		SquadRank:       squadRank,
		AppearanceShare: share,
	}
}
