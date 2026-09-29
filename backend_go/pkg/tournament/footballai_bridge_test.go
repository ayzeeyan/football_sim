package tournament

import (
	"fmt"
	"testing"

	"football_sim/pkg/footballai"
	"football_sim/pkg/managers"
	"football_sim/pkg/matchengine"
	"football_sim/pkg/models"
)

// bridgeTestModel builds a model for gating tests. The match head's bias is
// nudged so the untrained network emits plausible positive xG; without the
// nudge the bridge correctly rejects the untrained model's negative
// expectations (which TestMatchModelHintRejection covers separately).
func bridgeTestModel(t *testing.T) *footballai.Model {
	t.Helper()
	net, err := footballai.NewNetwork(footballai.DefaultConfig(), 7)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range net.Params() {
		if p.Name == "head.match.fc2.bias" {
			for i := range p.Data {
				p.Data[i] = 2.0 // plausible expected goals
			}
		}
	}
	return footballai.BuildModel(net, "test-hash")
}

// TestMatchModelHintRejection proves implausible model outputs never reach
// the engine: a plain untrained network must leave the config untouched.
func TestMatchModelHintRejection(t *testing.T) {
	net, err := footballai.NewNetwork(footballai.DefaultConfig(), 9)
	if err != nil {
		t.Fatal(err)
	}
	brain := footballai.NewBrainWithModel(footballai.BuildModel(net, "h"), footballai.AIConfig{UseMatchModel: true}, nil)
	tm := bridgeTestManager(brain)
	cfg := &matchengine.InstantMatchConfig{}
	tm.applyMatchModelHint(cfg, tm.Clubs["HOM"], tm.Clubs["AWY"], &Fixture{FixtureID: "l-MW01-HOM-AWY"})
	if cfg.XGHint != nil {
		t.Fatalf("untrained model's implausible hint was forwarded: %+v", cfg.XGHint)
	}
}

func bridgeTestClubs() (home, away *models.Club) {
	home = &models.Club{
		ClubID: "HOM", ClubName: "Home FC", OverallTeamRating: 82,
		Form: []string{"W", "W", "D", "L", "W"},
	}
	home.Squad = append(home.Squad, &models.Player{
		PlayerID: "H1", OVR: 82, Age: 26, Fitness: 90, Sharpness: 80, Morale: 75, Position: "CM",
	}, &models.Player{
		PlayerID: "H2", OVR: 80, Age: 29, Fitness: 60, Sharpness: 70, Morale: 70, Position: "ST", Injury: "Hamstring",
	})
	away = &models.Club{
		ClubID: "AWY", ClubName: "Away FC", OverallTeamRating: 75,
		Form: []string{"L", "L", "D", "D", "L"},
	}
	away.Squad = append(away.Squad, &models.Player{
		PlayerID: "A1", OVR: 75, Age: 24, Fitness: 85, Sharpness: 75, Morale: 68, Position: "CB",
	})
	return home, away
}

func bridgeTestManager(brain *footballai.Brain) *TournamentManager {
	home, away := bridgeTestClubs()
	return &TournamentManager{
		Clubs:    map[string]*models.Club{"HOM": home, "AWY": away},
		Managers: map[string]*managers.ManagerProfile{},
		AIBrain:  brain,
	}
}

// TestMatchModelHintGating proves the default simulation is untouched: no
// brain, or a brain with the match flag off, leaves the engine config without
// a hint. Only an explicitly enabled brain overlays a hint.
func TestMatchModelHintGating(t *testing.T) {
	fixture := &Fixture{FixtureID: "premier-league-MW01-HOM-AWY", Competition: "league"}
	testCases := []struct {
		name  string
		brain *footballai.Brain
		want  bool // hint expected?
	}{
		{"nil brain", nil, false},
		{"flags off", footballai.NewBrainWithModel(bridgeTestModel(t), footballai.AIConfig{}, nil), false},
		{"injury only", footballai.NewBrainWithModel(bridgeTestModel(t), footballai.AIConfig{UseInjuryModel: true}, nil), false},
		{"match enabled", footballai.NewBrainWithModel(bridgeTestModel(t), footballai.AIConfig{UseMatchModel: true}, nil), true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tm := bridgeTestManager(tc.brain)
			cfg := &matchengine.InstantMatchConfig{}
			tm.applyMatchModelHint(cfg, tm.Clubs["HOM"], tm.Clubs["AWY"], fixture)
			if (cfg.XGHint != nil) != tc.want {
				t.Fatalf("hint present=%v, want %v", cfg.XGHint != nil, tc.want)
			}
			if cfg.XGHint != nil {
				// The bridge only forwards plausible expectations.
				if cfg.XGHint.Home <= 0 || cfg.XGHint.Home > 6 ||
					cfg.XGHint.Away <= 0 || cfg.XGHint.Away > 6 {
					t.Fatalf("implausible hint forwarded: %+v", cfg.XGHint)
				}
			}
		})
	}
}

