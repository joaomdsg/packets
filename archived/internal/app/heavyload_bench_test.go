package app_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/joaomdsg/packets/internal/app"
)

// BenchmarkHeavyConcurrentCycle pushes the uncapped per-connect fan-out well past
// the core count (K up to 64 on a 20-core box) to observe how Resolve degrades
// under heavy concurrent load: every cycle fires 3 full-suite execs, so K cycles
// contend for CPU running 3K `go test ./...` invocations at once. Reports the
// effective wall-clock per cycle (latency under contention) and aggregate
// suite-exec throughput.
func BenchmarkHeavyConcurrentCycle(b *testing.B) {
	for _, K := range []int{8, 16, 32, 64} {
		b.Run(fmt.Sprintf("K=%d", K), func(b *testing.B) {
			type repo struct{ dir, base, fix string }
			repos := make([]repo, K)
			for i := range repos {
				repos[i].dir, repos[i].base, repos[i].fix = strengthenRepo(b)
			}
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				var wg sync.WaitGroup
				for k := 0; k < K; k++ {
					wg.Add(1)
					go func(k int) {
						defer wg.Done()
						_, _ = app.Resolve(context.Background(), repos[k].dir, repos[k].base, repos[k].fix, repos[k].fix,
							anchor(), goTestCmd, false, false)
					}(k)
				}
				wg.Wait()
			}
			b.ReportMetric(float64(K*3), "suite-execs/op")
			b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/float64(K)/1e6, "ms/cycle")
		})
	}
}

// BenchmarkSaturationSweep holds a fixed total amount of work (TOTAL cycles) and
// sweeps the concurrency width to find where contention overtakes parallelism —
// the knee the Board's cap is meant to sit at. Reports cycles/sec.
func BenchmarkSaturationSweep(b *testing.B) {
	const TOTAL = 48
	for _, width := range []int{1, 4, 8, 16, 24, 48} {
		b.Run(fmt.Sprintf("width=%d", width), func(b *testing.B) {
			type repo struct{ dir, base, fix string }
			repos := make([]repo, TOTAL)
			for i := range repos {
				repos[i].dir, repos[i].base, repos[i].fix = strengthenRepo(b)
			}
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				var next int64 = -1
				var wg sync.WaitGroup
				for w := 0; w < width; w++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						for {
							i := atomic.AddInt64(&next, 1)
							if i >= TOTAL {
								return
							}
							_, _ = app.Resolve(context.Background(), repos[i].dir, repos[i].base, repos[i].fix, repos[i].fix,
								anchor(), goTestCmd, false, false)
						}
					}()
				}
				wg.Wait()
			}
			secs := b.Elapsed().Seconds() / float64(b.N)
			b.ReportMetric(float64(TOTAL)/secs, "cycles/sec")
		})
	}
}
