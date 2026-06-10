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
	"github.com/joaomdsg/packets/internal/mutation"
	"github.com/joaomdsg/packets/internal/pipe"
	"github.com/joaomdsg/packets/internal/reanchor"
)

// The dispatch→review tie: when a funded work-order fills, the oracle's surviving
// mutants are its review questions — the test-debt the work left. Today they're
// discarded. Capturing them (off the economy ledger, like the connect-cycle
// findings) and surfacing the count on the order makes a funded order reviewable:
// you don't fire-and-forget; you see what you paid for left open. NOT parallel
// (shared liveReg + the resolveCycle seam).
func TestLiveCard_aFilledOrderShowsItsOpenReviewQuestions(t *testing.T) {
	resetConsumersForTest()
	restore := resolveCycle
	t.Cleanup(func() { resolveCycle = restore })
	resolveCycle = func(_ context.Context, _, _, _, _ string, _ reanchor.Anchor, _ []string, _, _ bool, _ chan<- pipe.TraceEvent) (Resolution, error) {
		return Resolution{Findings: []mutation.Finding{
			{File: "alpha.go", Line: 7, Outcome: mutation.Survived, Message: "mutated >= to >"},
			{File: "alpha.go", Line: 9, Outcome: mutation.Survived, Message: "mutated + to -"},
		}}, nil // no Record → no mint; isolate the diagnostic findings (two-scores)
	}

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "woq", "i")
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 1, ReasonTag: "catch"})) // balance 1
	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	require.NoError(t, log.AppendDispatch("d1", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "alpha.go", Line: 7}, own))
	registerSession("woq", LiveConfig{RepoDir: ".", BaseRev: "own-b", FixRev: "own-f", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	drainQueuedOrders("woq") // sync: fill order 1, capturing its findings

	require.Equal(t, 2, lookupLiveEntry("woq").orderQuestionCount(1),
		"the filled order's review questions are captured (off the economy ledger)")

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = defLog.Close() })

	body := bodyOf(vt.NewClient(t, server, "/?key=woq").HTML())
	require.Contains(t, body, "WO#1", "the filled order is shown on the card")
	require.Contains(t, body, "2 open questions", "with its reviewable test-debt — the dispatch→review tie")
	require.Contains(t, body, "wo=1", "the count DRILLS into that order's review (/review?...&wo=1)")

	// Drill: /review?wo=<id> reviews THAT order's questions (not the session's).
	orderBody := bodyOf(vt.NewClient(t, server, "/review?key=woq&wo=1").HTML())
	require.Contains(t, orderBody, "Reviewing WO#1", "the per-order review names the order")
	require.Contains(t, orderBody, "review-thread", "the order's questions render as anchored threads")
	require.Contains(t, orderBody, "alpha.go:7", "anchored to the order's surviving-mutant line")
	require.Contains(t, orderBody, "question: mutated &gt;= to &gt;", "carrying the order's finding as a question")

	// An order with no captured findings → calm empty per-order state, not the session's.
	emptyBody := bodyOf(vt.NewClient(t, server, "/review?key=woq&wo=999").HTML())
	require.Contains(t, emptyBody, "Reviewing WO#999", "the per-order review names the (unfilled) order")
	require.Contains(t, emptyBody, "No open questions for this order", "a calm empty state for an order with no surviving mutants")
}
