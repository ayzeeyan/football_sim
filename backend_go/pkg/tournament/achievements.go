package tournament

import (
	"encoding/json"
	"fmt"
	"sort"

	"football_sim/pkg/models"
)

// Achievement is one unlocked entry in the career milestone ledger. The
// ledger is append-only and persisted; UnlockKey deduplicates so a milestone
// can never fire twice for the same subject and season.
type Achievement struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"` // club, player, manager
	Title       string `json:"title"`
	Description string `json:"description"`
	SubjectID   string `json:"subject_id"`
	SubjectName string `json:"subject_name"`
	Season      string `json:"season"`
	Matchweek   int    `json:"matchweek"`
	UnlockKey   string `json:"unlock_key"`
}

// YoungestScorerRecord is the world's youngest goalscorer record. It only
// ever moves down: a goal scored at a younger age breaks it.
type YoungestScorerRecord struct {
	PlayerID   string `json:"player_id"`
	PlayerName string `json:"player_name"`
	ClubID     string `json:"club_id"`
	Age        int    `json:"age"`
	Season     string `json:"season"`
	Matchweek  int    `json:"matchweek"`
}

// achievementDefinitions is the fixed catalogue the UI renders. Order is the
// display order; IDs are stable API contract.
var achievementDefinitions = []Achievement{
	{ID: "unbeaten_20", Kind: "club", Title: "The Unbeaten March", Description: "A club strung together 20 league matches without defeat."},
	{ID: "unbeaten_100", Kind: "club", Title: "The Hundred", Description: "A club went 100 league matches unbeaten — a generational run."},
	{ID: "treble", Kind: "club", Title: "The Treble", Description: "Won the domestic league, the domestic cup, and the Champions League in one season."},
	{ID: "worst_to_champion", Kind: "club", Title: "Worst to Champion", Description: "Followed a bottom-three finish by winning the league the very next season."},
	{ID: "youngest_scorer", Kind: "player", Title: "Youngest Scorer", Description: "Set a new world record as the youngest goalscorer."},
	{ID: "goals_30_season", Kind: "player", Title: "Thirty in a Season", Description: "Scored 30 goals in a single league season."},
	{ID: "sacked_before_season_end", Kind: "manager", Title: "Sacked Before the Curtain", Description: "A manager was dismissed before the season reached its final matchweek."},
}

// AchievementDefinitions returns a copy of the catalogue.
func AchievementDefinitions() []Achievement {
	out := make([]Achievement, len(achievementDefinitions))
	copy(out, achievementDefinitions)
	return out
}

// GetAchievements returns the unlocked ledger, newest season and matchweek
// first, with a stable tiebreak on the unlock key.
func (tm *TournamentManager) GetAchievements() []Achievement {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	out := make([]Achievement, len(tm.Achievements))
	copy(out, tm.Achievements)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Season != out[j].Season {
			return out[i].Season > out[j].Season
		}
		if out[i].Matchweek != out[j].Matchweek {
			return out[i].Matchweek > out[j].Matchweek
		}
		return out[i].UnlockKey < out[j].UnlockKey
	})
	return out
}

// GetYoungestScorerRecord exposes the current record (nil before the first
// qualifying goal).
func (tm *TournamentManager) GetYoungestScorerRecord() *YoungestScorerRecord {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	if tm.YoungestScorer == nil {
		return nil
	}
	rec := *tm.YoungestScorer
	return &rec
}

// unlockAchievementUnlocked appends one achievement to the ledger and pushes
// the matching inbox news. Returns false when the unlock key already fired.
// Caller must hold tm.mu.
func (tm *TournamentManager) unlockAchievementUnlocked(a Achievement) bool {
	if a.UnlockKey == "" {
		return false
	}
	if tm.AchievementsFired == nil {
		tm.AchievementsFired = map[string]bool{}
	}
	if tm.AchievementsFired[a.UnlockKey] {
		return false
	}
	tm.AchievementsFired[a.UnlockKey] = true
	tm.Achievements = append(tm.Achievements, a)
	tm.PushInbox(
		MsgCategoryMilestone,
		fmt.Sprintf("MILESTONE: %s — %s", a.Title, a.SubjectName),
		a.Description,
		a.Matchweek,
		[]string{a.SubjectID},
		"",
		"",
	)
	return true
}

