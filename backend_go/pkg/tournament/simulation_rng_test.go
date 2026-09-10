package tournament

import "testing"

func TestSubsystemRNGStableAndIsolated(t *testing.T) {
	a := NewSubsystemRNG(123456)
	b := NewSubsystemRNG(123456)

	labels := []string{"matches", "transfers", "managers", "development", "live_match"}
	seen := map[int64]string{}
	for _, label := range labels {
		as := a.SeedFor(label)
		bs := b.SeedFor(label)
		if as != bs {
			t.Fatalf("same root seed produced different %s seed: %d vs %d", label, as, bs)
		}
		if previous, exists := seen[as]; exists {
			t.Fatalf("subsystems %s and %s unexpectedly share seed %d", previous, label, as)
		}
		seen[as] = label
	}

	matchA := a.New("matches")
	matchB := b.New("matches")
	for i := 0; i < 32; i++ {
		if x, y := matchA.Int63(), matchB.Int63(); x != y {
			t.Fatalf("same subsystem stream diverged at draw %d: %d vs %d", i, x, y)
		}
	}

	// Consuming transfers must not perturb matches because the streams are
	// independently derived rather than sharing one rand.Rand.
	transfer := a.New("transfers")
	for i := 0; i < 100; i++ {
		_ = transfer.Int63()
	}
	matchFresh := a.New("matches")
	matchControl := b.New("matches")
	for i := 0; i < 16; i++ {
		if x, y := matchFresh.Int63(), matchControl.Int63(); x != y {
			t.Fatalf("transfer consumption perturbed match stream at draw %d", i)
		}
	}
}

func TestSubsystemRNGDifferentRootsDiffer(t *testing.T) {
	a := NewSubsystemRNG(111)
	b := NewSubsystemRNG(222)
	if a.SeedFor("matches") == b.SeedFor("matches") {
		t.Fatal("different root seeds forced identical match seed")
	}
}
