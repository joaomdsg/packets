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

	"github.com/joaomdsg/agntpr/internal/catch"
	"github.com/joaomdsg/agntpr/internal/ledger"
	"github.com/joaomdsg/agntpr/internal/pipe"
	"github.com/joaomdsg/agntpr/internal/reanchor"
)

func TestCandidatesFromCatches_derivesADistinctNeighborTargetPerCatch(t *testing.T) {
	t.Parallel()
	logPath := filepath.Join(t.TempDir(), "c.jsonl")
	log, err := ledger.Open(logPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "x.go", Line: 50, BeforeRev: "b0", AfterRev: "f0", ReasonTag: "catch"}))

	cands := candidatesFromCatches(log)
	require.NotEmpty(t, cands, "a confirmed catch proposes a candidate in its neighborhood — work begets candidate work")
	// The candidate is a DISTINCT target (a real oracle question at a new anchor),
	// never a copy of the already-caught line (which the dedup gate would reject).
	for _, c := range cands {
		require.NotEqual(t, 50, c.Line, "the candidate explores a NEW line, not the already-caught one")
		require.Greater(t, c.Line, 50, "the candidate advances FORWARD from the catch (deterministic direction, so the oracle territory is well-defined)")
		require.Equal(t, "x.go", c.Path, "the candidate stays in the caught file's neighborhood")
	}
}

func TestFundableBacklog_regeneratesFromCatchesAfterTheConfigListIsDrained(t *testing.T) {
	t.Parallel()
	logPath := filepath.Join(t.TempDir(), "c.jsonl")
	log, err := ledger.Open(logPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "x.go", Line: 50, BeforeRev: "b0", AfterRev: "f0", ReasonTag: "catch"}))

	// A card with NO config backlog (or a fully-drawn one) still has fundable work:
	// the catch's neighborhood. Today fundableBacklog returns only the config list,
	// so a drained config means a dead end — this asserts the faucet refills.
	cfg := LiveConfig{BaseRev: "own-b", FixRev: "own-f", Anchor: anchorForCap()}
	f := fundableBacklog(cfg, log)
	require.NotEmpty(t, f, "a drained config backlog still yields derived candidate work — supply is a going concern, not a finite puddle")
}

func TestFundableBacklog_dedupsAConfigTargetThatAlsoDerivesFromACatch(t *testing.T) {
	t.Parallel()
	logPath := filepath.Join(t.TempDir(), "c.jsonl")
	log, err := ledger.Open(logPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })
	// A catch at line 49 derives a candidate at line 50; the config list ALSO holds
	// that exact target. It must appear ONCE in the fundable set — else the board's
	// BacklogRemaining double-counts the same distinct work.
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "x.go", Line: 49, BeforeRev: "b0", AfterRev: "f0", ReasonTag: "catch"}))
	dup := ledger.Target{BaseRev: "b0", FixRev: "f0", TipRev: "f0", Path: "x.go", Line: 50}
	cfg := LiveConfig{BaseRev: "own-b", FixRev: "own-f", Anchor: anchorForCap(), DispatchBacklog: []ledger.Target{dup}}

	count := 0
	for _, tgt := range fundableBacklog(cfg, log) {
		if tgt == dup {
			count++
		}
	}
	require.Equal(t, 1, count, "a target in BOTH the config list and from-catch supply appears once — no double-count of the same distinct work")
}

