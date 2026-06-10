package app

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via"
	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/mutation"
)

// The Monaco review editor (a client-side island, built in a later slice) mounts
// into a DOM subtree the server must NOT clobber on SSE re-render, and it reads the
// anchored "question:" threads as structured data — not by scraping the server's
// human-readable text. So /review emits (1) a data-ignore-morph island container
// for the editor to mount into, and (2) a machine-readable JSON payload of the same
// threads the server text renders, from the SAME projection — so the editor and the
// text can never disagree. This is the SERVER CONTRACT the editor depends on; the
// editor's own rendering is the client island, deliberately untested here. NOT
// parallel (shared liveReg/liveFabric).
func TestReviewCard_feedsAnchoredThreadsAsAStructuredIslandPayload(t *testing.T) {
	resetConsumersForTest()
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
	e.setFindings([]mutation.Finding{
		{File: "auth.go", Line: 12, Outcome: mutation.Survived, Message: "mutated >= to >; tests still pass"},
		{File: "auth.go", Line: 30, Outcome: mutation.Undetermined, Message: "mutated + to -; suite timed out"},
	})

	body := bodyOf(vt.NewClient(t, server, "/review").HTML())

	// (1) the island container the Monaco editor will mount into, shielded from SSE
	// morphing so the editor's own DOM is never clobbered by a re-render.
	require.Contains(t, body, "review-editor", "an island container for the editor to mount into")
	require.Contains(t, body, "data-ignore-morph", "the editor subtree is preserved across SSE re-renders")

	// (2) the structured payload — a JSON application block the editor reads, carrying
	// the SAME threads as the server text (one projection, no drift). Assert the exact
	// JSON shape the client contract depends on (load-bearing: strip it and the editor
	// has no data).
	require.Contains(t, body, `type="application/json"`, "the threads are emitted as a machine-readable payload")
	require.Contains(t, body, `"file":"auth.go"`, "each thread carries its file")
	require.Contains(t, body, `"line":12`, "anchored to its line for an editor decoration")
	require.Contains(t, body, `"tag":"question"`, "the Conventional-Comment tag")
	require.Contains(t, body, "tests still pass", "the finding's message body")
	require.Contains(t, body, `"line":30`, "every open question is in the payload, including the undetermined one")

	// The payload must be VALID JSON the client can parse — extract and unmarshal it.
	payload := jsonPayloadBetween(t, body, `type="application/json"`)
	island := decodeReviewIsland(t, payload)
	require.Len(t, island.Threads, 2, "both open questions are in the payload")
	require.Equal(t, "auth.go", island.Threads[0].File)
	require.Equal(t, 12, island.Threads[0].Line)
	require.Equal(t, "question", island.Threads[0].Tag)
}

// decodeReviewIsland parses the editor payload into the impl's reviewIsland
// contract (same package), so the test asserts against the real shape the client
// receives.
func decodeReviewIsland(t *testing.T, payload string) reviewIsland {
	t.Helper()
	var island reviewIsland
	require.NoError(t, json.Unmarshal([]byte(payload), &island), "the payload is valid JSON")
	return island
}

