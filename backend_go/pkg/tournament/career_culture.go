package tournament

import (
	"fmt"
	"sort"
	"strings"

	"football_sim/pkg/models"
)

const europeanRegistrationLimit = 25

// RefreshClubCultureUnlocked stamps captains, homegrown flags, UEFA lists,
// chemistry, fan heat, and power ranks. Called after squad roles so rank is known.
func (tm *TournamentManager) RefreshClubCultureUnlocked() {
	if tm == nil {
		return
	}
	for _, club := range tm.ClubsList {
		tm.refreshOneClubCultureUnlocked(club)
	}
	tm.refreshPowerRanksUnlocked()
}

// SyncCaptainFlagsUnlocked makes player armband flags match persisted club
// captain_id / vice_captain_id after a sharded restore. Club IDs are
// authoritative; stale is_captain bits on other squad members are cleared.
// Missing IDs (legacy saves) get a fresh culture pass instead of failing
// world validation.
func (tm *TournamentManager) SyncCaptainFlagsUnlocked() {
	if tm == nil {
		return
	}
	for _, club := range tm.ClubsList {
		if club == nil {
			continue
		}
		captainOK, viceOK := false, false
		for _, p := range club.Squad {
			if p == nil {
				continue
			}
			p.IsCaptain = club.CaptainID != "" && p.PlayerID == club.CaptainID
			p.IsViceCaptain = club.ViceCaptainID != "" && p.PlayerID == club.ViceCaptainID
			if p.IsCaptain {
				captainOK = true
			}
			if p.IsViceCaptain {
				viceOK = true
			}
		}
		if club.CaptainID != "" && !captainOK {
			club.CaptainID = ""
		}
		if club.ViceCaptainID != "" && !viceOK {
			club.ViceCaptainID = ""
		}
		if club.CaptainID == "" {
			tm.refreshOneClubCultureUnlocked(club)
		}
	}
}

func (tm *TournamentManager) refreshOneClubCultureUnlocked(club *models.Club) {
	if club == nil {
		return
	}
	originalCountry := map[string]string{}
	for _, other := range tm.ClubsList {
		if other != nil {
			originalCountry[other.ClubID] = other.Country
		}
	}
	type cand struct {
		p    *models.Player
		lead int
	}
	var leaders []cand
	homegrown := 0
	moraleSum, moraleN := 0, 0
	for _, p := range club.Squad {
		if p == nil {
			continue
		}
		p.IsCaptain = false
		p.IsViceCaptain = false
		p.EnsureLeadership()
		p.Homegrown = p.IsClubTrained(club.ClubID)
		if orig := p.OriginalClubID; orig != "" && originalCountry[orig] != "" && originalCountry[orig] == club.Country {
			p.AssociationTrained = true
		} else {
			p.AssociationTrained = p.Homegrown
		}
		p.RefreshVersatility()
		if p.Homegrown {
			homegrown++
		}
		if !p.OnLoan {
			leaders = append(leaders, cand{p, p.Leadership})
		}
		moraleSum += p.Morale
		moraleN++
	}
	sort.SliceStable(leaders, func(i, j int) bool {
		if leaders[i].lead != leaders[j].lead {
			return leaders[i].lead > leaders[j].lead
		}
		if leaders[i].p.OVR != leaders[j].p.OVR {
			return leaders[i].p.OVR > leaders[j].p.OVR
		}
		if leaders[i].p.Age != leaders[j].p.Age {
			return leaders[i].p.Age > leaders[j].p.Age
		}
		return leaders[i].p.PlayerID < leaders[j].p.PlayerID
	})
	club.CaptainID, club.ViceCaptainID = "", ""
	if len(leaders) > 0 {
		leaders[0].p.IsCaptain = true
		club.CaptainID = leaders[0].p.PlayerID
	}
	if len(leaders) > 1 {
		leaders[1].p.IsViceCaptain = true
		club.ViceCaptainID = leaders[1].p.PlayerID
	}
	tm.registerEuropeanSquadUnlocked(club)
	avgMorale := 70
	if moraleN > 0 {
		avgMorale = moraleSum / moraleN
	}
	chem := avgMorale
	if club.CaptainID != "" {
		chem += 6
	}
	if homegrown > 8 {
		homegrown = 8
	}
	chem += homegrown
	wins := 0
	for _, r := range club.Form {
		if r == "W" {
			wins++
		}
	}
	chem += wins
	chem -= club.MediaPressure / 12
	club.Chemistry = clampDynamics(chem, 20, 99)
	club.FanExpectation = fanExpectationFromBoard(club)
	club.MediaPressure = tm.mediaPressureUnlocked(club)
}

