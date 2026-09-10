package tournament

import (
	"fmt"
	"sort"
	"strings"

	"football_sim/pkg/managers"
	"football_sim/pkg/matchreport"
	"football_sim/pkg/models"
)

// ManagerHistoryEntry is the persistent audit trail for autonomous manager
// appointments, sackings, and the tactical identity that changed with them.
type ManagerHistoryEntry struct {
	SeasonName  string `json:"season_name"`
	Matchweek   int    `json:"matchweek"`
	ClubID      string `json:"club_id"`
	ClubName    string `json:"club_name"`
	Action      string `json:"action"`
	OldManager  string `json:"old_manager,omitempty"`
	NewManager  string `json:"new_manager"`
	OldStyle    string `json:"old_style,omitempty"`
	NewStyle    string `json:"new_style"`
	Reason      string `json:"reason"`
	JobSecurity string `json:"job_security"`
}

// DigestScorer is intentionally small: the macro simulator should provide the
// story of a matchweek without forcing the full match-report payload over the wire.
type DigestScorer struct {
	PlayerID string `json:"player_id,omitempty"`
	Name     string `json:"name"`
	Minute   int    `json:"minute"`
	Side     string `json:"side"`
	OwnGoal  bool   `json:"own_goal,omitempty"`
}

type DigestFixture struct {
	FixtureID    string         `json:"fixture_id"`
	Competition  string         `json:"competition"`
	Stage        string         `json:"stage,omitempty"`
	HomeID       string         `json:"home_id"`
	AwayID       string         `json:"away_id"`
	HomeName     string         `json:"home_name"`
	AwayName     string         `json:"away_name"`
	HomeShort    string         `json:"home_short"`
	AwayShort    string         `json:"away_short"`
	HomeGoals    int            `json:"home_goals"`
	AwayGoals    int            `json:"away_goals"`
	Scorers      []DigestScorer `json:"scorers"`
	MOTM         string         `json:"motm,omitempty"`
	RedCards     []string       `json:"red_cards"`
	IsUpset      bool           `json:"is_upset"`
	UpsetLabel   string         `json:"upset_label,omitempty"`
	DecidedBy    string         `json:"decided_by,omitempty"`
	PenaltyScore []int          `json:"penalty_score,omitempty"`
}

type TableMovement struct {
	ClubID    string `json:"club_id"`
	ClubName  string `json:"club_name"`
	ShortName string `json:"short_name"`
	Before    int    `json:"before"`
	After     int    `json:"after"`
	Delta     int    `json:"delta"` // positive = climbed the table
}

type WonderkidHighlight struct {
	PlayerID      string  `json:"player_id"`
	FullName      string  `json:"full_name"`
	ClubID        string  `json:"club_id"`
	ClubShort     string  `json:"club_short"`
	Age           int     `json:"age"`
	Goals         int     `json:"goals"`
	Assists       int     `json:"assists"`
	Appearances   int     `json:"appearances"`
	OVR           int     `json:"ovr"`
	OVRDelta      int     `json:"ovr_delta"`
	HeightDeltaCM float64 `json:"height_delta_cm"`
	WeightDeltaKG float64 `json:"weight_delta_kg"`
	Note          string  `json:"note,omitempty"`
}

type MatchweekDigest struct {
	SeasonName          string                `json:"season_name"`
	Matchweek           int                   `json:"matchweek"`
	Month               string                `json:"month"`
	Year                int                   `json:"year"`
	CalendarLabel       string                `json:"calendar_label"`
	Results             []DigestFixture       `json:"results"`
	UpsetOfTheWeek      *DigestFixture        `json:"upset_of_the_week,omitempty"`
	TableMovement       []TableMovement       `json:"table_movement"`
	WonderkidHighlights []WonderkidHighlight  `json:"wonderkid_highlights"`
	ManagerEvents       []ManagerHistoryEntry `json:"manager_events"`
	Played              int                   `json:"played"`
	Skipped             int                   `json:"skipped"`
	SeasonFinished      bool                  `json:"season_finished"`
	Champion            string                `json:"champion,omitempty"`
}

