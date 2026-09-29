package training

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"

	"football_sim/pkg/footballai"
)

// Split names used by the bootstrap datasets. Membership is fixed by the
// data files; the loader never reshuffles it.
const (
	SplitTrain = "train"
	SplitVal   = "val"
	SplitTest  = "test"
)

// Sample is one converted training example in the shared internal
// representation: raw input slots plus task-aligned targets.
type Sample struct {
	// Task is the head task. For router examples it is the event's task
	// (used for the task embedding); IsRouter marks the router loss.
	Task Task
	// IsRouter marks router_seed examples: they carry a teacher expert
	// distribution and no head target.
	IsRouter bool
	// Slots is the raw (unnormalized) input vector.
	Slots []float32
	// Target is aligned with the task spec outputs (already transformed for
	// log-space outputs).
	Target []float32
	// Teacher is the router teacher distribution over the four experts.
	Teacher []float32
	// Weight is the label-source sample weight.
	Weight float32
	// Split, GroupID and LabelSource are dataset metadata used for
	// filtering and leakage control.
	Split       string
	GroupID     int64
	LabelSource footballai.LabelSource

	// Auxiliary observed outcomes for evaluation metrics (Brier scores,
	// result calibration). They are never used as training inputs.
	AuxGoalDiff float32 // actual goal difference (match rows)
	AuxResult   int8    // 1 home win, 0 draw, -1 away win (match rows)
	AuxInjury   int8    // 1 injury occurred, 0 not, -1 unknown (injury rows)
}

// Task re-exports the task type to keep the training API self-documenting.
type Task = footballai.TaskType

// DatasetFiles lists the bootstrap dataset files in the data directory.
var DatasetFiles = []string{
	"router_seed.jsonl",
	"match_seed.jsonl",
	"player_seed.jsonl",
	"economy_club_seed.jsonl",
}

// rawRow is the on-disk JSONL schema. Feature/target values are kept raw so
// numeric, boolean, and string fields can be decoded per name.
type rawRow struct {
	SchemaVersion int                        `json:"schema_version"`
	LabelSource   string                     `json:"label_source"`
	Task          string                     `json:"task"`
	EventType     string                     `json:"event_type"`
	GroupID       int64                      `json:"group_id"`
	Split         string                     `json:"split"`
	Features      map[string]json.RawMessage `json:"features"`
	Target        map[string]json.RawMessage `json:"target"`
}

type rawRouterTarget struct {
	ExpertWeights map[string]float64 `json:"expert_weights"`
	Top2          []string           `json:"top2"`
}

// DatasetReader streams one JSONL file row by row.
type DatasetReader struct {
	file    *os.File
	scanner *bufio.Scanner
	path    string
	scratch *sampleBuilder
}

// OpenDataset opens a JSONL dataset for streaming.
func OpenDataset(path string) (*DatasetReader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("training: open %s: %w", path, err)
	}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	return &DatasetReader{file: f, scanner: sc, path: path, scratch: newSampleBuilder()}, nil
}

// Close releases the underlying file.
func (r *DatasetReader) Close() error { return r.file.Close() }

// Next decodes the next row into a Sample. It returns io.EOF at end of file.
func (r *DatasetReader) Next() (*Sample, error) {
	for {
		if !r.scanner.Scan() {
			if err := r.scanner.Err(); err != nil {
				return nil, fmt.Errorf("training: scan %s: %w", r.path, err)
			}
			return nil, io.EOF
		}
		line := r.scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var row rawRow
		if err := json.Unmarshal(line, &row); err != nil {
			return nil, fmt.Errorf("training: bad JSONL row in %s: %w", r.path, err)
		}
		s, err := r.scratch.convertRow(row)
		if err != nil {
			return nil, fmt.Errorf("training: convert row in %s: %w", r.path, err)
		}
		return s, nil
	}
}

// ---------------------------------------------------------------------------
// Row → Sample conversion. This is the single place that maps training-data
// JSONL field names onto the fixed input slot layout; the game runtime uses
// the strongly typed request structs instead.
// ---------------------------------------------------------------------------

type sampleBuilder struct {
	features map[string]json.RawMessage
	target   map[string]json.RawMessage
}

func newSampleBuilder() *sampleBuilder {
	return &sampleBuilder{}
}

func num(m map[string]json.RawMessage, key string) (float64, bool) {
	raw, ok := m[key]
	if !ok {
		return 0, false
	}
	var v float64
	if err := json.Unmarshal(raw, &v); err != nil {
		return 0, false
	}
	return v, true
}

