package tournament

import (
	"math/rand"
	"sort"

	"football_sim/pkg/models"
)

// WhatIfScoreline is one resolution of a fixture: the goals, the xG behind
// them, and the knockout decider when the tie required one.
type WhatIfScoreline struct {
	HomeGoals int     `json:"home_goals"`
	AwayGoals int     `json:"away_goals"`
	HomeXG    float64 `json:"home_xg"`
	AwayXG    float64 `json:"away_xg"`
	DecidedBy string  `json:"decided_by,omitempty"`
	Penalties []int   `json:"penalties,omitempty"`
}

// WhatIfClubDelta is one club's league-table position before and after the
// hypothetical result is swapped in.
type WhatIfClubDelta struct {
	ClubID    string `json:"club_id"`
	ShortName string `json:"short_name"`
	BeforePos int    `json:"before_pos"`
	BeforePts int    `json:"before_pts"`
	BeforeGD  int    `json:"before_gd"`
	AfterPos  int    `json:"after_pos"`
	AfterPts  int    `json:"after_pts"`
	AfterGD   int    `json:"after_gd"`
}

// WhatIfTable reports the hypothetical league-table movement. Cup fixtures
// carry no table, so Applicable is false and the deltas are nil.
type WhatIfTable struct {
	Applicable bool             `json:"applicable"`
	Home       *WhatIfClubDelta `json:"home,omitempty"`
	Away       *WhatIfClubDelta `json:"away,omitempty"`
}

// WhatIfResult is the full sandbox answer for one fixture: the recorded
// result (when the fixture has been played) and an alternative resolution
// computed under a scratch seed without touching the real universe.
type WhatIfResult struct {
	FixtureID    string           `json:"fixture_id"`
	Competition  string           `json:"competition"`
	Matchweek    int              `json:"matchweek"`
	ScratchSeed  int64            `json:"scratch_seed"`
	HomeID       string           `json:"home_id"`
	AwayID       string           `json:"away_id"`
	Actual       *WhatIfScoreline `json:"actual,omitempty"`
	Hypothetical WhatIfScoreline  `json:"hypothetical"`
	Table        *WhatIfTable     `json:"table,omitempty"`
}

// WhatIfSandbox resolves a fixture under a scratch seed and reports how the
// alternative scoreline would move the league table. It is strictly
// read-only against the real universe: the compute half of the slate path
// (computeSlateFixture) only reads the fixture and club views, the fixture is
// passed as a shallow copy so even an accidental write cannot land, and no
// apply half ever runs. The same fixtureID + scratchSeed pair always yields
// byte-identical output.
func (tm *TournamentManager) WhatIfSandbox(fixtureID string, scratchSeed int64) (*WhatIfResult, string) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	f := tm.findFixtureUnlocked(fixtureID)
	if f == nil {
		return nil, "Fixture not found."
	}
	home := tm.Clubs[f.HomeID]
	away := tm.Clubs[f.AwayID]
	if home == nil || away == nil {
		return nil, "Club not found."
	}

	sandboxFixture := *f
	rng := rand.New(rand.NewSource(slateSeed(scratchSeed, fixtureID)))
	computed, errMsg := tm.computeSlateFixture(&sandboxFixture, rng)
	if errMsg != "" {
		return nil, errMsg
	}

	result := &WhatIfResult{
		FixtureID:   f.FixtureID,
		Competition: f.Competition,
		Matchweek:   f.Matchweek,
		ScratchSeed: scratchSeed,
		HomeID:      f.HomeID,
		AwayID:      f.AwayID,
		Hypothetical: WhatIfScoreline{
			HomeGoals: computed.assembled.HomeGoals,
			AwayGoals: computed.assembled.AwayGoals,
			HomeXG:    computed.assembled.ShotMap.TotalHomeXG,
			AwayXG:    computed.assembled.ShotMap.TotalAwayXG,
		},
	}
	if computed.assembled.DecidedBy != nil {
		result.Hypothetical.DecidedBy = *computed.assembled.DecidedBy
	}
	if pens, ok := computed.assembled.Penalties.([]int); ok {
		result.Hypothetical.Penalties = pens
	}
	if f.Status == "finished" && f.HomeGoals != nil && f.AwayGoals != nil {
		actual := &WhatIfScoreline{HomeGoals: *f.HomeGoals, AwayGoals: *f.AwayGoals}
		if f.Report != nil {
			actual.HomeXG = f.Report.ShotMap.TotalHomeXG
			actual.AwayXG = f.Report.ShotMap.TotalAwayXG
			if f.Report.DecidedBy != nil {
				actual.DecidedBy = *f.Report.DecidedBy
			}
			if pens, ok := f.Report.Penalties.([]int); ok {
				actual.Penalties = pens
			}
		}
		result.Actual = actual
	}
	result.Table = tm.whatIfTableUnlocked(f, home, away, result.Actual, &result.Hypothetical)
	return result, ""
}

