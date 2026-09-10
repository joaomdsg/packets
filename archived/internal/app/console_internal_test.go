package app

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/pipe"
	"github.com/joaomdsg/packets/internal/reanchor"
)

// The needs-you/hero/settled regions must stay live even when the fold-
// visible fact that changes them is a catch minting AFTER an order's status
// already settled to "done" — no status or question change accompanies it,
// so a signature keyed only on those facts would miss the transition. NOT
// parallel (shared liveReg/liveFabric).
func TestLiveCard_heroStatRefreshesLiveWhenACatchMintsWithoutAStatusOrQuestionChange(t *testing.T) {
	restore := resolveCycle
	t.Cleanup(func() { resolveCycle = restore })
	resolveCycle = func(_ context.Context, _, _, _, _ string, _ reanchor.Anchor, _ []string, _, _ bool, _ chan<- pipe.TraceEvent) (Resolution, error) {
		return Resolution{Verdict: string(catch.NoCatch)}, nil
	}

	logPath := filepath.Join(t.TempDir(), "catches.jsonl")
	var server *httptest.Server
	viaApp, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: logPath,
	})
	require.NoError(t, err)
	server = httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = log.Close() })

	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 1, ReasonTag: "catch"}))
	require.NoError(t, log.AppendSend("d1", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "alpha.go", Line: 7}, own))
	require.NoError(t, log.AppendStatus(1, "done")) // done, not yet caught — held, not verified

	tc := vt.NewClient(t, server, "/")
	frames, cancel := tc.SSE()
	defer cancel()
	vt.AwaitFrame(t, frames, 10*time.Second, `console__hero-stat">0<`)

	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "alpha.go", Line: 7, ReasonTag: "catch", Source: "wo:1"}))

	vt.AwaitFrame(t, frames, 10*time.Second, `console__hero-stat">1<`)
}
