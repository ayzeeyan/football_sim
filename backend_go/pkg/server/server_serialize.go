package server

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"

	"football_sim/pkg/datamanager"
	"football_sim/pkg/growth"
	"football_sim/pkg/managers"
	"football_sim/pkg/matchreport"
	"football_sim/pkg/models"
	"football_sim/pkg/persistence"
	"football_sim/pkg/tournament"
	"football_sim/pkg/transfers"
)

// Career serialization helpers shared by every endpoint group.
func (s *Server) takeCareerSnapshotLocked() ([]byte, uint64) {
	if s.TransferEngine != nil && s.TournamentManager != nil {
		s.TransferEngine.CurrentMatchweek = s.TournamentManager.CurrentMatchweek
	}
	// Saves are machine-read snapshots. Compact encoding cuts allocation, disk
	// writes and JSON parsing on every simulated slate.
	data, err := json.Marshal(persistence.BuildSnapshot(s.TournamentManager, s.GrowthEngine, s.TransferEngine))
	if err != nil {
		log.Printf("[Save] failed to encode career snapshot: %v", err)
		return nil, 0
	}
	return data, s.saveGen.Add(1)
}

// commitCareerSnapshot writes encoded JSON only if it is still the latest
// generation. Must not be called while holding worldMu or wsMu.
func (s *Server) commitCareerSnapshot(data []byte, gen uint64) {
	if len(data) == 0 || gen == 0 {
		return
	}
	if s.saveGen.Load() != gen {
		return
	}
	s.saveMu.Lock()
	defer s.saveMu.Unlock()
	if s.saveGen.Load() != gen {
		return
	}
	if _, err := persistence.WriteSnapshotBytes(data, s.savePath); err != nil {
		log.Printf("[Save] failed to persist career snapshot: %v", err)
	}
}

// -----------------------------------------------------------------------------
// SERIALIZATION HELPERS
// -----------------------------------------------------------------------------

func (s *Server) serializePlayer(p *models.Player) map[string]interface{} {
	if p == nil {
		return nil
	}

	careerGoals := p.Goals + p.CareerGoals
	careerAssists := p.Assists + p.CareerAssists
	careerApps := p.Appearances + p.CareerApps
	starts, minutes := 0, 0
	for _, st := range p.CompetitionStats {
		if st == nil {
			continue
		}
		starts += st.Starts
		minutes += st.Minutes
	}

	return map[string]interface{}{
		"player_id":           p.PlayerID,
		"full_name":           p.FullName,
		"position":            p.Position,
		"ovr":                 p.OVR,
		"age":                 p.Age,
		"market_value_eur":    p.MarketValueEUR,
		"universe_wonderkid":  p.UniverseWonderkid,
		"player_source":       p.PlayerSource,
		"club_id":             p.ClubID,
		"goals":               p.Goals,
		"assists":             p.Assists,
		"appearances":         p.Appearances,
		"category":            p.Category,
		"formatted_value":     models.FormatCurrency(p.MarketValueEUR),
		"wage_eur":            p.WageEUR,
		"formatted_wage":      models.FormatWage(p.WageEUR),
		"contract_years":      p.ContractYears,
		"loyalty":             p.Loyalty,
		"career_goals":        careerGoals,
		"career_assists":      careerAssists,
		"career_apps":         careerApps,
		"own_goals":           p.OwnGoals,
		"suspended_matches":   p.SuspendedMatches,
		"injured_matches":     p.InjuredMatches,
		"injury":              p.Injury,
		"injury_history":      p.InjuryHistory,
		"availability":        p.AvailabilityNote("super-league", s.TournamentManager.CurrentMatchweek),
		"effective_ovr":       p.EffectiveOVR(),
		"consecutive_starts":  p.ConsecutiveStarts,
		"education":           p.Education,
		"education_label":     p.EducationLabel(),
		"education_pending":   p.EducationPending,
		"school_want":         p.SchoolWant,
		"school_want_label":   p.SchoolWantLine(),
		"school_track":        p.SchoolTrack,
		"school_track_label":  p.SchoolTrackLabel(),
		"secondary_position":  p.SecondaryPosition,
		"position_path":       p.PositionPath,
		"position_xp":         p.PositionXP,
		"position_options":    p.PositionOptions(),
		"best_goals":          p.BestGoals,
		"best_assists":        p.BestAssists,
		"best_season":         p.BestSeason,
		"personality":         p.Personality,
		"personality_title":   p.PersonalityTitle(),
		"personality_badge":   p.PersonalityBadge(),
		"personality_desc":    p.PersonalityInfo().Description,
		"mentor_id":           p.MentorID,
		"mentor_name":         p.MentorName,
		"mentor_ovr":          p.MentorOVR,
		"morale":              p.Morale,
		"morale_band":         p.MoraleBand(),
		"squad_role":          p.SquadRole,
		"fitness":             p.Fitness,
		"sharpness":           p.Sharpness,
		"transfer_requested":  p.TransferRequested,
		"form":                p.FormModifier(),
		"form_band":           p.FormBand(),
		"on_loan":             p.OnLoan,
		"parent_club_id":      p.ParentClubID,
		"loan_buy_clause_eur": p.LoanBuyClauseEUR,
		"formatted_buy_clause": func() interface{} {
			if p.LoanBuyClauseEUR <= 0 {
				return nil
			}
			return models.FormatCurrency(p.LoanBuyClauseEUR)
		}(),
		"competition_stats":   p.CompetitionStats,
		"starts":              starts,
		"minutes":             minutes,
		"is_captain":          p.IsCaptain,
		"is_vice_captain":     p.IsViceCaptain,
		"leadership":          p.Leadership,
		"homegrown":           p.Homegrown,
		"association_trained": p.AssociationTrained,
		"registered_europe":   p.RegisteredEurope,
		"clean_sheets":        p.CleanSheets,
		"career_clean_sheets": p.CareerCleanSheets,
		"versatility":         p.Versatility,
		"promise_kind":        p.PromiseKind,
		"promise_season":      p.PromiseSeason,
	}
}

