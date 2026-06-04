package app

import (
	"context"
	"errors"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-via/via"
	"github.com/go-via/via/vt"

	"github.com/joaomdsg/agntpr/internal/catch"
	"github.com/joaomdsg/agntpr/internal/ledger"
	"github.com/joaomdsg/agntpr/internal/pipe"
	"github.com/joaomdsg/agntpr/internal/reanchor"
)

func storeMax(p *int64, v int64) {
	for {
		old := atomic.LoadInt64(p)
		if v <= old || atomic.CompareAndSwapInt64(p, old, v) {
			return
		}
	}
}

func TestLiveServer_capsConcurrentCyclesAtMaxConcurrentWithoutDroppingAny(t *testing.T) {
	// Internal test (package app): it swaps the unexported resolveCycle seam, so it
	// cannot be an external pkg_test. NOT parallel — it shares the package-var
	// liveState and the global seam with the other live tests.
	const cap = 2
	const connects = 4

	var inflight, peak int64
	entered := make(chan struct{}, connects)
	release := make(chan struct{})

	restore := resolveCycle
	t.Cleanup(func() { resolveCycle = restore })
	resolveCycle = func(_ context.Context, _, _, _, _ string, _ reanchor.Anchor, _ []string, _, _ bool, _ chan<- pipe.TraceEvent) (Resolution, error) {
		storeMax(&peak, atomic.AddInt64(&inflight, 1))
		entered <- struct{}{}
		<-release // hold the slot until the test lets every admitted cycle go
		atomic.AddInt64(&inflight, -1)
		return Resolution{Verdict: string(catch.Catch), Record: &ledger.CatchRecord{Outcome: catch.Catch, ReasonTag: "catch"}}, nil
	}

	logPath := filepath.Join(t.TempDir(), "catches.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: logPath, MaxConcurrent: cap,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	// Each SSE subscription triggers an OnConnect → one cycle goroutine that
	// contends on the admission semaphore. Hold the frames + cancels so the live
	// connections (and their cycles) stay alive for the whole test.
	for i := 0; i < connects; i++ {
		tc := vt.NewClient(t, server, "/")
		_, cancel := tc.SSE()
		t.Cleanup(cancel)
	}

	for i := 0; i < cap; i++ {
		<-entered // exactly cap cycles are admitted; the rest block on acquire
	}
	assert.Equal(t, int64(cap), atomic.LoadInt64(&inflight), "the surplus connects block on acquire — only the cap is in flight")
	assert.Equal(t, int64(cap), atomic.LoadInt64(&peak), "concurrency never exceeded the cap while the slots were held")

	close(release) // let everything drain — the queued connects now acquire freed slots

	require.Eventually(t, func() bool {
		recs, e := log.Records()
		return e == nil && len(recs) == connects
	}, 10*time.Second, 5*time.Millisecond, "every connect's cycle eventually runs — the queue drains, it never drops a cycle")

	assert.Equal(t, int64(cap), atomic.LoadInt64(&peak), "peak in-flight never exceeded MaxConcurrent across the whole run — the lane is genuinely scarce")
}

func TestLiveServer_releasesTheSlotWhenACycleErrorsSoTheQueueNeverStalls(t *testing.T) {
	const cap = 1
	const connects = 2
	entered := make(chan struct{}, connects)

	restore := resolveCycle
	t.Cleanup(func() { resolveCycle = restore })
	resolveCycle = func(_ context.Context, _, _, _, _ string, _ reanchor.Anchor, _ []string, _, _ bool, _ chan<- pipe.TraceEvent) (Resolution, error) {
		entered <- struct{}{}
		return Resolution{}, errors.New("cycle failed") // an error must STILL release the slot
	}

	logPath := filepath.Join(t.TempDir(), "catches.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: logPath, MaxConcurrent: cap,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	for i := 0; i < connects; i++ {
		tc := vt.NewClient(t, server, "/")
		_, cancel := tc.SSE()
		t.Cleanup(cancel)
	}

	// With cap=1 the second cycle is queued. The first errors — and MUST release
	// its slot — for the second to ever run. A leaked slot shows up as the second
	// `entered` never arriving (a clear timeout), not a silent hang.
	for i := 0; i < connects; i++ {
		select {
		case <-entered:
		case <-time.After(20 * time.Second):
			t.Fatalf("only %d of %d cycles ran — an errored cycle leaked its slot", i, connects)
		}
	}
}

func anchorForCap() reanchor.Anchor {
	return reanchor.Anchor{Path: "adult.go", Start: 4, End: 4, LineHash: "x"}
}
