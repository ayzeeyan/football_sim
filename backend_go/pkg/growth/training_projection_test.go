package growth

import (
	"testing"

	"football_sim/pkg/models"
)

func TestProjectTrainingFocusIsDeterministic(t *testing.T) {
	young := &models.Player{PlayerID: "P1", FullName: "Kid", Age: 17, Category: "FWD", Sharpness: 80}
	gk := &models.Player{PlayerID: "P2", FullName: "Keeper", Age: 30, Category: "GK", Sharpness: 90}
	rusty := &models.Player{PlayerID: "P3", FullName: "Rusty", Age: 28, Category: "FWD", Sharpness: 40}
	mid := &models.Player{PlayerID: "P4", FullName: "Mid", Age: 26, Category: "MID", Sharpness: 85}

	for i := 0; i < 5; i++ {
		if got := ProjectTrainingFocus(young); got != "hypertrophy" {
			t.Fatalf("young player should project hypertrophy, got %q", got)
		}
		if got := ProjectTrainingFocus(gk); got != "technical" {
			t.Fatalf("GK should project technical, got %q", got)
		}
		if got := ProjectTrainingFocus(rusty); got != "technical" {
			t.Fatalf("low sharpness should project technical, got %q", got)
		}
		if got := ProjectTrainingFocus(mid); got != "tactical" {
			t.Fatalf("midfielder should project tactical, got %q", got)
		}
	}
	if got := ProjectTrainingFocus(nil); got != "tactical" {
		t.Fatalf("nil player should default to tactical, got %q", got)
	}
}

func TestProjectTrainingWeekIsPure(t *testing.T) {
	ge := NewGrowthEngine(42)
	p := &models.Player{PlayerID: "P1", FullName: "Kid", Age: 17, Category: "FWD", Sharpness: 80, OVR: 72}
	before := *p

	proj := ProjectTrainingWeek(p, false)
	if proj.Trainable {
		t.Fatal("non-prodigy projection must not be trainable")
	}
	if proj.Focus == "" || proj.Rationale == "" || len(proj.ProjectedGains) == 0 {
		t.Fatalf("incomplete projection: %+v", proj)
	}
	// Purity: the player and the engine state are untouched.
	if p.OVR != before.OVR || p.Sharpness != before.Sharpness || p.Age != before.Age {
		t.Fatal("projection must not mutate the player")
	}
	if ge.TrainingEnergy != ge.MaxTrainingEnergy {
		t.Fatalf("projection must not consume engine energy, got %d", ge.TrainingEnergy)
	}
	// Same inputs, same outputs — no hidden randomness.
	again := ProjectTrainingWeek(p, false)
	if again.Focus != proj.Focus || again.Rationale != proj.Rationale {
		t.Fatal("projection must be deterministic")
	}
	trainable := ProjectTrainingWeek(p, true)
	if !trainable.Trainable {
		t.Fatal("prodigy projection must be trainable")
	}
}
