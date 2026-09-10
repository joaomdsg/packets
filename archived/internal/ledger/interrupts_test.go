package ledger_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The weekly interrupt KPI must count REAL raised
// interrupts in a rolling window, not the earned-bandwidth fold Bandwidth()
// already provides — a block that is still open (uncleared) never earns
// bandwidth, but it already interrupted the Lead the moment it was raised.
func TestLog_interruptsSinceCountsRealInterruptsInTheTrailingWindow(t *testing.T) {
	t.Parallel()
	log := bandwidthLog(t)
	base := time.Unix(1_700_000_000, 0)

	require.NoError(t, log.AppendBlock("wo:1", base))
	require.NoError(t, log.AppendBlock("wo:2", base.Add(1*time.Hour)))
	require.NoError(t, log.AppendBlock("wo:3", base.Add(2*time.Hour)))

	tests := []struct {
		name  string
		since time.Time
		want  int
	}{
		{"a window starting before every block counts all three", base.Add(-1 * time.Minute), 3},
		{"a window starting exactly at a block's stamp counts it (inclusive lower bound)", base.Add(1 * time.Hour), 2},
		{"a window starting after every block counts none", base.Add(3 * time.Hour), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := log.InterruptsSince(tt.since)
			require.NoError(t, err)
			assert.Equal(t, tt.want, n)
		})
	}
}

func TestLog_interruptsSinceIsZeroWithNoBlocksLogged(t *testing.T) {
	t.Parallel()
	log := bandwidthLog(t)

	n, err := log.InterruptsSince(time.Unix(1_700_000_000, 0))
	require.NoError(t, err)
	assert.Equal(t, 0, n, "a session that has never interrupted the Lead counts zero")
}

// Bandwidth() only pays for a CLEARED block, so it would silently miss an
// interrupt the Lead has not yet answered — InterruptsSince must not inherit
// that blind spot, since the interrupt already happened regardless of
// whether it's since been cleared.
func TestLog_interruptsSinceCountsAnOpenBlockEvenThoughItEarnsNoBandwidth(t *testing.T) {
	t.Parallel()
	log := bandwidthLog(t)
	base := time.Unix(1_700_000_000, 0)
	require.NoError(t, log.AppendBlock("wo:1", base))

	n, err := log.InterruptsSince(base.Add(-1 * time.Minute))
	require.NoError(t, err)
	assert.Equal(t, 1, n, "an uncleared block still interrupted the Lead — it must count")

	bw, err := log.Bandwidth()
	require.NoError(t, err)
	assert.Equal(t, 0, bw, "sanity: the same open block earns nothing on the separate bandwidth meter")
}

// A block is raised once per id (recordQuestionBlocks in internal/app/live.go
// guards against re-blocking a still-open question) — a duplicate AppendBlock
// for the same id must not inflate the interrupt count, mirroring the
// first-stamp-wins dedup Bandwidth() already relies on.
func TestLog_interruptsSinceDoesNotDoubleCountADuplicateBlockForTheSameID(t *testing.T) {
	t.Parallel()
	log := bandwidthLog(t)
	base := time.Unix(1_700_000_000, 0)
	require.NoError(t, log.AppendBlock("wo:1", base))
	require.NoError(t, log.AppendBlock("wo:1", base.Add(1*time.Minute)))

	n, err := log.InterruptsSince(base.Add(-1 * time.Minute))
	require.NoError(t, err)
	assert.Equal(t, 1, n, "a re-raised block for an id already interrupted must not count twice")
}

// Clearing a block (AppendUnblock) must not itself register as a SECOND
// interrupt, nor erase the original one — the interrupt count reflects how
// many questions were RAISED, independent of how many were later cleared.
func TestLog_interruptsSinceCountsAClearedBlockExactlyOnce(t *testing.T) {
	t.Parallel()
	log := bandwidthLog(t)
	base := time.Unix(1_700_000_000, 0)
	require.NoError(t, log.AppendBlock("wo:1", base))
	require.NoError(t, log.AppendUnblock("wo:1", base.Add(1*time.Minute)))

	n, err := log.InterruptsSince(base.Add(-1 * time.Minute))
	require.NoError(t, err)
	assert.Equal(t, 1, n, "a cleared block was still exactly one interrupt — clearing it neither doubles nor erases the count")
}
