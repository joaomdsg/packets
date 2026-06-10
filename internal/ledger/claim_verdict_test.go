package ledger_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/ledger"
)

// A rejected bet must RESOLVE: once the host has verified a claim and found no
// catch, a durable rejection marker takes that target out of the in-flight set —
// otherwise a wrong bet lingers forever and the "N in flight" tally only ever
// grows. It still mints NOTHING (two-scores: a rejected bet is never a confirmed
// score, and now is no longer a pending one either).
func TestClaimsInFlight_aRejectedTargetLeavesTheInFlightSet(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := isolatedFab(t)
	log := ledger.Bind(f, "s", "i")

	require.NoError(t, mustPublishClaim(ctx, f, claimAt(4)))
	inflight, err := log.ClaimsInFlight()
	require.NoError(t, err)
	require.Equal(t, 1, inflight, "before the verdict, the bet is in flight")

	_, err = ledger.PublishClaimVerdict(ctx, f, "s", "i", ledger.ClaimVerdict{Target: claimAt(4).Target, Rejected: true})
	require.NoError(t, err)

	inflight, err = log.ClaimsInFlight()
	require.NoError(t, err)
	require.Equal(t, 0, inflight, "a rejected target is resolved — it is no longer a bet in flight")
	require.True(t, balanceIs(log, 0)(), "a rejected bet mints nothing — never a confirmed score")
}

// A rejection marker rides the SAME claim subtree as the claims themselves, so
// the in-flight projection must not mistake a verdict event for a fresh claim:
// a target with a rejection marker counts as zero in flight, never one, and a
// stray verdict (no matching claim) does not invent a phantom bet.
func TestClaimsInFlight_aRejectionMarkerIsNotMiscountedAsAClaim(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := isolatedFab(t)
	log := ledger.Bind(f, "s", "i")

	// One claim that gets rejected, plus a second distinct claim still pending.
	require.NoError(t, mustPublishClaim(ctx, f, claimAt(4)))
	require.NoError(t, mustPublishClaim(ctx, f, claimAt(5)))
	_, err := ledger.PublishClaimVerdict(ctx, f, "s", "i", ledger.ClaimVerdict{Target: claimAt(4).Target, Rejected: true})
	require.NoError(t, err)

	inflight, err := log.ClaimsInFlight()
	require.NoError(t, err)
	require.Equal(t, 1, inflight, "the rejected target drops out; only the still-pending bet remains in flight — the verdict event itself is not a claim")
}

// A verdict event must NEVER be counted as an in-flight claim. This is the
// uniquely-constraining check on the shared subtree: a lone verdict with no
// matching claim, and Rejected=false (so the rejection-exclusion path cannot
// mask the bug by removing what was wrongly added), must yield ZERO in flight.
// If ClaimsInFlight decoded the verdict payload AS a claim, it would invent a
// phantom bet here.
func TestClaimsInFlight_aVerdictEventIsNeverCountedAsAnInFlightClaim(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := isolatedFab(t)
	log := ledger.Bind(f, "s", "i")

	_, err := ledger.PublishClaimVerdict(ctx, f, "s", "i", ledger.ClaimVerdict{Target: claimAt(9).Target, Rejected: false})
	require.NoError(t, err)

	inflight, err := log.ClaimsInFlight()
	require.NoError(t, err)
	require.Equal(t, 0, inflight, "a verdict event is not a claim — it must never invent a phantom bet in flight")
}

// A malformed verdict event on the shared claim subtree must not break the
// in-flight projection: it is skipped (no error, no miscount), exactly as a
// malformed claim is — one garbage publish can't corrupt the tally.
func TestClaimsInFlight_skipsMalformedVerdictEvents(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := isolatedFab(t)
	log := ledger.Bind(f, "s", "i")

	_, err := f.Publish(ctx, fabric.EventSubject("s", "i", fabric.StatusClaim, "verdict"), []byte("not json"))
	require.NoError(t, err)
	require.NoError(t, mustPublishClaim(ctx, f, claimAt(4)))

	inflight, err := log.ClaimsInFlight()
	require.NoError(t, err, "a malformed verdict event must be skipped, not error the count")
	require.Equal(t, 1, inflight, "only the one valid claim is in flight; the garbage verdict is ignored")
}

