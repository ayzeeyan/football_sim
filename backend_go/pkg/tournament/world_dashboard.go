package tournament

import (
	"sort"

	"football_sim/pkg/models"
	"football_sim/pkg/transfers"
)

// WorldDashboard is the career-home snapshot: five-league leaders, Europe,
// the transfer market, injuries, sackings, wonderkids, and the next big games.
// Built under one read lock from indexed club/player lists — never map iteration.
func (tm *TournamentManager) WorldDashboard() map[string]interface{} {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	favID := tm.FavouriteClubID
	var nextFixture map[string]interface{}
	if favID != "" {
		_, fav, _, _ := tm.weekWatchUnlocked()
		if fav != nil && fav.Status != "finished" {
			nextFixture = tm.compactCompetitionFixtureUnlocked(fav)
		}
	}

	return map[string]interface{}{
		"world":             tm.World != nil,
		"season_name":       tm.SeasonName,
		"season_phase":      tm.SeasonPhase,
		"current_matchweek": tm.CurrentMatchweek,
		"max_matchweeks":    tm.MaxMatchweeks,
		"favourite_club_id": favID,
		"favourite_club":    compactClub(tm.Clubs[favID]),
		"next_fixture":      nextFixture,
		"league_leaders":    tm.dashboardLeagueLeadersUnlocked(),
		"europe":            tm.dashboardEuropeUnlocked(),
		"top_scorer":        tm.dashboardTopScorerUnlocked(),
		"biggest_transfers": tm.dashboardTransfersUnlocked(5),
		"injuries":          tm.dashboardInjuriesUnlocked(8),
		"sackings":          tm.dashboardSackingsUnlocked(5),
		"wonderkids":        tm.dashboardWonderkidsUnlocked(6),
		"upcoming_fixtures": tm.dashboardUpcomingUnlocked(8),
		"headlines":         tm.dashboardHeadlinesUnlocked(8),
		"transfer_window":   tm.dashboardWindowUnlocked(),
		"unread_inbox":      tm.dashboardUnreadUnlocked(),
		"power_rankings":    tm.powerRankingsUnlocked(10),
		"loan_watch":        tm.loanWatchUnlocked(8),
	}
}

func (tm *TournamentManager) weekWatchUnlocked() (favID string, fav *Fixture, cups []Fixture, mw int) {
	// WeekWatch takes the manager lock; callers already holding it use this copy.
	mw = tm.CurrentMatchweek
	if mw > tm.MaxMatchweeks {
		mw = tm.MaxMatchweeks
	}
	if mw < 1 {
		mw = 1
	}
	favID = tm.FavouriteClubID
	slate := tm.slateUnlocked(mw)
	var favPtr *Fixture
	for _, f := range slate {
		if favID != "" && (f.HomeID == favID || f.AwayID == favID) {
			if favPtr == nil || ((!tm.isWorldDomesticLeague(favPtr.Competition) && favPtr.Competition != "super-league") && (tm.isWorldDomesticLeague(f.Competition) || f.Competition == "super-league")) {
				favPtr = f
			}
		}
	}
	if favPtr != nil {
		cp := *favPtr
		fav = &cp
	}
	for _, f := range slate {
		if f.Competition != "ucl" && f.Competition != "super-cup" && !tm.isWorldKnockoutFixture(f) && tm.worldCompetitionUnlocked(f.Competition) == nil {
			continue
		}
		if fav != nil && f.FixtureID == fav.FixtureID {
			continue
		}
		cups = append(cups, *f)
	}
	if cups == nil {
		cups = []Fixture{}
	}
	return favID, fav, cups, mw
}

func (tm *TournamentManager) dashboardLeagueLeadersUnlocked() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, 5)
	if tm.World != nil {
		for _, def := range domesticLeagueDefinitions {
			table := tm.worldLeagueStandingsUnlocked(def.ID)
			if len(table) == 0 {
				continue
			}
			leader := table[0]
			entry := compactClub(leader)
			entry["competition_id"] = def.ID
			entry["competition_name"] = def.Name
			entry["position"] = 1
			if len(table) > 1 {
				entry["pts_gap"] = leader.Points - table[1].Points
				entry["second"] = table[1].ShortName
			}
			out = append(out, entry)
		}
		return out
	}
	clubs := append([]*models.Club(nil), tm.ClubsList...)
	models.SortClubs(clubs)
	if len(clubs) == 0 {
		return out
	}
	leader := clubs[0]
	entry := compactClub(leader)
	entry["competition_id"] = "super-league"
	entry["competition_name"] = "Super League"
	entry["position"] = 1
	if len(clubs) > 1 {
		entry["pts_gap"] = leader.Points - clubs[1].Points
		entry["second"] = clubs[1].ShortName
	}
	return []map[string]interface{}{entry}
}

