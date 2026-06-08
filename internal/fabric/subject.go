package fabric

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

// Status is the producer/commit-status token in the subject taxonomy. It
// demuxes authoritative source-of-truth events (minted) from discarded fan-out
// activity (scratch) so a projection rebuild can replay only the former.
type Status = string

const (
	StatusScratch Status = "scratch"
	StatusMinted  Status = "minted"
)

// EventSubject builds the canonical event subject
// packets.session.<session>.events.<instance>.<status>.<kind>. All event
// publishers must construct subjects through this function so the taxonomy
// stays the single source of demux truth.
func EventSubject(session, instance, status, kind string) string {
	return fmt.Sprintf("packets.session.%s.events.%s.%s.%s", session, instance, status, kind)
}

// ReplaySubject replays, in global sequence order, only the stored events whose
// subject matches the NATS filter (JetStream-native FilterSubject — the broker
// does the demux, not client-side string matching). Surviving events keep their
// original stream sequence, since seq is the authoritative cross-producer order.
func (f *Fabric) ReplaySubject(ctx context.Context, filter string) ([]Event, error) {
	sub, err := f.js.PullSubscribe(filter, "", nats.BindStream(streamName), nats.DeliverAll())
	if err != nil {
		return nil, fmt.Errorf("fabric: subscribe %s: %v", filter, err)
	}
	defer sub.Unsubscribe()

	// The consumer is created with nothing delivered yet, so NumPending is the
	// exact count of matching messages — fetch precisely that many rather than
	// guessing a drain timeout.
	ci, err := sub.ConsumerInfo()
	if err != nil {
		return nil, fmt.Errorf("fabric: consumer info %s: %v", filter, err)
	}
	remaining := int(ci.NumPending)

	var events []Event
	for remaining > 0 {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("fabric: fetch %s: %v", filter, err)
		}
		n := min(remaining, 256)
		// Per-batch deadline bounds a stalled fetch while still propagating
		// caller cancellation; nats rejects Context+MaxWait together.
		batchCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		msgs, err := sub.Fetch(n, nats.Context(batchCtx))
		cancel()
		if err != nil {
			return nil, fmt.Errorf("fabric: fetch %s: %v", filter, err)
		}
		for _, m := range msgs {
			meta, err := m.Metadata()
			if err != nil {
				return nil, fmt.Errorf("fabric: metadata %s: %v", filter, err)
			}
			events = append(events, Event{Subject: m.Subject, Seq: meta.Sequence.Stream, Data: m.Data})
			m.Ack()
		}
		remaining -= len(msgs)
	}
	return events, nil
}