// BatchSimResult is the stable commissioner API contract used by Sim Week,
// Sim Month, and Sim Season. It represents either league matchweeks or
// off-season transfer weeks without mixing the two phases in one batch.
type BatchSimResult struct {
	Status                 string            `json:"status"`
	Mode                   string            `json:"mode"`
	SeasonName             string            `json:"season_name"`
	SeasonPhase            string            `json:"season_phase"`
	CurrentMatchweek       int               `json:"current_matchweek"`
	StartMatchweek         int               `json:"start_matchweek,omitempty"`
	EndMatchweek           int               `json:"end_matchweek,omitempty"`
	WeeksAdvanced          int               `json:"weeks_advanced"`
	WeeksSimulated         int               `json:"weeks_simulated,omitempty"`
	OffSeasonWeeksAdvanced int               `json:"offseason_weeks_advanced,omitempty"`
	Played                 int               `json:"played"`
	Skipped                int               `json:"skipped"`
	Digests                []MatchweekDigest `json:"digests"`
	SeasonFinished         bool              `json:"season_finished"`
	AwardsReady            bool              `json:"awards_ready"`
	OffSeasonComplete      bool              `json:"offseason_complete,omitempty"`
	NewSeasonStarted       bool              `json:"new_season_started,omitempty"`
	Champion               string            `json:"champion,omitempty"`
	CalendarLabel          string            `json:"calendar_label"`
	Message                string            `json:"message,omitempty"`
}

type wonderkidDigestSnapshot struct {
	goals, assists, apps int
	ovr                  int
	height, weight       float64
}

func standingsPositions(clubs []*models.Club) map[string]int {
	out := make(map[string]int, len(clubs))
	for i, c := range clubs {
		if c != nil {
			out[c.ClubID] = i + 1
		}
	}
	return out
}

func (tm *TournamentManager) wonderkidDigestSnapshotsUnlocked() map[string]wonderkidDigestSnapshot {
	out := map[string]wonderkidDigestSnapshot{}
	for _, club := range tm.ClubsList {
		for _, p := range club.Squad {
			if p == nil || !p.UniverseWonderkid {
				continue
			}
			s := wonderkidDigestSnapshot{goals: p.Goals, assists: p.Assists, apps: p.Appearances, ovr: p.OVR}
			if tm.GrowthEngine != nil {
				if b := tm.GrowthEngine.Biometrics[p.PlayerID]; b != nil {
					s.height, s.weight = b.CurrentHeightCM, b.CurrentWeightKG
				}
			}
			out[p.PlayerID] = s
		}
	}
	return out
}

func digestClubNames(tm *TournamentManager, id string) (string, string) {
	if c := tm.Clubs[id]; c != nil {
		return c.ClubName, c.ShortName
	}
	return id, id
}

func digestFixtureFromReport(tm *TournamentManager, f *Fixture, before map[string]int) DigestFixture {
	homeName, homeShort := digestClubNames(tm, f.HomeID)
	awayName, awayShort := digestClubNames(tm, f.AwayID)
	row := DigestFixture{
		FixtureID: f.FixtureID, Competition: f.Competition, Stage: f.Stage,
		HomeID: f.HomeID, AwayID: f.AwayID, HomeName: homeName, AwayName: awayName,
		HomeShort: homeShort, AwayShort: awayShort, RedCards: []string{}, Scorers: []DigestScorer{},
		DecidedBy: f.DecidedBy,
	}
	if f.HomeGoals != nil {
		row.HomeGoals = *f.HomeGoals
	}
	if f.AwayGoals != nil {
		row.AwayGoals = *f.AwayGoals
	}
	if len(f.Penalties) > 0 {
		row.PenaltyScore = append([]int(nil), f.Penalties...)
	}
	if f.Report != nil {
		if f.Report.MOTM != nil {
			row.MOTM = f.Report.MOTM.FullName
		}
		for _, e := range f.Report.Events {
			switch e.Type {
			case "goal", "penalty", "corner_goal", "free_kick_goal", "own_goal":
				if e.Disallowed || e.Scorer == nil {
					continue
				}
				row.Scorers = append(row.Scorers, DigestScorer{
					PlayerID: e.Scorer.PlayerID, Name: e.Scorer.FullName, Minute: e.Minute, Side: e.Side, OwnGoal: e.Type == "own_goal",
				})
			case "red", "red_card":
				if e.Player != nil {
					row.RedCards = append(row.RedCards, e.Player.FullName)
				}
			}
		}
	}

	// Upsets are league-table narratives only; cup matches lack a meaningful
	// pre-match league-seed comparison once knockout rounds are underway.
	if f.Competition == "" || f.Competition == "super-league" {
		hp, ap := before[f.HomeID], before[f.AwayID]
		if row.HomeGoals > row.AwayGoals && hp >= 7 && ap > 0 && ap <= 3 {
			row.IsUpset = true
			row.UpsetLabel = fmt.Sprintf("%s stunned %d%s-place %s", homeShort, ap, ordinalSuffix(ap), awayShort)
		} else if row.AwayGoals > row.HomeGoals && ap >= 7 && hp > 0 && hp <= 3 {
			row.IsUpset = true
			row.UpsetLabel = fmt.Sprintf("%s stunned %d%s-place %s", awayShort, hp, ordinalSuffix(hp), homeShort)
		}
	}
	return row
}

