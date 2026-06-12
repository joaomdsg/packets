package app

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via"
	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/ledger"
)

// The board must surface a session's claims IN FLIGHT (producers' pending bets)
// as their own count, distinct from confirmed catches — never folded into the
// confirmed stock (the two-scores invariant on the fleet surface). NOT parallel
// (shared liveReg).
func TestBoardCard_showsClaimsInFlightDistinctFromConfirmed(t *testing.T) {
	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "cif", "i")

	// Two confirmed catches (distinct identities, on a different anchor than the
	// claims so the claims aren't seen as already-minted).
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 100, ReasonTag: "catch"}))
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 101, ReasonTag: "catch"}))
	// Three distinct unminted claims → three bets in flight.
	for i := 1; i <= 3; i++ {
		_, err := ledger.PublishClaim(ctx, f, "cif", "i", ledger.ClaimRecord{Target: ledger.Target{
			BaseRev: "b", FixRev: "fx", TipRev: "fx", Path: "a.go", Line: i,
		}})
		require.NoError(t, err)
	}
	registerSession("cif", LiveConfig{BaseRev: "own-b-cif", FixRev: "own-f", Anchor: anchorForCap()}, log)

	// Data: the row carries in-flight SEPARATELY from confirmed — pending bets
	// never inflate the confirmed stock.
	rows := BoardRows()
	r := rows[rowIndex(rows, "cif")]
	require.Equal(t, 3, r.InFlight, "three distinct unminted claims are three bets in flight")
	require.Equal(t, 2, r.Confirmed, "claims in flight must NOT be counted as confirmed catches")

	// Render: the board shows the in-flight count in its own span.
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = defLog.Close() })

	body := vt.NewClient(t, server, "/board").HTML()
	require.Contains(t, body, "in flight", "the board surfaces the claims-in-flight bet count")
	require.Contains(t, body, "board-row__inflight", "in-flight renders in its own span, distinct from the confirmed stock span")
	require.Contains(t, strings.ToLower(body), "3 in flight", "the cif row shows its three bets in flight")
}

// A producer's bet that the host VERIFIED and rejected is a resolved loss: it
// must surface on the board as its OWN count, distinct from pending in-flight
// bets and from confirmed catches. Otherwise a rejection is silently invisible
// (lie-green) — a bet leaves the in-flight count and shows up nowhere. NOT
// parallel (shared liveReg).
func TestBoardCard_showsVerifiedLostDistinctFromInFlightAndConfirmed(t *testing.T) {
	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "vl", "i")

	// One confirmed catch on a different anchor than the claims.
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 100, ReasonTag: "catch"}))
	// Three distinct claims; two of them are then verified-and-rejected, one stays pending.
	targets := make([]ledger.Target, 0, 3)
	for i := 1; i <= 3; i++ {
		tgt := ledger.Target{BaseRev: "b", FixRev: "fx", TipRev: "fx", Path: "a.go", Line: i}
		_, err := ledger.PublishClaim(ctx, f, "vl", "i", ledger.ClaimRecord{Target: tgt})
		require.NoError(t, err)
		targets = append(targets, tgt)
	}
	for _, tgt := range targets[:2] { // reject the first two
		_, err := ledger.PublishClaimVerdict(ctx, f, "vl", "i", ledger.ClaimVerdict{Target: tgt, Rejected: true})
		require.NoError(t, err)
	}
	registerSession("vl", LiveConfig{BaseRev: "own-b-vl", FixRev: "own-f", Anchor: anchorForCap()}, log)

	rows := BoardRows()
	r := rows[rowIndex(rows, "vl")]
	require.Equal(t, 2, r.Rejected, "two verified-and-rejected bets are two verified-losses")
	require.Equal(t, 1, r.InFlight, "the rejected bets left flight (C3a); only the one pending bet remains in flight")
	require.Equal(t, 1, r.Confirmed, "verified-losses must NOT be counted as confirmed catches")

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = defLog.Close() })

	body := vt.NewClient(t, server, "/board").HTML()
	require.Contains(t, body, "verified-lost", "the board surfaces the verified-loss count")
	require.Contains(t, body, "board-row__rejected", "verified-lost renders in its own span, distinct from in-flight and confirmed")
	require.Contains(t, strings.ToLower(body), "2 verified-lost", "the vl row shows its two verified-losses")
}

