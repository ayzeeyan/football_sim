package datamanager

import (
	"fmt"
	"math/rand"
	"testing"

	"football_sim/pkg/models"
)

func TestSeasonYouthIntakeStopsAtSustainableRosterTarget(t *testing.T) {
	clubs := make([]*models.Club, 12)
	for i := range clubs {
		club := &models.Club{ClubID: fmt.Sprintf("C%02d", i)}
		start := models.AcademyIntakeSquadTarget
		if i == 0 {
			start = 10
		} else if i == 1 {
			start = models.AcademyIntakeSquadTarget - 1
		}
		for j := 0; j < start; j++ {
			club.Squad = append(club.Squad, &models.Player{
				PlayerID: fmt.Sprintf("P%02d-%02d", i, j),
				FullName: fmt.Sprintf("Existing %02d %02d", i, j),
				ClubID:   club.ClubID,
			})
		}
		clubs[i] = club
	}

	rng := rand.New(rand.NewSource(20260914))
	for season := 0; season < 10; season++ {
		if _, err := RunSeasonYouthIntakeClubs(clubs, nil, rng); err != nil {
			t.Fatalf("season %d intake failed: %v", season+1, err)
		}
		for _, club := range clubs {
			if len(club.Squad) > models.AcademyIntakeSquadTarget {
				t.Fatalf("season %d club %s reached %d players; target is %d", season+1, club.ClubID, len(club.Squad), models.AcademyIntakeSquadTarget)
			}
		}
	}

	for _, club := range clubs {
		if got := len(club.Squad); got != models.AcademyIntakeSquadTarget {
			t.Fatalf("club %s settled at %d players; want %d", club.ClubID, got, models.AcademyIntakeSquadTarget)
		}
	}
}

func TestAcademyQualityMeaningfullyImprovesIntake(t *testing.T) {
	build := func(id string, quality int) *models.Club {
		return &models.Club{
			ClubID: id,
			Identity: models.ClubIdentity{
				AcademyQuality: quality,
			},
		}
	}
	low, high := build("LOW", 10), build("HIGH", 90)
	count := 20
	lowIntake, err := RunYouthIntakeClubs([]*models.Club{low}, &count, nil, rand.New(rand.NewSource(4422)))
	if err != nil {
		t.Fatal(err)
	}
	highIntake, err := RunYouthIntakeClubs([]*models.Club{high}, &count, nil, rand.New(rand.NewSource(4422)))
	if err != nil {
		t.Fatal(err)
	}
	if len(lowIntake) != count || len(highIntake) != count {
		t.Fatalf("intake size low=%d high=%d want %d", len(lowIntake), len(highIntake), count)
	}
	average := func(players []*models.Player) float64 {
		total := 0
		for _, player := range players {
			total += player.OVR
		}
		return float64(total) / float64(len(players))
	}
	if lowAvg, highAvg := average(lowIntake), average(highIntake); highAvg <= lowAvg {
		t.Fatalf("academy quality had no effect: low=%.2f high=%.2f", lowAvg, highAvg)
	}
}
