package fabric_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventSubject_buildsCanonicalTaxonomyPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                            string
		session, instance, status, kind string
		want                            string
	}{
		{
			name:    "minted revision",
			session: "s1", instance: "i1", status: fabric.StatusMinted, kind: "revision",
			want: "packets.session.s1.events.i1.minted.revision",
		},
		{
			name:    "scratch diff",
			session: "s1", instance: "i7", status: fabric.StatusScratch, kind: "diff",
			want: "packets.session.s1.events.i7.scratch.diff",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, fabric.EventSubject(tt.session, tt.instance, tt.status, tt.kind))
		})
	}
}

func TestSessionOf_extractsTheSessionFromACanonicalSubject(t *testing.T) {
	t.Parallel()
	subj := fabric.EventSubject("alpha", "i1", fabric.StatusMinted, "catch")
	assert.Equal(t, "alpha", fabric.SessionOf(subj))
}

func TestSessionOf_returnsEmptyForAMalformedSubject(t *testing.T) {
	t.Parallel()
	for _, s := range []string{
		"",
		"packets.session",                            // too few tokens
		"not.a.subject",                              // wrong arity + literals
		"x.session.a.events.i.minted.catch",          // wrong root literal
		"packets.notsession.a.events.i.minted.catch", // wrong "session" literal
		"packets.session.a.NOTevents.i.minted.catch", // wrong "events" literal
	} {
		assert.Equal(t, "", fabric.SessionOf(s), "subject %q must not yield a session", s)
	}
}

// The fleet filter is exactly the minted path with both the session and instance
// tokens wildcarded — so it matches every session's source-of-truth events and
// nothing scratch.
func TestFleetMintedSubject_isTheWildcardedMintedPath(t *testing.T) {
	t.Parallel()
	assert.Equal(t, fabric.EventSubject("*", "*", fabric.StatusMinted, ">"), fabric.FleetMintedSubject())
}

// The whole point of the scratch/minted split: a consumer rebuilding the
// source-of-truth projection must replay ONLY minted events and never see
// discarded fan-out (scratch) activity — and the surviving events must keep
// their original global sequences, since seq is the authoritative cross-producer
// order, not a per-filter renumbering.
func TestReplaySubject_demuxesScratchFanoutFromMintedSourceOfTruth(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, f.Close()) })

	// Same instance for all three: the filter must demux purely on the
	// scratch/minted status token, not on the instance token.
	minted1 := fabric.EventSubject("s1", "i1", fabric.StatusMinted, "revision")
	scratch := fabric.EventSubject("s1", "i1", fabric.StatusScratch, "revision")
	minted2 := fabric.EventSubject("s1", "i1", fabric.StatusMinted, "revision")

	seq1, err := f.Publish(ctx, minted1, []byte(`{"rev":1}`))
	require.NoError(t, err)
	_, err = f.Publish(ctx, scratch, []byte(`{"discarded":true}`))
	require.NoError(t, err)
	seq3, err := f.Publish(ctx, minted2, []byte(`{"rev":2}`))
	require.NoError(t, err)

	events, err := f.ReplaySubject(ctx, "packets.session.s1.events.*.minted.>")
	require.NoError(t, err)

	require.Len(t, events, 2)
	assert.Equal(t, fabric.Event{Subject: minted1, Seq: seq1, Data: []byte(`{"rev":1}`)}, events[0])
	assert.Equal(t, fabric.Event{Subject: minted2, Seq: seq3, Data: []byte(`{"rev":2}`)}, events[1],
		"surviving events keep their original global sequence, not a per-filter renumbering")
}

// An event log routinely exceeds one fetch batch; a filtered replay must page
// through the whole match set and preserve global order across batch
// boundaries, never truncating at the first page.
func TestReplaySubject_pagesAllMatchesInOrderAcrossBatchBoundary(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, f.Close()) })

	subject := fabric.EventSubject("s1", "i1", fabric.StatusMinted, "revision")
	const n = 300 // > the 256 fetch batch size
	for i := 0; i < n; i++ {
		_, err := f.Publish(ctx, subject, []byte(fmt.Sprintf(`{"rev":%d}`, i)))
		require.NoError(t, err)
	}

	events, err := f.ReplaySubject(ctx, "packets.session.s1.events.*.minted.>")
	require.NoError(t, err)

	require.Len(t, events, n)
	for i, e := range events {
		assert.Equal(t, uint64(i+1), e.Seq, "global sequence preserved and ordered across batches")
		assert.Equal(t, fmt.Sprintf(`{"rev":%d}`, i), string(e.Data))
	}
}

func TestReplaySubject_returnsEmptyWhenFilterMatchesNothing(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, f.Close()) })

	_, err = f.Publish(ctx, fabric.EventSubject("s1", "i1", fabric.StatusMinted, "revision"), []byte(`{}`))
	require.NoError(t, err)

	events, err := f.ReplaySubject(ctx, "packets.session.s1.events.*.scratch.>")
	require.NoError(t, err)
	assert.Empty(t, events)
}

// A replay can race a shutdown or deadline; if the caller cancels its context
// the replay must abort with an error rather than ignore cancellation and run
// the whole fetch loop to completion.
func TestReplaySubject_abortsWhenContextCanceled(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, f.Close()) })

	subject := fabric.EventSubject("s1", "i1", fabric.StatusMinted, "revision")
	// Enough events that the loop must fetch at least once; a canceled context
	// must short-circuit it.
	for i := 0; i < 300; i++ {
		_, err := f.Publish(ctx, subject, []byte(`{}`))
		require.NoError(t, err)
	}

	canceled, cancel := context.WithCancel(ctx)
	cancel()

	_, err = f.ReplaySubject(canceled, "packets.session.s1.events.*.minted.>")
	require.Error(t, err)
}
