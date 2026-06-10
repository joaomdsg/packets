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

	"github.com/joaomdsg/packets/internal/mutation"
	"github.com/joaomdsg/packets/internal/pipe"
	"github.com/joaomdsg/packets/internal/reanchor"
)

// A verdict can read green ("tested") while the oracle still left surviving
// mutants — honest test gaps the green hides. The card must surface a gated,
// calm "N open questions" badge so the Lead knows there is review work the verdict
// alone doesn't show. NOT parallel (shared globals).
func TestLiveCard_showsGatedOpenQuestionCountWhenOracleLeavesSurvivors(t *testing.T) {
	restore := resolveCycle
	t.Cleanup(func() { resolveCycle = restore })
	resolveCycle = func(_ context.Context, _, _, _, _ string, _ reanchor.Anchor, _ []string, _, _ bool, _ chan<- pipe.TraceEvent) (Resolution, error) {
		return Resolution{
			Verdict: "tested",
			Findings: []mutation.Finding{
				{File: "a.go", Line: 4, Outcome: mutation.Survived, Message: "mutated >= to >; tests still pass"},
				{File: "a.go", Line: 9, Outcome: mutation.Survived, Message: "mutated + to -; tests still pass"},
			},
		}, nil
	}

	logPath := filepath.Join(t.TempDir(), "c.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: logPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	tc := vt.NewClient(t, server, "/")
	frames, cancel := tc.SSE()
	defer cancel()
	frame := vt.AwaitFrame(t, frames, 10*time.Second, "2 open questions")
	require.Contains(t, frame, "review-questions", "the open-question count renders with its class hook")
}

// When the oracle left NO survivors, the verdict's green is honest — no question
// badge clutters the card. NOT parallel (shared globals).
func TestLiveCard_omitsTheQuestionBadgeWhenNoSurvivors(t *testing.T) {
	restore := resolveCycle
	t.Cleanup(func() { resolveCycle = restore })
	resolveCycle = func(_ context.Context, _, _, _, _ string, _ reanchor.Anchor, _ []string, _, _ bool, _ chan<- pipe.TraceEvent) (Resolution, error) {
		return Resolution{Verdict: "tested"}, nil // no findings — green is honest
	}

	logPath := filepath.Join(t.TempDir(), "c.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: logPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	tc := vt.NewClient(t, server, "/")
	frames, cancel := tc.SSE()
	defer cancel()
	// The verdict resolving to "tested" proves the cycle finished and the Questions
	// cell was written ("0") — that same frame must carry no question badge.
	frame := vt.AwaitFrame(t, frames, 10*time.Second, `data-state="tested"`)
	require.NotContains(t, frame, "review-questions", "a clean verdict shows no open-question badge")
}
