package app

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via"
	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/ledger"
)

// The attention-bandwidth meter is the second economy (R91); the card must SHOW it
// beside the balance so the Lead sees the responsiveness they have earned, not just
// the catch balance. A cleared block (fast) earns 3 — the card must surface it.
// NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_showsTheEarnedAttentionBandwidth(t *testing.T) {
	resetConsumersForTest()
	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "bwcard", "i")

	base := time.Unix(1_700_000_000, 0)
	require.NoError(t, log.AppendBlock("wo:1", base))
	require.NoError(t, log.AppendUnblock("wo:1", base.Add(30*time.Second))) // fast clear → 3
	registerSession("bwcard", LiveConfig{RepoDir: ".", BaseRev: "b", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = defLog.Close() })

	body := bodyOf(vt.NewClient(t, server, "/?key=bwcard").HTML())
	require.Contains(t, body, `data-state="bandwidth"`, "the card renders the attention-bandwidth meter")
	require.Contains(t, body, `data-bandwidth="3"`, "a fast-cleared block surfaces as 3 earned bandwidth")
}