func ordinalSuffix(n int) string {
	if n%100 >= 11 && n%100 <= 13 {
		return "th"
	}
	switch n % 10 {
	case 1:
		return "st"
	case 2:
		return "nd"
	case 3:
		return "rd"
	default:
		return "th"
	}
}

func (tm *TournamentManager) wonderkidHighlightsUnlocked(before map[string]wonderkidDigestSnapshot) []WonderkidHighlight {
	out := []WonderkidHighlight{}
	for _, club := range tm.ClubsList {
		for _, p := range club.Squad {
			if p == nil || !p.UniverseWonderkid {
				continue
			}
			prev, ok := before[p.PlayerID]
			if !ok {
				continue
			}
			h, w := prev.height, prev.weight
			if tm.GrowthEngine != nil {
				if b := tm.GrowthEngine.Biometrics[p.PlayerID]; b != nil {
					h, w = b.CurrentHeightCM, b.CurrentWeightKG
				}
			}
			goals := p.Goals - prev.goals
			assists := p.Assists - prev.assists
			apps := p.Appearances - prev.apps
			ovrDelta := p.OVR - prev.ovr
			hd := round2(h - prev.height)
			wd := round2(w - prev.weight)
			if goals == 0 && assists == 0 && apps == 0 && ovrDelta == 0 && hd == 0 && wd == 0 {
				continue
			}
			notes := []string{}
			if goals > 0 {
				notes = append(notes, fmt.Sprintf("%d goal(s)", goals))
			}
			if assists > 0 {
				notes = append(notes, fmt.Sprintf("%d assist(s)", assists))
			}
			if ovrDelta > 0 {
				notes = append(notes, fmt.Sprintf("+%d OVR", ovrDelta))
			}
			if hd > 0 || wd > 0 {
				notes = append(notes, "physical growth milestone")
			}
			out = append(out, WonderkidHighlight{
				PlayerID: p.PlayerID, FullName: p.FullName, ClubID: club.ClubID, ClubShort: club.ShortName,
				Age: p.Age, Goals: goals, Assists: assists, Appearances: apps, OVR: p.OVR, OVRDelta: ovrDelta,
				HeightDeltaCM: hd, WeightDeltaKG: wd, Note: strings.Join(notes, " · "),
			})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		si := out[i].Goals*5 + out[i].Assists*3 + out[i].OVRDelta*2 + out[i].Appearances
		sj := out[j].Goals*5 + out[j].Assists*3 + out[j].OVRDelta*2 + out[j].Appearances
		if si != sj {
			return si > sj
		}
		return out[i].FullName < out[j].FullName
	})
	return out
}

func round2(v float64) float64 {
	if v >= 0 {
		return float64(int(v*100+0.5)) / 100
	}
	return float64(int(v*100-0.5)) / 100
}

