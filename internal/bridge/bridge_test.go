package bridge_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/bridge"
	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/ledger"
)

func startFabric(t *testing.T) *fabric.Fabric {
	t.Helper()
	f, err := fabric.Start(context.Background(), t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	return f
}

func sampleCatch() ledger.CatchRecord {
	return ledger.CatchRecord{
		Outcome:           catch.Catch,
		Path:              "adult.go",
		Line:              4,
		BeforeRev:         "aaaa",
		AfterRev:          "bbbb",
		BeforeInventory:   []string{">="},
		AfterInventory:    []string{">="},
		MutantsConsidered: 1,
		ReasonTag:         "catch",
	}
}

// awaitBalance drains snapshots until one reports the wanted balance, proving
// the stream-driven feed converged on it. Intermediate re-renders are allowed.
func awaitBalance(t *testing.T, ch <-chan ledger.Projection, want int) {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		select {
		case p, ok := <-ch:
			require.True(t, ok, "stream closed before reaching balance %d", want)
			if p.Balance() == want {
				return
			}
		case <-deadline:
			t.Fatalf("timed out waiting for balance %d", want)
		}
	}
}

func TestWatch_reflectsEachCommittedEventInAFreshSnapshot(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := startFabric(t)
	log := ledger.Bind(f, "session", "i")

	ch, err := bridge.Watch(ctx, f, "session", "i")
	require.NoError(t, err)

	require.NoError(t, log.Append(sampleCatch()))
	awaitBalance(t, ch, 1)

	require.NoError(t, log.AppendSpend(1, "fund"))
	awaitBalance(t, ch, 0)
}

func TestWatch_replaysHistoryToALateSubscriber(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := startFabric(t)
	log := ledger.Bind(f, "session", "i")
	require.NoError(t, log.Append(sampleCatch())) // committed BEFORE Watch

	ch, err := bridge.Watch(ctx, f, "session", "i")
	require.NoError(t, err)

	awaitBalance(t, ch, 1) // a freshly-connected browser sees current state
}

func TestWatch_isolatesOneSessionFromAnother(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := startFabric(t)
	other := ledger.Bind(f, "other", "i")
	require.NoError(t, other.Append(sampleCatch())) // a peer session's mint

	watched := ledger.Bind(f, "watched", "i")
	ch, err := bridge.Watch(ctx, f, "watched", "i")
	require.NoError(t, err)

	require.NoError(t, watched.Append(sampleCatch()))
	awaitBalance(t, ch, 1) // sees only its own mint, never the peer's
}

func TestWatch_doesNotLeakWhenConsumerAbandonsTheStream(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	f := startFabric(t)
	log := ledger.Bind(f, "session", "i")

	ch, err := bridge.Watch(ctx, f, "session", "i")
	require.NoError(t, err)

	// Flood far past the channel buffer WITHOUT ever reading, so the feeder
	// goroutine blocks on its send. Cancel must still tear it down (the
	// send-guard), not leave it blocked forever.
	for i := 0; i < 100; i++ {
		r := sampleCatch()
		r.Line = 4 + i // distinct identities so each is a real, distinct event
		require.NoError(t, log.Append(r))
	}
	cancel()

	require.Eventually(t, func() bool {
		for {
			select {
			case _, ok := <-ch:
				if !ok {
					return true
				}
			default:
				return false
			}
		}
	}, 3*time.Second, 10*time.Millisecond, "feeder did not close after ctx cancel")
}

func TestWatch_closesTheStreamWhenContextCanceled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	f := startFabric(t)
	log := ledger.Bind(f, "session", "i")

	ch, err := bridge.Watch(ctx, f, "session", "i")
	require.NoError(t, err)

	// Prove the stream is live and emitting BEFORE cancel, so a stub that just
	// closes immediately cannot pass: it would never reach balance 1.
	require.NoError(t, log.Append(sampleCatch()))
	awaitBalance(t, ch, 1)

	cancel()
	require.Eventually(t, func() bool {
		select {
		case _, ok := <-ch:
			return !ok
		default:
			return false
		}
	}, 3*time.Second, 10*time.Millisecond, "stream did not close after ctx cancel")
}