func fanExpectationFromBoard(club *models.Club) int {
	if club == nil {
		return 55
	}
	switch {
	case club.ExpectedFinish <= 1:
		return 92
	case club.ExpectedFinish <= 4:
		return 80
	case club.ExpectedFinish <= 8:
		return 68
	case club.ExpectedFinish <= 12:
		return 55
	default:
		return 42
	}
}

func (tm *TournamentManager) mediaPressureUnlocked(club *models.Club) int {
	if club == nil || club.Played < 4 {
		return clampDynamics(club.MediaPressure-2, 0, 100)
	}
	table := tm.clubLeagueTableUnlocked(club)
	pos := 0
	for i, c := range table {
		if c != nil && c.ClubID == club.ClubID {
			pos = i + 1
			break
		}
	}
	if pos == 0 {
		return club.MediaPressure
	}
	expected := club.ExpectedFinish
	if expected <= 0 {
		expected = len(table)/2 + 1
	}
	pressure := 35 + (pos-expected)*8
	streak := 0
	for i := len(club.Form) - 1; i >= 0 && i >= len(club.Form)-5; i-- {
		if club.Form[i] == "L" {
			streak++
		} else {
			break
		}
	}
	pressure += streak * 6
	if pos < expected {
		pressure -= (expected - pos) * 4
	}
	return clampDynamics(pressure, 5, 96)
}

func (tm *TournamentManager) registerEuropeanSquadUnlocked(club *models.Club) {
	if club == nil {
		return
	}
	type row struct {
		p         *models.Player
		homegrown bool
		ovr       int
	}
	var pool []row
	for _, p := range club.Squad {
		if p == nil {
			continue
		}
		p.RegisteredEurope = false
		if p.OnLoan && p.ParentClubID == club.ClubID {
			continue
		}
		pool = append(pool, row{p: p, homegrown: p.Homegrown || p.AssociationTrained, ovr: p.OVR})
	}
	sort.SliceStable(pool, func(i, j int) bool {
		if pool[i].homegrown != pool[j].homegrown {
			return pool[i].homegrown && !pool[j].homegrown
		}
		if pool[i].ovr != pool[j].ovr {
			return pool[i].ovr > pool[j].ovr
		}
		return pool[i].p.PlayerID < pool[j].p.PlayerID
	})
	// Guarantee up to eight club/association-trained names, then fill on quality.
	picked := 0
	homegrownTaken := 0
	for _, r := range pool {
		if picked >= europeanRegistrationLimit {
			break
		}
		if r.homegrown && homegrownTaken < 8 {
			r.p.RegisteredEurope = true
			picked++
			homegrownTaken++
		}
	}
	for _, r := range pool {
		if picked >= europeanRegistrationLimit {
			break
		}
		if r.p.RegisteredEurope {
			continue
		}
		r.p.RegisteredEurope = true
		picked++
	}
}

func (tm *TournamentManager) refreshPowerRanksUnlocked() {
	type row struct {
		club  *models.Club
		score int
	}
	rows := make([]row, 0, len(tm.ClubsList))
	for _, club := range tm.ClubsList {
		if club == nil {
			continue
		}
		pace := club.Points * 4
		if club.Played > 0 {
			pace = (club.Points * 38) / club.Played
		}
		score := pace*3 + club.GoalDifference + club.OverallTeamRating*2 + club.Coefficient/2 + club.Chemistry/5
		rows = append(rows, row{club, score})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].score != rows[j].score {
			return rows[i].score > rows[j].score
		}
		if rows[i].club.OverallTeamRating != rows[j].club.OverallTeamRating {
			return rows[i].club.OverallTeamRating > rows[j].club.OverallTeamRating
		}
		return rows[i].club.ClubID < rows[j].club.ClubID
	})
	for i, r := range rows {
		r.club.PowerRank = i + 1
	}
}

func (tm *TournamentManager) runCultureWeeklyUnlocked(completedMW int) {
	tm.RefreshClubCultureUnlocked()
	tm.evaluatePromisesUnlocked(completedMW)
	tm.maybeLoanReportsUnlocked(completedMW)
	tm.maybeTacticalEvolutionUnlocked(completedMW)
	tm.heatTitleRivalriesUnlocked()
}

