package training

import (
	"math"

	"football_sim/pkg/footballai"
)

// groupMaskOf returns a per-slot boolean mask of the groups a sample's task
// fills. Normalization statistics are computed only over filled slots so
// zero-filled irrelevant groups do not distort the estimates.
func groupMaskOf(s *Sample) [footballai.InputWidth]bool {
	var mask [footballai.InputWidth]bool
	enable := func(start, width int) {
		for i := start; i < start+width; i++ {
			mask[i] = true
		}
	}
	player := func() { enable(0, 24) }
	club := func() { enable(24, 5) }
	match := func() { enable(29, 17) }
	fin := func() { enable(46, 8) }
	board := func() { enable(59, 7) }
	world := func() { enable(66, 4) }
	mgr := func() { enable(70, 11) }

	if s.IsRouter {
		world()
		mgr()
		return mask
	}
	switch s.Task {
	case footballai.TaskMatchPrediction:
		match()
		world()
		mgr()
	case footballai.TaskInjuryRisk:
		player()
		world()
	case footballai.TaskRotation:
		player()
		world()
		mgr()
	case footballai.TaskDevelopment, footballai.TaskDecline:
		player()
		world()
	case footballai.TaskValuation, footballai.TaskNegotiation, footballai.TaskContract:
		player()
		club()
		fin()
		world()
	case footballai.TaskBoardPatience:
		player()
		club()
		board()
		mgr()
		world()
	}
	return mask
}

// computeNormMeta derives per-slot mean/std from the training split using
// Welford's method. The result is installed on the network and exported in
// the .fmoe file; inference never recomputes it.
func computeNormMeta(train []*Sample) footballai.NormMeta {
	meta := footballai.DefaultNormMeta()
	type acc struct {
		count float64
		mean  float64
		m2    float64
	}
	accs := make([]acc, footballai.InputWidth)

	for _, s := range train {
		mask := groupMaskOf(s)
		for i := 0; i < footballai.InputWidth; i++ {
			if !mask[i] {
				continue
			}
			var v float64
			switch meta.Slots[i].Kind {
			case footballai.NormLogZ:
				v = math.Log1p(float64(max64(float64(s.Slots[i]), 0)))
			case footballai.NormID:
				continue
			default:
				v = float64(s.Slots[i])
			}
			a := &accs[i]
			a.count++
			delta := v - a.mean
			a.mean += delta / a.count
			a.m2 += delta * (v - a.mean)
		}
	}
	for i := range meta.Slots {
		if meta.Slots[i].Kind == footballai.NormID {
			continue
		}
		a := accs[i]
		if a.count < 2 {
			meta.Slots[i].Mean = 0
			meta.Slots[i].Std = 1
			continue
		}
		std := math.Sqrt(a.m2 / (a.count - 1))
		if std < 1e-6 {
			std = 1
		}
		meta.Slots[i].Mean = float32(a.mean)
		meta.Slots[i].Std = float32(std)
	}
	return meta
}

func max64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// computeBaselines evaluates simple deterministic baselines on the test
// split so the learned model is never accepted merely because it trained.
func computeBaselines(train, val, test []*Sample) map[string]map[string]float64 {
	out := map[string]map[string]float64{}

	// Injury: fixed mean-rate heuristic.
	var injurySum float64
	var injuryN float64
	for _, s := range train {
		if s.Task == footballai.TaskInjuryRisk && len(s.Target) > 0 {
			injurySum += float64(s.Target[0])
			injuryN++
		}
	}
	meanRate := 0.05
	if injuryN > 0 {
		meanRate = injurySum / injuryN
	}
	var brier float64
	var brierN float64
	for _, s := range test {
		if s.Task == footballai.TaskInjuryRisk && s.AuxInjury >= 0 {
			d := meanRate - float64(s.AuxInjury)
			brier += d * d
			brierN++
		}
	}
	if brierN > 0 {
		out["injury_risk"] = map[string]float64{"baseline_brier": brier / brierN}
	}

	// Match: rating-difference heuristic.
	var maeGD, maeXG, n float64
	for _, s := range test {
		if s.Task != footballai.TaskMatchPrediction || len(s.Target) < 3 {
			continue
		}
		diff := float64(s.Slots[footballai.SlotHomeRating] - s.Slots[footballai.SlotAwayRating])
		baseGD := 0.25 + diff*0.04
		baseHome := 1.35 + diff*0.03
		baseAway := 1.10 - diff*0.03
		maeGD += math.Abs(baseGD - float64(s.AuxGoalDiff))
		maeXG += (math.Abs(baseHome-float64(s.Target[0])) + math.Abs(baseAway-float64(s.Target[1]))) / 2
		n++
	}
	if n > 0 {
		out["match_prediction"] = map[string]float64{
			"baseline_mae_goal_diff": maeGD / n,
			"baseline_mae_xg":        maeXG / n,
		}
	}

	// Valuation: premium = 1 (the deterministic baseline itself).
	var mape, mapeN float64
	for _, s := range test {
		if s.Task != footballai.TaskValuation || len(s.Target) == 0 {
			continue
		}
		target := math.Exp(float64(s.Target[0]))
		if target > 0 {
			mape += math.Abs(1-target) / target
			mapeN++
		}
	}
	if mapeN > 0 {
		out["valuation"] = map[string]float64{"baseline_mape_premium": mape / mapeN}
	}

	// Board patience: constant mean sack probability.
	var sackSum, sackN float64
	for _, s := range train {
		if s.Task == footballai.TaskBoardPatience && len(s.Target) > 0 {
			sackSum += float64(s.Target[0])
			sackN++
		}
	}
	if sackN > 0 {
		meanSack := sackSum / sackN
		var b float64
		var bn float64
		for _, s := range test {
			if s.Task == footballai.TaskBoardPatience && len(s.Target) > 0 {
				d := meanSack - float64(s.Target[0])
				b += d * d
				bn++
			}
		}
		if bn > 0 {
			out["board_patience"] = map[string]float64{"baseline_mse_sack": b / bn}
		}
	}
	return out
}