func (tm *TournamentManager) simulateWeekWithDigestUnlocked() MatchweekDigest {
	mw := tm.CurrentMatchweek
	beforeTable := tm.standingsUnlocked()
	beforePos := standingsPositions(beforeTable)
	beforeKids := tm.wonderkidDigestSnapshotsUnlocked()
	managerHistoryStart := len(tm.ManagerHistory)
	slate := tm.slateUnlocked(mw)
	ids := make([]string, 0, len(slate))
	for _, f := range slate {
		if f != nil && f.Status == "scheduled" {
			ids = append(ids, f.FixtureID)
		}
	}

	res := tm.simulateRemainingUnlocked()
	digest := MatchweekDigest{
		SeasonName: tm.SeasonName, Matchweek: mw, Month: MonthLabel(mw), Year: CalendarYear(tm.SeasonName, mw),
		CalendarLabel: CalendarLabel(tm.SeasonName, mw), Results: []DigestFixture{}, TableMovement: []TableMovement{},
		WonderkidHighlights: []WonderkidHighlight{}, ManagerEvents: []ManagerHistoryEntry{},
	}
	if played, ok := res["played"].(int); ok {
		digest.Played = played
	}
	if skipped, ok := res["skipped"].(int); ok {
		digest.Skipped = skipped
	}
	for _, id := range ids {
		f := tm.findFixtureUnlocked(id)
		if f == nil || f.Status != "finished" {
			continue
		}
		row := digestFixtureFromReport(tm, f, beforePos)
		digest.Results = append(digest.Results, row)
		if row.IsUpset {
			cp := row
			if digest.UpsetOfTheWeek == nil {
				digest.UpsetOfTheWeek = &cp
			} else {
				// Prefer the lower-ranked winner when there are multiple shocks.
				currentWinnerPos := 0
				candidateWinnerPos := 0
				if digest.UpsetOfTheWeek.HomeGoals > digest.UpsetOfTheWeek.AwayGoals {
					currentWinnerPos = beforePos[digest.UpsetOfTheWeek.HomeID]
				} else {
					currentWinnerPos = beforePos[digest.UpsetOfTheWeek.AwayID]
				}
				if row.HomeGoals > row.AwayGoals {
					candidateWinnerPos = beforePos[row.HomeID]
				} else {
					candidateWinnerPos = beforePos[row.AwayID]
				}
				if candidateWinnerPos > currentWinnerPos {
					digest.UpsetOfTheWeek = &cp
				}
			}
		}
	}

	after := tm.standingsUnlocked()
	afterPos := standingsPositions(after)
	for _, c := range after {
		b := beforePos[c.ClubID]
		a := afterPos[c.ClubID]
		if b == 0 || a == 0 || a == b {
			continue
		}
		digest.TableMovement = append(digest.TableMovement, TableMovement{
			ClubID: c.ClubID, ClubName: c.ClubName, ShortName: c.ShortName, Before: b, After: a, Delta: b - a,
		})
	}
	sort.SliceStable(digest.TableMovement, func(i, j int) bool {
		ai, aj := digest.TableMovement[i].Delta, digest.TableMovement[j].Delta
		if ai < 0 {
			ai = -ai
		}
		if aj < 0 {
			aj = -aj
		}
		if ai != aj {
			return ai > aj
		}
		return digest.TableMovement[i].After < digest.TableMovement[j].After
	})
	digest.WonderkidHighlights = tm.wonderkidHighlightsUnlocked(beforeKids)
	if managerHistoryStart < len(tm.ManagerHistory) {
		digest.ManagerEvents = append(digest.ManagerEvents, tm.ManagerHistory[managerHistoryStart:]...)
	}
	digest.SeasonFinished = tm.SeasonPhase == "transfer_window"
	if digest.SeasonFinished {
		digest.Champion = tm.championNameUnlocked()
	}
	return digest
}

// SimulateSlateWithDigest advances exactly one matchweek and returns a rich
// commissioner digest. It is the macro equivalent of SimulateRemaining.
func (tm *TournamentManager) SimulateSlateWithDigest() MatchweekDigest {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if tm.SeasonPhase != "season" || tm.CurrentMatchweek > tm.MaxMatchweeks {
		return MatchweekDigest{
			SeasonName: tm.SeasonName, Matchweek: tm.CurrentMatchweek, Results: []DigestFixture{},
			TableMovement: []TableMovement{}, WonderkidHighlights: []WonderkidHighlight{}, ManagerEvents: []ManagerHistoryEntry{},
			SeasonFinished: tm.SeasonPhase == "transfer_window", Champion: tm.championNameUnlocked(),
		}
	}
	return tm.simulateWeekWithDigestUnlocked()
}

