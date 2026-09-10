package tournament

import (
	"fmt"
	"time"
)

// InboxItem represents a journalistic or narrative headline in the user's news wire.
type InboxItem struct {
	ID         string    `json:"id"`
	Timestamp  string    `json:"timestamp"`
	Matchweek  int       `json:"matchweek"`
	SeasonName string    `json:"season_name"`
	Category   string    `json:"category"` // match, transfer, wonderkid, honour, race, cup, system, injury, dugout, youth, nxgn
	Headline   string    `json:"headline"`
	Body       string    `json:"body"`
	ClubIDs    []string  `json:"club_ids"`
	PlayerID   string    `json:"player_id,omitempty"`
	FixtureID  string    `json:"fixture_id,omitempty"`
	Unread     bool      `json:"unread"`
	CreatedAt  time.Time `json:"created_at"`
}

// InboxFeed wraps the list of inbox items and unread count.
type InboxFeed struct {
	Unread           int         `json:"unread"`
	Items            []InboxItem `json:"items"`
	SeasonName       string      `json:"season_name"`
	CurrentMatchweek int         `json:"current_matchweek"`
}

// NewInboxItem creates an initialized InboxItem.
func NewInboxItem(
	id string,
	category string,
	headline string,
	body string,
	matchweek int,
	seasonName string,
	clubIDs []string,
	playerID string,
	fixtureID string,
) InboxItem {
	return InboxItem{
		ID:         id,
		Timestamp:  fmt.Sprintf("MW %d", matchweek),
		Matchweek:  matchweek,
		SeasonName: seasonName,
		Category:   category,
		Headline:   headline,
		Body:       body,
		ClubIDs:    clubIDs,
		PlayerID:   playerID,
		FixtureID:  fixtureID,
		Unread:     true,
		CreatedAt:  time.Now(),
	}
}
