package tournament

import (
	"encoding/json"
	"fmt"
	"math/rand"

	"football_sim/pkg/matchreport"
	"football_sim/pkg/models"
)

// Fixture represents a scheduled or played match.
type Fixture struct {
	FixtureID       string                   `json:"fixture_id"`
	Matchweek       int                      `json:"matchweek"`
	Competition     string                   `json:"competition"` // super-league, ucl, super-cup
	Stage           string                   `json:"stage"`       // league, group, qf, sf, final
	HomeID          string                   `json:"home_id"`
	AwayID          string                   `json:"away_id"`
	Home            *models.Club             `json:"home"`
	Away            *models.Club             `json:"away"`
	Status          string                   `json:"status"` // scheduled, playing, finished
	HomeGoals       *int                     `json:"home_goals"`
	AwayGoals       *int                     `json:"away_goals"`
	Report          *matchreport.MatchReport `json:"report,omitempty"`
	Weather         string                   `json:"weather"`
	DerbyName       string                   `json:"derby_name,omitempty"`
	DerbyHeat       int                      `json:"derby_heat"`
	IsHighHeatDerby bool                     `json:"is_high_heat_derby"`
	Leg             int                      `json:"leg,omitempty"`
	TieID           string                   `json:"tie_id,omitempty"`
	Method          string                   `json:"method,omitempty"`
	Referee         string                   `json:"referee,omitempty"`
	DecidedBy       string                   `json:"decided_by,omitempty"`
	Penalties       []int                    `json:"penalties,omitempty"`
}

// GenerateLeagueFixtures creates a 44-round quadruple round-robin calendar for 12 clubs.
// 4 cycles of 11 rounds:
// Cycle 1: Rounds 1 to 11 (H -> A)
// Cycle 2: Rounds 12 to 22 (A -> H, reversed)
// Cycle 3: Rounds 23 to 33 (H -> A)
// Cycle 4: Rounds 34 to 44 (A -> H, reversed)
// Perfectly balanced: exactly 22 home and 22 away games per club, 6 fixtures per week (264 total).
func GenerateLeagueFixtures(clubs []*models.Club, rng *rand.Rand) []Fixture {
	n := len(clubs)
	if n < 2 {
		return nil
	}

	// Berger tables / polygon round-robin algorithm
	rounds := n - 1 // 11 rounds per full cycle
	var singleCycle [][]Fixture

	clubList := make([]*models.Club, n)
	copy(clubList, clubs)

	for round := 0; round < rounds; round++ {
		var roundFixtures []Fixture
		for i := 0; i < n/2; i++ {
			homeIdx := (round + i) % (n - 1)
			awayIdx := (n - 1 - i + round) % (n - 1)
			if i == 0 {
				awayIdx = n - 1
			}

			home := clubList[homeIdx]
			away := clubList[awayIdx]

			// Alternate home/away based on round
			if (round+i)%2 == 1 {
				home, away = away, home
			}

			derbyName := GetDerbyName(home.ClubID, away.ClubID)
			roundFixtures = append(roundFixtures, Fixture{
				Competition: "super-league",
				Stage:       "league",
				HomeID:      home.ClubID,
				AwayID:      away.ClubID,
				Home:        home,
				Away:        away,
				Status:      "scheduled",
				DerbyName:   derbyName,
				DerbyHeat:   50,
			})
		}
		singleCycle = append(singleCycle, roundFixtures)
	}

	var allFixtures []Fixture
	mw := 1

	// 4 cycles of 11 rounds = 44 matchweeks
	for cycle := 1; cycle <= 4; cycle++ {
		reverseVenues := (cycle%2 == 0)
		for _, round := range singleCycle {
			for _, f := range round {
				homeID := f.HomeID
				awayID := f.AwayID
				home := f.Home
				away := f.Away
				if reverseVenues {
					homeID, awayID = f.AwayID, f.HomeID
					home, away = f.Away, f.Home
				}
				allFixtures = append(allFixtures, Fixture{
					FixtureID:   fmt.Sprintf("MW%d-%s-%s", mw, homeID, awayID),
					Matchweek:   mw,
					Competition: "super-league",
					Stage:       LeaguePhase(mw),
					HomeID:      homeID,
					AwayID:      awayID,
					Home:        home,
					Away:        away,
					Status:      "scheduled",
					Weather:     WeatherOptions[mw%len(WeatherOptions)],
					DerbyName:   GetDerbyName(homeID, awayID),
					DerbyHeat:   50,
				})
			}
			mw++
		}
	}

	return allFixtures
}


type fixtureAlias Fixture

type rawFixtureData struct {
	fixtureAlias
	ID string `json:"id"`
}

// UnmarshalJSON supports both fixture_id and id key representations.
func (f *Fixture) UnmarshalJSON(data []byte) error {
	var raw rawFixtureData
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*f = Fixture(raw.fixtureAlias)
	if f.FixtureID == "" && raw.ID != "" {
		f.FixtureID = raw.ID
	}
	return nil
}
