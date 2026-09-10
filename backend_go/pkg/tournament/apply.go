package tournament

import (
	"fmt"
	"math/rand"
	"strings"

	"football_sim/pkg/growth"
	"football_sim/pkg/matchreport"
	"football_sim/pkg/models"
	"football_sim/pkg/transfers"
)

// Post-match application ported from tournament.py: player ledgers, market
// movements, prodigy growth attribution, and injury rolls. Every simulated
// fixture must pass through ApplyMatchReport so season totals, bans,
// fatigue, valuations, XP, and treatment-room state stay faithful.

// ApplyPlayerMatchStats banks goals, assists, own goals, appearances, and
// red-card bans from a finished report, then tracks fatigue and decays
// served suspensions and injuries for unused squad players.
func ApplyPlayerMatchStats(homeClub, awayClub *models.Club, report *matchreport.MatchReport) {
	if report == nil {
		return
	}
	find := func(pid string) *models.Player {
		for _, club := range []*models.Club{homeClub, awayClub} {
			if club == nil {
				continue
			}
			for _, p := range club.Squad {
				if p.PlayerID == pid {
					return p
				}
			}
		}
		return nil
	}

	for _, e := range report.Events {
		if e.Disallowed {
			continue
		}
		switch e.Type {
		case "goal", "penalty", "corner_goal", "free_kick_goal":
			if e.Scorer != nil {
				if scorer := find(e.Scorer.PlayerID); scorer != nil {
					scorer.RecordGoal()
				}
			}
			if e.Assister != nil {
				if a := find(e.Assister.PlayerID); a != nil {
					a.RecordAssist()
				}
			}
		case "own_goal":
			if e.Scorer != nil {
				if scorer := find(e.Scorer.PlayerID); scorer != nil {
					scorer.OwnGoals++
				}
			}
		case "red":
			if e.Player != nil {
				if booked := find(e.Player.PlayerID); booked != nil {
					booked.SuspendedMatches++
				}
			}
		}
	}

	starterIDs := map[string]bool{}
	for _, row := range append(append([]matchreport.MatchPlayerRow{}, report.HomeXI...), report.AwayXI...) {
		if row.Played {
			starterIDs[row.PlayerID] = true
		}
	}

	used := map[string]bool{}
	for _, club := range []*models.Club{homeClub, awayClub} {
		if club == nil {
			continue
		}
		var rows []matchreport.MatchPlayerRow
		if club == homeClub {
			rows = append(append([]matchreport.MatchPlayerRow{}, report.HomeXI...), report.HomeBench...)
		} else {
			rows = append(append([]matchreport.MatchPlayerRow{}, report.AwayXI...), report.AwayBench...)
		}
		squad := make(map[string]*models.Player, len(club.Squad))
		for _, p := range club.Squad {
			squad[p.PlayerID] = p
		}
		for _, row := range rows {
			if !row.Played || row.Minutes <= 0 {
				continue
			}
			player, ok := squad[row.PlayerID]
			if !ok {
				continue
			}
			player.RecordAppearance()
			used[player.PlayerID] = true
		}
	}

	for _, club := range []*models.Club{homeClub, awayClub} {
		if club == nil {
			continue
		}
		for _, p := range club.Squad {
			if starterIDs[p.PlayerID] {
				p.ConsecutiveStarts++
			} else {
				p.ConsecutiveStarts = 0
			}
			if used[p.PlayerID] {
				continue
			}
			if p.SuspendedMatches > 0 {
				p.SuspendedMatches--
			}
			if p.InjuredMatches > 0 {
				p.InjuredMatches--
				if p.InjuredMatches <= 0 {
					p.InjuredMatches = 0
					p.Injury = ""
				}
			}
		}
	}
}