func TestLiveCard_supplyRefillsFromItsOwnCatchesSoSpendNeverSilentlyDeadEnds(t *testing.T) {
	// Integration: a card with a drained config backlog but a confirmed catch must
	// still fund work on Spend — the derived candidate runs, mints a NEW distinct
	// catch, and that catch seeds the NEXT candidate, so BacklogRemaining never
	// reaches a silent zero. NOT parallel (shared globals).
	restore := resolveCycle
	t.Cleanup(func() { resolveCycle = restore })
	resolveCycle = func(_ context.Context, _, base, fix, _ string, anchor reanchor.Anchor, _ []string, _, _ bool, _ chan<- pipe.TraceEvent) (Resolution, error) {
		if anchor.Start >= 50 { // candidate territory (the seed catch + its neighborhood); the connect cycle (line 4) mints nothing
			return Resolution{Verdict: string(catch.Catch), Record: &ledger.CatchRecord{
				Outcome: catch.Catch, Path: anchor.Path, Line: anchor.Start, BeforeRev: base, AfterRev: fix, ReasonTag: "catch",
			}}, nil
		}
		return Resolution{}, nil
	}

	logPath := filepath.Join(t.TempDir(), "c.jsonl")
	seed, err := ledger.Open(logPath)
	require.NoError(t, err)
	require.NoError(t, seed.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "x.go", Line: 50, BeforeRev: "b0", AfterRev: "f0", ReasonTag: "catch"}))
	require.NoError(t, seed.Close())

	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: logPath, // NO DispatchBacklog — config supply is empty
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	before, err := log.Records()
	require.NoError(t, err)

	tc := vt.NewClient(t, server, "/")
	frames, cancel := tc.SSE()
	defer cancel()
	vt.AwaitFrame(t, frames, 10*time.Second, `data-balance="1"`)

	require.Equal(t, 200, tc.Action((&LiveCard{}).Spend).Fire()) // funds a DERIVED candidate (config is empty)
	require.Eventually(t, func() bool {
		recs, e := log.Records()
		return e == nil && len(recs) > len(before) // the derived candidate ran and minted a NEW catch
	}, 10*time.Second, 10*time.Millisecond, "the spend funded derived work that minted back — supply refilled from its own output")
}

func TestLiveCard_aDerivedCandidateReproducingASeenCatchIsAnHonestLoss(t *testing.T) {
	// A derived candidate the oracle resolves to an identity ALREADY in stock funds
	// a real order that RUNS but mints NOTHING (the dedup gate) — spent 1, got 0, an
	// honest loss; the economy never inflates from re-exploring caught ground.
	restore := resolveCycle
	t.Cleanup(func() { resolveCycle = restore })
	seen := ledger.CatchRecord{Outcome: catch.Catch, Path: "x.go", Line: 50, BeforeRev: "b0", AfterRev: "f0", ReasonTag: "catch"}
	resolveCycle = func(_ context.Context, _, _, _, _ string, anchor reanchor.Anchor, _ []string, _, _ bool, _ chan<- pipe.TraceEvent) (Resolution, error) {
		if anchor.Start >= 50 {
			r := seen // every candidate run reproduces the SEEN identity → deduped
			return Resolution{Verdict: string(catch.Catch), Record: &r}, nil
		}
		return Resolution{}, nil
	}

	logPath := filepath.Join(t.TempDir(), "c.jsonl")
	s, err := ledger.Open(logPath)
	require.NoError(t, err)
	require.NoError(t, s.Append(seen))                                                                                                                  // the identity the candidate will reproduce
	require.NoError(t, s.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "x.go", Line: 70, BeforeRev: "b0", AfterRev: "f0", ReasonTag: "catch"})) // a balance to spend
	require.NoError(t, s.Close())

	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: logPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	tc := vt.NewClient(t, server, "/")
	frames, cancel := tc.SSE()
	defer cancel()
	vt.AwaitFrame(t, frames, 10*time.Second, `data-balance="2"`)

	require.Equal(t, 200, tc.Action((&LiveCard{}).Spend).Fire())
	require.Eventually(t, func() bool {
		c, e := log.DispatchStatusCounts()
		return e == nil && c.Done == 1
	}, 10*time.Second, 10*time.Millisecond, "the derived candidate ran to done")

	recs, err := log.Records()
	require.NoError(t, err)
	require.Len(t, recs, 2, "the run reproduced a seen identity → minted NOTHING; the confirmed-catch count is unchanged (honest loss)")
	bal, err := log.Balance()
	require.NoError(t, err)
	require.Equal(t, 1, bal, "spent 1, minted 0 — an honest loss, never inflation from re-exploring caught ground")
}
