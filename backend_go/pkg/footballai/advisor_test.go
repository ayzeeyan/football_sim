package footballai

import "testing"

func TestInjuryCalibrationRatio(t *testing.T) {
	// Average model risk maps to a neutral ratio of 1 (float32 conversion
	// of the constant may carry a few ulps of noise).
	if got := InjuryCalibrationRatio(MeanBootstrapInjuryProb); absF64(got-1) > 1e-6 {
		t.Fatalf("mean risk -> %g, want 1", got)
	}
	// The corridor bounds hold at the extremes.
	if got := InjuryCalibrationRatio(0); got != 0.4 {
		t.Fatalf("zero risk -> %g, want 0.4", got)
	}
	if got := InjuryCalibrationRatio(1); got != 2.5 {
		t.Fatalf("certainty -> %g, want 2.5", got)
	}
	if got := InjuryCalibrationRatio(2 * MeanBootstrapInjuryProb); absF64(got-2) > 1e-6 {
		t.Fatalf("double risk -> %g, want 2", got)
	}
}

func absF64(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func TestPositionFromGamePos(t *testing.T) {
	cases := map[string]Position{
		"GK": PosGK,
		"CB": PosDEF, "LB": PosDEF, "RB": PosDEF, "LWB": PosDEF, "RWB": PosDEF,
		"CDM": PosMID, "CM": PosMID, "CAM": PosMID,
		"LW": PosFWD, "RW": PosFWD, "CF": PosFWD, "ST": PosFWD,
		"": PosFWD, "XX": PosFWD, "gk": PosFWD, // unmapped falls back, never panics
	}
	for in, want := range cases {
		if got := PositionFromGamePos(in); got != want {
			t.Fatalf("PositionFromGamePos(%q) = %d, want %d", in, got, want)
		}
	}
}
