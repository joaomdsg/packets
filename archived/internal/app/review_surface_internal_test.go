package app

import (
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/mutation"
)

// The card's badge only COUNTS the open questions; the Lead needs a surface to
// READ them. /review renders the session's open "question:" threads — each a
// surviving mutant the fix oracle found — anchored to its File:Line with the
// Conventional-Comment body. NOT parallel (shared liveReg/liveFabric).
func TestReviewCard_rendersOpenQuestionThreadsForASession(t *testing.T) {
	resetConsumersForTest()
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	viaApp, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server = httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = log.Close() })

	// Seed the default session's open-questions cache (as the connect cycle would).
	e := lookupLiveEntry(defaultSessionKey)
	require.NotNil(t, e)
	e.setFindings([]mutation.Finding{
		{File: "auth.go", Line: 12, Outcome: mutation.Survived, Message: "mutated >= to >; tests still pass"},
		{File: "auth.go", Line: 30, Outcome: mutation.Undetermined, Message: "mutated + to -; suite timed out"},
	})

	body := bodyOf(vt.NewClient(t, server, "/review").HTML())
	require.Contains(t, body, "review-thread", "the open questions render as anchored threads")
	require.Contains(t, body, "auth.go:12", "a thread is anchored to its file:line")
	require.Contains(t, body, "question: mutated", "with the Conventional-Comment body (question: tag)")
	require.Contains(t, body, "tests still pass", "carrying the finding's message")
	require.Contains(t, body, "auth.go:30", "every open question renders, including the undetermined one")
}

// A long-running session's OLDEST packet must still be inspectable. packetForOrder
// folds ALL of a session's dispatches unboundedly, so once more than 50 orders exist
// the first one is a packet the Console can name but orderTarget must still resolve
// the SAME order's revisions/anchor — otherwise the Inspector renders a packet with no
// diff to show, a silent split between "found" and "inspectable" for the same order.
// NOT parallel (shared liveReg/liveFabric).
func TestPacketTarget_resolvesTheOldestPacketEvenPastTheRecentSendWindow(t *testing.T) {
	resetConsumersForTest()
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	// 56 dispatches need 56 bandwidth; each cleared block/unblock pair earns up to 3.
	now := time.Now()
	for i := 0; i < 20; i++ {
		id := "seed-bw-" + strconv.Itoa(i)
		require.NoError(t, log.AppendBlock(id, now))
		require.NoError(t, log.AppendUnblock(id, now))
	}

	require.NoError(t, log.AppendLiveSend("liveorder", ledger.Target{BaseRev: "b", FixRev: "oldest-fix", Path: "old.go", Line: 1}, ledger.Target{}))
	for i := 0; i < 55; i++ {
		require.NoError(t, log.AppendLiveSend("liveorder", ledger.Target{BaseRev: "b", FixRev: "later-fix", Path: "later.go", Line: 1}, ledger.Target{}))
	}

	tgt, ok := orderTarget(log, 1)
	require.True(t, ok, "order #1 must still resolve once 55 later orders exist on the same session")
	require.Equal(t, "oldest-fix", tgt.FixRev, "the resolved target is order #1's OWN revisions, not a later order's")

	newest, ok := orderTarget(log, 56)
	require.True(t, ok, "the newest order must still resolve too")
	require.Equal(t, "later-fix", newest.FixRev, "the newest order's own revisions, unaffected by widening the window")
}
