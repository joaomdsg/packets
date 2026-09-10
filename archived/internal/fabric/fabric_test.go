package fabric_test

import (
	"context"
	"testing"

	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFabric_replaysPublishedEventsInOrderWithMonotonicSeq(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, f.Close()) })

	const subject = "packets.session.s1.events.i1.minted.revision"

	seq1, err := f.Publish(ctx, subject, []byte(`{"rev":1}`))
	require.NoError(t, err)
	seq2, err := f.Publish(ctx, subject, []byte(`{"rev":2}`))
	require.NoError(t, err)

	assert.Equal(t, uint64(1), seq1, "first publish is seq 1")
	assert.Equal(t, uint64(2), seq2, "seq is monotonic")

	events, err := f.Replay(ctx)
	require.NoError(t, err)

	require.Len(t, events, 2)
	assert.Equal(t, fabric.Event{Subject: subject, Seq: 1, Data: []byte(`{"rev":1}`)}, events[0])
	assert.Equal(t, fabric.Event{Subject: subject, Seq: 2, Data: []byte(`{"rev":2}`)}, events[1])
}

func TestFabric_replaysEmptyWhenNoEventsPublished(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, f.Close()) })

	events, err := f.Replay(ctx)
	require.NoError(t, err)
	assert.Empty(t, events)
}

// The log is the authoritative source of truth (Phase-0 locked decision):
// events must survive a full process restart, replayed from the same storage
// dir with sequences intact. A non-durable in-memory implementation fails here.
func TestFabric_replaysEventsAfterRestartFromSameStorage(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := t.TempDir()

	const subject = "packets.session.s1.events.i1.minted.revision"

	f1, err := fabric.Start(ctx, dir)
	require.NoError(t, err)
	_, err = f1.Publish(ctx, subject, []byte(`{"rev":1}`))
	require.NoError(t, err)
	require.NoError(t, f1.Close())

	f2, err := fabric.Start(ctx, dir)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, f2.Close()) })

	events, err := f2.Replay(ctx)
	require.NoError(t, err)

	require.Len(t, events, 1)
	assert.Equal(t, fabric.Event{Subject: subject, Seq: 1, Data: []byte(`{"rev":1}`)}, events[0])
}
