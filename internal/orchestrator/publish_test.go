package orchestrator_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/diff"
	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/orchestrator"
	"github.com/joaomdsg/packets/internal/settle"
)

// A minted revision must travel producer → taxonomy → bus → consumer intact:
// it lands on the canonical minted-revision subject and its payload decodes
// back to the same revision a consumer can rebuild state from.
func TestPublishRevision_roundTripsMintedRevisionThroughTheBus(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, f.Close()) })

	// Distinctive values + a nested Hunk: defeats a hardcoded payload and proves
	// the diff's nested structure survives the JSON round-trip.
	wantDiff := diff.Diff{Files: []diff.FileDiff{{
		Path: "main.go", Added: 7, Deleted: 3,
		Hunks: []diff.Hunk{{OldStart: 4, OldLines: 3, NewStart: 4, NewLines: 7}},
	}}}
	out := orchestrator.TurnOutcome{Minted: true, SHA: "deadbeef", Added: 7, Deleted: 3, Diff: wantDiff}

	seq, err := orchestrator.PublishRevision(ctx, f, "s1", "i1", out)
	require.NoError(t, err)

	events, err := f.ReplaySubject(ctx, "packets.session.s1.events.*.minted.>")
	require.NoError(t, err)
	require.Len(t, events, 1)

	assert.Equal(t, fabric.EventSubject("s1", "i1", fabric.StatusMinted, "revision"), events[0].Subject)
	assert.Equal(t, seq, events[0].Seq, "the returned sequence is the event's authoritative stream position")

	rev, err := orchestrator.DecodeRevision(events[0].Data)
	require.NoError(t, err)
	assert.Equal(t, orchestrator.RevisionEvent{
		SHA:     "deadbeef",
		Added:   7,
		Deleted: 3,
		Diff:    wantDiff,
	}, rev)
}

// Only a real mint may reach the source-of-truth subject. A blocked or no-edit
// turn must publish nothing — otherwise a consumer would rebuild state from a
// revision that never landed.
func TestPublishRevision_refusesToForgeARevisionFromAnUnmintedTurn(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		out  orchestrator.TurnOutcome
	}{
		{"secret-blocked", orchestrator.TurnOutcome{Secrets: []settle.SecretHit{{File: "x", Line: 1, Rule: "aws"}}}},
		{"no-edit", orchestrator.TurnOutcome{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			f, err := fabric.Start(ctx, t.TempDir())
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, f.Close()) })

			_, err = orchestrator.PublishRevision(ctx, f, "s1", "i1", tt.out)
			require.Error(t, err)

			// Replay the ENTIRE log, not just the minted filter: "publish
			// nothing" must hold for every subject, so a leak to scratch (or any
			// other subject) is caught too.
			events, err := f.Replay(ctx)
			require.NoError(t, err)
			assert.Empty(t, events, "an unminted turn must publish nothing to any subject")
		})
	}
}

func TestDecodeRevision_returnsErrorOnMalformedPayload(t *testing.T) {
	t.Parallel()
	_, err := orchestrator.DecodeRevision([]byte("not json"))
	assert.Error(t, err)
}