// evaluateWeeklyAchievementsUnlocked runs the per-matchweek achievement
// checks: unbeaten-run bookkeeping, the youngest-scorer record, and the
// 30-goal season. Deterministic: no RNG, slice-order iteration only.
// Caller must hold tm.mu.
func (tm *TournamentManager) evaluateWeeklyAchievementsUnlocked(completedMW int) {
	if tm.ClubUnbeatenRuns == nil {
		tm.ClubUnbeatenRuns = map[string]int{}
	}
	applyRun := func(fixtures []Fixture) {
		for i := range fixtures {
			f := &fixtures[i]
			if f.Matchweek != completedMW || f.Status != "finished" || f.HomeGoals == nil || f.AwayGoals == nil {
				continue
			}
			if !tm.isLeagueCompetition(f.Competition) {
				continue
			}
			hg, ag := *f.HomeGoals, *f.AwayGoals
			switch {
			case hg > ag:
				tm.ClubUnbeatenRuns[f.HomeID]++
				tm.ClubUnbeatenRuns[f.AwayID] = 0
			case ag > hg:
				tm.ClubUnbeatenRuns[f.AwayID]++
				tm.ClubUnbeatenRuns[f.HomeID] = 0
			default:
				tm.ClubUnbeatenRuns[f.HomeID]++
				tm.ClubUnbeatenRuns[f.AwayID]++
			}
		}
	}
	applyRun(tm.Fixtures)
	if tm.World != nil {
		applyRun(tm.World.Fixtures)
	}

	// Unbeaten thresholds, clubs in list order.
	titles := map[int]string{20: "The Unbeaten March", 100: "The Hundred"}
	for _, club := range tm.ClubsList {
		if club == nil {
			continue
		}
		run := tm.ClubUnbeatenRuns[club.ClubID]
		if run != 20 && run != 100 {
			continue
		}
		id := fmt.Sprintf("unbeaten_%d", run)
		tm.unlockAchievementUnlocked(Achievement{
			ID: id, Kind: "club", Title: titles[run],
			Description: fmt.Sprintf("%s reached a %d-match unbeaten league run.", club.ClubName, run),
			SubjectID:   club.ClubID, SubjectName: club.ClubName,
			Season: tm.SeasonName, Matchweek: completedMW,
			UnlockKey: fmt.Sprintf("%s:%s:%d", id, club.ClubID, run),
		})
	}

	// Youngest-scorer record from this week's finished reports.
	scanGoals := func(fixtures []Fixture) {
		for i := range fixtures {
			f := &fixtures[i]
			if f.Matchweek != completedMW || f.Status != "finished" || f.Report == nil {
				continue
			}
			for _, ev := range f.Report.Events {
				if !isGoalEventType(ev.Type) || ev.Scorer == nil || ev.Scorer.Age <= 0 {
					continue
				}
				if tm.YoungestScorer != nil && ev.Scorer.Age >= tm.YoungestScorer.Age {
					continue
				}
				clubID := f.AwayID
				if ev.Side == "home" {
					clubID = f.HomeID
				}
				tm.YoungestScorer = &YoungestScorerRecord{
					PlayerID: ev.Scorer.PlayerID, PlayerName: ev.Scorer.FullName,
					ClubID: clubID, Age: ev.Scorer.Age,
					Season: tm.SeasonName, Matchweek: completedMW,
				}
				tm.unlockAchievementUnlocked(Achievement{
					ID: "youngest_scorer", Kind: "player", Title: "Youngest Scorer",
					Description: fmt.Sprintf("%s became the youngest goalscorer in world history at %d.", ev.Scorer.FullName, ev.Scorer.Age),
					SubjectID:   ev.Scorer.PlayerID, SubjectName: ev.Scorer.FullName,
					Season: tm.SeasonName, Matchweek: completedMW,
					UnlockKey: fmt.Sprintf("youngest_scorer:%s:%d", ev.Scorer.PlayerID, ev.Scorer.Age),
				})
			}
		}
	}
	scanGoals(tm.Fixtures)
	if tm.World != nil {
		scanGoals(tm.World.Fixtures)
	}

	// Thirty goals in a season, clubs in list order.
	for _, club := range tm.ClubsList {
		for _, p := range club.Squad {
			if p != nil && p.Goals >= 30 {
				tm.unlockAchievementUnlocked(Achievement{
					ID: "goals_30_season", Kind: "player", Title: "Thirty in a Season",
					Description: fmt.Sprintf("%s scored 30 goals in the %s league season.", p.FullName, tm.SeasonName),
					SubjectID:   p.PlayerID, SubjectName: p.FullName,
					Season: tm.SeasonName, Matchweek: completedMW,
					UnlockKey: fmt.Sprintf("goals_30_season:%s:%s", p.PlayerID, tm.SeasonName),
				})
			}
		}
	}
}