// SimulateBatchWeeks runs up to count league matchweeks and stops cleanly at
// the season boundary. It intentionally never enters off-season market time.
func (tm *TournamentManager) SimulateBatchWeeks(count int) BatchSimResult {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if count < 1 {
		count = 1
	}
	start := tm.CurrentMatchweek
	result := BatchSimResult{
		Status: "success", Mode: "batch", SeasonName: tm.SeasonName, SeasonPhase: tm.SeasonPhase,
		CurrentMatchweek: tm.CurrentMatchweek, StartMatchweek: start, EndMatchweek: tm.CurrentMatchweek,
		Digests: []MatchweekDigest{},
	}

	// Defensive phase validation guard
	if tm.SeasonPhase != "season" && tm.SeasonPhase != "transfer_window" {
		result.Status = "error"
		result.Message = fmt.Sprintf("Unknown or unhandled season phase: %s", tm.SeasonPhase)
		return result
	}

	if tm.SeasonPhase != "season" || tm.CurrentMatchweek > tm.MaxMatchweeks {
		result.SeasonFinished = tm.SeasonPhase == "transfer_window"
		result.AwardsReady = result.SeasonFinished
		result.Champion = tm.championNameUnlocked()
		result.CalendarLabel = CalendarLabel(tm.SeasonName, tm.MaxMatchweeks)
		if result.SeasonFinished {
			result.Message = "Season complete. The awards ceremony is ready."
		}
		return result
	}

	type progressSig struct {
		SeasonName  string
		SeasonPhase string
		Matchweek   int
	}

	maxIterations := count + 5
	iterations := 0
	for i := 0; i < count && tm.SeasonPhase == "season" && tm.CurrentMatchweek <= tm.MaxMatchweeks; i++ {
		iterations++
		if iterations > maxIterations {
			result.Status = "error"
			result.Message = "Macro simulation iteration limit exceeded without progress"
			return result
		}

		beforeSig := progressSig{
			SeasonName:  tm.SeasonName,
			SeasonPhase: tm.SeasonPhase,
			Matchweek:   tm.CurrentMatchweek,
		}

		d := tm.simulateWeekWithDigestUnlocked()
		result.Digests = append(result.Digests, d)
		result.WeeksSimulated++
		result.WeeksAdvanced++
		result.Played += d.Played
		result.Skipped += d.Skipped

		afterSig := progressSig{
			SeasonName:  tm.SeasonName,
			SeasonPhase: tm.SeasonPhase,
			Matchweek:   tm.CurrentMatchweek,
		}

		if beforeSig == afterSig {
			result.Status = "error"
			result.Message = "Macro simulation stalled: world state failed to make progress"
			return result
		}
	}
	result.EndMatchweek = tm.CurrentMatchweek
	result.CurrentMatchweek = tm.CurrentMatchweek
	result.SeasonPhase = tm.SeasonPhase
	result.SeasonFinished = tm.SeasonPhase == "transfer_window"
	result.AwardsReady = result.SeasonFinished
	if result.SeasonFinished {
		result.Champion = tm.championNameUnlocked()
		result.CalendarLabel = CalendarLabel(tm.SeasonName, tm.MaxMatchweeks)
		result.Message = "Season complete. The awards ceremony is ready."
	} else {
		result.CalendarLabel = CalendarLabel(tm.SeasonName, tm.CurrentMatchweek)
		result.Message = fmt.Sprintf("Advanced %d matchweek(s).", result.WeeksAdvanced)
	}
	return result
}

