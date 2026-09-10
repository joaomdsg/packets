package app

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/mutation"
)

// An order that DID leave open questions keeps its "N open questions" drill link
// (the test-debt affordance) and must NOT also sprout an "inspect diffs" link —
// the two are one link with different framing, never both. This locks the
// existing Questions>0 path against the new settled-link change.
// NOT parallel (shared liveReg/liveFabric).
func TestLiveCard_PacketWithOpenQuestionsKeepsTheQuestionsLinkNotInspect(t *testing.T) {
	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "qlinkc", "i")

	require.NoError(t, log.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 100, ReasonTag: "catch"}))
	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	require.NoError(t, log.AppendSend("d1", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "alpha.go", Line: 7}, own))
	require.NoError(t, log.AppendStatus(1, "done"))
	registerSession("qlinkc", LiveConfig{BaseRev: "own-b-qlinkc", FixRev: "own-f", Anchor: anchorForCap()}, log)
	e := lookupLiveEntry("qlinkc")
	require.NotNil(t, e)
	// The filled order left a surviving mutant → one open question for PKT#1.
	e.setOrderFindings(1, []mutation.Finding{{File: "alpha.go", Line: 7, Outcome: mutation.Survived, Message: "mutated >= to >"}})

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server = httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	body := bodyOf(vt.NewClient(t, server, "/?key=qlinkc").HTML())
	require.Contains(t, body, `href="/review?key=qlinkc&amp;wo=1"`, "the order still links into its review")
	require.Contains(t, body, "open questions", "a questioned order keeps the test-debt framing")
	require.NotContains(t, body, "inspect diffs",
		"an order with open questions shows ONE link (open questions), never also an inspect link")
}
