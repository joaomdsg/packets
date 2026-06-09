package ledger_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/ledger"
)

func TestPublishClaim_roundTripsFromTheClaimSubtreeWithoutMinting(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	f := isolatedFab(t)

	claim := ledger.ClaimRecord{Target: ledger.Target{
		BaseRev: "base", FixRev: "fix", TipRev: "fix", Path: "a.go", Line: 4,
	}}
	_, err := ledger.PublishClaim(ctx, f, "s", "i", claim)
	require.NoError(t, err)

	// It lands on the claim subtree and decodes back to the same submission.
	events, err := f.ReplaySubject(ctx, fabric.EventSubject("s", "i", fabric.StatusClaim, ">"))
	require.NoError(t, err)
	require.Len(t, events, 1)
	got, err := ledger.DecodeClaim(events[0].Data)
	require.NoError(t, err)
	assert.Equal(t, claim, got)

	// The core safety invariant: a producer's claim mints NOTHING on its own —
	// the minted projection (folded from minted events) is untouched.
	p, err := ledger.ReplayProjection(ctx, f, "s", "i")
	require.NoError(t, err)
	assert.Equal(t, 0, p.Balance(), "a claim must not credit the balance")
	assert.Empty(t, p.Records(), "a claim is not a minted catch")
}
