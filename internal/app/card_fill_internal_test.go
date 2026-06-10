package app

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via"
	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/ledger"
)

// "Watch it fill": when a funded work-order is running, the card shows it filling
// LIVE — the cycle's beats accruing as the oracle works (the felt "watch the work
// happen" beat), not just queued→running→done counts. The filling order's beats are
// buffered per-session (the background drain has no request ctx, so it writes a
// buffer the card's Stream polls — mirroring the dispatch-tally poll). NOT parallel
// (shared liveReg).
func TestLiveCard_showsAnOrderFillingLiveWithItsBeats(t *testing.T) {
	resetConsumersForTest()
	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "wf", "i")
	registerSession("wf", LiveConfig{RepoDir: ".", BaseRev: "b", FixRev: "f", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	// Simulate an order mid-fill: the runner has started it and beats are accruing.
	e := lookupLiveEntry("wf")
	require.NotNil(t, e)
	e.startFill(3)
	e.addFillBeat("oracle-base")
	e.addFillBeat("oracle-fix")

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = defLog.Close() })

	body := bodyOf(vt.NewClient(t, server, "/?key=wf").HTML())
	require.Contains(t, body, "filling WO#3", "the card shows which order is filling, live")
	require.Contains(t, body, "oracle-fix", "with the cycle's beats as the oracle works")

	// Once the fill completes, the live filling row clears.
	e.endFill()
	body2 := bodyOf(vt.NewClient(t, server, "/?key=wf").HTML())
	require.NotContains(t, body2, "filling WO#3", "the filling row clears when the order is done")
}
