package app

import (
	"context"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/agntpr/internal/catch"
	"github.com/joaomdsg/agntpr/internal/ledger"
	"github.com/joaomdsg/agntpr/internal/pipe"
	"github.com/joaomdsg/agntpr/internal/reanchor"
)

func TestDrainQueuedOrders_terminatesWhenAStatusWriteFailsPermanently(t *testing.T) {
	// A funded order whose status can NEVER advance (here: the ledger write handle
	// is closed, so every AppendStatus fails while reads via scan still work) must
	// not spin the runner forever re-running the suite under a held runMu. The
	// per-order attempts cap bounds it: the drain RETURNS and the cycle fires at
	// most maxOrderAttempts times. NOT parallel (shared globals).
	restore := resolveCycle
	t.Cleanup(func() { resolveCycle = restore })
	var calls int64
	resolveCycle = func(_ context.Context, _, _, _, _ string, _ reanchor.Anchor, _ []string, _, _ bool, _ chan<- pipe.TraceEvent) (Resolution, error) {
		atomic.AddInt64(&calls, 1)
		return Resolution{}, nil
	}

	logPath := filepath.Join(t.TempDir(), "catches.jsonl")
	log, err := ledger.Open(logPath)
	require.NoError(t, err)
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Line: 1, ReasonTag: "catch"}))
	require.NoError(t, log.AppendDispatch("dispatch", woTargetN(1), ledger.Target{})) // fund order 1 while the handle is open
	registerSession("term-key", LiveConfig{RepoDir: ".", TestCmd: []string{"true"}, LedgerPath: logPath}, log)
	require.NoError(t, log.Close()) // now every AppendStatus write fails; QueuedWorkOrders (its own read handle) still sees the queued order

	done := make(chan struct{})
	go func() {
		drainQueuedOrders("term-key")
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("drainQueuedOrders did not terminate — the runner spins on a permanently-failing status write")
	}
	require.LessOrEqual(t, atomic.LoadInt64(&calls), int64(maxOrderAttempts), "the cycle fires at most maxOrderAttempts times for a stuck order — bounded, never an unbounded suite-exec burn")
}