func (tm *TournamentManager) dashboardEuropeUnlocked() map[string]interface{} {
	out := map[string]interface{}{
		"stage": "", "leader": nil, "champion": nil, "competition_id": "champions-league",
		"name": "UEFA Champions League",
	}
	if tm.World == nil {
		out["stage"] = tm.UCLStage
		if tm.UCLChampionID != "" {
			out["champion"] = compactClub(tm.Clubs[tm.UCLChampionID])
		}
		return out
	}
	comp := tm.worldCompetitionUnlocked("champions-league")
	if comp == nil {
		return out
	}
	out["stage"] = comp.Stage
	out["name"] = comp.Name
	if comp.ChampionID != "" {
		out["champion"] = compactClub(tm.Clubs[comp.ChampionID])
	}
	table := tm.worldEuropeanStandingsUnlocked(comp)
	if len(table) > 0 {
		row := compactClub(table[0])
		if rec := comp.Records[table[0].ClubID]; rec != nil {
			row["pts"] = rec.Points
			row["p"] = rec.Played
			row["gd"] = rec.GoalDifference
		}
		out["leader"] = row
	}
	return out
}

func (tm *TournamentManager) dashboardTopScorerUnlocked() map[string]interface{} {
	var best *models.Player
	bestGoals := -1
	for _, club := range tm.ClubsList {
		if club == nil {
			continue
		}
		for _, p := range club.Squad {
			if p == nil {
				continue
			}
			g := p.Goals
			if best == nil || g > bestGoals || (g == bestGoals && p.Assists > best.Assists) || (g == bestGoals && p.Assists == best.Assists && p.PlayerID < best.PlayerID) {
				best = p
				bestGoals = g
			}
		}
	}
	if best == nil {
		return nil
	}
	club := tm.Clubs[best.ClubID]
	row := map[string]interface{}{
		"player_id": best.PlayerID, "full_name": best.FullName, "position": best.Position,
		"ovr": best.OVR, "goals": best.Goals, "assists": best.Assists, "appearances": best.Appearances,
		"club_id": best.ClubID, "is_wonderkid": best.UniverseWonderkid,
	}
	if club != nil {
		row["club_name"] = club.ClubName
		row["club_short"] = club.ShortName
		row["primary_color"] = club.PrimaryColor
	}
	return row
}

func (tm *TournamentManager) dashboardTransfersUnlocked(n int) []map[string]interface{} {
	if tm.TransferEngine == nil || n <= 0 {
		return []map[string]interface{}{}
	}
	all := append([]transfers.CompletedTransfer(nil), tm.TransferEngine.CompletedTransfers...)
	sort.SliceStable(all, func(i, j int) bool {
		if all[i].FeeEUR != all[j].FeeEUR {
			return all[i].FeeEUR > all[j].FeeEUR
		}
		if all[i].Matchweek != all[j].Matchweek {
			return all[i].Matchweek > all[j].Matchweek
		}
		return all[i].PlayerID < all[j].PlayerID
	})
	if len(all) > n {
		all = all[:n]
	}
	out := make([]map[string]interface{}, 0, len(all))
	for _, tr := range all {
		out = append(out, map[string]interface{}{
			"player_id": tr.PlayerID, "player_name": tr.PlayerName, "player_pos": tr.PlayerPos,
			"player_ovr": tr.PlayerOVR, "is_wonderkid": tr.IsWonderkid,
			"seller_id": tr.SellerID, "seller_name": tr.SellerName, "seller_short": tr.SellerShort,
			"buyer_id": tr.BuyerID, "buyer_name": tr.BuyerName, "buyer_short": tr.BuyerShort,
			"fee_eur": tr.FeeEUR, "formatted_fee": tr.FormattedFee, "matchweek": tr.Matchweek,
		})
	}
	return out
}

func (tm *TournamentManager) dashboardInjuriesUnlocked(n int) []map[string]interface{} {
	type row struct {
		p *models.Player
		c *models.Club
	}
	var rows []row
	for _, club := range tm.ClubsList {
		if club == nil {
			continue
		}
		for _, p := range club.Squad {
			if p == nil || p.InjuredMatches <= 0 {
				continue
			}
			rows = append(rows, row{p, club})
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].p.OVR != rows[j].p.OVR {
			return rows[i].p.OVR > rows[j].p.OVR
		}
		if rows[i].p.InjuredMatches != rows[j].p.InjuredMatches {
			return rows[i].p.InjuredMatches > rows[j].p.InjuredMatches
		}
		return rows[i].p.PlayerID < rows[j].p.PlayerID
	})
	if len(rows) > n {
		rows = rows[:n]
	}
	out := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		out = append(out, map[string]interface{}{
			"player_id": r.p.PlayerID, "full_name": r.p.FullName, "position": r.p.Position,
			"ovr": r.p.OVR, "injury": r.p.Injury, "injured_matches": r.p.InjuredMatches,
			"club_id": r.c.ClubID, "club_name": r.c.ClubName, "club_short": r.c.ShortName,
		})
	}
	return out
}

