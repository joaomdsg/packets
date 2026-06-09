package ledger_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/ledger"
)

func claimAt(line int) ledger.ClaimRecord {
	return ledger.ClaimRecord{Target: ledger.Target{
		BaseRev: "base", FixRev: "fix", TipRev: "fix", Path: "a.go", Line: line,
	}}
}

// confirmFromClaim is a stub verifier standing in for the real (sandboxed, #6c)
// oracle: it confirms a claim into a catch keyed to that claim's target, so two
// DISTINCT claims confirm into two distinct catch identities and a replayed claim
// confirms into the same one. It executes no code.
func confirmFromClaim(c ledger.ClaimRecord) (*ledger.CatchRecord, error) {
	r := sampleRecord()
	r.Path, r.Line = c.Target.Path, c.Target.Line
	r.BeforeRev, r.AfterRev = c.Target.BaseRev, c.Target.FixRev
	return &r, nil
}

// balanceIs is a goroutine-safe poll condition for Eventually/Never: it never
// uses require (testify runs the condition in its own goroutine that can outlive
// the assertion and race teardown), and tolerates a closed fabric by reporting
// false rather than erroring.
func balanceIs(l *ledger.Log, want int) func() bool {
	return func() bool {
		b, err := l.Balance()
		return err == nil && b == want
	}
}

func TestConsumeClaims_mintsDistinctClaimsButDedupesAReplay(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := isolatedFab(t)
	log := ledger.Bind(f, "s", "i")
	go func() { _ = log.ConsumeClaims(ctx, confirmFromClaim, 30*time.Second) }()

	require.NoError(t, mustPublishClaim(ctx, f, claimAt(4)))
	require.Eventually(t, balanceIs(log, 1),
		3*time.Second, 20*time.Millisecond, "a confirmed claim must mint a catch")

	// A DISTINCT claim mints a second, distinct catch — the consumer keeps working,
	// it didn't merely mint once and stop.
	require.NoError(t, mustPublishClaim(ctx, f, claimAt(5)))
	require.Eventually(t, balanceIs(log, 2),
		3*time.Second, 20*time.Millisecond, "a distinct claim must mint a distinct catch")

	// Re-submitting an earlier claim reproduces its catch identity — Append's
	// farm-denial gate refuses it, so the balance never climbs past 2.
	require.NoError(t, mustPublishClaim(ctx, f, claimAt(4)))
	require.Never(t, func() bool { return !balanceIs(log, 2)() },
		500*time.Millisecond, 50*time.Millisecond, "a replayed claim must mint nothing more")
}

func TestConsumeClaims_mintsNothingWhenTheVerifierRejects(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := isolatedFab(t)
	log := ledger.Bind(f, "s", "i")
	reject := func(ledger.ClaimRecord) (*ledger.CatchRecord, error) { return nil, nil }
	go func() { _ = log.ConsumeClaims(ctx, reject, 30*time.Second) }()

	require.NoError(t, mustPublishClaim(ctx, f, claimAt(4)))
	require.Never(t, func() bool { return !balanceIs(log, 0)() },
		500*time.Millisecond, 50*time.Millisecond, "a rejected claim must mint nothing")
}

func TestConsumeClaims_mintsNothingWhenTheVerifierErrsAndKeepsConsuming(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := isolatedFab(t)
	log := ledger.Bind(f, "s", "i")
	// A verifier ERROR is a distinct path from a nil verdict (the oracle blew up vs.
	// it ran and said no): it must mint nothing AND not tear the consumer down — a
	// later valid claim, run through a confirming verifier, still mints.
	calls := 0
	verify := func(c ledger.ClaimRecord) (*ledger.CatchRecord, error) {
		calls++
		if calls == 1 {
			return nil, errors.New("oracle exploded")
		}
		return confirmFromClaim(c)
	}
	go func() { _ = log.ConsumeClaims(ctx, verify, 30*time.Second) }()

	require.NoError(t, mustPublishClaim(ctx, f, claimAt(4)))
	require.Never(t, func() bool { return !balanceIs(log, 0)() },
		500*time.Millisecond, 50*time.Millisecond, "a verifier error must mint nothing")
	require.NoError(t, mustPublishClaim(ctx, f, claimAt(5)))
	require.Eventually(t, balanceIs(log, 1),
		3*time.Second, 20*time.Millisecond, "a valid claim after a verifier error must still mint")
}

