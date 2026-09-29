package training

import (
	"os"
	"path/filepath"
	"testing"
)

// BenchmarkDatasetParsing measures streaming JSONL decode throughput.
func BenchmarkDatasetParsing(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "bench.jsonl")
	row := `{"schema_version":1,"label_source":"bootstrap_teacher_v1","task":"match_prediction","group_id":1,"split":"train","features":{"home_rating":82.5,"away_rating":75.9,"home_form_modifier":-3.2,"away_form_modifier":1.6,"home_fitness":87,"away_fitness":85,"home_fatigue":2.7,"away_fatigue":0,"tactical_edge":-0.32,"match_importance":0.16,"derby":false,"rain":true,"european_night":false,"home_absences":1,"away_absences":2,"home_attack_bias":0.49,"away_attack_bias":0.07},"target":{"expected_home_goals":2.17,"expected_away_goals":0.86,"expected_goal_diff":1.3,"home_goals":1,"away_goals":1,"goal_diff":0,"result":"draw"}}`
	content := ""
	for i := 0; i < 1000; i++ {
		content += row + "\n"
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r, err := OpenDataset(path)
		if err != nil {
			b.Fatal(err)
		}
		n := 0
		for {
			_, err := r.Next()
			if err != nil {
				break
			}
			n++
		}
		r.Close()
		if n != 1000 {
			b.Fatalf("parsed %d rows", n)
		}
	}
}