func (s *Server) serializeClub(c *models.Club) map[string]interface{} {
	return s.serializeClubRecord(c, nil)
}

func (s *Server) serializeClubRecord(c *models.Club, rec *models.CompetitionRecord) map[string]interface{} {
	if c == nil {
		return nil
	}

	mgr := s.TournamentManager.Managers[c.ClubID]
	p, w, d, l := c.Played, c.Won, c.Drawn, c.Lost
	gf, ga, gd, pts := c.GoalsFor, c.GoalsAgainst, c.GoalDifference, c.Points
	form := c.Form
	if rec != nil {
		p, w, d, l = rec.Played, rec.Won, rec.Drawn, rec.Lost
		gf, ga, gd, pts = rec.GoalsFor, rec.GoalsAgainst, rec.GoalDifference, rec.Points
		form = rec.Form
	}
	warchest := c.Finances.TransferBudget
	return map[string]interface{}{
		"club_id":                     c.ClubID,
		"club_name":                   c.ClubName,
		"short_name":                  c.ShortName,
		"league":                      c.League,
		"country":                     c.Country,
		"home_stadium":                c.HomeStadium,
		"stadium_capacity":            c.StadiumCapacity,
		"overall_team_rating":         c.OverallTeamRating,
		"squad_size":                  c.SquadSize,
		"squad_avg_ovr":               c.SquadAvgOVR,
		"primary_color":               c.PrimaryColor,
		"secondary_color":             c.SecondaryColor,
		"p":                           p,
		"w":                           w,
		"d":                           d,
		"l":                           l,
		"gf":                          gf,
		"ga":                          ga,
		"gd":                          gd,
		"pts":                         pts,
		"form":                        form,
		"morale":                      c.Morale,
		"reputation":                  c.Identity.Reputation,
		"budget_eur":                  warchest,
		"transfer_warchest_eur":       warchest,
		"formatted_transfer_warchest": models.FormatCurrency(warchest),
		"identity": map[string]interface{}{
			"reputation":              c.Identity.Reputation,
			"historical_prestige":     c.Identity.HistoricalPrestige,
			"financial_power":         c.Identity.FinancialPower,
			"board_patience":          c.Identity.BoardPatience,
			"academy_quality":         c.Identity.AcademyQuality,
			"recruitment_ambition":    c.Identity.RecruitmentAmbition,
			"youth_preference":        c.Identity.YouthPreference,
			"transfer_aggressiveness": c.Identity.TransferAggressiveness,
			"selling_tendency":        c.Identity.SellingTendency,
		},
		"finances": map[string]interface{}{
			"transfer_budget":  c.Finances.TransferBudget,
			"balance":          c.Finances.Balance,
			"wage_budget":      c.Finances.WageBudget,
			"wage_cap":         c.WageCap(),
			"wage_bill":        c.WageBill(),
			"european_revenue": c.Finances.EuropeanRevenue,
		},
		"coefficient":        c.Coefficient,
		"board_objective":    c.BoardObjective,
		"expected_finish":    c.ExpectedFinish,
		"captain_id":         c.CaptainID,
		"vice_captain_id":    c.ViceCaptainID,
		"fan_expectation":    c.FanExpectation,
		"media_pressure":     c.MediaPressure,
		"chemistry":          c.Chemistry,
		"power_rank":         c.PowerRank,
		"season_attendance":  c.SeasonAttendance,
		"attendance_matches": c.AttendanceMatches,
		"manager":            s.serializeManager(mgr),
	}
}

