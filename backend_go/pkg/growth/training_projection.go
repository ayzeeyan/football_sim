package growth

import (
	"fmt"

	"football_sim/pkg/models"
)

// TrainingProjection is the read-only "what the staff do" plan for one
// player this matchweek. It is a projection only: no state is mutated and
// no randomness is consumed, so calling it never advances the world.
type TrainingProjection struct {
	Focus          string   `json:"focus"`
	Rationale      string   `json:"rationale"`
	ProjectedGains []string `json:"projected_gains"`
	Trainable      bool     `json:"trainable"`
}

// ProjectTrainingFocus returns the regimen the club staff would emphasise
// this week. The choice is deterministic — derived from age, category,
// sharpness, and fitness — so the same player always projects the same plan.
func ProjectTrainingFocus(p *models.Player) string {
	if p == nil {
		return "tactical"
	}
	switch {
	case p.Age <= 18:
		// Physical foundations come first while the body is still developing.
		return "hypertrophy"
	case p.Sharpness < 60:
		// Low sharpness responds fastest to technical ball work.
		return "technical"
	case p.Category == "MID" || p.Category == "DEF":
		return "tactical"
	case p.Category == "GK":
		return "technical"
	default:
		return "technical"
	}
}

// ProjectTrainingWeek builds the staff projection for a player. Prodigies
// (trainable) can additionally be directed through the interactive planner;
// everyone else is staff-managed and shown this plan read-only.
func ProjectTrainingWeek(p *models.Player, trainable bool) TrainingProjection {
	if p == nil {
		return TrainingProjection{Focus: "tactical", Rationale: "Player not found.", Trainable: trainable}
	}
	focus := ProjectTrainingFocus(p)
	proj := TrainingProjection{Focus: focus, Trainable: trainable}
	switch focus {
	case "hypertrophy":
		proj.Rationale = fmt.Sprintf("%s is %d; staff prioritise physical foundations while the body develops.", p.FullName, p.Age)
		proj.ProjectedGains = []string{
			"Strength and stamina conditioning",
			"Lean mass toward the 5 kg lifetime cap",
		}
	case "technical":
		if p.Sharpness < 60 {
			proj.Rationale = fmt.Sprintf("Sharpness is %d; staff rebuild it with technical ball work.", p.Sharpness)
		} else {
			proj.Rationale = "Staff emphasise technical reps: first touch, finishing, and composure."
		}
		proj.ProjectedGains = []string{
			"Technical attribute progression",
			"Sharpness recovery",
		}
	default:
		proj.Rationale = "Staff run position-specific tactical work: scanning, pressing triggers, and shape."
		proj.ProjectedGains = []string{
			"Position XP toward a secondary role",
			"Decision-making under pressure",
		}
	}
	return proj
}
