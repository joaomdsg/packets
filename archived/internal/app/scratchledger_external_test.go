package app_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/ledger"
)

// scratchLog binds a Log to a throwaway fabric for unit-testing a projection in
// isolation (no server). Each call gets its own embedded server, closed on cleanup.
func scratchLog(t *testing.T) *ledger.Log {
	t.Helper()
	f, err := fabric.Start(context.Background(), t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	return ledger.Bind(f, "scratch", "i")
}
