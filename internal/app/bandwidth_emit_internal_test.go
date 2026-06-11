package app

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-via/via"
	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/mutation"
)

// A surfaced review question is the producer asking for the Lead's input — a
// block. recordQuestionBlocks logs one block per question id, ONCE (a re-surfaced
// question from a later cycle never re-blocks), so the bandwidth interval starts
// when the question first appears. NOT parallel (shared globals).
func TestLiveEntry_recordQuestionBlocksLogsOneBlockPerQuestion(t *testing.T) {
	resetConsumersForTest()
	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "blk", "i")
	registerSession("blk", LiveConfig{RepoDir: ".", BaseRev: "b", Anchor: anchorForCap()}, log)

	e := lookupLiveEntry("blk")
	qs := []mutation.Finding{{File: "main.go", Line: 6, Outcome: mutation.Survived}}
	e.recordQuestionBlocks(qs)
	e.recordQuestionBlocks(qs) // a later cycle re-finds the same survivor — must not re-block

	// An open block (no unblock yet) earns nothing, but it is logged: clearing it
	// later will pay, proving the block was recorded.
	bw, err := log.Bandwidth()
	require.NoError(t, err)
	assert.Equal(t, 0, bw, "an open (un-cleared) question block earns no bandwidth yet")
}

// Clearing a surfaced question — the Lead answered it — is an unblock, and the
// block→unblock pair earns bandwidth weighted by how fast they answered. NOT
// parallel (shared globals).
func TestLiveEntry_clearingASurfacedQuestionEarnsBandwidth(t *testing.T) {
	resetConsumersForTest()
	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "clr", "i")
	registerSession("clr", LiveConfig{RepoDir: ".", BaseRev: "b", Anchor: anchorForCap()}, log)

	e := lookupLiveEntry("clr")
	e.recordQuestionBlocks([]mutation.Finding{{File: "main.go", Line: 6, Outcome: mutation.Survived}})
	e.recordQuestionUnblock("main.go", 6)

	bw, err := log.Bandwidth()
	require.NoError(t, err)
	assert.Positive(t, bw, "answering a surfaced question clears its block and earns attention bandwidth")
}

// An unblock for a question that was never surfaced (no block) earns nothing — the
// award only ever redeems against a logged block→unblock pair. NOT parallel.
func TestLiveEntry_unblockWithoutAPriorBlockEarnsNothing(t *testing.T) {
	resetConsumersForTest()
	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "noblk", "i")
	registerSession("noblk", LiveConfig{RepoDir: ".", BaseRev: "b", Anchor: anchorForCap()}, log)

	e := lookupLiveEntry("noblk")
	e.recordQuestionUnblock("ghost.go", 9)

	bw, err := log.Bandwidth()
	require.NoError(t, err)
	assert.Equal(t, 0, bw, "an unblock with no matching block earns nothing")
}

// End to end through the real review action: a surfaced (blocked) question that the
// Lead answers with a KILLING test clears the block and earns attention bandwidth,
// while the catch balance stays untouched (the unblock moves only the second meter).
// NOT parallel (shared liveReg + the re-run seam).
func TestReviewCard_aKillingAnswerEarnsAttentionBandwidth(t *testing.T) {
	resetConsumersForTest()
	restore := rerunWithOverlay
	t.Cleanup(func() { rerunWithOverlay = restore })
	rerunWithOverlay = func(_ context.Context, _, _, _ string, _ int, _ []string, _ map[string]string) ([]mutation.Finding, error) {
		return nil, nil // the reviewer's test killed the mutant — no survivors remain
	}

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	e := lookupLiveEntry(defaultSessionKey)
	require.NotNil(t, e)
	q := []mutation.Finding{{File: "main.go", Line: 6, Outcome: mutation.Survived, Message: "mutated >= to >"}}
	e.setFindings(q)
	e.recordQuestionBlocks(q) // the question surfaced — its block interval is open
	balBefore, _ := log.Balance()

	tc := vt.NewClient(t, server, "/review")
	require.Equal(t, 200, tc.Action((&ReviewCard{}).AnswerQuestion).
		WithSignal("answerfile", "main.go").
		WithSignal("answerline", "6").
		WithSignal("answertest", "package main\nimport \"testing\"\nfunc TestB(t *testing.T){}\n").
		Fire())

	bw, err := log.Bandwidth()
	require.NoError(t, err)
	assert.Positive(t, bw, "a killing answer clears the question's block and earns bandwidth")
	balAfter, _ := log.Balance()
	assert.Equal(t, balBefore, balAfter, "FIREWALL: the unblock moves only the bandwidth meter, never the catch balance")
}
