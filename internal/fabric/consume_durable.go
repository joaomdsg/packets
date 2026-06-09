package fabric

import (
	"context"
	"errors"
	"fmt"

	"github.com/nats-io/nats.go"
)

// ConsumeDurable processes the stream's matching events through a DURABLE work
// consumer: unlike Subscribe (an ephemeral DeliverAll replay, right for an
// SSE-reconnecting browser), a durable consumer RESUMES past what it has already
// acked when it is rebound. So a restart does NOT replay — and re-process — every
// matching event from the start of the log.
//
// Each event is handed to handle; it is acked ONLY after handle returns nil. A
// handle that returns an error leaves the message UN-acked and Naks it for
// redelivery (the work is not silently lost), and the loop keeps going so one bad
// message can't wedge the stream. The durable is created once (idempotent across
// rebinds) and is NOT deleted on teardown, so the next bind resumes. Canceling
// ctx is the only teardown; the caller MUST cancel ctx when done.
func (f *Fabric) ConsumeDurable(ctx context.Context, durable, filter string, handle func(Event) error) error {
	if _, err := f.js.AddConsumer(streamName, &nats.ConsumerConfig{
		Durable:       durable,
		AckPolicy:     nats.AckExplicitPolicy,
		DeliverPolicy: nats.DeliverAllPolicy,
		FilterSubject: filter,
	}); err != nil && !errors.Is(err, nats.ErrConsumerNameAlreadyInUse) {
		return fmt.Errorf("fabric: durable consumer %s: %v", durable, err)
	}

	sub, err := f.js.PullSubscribe(filter, durable, nats.BindStream(streamName))
	if err != nil {
		return fmt.Errorf("fabric: bind durable %s: %v", durable, err)
	}
	defer sub.Unsubscribe() //nolint:errcheck // a durable consumer survives unsubscribe; this drops only the local binding

	for {
		msgs, err := sub.Fetch(1, nats.Context(ctx))
		if err != nil {
			// A ctx cancellation is the intended teardown: report it as such (the
			// caller treats ctx.Err() as a clean stop). But a Fetch error while ctx
			// is STILL live means the stream/connection died under us — surface that
			// real error instead of masking it as a nil/clean shutdown, so a
			// supervisor can tell "asked to stop" from "the log fell out from under me".
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr // ctx canceled: stop, leaving the durable in place to resume
			}
			return fmt.Errorf("fabric: fetch durable %s: %v", durable, err)
		}
		for _, m := range msgs {
			meta, err := m.Metadata()
			if err != nil {
				_ = m.Nak()
				continue
			}
			if err := handle(Event{Subject: m.Subject, Seq: meta.Sequence.Stream, Data: m.Data}); err != nil {
				_ = m.Nak() // not finished with: redeliver rather than drop
				continue
			}
			_ = m.Ack()
		}
	}
}
