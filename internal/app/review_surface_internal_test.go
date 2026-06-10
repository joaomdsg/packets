package app

import (
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via"
	"github.com/go-via/via/vt"

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
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
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

// With no surviving mutants (or before a cycle ran), /review shows a calm empty
// state — never a fabricated or alarming surface. NOT parallel.
func TestReviewCard_showsCalmEmptyStateWhenNoOpenQuestions(t *testing.T) {
	resetConsumersForTest()
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	body := bodyOf(vt.NewClient(t, server, "/review").HTML())
	require.NotContains(t, body, "review-thread", "no threads when the oracle left no survivors")
	require.Contains(t, body, "No open questions", "a calm empty state, not a blank or alarming surface")
}
