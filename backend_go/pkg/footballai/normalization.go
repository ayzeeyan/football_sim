package footballai

import (
	"math"
	"math/rand"
)

// SlotNorm describes how one input slot is normalized. The exact same
// metadata is stored in the .fmoe file and applied at training and inference
// time; nothing is recomputed independently at runtime.
type SlotNorm struct {
	Name string  `json:"name"`
	Kind string  `json:"kind"` // "z", "logz", or "id"
	Mean float32 `json:"mean"`
	Std  float32 `json:"std"`
}

// NormMeta is the full normalization metadata block.
type NormMeta struct {
	SchemaVersion uint16     `json:"schema_version"`
	Slots         []SlotNorm `json:"slots"`
}

// Slot norm kinds.
const (
	NormZ    = "z"    // (x - mean) / std
	NormLogZ = "logz" // (log1p(x) - mean) / std, for money values
	NormID   = "id"   // identity (one-hot, flags)
)

// slotNormKind returns the default normalization kind for a slot. One-hot
// and flag slots stay raw; money slots use a log transform; everything else
// is z-scored with statistics computed from the training split.
func slotNormKind(slot int) string {
	switch {
	case slot >= slotPositionOneHot && slot < slotPositionOneHot+4:
		return NormID
	case slot >= slotPersonality && slot < slotPersonality+5:
		return NormID
	case slot >= slotSquadRoleOneHot && slot < slotSquadRoleOneHot+5:
		return NormID
	case slot == slotRecentInjury, slot == slotDerby, slot == slotRain,
		slot == slotEuropeanNight:
		return NormID
	case slot == slotBaselineAnchor, slot == slotBaselineWage,
		slot == slotBidPrice, slot == slotAskingPrice:
		return NormLogZ
	default:
		return NormZ
	}
}

// DefaultNormMeta returns identity normalization (mean 0, std 1) for every
// slot, with the correct per-slot kinds. The trainer overwrites mean/std
// with training-split statistics before export.
func DefaultNormMeta() NormMeta {
	names := SlotNames()
	slots := make([]SlotNorm, InputWidth)
	for i := 0; i < InputWidth; i++ {
		slots[i] = SlotNorm{Name: names[i], Kind: slotNormKind(i), Mean: 0, Std: 1}
	}
	return NormMeta{SchemaVersion: NormSchemaVersion, Slots: slots}
}

// Apply normalizes one raw slot vector into a new vector.
func (m NormMeta) Apply(x []float32) []float32 {
	out := make([]float32, len(x))
	for i, s := range m.Slots {
		v := x[i]
		switch s.Kind {
		case NormLogZ:
			out[i] = float32((float64(math.Log1p(float64(max32(v, 0)))) - float64(s.Mean)) / float64(s.Std))
		case NormZ:
			out[i] = (v - s.Mean) / s.Std
		default:
			out[i] = v
		}
	}
	return out
}

// ApplyVec is an alias of Apply for readability at call sites.
func (m NormMeta) ApplyVec(x []float32) []float32 { return m.Apply(x) }

// newDeterministicRand returns a rand.Rand with the given seed, guaranteed
// non-zero.
func newDeterministicRand(seed int64) *rand.Rand {
	if seed == 0 {
		seed = 1
	}
	return rand.New(rand.NewSource(seed))
}
