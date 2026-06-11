package ledger_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/ledger"
)

func bandwidthLog(t *testing.T) *ledger.Log {
	t.Helper()
	f, err := fabric.Start(context.Background(), t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	return ledger.Bind(f, "bw", "i")
}

func TestLog_bandwidthIsZeroWithNoEvents(t *testing.T) {
	t.Parallel()
	log := bandwidthLog(t)

	bw, err := log.Bandwidth()
	require.NoError(t, err)
	assert.Equal(t, 0, bw, "an economy that has unblocked nothing has no attention bandwidth")
}

func TestLog_anOpenBlockEarnsNoBandwidth(t *testing.T) {
	t.Parallel()
	log := bandwidthLog(t)
	base := time.Unix(1_700_000_000, 0)

	require.NoError(t, log.AppendBlock("wo:1", base))

	bw, err := log.Bandwidth()
	require.NoError(t, err)
	assert.Equal(t, 0, bw, "a block the Lead has not yet cleared earns nothing — only an UNBLOCK pays")
}

// The award folds BOTH axes the Lead chose: a throughput base (you cleared a
// block at all) plus a latency bonus (how fast). Each award redeems against one
// logged block→unblock pair, so it is grounded in facts, never inferred.
func TestLog_clearingABlockEarnsThroughputBasePlusLatencyBonus(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		latency time.Duration
		want    int
	}{
		{"fast unblock earns base + full bonus", 30 * time.Second, 3},
		{"at the fast threshold still fast", 2 * time.Minute, 3},
		{"medium unblock earns base + partial bonus", 10 * time.Minute, 2},
		{"at the medium threshold still medium", 15 * time.Minute, 2},
		{"slow unblock earns the throughput base only", 1 * time.Hour, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			log := bandwidthLog(t)
			base := time.Unix(1_700_000_000, 0)
			require.NoError(t, log.AppendBlock("wo:1", base))
			require.NoError(t, log.AppendUnblock("wo:1", base.Add(tt.latency)))

			bw, err := log.Bandwidth()
			require.NoError(t, err)
			assert.Equal(t, tt.want, bw)
		})
	}
}

func TestLog_bandwidthSumsAcrossClearedBlocks(t *testing.T) {
	t.Parallel()
	log := bandwidthLog(t)
	base := time.Unix(1_700_000_000, 0)

	require.NoError(t, log.AppendBlock("wo:1", base))
	require.NoError(t, log.AppendUnblock("wo:1", base.Add(30*time.Second))) // fast → 3
	require.NoError(t, log.AppendBlock("wo:2", base))
	require.NoError(t, log.AppendUnblock("wo:2", base.Add(1*time.Hour))) // slow → 1

	bw, err := log.Bandwidth()
	require.NoError(t, err)
	assert.Equal(t, 4, bw, "bandwidth is the sum of awards across every cleared block")
}

// A block is cleared ONCE: a duplicate unblock for the same id (a double-submit)
// must never pay twice — the award redeems against the FIRST clearing.
func TestLog_duplicateUnblockDoesNotDoubleCount(t *testing.T) {
	t.Parallel()
	log := bandwidthLog(t)
	base := time.Unix(1_700_000_000, 0)

	require.NoError(t, log.AppendBlock("wo:1", base))
	require.NoError(t, log.AppendUnblock("wo:1", base.Add(30*time.Second)))
	require.NoError(t, log.AppendUnblock("wo:1", base.Add(45*time.Second)))

	bw, err := log.Bandwidth()
	require.NoError(t, err)
	assert.Equal(t, 3, bw, "a second unblock for the same block earns nothing — it is already cleared")
}

// An unblock latency can never go negative even on a clock skew between the block
// and unblock stamps — the bonus floors at the slow tier (the throughput base).
func TestLog_negativeLatencyFloorsAtTheThroughputBase(t *testing.T) {
	t.Parallel()
	log := bandwidthLog(t)
	base := time.Unix(1_700_000_000, 0)

	require.NoError(t, log.AppendBlock("wo:1", base))
	require.NoError(t, log.AppendUnblock("wo:1", base.Add(-5*time.Minute)))

	bw, err := log.Bandwidth()
	require.NoError(t, err)
	assert.Equal(t, 1, bw, "a skewed (negative) latency never pays more than the throughput base")
}
