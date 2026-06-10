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

// The Spend action — the central economic move of the whole product (turn a
// confirmed catch into a funded work-order) — is dead unless the card RENDERS a
// control bound to it. Without a rendered trigger the Lead can read the balance
// but can never act on it, so the loop never closes from the UI. When there is
// balance to spend, the card must render a control wired to the Spend action.
// NOT parallel (shared globals).
func TestLiveCard_rendersASpendControlWhenThereIsBalanceToSpend(t *testing.T) {
	restore := resolveCycle
	t.Cleanup(func() { resolveCycle = restore })
	resolveCycle = func(_ context.Context, _, _, _, _ string, _ reanchor.Anchor, _ []string, _, _ bool, _ chan<- pipe.TraceEvent) (Resolution, error) {
		return Resolution{}, nil // no mint — isolate the balance to the seeded catches
	}

	logPath := filepath.Join(t.TempDir(), "catches.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: logPath,
		DispatchBacklog: []ledger.Target{woDispatchTarget()},
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Line: 1, ReasonTag: "catch"}))

	body := bodyOf(vt.NewClient(t, server, "/").HTML())
	// The control must POST the real Spend action — the same one the existing spend
	// tests fire — so the rendered button actually drives the economy, not a dead
	// label. via renders an action binding as @post('/_action/Spend').
	require.Contains(t, body, "/_action/Spend",
		"the card must render a control bound to the Spend action so the Lead can fund work from the UI")
	require.Contains(t, body, "spend-action",
		"the spend control carries its class hook so the stylesheet can style it")
	// The control names the honest move it makes — spend a catch to fund work — so
	// the Lead knows what the click does, not a bare verb. (The exact target named
	// is locked by the spend-preview tests; here we only assert the spend-and-fund
	// framing is present.)
	require.Contains(t, body, "Spend a catch → fund ",
		"the spend control names the real outcome of the click")
}

// Offering a "Spend" control with an empty balance is dishonest — there is
// nothing to spend, and a click is a silent no-op. A session with no balance must
// NOT render the spend control. NOT parallel (shared globals).
func TestLiveCard_hidesTheSpendControlWhenBalanceIsZero(t *testing.T) {
	restore := resolveCycle
	t.Cleanup(func() { resolveCycle = restore })
	resolveCycle = func(_ context.Context, _, _, _, _ string, _ reanchor.Anchor, _ []string, _, _ bool, _ chan<- pipe.TraceEvent) (Resolution, error) {
		return Resolution{}, nil // no mint — the balance stays 0
	}

	logPath := filepath.Join(t.TempDir(), "catches.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: logPath,
		DispatchBacklog: []ledger.Target{woDispatchTarget()},
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	body := bodyOf(vt.NewClient(t, server, "/").HTML())
	// Assert on the ACTION BINDING, not the word "Spend" — the onboarding copy
	// legitimately contains the word but binds no action.
	require.NotContains(t, body, "/_action/Spend",
		"a zero-balance session must not offer a spend control with nothing to spend")
}

// Spending the LAST catch must retract the control LIVE: View re-evaluates the
// balance > 0 guard on every render, so the drain-to-zero re-render frame must
// drop the spend control — otherwise a dead button lingers, inviting a click that
// is now a silent no-op. This locks the disappearance over SSE (the initial-render
// case above only covers a session that never had balance). NOT parallel (shared
// globals).
func TestLiveCard_spendControlRetractsLiveWhenTheLastCatchIsSpent(t *testing.T) {
	restore := resolveCycle
	t.Cleanup(func() { resolveCycle = restore })
	resolveCycle = func(_ context.Context, _, _, _, _ string, _ reanchor.Anchor, _ []string, _, _ bool, _ chan<- pipe.TraceEvent) (Resolution, error) {
		return Resolution{}, nil // no mint — the only balance movement is the Lead's Spend
	}

	logPath := filepath.Join(t.TempDir(), "catches.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: logPath,
		DispatchBacklog: []ledger.Target{woDispatchTarget()},
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Line: 1, ReasonTag: "catch"}))

	tc := vt.NewClient(t, server, "/")
	frames, cancel := tc.SSE()
	defer cancel()
	// The control is present while there is a catch to spend.
	vt.AwaitFrame(t, frames, 10*time.Second, "/_action/Spend")

	// Spend the lone catch: the drain-to-zero frame must show balance 0 AND no
	// longer carry the spend control.
	require.Equal(t, 200, tc.Action((&LiveCard{}).Spend).Fire())
	drained := vt.AwaitFrame(t, frames, 10*time.Second, `data-balance="0"`)
	require.NotContains(t, drained, "/_action/Spend",
		"the spend control must retract once the last catch is spent — no dead button")
}