// ApplyMarketMovements shifts valuations with match performance for both
// starting XIs (Python: _apply_market_movements). Big risers hit the
// transfer wire when a transfer engine is attached.
func (tm *TournamentManager) ApplyMarketMovements(homeClub, awayClub *models.Club, report *matchreport.MatchReport, matchweek int) {
	if report == nil {
		return
	}
	pairs := []struct {
		club *models.Club
		rows []matchreport.MatchPlayerRow
	}{
		{homeClub, report.HomeXI},
		{awayClub, report.AwayXI},
	}
	for _, pr := range pairs {
		if pr.club == nil {
			continue
		}
		squad := make(map[string]*models.Player, len(pr.club.Squad))
		for _, p := range pr.club.Squad {
			squad[p.PlayerID] = p
		}
		for _, row := range pr.rows {
			player, ok := squad[row.PlayerID]
			if !ok || player.MarketValueEUR <= 0 {
				continue
			}
			rating := 6.4
			if row.Rating != nil {
				rating = *row.Rating
			}
			card := ""
			if row.Card != nil {
				card = *row.Card
			}
			mult := 1.0 +
				0.06*float64(row.MatchGoals) +
				0.03*float64(row.MatchAssists)
			if rating >= 8.0 {
				mult += 0.05
			}
			if rating <= 6.0 {
				mult -= 0.05
			}
			if card == "red" {
				mult -= 0.05
			}
			if player.Age <= 21 && rating >= 7.0 {
				mult += 0.02
			}
			if mult < 0.92 {
				mult = 0.92
			}
			if mult > 1.15 {
				mult = 1.15
			}
			if mult == 1.0 {
				continue
			}
			oldValue := player.MarketValueEUR
			newValue := int64(float64(oldValue) * mult)
			if newValue < 500000 {
				newValue = 500000
			}
			player.MarketValueEUR = newValue
			models.ClampPlayer(player)
			if tm.TransferEngine != nil && player.MarketValueEUR >= int64(float64(oldValue)*1.12) {
				seller := pr.club
				if c, ok := tm.Clubs[player.ClubID]; ok && c != nil {
					seller = c
				}
				short := ""
				if seller != nil {
					short = seller.ShortName
				}
				tm.TransferEngine.TransferFeed = append([]transfers.TransferFeedItem{
					{
						Headline:    fmt.Sprintf("Market watch: %s's value is climbing after starring for %s.", player.FullName, short),
						Category:    "RUMOR",
						IsWonderkid: player.UniverseWonderkid,
						Matchweek:   matchweek,
						Timestamp:   fmt.Sprintf("MW %d", matchweek),
					},
				}, tm.TransferEngine.TransferFeed...)
			}
		}
	}
}

// AttributeProdigyPerformance applies real-event match XP to a club's
// franchise prodigy and syncs his OVR/composure (Python:
// _attribute_prodigy_performance). Returns growth event lines.
func AttributeProdigyPerformance(club *models.Club, report *matchreport.MatchReport, side string, ge *growth.GrowthEngine) []string {
	if club == nil || report == nil || ge == nil {
		return nil
	}
	var prodigy *models.Player
	for _, p := range club.Squad {
		if p.UniverseWonderkid {
			prodigy = p
			break
		}
	}
	if prodigy == nil {
		return nil
	}

	goals, assists := 0, 0
	for _, e := range report.Events {
		if (e.Type == "goal" || e.Type == "penalty" || e.Type == "corner_goal" || e.Type == "free_kick_goal") && e.Side == side {
			if e.Scorer != nil && e.Scorer.PlayerID == prodigy.PlayerID {
				goals++
			}
			if e.Assister != nil && e.Assister.PlayerID == prodigy.PlayerID {
				assists++
			}
		}
	}

	var rows []matchreport.MatchPlayerRow
	if side == "home" {
		rows = append(append([]matchreport.MatchPlayerRow{}, report.HomeXI...), report.HomeBench...)
	} else {
		rows = append(append([]matchreport.MatchPlayerRow{}, report.AwayXI...), report.AwayBench...)
	}
	baseRating := -1.0
	found := false
	for _, row := range rows {
		if row.PlayerID != prodigy.PlayerID {
			continue
		}
		found = true
		if !row.Played || row.Minutes <= 0 {
			return nil
		}
		if row.Rating == nil {
			return nil
		}
		baseRating = *row.Rating
		break
	}
	if !found {
		return nil
	}

	var opts growth.MatchXPOptions
	if prodigy.MentorOVR != 0 {
		opts.MentorOVR = &prodigy.MentorOVR
	}
	if prodigy.MentorName != "" {
		opts.MentorName = &prodigy.MentorName
	}
	if prodigy.Personality != "" {
		opts.Personality = &prodigy.Personality
	}
	events := ge.ApplyMatchXP(prodigy.PlayerID, prodigy.FullName, prodigy.Category, baseRating, goals, assists, opts)

	// Match XP shares the same annual ceiling as autonomous and season-end
	// development. Clamp the underlying technical matrix here so repeated cup
	// and league appearances cannot hide growth beyond the displayed rating.
	prodigy.OVR = ge.EnforceSeasonOVRCap(prodigy.PlayerID, prodigy.Category)
	if attrs, ok := ge.Attributes[prodigy.PlayerID]; ok && attrs != nil {
		prodigy.Composure = attrs.Composure
	}
	return events
}

var seriousInjuryKinds = []string{"ACL tear", "meniscus tear", "ruptured cruciate ligament"}
var minorInjuryKinds = []string{"knock", "hamstring strain", "ankle sprain", "thigh strain", "calf issue"}

func capitalizeKind(kind string) string {
	if kind == "" {
		return kind
	}
	return strings.ToUpper(kind[:1]) + kind[1:]
}

