// Package bridge feeds one session's economy from the authoritative event
// stream to a renderer: it subscribes to the session's minted subtree and emits
// a fresh projection snapshot on every committed event. This is the read side of
// the NATS→SSE browser bridge — the rendered board reflects the stream, so it
// can never disagree with the real ledger (and a future cross-process producer's
// events drive the same render).
package bridge

import (
	"context"

	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/ledger"
)

// Watch subscribes to session+instance's minted economy subtree and returns a
// channel that yields a freshly-folded ledger.Projection for every committed
// event — history first (so a late subscriber sees current state), then live.
// Re-folding the whole projection per event reuses the canonical fold rather
// than duplicating it; the stream is small and the read is cheap at prototype
// scale.
//
// Canceling ctx is the only teardown: it stops the underlying subscription and
// closes the returned channel. The caller MUST cancel ctx when done.
func Watch(ctx context.Context, f *fabric.Fabric, session, instance string) (<-chan ledger.Projection, error) {
	filter := fabric.EventSubject(session, instance, fabric.StatusMinted, "*")
	events, err := f.Subscribe(ctx, filter)
	if err != nil {
		return nil, err
	}

	out := make(chan ledger.Projection, 64)
	go func() {
		defer close(out)
		for range events {
			p, err := ledger.ReplayProjection(ctx, f, session, instance)
			if err != nil {
				return
			}
			select {
			case out <- p:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, nil
}