// TestInjuryChanceGating proves the injury adjustment is identity without an
// enabled flag, and bounded by its corridor when enabled.
func TestInjuryChanceGating(t *testing.T) {
	player := &models.Player{
		PlayerID: "P1", OVR: 78, Age: 28, Fitness: 82, Sharpness: 70, Morale: 72, Position: "MID",
	}
	const base = 0.05
	if got := bridgeTestManager(nil).adjustInjuryChance(base, player, "f1"); got != base {
		t.Fatalf("nil brain changed the chance: %g", got)
	}
	disabled := footballai.NewBrainWithModel(bridgeTestModel(t), footballai.AIConfig{}, nil)
	if got := bridgeTestManager(disabled).adjustInjuryChance(base, player, "f1"); got != base {
		t.Fatalf("disabled brain changed the chance: %g", got)
	}
	enabled := footballai.NewBrainWithModel(bridgeTestModel(t), footballai.AIConfig{UseInjuryModel: true}, nil)
	got := bridgeTestManager(enabled).adjustInjuryChance(base, player, "f1")
	// The corridor formula (0.7 + 0.3*ratio) with ratio in [0.4, 2.5]
	// bounds the adjustment to [0.82x, 1.45x].
	if got < 0.82*base-1e-9 || got > 1.45*base+1e-9 {
		t.Fatalf("adjustment outside corridor: %g (base %g)", got, base)
	}
}

// TestInjuryChanceDeterministic proves the model overlay is deterministic:
// the same state produces the same adjustment every time.
func TestInjuryChanceDeterministic(t *testing.T) {
	player := &models.Player{PlayerID: "P1", OVR: 78, Age: 28, Position: "ST"}
	brain := footballai.NewBrainWithModel(bridgeTestModel(t), footballai.AIConfig{UseInjuryModel: true}, nil)
	tm := bridgeTestManager(brain)
	first := tm.adjustInjuryChance(0.05, player, "f1")
	for i := 0; i < 10; i++ {
		if got := tm.adjustInjuryChance(0.05, player, "f1"); got != first {
			t.Fatalf("non-deterministic adjustment: %g vs %g", got, first)
		}
	}
}

// TestBridgeFeatureMapping sanity-checks the feature builders used by the
// bridge: positions map, form modifiers sign correctly, squad helpers count.
func TestBridgeFeatureMapping(t *testing.T) {
	home, away := bridgeTestClubs()
	tm := bridgeTestManager(nil)

	f := tm.playerFeatures(home.Squad[0])
	if f.Position != footballai.PosMID {
		t.Fatalf("position mapping wrong: %d", f.Position)
	}
	if f.Potential != f.OVR {
		t.Fatalf("neutral potential expected, got %g vs %g", f.Potential, f.OVR)
	}
	injured := tm.playerFeatures(home.Squad[1])
	if !injured.RecentInjury {
		t.Fatal("injured player must set RecentInjury")
	}

	if m := clubFormModifier(home); m != 6 { // W W D L W = +3+3+0-3+3
		t.Fatalf("home form modifier %g, want 6", m)
	}
	if m := clubFormModifier(away); m != -9 { // L L D D L
		t.Fatalf("away form modifier %g, want -9", m)
	}
	if n := squadInjuries(home); n != 1 {
		t.Fatalf("injury count %d, want 1", n)
	}
	if fit := squadAvgFitness(home); fit != 75 { // (90+60)/2
		t.Fatalf("avg fitness %g, want 75", fit)
	}
	_ = fmt.Sprintf
}