func boolean(m map[string]json.RawMessage, key string) (bool, bool) {
	raw, ok := m[key]
	if !ok {
		return false, false
	}
	var v bool
	if err := json.Unmarshal(raw, &v); err != nil {
		return false, false
	}
	return v, true
}

func text(m map[string]json.RawMessage, key string) (string, bool) {
	raw, ok := m[key]
	if !ok {
		return "", false
	}
	var v string
	if err := json.Unmarshal(raw, &v); err != nil {
		return "", false
	}
	return v, true
}

// fillPlayer writes player group slots from dataset feature names.
func fillPlayer(f map[string]json.RawMessage, slots []float32) {
	set := func(slot int, name string) {
		if v, ok := num(f, name); ok {
			slots[slot] = float32(v)
		}
	}
	set(footballai.SlotAge, "age")
	set(footballai.SlotOVR, "ovr")
	set(footballai.SlotPotential, "potential")
	set(footballai.SlotFitness, "fitness")
	set(footballai.SlotSharpness, "sharpness")
	set(footballai.SlotFatigue, "fatigue")
	set(footballai.SlotConsecutiveStarts, "consecutive_starts")
	set(footballai.SlotMinutes14d, "minutes_last_14_days")
	set(footballai.SlotMorale, "morale")
	set(footballai.SlotFormModifier, "form_modifier")
	if v, ok := boolean(f, "recent_injury"); ok && v {
		slots[footballai.SlotRecentInjury] = 1
	}
	set(footballai.SlotPlayerImportance, "match_importance")
	set(footballai.SlotReplacementOVR, "replacement_ovr")
	set(footballai.SlotAcademyQuality, "academy_quality")
	set(footballai.SlotTrainingQuality, "training_quality")
	if s, ok := text(f, "position_group"); ok {
		if p, ok := footballai.PositionFromName(s); ok {
			slots[footballai.SlotPositionOneHot+int(p)] = 1
		}
	}
	if s, ok := text(f, "personality"); ok {
		if p, ok := footballai.PersonalityFromName(s); ok {
			slots[footballai.SlotPersonality+int(p)] = 1
		}
	}
}

func fillClub(f map[string]json.RawMessage, slots []float32) {
	set := func(slot int, name string) {
		if v, ok := num(f, name); ok {
			slots[slot] = float32(v)
		}
	}
	set(footballai.SlotClubReputation, "club_reputation")
	set(footballai.SlotFinancialPower, "financial_power")
	set(footballai.SlotSellingTendency, "selling_tendency")
	set(footballai.SlotRecruitmentAmbition, "recruitment_ambition")
	set(footballai.SlotFinancialPressure, "financial_pressure")
}

func fillMatch(f map[string]json.RawMessage, slots []float32) {
	set := func(slot int, name string) {
		if v, ok := num(f, name); ok {
			slots[slot] = float32(v)
		}
	}
	set(footballai.SlotHomeRating, "home_rating")
	set(footballai.SlotAwayRating, "away_rating")
	set(footballai.SlotHomeForm, "home_form_modifier")
	set(footballai.SlotAwayForm, "away_form_modifier")
	set(footballai.SlotHomeFitness, "home_fitness")
	set(footballai.SlotAwayFitness, "away_fitness")
	set(footballai.SlotHomeFatigue, "home_fatigue")
	set(footballai.SlotAwayFatigue, "away_fatigue")
	set(footballai.SlotTacticalEdge, "tactical_edge")
	set(footballai.SlotMatchImportance, "match_importance")
	if v, ok := boolean(f, "derby"); ok && v {
		slots[footballai.SlotDerby] = 1
	}
	if v, ok := boolean(f, "rain"); ok && v {
		slots[footballai.SlotRain] = 1
	}
	if v, ok := boolean(f, "european_night"); ok && v {
		slots[footballai.SlotEuropeanNight] = 1
	}
	set(footballai.SlotHomeAbsences, "home_absences")
	set(footballai.SlotAwayAbsences, "away_absences")
	set(footballai.SlotHomeAttackBias, "home_attack_bias")
	set(footballai.SlotAwayAttackBias, "away_attack_bias")
}

