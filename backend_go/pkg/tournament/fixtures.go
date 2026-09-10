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

// GenerateLeagueFixtures creates four balanced round-robin cycles.
//
// In the canonical 12-club Super League, every club plays every opponent four
// times: twice home and twice away. That yields 44 league matches per club,
// 44 matchweeks, 6 fixtures per matchweek, and 264 league fixtures overall.
// The same circle method also supports odd-sized test leagues via a virtual bye.
// Pairings are deterministic and do not consume the supplied RNG.
func GenerateLeagueFixtures(clubs []*models.Club, rng *rand.Rand) []Fixture {
	_ = rng
	if len(clubs) < 2 {
		return nil
	}

	rotation := append([]*models.Club(nil), clubs...)
	if len(rotation)%2 != 0 {
		rotation = append(rotation, nil)
	}

	slots := len(rotation)
	roundsPerCycle := slots - 1
	baseRounds := make([][]Fixture, 0, roundsPerCycle)

	for round := 0; round < roundsPerCycle; round++ {
		roundFixtures := make([]Fixture, 0, len(clubs)/2)
		for i := 0; i < slots/2; i++ {
			a := rotation[i]
			b := rotation[slots-1-i]
			if a == nil || b == nil {
				continue
			}
			home, away := a, b
			if (round+i)%2 != 0 {
				home, away = away, home
			}
			roundFixtures = append(roundFixtures, Fixture{
				Competition: "super-league",
				Stage:       "league",
				HomeID:      home.ClubID,
				AwayID:      away.ClubID,
				Home:        home,
				Away:        away,
				Status:      "scheduled",
				DerbyName:   GetDerbyName(home.ClubID, away.ClubID),
				DerbyHeat:   50,
			})
		}
		baseRounds = append(baseRounds, roundFixtures)

		last := rotation[slots-1]
		copy(rotation[2:], rotation[1:slots-1])
		rotation[1] = last
	}

	fixturesPerCycle := len(clubs) * (len(clubs) - 1) / 2
	allFixtures := make([]Fixture, 0, fixturesPerCycle*4)
	matchweek := 1
	for cycle := 0; cycle < 4; cycle++ {
		reverseVenues := cycle%2 == 1
		for _, round := range baseRounds {
			for _, base := range round {
				home, away := base.Home, base.Away
				homeID, awayID := base.HomeID, base.AwayID
				if reverseVenues {
					home, away = base.Away, base.Home
					homeID, awayID = base.AwayID, base.HomeID
				}
				allFixtures = append(allFixtures, Fixture{
					FixtureID:   fmt.Sprintf("MW%d-%s-%s", matchweek, homeID, awayID),
					Matchweek:   matchweek,
					Competition: "super-league",
					Stage:       LeaguePhase(matchweek),
					HomeID:      homeID,
					AwayID:      awayID,
					Home:        home,
					Away:        away,
					Status:      "scheduled",
					Weather:     WeatherOptions[(matchweek-1)%len(WeatherOptions)],
					DerbyName:   GetDerbyName(homeID, awayID),
					DerbyHeat:   50,
				})
			}
			matchweek++
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