// AdvanceOffSeasonWeeks advances the autonomous transfer market without
// touching league fixtures. Sim Week/Month use this while the season is over.
func (tm *TournamentManager) AdvanceOffSeasonWeeks(count int) BatchSimResult {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if count < 1 {
		count = 1
	}
	result := BatchSimResult{
		Status: "success", Mode: "batch", SeasonName: tm.SeasonName, SeasonPhase: tm.SeasonPhase,
		CurrentMatchweek: tm.CurrentMatchweek, StartMatchweek: tm.CurrentMatchweek, EndMatchweek: tm.CurrentMatchweek,
		Digests: []MatchweekDigest{}, SeasonFinished: tm.SeasonPhase == "transfer_window",
	}
	if tm.SeasonPhase != "transfer_window" {
		result.Status = "error"
		return result
	}
	if tm.TransferEngine == nil {
		result.OffSeasonComplete = true
		return result
	}
	for i := 0; i < count && tm.TransferEngine.CurrentWeek <= 12; i++ {
		tm.TransferEngine.AdvanceOpenWindow()
		result.OffSeasonWeeksAdvanced++
		result.WeeksAdvanced++
	}
	result.OffSeasonComplete = tm.TransferEngine.CurrentWeek > 12
	result.CalendarLabel = fmt.Sprintf("Off-season · transfer week %d/12", minInt(tm.TransferEngine.CurrentWeek, 12))
	return result
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (tm *TournamentManager) recentLeagueOutcomesUnlocked(clubID string, throughMW, limit int) []string {
	type played struct {
		mw      int
		fixture string
		outcome string
	}
	rows := []played{}
	for i := range tm.Fixtures {
		f := &tm.Fixtures[i]
		if f.Status != "finished" || f.HomeGoals == nil || f.AwayGoals == nil || f.Matchweek > throughMW {
			continue
		}
		if f.HomeID != clubID && f.AwayID != clubID {
			continue
		}
		gf, ga := *f.HomeGoals, *f.AwayGoals
		if f.AwayID == clubID {
			gf, ga = ga, gf
		}
		outcome := "D"
		if gf > ga {
			outcome = "W"
		} else if gf < ga {
			outcome = "L"
		}
		rows = append(rows, played{mw: f.Matchweek, fixture: f.FixtureID, outcome: outcome})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].mw != rows[j].mw {
			return rows[i].mw > rows[j].mw
		}
		return rows[i].fixture > rows[j].fixture
	})
	out := []string{}
	for i := 0; i < len(rows) && i < limit; i++ {
		out = append(out, rows[i].outcome)
	}
	return out
}

func (tm *TournamentManager) managerSecurityUnlocked(club *models.Club, throughMW int) (string, string, bool) {
	if club == nil || throughMW < 8 || club.Played < 6 {
		return "Safe", "Board backing remains firm.", false
	}
	positions := standingsPositions(tm.standingsUnlocked())
	pos := positions[club.ClubID]
	outcomes := tm.recentLeagueOutcomesUnlocked(club.ClubID, throughMW, 6)
	if len(outcomes) == 0 {
		return "Safe", "No recent league sample yet.", false
	}
	points := 0
	winless := 0
	lossStreak := 0
	for i, r := range outcomes {
		switch r {
		case "W":
			points += 3
		case "D":
			points++
		}
		if r != "W" {
			winless++
		}
		if i == lossStreak && r == "L" {
			lossStreak++
		}
	}
	ppg := float64(points) / float64(len(outcomes))
	bottomThree := pos >= len(tm.ClubsList)-2
	eliteCrisis := club.OverallTeamRating >= 84 && pos >= 8 && ppg < 1.0
	resultsCrisis := bottomThree && (lossStreak >= 4 || (len(outcomes) >= 6 && winless >= 6))
	if eliteCrisis || resultsCrisis {
		reason := fmt.Sprintf("%dth place; %.2f PPG over the last %d", pos, ppg, len(outcomes))
		if lossStreak >= 4 {
			reason += fmt.Sprintf("; %d straight league defeats", lossStreak)
		} else if winless >= 6 {
			reason += "; winless in six"
		}
		return "Hot Seat", reason, true
	}
	if pos >= 9 || (club.OverallTeamRating >= 84 && pos >= 7 && ppg < 1.3) || winless >= 4 {
		return "Under Pressure", fmt.Sprintf("%dth place; %.2f PPG over the last %d", pos, ppg, len(outcomes)), false
	}
	return "Safe", fmt.Sprintf("%dth place; %.2f PPG over the last %d", pos, ppg, len(outcomes)), false
}

// ManagerJobSecurity returns the live board assessment shown in commissioner UI.
func (tm *TournamentManager) ManagerJobSecurity(clubID string) (string, string) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	club := tm.Clubs[clubID]
	mw := tm.CurrentMatchweek - 1
	if mw < 1 {
		mw = 1
	}
	status, reason, _ := tm.managerSecurityUnlocked(club, mw)
	return status, reason
}

