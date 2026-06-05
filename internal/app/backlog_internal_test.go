package app

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via"
	"github.com/go-via/via/vt"

	"github.com/joaomdsg/agntpr/internal/catch"
	"github.com/joaomdsg/agntpr/internal/ledger"
	"github.com/joaomdsg/agntpr/internal/pipe"
	"github.com/joaomdsg/agntpr/internal/reanchor"
)

func woTargetN(i int) ledger.Target {
	s := strconv.Itoa(i)
	return ledger.Target{BaseRev: "wo-base-" + s, FixRev: "wo-fix-" + s, TipRev: "wo-fix-" + s, Path: "other.go", Line: 9 + i}
}

func catchForTarget(t ledger.Target) *ledger.CatchRecord {
	return &ledger.CatchRecord{Outcome: catch.Catch, Path: t.Path, Line: t.Line, BeforeRev: t.BaseRev, AfterRev: t.FixRev, ReasonTag: "catch"}
}

func TestLiveCard_spendsDrawDistinctWorkFromTheBacklogThenExhaustHonestly(t *testing.T) {
	// The work-source: a Spend pulls the NEXT unconsumed backlog target (head-first),
	// so two spends fund TWO DISTINCT runs that each mint back — the loop compounds
	// more than once. Once the backlog is drawn down, a further Spend is an HONEST
	// no-op (balance unchanged, no order appended), not a silent dedup'd loss.
	restore := resolveCycle
	t.Cleanup(func() { resolveCycle = restore })
	t1, t2 := woTargetN(1), woTargetN(2)
	resolveCycle = func(_ context.Context, _, base, _, _ string, _ reanchor.Anchor, _ []string, _, _ bool, _ chan<- pipe.TraceEvent) (Resolution, error) {
		switch base {
		case t1.BaseRev:
			return Resolution{Verdict: string(catch.Catch), Record: catchForTarget(t1)}, nil
		case t2.BaseRev:
			return Resolution{Verdict: string(catch.Catch), Record: catchForTarget(t2)}, nil
		}
		return Resolution{}, nil // the connect-cycle mints nothing
	}

	logPath := filepath.Join(t.TempDir(), "catches.jsonl")
	seed, err := ledger.Open(logPath)
	require.NoError(t, err)
	require.NoError(t, seed.Append(ledger.CatchRecord{Outcome: catch.Catch, Line: 1, ReasonTag: "catch"}))
	require.NoError(t, seed.Append(ledger.CatchRecord{Outcome: catch.Catch, Line: 2, ReasonTag: "catch"}))
	require.NoError(t, seed.Close())

	cfg := LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: logPath, DispatchBacklog: []ledger.Target{t1, t2},
	}
	var server *httptest.Server
	_, log, err := NewServer(cfg, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	tc := vt.NewClient(t, server, "/")
	frames, cancel := tc.SSE()
	defer cancel()
	vt.AwaitFrame(t, frames, 10*time.Second, `data-balance="2"`)

	require.Equal(t, 200, tc.Action((&LiveCard{}).Spend).Fire()) // funds T1
	require.Equal(t, 200, tc.Action((&LiveCard{}).Spend).Fire()) // funds T2 (head advances — NOT T1 again)

	require.Eventually(t, func() bool {
		c, e := log.DispatchStatusCounts()
		return e == nil && c.Done == 2
	}, 10*time.Second, 10*time.Millisecond, "both backlog targets ran to done")

	recs, err := log.Records()
	require.NoError(t, err)
	reinvested := map[string]bool{}
	for _, r := range recs {
		if r.Producer == "wo:1" || r.Producer == "wo:2" {
			reinvested[r.BeforeRev] = true
		}
	}
	require.Len(t, reinvested, 2, "two DISTINCT targets minted back — the mint ties to distinct work consumed, never to dispatch acts")

	pendingBefore, err := log.PendingDispatches()
	require.NoError(t, err)
	balBefore, err := log.Balance()
	require.NoError(t, err)

	require.Equal(t, 200, tc.Action((&LiveCard{}).Spend).Fire()) // backlog exhausted → honest no-op (no runner launched)

	pendingAfter, err := log.PendingDispatches()
	require.NoError(t, err)
	require.Equal(t, pendingBefore, pendingAfter, "an exhausted backlog funds NO further order — not a silent dedup loss")
	balAfter, err := log.Balance()
	require.NoError(t, err)
	require.Equal(t, balBefore, balAfter, "the exhausted-backlog spend debited nothing — balance unchanged")
}

func TestNextUnconsumedTarget_isLogDerivedAndSurvivesReopen(t *testing.T) {
	// Consumption is a pure projection of the persisted log (a target is consumed
	// once a funded work-order carries it), so the head advances deterministically
	// and a reopen re-derives the same exhaustion — no in-memory head pointer.
	t1, t2 := woTargetN(1), woTargetN(2)
	logPath := filepath.Join(t.TempDir(), "catches.jsonl")
	l, err := ledger.Open(logPath)
	require.NoError(t, err)
	require.NoError(t, l.Append(ledger.CatchRecord{Outcome: catch.Catch, Line: 1, ReasonTag: "catch"}))
	require.NoError(t, l.Append(ledger.CatchRecord{Outcome: catch.Catch, Line: 2, ReasonTag: "catch"}))
	cfg := LiveConfig{DispatchBacklog: []ledger.Target{t1, t2}}

	got, ok := nextUnconsumedTarget(cfg, l)
	require.True(t, ok)
	require.Equal(t, t1, got, "head-first: the first unconsumed target is T1")
	require.NoError(t, l.AppendDispatch("dispatch", t1, ownTargetOf(cfg)))
	got, ok = nextUnconsumedTarget(cfg, l)
	require.True(t, ok)
	require.Equal(t, t2, got, "after T1 is funded it is consumed — the head advances to T2")
	require.NoError(t, l.AppendDispatch("dispatch", t2, ownTargetOf(cfg)))
	_, ok = nextUnconsumedTarget(cfg, l)
	require.False(t, ok, "both consumed → backlog exhausted")
	require.NoError(t, l.Close())

	reopened, err := ledger.Open(logPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = reopened.Close() })
	_, ok = nextUnconsumedTarget(cfg, reopened)
	require.False(t, ok, "exhaustion is log-derived — it survives a reopen with no in-memory head pointer")
}

func TestNextUnconsumedTarget_emptyBacklogIsExhausted(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "catches.jsonl")
	l, err := ledger.Open(logPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = l.Close() })
	_, ok := nextUnconsumedTarget(LiveConfig{}, l)
	require.False(t, ok, "no backlog → no distinct work to fund")
}

func TestNextUnconsumedTarget_skipsTheCardsOwnWorkSoItCannotStallTheHead(t *testing.T) {
	// A backlog entry equal to the card's OWN caught cycle would be refused by
	// AppendDispatch and so never become consumed — left in the backlog it would
	// stall the head forever, starving the targets behind it. nextUnconsumedTarget
	// must skip own work and advance to the next fundable distinct target.
	t2 := woTargetN(2)
	cfg := LiveConfig{BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap()}
	own := ownTargetOf(cfg)
	cfg.DispatchBacklog = []ledger.Target{own, t2}

	logPath := filepath.Join(t.TempDir(), "catches.jsonl")
	l, err := ledger.Open(logPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = l.Close() })

	got, ok := nextUnconsumedTarget(cfg, l)
	require.True(t, ok)
	require.Equal(t, t2, got, "own work is skipped (it could never be funded) — the head advances past it")
}
