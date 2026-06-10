package app

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via"
	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/pipe"
	"github.com/joaomdsg/packets/internal/reanchor"
)

// Before clicking Spend the Lead should know WHAT it funds, not act blind. The
// control names the actual next target nextUnconsumedTarget would pick — turning a
// blind verb into an informed choice. NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_spendControlPreviewsTheNextTargetItWouldFund(t *testing.T) {
	restore := resolveCycle
	t.Cleanup(func() { resolveCycle = restore })
	resolveCycle = func(_ context.Context, _, _, _, _ string, _ reanchor.Anchor, _ []string, _, _ bool, _ chan<- pipe.TraceEvent) (Resolution, error) {
		return Resolution{}, nil // no mint — isolate the balance to the seeded catch
	}

	logPath := filepath.Join(t.TempDir(), "catches.jsonl")
	var server *httptest.Server
	// A hand-seeded backlog target is the head of the fundable queue, so the
	// preview names it exactly.
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: logPath,
		DispatchBacklog: []ledger.Target{{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "preview.go", Line: 42}},
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Line: 1, ReasonTag: "catch"}))

	body := vt.NewClient(t, server, "/").HTML()
	require.Contains(t, body, "fund preview.go:42",
		"the Spend control previews the exact target the next spend would fund")
}

// spendButtonLabel must name the real next target when there is fundable work, and
// fall back to a generic label only when there is none — covering both branches
// (the no-target branch is near-unreachable through the UI, so it is locked here at
// the pure-function level).
func TestSpendButtonLabel_namesTheNextTargetOrFallsBackWhenNoneFundable(t *testing.T) {
	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })

	// No fundable work (empty log, empty config) → the honest generic fallback.
	empty := ledger.Bind(f, "empty", "i")
	require.Equal(t, "Spend a catch → fund a work-order", spendButtonLabel(LiveConfig{}, empty))

	// A fundable target present → the label names it exactly.
	cfg := LiveConfig{DispatchBacklog: []ledger.Target{{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "alpha.go", Line: 8}}}
	withWork := ledger.Bind(f, "withwork", "i")
	require.Equal(t, "Spend a catch → fund alpha.go:8", spendButtonLabel(cfg, withWork))
}
