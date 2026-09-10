package models

import (
	"sort"
	"strings"
)

// CompetitionRecord tracks a team's standings performance in a specific competition.
type CompetitionRecord struct {
	Played         int      `json:"p"`
	Won            int      `json:"w"`
	Drawn          int      `json:"d"`
	Lost           int      `json:"l"`
	GoalsFor       int      `json:"gf"`
	GoalsAgainst   int      `json:"ga"`
	GoalDifference int      `json:"gd"`
	Points         int      `json:"pts"`
	Form           []string `json:"form"`
}

// UpdateResult updates record tallies and form history with the given match score.
func (cr *CompetitionRecord) UpdateResult(gf, ga int) {
	cr.Played++
	cr.GoalsFor += gf
	cr.GoalsAgainst += ga
	cr.GoalDifference = cr.GoalsFor - cr.GoalsAgainst

	if gf > ga {
		cr.Won++
		cr.Points += 3
		cr.Form = append(cr.Form, "W")
	} else if gf == ga {
		cr.Drawn++
		cr.Points += 1
		cr.Form = append(cr.Form, "D")
	} else {
		cr.Lost++
		cr.Form = append(cr.Form, "L")
	}

	if len(cr.Form) > 5 {
		cr.Form = cr.Form[len(cr.Form)-5:]
	}
}

// Reset resets all record statistics and clears form history.
func (cr *CompetitionRecord) Reset() {
	cr.Played = 0
	cr.Won = 0
	cr.Drawn = 0
	cr.Lost = 0
	cr.GoalsFor = 0
	cr.GoalsAgainst = 0
	cr.GoalDifference = 0
	cr.Points = 0
	cr.Form = []string{}
}

// StandingsRow represents a single team's row in a formatted league table.
type StandingsRow struct {
	Position       int      `json:"position"`
	ClubID         string   `json:"club_id"`
	ClubName       string   `json:"club_name"`
	ShortName      string   `json:"short_name"`
	Played         int      `json:"played"`
	Won            int      `json:"won"`
	Drawn          int      `json:"drawn"`
	Lost           int      `json:"lost"`
	GoalsFor       int      `json:"goals_for"`
	GoalsAgainst   int      `json:"goals_against"`
	GoalDifference int      `json:"goal_difference"`
	Points         int      `json:"points"`
	TeamRating     int      `json:"team_rating"`
	Form           []string `json:"form"`
}

// StandingsTable is a slice of StandingsRow supporting deterministic sorting.
type StandingsTable []StandingsRow

func (st StandingsTable) Len() int {
	return len(st)
}

func (st StandingsTable) Swap(i, j int) {
	st[i], st[j] = st[j], st[i]
}

// Less sorts by: Points desc > GD desc > GF desc > TeamRating desc > ClubName asc.
func (st StandingsTable) Less(i, j int) bool {
	if st[i].Points != st[j].Points {
		return st[i].Points > st[j].Points
	}
	if st[i].GoalDifference != st[j].GoalDifference {
		return st[i].GoalDifference > st[j].GoalDifference
	}
	if st[i].GoalsFor != st[j].GoalsFor {
		return st[i].GoalsFor > st[j].GoalsFor
	}
	if st[i].TeamRating != st[j].TeamRating {
		return st[i].TeamRating > st[j].TeamRating
	}
	return strings.ToLower(st[i].ClubName) < strings.ToLower(st[j].ClubName)
}

// SortStandings sorts the table in-place and renumbers the Position field 1..N.
func SortStandings(table StandingsTable) {
	sort.Sort(table)
	for i := range table {
		table[i].Position = i + 1
	}
}

// SortClubs sorts a slice of clubs by Points desc > GD desc > GF desc > OverallTeamRating desc > ClubName asc.
func SortClubs(clubs []*Club) {
	sort.SliceStable(clubs, func(i, j int) bool {
		if clubs[i].Points != clubs[j].Points {
			return clubs[i].Points > clubs[j].Points
		}
		if clubs[i].GoalDifference != clubs[j].GoalDifference {
			return clubs[i].GoalDifference > clubs[j].GoalDifference
		}
		if clubs[i].GoalsFor != clubs[j].GoalsFor {
			return clubs[i].GoalsFor > clubs[j].GoalsFor
		}
		if clubs[i].OverallTeamRating != clubs[j].OverallTeamRating {
			return clubs[i].OverallTeamRating > clubs[j].OverallTeamRating
		}
		return strings.ToLower(clubs[i].ClubName) < strings.ToLower(clubs[j].ClubName)
	})
}
