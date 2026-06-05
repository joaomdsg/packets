package app_test

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-via/via"
	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/app"
)

// countingTestCmd records one byte per suite execution to an absolute counter
// path (outside the runner's per-worker copies and integrate worktree — the one
// place every suite-exec is observed), then runs the real suite. len(counter) is
// the exact number of full-suite runs.
func countingTestCmd(counterPath string) []string {
	return []string{"sh", "-c", "printf x >> '" + counterPath + "' && exec env -u GOROOT go test ./..."}
}

func suiteExecCount(t *testing.T, counterPath string) int {
	t.Helper()
	b, err := os.ReadFile(counterPath)
	if os.IsNotExist(err) {
		return 0
	}
	require.NoError(t, err)
	return len(b)
}

// strengthenRepo builds a fresh repo whose base→fix is a test-only strengthen on
// the `>=` anchor line — a cycle that mints exactly one Catch and fires exactly
// 3 suite-execs (M_base 1 + M_fix 1 + integrate 1).
func strengthenRepo(t testing.TB) (dir, base, fix string) {
	t.Helper()
	dir = initRepo(t)
	write(t, dir, "go.mod", "module adultapp\n\ngo 1.23\n")
	write(t, dir, "adult.go", adultGo)
	write(t, dir, "adult_test.go", weakTest)
	base = commitAll(t, dir, "base")
	write(t, dir, "adult_test.go", strongTest)
	fix = commitAll(t, dir, "strengthen the test")
	return dir, base, fix
}

func TestResolve_concurrentCyclesFanOutUncappedToThreeSuitesEach(t *testing.T) {
	const K = 3
	counter := t.TempDir() + "/suite-execs"

	type repo struct{ dir, base, fix string }
	repos := make([]repo, K)
	for i := range repos {
		repos[i].dir, repos[i].base, repos[i].fix = strengthenRepo(t)
	}

	results := make([]app.Resolution, K)
	errs := make([]error, K)
	var wg sync.WaitGroup
	start := make(chan struct{}) // barrier so the K cycles genuinely overlap, not a time.Sleep
	for i := 0; i < K; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			results[i], errs[i] = app.Resolve(context.Background(), repos[i].dir, repos[i].base, repos[i].fix, repos[i].fix,
				anchor(), countingTestCmd(counter), false, false)
		}(i)
	}
	close(start)
	wg.Wait()

	for i := range errs {
		require.NoErrorf(t, errs[i], "cycle %d", i)
	}
	assert.Equal(t, K*3, suiteExecCount(t, counter),
		"K concurrent cycles fire 3K full-suite executions — the per-connect fan-out is uncapped (no queue/cap), the multiplier the Board must bound")
}

func TestLiveServer_sharesOneLedgerAcrossConnectsSoTheStockAccumulates(t *testing.T) {
	dir, base, fix := strengthenRepo(t)
	logPath := t.TempDir() + "/catches.jsonl"
	var server *httptest.Server
	_, log, err := app.NewServer(app.LiveConfig{
		RepoDir: dir, BaseRev: base, FixRev: fix, TipRev: fix, Anchor: anchor(),
		TestCmd: goTestCmd, LedgerPath: logPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	// Two SEQUENTIAL connects against the one default liveReg entry, each running
	// the SAME repo/base/fix/anchor cycle. Both reach a Catch verdict, but they
	// reproduce the SAME catch identity (same revs + anchor), so the identity-dedup
	// gate (#16e) mints it ONCE — the second connect re-runs the same work and is
	// an honest no-op, not a second credit. This is the latent double-mint the
	// gate closes: minting per-connect would have farmed one catch into two.
	for i := 0; i < 2; i++ {
		tc := vt.NewClient(t, server, "/")
		frames, cancel := tc.SSE()
		vt.AwaitFrame(t, frames, 60*time.Second, `data-state="catch"`)
		cancel()
	}

	recs, err := log.Records()
	require.NoError(t, err)
	assert.Len(t, recs, 1, "two connects on the SAME cycle mint ONE catch — the re-run reproduces the same identity and is deduped, never double-counted")
}

func BenchmarkConcurrentCycle(b *testing.B) {
	for _, K := range []int{1, 2, 4} {
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
		})
	}
}