func (tm *TournamentManager) evaluatePromisesUnlocked(completedMW int) {
	if completedMW < 8 {
		return
	}
	brokenNotes := 0
	for _, club := range tm.ClubsList {
		if club == nil {
			continue
		}
		for _, p := range club.Squad {
			if p == nil || p.PromiseKind == "" {
				continue
			}
			if p.PromiseSeason != "" && p.PromiseSeason != tm.SeasonName {
				p.PromiseKind = ""
				continue
			}
			broken := false
			kept := false
			switch p.PromiseKind {
			case "minutes", "role":
				need := club.Played / 3
				if p.SquadRole == models.RoleCrucial {
					need = club.Played / 2
				}
				if club.Played >= 8 && p.Appearances < need {
					broken = true
				} else if club.Played >= 8 && p.Appearances >= need {
					kept = true
				}
			case "loan":
				if p.OnLoan {
					kept = true
				} else if completedMW >= p.PromiseMatchweek+6 {
					broken = true
				}
			case "contract":
				if p.ContractYears >= 2 {
					kept = true
				} else if completedMW >= p.PromiseMatchweek+8 {
					broken = true
				}
			case "europe":
				apps := 0
				if p.CompetitionStats != nil {
					for _, id := range []string{"champions-league", "europa-league", "conference-league", "ucl"} {
						if st := p.CompetitionStats[id]; st != nil {
							apps += st.Appearances
						}
					}
				}
				if apps > 0 {
					kept = true
				} else if completedMW >= 12 && tm.clubInEuropeUnlocked(club.ClubID) {
					broken = true
				}
			}
			if broken {
				p.AdjustMorale(-4)
				if p.Morale < 40 {
					p.TransferRequested = true
				}
				if brokenNotes < 2 {
					tm.PushInbox("dugout", p.FullName+" says a promise was broken",
						fmt.Sprintf("%s was promised %s at %s and no longer trusts the plan.", p.FullName, p.PromiseKind, club.ShortName),
						completedMW, []string{club.ClubID}, p.PlayerID, "")
					brokenNotes++
				}
				p.PromiseKind, p.PromiseSeason, p.PromiseMatchweek = "", "", 0
			} else if kept {
				p.AdjustMorale(3)
				p.PromiseKind, p.PromiseSeason, p.PromiseMatchweek = "", "", 0
			}
		}
	}
}

func (tm *TournamentManager) maybeLoanReportsUnlocked(completedMW int) {
	if completedMW < 4 || completedMW%4 != 0 {
		return
	}
	type line struct {
		id   string
		text string
	}
	var lines []line
	for _, club := range tm.ClubsList {
		if club == nil {
			continue
		}
		for _, p := range club.Squad {
			if p == nil || !p.OnLoan {
				continue
			}
			parent := p.ParentClubID
			minutes := 0
			if p.CompetitionStats != nil {
				for _, st := range p.CompetitionStats {
					if st != nil {
						minutes += st.Minutes
					}
				}
			}
			lines = append(lines, line{id: p.PlayerID, text: fmt.Sprintf("%s (%s, parent %s): %d apps, %d G, %d' — %s form",
				p.FullName, club.ShortName, parent, p.Appearances, p.Goals, minutes, p.FormBand())})
		}
	}
	if len(lines) == 0 {
		return
	}
	sort.SliceStable(lines, func(i, j int) bool { return lines[i].id < lines[j].id })
	body := "Loan watch this month:\n"
	limit := len(lines)
	if limit > 8 {
		limit = 8
	}
	for i := 0; i < limit; i++ {
		body += "• " + lines[i].text + "\n"
	}
	tm.PushInbox("youth", "Loan reports land", strings.TrimSpace(body), completedMW, nil, "", "")
}

func (tm *TournamentManager) maybeTacticalEvolutionUnlocked(completedMW int) {
	if completedMW < 10 {
		return
	}
	cycle := []string{"high_press", "possession", "low_block", "free_flowing"}
	for _, club := range tm.ClubsList {
		if club == nil {
			continue
		}
		mgr := tm.Managers[club.ClubID]
		if mgr == nil || mgr.Adaptability < 55 {
			continue
		}
		if last := tm.ManagerLastChange[club.ClubID]; last > 0 && completedMW-last < 8 {
			continue
		}
		if len(club.Form) < 3 {
			continue
		}
		n := len(club.Form)
		if club.Form[n-1] != "L" || club.Form[n-2] != "L" || club.Form[n-3] != "L" {
			continue
		}
		cur := mgr.CanonicalStyle()
		idx := 0
		for i, k := range cycle {
			if k == cur {
				idx = i
				break
			}
		}
		next := cycle[(idx+1)%len(cycle)]
		if next == cur {
			continue
		}
		old := mgr.Tactic()
		mgr.Style = next
		if tm.ManagerLastChange == nil {
			tm.ManagerLastChange = map[string]int{}
		}
		tm.ManagerLastChange[club.ClubID] = completedMW
		tm.PushInbox("dugout", fmt.Sprintf("%s tweak their shape", club.ShortName),
			fmt.Sprintf("%s abandon %s for %s after three straight league defeats.", mgr.Name, old, mgr.Tactic()),
			completedMW, []string{club.ClubID}, "", "")
	}
}