// unlockSackingAchievementUnlocked records a mid-season dismissal. Sackings
// in the final matchweek are part of the season's closing chapter and do not
// qualify. Caller must hold tm.mu.
func (tm *TournamentManager) unlockSackingAchievementUnlocked(club *models.Club, managerName string, completedMW int) {
	if club == nil || completedMW >= tm.MaxMatchweeks {
		return
	}
	tm.unlockAchievementUnlocked(Achievement{
		ID: "sacked_before_season_end", Kind: "manager", Title: "Sacked Before the Curtain",
		Description: fmt.Sprintf("%s was dismissed by %s with weeks of the season still to play.", managerName, club.ClubName),
		SubjectID:   club.ClubID, SubjectName: managerName,
		Season: tm.SeasonName, Matchweek: completedMW,
		UnlockKey: fmt.Sprintf("sacked_before_season_end:%s:%s:%s", club.ClubID, managerName, tm.SeasonName),
	})
}

// evaluateSeasonAchievementsUnlocked runs the end-of-season checks after the
// season history has been archived: the treble and worst-to-champion.
// Caller must hold tm.mu.
func (tm *TournamentManager) evaluateSeasonAchievementsUnlocked() {
	// Treble: domestic league + domestic cup + Champions League in one
	// season. Domestic cups only exist in world careers, so the treble is a
	// world-only milestone.
	if tm.World != nil {
		for _, club := range tm.ClubsList {
			if club == nil {
				continue
			}
			league, cup, ucl := false, false, false
			for _, def := range domesticLeagueDefinitions {
				if comp := tm.worldCompetitionUnlocked(def.ID); comp != nil && comp.ChampionID == club.ClubID {
					league = true
				}
			}
			for _, def := range domesticCupDefinitions {
				if comp := tm.worldCompetitionUnlocked(def.ID); comp != nil && comp.ChampionID == club.ClubID {
					cup = true
				}
			}
			for _, def := range europeanDefinitions {
				if def.ID != "champions-league" {
					continue
				}
				if comp := tm.worldCompetitionUnlocked(def.ID); comp != nil && comp.ChampionID == club.ClubID {
					ucl = true
				}
			}
			if league && cup && ucl {
				tm.unlockAchievementUnlocked(Achievement{
					ID: "treble", Kind: "club", Title: "The Treble",
					Description: fmt.Sprintf("%s swept their domestic league, domestic cup, and the Champions League in %s.", club.ClubName, tm.SeasonName),
					SubjectID:   club.ClubID, SubjectName: club.ClubName,
					Season: tm.SeasonName, Matchweek: tm.MaxMatchweeks,
					UnlockKey: fmt.Sprintf("treble:%s:%s", club.ClubID, tm.SeasonName),
				})
			}
		}
	}

	// Worst-to-champion: bottom three last season, champion this season.
	for _, club := range tm.ClubsList {
		if club == nil {
			continue
		}
		history := tm.ClubSeasonHistory[club.ClubID]
		if len(history) < 2 {
			continue
		}
		prevPos := historyPosition(history[len(history)-2])
		thisPos := historyPosition(history[len(history)-1])
		if thisPos != 1 || prevPos == 0 {
			continue
		}
		leagueSize := 20
		if id := leagueIDForSelector(club.League); id != "" {
			if comp := tm.worldCompetitionUnlocked(id); comp != nil {
				leagueSize = len(comp.ParticipantIDs)
			}
		}
		if prevPos > leagueSize-3 {
			tm.unlockAchievementUnlocked(Achievement{
				ID: "worst_to_champion", Kind: "club", Title: "Worst to Champion",
				Description: fmt.Sprintf("%s finished %dth one season and won the league the next.", club.ClubName, prevPos),
				SubjectID:   club.ClubID, SubjectName: club.ClubName,
				Season: tm.SeasonName, Matchweek: tm.MaxMatchweeks,
				UnlockKey: fmt.Sprintf("worst_to_champion:%s:%s", club.ClubID, tm.SeasonName),
			})
		}
	}
}

// historyPosition reads a club-season position that survives JSON round
// trips (int in memory, float64 after a save/load cycle).
func historyPosition(row map[string]interface{}) int {
	switch v := row["position"].(type) {
	case int:
		return v
	case float64:
		return int(v)
	case json.Number:
		if n, err := v.Int64(); err == nil {
			return int(n)
		}
	}
	return 0
}

// isLeagueCompetition reports whether a fixture counts toward unbeaten-run
// bookkeeping: domestic leagues and the classic Super League.
func (tm *TournamentManager) isLeagueCompetition(comp string) bool {
	if comp == "super-league" || comp == "" {
		return true
	}
	return tm.isWorldDomesticLeague(comp)
}

// isGoalEventType reports whether a report event type credits the scorer
// with a goal (own goals credit the beneficiary, not the scorer).
func isGoalEventType(t string) bool {
	switch t {
	case "goal", "penalty", "corner_goal", "free_kick_goal":
		return true
	}
	return false
}
