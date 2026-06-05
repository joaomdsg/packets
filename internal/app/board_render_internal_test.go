package app

import (
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via"
	"github.com/go-via/via/vt"

	"github.com/joaomdsg/agntpr/internal/ledger"
)

func TestBoardCard_rendersACalmRowPerCardAsActivityNeverLeverage(t *testing.T) {
	// The fleet board is the cross-card surface: it lists every registered session
	// with its activity (queued/running/done), framed as ACTIVITY — it must NEVER
	// label or rank cards by leverage/priority (blocked-downstream is uncomputable;
	// faking it would mis-point the Lead's attention). NOT parallel (shared globals).
	t1, t2 := woTargetN(1), woTargetN(2)
	own := ownTargetOf(LiveConfig{BaseRev: "own-b-bcB", FixRev: "own-f", Anchor: anchorForCap()})
	logB := boardSession(t, "bcB", 3, []ledger.Target{t1, t2})
	require.NoError(t, logB.AppendDispatch("d", t1, own))
	require.NoError(t, logB.AppendDispatch("d", t2, own))
	boardSession(t, "bcA", 1, nil)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = defLog.Close() })

	// The board is a static read-only projection — its content is the initial
	// rendered page (the GET body), not an SSE patch stream.
	f := vt.NewClient(t, server, "/board").HTML()
	require.Contains(t, f, `data-state="board"`, "the board is mounted and rendered at /board")
	require.Contains(t, f, "bcB", "the board lists the bcB session row")
	require.Contains(t, f, "bcA", "the board lists the bcA session row")
	require.Contains(t, f, "queued", "rows surface queued ACTIVITY")
	require.Contains(t, f, "misses", "rows surface the honest-loss MISS tally — a bet that didn't pay is visible, not discarded")
	low := strings.ToLower(f)
	for _, banned := range []string{"leverage", "priority", "rank", "highest-impact"} {
		require.NotContainsf(t, low, banned, "the board is ACTIVITY, never %q — leverage is uncomputable and must never be faked", banned)
	}
}
