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

// While a live work-order fills, the card shows the agent's LATEST activity
// ("editing auth.go") so the Lead watches a real worker move in real time — and
// it clears when the fill is done (absent on dead-air, no lingering ghost beat).
// NOT parallel (shared liveReg).
func TestLiveCard_showsTheLiveAgentsLatestActivityWhileFilling(t *testing.T) {
	resetConsumersForTest()
	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "wact", "i")
	registerSession("wact", LiveConfig{RepoDir: ".", BaseRev: "b", FixRev: "f", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	e := lookupLiveEntry("wact")
	require.NotNil(t, e)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = defLog.Close() })

	// Filling but no beat yet — dead-air. The activity line must be ABSENT (no
	// spinner, no empty ghost line); only a real beat surfaces it.
	e.startFill(3)
	bodyDeadAir := bodyOf(vt.NewClient(t, server, "/?key=wact").HTML())
	require.NotContains(t, bodyDeadAir, "order-activity", "no activity line during dead-air (filling, no beat yet)")

	// The harness streams a beat into the per-session buffer → it shows live.
	e.addActivityBeat("editing auth.go")
	body := bodyOf(vt.NewClient(t, server, "/?key=wact").HTML())
	require.Contains(t, body, "order-activity", "the activity line appears once a beat streams")
	require.Contains(t, body, "editing auth.go", "the card shows the agent's latest live activity while filling")

	// Once the fill completes, the live activity line clears (no ghost beat).
	e.endFill()
	body2 := bodyOf(vt.NewClient(t, server, "/?key=wact").HTML())
	require.NotContains(t, body2, "editing auth.go", "the activity line clears when the order is done")
}