func TestConsumeClaims_survivesAMalformedClaimAndKeepsConsuming(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := isolatedFab(t)
	log := ledger.Bind(f, "s", "i")
	go func() { _ = log.ConsumeClaims(ctx, confirmFromClaim, 30*time.Second) }()

	// Garbage on the claim subtree must not tear the consumer down — a later valid
	// claim still mints.
	_, err := f.Publish(ctx, fabric.EventSubject("s", "i", fabric.StatusClaim, "work"), []byte("not json"))
	require.NoError(t, err)
	require.NoError(t, mustPublishClaim(ctx, f, claimAt(4)))
	require.Eventually(t, balanceIs(log, 1),
		3*time.Second, 20*time.Millisecond, "a valid claim after a malformed one must still mint")
}

// The equivalence lock: a verdict minted through the claim consumer yields the
// SAME economy as the same record appended directly in-process — the consumer is
// a new INPUT to the one mint path, not a second economy.
func TestConsumeClaims_mintsTheSameEconomyAsADirectAppend(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	direct := ledger.Bind(isolatedFab(t), "s", "i")
	directRec, err := confirmFromClaim(claimAt(4))
	require.NoError(t, err)
	require.NoError(t, direct.Append(*directRec))

	fb := isolatedFab(t)
	viaClaim := ledger.Bind(fb, "s", "i")
	go func() { _ = viaClaim.ConsumeClaims(ctx, confirmFromClaim, 30*time.Second) }()
	require.NoError(t, mustPublishClaim(ctx, fb, claimAt(4)))
	require.Eventually(t, balanceIs(viaClaim, 1),
		3*time.Second, 20*time.Millisecond, "the claim must mint")

	pd, err := direct.Records()
	require.NoError(t, err)
	pv, err := viaClaim.Records()
	require.NoError(t, err)
	require.Equal(t, pd, pv, "claim-minted economy must match the direct-append economy")
}

// Restarting the claim consumer must NOT re-verify claims it already processed.
// Verification is the scarce resource (each is a full sandboxed oracle run), so a
// consumer that replayed the whole claim backlog from seq 0 on every restart
// would let the verifier be starved by its own history. The durable consumer
// resumes past what it already acked, so a restart re-verifies nothing already done.
func TestConsumeClaims_restartDoesNotReverifyAlreadyProcessedClaims(t *testing.T) {
	t.Parallel()
	f := isolatedFab(t)
	log := ledger.Bind(f, "s", "i")

	counting := func(n *atomic.Int32) ledger.Verifier {
		return func(c ledger.ClaimRecord) (*ledger.CatchRecord, error) {
			n.Add(1)
			return confirmFromClaim(c)
		}
	}

	// Run 1: consume, verify (counting), mint, then stop.
	ctx1, cancel1 := context.WithCancel(context.Background())
	var count1 atomic.Int32
	go func() { _ = log.ConsumeClaims(ctx1, counting(&count1), 30*time.Second) }()
	require.NoError(t, mustPublishClaim(ctx1, f, claimAt(4)))
	require.Eventually(t, balanceIs(log, 1),
		3*time.Second, 20*time.Millisecond, "the claim must be verified and minted on the first run")
	time.Sleep(250 * time.Millisecond) // let the post-mint ack land before tearing the consumer down
	cancel1()

	// Run 2: same Log → same durable. The already-acked claim must NOT be
	// redelivered, so the (expensive) verifier is never invoked again for it.
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	var count2 atomic.Int32
	go func() { _ = log.ConsumeClaims(ctx2, counting(&count2), 30*time.Second) }()

	require.Never(t, func() bool { return count2.Load() > 0 },
		1500*time.Millisecond, 50*time.Millisecond,
		"a restart must resume past the already-processed claim, never re-running the verifier over it")
	require.True(t, balanceIs(log, 1)(), "the economy is unchanged by the restart")
}

// A session token may legally contain characters that are valid in a NATS
// SUBJECT token (e.g. '/') but INVALID in a durable-consumer NAME. The claim
// consumer must sanitize the durable name derived from session+instance, or it
// can't bind its consumer at all and verifies nothing. '/' passes ValidToken
// (which only forbids ". \t*>"), so it's a real input, not a hypothetical.
func TestConsumeClaims_bindsDespiteDurableUnsafeSessionTokens(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := isolatedFab(t)
	log := ledger.Bind(f, "work/123", "inst/9")
	go func() { _ = log.ConsumeClaims(ctx, confirmFromClaim, 30*time.Second) }()

	_, err := ledger.PublishClaim(ctx, f, "work/123", "inst/9", claimAt(4))
	require.NoError(t, err)
	require.Eventually(t, balanceIs(log, 1),
		3*time.Second, 20*time.Millisecond,
		"a claim on a session whose token has durable-unsafe chars must still verify+mint — the durable name was sanitized")
}

func mustPublishClaim(ctx context.Context, f *fabric.Fabric, c ledger.ClaimRecord) error {
	_, err := ledger.PublishClaim(ctx, f, "s", "i", c)
	return err
}
