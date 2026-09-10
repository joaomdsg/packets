package fabric_test

import (
	"context"
	"testing"
	"time"

	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SubscribeNew is the arrival-watcher primitive for auto-parking: it must
// ignore the entire stored history — a parked session must NOT be woken by old
// claims — and deliver only events published after it subscribes, carrying
// their true stream sequence.
func TestSubscribeNew_ignoresHistoryAndDeliversOnlyLaterEvents(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, f.Close()) })

	subject := fabric.EventSubject("s1", "i1", fabric.StatusClaim, "claim")

	// Published BEFORE the subscription: this old claim must never be delivered,
	// or a parked session would wake the instant it started watching.
	_, err = f.Publish(ctx, subject, []byte(`{"old":true}`))
	require.NoError(t, err)

	ch, err := f.SubscribeNew(ctx, "packets.session.s1.events.*.claim.>")
	require.NoError(t, err)

	// Published AFTER: the only event the watcher should ever see.
	seq, err := f.Publish(ctx, subject, []byte(`{"new":true}`))
	require.NoError(t, err)

	assert.Equal(t, fabric.Event{Subject: subject, Seq: seq, Data: []byte(`{"new":true}`)}, recv(t, ch),
		"SubscribeNew must skip pre-subscription history and deliver the new event with its stream seq")
}

// The filter must demux on the live (DeliverNew) path too: a non-matching event
// published after subscribe is never delivered, only the matching one.
func TestSubscribeNew_demuxesNonMatchingLiveEvents(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, f.Close()) })

	claim := fabric.EventSubject("s1", "i1", fabric.StatusClaim, "claim")
	verdict := fabric.EventSubject("s1", "i1", fabric.StatusClaim, "verdict")

	ch, err := f.SubscribeNew(ctx, "packets.session.s1.events.*.claim.claim")
	require.NoError(t, err)

	// A non-matching verdict is published BEFORE the matching claim; if the demux
	// leaked it, it would surface as the first delivered event.
	_, err = f.Publish(ctx, verdict, []byte(`{"verdict":true}`))
	require.NoError(t, err)
	seq, err := f.Publish(ctx, claim, []byte(`{"claim":true}`))
	require.NoError(t, err)

	assert.Equal(t, fabric.Event{Subject: claim, Seq: seq, Data: []byte(`{"claim":true}`)}, recv(t, ch),
		"a non-matching live event must be filtered out, not merely deferred")
}

// Canceling the context tears the subscription down and closes the channel —
// the same teardown contract as Subscribe, proven after a real live delivery so
// an always-closed channel cannot pass.
func TestSubscribeNew_closesChannelOnContextCancel(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())

	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, f.Close()) })

	subject := fabric.EventSubject("s1", "i1", fabric.StatusClaim, "claim")
	ch, err := f.SubscribeNew(ctx, "packets.session.s1.events.*.claim.>")
	require.NoError(t, err)

	_, err = f.Publish(ctx, subject, []byte(`{"new":true}`))
	require.NoError(t, err)
	recv(t, ch)

	cancel()

	closed := make(chan struct{})
	go func() {
		for range ch {
		}
		close(closed)
	}()
	select {
	case <-closed:
	case <-time.After(3 * time.Second):
		t.Fatal("channel was not closed after context cancellation (leaked subscription)")
	}
}