// A PublishClaimVerdict payload round-trips through DecodeClaimVerdict — the
// verdict marker is a first-class, decodable record, not opaque bytes, so a
// reader (the in-flight projection, the SSE bridge) can recover the target it
// resolves.
func TestClaimVerdict_roundTripsThroughDecode(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := isolatedFab(t)

	want := ledger.ClaimVerdict{Target: claimAt(7).Target, Rejected: true}
	_, err := ledger.PublishClaimVerdict(ctx, f, "s", "i", want)
	require.NoError(t, err)
	// Pin the encode/decode identity of the verdict itself: the wire form a reader
	// recovers must be the same target+rejected flag that was published.
	data, err := json.Marshal(want)
	require.NoError(t, err)
	got, err := ledger.DecodeClaimVerdict(data)
	require.NoError(t, err)
	require.Equal(t, want, got, "a published verdict decodes back to the same target+rejected flag")
}

// THE NON-VACUOUS DISTINCTION. A clean no-catch verdict (the verifier ran and
// said "no catch") resolves the bet OUT of flight; a verifier ERROR (the cage
// blew up — transient) must NOT, because branding a valid claim "rejected" on a
// flake would silently discard recoverable work. The two nil-record paths
// (nil,nil vs nil,err) must diverge.
func TestConsumeClaims_aNoCatchRejectsTheTargetButAnErrorLeavesItInFlight(t *testing.T) {
	t.Parallel()

	t.Run("clean no-catch resolves the bet out of flight", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		f := isolatedFab(t)
		log := ledger.Bind(f, "s", "i")
		reject := func(ledger.ClaimRecord) (*ledger.CatchRecord, error) { return nil, nil }
		go func() { _ = log.ConsumeClaims(ctx, reject, 30*time.Second, nil) }()

		require.NoError(t, mustPublishClaim(ctx, f, claimAt(4)))
		require.Eventually(t, func() bool {
			n, err := log.ClaimsInFlight()
			return err == nil && n == 0
		}, 3*time.Second, 20*time.Millisecond, "a verified no-catch must write a rejection marker that clears the bet")
		require.True(t, balanceIs(log, 0)(), "a rejected bet mints nothing")
	})

	t.Run("a transient verifier error leaves the bet in flight, never branded rejected", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		f := isolatedFab(t)
		log := ledger.Bind(f, "s", "i")
		errVerify := func(ledger.ClaimRecord) (*ledger.CatchRecord, error) { return nil, errors.New("cage exploded") }
		go func() { _ = log.ConsumeClaims(ctx, errVerify, 30*time.Second, nil) }()

		require.NoError(t, mustPublishClaim(ctx, f, claimAt(4)))
		// The claim is consumed (acked) but neither minted nor rejected: it stays a
		// bet in flight, resubmittable — the error must never have written a marker.
		require.Never(t, func() bool {
			n, err := log.ClaimsInFlight()
			return err == nil && n == 0
		}, 1500*time.Millisecond, 50*time.Millisecond, "a transient error must not resolve the bet — it remains in flight")
		require.True(t, balanceIs(log, 0)(), "an errored verify mints nothing")
	})
}

// A confirmed claim leaves the in-flight set via its MINT, not via a rejection
// marker — the confirmed path is unchanged and never writes a (contradictory)
// rejection for a target it just minted.
func TestConsumeClaims_aConfirmedClaimLeavesFlightByMintingNotByRejection(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := isolatedFab(t)
	log := ledger.Bind(f, "s", "i")
	go func() { _ = log.ConsumeClaims(ctx, confirmFromClaim, 30*time.Second, nil) }()

	require.NoError(t, mustPublishClaim(ctx, f, claimAt(4)))
	require.Eventually(t, balanceIs(log, 1),
		3*time.Second, 20*time.Millisecond, "a confirmed claim mints a catch")
	inflight, err := log.ClaimsInFlight()
	require.NoError(t, err)
	require.Equal(t, 0, inflight, "the confirmed target left flight by minting — not in flight, not a phantom from a stray rejection")
}