func (s *Server) serializeManager(m *managers.ManagerProfile) map[string]interface{} {
	if m == nil {
		return nil
	}

	arch := m.ArchetypeInfo()
	return map[string]interface{}{
		"name":                m.Name,
		"tactic":              m.Tactic(),
		"style":               m.Style,
		"manager_style":       managerStyle(m),
		"canonical_style":     m.CanonicalStyle(),
		"dogma_title":         m.DogmaTitle(),
		"description":         arch.Description,
		"line_height":         arch.LineHeight,
		"press_intensity":     arch.PressIntensity,
		"tempo":               arch.Tempo,
		"focus":               m.FocusLabel(),
		"budget_eur":          m.BudgetEur,
		"formatted_budget":    models.FormatCurrency(m.BudgetEur),
		"adaptability":        m.Adaptability,
		"archetype":           m.CanonicalStyle(),
		"archetype_label":     m.DogmaTitle(),
		"job_security":        m.JobSecurity,
		"appointed_season":    m.AppointedSeason,
		"appointed_matchweek": m.AppointedMatchweek,
		"history":             m.History,
	}
}

func (s *Server) serializeFixture(f *tournament.Fixture) map[string]interface{} {
	if f == nil {
		return nil
	}

	home := s.TournamentManager.Clubs[f.HomeID]
	away := s.TournamentManager.Clubs[f.AwayID]

	derby := f.DerbyName
	if derby == "" {
		derby = tournament.GetDerbyName(f.HomeID, f.AwayID)
	}
	heat := f.DerbyHeat
	if derby != "" && heat == 0 {
		heat = 50
		if v, ok := s.TournamentManager.DerbyHeat[derby]; ok {
			heat = v
		}
	}
	homeJSON := s.serializeClub(home)
	awayJSON := s.serializeClub(away)
	var preview map[string]interface{}
	if f.Status == "finished" {
		// The finished-match report already contains its actual lineups and
		// statistics. Rebuilding expected XIs, availability notes, and table
		// positions here only to show the venue wastes work on every report.
		preview = map[string]interface{}{}
		if home != nil {
			preview["venue"] = home.HomeStadium
			preview["capacity"] = home.StadiumCapacity
		}
	} else {
		preview = s.fixturePreview(f, home, away)
	}
	out := map[string]interface{}{
		"id":                 f.FixtureID,
		"fixture_id":         f.FixtureID,
		"matchweek":          f.Matchweek,
		"competition":        f.Competition,
		"stage":              f.Stage,
		"leg":                f.Leg,
		"tie_id":             f.TieID,
		"home_id":            f.HomeID,
		"away_id":            f.AwayID,
		"home":               homeJSON,
		"away":               awayJSON,
		"home_club":          homeJSON,
		"away_club":          awayJSON,
		"status":             f.Status,
		"method":             f.Method,
		"home_goals":         f.HomeGoals,
		"away_goals":         f.AwayGoals,
		"weather":            s.TournamentManager.ResolveWeather(f),
		"derby":              derby,
		"derby_name":         derby,
		"is_derby":           derby != "",
		"derby_heat":         heat,
		"is_high_heat_derby": f.IsHighHeatDerby || heat > 70,
		"referee":            f.Referee,
		"decided_by":         f.DecidedBy,
		"penalties":          f.Penalties,
		"home_manager":       s.serializeManager(s.TournamentManager.Managers[f.HomeID]),
		"away_manager":       s.serializeManager(s.TournamentManager.Managers[f.AwayID]),
		"preview":            preview,
		"head_to_head":       s.TournamentManager.FixtureHeadToHead(f),
		"night":              s.TournamentManager.EuropeanNight(f),
		"events":             []interface{}{},
		"home_xi":            []interface{}{},
		"away_xi":            []interface{}{},
		"home_bench":         []interface{}{},
		"away_bench":         []interface{}{},
	}
	if f.Report != nil {
		out["report"] = f.Report
		out["events"] = f.Report.Events
		out["home_xi"] = f.Report.HomeXI
		out["away_xi"] = f.Report.AwayXI
		out["home_bench"] = f.Report.HomeBench
		out["away_bench"] = f.Report.AwayBench
		out["stats"] = f.Report.Stats
		out["team_stats"] = map[string]interface{}{
			"home": wireTeamStats(f.Report.Stats.Home),
			"away": wireTeamStats(f.Report.Stats.Away),
		}
		out["motm"] = f.Report.MOTM
		out["ht_home"] = f.Report.HTHome
		out["ht_away"] = f.Report.HTAway
		out["attendance"] = f.Report.Attendance
		out["shot_map"] = f.Report.ShotMap
		out["touch_heatmap"] = f.Report.Heatmap
		out["press_conference"] = f.Report.PressConference
		if f.Report.Referee != "" {
			out["referee"] = f.Report.Referee
		}
		if f.Report.DecidedBy != nil {
			out["decided_by"] = *f.Report.DecidedBy
		}
		if f.Report.Penalties != nil {
			out["penalties"] = f.Report.Penalties
		}
	} else if f.ReportSummary != nil {
		// Aged fixture: the full report was archived into a summary.
		out["report_summary"] = f.ReportSummary
		out["ht_home"] = f.ReportSummary.HTHome
		out["ht_away"] = f.ReportSummary.HTAway
		out["attendance"] = f.ReportSummary.Attendance
		if f.ReportSummary.Referee != "" {
			out["referee"] = f.ReportSummary.Referee
		}
		if f.ReportSummary.DecidedBy != "" {
			out["decided_by"] = f.ReportSummary.DecidedBy
		}
	}
	return out
}

