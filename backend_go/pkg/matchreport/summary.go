package matchreport

import (
	"strconv"
)

// ReportSummary is the long-term archival form of a finished match. Full
// MatchReport payloads (events, player rows, stats) are kept for a short
// recent window; beyond that the report is replaced by this summary so a
// career save does not grow without bound. Summaries retain the identifiers
// required by the domain invariants: every scorer and dismissed player keeps
// both player and club IDs via the flat event attribution fields.
type ReportSummary struct {
	HomeGoals      int             `json:"home_goals"`
	AwayGoals      int             `json:"away_goals"`
	HTHome         int             `json:"ht_home"`
	HTAway         int             `json:"ht_away"`
	HomeFormation  string          `json:"home_formation,omitempty"`
	AwayFormation  string          `json:"away_formation,omitempty"`
	Scorers        []SummaryScorer `json:"scorers,omitempty"`
	RedCards       []SummaryCard   `json:"red_cards,omitempty"`
	MOTM           *SummaryPlayer  `json:"motm,omitempty"`
	PossessionHome int             `json:"possession_home"`
	PossessionAway int             `json:"possession_away"`
	ShotsHome      int             `json:"shots_home"`
	ShotsAway      int             `json:"shots_away"`
	XGHome         float64         `json:"xg_home"`
	XGAway         float64         `json:"xg_away"`
	Attendance     int             `json:"attendance,omitempty"`
	Referee        string          `json:"referee,omitempty"`
	Weather        string          `json:"weather,omitempty"`
	DecidedBy      string          `json:"decided_by,omitempty"`
}

// SummaryScorer keeps goal attribution with immutable identifiers.
type SummaryScorer struct {
	Minute     int    `json:"minute"`
	PlayerID   string `json:"player_id"`
	PlayerName string `json:"player_name"`
	ClubID     string `json:"club_id"`
	Type      string `json:"type"`
}

// SummaryCard keeps dismissal attribution with immutable identifiers.
type SummaryCard struct {
	Minute     int    `json:"minute"`
	PlayerID   string `json:"player_id"`
	PlayerName string `json:"player_name"`
	ClubID     string `json:"club_id"`
}

// SummaryPlayer is the compact MOTM reference.
type SummaryPlayer struct {
	PlayerID string `json:"player_id"`
	FullName string `json:"full_name"`
	Rating   string `json:"rating,omitempty"`
}

// SummarizeReport distills a finished match report into its archival summary.
// It is pure: no randomness, no clock, no mutation of the source report.
func SummarizeReport(r *MatchReport) *ReportSummary {
	if r == nil {
		return nil
	}
	s := &ReportSummary{
		HomeGoals:      r.HomeGoals,
		AwayGoals:      r.AwayGoals,
		HTHome:         r.HTHome,
		HTAway:         r.HTAway,
		HomeFormation:  r.HomeFormation,
		AwayFormation:  r.AwayFormation,
		PossessionHome: r.Stats.Home.Possession,
		PossessionAway: r.Stats.Away.Possession,
		ShotsHome:      r.Stats.Home.Shots,
		ShotsAway:      r.Stats.Away.Shots,
		XGHome:         r.Stats.Home.XG,
		XGAway:         r.Stats.Away.XG,
		Attendance:     r.Attendance,
		Referee:        r.Referee,
		Weather:        r.Weather,
	}
	if r.DecidedBy != nil {
		s.DecidedBy = *r.DecidedBy
	}
	if r.MOTM != nil {
		motm := &SummaryPlayer{PlayerID: r.MOTM.PlayerID, FullName: r.MOTM.FullName}
		if r.MOTM.Rating != nil {
			motm.Rating = strconv.FormatFloat(*r.MOTM.Rating, 'f', 1, 64)
		}
		s.MOTM = motm
	}
	for _, e := range r.Events {
		if e.Disallowed {
			continue
		}
		playerID, playerName, clubID := e.PlayerID, e.PlayerName, e.ClubID
		if playerID == "" && e.Scorer != nil {
			playerID, playerName = e.Scorer.PlayerID, e.Scorer.FullName
		}
		if playerID == "" {
			continue
		}
		switch e.Type {
		case "goal", "penalty", "corner_goal", "free_kick_goal", "own_goal":
			s.Scorers = append(s.Scorers, SummaryScorer{
				Minute:     e.Minute,
				PlayerID:   playerID,
				PlayerName: playerName,
				ClubID:     clubID,
				Type:       e.Type,
			})
		case "red":
			s.RedCards = append(s.RedCards, SummaryCard{
				Minute:     e.Minute,
				PlayerID:   playerID,
				PlayerName: playerName,
				ClubID:     clubID,
			})
		}
	}
	return s
}
