// Package footballai implements FootballMoE: a small hierarchical sparse
// top-2 mixture-of-experts neural network that provides learned predictions
// for the football simulation. The network is trained offline and loaded as
// an immutable .fmoe artifact; it never mutates its weights at runtime.
package footballai

// TaskType identifies which task head a request is routed to. IDs are stable:
// never reorder or reuse values; append new tasks at the end.
type TaskType uint16

const (
	TaskMatchPrediction TaskType = iota
	TaskLineupSelection
	TaskRotation
	TaskInjuryRisk
	TaskDevelopment
	TaskDecline
	TaskValuation
	TaskTransferBid
	TaskNegotiation
	TaskContract
	TaskBoardPatience
	TaskManagerHiring
	TaskSetPiece
	TaskCrowd

	// TaskRouter is a training-only task used to warm-start the MoE router
	// from router_seed.jsonl. It has no task head and no runtime request.
	TaskRouter TaskType = 100

	numRuntimeTasks = 14
)

// TaskName returns the canonical string form of a task.
func TaskName(t TaskType) string {
	switch t {
	case TaskMatchPrediction:
		return "match_prediction"
	case TaskLineupSelection:
		return "lineup_selection"
	case TaskRotation:
		return "rotation"
	case TaskInjuryRisk:
		return "injury_risk"
	case TaskDevelopment:
		return "development"
	case TaskDecline:
		return "decline"
	case TaskValuation:
		return "valuation"
	case TaskTransferBid:
		return "transfer_bid"
	case TaskNegotiation:
		return "negotiation"
	case TaskContract:
		return "contract"
	case TaskBoardPatience:
		return "board_patience"
	case TaskManagerHiring:
		return "manager_hiring"
	case TaskSetPiece:
		return "set_piece"
	case TaskCrowd:
		return "crowd"
	case TaskRouter:
		return "router"
	}
	return "unknown"
}

// TaskFromName resolves the canonical task string used by training data.
func TaskFromName(s string) (TaskType, bool) {
	switch s {
	case "match_prediction":
		return TaskMatchPrediction, true
	case "lineup_selection":
		return TaskLineupSelection, true
	case "rotation":
		return TaskRotation, true
	case "injury_risk":
		return TaskInjuryRisk, true
	case "development":
		return TaskDevelopment, true
	case "decline":
		return TaskDecline, true
	case "valuation":
		return TaskValuation, true
	case "transfer_bid":
		return TaskTransferBid, true
	case "negotiation":
		return TaskNegotiation, true
	case "contract":
		return TaskContract, true
	case "board_patience":
		return TaskBoardPatience, true
	case "manager_hiring":
		return TaskManagerHiring, true
	case "set_piece":
		return TaskSetPiece, true
	case "crowd":
		return TaskCrowd, true
	case "router":
		return TaskRouter, true
	}
	return 0, false
}

// LabelSource records where a training label came from. Real outcomes should
// eventually outweigh bootstrap teacher labels; weights are configurable at
// training time and never baked into the model.
type LabelSource string

const (
	LabelBootstrapTeacher LabelSource = "bootstrap_teacher_v1"
	LabelSimulation       LabelSource = "simulation_outcome"
	LabelHistorical       LabelSource = "historical_result"
	LabelHumanCurated     LabelSource = "human_curated"
)

// Position is the coarse position group used by player-level requests.
type Position uint8

const (
	PosGK Position = iota
	PosDEF
	PosMID
	PosFWD

	NumPositions = 4
)

// PositionFromName maps training-data position groups to stable IDs.
func PositionFromName(s string) (Position, bool) {
	switch s {
	case "GK":
		return PosGK, true
	case "DEF":
		return PosDEF, true
	case "MID":
		return PosMID, true
	case "FWD":
		return PosFWD, true
	}
	return 0, false
}

// Personality is the player personality archetype used in player features.
type Personality uint8

const (
	PersonalityAcademicDual Personality = iota
	PersonalityBigGamePerformer
	PersonalityDedicatedPro
	PersonalityFlamboyantStar
	PersonalitySnake

	NumPersonalities = 5
)

// PersonalityFromName maps training-data personality strings to stable IDs.
func PersonalityFromName(s string) (Personality, bool) {
	switch s {
	case "academic_dual":
		return PersonalityAcademicDual, true
	case "big_game_performer":
		return PersonalityBigGamePerformer, true
	case "dedicated_pro":
		return PersonalityDedicatedPro, true
	case "flamboyant_star":
		return PersonalityFlamboyantStar, true
	case "snake":
		return PersonalitySnake, true
	}
	return 0, false
}

// SquadRole classifies a player's standing within a squad.
type SquadRole uint8

const (
	SquadRoleStar SquadRole = iota
	SquadRoleImportant
	SquadRoleStarter
	SquadRoleRotation
	SquadRoleProspect

	NumSquadRoles = 5
)

// SquadRoleFromName maps training-data squad roles to stable IDs.
func SquadRoleFromName(s string) (SquadRole, bool) {
	switch s {
	case "star":
		return SquadRoleStar, true
	case "important":
		return SquadRoleImportant, true
	case "starter":
		return SquadRoleStarter, true
	case "rotation":
		return SquadRoleRotation, true
	case "prospect":
		return SquadRoleProspect, true
	}
	return 0, false
}

// ExpertID enumerates the four semantic experts. Order matches the router
// output and the manifest expert_order.
type ExpertID int

const (
	ExpertMatch ExpertID = iota
	ExpertPlayer
	ExpertEconomy
	ExpertClub

	NumExperts = 4
)

// ExpertName returns the canonical expert name for debugging output.
func ExpertName(e ExpertID) string {
	switch e {
	case ExpertMatch:
		return "Match"
	case ExpertPlayer:
		return "Player"
	case ExpertEconomy:
		return "Economy"
	case ExpertClub:
		return "Club"
	}
	return "Unknown"
}