// serializeFixtureSummary is used by matchweek lists. It keeps the fields shown
// on fixture cards while leaving player previews and full reports to the
// per-fixture endpoint, where they are loaded only when opened.
func (s *Server) serializeFixtureSummary(f *tournament.Fixture) map[string]interface{} {
	if f == nil {
		return nil
	}

	clubSummary := func(c *models.Club) map[string]interface{} {
		if c == nil {
			return nil
		}
		form := c.Form
		if form == nil {
			form = []string{}
		}
		return map[string]interface{}{
			"club_id":             c.ClubID,
			"club_name":           c.ClubName,
			"short_name":          c.ShortName,
			"league":              c.League,
			"country":             c.Country,
			"home_stadium":        c.HomeStadium,
			"stadium_capacity":    c.StadiumCapacity,
			"overall_team_rating": c.OverallTeamRating,
			"squad_size":          c.SquadSize,
			"squad_avg_ovr":       c.SquadAvgOVR,
			"primary_color":       c.PrimaryColor,
			"secondary_color":     c.SecondaryColor,
			"p":                   c.Played,
			"w":                   c.Won,
			"d":                   c.Drawn,
			"l":                   c.Lost,
			"gf":                  c.GoalsFor,
			"ga":                  c.GoalsAgainst,
			"gd":                  c.GoalDifference,
			"pts":                 c.Points,
			"form":                form,
			"manager":             nil,
		}
	}
	compactPreview := func(home, away *models.Club) interface{} {
		if home == nil || away == nil {
			return nil
		}
		formTail := func(form []string) []string {
			if len(form) > 5 {
				form = form[len(form)-5:]
			}
			if form == nil {
				return []string{}
			}
			return form
		}
		missing := func(club *models.Club) []interface{} {
			rows := make([]interface{}, 0)
			for _, player := range club.Squad {
				if player == nil || !player.IsUnavailable(f.Competition, f.Matchweek) {
					continue
				}
				rows = append(rows, map[string]interface{}{
					"player_id": player.PlayerID,
					"full_name": player.FullName,
				})
			}
			return rows
		}
		return map[string]interface{}{
			"kickoff_note": s.TournamentManager.KickoffNote(f, home, away),
			"home_form":    formTail(home.Form),
			"away_form":    formTail(away.Form),
			"home_missing": missing(home),
			"away_missing": missing(away),
		}
	}

	derby := f.DerbyName
	if derby == "" {
		derby = tournament.GetDerbyName(f.HomeID, f.AwayID)
	}
	heat := f.DerbyHeat
	if derby != "" && heat == 0 {
		heat = 50
		if value, ok := s.TournamentManager.DerbyHeat[derby]; ok {
			heat = value
		}
	}

	events := make([]interface{}, 0)
	var decidedBy interface{} = f.DecidedBy
	var penalties interface{} = f.Penalties
	var htHome interface{}
	var htAway interface{}
	var attendance interface{}
	var motm interface{}
	if f.Report != nil {
		for _, event := range f.Report.Events {
			switch event.Type {
			case "goal", "penalty", "corner_goal", "free_kick_goal", "own_goal", "yellow", "red":
				compactEvent := map[string]interface{}{
					"seq":         event.Seq,
					"minute":      event.Minute,
					"type":        event.Type,
					"side":        event.Side,
					"scorer":      event.Scorer,
					"player":      event.Player,
					"player_id":   event.PlayerID,
					"player_name": event.PlayerName,
					"sent_off":    event.SentOff,
					"disallowed":  event.Disallowed,
				}
				if event.Beneficiary != "" {
					compactEvent["beneficiary"] = event.Beneficiary
				}
				events = append(events, compactEvent)
			}
		}
		htHome = f.Report.HTHome
		htAway = f.Report.HTAway
		attendance = f.Report.Attendance
		if f.Report.MOTM != nil {
			motm = map[string]interface{}{
				"player_id": f.Report.MOTM.PlayerID,
				"full_name": f.Report.MOTM.FullName,
				"rating":    f.Report.MOTM.Rating,
			}
		}
		if f.Report.DecidedBy != nil {
			decidedBy = *f.Report.DecidedBy
		}
		if f.Report.Penalties != nil {
			penalties = f.Report.Penalties
		}
	}

	home := s.TournamentManager.Clubs[f.HomeID]
	away := s.TournamentManager.Clubs[f.AwayID]
	var night interface{}
	if f.Competition == "ucl" || f.Competition == "super-cup" {
		night = s.TournamentManager.EuropeanNight(f)
	}

	return map[string]interface{}{
		"id":                 f.FixtureID,
		"fixture_id":         f.FixtureID,
		"matchweek":          f.Matchweek,
		"competition":        f.Competition,
		"stage":              f.Stage,
		"leg":                f.Leg,
		"tie_id":             f.TieID,
		"home_id":            f.HomeID,
		"away_id":            f.AwayID,
		"home":               clubSummary(home),
		"away":               clubSummary(away),
		"status":             f.Status,
		"method":             f.Method,
		"home_goals":         f.HomeGoals,
		"away_goals":         f.AwayGoals,
		"weather":            s.TournamentManager.ResolveWeather(f),
		"derby":              derby,
		"derby_name":         derby,
		"is_derby":           derby != "",
		"derby_heat":         heat,
		"is_high_heat_derby": f.IsHighHeatDerby || heat > 70,
		"referee":            f.Referee,
		"decided_by":         decidedBy,
		"penalties":          penalties,
		"night":              night,
		"events":             events,
		"home_xi":            []interface{}{},
		"away_xi":            []interface{}{},
		"home_bench":         []interface{}{},
		"away_bench":         []interface{}{},
		"stats":              nil,
		"motm":               motm,
		"ht_home":            htHome,
		"ht_away":            htAway,
		"attendance":         attendance,
		"preview":            compactPreview(home, away),
		"head_to_head":       []interface{}{},
	}
}