func (tm *TournamentManager) dashboardSackingsUnlocked(n int) []map[string]interface{} {
	out := []map[string]interface{}{}
	for i := len(tm.ManagerHistory) - 1; i >= 0 && len(out) < n; i-- {
		e := tm.ManagerHistory[i]
		action := e.Action
		if action == "" {
			action = e.Reason
		}
		out = append(out, map[string]interface{}{
			"season_name": e.SeasonName, "matchweek": e.Matchweek, "club_id": e.ClubID,
			"club_name": e.ClubName, "action": action, "old_manager": e.OldManager,
			"new_manager": e.NewManager, "reason": e.Reason, "job_security": e.JobSecurity,
		})
	}
	return out
}

func (tm *TournamentManager) dashboardWonderkidsUnlocked(n int) []map[string]interface{} {
	watch := tm.prodigyWatchUnlocked(n)
	if len(watch) > 0 {
		return watch
	}
	return []map[string]interface{}{}
}

func (tm *TournamentManager) prodigyWatchUnlocked(n int) []map[string]interface{} {
	type item struct {
		p     *models.Player
		c     *models.Club
		score float64
	}
	seen := map[string]bool{}
	var rows []item
	for _, c := range tm.ClubsList {
		if c == nil {
			continue
		}
		for _, p := range c.Squad {
			if p == nil || !p.UniverseWonderkid || seen[p.PlayerID] {
				continue
			}
			seen[p.PlayerID] = true
			rows = append(rows, item{p: p, c: c, score: tm.goldenBoyScoreUnlocked(p)})
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].score != rows[j].score {
			return rows[i].score > rows[j].score
		}
		if rows[i].p.OVR != rows[j].p.OVR {
			return rows[i].p.OVR > rows[j].p.OVR
		}
		if rows[i].p.FullName != rows[j].p.FullName {
			return rows[i].p.FullName < rows[j].p.FullName
		}
		return rows[i].p.PlayerID < rows[j].p.PlayerID
	})
	if n > 0 && len(rows) > n {
		rows = rows[:n]
	}
	out := make([]map[string]interface{}, 0, len(rows))
	for i, row := range rows {
		p, c := row.p, row.c
		entry := map[string]interface{}{
			"rank": i + 1, "player_id": p.PlayerID, "full_name": p.FullName,
			"club_id": p.ClubID, "age": p.Age, "position": p.Position,
			"ovr": p.OVR, "goals": p.Goals, "assists": p.Assists, "appearances": p.Appearances,
		}
		if c != nil {
			entry["club_name"], entry["club_short"] = c.ClubName, c.ShortName
		}
		out = append(out, entry)
	}
	return out
}

func (tm *TournamentManager) dashboardUpcomingUnlocked(n int) []map[string]interface{} {
	mw := tm.CurrentMatchweek
	if mw < 1 {
		mw = 1
	}
	if mw > tm.MaxMatchweeks {
		mw = tm.MaxMatchweeks
	}
	slate := tm.slateUnlocked(mw)
	type ranked struct {
		f     *Fixture
		score int
	}
	var rows []ranked
	for _, f := range slate {
		if f == nil || f.Status == "finished" {
			continue
		}
		home, away := tm.Clubs[f.HomeID], tm.Clubs[f.AwayID]
		hOVR, aOVR := 0, 0
		if home != nil {
			hOVR = home.OverallTeamRating
		}
		if away != nil {
			aOVR = away.OverallTeamRating
		}
		score := hOVR + aOVR
		if f.DerbyName != "" || f.IsHighHeatDerby {
			score += 12
		}
		if tm.isWorldKnockoutFixture(f) || f.Competition == "champions-league" || f.Competition == "ucl" {
			score += 8
		}
		rows = append(rows, ranked{f, score})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].score != rows[j].score {
			return rows[i].score > rows[j].score
		}
		return rows[i].f.FixtureID < rows[j].f.FixtureID
	})
	if len(rows) > n {
		rows = rows[:n]
	}
	out := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		out = append(out, tm.compactCompetitionFixtureUnlocked(r.f))
	}
	return out
}

