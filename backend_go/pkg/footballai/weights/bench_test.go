package weights

import (
	"path/filepath"
	"testing"

	"football_sim/pkg/footballai"
)

// BenchmarkModelLoad measures .fmoe load time (checksum + architecture build
// + full tensor ingest).
func BenchmarkModelLoad(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "bench.fmoe")
	net, err := footballai.NewNetwork(footballai.DefaultConfig(), 1)
	if err != nil {
		b.Fatal(err)
	}
	if _, err := Save(path, net); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Load(path); err != nil {
			b.Fatal(err)
		}
	}
}
