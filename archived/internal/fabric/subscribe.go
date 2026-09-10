package fabric

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
)

// Subscribe delivers, in global sequence order on the returned channel, every
// stored event matching the NATS filter (catch-up from the start of the log)
// and then every later-published matching event live — the SSE-reconnect
// contract: history first, then tail.
//
// A single goroutine owns the channel — it is the sole sender and the sole
// closer — so close can never race a send.
//
// Canceling ctx is the ONLY teardown: it unblocks the fetch (and any pending
// send), unsubscribes the server-side ephemeral consumer, and closes the
// channel. The caller MUST cancel ctx when done. There is no Close method;
// abandoning the channel without canceling leaks the goroutine (and the
// consumer) forever once the buffer fills — the goroutine then blocks on the
// send. Always pair Subscribe with a deferred cancel and either drain the
// channel to close or cancel to stop early.
func (f *Fabric) Subscribe(ctx context.Context, filter string) (<-chan Event, error) {
	sub, err := f.js.PullSubscribe(filter, "", nats.BindStream(streamName), nats.DeliverAll())
	if err != nil {
		return nil, fmt.Errorf("fabric: subscribe %s: %v", filter, err)
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
