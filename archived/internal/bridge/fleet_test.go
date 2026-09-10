package bridge_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/bridge"
	"github.com/joaomdsg/packets/internal/ledger"
)

// awaitFleet drains fleet snapshots until one satisfies pred, proving the
// cross-session feed converged on it. Intermediate snapshots are allowed.
func awaitFleet(t *testing.T, ch <-chan map[string]ledger.FleetView, pred func(map[string]ledger.FleetView) bool) {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		select {
		case m, ok := <-ch:
			require.True(t, ok, "fleet stream closed before the predicate held")
			if pred(m) {
				return
			}
		case <-deadline:
			t.Fatal("timed out waiting for the fleet predicate to hold")
		}
	}
}

func TestWatchFleet_reflectsEverySessionsEconomyInOneSnapshot(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := startFabric(t)
	alpha := ledger.Bind(f, "alpha", "i")
	beta := ledger.Bind(f, "beta", "i")

	ch, err := bridge.WatchFleet(ctx, f)
	require.NoError(t, err)

	require.NoError(t, alpha.Append(sampleCatch()))
	awaitFleet(t, ch, func(m map[string]ledger.FleetView) bool {
		return m["alpha"].Balance() == 1
	})

	require.NoError(t, beta.Append(sampleCatch()))
	// One snapshot carries BOTH sessions — the cross-session aggregate, not a
	// single session's feed.
	awaitFleet(t, ch, func(m map[string]ledger.FleetView) bool {
		return m["alpha"].Balance() == 1 && m["beta"].Balance() == 1
	})
}

func TestWatchFleet_doesNotLeakWhenConsumerAbandonsTheStream(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	f := startFabric(t)
	log := ledger.Bind(f, "alpha", "i")

	ch, err := bridge.WatchFleet(ctx, f)
	require.NoError(t, err)

	// Flood past the channel buffer WITHOUT reading, so the feeder blocks on its
	// send; cancel must still tear it down via the send-guard.
	for i := 0; i < 100; i++ {
		r := sampleCatch()
		r.Line = 4 + i
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
	}, 3*time.Second, 10*time.Millisecond, "fleet feeder did not close after ctx cancel")
}

func TestWatchFleet_closesTheStreamWhenContextCanceled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	f := startFabric(t)
	alpha := ledger.Bind(f, "alpha", "i")

	ch, err := bridge.WatchFleet(ctx, f)
	require.NoError(t, err)

	require.NoError(t, alpha.Append(sampleCatch()))
	awaitFleet(t, ch, func(m map[string]ledger.FleetView) bool {
		return m["alpha"].Balance() == 1
	}) // alive before cancel

	cancel()
	require.Eventually(t, func() bool {
		select {
		case _, ok := <-ch:
			return !ok
		default:
			return false
		}
	}, 3*time.Second, 10*time.Millisecond, "fleet stream did not close after ctx cancel")
}
