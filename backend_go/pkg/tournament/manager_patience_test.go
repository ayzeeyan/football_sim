package tournament

import (
	"testing"

	"football_sim/pkg/models"
)

func TestBoardPatienceChangesHotSeatPersistence(t *testing.T) {
	impatient := &models.Club{Identity: models.ClubIdentity{BoardPatience: 20}}
	balanced := &models.Club{Identity: models.ClubIdentity{BoardPatience: 55}}
	patient := &models.Club{Identity: models.ClubIdentity{BoardPatience: 90}}
	if got := boardHotSeatWeeks(impatient); got != 1 { t.Fatalf("impatient board requires %d hot weeks, want 1", got) }
	if got := boardHotSeatWeeks(balanced); got != 2 { t.Fatalf("balanced board requires %d hot weeks, want 2", got) }
	if got := boardHotSeatWeeks(patient); got != 3 { t.Fatalf("patient board requires %d hot weeks, want 3", got) }
}

func TestBoardPatienceEffectIsBounded(t *testing.T) {
	for patience := 0; patience <= 100; patience++ {
		got := boardHotSeatWeeks(&models.Club{Identity: models.ClubIdentity{BoardPatience: patience}})
		if got < 1 || got > 3 { t.Fatalf("patience=%d produced unbounded persistence %d", patience, got) }
	}
}
