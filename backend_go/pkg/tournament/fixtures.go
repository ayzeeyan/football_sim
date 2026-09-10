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

// GenerateLeagueFixtures creates a deterministic home-and-away round-robin.
//
// For N clubs each club plays exactly 2*(N-1) league matches and the league
// contains N*(N-1) fixtures in total. Every round contains N/2 fixtures when N
// is even. For an odd-sized league the circle method adds one virtual bye, so
// exactly one club sits out each round without creating a fake fixture.
//
// rng is retained in the signature because fixture generation is part of the
// seeded tournament API, but pairings themselves deliberately do not consume
// randomness: regenerating the same league must produce the same calendar.
func GenerateLeagueFixtures(clubs []*models.Club, rng *rand.Rand) []Fixture {
	_ = rng
	if len(clubs) < 2 {
		return nil
	}

	// Work on a copy so schedule rotation never mutates authoritative club order.
	rotation := append([]*models.Club(nil), clubs...)
	if len(rotation)%2 != 0 {
		rotation = append(rotation, nil) // virtual bye
	}

	slots := len(rotation)
	roundsPerLeg := slots - 1
	firstLeg := make([][]Fixture, 0, roundsPerLeg)

	for round := 0; round < roundsPerLeg; round++ {
		roundFixtures := make([]Fixture, 0, len(clubs)/2)
		for i := 0; i < slots/2; i++ {
			a := rotation[i]
			b := rotation[slots-1-i]
			if a == nil || b == nil {
				continue
			}

			home, away := a, b
		// Alternating the anchor fixture as well as the remaining pairs avoids
		// giving the fixed club an extreme home/away run in the first leg.
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
		firstLeg = append(firstLeg, roundFixtures)

		// Circle method: keep index 0 fixed and rotate every other slot right.
		last := rotation[slots-1]
		copy(rotation[2:], rotation[1:slots-1])
		rotation[1] = last
	}

	allFixtures := make([]Fixture, 0, len(clubs)*(len(clubs)-1))
	matchweek := 1
	for leg := 0; leg < 2; leg++ {
		for _, round := range firstLeg {
			for _, base := range round {
				home, away := base.Home, base.Away
				homeID, awayID := base.HomeID, base.AwayID
				if leg == 1 {
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
