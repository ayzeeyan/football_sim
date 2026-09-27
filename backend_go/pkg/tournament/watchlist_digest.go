package tournament

import (
	"fmt"
	"strings"
)

// generateWatchlistDigestUnlocked pushes one inbox item per matchweek
// summarizing watched-entity activity from the week's finished fixtures:
// results for watched clubs and goals for watched players. It is a pure
// read of already-resolved state — no simulation, no randomness — so the
// digest is identical across replays of the same week.
func (tm *TournamentManager) generateWatchlistDigestUnlocked(completedMW int) {
	if len(tm.Watch.Clubs) == 0 && len(tm.Watch.Players) == 0 {
		return
	}
	watchedClubs := make(map[string]bool, len(tm.Watch.Clubs))
	for _, id := range tm.Watch.Clubs {
		watchedClubs[id] = true
	}
	watchedPlayers := make(map[string]bool, len(tm.Watch.Players))
	for _, id := range tm.Watch.Players {
		watchedPlayers[id] = true
	}

	var lines []string
	scan := func(fixtures []Fixture) {
		for i := range fixtures {
			f := &fixtures[i]
			if f.Status != "finished" || f.Matchweek != completedMW || f.HomeGoals == nil || f.AwayGoals == nil {
				continue
			}
			if watchedClubs[f.HomeID] || watchedClubs[f.AwayID] {
				home := tm.Clubs[f.HomeID]
				away := tm.Clubs[f.AwayID]
				homeName, awayName := f.HomeID, f.AwayID
				if home != nil {
					homeName = home.ShortName
				}
				if away != nil {
					awayName = away.ShortName
				}
				lines = append(lines, fmt.Sprintf("%s %d–%d %s (%s).", homeName, *f.HomeGoals, *f.AwayGoals, awayName, prettyCompetitionID(f.Competition)))
			}
			if f.Report == nil || len(watchedPlayers) == 0 {
				continue
			}
			for _, e := range f.Report.Events {
				if e.Disallowed {
					continue
				}
				switch e.Type {
				case "goal", "penalty", "corner_goal", "free_kick_goal":
					if e.Scorer != nil && watchedPlayers[e.Scorer.PlayerID] {
						lines = append(lines, fmt.Sprintf("%s scored for %s in the %d' (%s).", e.Scorer.FullName, sideClubShort(tm, f, e.Side), e.Minute, prettyCompetitionID(f.Competition)))
					}
				case "own_goal":
					if e.Scorer != nil && watchedPlayers[e.Scorer.PlayerID] {
						lines = append(lines, fmt.Sprintf("%s put through his own net in the %d' (%s).", e.Scorer.FullName, e.Minute, prettyCompetitionID(f.Competition)))
					}
				}
			}
		}
	}
	scan(tm.Fixtures)
	scan(tm.UCLFixtures)
	scan(tm.SuperCupFixtures)
	if tm.World != nil {
		scan(tm.World.Fixtures)
	}
	if len(lines) == 0 {
		return
	}
	// Cap the digest so a busy week cannot flood the inbox.
	if len(lines) > 8 {
		lines = lines[:8]
	}
	clubIDs := append([]string{}, tm.Watch.Clubs...)
	tm.PushInbox("watch",
		fmt.Sprintf("Watchlist: %d update%s from matchweek %d", len(lines), plural(len(lines)), completedMW),
		strings.Join(lines, " "), completedMW, clubIDs, "", "")
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func sideClubShort(tm *TournamentManager, f *Fixture, side string) string {
	id := f.AwayID
	if side == "home" {
		id = f.HomeID
	}
	if club := tm.Clubs[id]; club != nil {
		return club.ShortName
	}
	return id
}

func prettyCompetitionID(id string) string {
	if id == "" {
		return "league"
	}
	return strings.ReplaceAll(id, "-", " ")
}
