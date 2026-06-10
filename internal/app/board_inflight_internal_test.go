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
