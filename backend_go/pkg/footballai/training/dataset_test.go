package training

import (
	"os"
	"path/filepath"
	"testing"

	"football_sim/pkg/footballai"
)

func writeTempJSONL(t *testing.T, rows []string) string {
	t.Helper()
	dir := t.TempDir()
	for i, name := range DatasetFiles {
		path := filepath.Join(dir, name)
		var content string
		if i == 0 && name == "match_seed.jsonl" || name == "match_seed.jsonl" {
			content = ""
		}
		_ = content
		if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// Write real rows into the right files.
	matchRows := []string{}
	routerRows := []string{}
	playerRows := []string{}
	econRows := []string{}
	for _, r := range rows {
		switch {
		case contains(r, `"task":"match_prediction"`):
			matchRows = append(matchRows, r)
		case contains(r, `"task":"router"`):
			routerRows = append(routerRows, r)
		case contains(r, `"task":"injury_risk"`) || contains(r, `"task":"rotation"`) ||
			contains(r, `"task":"development"`) || contains(r, `"task":"decline"`):
			playerRows = append(playerRows, r)
		default:
			econRows = append(econRows, r)
		}
	}
	write := func(name string, rs []string) {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(joinLines(rs)), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("match_seed.jsonl", matchRows)
	write("router_seed.jsonl", routerRows)
	write("player_seed.jsonl", playerRows)
	write("economy_club_seed.jsonl", econRows)
	return dir
}

func joinLines(rs []string) string {
	out := ""
	for _, r := range rs {
		out += r + "\n"
	}
	return out
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}

func TestDatasetParsingAndSplits(t *testing.T) {
	dir := writeTempJSONL(t, []string{
		`{"schema_version":1,"label_source":"bootstrap_teacher_v1","task":"match_prediction","group_id":1,"split":"train","features":{"home_rating":82.5,"away_rating":75.9,"home_form_modifier":-3.2,"away_form_modifier":1.6,"home_fitness":87,"away_fitness":85,"home_fatigue":2.7,"away_fatigue":0.0,"tactical_edge":-0.32,"match_importance":0.16,"derby":false,"rain":true,"european_night":false,"home_absences":1,"away_absences":2,"home_attack_bias":0.49,"away_attack_bias":0.07},"target":{"expected_home_goals":2.17,"expected_away_goals":0.86,"expected_goal_diff":1.30,"home_goals":1,"away_goals":1,"goal_diff":0,"result":"draw"}}`,
		`{"schema_version":1,"label_source":"bootstrap_teacher_v1","task":"match_prediction","group_id":2,"split":"val","features":{"home_rating":76.8,"away_rating":72.5,"home_fitness":78,"away_fitness":87},"target":{"expected_home_goals":1.95,"expected_away_goals":1.4,"expected_goal_diff":0.55,"goal_diff":4,"result":"home"}}`,
		`{"schema_version":1,"label_source":"bootstrap_teacher_v1","task":"router","event_type":"BOARD_SACK_DECISION","group_id":3,"split":"train","features":{"manager_risk":0.55,"manager_youth_preference":0.28,"manager_financial_caution":0.79,"manager_patience":0.83,"match_importance":0.21,"player_fatigue":0.09,"transfer_pressure":0.44,"club_pressure":0.35},"target":{"expert_weights":{"match":0.27,"player":0.001,"economy":0.005,"club":0.72},"top2":["club","match"]}}`,
		`{"schema_version":1,"label_source":"bootstrap_teacher_v1","task":"injury_risk","group_id":4,"split":"train","features":{"age":27,"ovr":77.7,"potential":77.7,"fitness":83.5,"sharpness":76.4,"fatigue":5.4,"consecutive_starts":5,"minutes_last_14_days":232.3,"morale":84.8,"form_modifier":-3.7,"recent_injury":false,"position_group":"FWD","personality":"academic_dual","match_importance":0.72,"replacement_ovr":74.3,"academy_quality":0.07,"training_quality":0.61},"target":{"injury_probability":0.08,"injury_occurred":true}}`,
		`{"schema_version":1,"label_source":"bootstrap_teacher_v1","task":"valuation","group_id":5,"split":"test","features":{"age":20,"ovr":70.9,"potential":70.9,"contract_years":3,"loyalty":100,"morale":77.8,"form_modifier":3.4,"club_reputation":38.5,"financial_power":87.8,"selling_tendency":0.53,"recruitment_ambition":0.97,"squad_role":"rotation","baseline_anchor_eur":20715594.1,"baseline_wage_weekly_eur":18440.49},"target":{"premium_multiplier":1.505,"market_value_eur":31177206.28}}`,
	})

	train, err := LoadSplit(dir, SplitTrain, func(string) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	if len(train) != 3 {
		t.Fatalf("expected 3 train samples (match, router, injury), got %d", len(train))
	}
	val, err := LoadSplit(dir, SplitVal, func(string) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	if len(val) != 1 || val[0].Task != footballai.TaskMatchPrediction {
		t.Fatalf("val split wrong: %d", len(val))
	}
	test, err := LoadSplit(dir, SplitTest, func(string) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	if len(test) != 1 || test[0].Task != footballai.TaskValuation {
		t.Fatal("test split wrong")
	}

	// Locate rows by task; LoadSplit streams files in manifest order, so
	// positional assumptions would be fragile.
	var m, r, injury *Sample
	for _, s := range train {
		switch {
		case s.IsRouter:
			r = s
		case s.Task == footballai.TaskMatchPrediction:
			m = s
		case s.Task == footballai.TaskInjuryRisk:
			injury = s
		}
	}
	if m == nil || r == nil || injury == nil {
		t.Fatalf("train split incomplete: match=%v router=%v injury=%v", m != nil, r != nil, injury != nil)
	}
	if m.Slots[footballai.SlotHomeRating] != 82.5 || m.Slots[footballai.SlotRain] != 1 {
		t.Fatal("match features not slotted")
	}
	if m.Target[0] != 2.17 || m.Target[3] != 1.30 { // uncertainty = |0 - 1.30|
		t.Fatalf("match target wrong: %v", m.Target)
	}
	if m.AuxResult != 0 || m.AuxGoalDiff != 0 {
		t.Fatal("match aux wrong")
	}

	// Router sample.
	if !r.IsRouter {
		t.Fatal("router flag missing")
	}
	if r.Task != footballai.TaskBoardPatience {
		t.Fatalf("router event task wrong: %v", r.Task)
	}
	if r.Teacher[footballai.ExpertClub] != 0.72 {
		t.Fatalf("teacher weights wrong: %v", r.Teacher)
	}
	if r.Slots[footballai.SlotMgrRisk] != 0.55 || r.Slots[footballai.SlotClubPressure] != 0.35 {
		t.Fatal("router context slots wrong")
	}

	// Injury sample.
	if injury.Target[0] != 0.08 || injury.AuxInjury != 1 {
		t.Fatalf("injury target/aux wrong: %v %d", injury.Target, injury.AuxInjury)
	}
	if injury.Slots[footballai.SlotPositionOneHot+int(footballai.PosFWD)] != 1 {
		t.Fatal("position one-hot missing")
	}

	// Valuation log-space target.
	v := test[0]
	wantLog := float32(0.4081) // log(1.505)
	if v.Target[0] < wantLog-0.001 || v.Target[0] > wantLog+0.001 {
		t.Fatalf("valuation log target wrong: %v", v.Target)
	}
	if v.Slots[footballai.SlotSquadRoleOneHot+int(footballai.SquadRoleRotation)] != 1 {
		t.Fatal("squad role one-hot missing")
	}
}

func TestStreamingReader(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "r.jsonl")
	rows := ""
	for i := 0; i < 100; i++ {
		rows += `{"schema_version":1,"label_source":"simulation_outcome","task":"development","group_id":1,"split":"train","features":{"age":25,"ovr":80},"target":{"expected_ovr_delta_season":0.5}}` + "\n"
	}
	if err := os.WriteFile(path, []byte(rows), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := OpenDataset(path)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	count := 0
	for {
		s, err := r.Next()
		if err != nil && err.Error() == "EOF" {
			break
		}
		if err != nil {
			// io.EOF comparison
			if count == 100 {
				break
			}
			t.Fatal(err)
		}
		if s.LabelSource != footballai.LabelSimulation {
			t.Fatalf("label source wrong: %v", s.LabelSource)
		}
		count++
	}
	if count != 100 {
		t.Fatalf("streamed %d rows, want 100", count)
	}
}
