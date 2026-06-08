package bridge

import (
	"context"

	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/ledger"
)

// WatchFleet subscribes to the whole fabric's minted economy and emits a fresh
// per-session projection map (ledger.FleetProjection) on every committed event,
// across all sessions — history first (a late subscriber sees current state),
// then live. It is the cross-session board's stream-driven feed: the board
// reflects every session off the one stream, regardless of which producer wrote
// it.
//
// Like Watch, re-folding the whole fleet per event reuses the canonical fold,
// and canceling ctx is the only teardown — it stops the subscription and closes
// the channel, with a send guard so an abandoned consumer cannot leak the feeder
// goroutine. The caller MUST cancel ctx when done.
func WatchFleet(ctx context.Context, f *fabric.Fabric) (<-chan map[string]ledger.Projection, error) {
	events, err := f.Subscribe(ctx, fabric.FleetMintedSubject())
	if err != nil {
		return nil, err
	}

	out := make(chan map[string]ledger.Projection, 64)
	go func() {
		defer close(out)
		for range events {
			fleet, err := ledger.FleetProjection(ctx, f)
			if err != nil {
				return
			}
			select {
			case out <- fleet:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, nil
}