func fillFinancial(f map[string]json.RawMessage, slots []float32) {
	set := func(slot int, name string) {
		if v, ok := num(f, name); ok {
			slots[slot] = float32(v)
		}
	}
	set(footballai.SlotContractYears, "contract_years")
	set(footballai.SlotLoyalty, "loyalty")
	set(footballai.SlotBaselineAnchor, "baseline_anchor_eur")
	set(footballai.SlotBaselineWage, "baseline_wage_weekly_eur")
	set(footballai.SlotBidPrice, "bid_price_eur")
	set(footballai.SlotAskingPrice, "asking_price_eur")
	set(footballai.SlotWageHeadroom, "wage_headroom")
	if s, ok := text(f, "squad_role"); ok {
		if r, ok := footballai.SquadRoleFromName(s); ok {
			slots[footballai.SlotSquadRoleOneHot+int(r)] = 1
		}
	}
}

func fillBoard(f map[string]json.RawMessage, slots []float32) {
	set := func(slot int, name string) {
		if v, ok := num(f, name); ok {
			slots[slot] = float32(v)
		}
	}
	set(footballai.SlotPointsPerGame, "points_per_game")
	set(footballai.SlotExpectedPPG, "expected_points_per_game")
	set(footballai.SlotLast5PPG, "last5_points_per_game")
	set(footballai.SlotBoardPatience, "board_patience")
	set(footballai.SlotSpendRatio, "spend_ratio_vs_expectation")
	set(footballai.SlotManagerTenureMonths, "manager_tenure_months")
	set(footballai.SlotCupProgress, "cup_progress_score")
}

func fillWorld(f map[string]json.RawMessage, slots []float32) {
	set := func(slot int, name string) {
		if v, ok := num(f, name); ok {
			slots[slot] = float32(v)
		}
	}
	set(footballai.SlotWorldImportance, "match_importance")
	set(footballai.SlotWorldFatigue, "player_fatigue")
	set(footballai.SlotTransferPressure, "transfer_pressure")
	set(footballai.SlotClubPressure, "club_pressure")
}

func fillManager(f map[string]json.RawMessage, slots []float32) {
	set := func(slot int, name string) {
		if v, ok := num(f, name); ok {
			slots[slot] = float32(v)
		}
	}
	set(footballai.SlotMgrRisk, "manager_risk")
	set(footballai.SlotMgrYouthPreference, "manager_youth_preference")
	set(footballai.SlotMgrFinancialCaution, "manager_financial_caution")
	set(footballai.SlotMgrPatience, "manager_patience")
	set(footballai.SlotMgrRotationPreference, "manager_rotation_preference")
	set(footballai.SlotMgrAttacking, "manager_attacking")
	set(footballai.SlotMgrPressing, "manager_pressing")
	set(footballai.SlotMgrReputation, "manager_reputation")
	set(footballai.SlotMgrExperience, "manager_experience")
	set(footballai.SlotMgrAdaptability, "manager_adaptability")
	set(footballai.SlotMgrTenure, "manager_tenure_months")
}

// eventTask maps router event types to their runtime task, so router
// examples exercise the same task embedding the live tasks use.
func eventTask(event string) (Task, bool) {
	switch event {
	case "MATCH_PREDICTION":
		return footballai.TaskMatchPrediction, true
	case "LINEUP_SELECTION":
		return footballai.TaskLineupSelection, true
	case "ROTATION_DECISION":
		return footballai.TaskRotation, true
	case "INJURY_RISK":
		return footballai.TaskInjuryRisk, true
	case "PLAYER_DEVELOPMENT":
		return footballai.TaskDevelopment, true
	case "PLAYER_DECLINE":
		return footballai.TaskDecline, true
	case "VALUATION":
		return footballai.TaskValuation, true
	case "TRANSFER_BID":
		return footballai.TaskTransferBid, true
	case "NEGOTIATION":
		return footballai.TaskNegotiation, true
	case "CONTRACT_RENEWAL":
		return footballai.TaskContract, true
	case "BOARD_SACK_DECISION":
		return footballai.TaskBoardPatience, true
	case "MANAGER_HIRING":
		return footballai.TaskManagerHiring, true
	case "SET_PIECE":
		return footballai.TaskSetPiece, true
	case "CROWD_RESPONSE":
		return footballai.TaskCrowd, true
	}
	return 0, false
}

