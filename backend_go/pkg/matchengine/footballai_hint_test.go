package matchengine

import "testing"

// TestBlendLambdaWithHint verifies the corridor policy that keeps learned
// expected goals advisory: identical hints change nothing, extreme hints stay
// inside the engine's calibrated range.
func TestBlendLambdaWithHint(t *testing.T) {
	const base = 1.5

	// A hint equal to the base rate is a no-op.
	if got := BlendLambdaWithHint(base, base); got != base {
		t.Fatalf("equal hint moved the rate: %g", got)
	}
	// Sane hints blend with alpha = 0.25.
	if got := BlendLambdaWithHint(base, 2.5); got != 1.75 {
		t.Fatalf("blend wrong: %g", got)
	}
	// Extreme hints clamp inside [0.6x, 1.67x] of the base rate.
	if got := BlendLambdaWithHint(base, 100); got > 1.67*base+1e-9 {
		t.Fatalf("upper corridor violated: %g", got)
	}
	if got := BlendLambdaWithHint(base, 0); got < 0.6*base-1e-9 {
		t.Fatalf("lower corridor violated: %g", got)
	}
	// Tiny base rates never collapse to zero.
	if got := BlendLambdaWithHint(0.05, 0); got < 0.2 {
		t.Fatalf("floor violated: %g", got)
	}
	// The blend is monotone in the hint inside the corridor.
	prev := BlendLambdaWithHint(base, 0)
	for h := 0.25; h <= 4.0; h += 0.25 {
		got := BlendLambdaWithHint(base, h)
		if got < prev-1e-9 {
			t.Fatalf("blend not monotone at hint %g: %g < %g", h, got, prev)
		}
		prev = got
	}
}
