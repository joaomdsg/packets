package fabric

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
)

// SubscribeNew delivers, in sequence order on the returned channel, ONLY events
// matching the filter that are published AFTER this call establishes the
// consumer — it never replays stored history. It is the tail-only sibling of
// Subscribe (which replays history then tails): the arrival-watcher primitive
// for a parked socket, which must wake on the next new claim without being
// falsely woken by the whole claim backlog.
//
// The DeliverNew start position is resolved to a concrete stream sequence when
// the ephemeral consumer is created here (synchronously, before returning), so
// an event stored before this call can never be delivered and an event
// published after this returns is always tailed.
//
// Teardown matches Subscribe: canceling ctx is the ONLY teardown — it unblocks
// the fetch, unsubscribes the ephemeral consumer, and closes the channel. A
// single goroutine owns the channel (sole sender and closer), so close can
// never race a send. The caller MUST cancel ctx when done.
func (f *Fabric) SubscribeNew(ctx context.Context, filter string) (<-chan Event, error) {
	sub, err := f.js.PullSubscribe(filter, "", nats.BindStream(streamName), nats.DeliverNew())
	if err != nil {
		return nil, fmt.Errorf("fabric: subscribe-new %s: %v", filter, err)
	}

	ch := make(chan Event, 64)
	go func() {
		defer close(ch)
		defer sub.Unsubscribe()
		for {
			msgs, err := sub.Fetch(1, nats.Context(ctx))
			if err != nil {
				return // ctx canceled or stream gone: stop tailing, close channel
			}
			for _, m := range msgs {
				meta, err := m.Metadata()
				if err != nil {
					return
				}
				m.Ack()
				select {
				case ch <- Event{Subject: m.Subject, Seq: meta.Sequence.Stream, Data: m.Data}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return ch, nil
}