func (tm *TournamentManager) evaluateManagerTenure(completedMW int) {
	if completedMW < 8 {
		return
	}
	if tm.ManagerConsecutiveHot == nil {
		tm.ManagerConsecutiveHot = map[string]int{}
	}
	if tm.ManagerLastChange == nil {
		tm.ManagerLastChange = map[string]int{}
	}
	for _, club := range tm.ClubsList {
		mgr := tm.Managers[club.ClubID]
		if mgr == nil {
			continue
		}
		status, reason, sackCandidate := tm.managerSecurityUnlocked(club, completedMW)
		mgr.JobSecurity = status
		if !sackCandidate {
			if status == "Safe" {
				tm.ManagerConsecutiveHot[club.ClubID] = 0
			}
			continue
		}
		tm.ManagerConsecutiveHot[club.ClubID]++
		if tm.ManagerConsecutiveHot[club.ClubID] < 2 {
			continue
		}
		if last := tm.ManagerLastChange[club.ClubID]; last > 0 && completedMW-last < 6 {
			continue
		}
		oldStyle := mgr.CanonicalStyle()
		old, next := managers.AppointManager(tm.Managers, club, tm.RNG)
		if old == nil || next == nil {
			continue
		}
		next.History = append([]managers.ManagerHistoryEntry(nil), old.History...)
		next.History = append(next.History, managers.ManagerHistoryEntry{
			ClubID: club.ClubID, ClubName: club.ClubName, ManagerName: old.Name, Style: oldStyle,
			AppointedSeason: old.AppointedSeason, AppointedMatchweek: old.AppointedMatchweek,
			DepartedSeason: tm.SeasonName, DepartedMatchweek: completedMW, Reason: reason,
		})
		next.AppointedSeason = tm.SeasonName
		next.AppointedMatchweek = completedMW + 1
		next.JobSecurity = "Safe"
		if tm.TransferEngine != nil && tm.TransferEngine.Managers != nil {
			tm.TransferEngine.Managers[club.ClubID] = next
		}
		entry := ManagerHistoryEntry{
			SeasonName: tm.SeasonName, Matchweek: completedMW, ClubID: club.ClubID, ClubName: club.ClubName,
			Action: "sacked", OldManager: old.Name, NewManager: next.Name, OldStyle: oldStyle,
			NewStyle: next.CanonicalStyle(), Reason: reason, JobSecurity: "Hot Seat",
		}
		tm.ManagerHistory = append(tm.ManagerHistory, entry)
		tm.ManagerLastChange[club.ClubID] = completedMW
		tm.ManagerConsecutiveHot[club.ClubID] = 0
		tm.PushInbox(
			"manager",
			fmt.Sprintf("BREAKING: %s sack %s; %s appointed", club.ClubName, old.Name, next.Name),
			fmt.Sprintf("%s. %s arrive with a %s identity after the board moved on from %s.", reason, next.Name, next.Tactic(), old.Name),
			completedMW, []string{club.ClubID}, "", "",
		)
	}
}

// GoldenBoyScore is the published commissioner score used by both the
// leaderboard and end-of-season award. Team success is a small bonus so
// individual performance remains the dominant signal.
func (tm *TournamentManager) GoldenBoyScore(p *models.Player) float64 {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.goldenBoyScoreUnlocked(p)
}

func (tm *TournamentManager) goldenBoyScoreUnlocked(p *models.Player) float64 {
	if p == nil || !p.UniverseWonderkid || p.Age > 21 {
		return 0
	}
	pos := 12
	for i, c := range tm.standingsUnlocked() {
		if c.ClubID == p.ClubID {
			pos = i + 1
			break
		}
	}
	teamBonus := float64(maxInt(0, 13-pos)) * 1.5
	trophyBonus := 0.0
	if tm.UCLChampionID == p.ClubID {
		trophyBonus += 12
	}
	if tm.SuperCupChampionID == p.ClubID {
		trophyBonus += 6
	}
	return round2(float64(p.OVR)*2 + float64(p.Goals)*5 + float64(p.Assists)*3 + float64(p.Appearances)*0.75 + teamBonus + trophyBonus)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// matchRowsForWonderkid is shared by digest/leaderboard extensions when a
// caller wants an average match rating without storing another season ledger.
func matchRowsForWonderkid(f *Fixture, pid string) []matchreport.MatchPlayerRow {
	if f == nil || f.Report == nil || pid == "" {
		return nil
	}
	rows := append([]matchreport.MatchPlayerRow{}, f.Report.HomeXI...)
	rows = append(rows, f.Report.HomeBench...)
	rows = append(rows, f.Report.AwayXI...)
	rows = append(rows, f.Report.AwayBench...)
	out := []matchreport.MatchPlayerRow{}
	for _, r := range rows {
		if r.PlayerID == pid && r.Played {
			out = append(out, r)
		}
	}
	return out
}