// MaybeInjure rolls in-match knocks and rare long-term injuries (Python:
// _maybe_injure). At most one casualty per club per fixture; keepers are
// protected when they are the last available.
func (tm *TournamentManager) MaybeInjure(homeClub, awayClub *models.Club, report *matchreport.MatchReport, matchweek int, fixtureID string) {
	if report == nil {
		return
	}
	rng := tm.RNG
	if rng == nil {
		rng = rand.New(rand.NewSource(1))
	}
	type sideClub struct {
		side string
		club *models.Club
		rows []matchreport.MatchPlayerRow
	}
	sides := []sideClub{
		{"home", homeClub, append(append([]matchreport.MatchPlayerRow{}, report.HomeXI...), report.HomeBench...)},
		{"away", awayClub, append(append([]matchreport.MatchPlayerRow{}, report.AwayXI...), report.AwayBench...)},
	}
	for _, sc := range sides {
		club := sc.club
		if club == nil {
			continue
		}
		already := 0
		for _, p := range club.Squad {
			if p.InjuredMatches > 0 {
				already++
			}
		}
		if already >= 3 {
			continue
		}
		var gks []*models.Player
		for _, p := range club.Squad {
			if p.Category == "GK" {
				gks = append(gks, p)
			}
		}
		rows := append([]matchreport.MatchPlayerRow(nil), sc.rows...)
		rng.Shuffle(len(rows), func(i, j int) { rows[i], rows[j] = rows[j], rows[i] })

		highPress := false
		if mgr, ok := tm.Managers[club.ClubID]; ok && mgr != nil &&
			(mgr.CanonicalStyle() == "high_press" || mgr.Style == "high_press") {
			highPress = true
		}
		baseChance := 0.05
		if highPress {
			baseChance = 0.08
		}

		for _, row := range rows {
			if !row.Played || row.Minutes < 30 {
				continue
			}
			var player *models.Player
			for _, p := range club.Squad {
				if p.PlayerID == row.PlayerID {
					player = p
					break
				}
			}
			if player == nil || player.SuspendedMatches > 0 || player.InjuredMatches > 0 {
				continue
			}
			if player.Category == "GK" {
				available := 0
				for _, g := range gks {
					if g.SuspendedMatches <= 0 && g.InjuredMatches <= 0 {
						available++
					}
				}
				if available <= 1 {
					continue
				}
			}
			chance := baseChance
			if player.UniverseWonderkid {
				chance *= 0.65
			}
			if row.Minutes >= 80 {
				chance *= 1.15
			}
			if rng.Float64() > chance {
				continue
			}

			if rng.Float64() < 0.025 {
				games := 15 + rng.Intn(11)
				kind := seriousInjuryKinds[rng.Intn(len(seriousInjuryKinds))]
				player.InjuredMatches = games
				player.Injury = kind
				tm.PushInbox(
					"injury",
					fmt.Sprintf("CRUSHING BLOW: %s suffers %s", player.FullName, kind),
					fmt.Sprintf("Devastating news for %s: medical scans confirm %s has suffered a severe %s and is ruled out for %d matches. A massive setback for the squad.", club.ClubName, player.FullName, kind, games),
					matchweek,
					[]string{club.ClubID}, player.PlayerID, fixtureID,
				)
			} else {
				games := 1 + rng.Intn(3)
				kind := minorInjuryKinds[rng.Intn(len(minorInjuryKinds))]
				player.InjuredMatches = games
				player.Injury = kind
				unit := "matches"
				if games == 1 {
					unit = "match"
				}
				tm.PushInbox(
					"injury",
					fmt.Sprintf("%s: %s out %d %s", capitalizeKind(kind), player.FullName, games, unit),
					fmt.Sprintf("%s will be without him. The XI changes.", club.ClubName),
					matchweek,
					[]string{club.ClubID}, player.PlayerID, fixtureID,
				)
			}
			break
		}
	}
}

// ApplyMatchReport lands a finished report on both clubs: ledgers, market,
// prodigy growth, and injuries. Returns growth event lines (newest first,
// capped the way the Python wire keeps the latest eight).
func (tm *TournamentManager) ApplyMatchReport(homeClub, awayClub *models.Club, report *matchreport.MatchReport, matchweek int, fixtureID string) []string {
	ApplyPlayerMatchStats(homeClub, awayClub, report)
	tm.ApplyMarketMovements(homeClub, awayClub, report, matchweek)
	var growthEvents []string
	growthEvents = append(growthEvents, AttributeProdigyPerformance(homeClub, report, "home", tm.GrowthEngine)...)
	growthEvents = append(growthEvents, AttributeProdigyPerformance(awayClub, report, "away", tm.GrowthEngine)...)
	// At most two clubs × two level-up lines can ever accumulate here, so no
	// truncation is needed (Python's [:8] is likewise unreachable).
	tm.MaybeInjure(homeClub, awayClub, report, matchweek, fixtureID)
	tm.mentorDramaForReport(homeClub, awayClub, report, matchweek, fixtureID)
	return growthEvents
}
