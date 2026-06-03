package mutation

import (
	"context"
	"testing"
)

// BenchmarkRunManySites characterises the dominant cost of the oracle:
// one test-suite run per mutated operator site. The fixture has 30 sites,
// so ns/op ≈ 30 × (one `go test` invocation). Run with -benchtime=1x for
// a single honest wall-clock sample; the first run also pays compilation,
// later mutants benefit from the build cache — exactly as the settle loop
// would. Reported per-mutant via b.ReportMetric.
func BenchmarkRunManySites(b *testing.B) {
	for i := 0; i < b.N; i++ {
		findings, err := Run(context.Background(), Options{
			Dir:     "testdata/bench_many",
			File:    "many.go",
			TestCmd: goTestCmd,
		})
		if err != nil {
			b.Fatal(err)
		}
		b.ReportMetric(float64(len(findings)), "survivors")
	}
}