func (s *Server) clubFixtureContext(clubID string) string {
	mw := 1
	if s.TournamentManager != nil && s.TournamentManager.CurrentMatchweek > 0 {
		mw = s.TournamentManager.CurrentMatchweek
	}
	if s.TournamentManager != nil {
		for _, f := range s.TournamentManager.GetSlate(mw) {
			if f.HomeID == clubID || f.AwayID == clubID {
				return models.FixtureContext(f.Competition, f.Matchweek)
			}
		}
	}
	return models.FixtureContext("super-league", mw)
}

func (s *Server) fixturePreview(f *tournament.Fixture, home, away *models.Club) map[string]interface{} {
	if f == nil || home == nil || away == nil || s.TournamentManager == nil {
		return nil
	}
	fx := models.FixtureContext(f.Competition, f.Matchweek)
	table := s.TournamentManager.GetStandings()
	pos := map[string]int{}
	for i, c := range table {
		pos[c.ClubID] = i + 1
	}
	serializeXI := func(players []*models.Player) []map[string]interface{} {
		out := make([]map[string]interface{}, 0, len(players))
		for _, p := range players {
			if p == nil {
				continue
			}
			row := s.serializePlayer(p)
			row["availability"] = p.AvailabilityNote(f.Competition, f.Matchweek)
			if note := s.grewNoteFor(p); note != "" {
				row["grew_note"] = note
			}
			out = append(out, row)
		}
		return out
	}
	serializeSlottedXI := func(slots []models.StartingSlot) []map[string]interface{} {
		out := make([]map[string]interface{}, 0, len(slots))
		for _, entry := range slots {
			p := entry.Player
			if p == nil {
				continue
			}
			row := s.serializePlayer(p)
			row["starting_slot"] = entry.Slot
			row["tactical_slot"] = entry.Slot
			natural := entry.NaturalPosition
			if natural == "" {
				natural = p.Position
			}
			row["natural_position"] = natural
			fit := entry.PositionFit
			if fit == "" {
				fit = models.PositionFitForPlayer(p, entry.Slot)
			}
			row["position_fit"] = string(fit)
			row["availability"] = p.AvailabilityNote(f.Competition, f.Matchweek)
			if note := s.grewNoteFor(p); note != "" {
				row["grew_note"] = note
			}
			out = append(out, row)
		}
		return out
	}
	missing := func(club *models.Club) []map[string]interface{} {
		out := make([]map[string]interface{}, 0)
		for _, p := range club.Squad {
			if p == nil || !p.IsUnavailable(f.Competition, f.Matchweek) {
				continue
			}
			row := s.serializePlayer(p)
			row["availability"] = p.AvailabilityNote(f.Competition, f.Matchweek)
			out = append(out, row)
		}
		return out
	}
	xiAvg := func(xi []*models.Player) int {
		if len(xi) == 0 {
			return 0
		}
		sum := 0
		n := 0
		for _, p := range xi {
			if p == nil {
				continue
			}
			sum += p.OVR
			n++
		}
		if n == 0 {
			return 0
		}
		return int(math.Round(float64(sum) / float64(n)))
	}
	homeStyle, homeFocus, awayStyle, awayFocus := "", "", "", ""
	if mgr := s.TournamentManager.Managers[home.ClubID]; mgr != nil {
		homeStyle, homeFocus = mgr.Style, mgr.Focus
	}
	if mgr := s.TournamentManager.Managers[away.ClubID]; mgr != nil {
		awayStyle, awayFocus = mgr.Style, mgr.Focus
	}
	homeSlots := home.GetStartingElevenSlotsWithBias(homeStyle, homeFocus, fx)
	awaySlots := away.GetStartingElevenSlotsWithBias(awayStyle, awayFocus, fx)
	homeXI := make([]*models.Player, 0, len(homeSlots))
	awayXI := make([]*models.Player, 0, len(awaySlots))
	for _, entry := range homeSlots {
		if entry.Player != nil {
			homeXI = append(homeXI, entry.Player)
		}
	}
	for _, entry := range awaySlots {
		if entry.Player != nil {
			awayXI = append(awayXI, entry.Player)
		}
	}
	homeForm := home.Form
	if len(homeForm) > 5 {
		homeForm = homeForm[len(homeForm)-5:]
	}
	awayForm := away.Form
	if len(awayForm) > 5 {
		awayForm = awayForm[len(awayForm)-5:]
	}
	homePos, awayPos := interface{}(nil), interface{}(nil)
	if p := pos[home.ClubID]; p > 0 {
		homePos = p
	}
	if p := pos[away.ClubID]; p > 0 {
		awayPos = p
	}
	return map[string]interface{}{
		"kickoff_note":    s.TournamentManager.KickoffNote(f, home, away),
		"venue":           home.HomeStadium,
		"capacity":        home.StadiumCapacity,
		"home_formation":  models.FormationForManager(homeStyle, home.ClubID),
		"away_formation":  models.FormationForManager(awayStyle, away.ClubID),
		"home_form":       homeForm,
		"away_form":       awayForm,
		"home_pos":        homePos,
		"away_pos":        awayPos,
		"home_pts":        home.Points,
		"away_pts":        away.Points,
		"home_xi":         serializeSlottedXI(homeSlots),
		"away_xi":         serializeSlottedXI(awaySlots),
		"home_bench":      serializeXI(home.GetBench(homeXI, 7, fx)),
		"away_bench":      serializeXI(away.GetBench(awayXI, 7, fx)),
		"home_missing":    missing(home),
		"away_missing":    missing(away),
		"home_xi_avg":     xiAvg(homeXI),
		"away_xi_avg":     xiAvg(awayXI),
		"home_key_player": s.serializePlayer(clubKeyPlayer(home)),
		"away_key_player": s.serializePlayer(clubKeyPlayer(away)),
	}
}