// convertRow converts one raw JSONL row into a Sample.
func (b *sampleBuilder) convertRow(row rawRow) (*Sample, error) {
	if row.LabelSource == "" {
		row.LabelSource = string(footballai.LabelBootstrapTeacher)
	}
	s := &Sample{
		Slots:       make([]float32, footballai.InputWidth),
		Split:       row.Split,
		GroupID:     row.GroupID,
		LabelSource: footballai.LabelSource(row.LabelSource),
		Weight:      1,
	}
	f := row.Features
	switch row.Task {
	case "router":
		task, ok := eventTask(row.EventType)
		if !ok {
			task = footballai.TaskRouter
		}
		s.Task = task
		s.IsRouter = true
		fillManager(f, s.Slots)
		fillWorld(f, s.Slots)
		var rt rawRouterTarget
		if err := decodeRouterTarget(row.Target, &rt); err != nil {
			return nil, err
		}
		s.Teacher = []float32{
			float32(rt.ExpertWeights["match"]),
			float32(rt.ExpertWeights["player"]),
			float32(rt.ExpertWeights["economy"]),
			float32(rt.ExpertWeights["club"]),
		}
	case "match_prediction":
		s.Task = footballai.TaskMatchPrediction
		fillMatch(f, s.Slots)
		fillWorld(f, s.Slots)
		expHome, _ := num(row.Target, "expected_home_goals")
		expAway, _ := num(row.Target, "expected_away_goals")
		expDiff, _ := num(row.Target, "expected_goal_diff")
		actDiff, _ := num(row.Target, "goal_diff")
		s.Target = []float32{
			float32(expHome), float32(expAway), float32(expDiff),
			float32(math.Abs(actDiff - expDiff)), // uncertainty proxy
		}
		s.AuxGoalDiff = float32(actDiff)
		switch res, _ := text(row.Target, "result"); res {
		case "home":
			s.AuxResult = 1
		case "away":
			s.AuxResult = -1
		default:
			s.AuxResult = 0
		}
	case "injury_risk", "rotation", "development", "decline":
		task, _ := footballai.TaskFromName(row.Task)
		s.Task = task
		fillPlayer(f, s.Slots)
		fillWorld(f, s.Slots)
		spec := footballai.TaskSpecs()[task]
		s.Target = make([]float32, spec.OutputWidth())
		if occ, ok := boolean(row.Target, "injury_occurred"); ok {
			if occ {
				s.AuxInjury = 1
			} else {
				s.AuxInjury = 0
			}
		} else {
			s.AuxInjury = -1
		}
		for i, out := range spec.Outputs {
			var v float64
			switch out.Name {
			case "injury_probability":
				v, _ = num(row.Target, "injury_probability")
			case "start_probability":
				v, _ = num(row.Target, "start_probability")
			case "rest_value":
				v, _ = num(row.Target, "rest_value")
			case "ovr_delta_season":
				v, _ = num(row.Target, "expected_ovr_delta_season")
			case "ovr_decline_season":
				v, _ = num(row.Target, "expected_ovr_decline_season")
			}
			s.Target[i] = footballai.TargetTransform(out.Kind, float32(v))
		}
	case "valuation", "negotiation", "contract", "board_patience":
		task, _ := footballai.TaskFromName(row.Task)
		s.Task = task
		fillPlayer(f, s.Slots)
		fillClub(f, s.Slots)
		fillFinancial(f, s.Slots)
		fillBoard(f, s.Slots)
		fillWorld(f, s.Slots)
		spec := footballai.TaskSpecs()[task]
		s.Target = make([]float32, spec.OutputWidth())
		for i, out := range spec.Outputs {
			var v float64
			switch out.Name {
			case "premium_multiplier":
				v, _ = num(row.Target, "premium_multiplier")
			case "accept_probability":
				v, _ = num(row.Target, "accept_probability")
			case "counteroffer_ratio_to_ask":
				v, _ = num(row.Target, "counteroffer_ratio_to_ask")
			case "walk_away_probability":
				v, _ = num(row.Target, "walk_away_probability")
			case "renew_probability":
				v, _ = num(row.Target, "renew_probability")
			case "wage_demand_ratio":
				wage, _ := num(row.Target, "demanded_wage_weekly_eur")
				base, _ := num(f, "baseline_wage_weekly_eur")
				if base > 0 && wage > 0 {
					v = wage / base
				}
			case "sack_probability":
				v, _ = num(row.Target, "sack_probability")
			}
			s.Target[i] = footballai.TargetTransform(out.Kind, float32(v))
		}
	default:
		return nil, fmt.Errorf("unknown task %q", row.Task)
	}
	return s, nil
}

func decodeRouterTarget(target map[string]json.RawMessage, rt *rawRouterTarget) error {
	raw, ok := target["expert_weights"]
	if !ok {
		return fmt.Errorf("router target missing expert_weights")
	}
	if err := json.Unmarshal(raw, &rt.ExpertWeights); err != nil {
		return fmt.Errorf("bad expert_weights: %w", err)
	}
	return nil
}
