package tournament

import (
	"encoding/binary"
	"hash/fnv"
	"math/rand"
)

// SubsystemRNG derives stable independent pseudo-random streams from one
// universe-owned seed. Adding a random call in one subsystem therefore does
// not consume another subsystem's sequence.
type SubsystemRNG struct {
	RootSeed int64
}

func NewSubsystemRNG(seed int64) SubsystemRNG {
	if seed == 0 {
		seed = 20260907
	}
	return SubsystemRNG{RootSeed: seed}
}

// SeedFor deterministically derives a non-zero signed seed for a subsystem.
func (s SubsystemRNG) SeedFor(subsystem string) int64 {
	h := fnv.New64a()
	var root [8]byte
	binary.LittleEndian.PutUint64(root[:], uint64(s.RootSeed))
	_, _ = h.Write(root[:])
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(subsystem))
	seed := int64(h.Sum64() & 0x7fffffffffffffff)
	if seed == 0 {
		return 1
	}
	return seed
}

// New returns a deterministic rand.Rand dedicated to one subsystem.
func (s SubsystemRNG) New(subsystem string) *rand.Rand {
	return rand.New(rand.NewSource(s.SeedFor(subsystem)))
}