// grewNoteFor returns the pre-match XI one-liner when a prodigy has grown
// since the August baseline, or "" otherwise. Read-only: puberty curves and
// composure are untouched.
func (s *Server) grewNoteFor(p *models.Player) string {
	if p == nil || !p.UniverseWonderkid || s.GrowthEngine == nil {
		return ""
	}
	data, ok := s.GrowthEngine.GetProdigyData(p.PlayerID, p.Category)
	if !ok {
		return ""
	}
	gain, ok := data["height_gain_cm"].(float64)
	if !ok || gain < 0.1 {
		return ""
	}
	return fmt.Sprintf("+%.1f cm since August ΓÇö growing into the shirt.", gain)
}

func (s *Server) findLiveFixture(homeID, awayID string) *tournament.Fixture {
	mw := s.TournamentManager.CurrentMatchweek
	var match *tournament.Fixture
	for _, f := range s.TournamentManager.GetSlate(mw) {
		if f.HomeID == homeID && f.AwayID == awayID && f.Status == "scheduled" {
			// Club pairs can meet in more than one competition in the same
			// matchweek. Without a fixture ID, do not guess which result the
			// viewer intended to commit.
			if match != nil {
				return nil
			}
			cp := f
			match = &cp
		}
	}
	return match
}

func (s *Server) clearLiveFixtureSelection() {
	s.liveFixtureID = ""
	if s.LiveMatchEngine != nil {
		s.LiveMatchEngine.ClearFixtureContext()
	}
}

func copyStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func (s *Server) prodigyDrawPayload(homes map[string]string) map[string]interface{} {
	if len(homes) == 0 {
		homes = datamanager.DefaultProdigyHomes()
	}
	return map[string]interface{}{
		"homes": copyStringMap(homes),
		"draw":  s.DataManager.DescribeProdigyDraw(homes),
	}
}

func resolveNewCareerHomes(shuffle bool, homes map[string]string, rng *rand.Rand) map[string]string {
	if len(homes) > 0 {
		return copyStringMap(homes)
	}
	if shuffle {
		return datamanager.ShuffleProdigyHomes(rng)
	}
	return datamanager.DefaultProdigyHomes()
}

func (s *Server) resetLiveMatchDefault() {
	s.lastCommittedLiveInstance = -1
	s.liveFixtureID = ""
	s.wsForceFull = true
	s.wsLastHomeScore = 0
	s.wsLastAwayScore = 0
	s.wsLastEvents = 0
	if s.LiveMatchEngine == nil || s.TournamentManager == nil {
		return
	}
	home := s.TournamentManager.Clubs["LAL-RMA"]
	away := s.TournamentManager.Clubs["EPL-ARS"]
	if home == nil || away == nil {
		if len(s.TournamentManager.ClubsList) >= 2 {
			home = s.TournamentManager.ClubsList[0]
			away = s.TournamentManager.ClubsList[1]
		}
	}
	if home == nil || away == nil {
		s.LiveMatchEngine.ClearFixtureContext()
		return
	}
	s.LiveMatchEngine.SetClubs(home, away, s.TournamentManager.Managers[home.ClubID], s.TournamentManager.Managers[away.ClubID])
}