func (tm *TournamentManager) heatTitleRivalriesUnlocked() {
	if tm.DerbyHeat == nil {
		tm.DerbyHeat = map[string]int{}
	}
	byLeague := map[string][]*models.Club{}
	for _, club := range tm.ClubsList {
		if club == nil {
			continue
		}
		byLeague[club.League] = append(byLeague[club.League], club)
	}
	for _, clubs := range byLeague {
		ranked := append([]*models.Club(nil), clubs...)
		models.SortClubs(ranked)
		n := len(ranked)
		hot := map[string]bool{}
		for i, c := range ranked {
			if i < 4 || (n >= 6 && i >= n-3) {
				hot[c.ClubID] = true
			}
		}
		for i := 0; i < len(ranked); i++ {
			for j := i + 1; j < len(ranked); j++ {
				a, b := ranked[i], ranked[j]
				name := GetDerbyName(a.ClubID, b.ClubID)
				if name == "" {
					continue
				}
				if hot[a.ClubID] && hot[b.ClubID] {
					h := tm.DerbyHeat[name]
					if h < 50 {
						h = 50
					}
					if h < 85 {
						h += 2
					}
					tm.DerbyHeat[name] = h
				}
			}
		}
	}
}

func (tm *TournamentManager) pushSeasonPreviewUnlocked() {
	if tm == nil || len(tm.ClubsList) == 0 {
		return
	}
	ranked := append([]*models.Club(nil), tm.ClubsList...)
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].OverallTeamRating != ranked[j].OverallTeamRating {
			return ranked[i].OverallTeamRating > ranked[j].OverallTeamRating
		}
		if ranked[i].Identity.Reputation != ranked[j].Identity.Reputation {
			return ranked[i].Identity.Reputation > ranked[j].Identity.Reputation
		}
		return ranked[i].ClubID < ranked[j].ClubID
	})
	favs := []string{}
	for i := 0; i < len(ranked) && i < 3; i++ {
		favs = append(favs, ranked[i].ShortName)
	}
	scrap := []string{}
	for i := len(ranked) - 1; i >= 0 && len(scrap) < 3; i-- {
		scrap = append(scrap, ranked[i].ShortName)
	}
	body := fmt.Sprintf("%s open as the sides to beat. %s start closer to the trapdoor. Board briefs already sit on every desk.",
		strings.Join(favs, ", "), strings.Join(scrap, ", "))
	if fav := tm.Clubs[tm.FavouriteClubID]; fav != nil {
		body += fmt.Sprintf(" %s are briefed for %s (target #%d).", fav.ClubName, fav.BoardObjective, fav.ExpectedFinish)
	}
	tm.PushInbox("race", tm.SeasonName+" season preview", body, 1, nil, "", "")
}

func (tm *TournamentManager) powerRankingsUnlocked(n int) []map[string]interface{} {
	ranked := append([]*models.Club(nil), tm.ClubsList...)
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].PowerRank != ranked[j].PowerRank && ranked[i].PowerRank > 0 && ranked[j].PowerRank > 0 {
			return ranked[i].PowerRank < ranked[j].PowerRank
		}
		if ranked[i].OverallTeamRating != ranked[j].OverallTeamRating {
			return ranked[i].OverallTeamRating > ranked[j].OverallTeamRating
		}
		return ranked[i].ClubID < ranked[j].ClubID
	})
	if n > 0 && len(ranked) > n {
		ranked = ranked[:n]
	}
	out := make([]map[string]interface{}, 0, len(ranked))
	for _, c := range ranked {
		out = append(out, map[string]interface{}{
			"rank": c.PowerRank, "club_id": c.ClubID, "club_name": c.ClubName, "short_name": c.ShortName,
			"league": c.League, "ovr": c.OverallTeamRating, "pts": c.Points, "coefficient": c.Coefficient,
			"chemistry": c.Chemistry, "media_pressure": c.MediaPressure, "captain_id": c.CaptainID,
			"primary_color": c.PrimaryColor,
		})
	}
	return out
}

func (tm *TournamentManager) loanWatchUnlocked(n int) []map[string]interface{} {
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
			if p != nil && p.OnLoan {
				rows = append(rows, row{p, club})
			}
		}
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].p.PlayerID < rows[j].p.PlayerID })
	if n > 0 && len(rows) > n {
		rows = rows[:n]
	}
	out := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		out = append(out, map[string]interface{}{
			"player_id": r.p.PlayerID, "full_name": r.p.FullName, "position": r.p.Position,
			"ovr": r.p.OVR, "club_id": r.c.ClubID, "club_short": r.c.ShortName,
			"parent_club_id": r.p.ParentClubID, "appearances": r.p.Appearances,
			"goals": r.p.Goals, "assists": r.p.Assists, "form_band": r.p.FormBand(),
		})
	}
	return out
}