func (tm *TournamentManager) dashboardHeadlinesUnlocked(n int) []map[string]interface{} {
	out := []map[string]interface{}{}
	for i := 0; i < len(tm.Inbox) && len(out) < n; i++ {
		item := tm.Inbox[i]
		out = append(out, map[string]interface{}{
			"id": item.ID, "category": item.Category, "headline": item.Headline,
			"body": item.Body, "matchweek": item.Matchweek, "unread": item.Unread,
			"player_id": item.PlayerID, "fixture_id": item.FixtureID, "club_ids": item.ClubIDs,
		})
	}
	return out
}

func (tm *TournamentManager) dashboardWindowUnlocked() map[string]interface{} {
	te := tm.TransferEngine
	if te == nil {
		return map[string]interface{}{"open": false, "type": "CLOSED", "week": 0, "weeks": 0}
	}
	return map[string]interface{}{
		"open":  te.IsWindowOpen(),
		"type":  te.WindowType,
		"week":  te.CurrentWeek,
		"weeks": te.WindowWeeks(),
	}
}

func (tm *TournamentManager) dashboardUnreadUnlocked() int {
	n := 0
	for _, item := range tm.Inbox {
		if item.Unread {
			n++
		}
	}
	return n
}

func (tm *TournamentManager) competitionLeadersUnlocked(compID string) (scorers, assisters []map[string]interface{}) {
	type row struct {
		p  *models.Player
		c  *models.Club
		st *models.CompetitionSeasonStats
	}
	var rows []row
	for _, club := range tm.ClubsList {
		if club == nil {
			continue
		}
		for _, p := range club.Squad {
			if p == nil || p.CompetitionStats == nil {
				continue
			}
			st := p.CompetitionStats[compID]
			if st == nil || (st.Goals == 0 && st.Assists == 0 && st.Appearances == 0) {
				continue
			}
			rows = append(rows, row{p, club, st})
		}
	}
	scorerCopy := append([]row(nil), rows...)
	sort.SliceStable(scorerCopy, func(i, j int) bool {
		if scorerCopy[i].st.Goals != scorerCopy[j].st.Goals {
			return scorerCopy[i].st.Goals > scorerCopy[j].st.Goals
		}
		if scorerCopy[i].st.Assists != scorerCopy[j].st.Assists {
			return scorerCopy[i].st.Assists > scorerCopy[j].st.Assists
		}
		return scorerCopy[i].p.PlayerID < scorerCopy[j].p.PlayerID
	})
	assistCopy := append([]row(nil), rows...)
	sort.SliceStable(assistCopy, func(i, j int) bool {
		if assistCopy[i].st.Assists != assistCopy[j].st.Assists {
			return assistCopy[i].st.Assists > assistCopy[j].st.Assists
		}
		if assistCopy[i].st.Goals != assistCopy[j].st.Goals {
			return assistCopy[i].st.Goals > assistCopy[j].st.Goals
		}
		return assistCopy[i].p.PlayerID < assistCopy[j].p.PlayerID
	})
	pack := func(src []row, n int) []map[string]interface{} {
		if len(src) > n {
			src = src[:n]
		}
		out := make([]map[string]interface{}, 0, len(src))
		for _, r := range src {
			out = append(out, map[string]interface{}{
				"player_id": r.p.PlayerID, "full_name": r.p.FullName, "position": r.p.Position,
				"ovr": r.p.OVR, "goals": r.st.Goals, "assists": r.st.Assists,
				"appearances": r.st.Appearances, "starts": r.st.Starts, "minutes": r.st.Minutes,
				"club_id": r.c.ClubID, "club_name": r.c.ClubName, "club_short": r.c.ShortName,
			})
		}
		return out
	}
	return pack(scorerCopy, 8), pack(assistCopy, 8)
}

func (tm *TournamentManager) competitionHistoryUnlocked(compID string) []map[string]interface{} {
	out := []map[string]interface{}{}
	for _, raw := range tm.SeasonHistory {
		if raw == nil {
			continue
		}
		season, _ := raw["season_name"].(string)
		entry := map[string]interface{}{"season_name": season}
		found := false
		if champs, ok := raw["competition_champions"].(map[string]string); ok {
			if id := champs[compID]; id != "" {
				entry["champion"] = compactClub(tm.Clubs[id])
				entry["champion_id"] = id
				found = true
			}
		}
		if !found {
			if champs, ok := raw["competition_champions"].(map[string]interface{}); ok {
				if v, ok := champs[compID].(string); ok && v != "" {
					entry["champion"] = compactClub(tm.Clubs[v])
					entry["champion_id"] = v
					found = true
				}
			}
		}
		if !found {
			if leagues, ok := raw["league_champions"].(map[string]interface{}); ok {
				if club, ok := leagues[compID].(*models.Club); ok && club != nil {
					entry["champion"] = compactClub(club)
					entry["champion_id"] = club.ClubID
					found = true
				}
			}
		}
		if found {
			out = append(out, entry)
		}
	}
	return out
}
