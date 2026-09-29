package footballai

import "math"

// Loss and gradient helpers shared by the trainer and the evaluation code.
// Each function returns the loss value and dLoss/dRawOutput for one output
// dimension, where "raw" is the head's linear output BEFORE the output kind
// activation (sigmoid for probabilities, exp for log multipliers).

const lossEps = 1e-7

// LossAndGrad computes the loss and raw-output gradient for one dimension.
func LossAndGrad(spec OutputSpec, raw, target float32, sampleWeight float32) (float32, float32) {
	switch spec.Kind {
	case OutProb:
		// Soft-label binary cross entropy on the sigmoid output.
		p := sigmoid(raw)
		p = clamp01(p)
		loss := -(target*float32(math.Log(float64(p+lossEps))) +
			(1-target)*float32(math.Log(float64(1-p+lossEps))))
		grad := p - target // d/draw of BCE∘sigmoid
		return loss * sampleWeight, grad * sampleWeight
	case OutLogMult:
		// Huber on log-space value; raw is already the log value.
		return huber(raw, target, spec.HuberDelta, sampleWeight)
	default:
		return spec.lossReg(raw, target, sampleWeight)
	}
}

func (s OutputSpec) lossReg(raw, target, sampleWeight float32) (float32, float32) {
	switch s.Loss {
	case LossMSE:
		d := raw - target
		return d * d * sampleWeight, 2 * d * sampleWeight
	default: // LossHuber
		return huber(raw, target, s.HuberDelta, sampleWeight)
	}
}

// huber returns the Huber loss and gradient, scaled by sampleWeight.
func huber(pred, target, delta, sampleWeight float32) (float32, float32) {
	d := pred - target
	a := d
	if a < 0 {
		a = -a
	}
	if a <= delta {
		return 0.5 * d * d * sampleWeight, d * sampleWeight
	}
	loss := delta * (a - 0.5*delta)
	grad := delta
	if d < 0 {
		grad = -delta
	}
	return loss * sampleWeight, grad * sampleWeight
}

// Activate applies an output kind activation to a raw head output.
func Activate(kind OutputKind, raw float32) float32 {
	switch kind {
	case OutProb:
		return clamp01(sigmoid(raw))
	case OutLogMult:
		return float32(math.Exp(float64(clampFloat(raw, -20, 20))))
	default:
		return raw
	}
}

// TargetTransform converts a raw dataset/observation value into the training
// target aligned with the output kind.
func TargetTransform(kind OutputKind, value float32) float32 {
	switch kind {
	case OutLogMult:
		return float32(math.Log(float64(max32(value, lossEps))))
	default:
		return value
	}
}

func clamp01(p float32) float32 {
	return clampFloat(p, lossEps, 1-lossEps)
}

func clampFloat(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func max32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

// KLToTeacher returns the cross-entropy of a predicted router distribution
// against a teacher distribution: -sum t_k log p_k.
func KLToTeacher(probs, teacher []float32) float32 {
	loss := float32(0)
	for k, t := range teacher {
		loss -= t * float32(math.Log(float64(probs[k]+lossEps)))
	}
	return loss
}