// bootFreshCareer rebuilds the universe from one explicit seed so identical
// seeds deal identical careers. Every subsystem draws from an independent
// SubsystemRNG stream, mirroring cmd/server boot.
func (s *Server) bootFreshCareer(homes map[string]string, shuffle bool, seed int64) error {
	datasetPath := ""
	if s.DataManager != nil {
		datasetPath = s.DataManager.JSONPath
	}
	previousDefault := growth.GetDefaultGrowthEngine()
	rng := tournament.NewSubsystemRNG(seed)
	ge := growth.NewGrowthEngine(rng.SeedFor("development"))
	dm := datamanager.NewDataManager(datasetPath, ge)
	if dm == nil || len(dm.ClubsList) < 2 {
		if previousDefault != nil {
			growth.SetDefaultGrowthEngine(previousDefault)
		}
		return errFreshCareer
	}
	dm.SetSeed(rng.SeedFor("datamanager"))
	dm.ApplyProdigyHomes(homes)
	tm := tournament.NewEuropeanWorldManager(dm.ClubsList, ge, rng.SeedFor("matches"))
	te := transfers.NewTransferEngine(dm.ClubsList, tm.Managers, rng.SeedFor("transfers"))
	tm.TransferEngine = te
	tm.ProdigyHomes = copyStringMap(dm.ProdigyHomes)
	tm.LastCareerShuffle = shuffle

	s.GrowthEngine = ge
	s.DataManager = dm
	s.TournamentManager = tm
	s.TransferEngine = te
	s.resetLiveMatchDefault()
	if s.LiveMatchEngine != nil {
		s.LiveMatchEngine.RNG = rng.New("live_match")
	}
	return nil
}

func (s *Server) findCurrentLeagueFixture(homeID, awayID string) *tournament.Fixture {
	if s.TournamentManager == nil || s.TournamentManager.CurrentMatchweek > s.TournamentManager.MaxMatchweeks {
		return nil
	}
	for _, f := range s.TournamentManager.GetMatchweekFixtures(s.TournamentManager.CurrentMatchweek) {
		if f.HomeID == homeID && f.AwayID == awayID {
			cp := f
			return &cp
		}
	}
	return nil
}

func (s *Server) findPlayer(playerID string) (*models.Player, *models.Club) {
	for _, club := range s.TournamentManager.ClubsList {
		for _, p := range club.Squad {
			if p.PlayerID == playerID {
				return p, club
			}
		}
	}
	return nil, nil
}

// managerStyle renders a manager's tactical identity as a single label,
// preferring the explicit style and falling back to the canonical archetype.
func managerStyle(m *managers.ManagerProfile) string {
	if m == nil {
		return ""
	}
	if m.Style != "" {
		return m.Style
	}
	return m.CanonicalStyle()
}

// wireTeamStats adapts aggregate team stats for the JSON wire.
func wireTeamStats(ts matchreport.TeamStats) map[string]interface{} {
	return map[string]interface{}{
		"possession":      ts.Possession,
		"shots":           ts.Shots,
		"shots_on_target": ts.ShotsOn,
		"xg":              ts.XG,
		"passes":          ts.Passes,
		"pass_accuracy":   ts.PassAccuracy,
		"corners":         ts.Corners,
		"fouls":           ts.Fouls,
		"yellow_cards":    ts.YellowCards,
		"red_cards":       ts.RedCards,
		"saves":           ts.Saves,
		"big_chances":     ts.BigChances,
	}
}

// clubKeyPlayer returns the highest-rated player in a club squad, or nil when
// the squad is empty. Used for pre-match key-player billing.
func clubKeyPlayer(club *models.Club) *models.Player {
	if club == nil {
		return nil
	}
	var best *models.Player
	for _, p := range club.Squad {
		if p == nil {
			continue
		}
		if best == nil || p.OVR > best.OVR {
			best = p
		}
	}
	return best
}
