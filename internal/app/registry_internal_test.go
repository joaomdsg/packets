package app

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/ledger"
)

func TestLiveRegistry_resolvesAKeyElseFallsBackToTheDefault(t *testing.T) {
	// Internal test (package app): exercises the unexported keyed lookup. NOT
	// parallel — it shares the package-global liveReg with the other live tests.
	logPath := filepath.Join(t.TempDir(), "catches.jsonl")
	log, err := ledger.Open(logPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })
	setLiveState(LiveConfig{MaxConcurrent: 3, LedgerPath: logPath}, log)

	cfg, gotLog := readLiveState(defaultSessionKey)
	assert.Equal(t, 3, cfg.MaxConcurrent, "a registered key resolves to its own entry (the hit path)")
	assert.Same(t, log, gotLog, "the hit returns the registered ledger, not a copy")

	cfg2, gotLog2 := readLiveState("unregistered-tab-id")
	assert.Equal(t, 3, cfg2.MaxConcurrent, "an unknown key falls back to the default entry — preserves single-card behavior")
	assert.Same(t, log, gotLog2, "the fallback resolves to the one seeded ledger")

	sem := cycleSem("unregistered-tab-id")
	require.NotNil(t, sem, "the admission sem resolves through the same fallback")
	assert.Equal(t, 3, cap(sem), "the fallback sem carries the registered cap")
	assert.Equal(t, cycleSem(defaultSessionKey), sem, "fallback yields the SAME sem instance — one entry, one cap, not a copy")
}
