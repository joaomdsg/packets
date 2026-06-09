package ledger_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/ledger"
)

// THE TWO-SCORES INVARIANT: a pending claim is a gray BET in flight, never a
// confirmed catch. It must count toward ClaimsInFlight and NEVER toward Balance,
// and on minting it MOVES from one to the other — never double-counted. Getting
// this wrong would show a producer's unverified bet as a real score.
func TestClaimsInFlight_pendingIsABetNeverCountedAsAConfirmedCatch(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := isolatedFab(t)
	log := ledger.Bind(f, "s", "i")

	require.NoError(t, mustPublishClaim(ctx, f, claimAt(4)))

	inflight, err := log.ClaimsInFlight()
	require.NoError(t, err)
	require.Equal(t, 1, inflight, "a published-but-unverified claim is one bet in flight")
	require.True(t, balanceIs(log, 0)(), "a pending claim must NOT count as a confirmed catch")

	// Mint it directly (synchronous) — the verifier confirmed it.
	rec, err := confirmFromClaim(claimAt(4))
	require.NoError(t, err)
	require.NoError(t, log.Append(*rec))

	inflight, err = log.ClaimsInFlight()
	require.NoError(t, err)
	require.Equal(t, 0, inflight, "a minted target is resolved — no longer in flight")
	require.True(t, balanceIs(log, 1)(), "now it is a confirmed catch")
}

// Distinct claim targets each count once as in flight.
func TestClaimsInFlight_countsEachDistinctUnmintedTarget(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := isolatedFab(t)
	log := ledger.Bind(f, "s", "i")

	require.NoError(t, mustPublishClaim(ctx, f, claimAt(4)))
	require.NoError(t, mustPublishClaim(ctx, f, claimAt(5)))

	inflight, err := log.ClaimsInFlight()
	require.NoError(t, err)
	require.Equal(t, 2, inflight, "two distinct unminted targets are two bets in flight")
}

// A producer replaying the SAME target is one unit of work, not two — the
// in-flight count must dedupe by target identity, not by raw claim events.
func TestClaimsInFlight_dedupesReplayedTargets(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := isolatedFab(t)
	log := ledger.Bind(f, "s", "i")

	require.NoError(t, mustPublishClaim(ctx, f, claimAt(4)))
	require.NoError(t, mustPublishClaim(ctx, f, claimAt(4))) // replay of the same target

	inflight, err := log.ClaimsInFlight()
	require.NoError(t, err)
	require.Equal(t, 1, inflight, "a replayed target is one bet, not two")
}

// A malformed event on the claim subtree is not a claim in flight: ClaimsInFlight
// skips it (no error, no miscount) so one garbage publish can't break the count.
func TestClaimsInFlight_skipsMalformedClaimEvents(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := isolatedFab(t)
	log := ledger.Bind(f, "s", "i")

	_, err := f.Publish(ctx, fabric.EventSubject("s", "i", fabric.StatusClaim, "work"), []byte("not json"))
	require.NoError(t, err)
	require.NoError(t, mustPublishClaim(ctx, f, claimAt(4)))

	inflight, err := log.ClaimsInFlight()
	require.NoError(t, err, "a malformed claim event must be skipped, not error the count")
	require.Equal(t, 1, inflight, "only the one valid claim is in flight")
}

// Two claims for the SAME catch identity (base/fix/path/line) but a different
// trunk TipRev are one unit of work, not two — TipRev is not part of the catch
// identity (matching Append's dedupe). So they count once in flight. Pins that
// the dedupe keys on the identity tuple, not the full Target.
func TestClaimsInFlight_dedupesSameIdentityAcrossDifferentTipRevs(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := isolatedFab(t)
	log := ledger.Bind(f, "s", "i")

	a := ledger.ClaimRecord{Target: ledger.Target{BaseRev: "base", FixRev: "fix", TipRev: "tipA", Path: "a.go", Line: 4}}
	b := ledger.ClaimRecord{Target: ledger.Target{BaseRev: "base", FixRev: "fix", TipRev: "tipB", Path: "a.go", Line: 4}}
	_, err := ledger.PublishClaim(ctx, f, "s", "i", a)
	require.NoError(t, err)
	_, err = ledger.PublishClaim(ctx, f, "s", "i", b)
	require.NoError(t, err)

	inflight, err := log.ClaimsInFlight()
	require.NoError(t, err)
	require.Equal(t, 1, inflight, "same catch identity over two tips is one bet — dedupe ignores TipRev")
}

// A fresh log has nothing in flight.
func TestClaimsInFlight_isZeroWithNoClaims(t *testing.T) {
	t.Parallel()
	f := isolatedFab(t)
	log := ledger.Bind(f, "s", "i")

	inflight, err := log.ClaimsInFlight()
	require.NoError(t, err)
	require.Equal(t, 0, inflight, "no claims published → nothing in flight")
}
