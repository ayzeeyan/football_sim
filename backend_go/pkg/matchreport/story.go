package matchreport

// StoryFact is a compact, match-derived beat the UI can turn into prose.
// Kinds are factual labels only — never causal claims.
type StoryFact struct {
	Kind       string `json:"kind"`
	Minute     int    `json:"minute,omitempty"`
	PlayerID   string `json:"player_id,omitempty"`
	PlayerName string `json:"player_name,omitempty"`
	Side       string `json:"side,omitempty"`
}

// TableDelta is one club's league place before and after a finished match.
type TableDelta struct {
	ClubID    string `json:"club_id"`
	ShortName string `json:"short_name"`
	BeforePos int    `json:"before_pos"`
	AfterPos  int    `json:"after_pos"`
	BeforePts int    `json:"before_pts"`
	AfterPts  int    `json:"after_pts"`
	BeforeGD  int    `json:"before_gd"`
	AfterGD   int    `json:"after_gd"`
}

// NearbyStanding is a compact row for the post-match table snapshot.
type NearbyStanding struct {
	Pos       int    `json:"pos"`
	ClubID    string `json:"club_id"`
	ShortName string `json:"short_name"`
	Pts       int    `json:"pts"`
	GD        int    `json:"gd"`
}

// TableImpact records league movement caused by this fixture.
type TableImpact struct {
	Applicable bool             `json:"applicable"`
	Home       TableDelta       `json:"home"`
	Away       TableDelta       `json:"away"`
	Nearby     []NearbyStanding `json:"nearby,omitempty"`
}

func eventPlayerID(e MatchEventItem) string {
	if e.PlayerID != "" {
		return e.PlayerID
	}
	if e.Scorer != nil && e.Scorer.PlayerID != "" {
		return e.Scorer.PlayerID
	}
	if e.Player != nil && e.Player.PlayerID != "" {
		return e.Player.PlayerID
	}
	return ""
}

func eventPlayerName(e MatchEventItem) string {
	if e.PlayerName != "" {
		return e.PlayerName
	}
	if e.Scorer != nil && e.Scorer.FullName != "" {
		return e.Scorer.FullName
	}
	if e.Player != nil && e.Player.FullName != "" {
		return e.Player.FullName
	}
	if e.PlayerIn != nil && e.PlayerIn.FullName != "" {
		return e.PlayerIn.FullName
	}
	return ""
}

func isScoringEvent(e MatchEventItem) bool {
	if e.Disallowed {
		return false
	}
	switch e.Type {
	case "goal", "penalty", "own_goal", "corner_goal", "free_kick_goal":
		return true
	default:
		return false
	}
}

func scoringSide(e MatchEventItem) string {
	if e.Type == "own_goal" {
		if e.Beneficiary == "home" || e.Beneficiary == "away" {
			return e.Beneficiary
		}
		if e.Side == "home" {
			return "away"
		}
		if e.Side == "away" {
			return "home"
		}
	}
	return e.Side
}

func factFromEvent(kind string, e MatchEventItem, side string) StoryFact {
	return StoryFact{
		Kind:       kind,
		Minute:     e.Minute,
		PlayerID:   eventPlayerID(e),
		PlayerName: eventPlayerName(e),
		Side:       side,
	}
}

// DetectStoryFacts walks a finished timeline and records factual beats.
func DetectStoryFacts(events []MatchEventItem, homeGoals, awayGoals int, decidedBy *string) []StoryFact {
	facts := make([]StoryFact, 0, 8)
	home, away := 0, 0
	var opening *MatchEventItem
	var openingSide string
	var lastLeadChange *MatchEventItem
	var lastLeadSide string
	homeMaxTrail, awayMaxTrail := 0, 0
	goalCounts := map[string]struct {
		n    int
		name string
		side string
	}{}

	for i := range events {
		e := events[i]
		if e.Type == "red" || e.Type == "red_card" {
			facts = append(facts, factFromEvent("red_card", e, e.Side))
		}
		if e.Type == "penalty" {
			facts = append(facts, factFromEvent("penalty", e, scoringSide(e)))
		}
		if !isScoringEvent(e) {
			continue
		}
		side := scoringSide(e)
		prevHome, prevAway := home, away
		if side == "home" {
			home++
		} else if side == "away" {
			away++
		}
		if opening == nil {
			cpy := e
			opening = &cpy
			openingSide = side
			facts = append(facts, factFromEvent("opening_goal", e, side))
		}
		if (home == away) && (prevHome != prevAway) {
			facts = append(facts, factFromEvent("equalizer", e, side))
		}
		if home > away {
			if away-home < awayMaxTrail {
				awayMaxTrail = away - home
			}
			if prevHome <= prevAway {
				cpy := e
				lastLeadChange = &cpy
				lastLeadSide = "home"
			}
		} else if away > home {
			if home-away < homeMaxTrail {
				homeMaxTrail = home - away
			}
			if prevAway <= prevHome {
				cpy := e
				lastLeadChange = &cpy
				lastLeadSide = "away"
			}
		}
		if pid := eventPlayerID(e); pid != "" && e.Type != "own_goal" {
			row := goalCounts[pid]
			row.n++
			row.name = eventPlayerName(e)
			row.side = side
			goalCounts[pid] = row
		}
	}

	if lastLeadChange != nil && home != away {
		kind := "winner"
		if lastLeadChange.Minute >= 80 {
			kind = "late_winner"
		}
		facts = append(facts, factFromEvent(kind, *lastLeadChange, lastLeadSide))
	}

	if homeGoals > awayGoals && homeMaxTrail <= -2 {
		facts = append(facts, StoryFact{Kind: "two_goal_comeback", Side: "home"})
	} else if awayGoals > homeGoals && awayMaxTrail <= -2 {
		facts = append(facts, StoryFact{Kind: "two_goal_comeback", Side: "away"})
	} else if homeGoals > awayGoals && homeMaxTrail <= -1 {
		facts = append(facts, StoryFact{Kind: "comeback", Side: "home"})
	} else if awayGoals > homeGoals && awayMaxTrail <= -1 {
		facts = append(facts, StoryFact{Kind: "comeback", Side: "away"})
	}

	for _, row := range goalCounts {
		if row.n >= 3 {
			facts = append(facts, StoryFact{Kind: "hat_trick", PlayerName: row.name, Side: row.side})
		}
	}

	if homeGoals-awayGoals >= 3 {
		facts = append(facts, StoryFact{Kind: "multi_goal_lead", Side: "home"})
	} else if awayGoals-homeGoals >= 3 {
		facts = append(facts, StoryFact{Kind: "multi_goal_lead", Side: "away"})
	}

	if awayGoals == 0 && homeGoals > 0 {
		facts = append(facts, StoryFact{Kind: "clean_sheet", Side: "home"})
	}
	if homeGoals == 0 && awayGoals > 0 {
		facts = append(facts, StoryFact{Kind: "clean_sheet", Side: "away"})
	}

	if decidedBy != nil {
		switch *decidedBy {
		case "extra_time":
			facts = append(facts, StoryFact{Kind: "extra_time"})
		case "penalties":
			facts = append(facts, StoryFact{Kind: "penalty_shootout"})
		}
	}

	_ = openingSide
	return facts
}