// The editor renders the reviewed FILE with the questions anchored to its lines, so
// it needs the file's SOURCE, not just the line numbers. The payload carries each
// referenced file's content at the reviewed (fix) revision, read through an
// injected git-show seam. Without the source the editor would show line numbers
// against a blank document. NOT parallel (shared liveReg + the reader seam).
func TestReviewCard_feedsTheReviewedFileSourceForTheEditorToRender(t *testing.T) {
	resetConsumersForTest()
	restore := reviewFileReader
	t.Cleanup(func() { reviewFileReader = restore })
	const src = "package auth\n\nfunc ok(n int) bool {\n\treturn n >= 18\n}\n"
	reviewFileReader = func(_ context.Context, _, _, path string) (string, error) {
		if path == "auth.go" {
			return src, nil
		}
		return "", errors.New("unexpected path")
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
	e.setFindings([]mutation.Finding{
		{File: "auth.go", Line: 4, Outcome: mutation.Survived, Message: "mutated >= to >; tests still pass"},
	})

	body := bodyOf(vt.NewClient(t, server, "/review").HTML())
	island := decodeReviewIsland(t, jsonPayloadBetween(t, body, `type="application/json"`))
	require.Equal(t, src, island.Files["auth.go"], "the editor gets the reviewed file's source to render")
	require.Len(t, island.Threads, 1, "the threads still ride alongside the file source")
	require.Equal(t, 4, island.Threads[0].Line)
}

// If a referenced file's source can't be read at the reviewed revision (a lost
// anchor, a deleted file, a transient git error), the payload OMITS that file's
// content rather than emitting an empty-string lie — the editor still gets the
// threads and degrades to anchoring against no source, and the surface never
// breaks. NOT parallel (shared liveReg + the reader seam).
func TestReviewCard_omitsFileSourceItCannotReadRatherThanLie(t *testing.T) {
	resetConsumersForTest()
	restore := reviewFileReader
	t.Cleanup(func() { reviewFileReader = restore })
	reviewFileReader = func(_ context.Context, _, _, _ string) (string, error) {
		return "", errors.New("file not found at rev")
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
	e.setFindings([]mutation.Finding{
		{File: "gone.go", Line: 7, Outcome: mutation.Survived, Message: "mutated < to <=; tests still pass"},
	})

	body := bodyOf(vt.NewClient(t, server, "/review").HTML())
	island := decodeReviewIsland(t, jsonPayloadBetween(t, body, `type="application/json"`))
	_, present := island.Files["gone.go"]
	require.False(t, present, "an unreadable file's source is omitted, not emitted as an empty-string lie")
	require.Len(t, island.Threads, 1, "the threads still render so the question is not lost")
}

// The island is inert without the client editor: /review must ship the Monaco
// loader + a bootstrap that mounts the read-only editor over the payload. This
// asserts the WIRING is emitted (the loader script + the mount call) — the editor's
// actual rendering is the client island, browser-verified, not asserted here. The
// server-rendered text threads remain regardless, so a JS failure degrades to them.
// NOT parallel (shared liveReg/liveFabric).
func TestReviewCard_shipsTheMonacoEditorWiringOverThePayload(t *testing.T) {
	resetConsumersForTest()
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
	e.setFindings([]mutation.Finding{
		{File: "auth.go", Line: 12, Outcome: mutation.Survived, Message: "mutated >= to >; tests still pass"},
	})

	body := bodyOf(vt.NewClient(t, server, "/review").HTML())
	require.Contains(t, body, "cdn.jsdelivr.net/npm/monaco-editor", "the Monaco loader is shipped from a pinned CDN")
	require.Contains(t, body, "monaco.editor.create", "a bootstrap mounts the read-only editor over the payload")
	require.Contains(t, body, "review-threads-data", "the bootstrap reads the structured payload, not the server text")
	// The server-rendered text thread survives as the no-JS / failure fallback.
	require.Contains(t, body, "review-thread__body", "the text threads remain as the progressive-enhancement fallback")
}

// With no open questions there is nothing for the editor to show, so the island and
// its payload are omitted entirely — the surface stays calm (the existing empty
// state is the only thing rendered), and no editor is scaffolded over an empty set.
// NOT parallel.
func TestReviewCard_omitsTheEditorIslandWhenNoOpenQuestions(t *testing.T) {
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
	require.NotContains(t, body, "review-editor", "no editor island when there are no questions to show")
	require.NotContains(t, body, `type="application/json"`, "no payload when there is nothing to feed the editor")
}

// jsonPayloadBetween extracts the text inside the first <script ...type="application/json"...>
// ...</script> block, so the test can parse the real payload the client would.
func jsonPayloadBetween(t *testing.T, body, marker string) string {
	t.Helper()
	m := strings.Index(body, marker)
	require.GreaterOrEqual(t, m, 0, "payload script present")
	open := strings.Index(body[m:], ">")
	require.GreaterOrEqual(t, open, 0, "payload script opening tag closes")
	start := m + open + 1
	end := strings.Index(body[start:], "</script>")
	require.GreaterOrEqual(t, end, 0, "payload script closes")
	return body[start : start+end]
}