// A pending/lost BET must not blend into the confirmed CATCH tally at a glance:
// the bet lifecycle (in-flight + verified-lost) is sealed into one explicitly-
// labelled "bets" cluster, structurally distinct from the confirmed stock span.
// This carries the two-scores separation by STRUCTURE, not by hoping a reader
// parses each label — without a stylesheet (the repo has no CSS). NOT parallel
// (shared liveReg).
func TestBoardCard_sealsTheBetLifecycleIntoOneClusterApartFromConfirmedStock(t *testing.T) {
	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "bc", "i")

	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 100, ReasonTag: "catch"}))
	for i := 1; i <= 2; i++ {
		tgt := ledger.Target{BaseRev: "b", FixRev: "fx", TipRev: "fx", Path: "a.go", Line: i}
		_, err := ledger.PublishClaim(ctx, f, "bc", "i", ledger.ClaimRecord{Target: tgt})
		require.NoError(t, err)
		if i == 1 { // reject one so both bet states are present
			_, err := ledger.PublishClaimVerdict(ctx, f, "bc", "i", ledger.ClaimVerdict{Target: tgt, Rejected: true})
			require.NoError(t, err)
		}
	}
	registerSession("bc", LiveConfig{BaseRev: "own-b-bc", FixRev: "own-f", Anchor: anchorForCap()}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = defLog.Close() })

	// Scope to the rendered BODY: the base stylesheet in <head> legitimately
	// contains the class names as CSS selectors, which would otherwise fool the
	// structural index assertions below. This test is about body STRUCTURE.
	body := bodyOf(vt.NewClient(t, server, "/board").HTML())

	// The grouping cluster + its explicit "bets:" label exist.
	require.Contains(t, body, "board-row__bets", "the bet lifecycle is sealed in its own grouping container")
	require.Contains(t, body, "board-row__bets-label", "the bets cluster is explicitly labelled")
	require.Contains(t, strings.ToLower(body), "bets:", "the cluster carries the 'bets:' boundary so a bet can't blend into the stock")

	// Structure: the confirmed stock span comes BEFORE the bets cluster (outside
	// it), and both bet spans come AFTER the cluster opens (inside it) — so the
	// caught stock and the pending/lost bets are not interleaved into one tally.
	iStock := strings.Index(body, "board-row__stock")
	iBets := strings.Index(body, "board-row__bets")
	iInflight := strings.Index(body, "board-row__inflight")
	iRejected := strings.Index(body, "board-row__rejected")
	require.Greater(t, iStock, -1)
	require.Greater(t, iBets, iStock, "the confirmed stock renders before (outside) the bets cluster")
	require.Greater(t, iInflight, iBets, "in-flight is inside the bets cluster")
	require.Greater(t, iRejected, iBets, "verified-lost is inside the bets cluster")

	// True CONTAINMENT, not mere ordering: both bet spans must fall before the
	// bets container closes — a flat sibling layout that merely orders a "bets:"
	// span ahead of the counts would NOT seal them, defeating the grouping.
	betsOpen := strings.Index(body, `class="board-row__bets"`)
	require.Greater(t, betsOpen, -1, "the bets cluster is a real container element")
	betsClose := betsOpen + strings.Index(body[betsOpen:], "</div>")
	require.Greater(t, betsClose, betsOpen, "the bets container has a closing tag")
	require.Less(t, iInflight, betsClose, "in-flight closes inside the bets container, not as an outside sibling")
	require.Less(t, iRejected, betsClose, "verified-lost closes inside the bets container, not as an outside sibling")
	require.Greater(t, strings.Index(body, "board-row__balance"), betsClose, "the non-bet spans (balance, …) resume AFTER the cluster closes")

	// The inner hooks are unchanged (future color seam) and still carry counts.
	require.Contains(t, strings.ToLower(body), "1 in flight")
	require.Contains(t, strings.ToLower(body), "1 verified-lost")
	require.Contains(t, body, "1 confirmed", "the caught stock is still its own count")
}

// The board makes the funded work-order ROUND-TRIP legible: each session's recent
// dispatches show their target and CAUGHT/MISSED outcome, distinct from the
// confirmed stock and the pending-bets cluster. This is VISION's "watch a funded
// order resolve" — honest per-order outcomes, never a fabricated rank. NOT parallel.
func TestBoardCard_showsRecentDispatchesWithCaughtOrMissedOutcome(t *testing.T) {
	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "disp", "i")

	// Fund two dispatches (two catches → balance 2), run both: WO#1 mints (caught),
	// WO#2 does not (missed).
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 100, ReasonTag: "catch"}))
	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 101, ReasonTag: "catch"}))
	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	require.NoError(t, log.AppendDispatch("d1", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "alpha.go", Line: 7}, own))
	require.NoError(t, log.AppendDispatch("d2", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "beta.go", Line: 9}, own))
	require.NoError(t, log.AppendStatus(1, "done"))
	require.NoError(t, log.AppendStatus(2, "done"))
	caught := ledger.CatchRecord{Outcome: catch.Catch, Path: "alpha.go", Line: 7, ReasonTag: "catch", Producer: "wo:1"}
	require.NoError(t, log.Append(caught))
	registerSession("disp", LiveConfig{BaseRev: "own-b-disp", FixRev: "own-f", Anchor: anchorForCap()}, log)

	r := BoardRows()[rowIndex(BoardRows(), "disp")]
	require.Len(t, r.Dispatches, 2, "both funded orders are surfaced")

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = defLog.Close() })

	body := vt.NewClient(t, server, "/board").HTML()
	require.Contains(t, body, "board-row__dispatches", "recent dispatches render in their own cluster")
	require.Contains(t, body, "pk-chip board-row__dispatch", "the read-only dispatch line composes the mono .pk-chip")
	require.Contains(t, body, "WO#1", "the caught order is shown by id")
	require.Contains(t, body, "alpha.go:7", "with its target")
	require.Contains(t, body, "caught", "WO#1 minted → caught")
	require.Contains(t, body, "WO#2", "the missed order is shown too")
	require.Contains(t, body, "missed", "WO#2 ran but minted nothing → missed")
}

// bodyOf returns the <body> portion of a rendered page, dropping the <head>.
// Structural index assertions must scope to the body: the base stylesheet in the
// head contains class names as CSS selectors, which would otherwise be matched by
// a whole-page substring index. (Plain Contains checks are unaffected.)
func bodyOf(html string) string {
	if i := strings.Index(html, "</head>"); i >= 0 {
		return html[i:]
	}
	return html
}
