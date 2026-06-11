package orchestrator_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/orchestrator"
	"github.com/joaomdsg/packets/internal/translate"
)

// The live agent's activity must travel producer → taxonomy → bus → consumer
// intact so the surface can show a real run as it happens: it lands on the
// scratch/activity subject (non-authoritative — never the minted economy) and
// its payload decodes back to the same events the agent emitted.
func TestPublishActivity_roundTripsAgentActivityThroughTheBus(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, f.Close()) })

	want := []translate.UIEvent{
		{Type: "activity.agent", Kind: "thinking", Detail: "considering the error path"},
		{Type: "activity.agent", Kind: "editing", Detail: "internal/auth/token.go"},
	}

	seq, err := orchestrator.PublishActivity(ctx, f, "s1", "i1", want)
	require.NoError(t, err)

	events, err := f.ReplaySubject(ctx, "packets.session.s1.events.*.scratch.>")
	require.NoError(t, err)
	require.Len(t, events, 1)

	assert.Equal(t, fabric.EventSubject("s1", "i1", fabric.StatusScratch, "activity"), events[0].Subject)
	assert.Equal(t, seq, events[0].Seq, "the returned sequence is the event's authoritative stream position")

	got, err := orchestrator.DecodeActivity(events[0].Data)
	require.NoError(t, err)
	assert.Equal(t, want, got, "the activity batch must survive the bus round-trip intact")
}

// An empty activity batch carries no information; publishing it would be bus
// noise (and a needless scratch refold for every live viewer), so it must
// publish nothing and error — never leak an empty event to any subject.
func TestPublishActivity_refusesToPublishAnEmptyBatch(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, f.Close()) })

	_, err = orchestrator.PublishActivity(ctx, f, "s1", "i1", nil)
	require.Error(t, err)

	events, err := f.Replay(ctx)
	require.NoError(t, err)
	assert.Empty(t, events, "an empty activity batch must publish nothing to any subject")
}

func TestDecodeActivity_returnsErrorOnMalformedPayload(t *testing.T) {
	t.Parallel()
	_, err := orchestrator.DecodeActivity([]byte("not json"))
	assert.Error(t, err)
}
