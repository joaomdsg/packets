package fabric_test

import (
	"context"
	"testing"
	"time"

	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recv reads one event from ch, failing the test if none arrives in time — a
// blocked subscription must surface as a failure, never a hung test.
func recv(t *testing.T, ch <-chan fabric.Event) fabric.Event {
	t.Helper()
	select {
	case e, ok := <-ch:
		require.True(t, ok, "channel closed before an expected event arrived")
		return e
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for an event")
		return fabric.Event{}
	}
}

// The SSE-reconnect contract: a subscriber must first receive the full matching
// history (catch-up from the start of the log) and THEN every later event live,
// all in global sequence order with original sequences preserved.
func TestSubscribe_replaysHistoryThenStreamsLiveInOrder(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, f.Close()) })

	subject := fabric.EventSubject("s1", "i1", fabric.StatusMinted, "revision")

	seq1, err := f.Publish(ctx, subject, []byte(`{"rev":1}`))
	require.NoError(t, err)
	seq2, err := f.Publish(ctx, subject, []byte(`{"rev":2}`))
	require.NoError(t, err)

	ch, err := f.Subscribe(ctx, "packets.session.s1.events.*.minted.>")
	require.NoError(t, err)

	assert.Equal(t, fabric.Event{Subject: subject, Seq: seq1, Data: []byte(`{"rev":1}`)}, recv(t, ch))
	assert.Equal(t, fabric.Event{Subject: subject, Seq: seq2, Data: []byte(`{"rev":2}`)}, recv(t, ch))

	seq3, err := f.Publish(ctx, subject, []byte(`{"rev":3}`))
	require.NoError(t, err)
	assert.Equal(t, fabric.Event{Subject: subject, Seq: seq3, Data: []byte(`{"rev":3}`)}, recv(t, ch),
		"a live event published after subscribe is tailed, in order after the history")
}

// A minted-only subscriber must never observe discarded scratch fan-out, even
// on the live path — the demux is not just a replay-time concern.
func TestSubscribe_demuxesScratchFromLiveMintedStream(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, f.Close()) })

	minted := fabric.EventSubject("s1", "i1", fabric.StatusMinted, "revision")
	scratch := fabric.EventSubject("s1", "i1", fabric.StatusScratch, "revision")

	ch, err := f.Subscribe(ctx, "packets.session.s1.events.*.minted.>")
	require.NoError(t, err)

	// Scratch is interleaved BETWEEN the two minted events: if the demux leaked
	// it, it would surface as the second delivered event instead of minted2.
	seq1, err := f.Publish(ctx, minted, []byte(`{"rev":1}`))
	require.NoError(t, err)
	_, err = f.Publish(ctx, scratch, []byte(`{"discarded":true}`))
	require.NoError(t, err)
	seq2, err := f.Publish(ctx, minted, []byte(`{"rev":2}`))
	require.NoError(t, err)

	assert.Equal(t, fabric.Event{Subject: minted, Seq: seq1, Data: []byte(`{"rev":1}`)}, recv(t, ch))
	assert.Equal(t, fabric.Event{Subject: minted, Seq: seq2, Data: []byte(`{"rev":2}`)}, recv(t, ch),
		"the interleaved scratch event must be absent from the stream, not merely deferred")
}

// Canceling the context must tear the subscription down and close the channel
// so a range loop terminates — and (under -race) prove no send-on-closed-channel
// panic and no goroutine left blocked on delivery.
func TestSubscribe_closesChannelOnContextCancelWithoutLeak(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())

	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, f.Close()) })

	subject := fabric.EventSubject("s1", "i1", fabric.StatusMinted, "revision")
	ch, err := f.Subscribe(ctx, "packets.session.s1.events.*.minted.>")
	require.NoError(t, err)

	// Force real live delivery first, so an always-closed channel can't pass:
	// the subscription must be genuinely tailing before we tear it down.
	_, err = f.Publish(ctx, subject, []byte(`{"rev":1}`))
	require.NoError(t, err)
	recv(t, ch)

	cancel()

	// Drain to completion: the channel must close, so this range returns.
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
