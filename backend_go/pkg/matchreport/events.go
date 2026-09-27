package matchreport

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	extraClockRe  = regexp.MustCompile(`(\d+\+\d+)'`)
	simpleClockRe = regexp.MustCompile(`(\d+)'`)
)

// ClockOnlyDisplay stores a minute stamp and never a presentation sentence.
func ClockOnlyDisplay(minute int, display string) string {
	if m := extraClockRe.FindString(display); m != "" {
		return m
	}
	if m := simpleClockRe.FindString(display); m != "" {
		return m
	}
	if minute > 0 {
		return fmt.Sprintf("%d'", minute)
	}
	return strings.TrimSpace(display)
}

func nestedID(p *MiniPlayer) string {
	if p == nil {
		return ""
	}
	return p.PlayerID
}

func nestedName(p *MiniPlayer) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(p.FullName)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// NormalizeEventFacts flattens player/assist identity and rewrites Display to
// a clock stamp so consumers never have to parse "Name 43' (Assist: ...)".
func NormalizeEventFacts(events []MatchEventItem) {
	for i := range events {
		e := &events[i]
		e.PlayerID = firstNonEmpty(e.PlayerID, nestedID(e.Scorer), nestedID(e.Player), nestedID(e.PlayerIn))
		e.PlayerName = firstNonEmpty(e.PlayerName, nestedName(e.Scorer), nestedName(e.Player), nestedName(e.PlayerIn))
		if e.Assister != nil {
			e.AssistPlayerID = firstNonEmpty(e.AssistPlayerID, e.Assister.PlayerID)
			e.AssistPlayerName = firstNonEmpty(e.AssistPlayerName, e.Assister.FullName)
		}
		e.Display = ClockOnlyDisplay(e.Minute, e.Display)
	}
}

// AssignEventClubs fills missing club identity from the fixture sides.
func AssignEventClubs(events []MatchEventItem, homeID, homeName, awayID, awayName string) {
	for i := range events {
		e := &events[i]
		if e.ClubID != "" && e.ClubName != "" {
			continue
		}
		side := e.Side
		if side == "away" {
			if e.ClubID == "" {
				e.ClubID = awayID
			}
			if e.ClubName == "" {
				e.ClubName = awayName
			}
			continue
		}
		if e.ClubID == "" {
			e.ClubID = homeID
		}
		if e.ClubName == "" {
			e.ClubName = homeName
		}
	}
}
