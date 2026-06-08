package app

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via"
	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/pipe"
	"github.com/joaomdsg/packets/internal/reanchor"
)

func TestLiveCard_distinctSessionKeysHaveIsolatedBalances(t *testing.T) {
	// Internal test (package app): swaps the unexported resolveCycle seam + reaches
	// registerSession, so it cannot be external. NOT parallel — shares liveReg +
	// resolveCycle with the other live tests. The property: two session keys are
	// ISOLATED economies — a mint into one never credits the other, a spend on one
	// never debits the other (the R18 farm-denial verdict, per-session enforced).
	restore := resolveCycle
	t.Cleanup(func() { resolveCycle = restore })
	resolveCycle = func(_ context.Context, _, base, _, _ string, _ reanchor.Anchor, _ []string, _, _ bool, _ chan<- pipe.TraceEvent) (Resolution, error) {
		if base == woDispatchTarget().BaseRev {
			return Resolution{}, nil // the dispatched run mints nothing here, so the spend's drain stays observable
		}
		return Resolution{Verdict: string(catch.Catch), Record: &ledger.CatchRecord{Outcome: catch.Catch, ReasonTag: "catch"}}, nil
	}

	dir := t.TempDir()
	defPath := filepath.Join(dir, "default.jsonl")
	var server *httptest.Server
	_, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = defLog.Close() })

	logA := ledger.Bind(liveFabric, "ssnA", ledgerInstance)
	logB := ledger.Bind(liveFabric, "ssnB", ledgerInstance)
	registerSession("ssnA", LiveConfig{RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(), TestCmd: []string{"true"}, DispatchBacklog: []ledger.Target{woDispatchTarget()}}, logA)
	registerSession("ssnB", LiveConfig{RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(), TestCmd: []string{"true"}, DispatchBacklog: []ledger.Target{woDispatchTarget()}}, logB)

	ca := vt.NewClient(t, server, "/?key=ssnA")
	fa, cancelA := ca.SSE()
	defer cancelA()
	vt.AwaitFrame(t, fa, 10*time.Second, `data-state="catch"`)

	cb := vt.NewClient(t, server, "/?key=ssnB")
	fb, cancelB := cb.SSE()
	defer cancelB()
	vt.AwaitFrame(t, fb, 10*time.Second, `data-state="catch"`)

	balA, err := logA.Balance()
	require.NoError(t, err)
	balB, err := logB.Balance()
	require.NoError(t, err)
	require.Equal(t, 1, balA, "ssnA's connect minted exactly one catch into ITS ledger")
	require.Equal(t, 1, balB, "ssnB's connect minted into ITS ledger — not shared, never 2")

	require.Equal(t, 200, ca.Action((&LiveCard{Key: "ssnA"}).Spend).Fire())
	require.Eventually(t, func() bool {
		b, e := logA.Balance()
		return e == nil && b == 0
	}, 10*time.Second, 5*time.Millisecond, "the spend on ssnA drained ITS balance to 0")

	balB, err = logB.Balance()
	require.NoError(t, err)
	require.Equal(t, 1, balB, "ssnB is untouched by ssnA's spend — isolated economies")
}