// hypoTableRow is a club's projected table row. Clubs are never mutated: the
// projection sorts a lightweight copy with the same comparator as
// models.SortClubs.
type hypoTableRow struct {
	club             *models.Club
	pts, gd, gf, ovr int
}

// whatIfTableUnlocked computes the hypothetical league-table movement. The
// real table already includes a finished fixture's result, so the swap is
// subtract-actual-then-add-hypothetical; for scheduled fixtures only the
// hypothetical is added.
func (tm *TournamentManager) whatIfTableUnlocked(f *Fixture, home, away *models.Club, actual, hypo *WhatIfScoreline) *WhatIfTable {
	if !tm.isWorldDomesticLeague(f.Competition) && f.Competition != "super-league" && f.Competition != "" {
		return &WhatIfTable{Applicable: false}
	}
	table := tm.standingsForCompetitionUnlocked(f.Competition)
	if len(table) == 0 || home == nil || away == nil {
		return &WhatIfTable{Applicable: false}
	}

	rows := make([]hypoTableRow, 0, len(table))
	for _, c := range table {
		if c == nil {
			continue
		}
		r := hypoTableRow{club: c, pts: c.Points, gd: c.GoalDifference, gf: c.GoalsFor, ovr: c.OverallTeamRating}
		if c.ClubID == home.ClubID || c.ClubID == away.ClubID {
			if actual != nil {
				r = applyWhatIfResult(r, c.ClubID == home.ClubID, actual, -1)
			}
			r = applyWhatIfResult(r, c.ClubID == home.ClubID, hypo, 1)
		}
		rows = append(rows, r)
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].pts != rows[j].pts {
			return rows[i].pts > rows[j].pts
		}
		if rows[i].gd != rows[j].gd {
			return rows[i].gd > rows[j].gd
		}
		if rows[i].gf != rows[j].gf {
			return rows[i].gf > rows[j].gf
		}
		if rows[i].ovr != rows[j].ovr {
			return rows[i].ovr > rows[j].ovr
		}
		return rows[i].club.ClubName < rows[j].club.ClubName
	})

	delta := func(c *models.Club) *WhatIfClubDelta {
		if c == nil {
			return nil
		}
		d := &WhatIfClubDelta{ClubID: c.ClubID, ShortName: c.ShortName, BeforePos: clubPos(table, c.ClubID), BeforePts: c.Points, BeforeGD: c.GoalDifference}
		for i, r := range rows {
			if r.club != nil && r.club.ClubID == c.ClubID {
				d.AfterPos = i + 1
				d.AfterPts = r.pts
				d.AfterGD = r.gd
				break
			}
		}
		return d
	}
	return &WhatIfTable{Applicable: true, Home: delta(home), Away: delta(away)}
}

// applyWhatIfResult adds (dir=1) or removes (dir=-1) one result from a
// projected table row. isHome selects which side of the scoreline the club
// occupies.
func applyWhatIfResult(r hypoTableRow, isHome bool, s *WhatIfScoreline, dir int) hypoTableRow {
	scored, conceded := s.HomeGoals, s.AwayGoals
	if !isHome {
		scored, conceded = s.AwayGoals, s.HomeGoals
	}
	r.gf += dir * scored
	r.gd += dir * (scored - conceded)
	switch {
	case scored > conceded:
		r.pts += dir * 3
	case scored == conceded:
		r.pts += dir * 1
	}
	return r
}
