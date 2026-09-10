package app

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/diff"
	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/ledger"
)

// A durable annotation on a file the packet changed — and its reply — must
// surface in the Inspector rail, folded from the log and scoped to this order's
// diff. This is the end-to-end proof that the fold + render are actually wired
// into the review surface, not just unit-tested in isolation. NOT parallel
// (swaps the diff seam + shares liveReg/liveFabric).
func TestReviewSurface_durableAnnotationAndItsReplyShowInTheRail(t *testing.T) {
	resetConsumersForTest()
	// Stub the differ so the annotation's file counts as changed for this order,
	// so the file-scoped fold keeps it.
	swapTreeSeams(t, []string{"alpha.go"}, nil,
		diff.Diff{Files: []diff.FileDiff{{Path: "alpha.go", Added: 1, Deleted: 0}}})

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "annrail", "i")
	fundSend(t, log, "d1", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "alpha.go", Line: 7})
	require.NoError(t, log.AppendAnnotation(ledger.AnnotationRecord{
		ID: "root", File: "alpha.go", StartLine: 7, Author: "lead", Body: "guard the negative amount"}))
	require.NoError(t, log.AppendAnnotation(ledger.AnnotationRecord{
		ID: "r1", ParentID: "root", File: "alpha.go", StartLine: 7, Author: "agent", Body: "added the guard"}))
	registerSession("annrail", LiveConfig{RepoDir: ".", BaseRev: "own-b", FixRev: "own-f", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server := httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	body := bodyOf(vt.NewClient(t, server, "/review?key=annrail&wo=1").HTML())
	assert.Contains(t, body, "guard the negative amount", "the durable annotation renders in the rail")
	assert.Contains(t, body, "added the guard", "its reply renders nested beneath it")
	assert.Contains(t, body, `annotation-card__reply"`, "the reply is a nested reply row, not a top-level card")
	// The reply form is present and its input is bound (signals framework-initialized
	// on the real render), so the Lead can answer the thread in place.
	assert.Contains(t, body, "annotation-card__reply-form", "the card offers a reply form")
	assert.Contains(t, body, `data-bind="replytext"`, "the reply input is bound to the reply signal")
	assert.Contains(t, body, "ReplyToAnnotation", "wired to the reply action")
}

// An annotation on a file this packet did NOT change must not leak onto its
// rail — annotations are scoped to the diff being inspected.
func TestReviewSurface_annotationOnAnUnrelatedFileIsNotShown(t *testing.T) {
	resetConsumersForTest()
	swapTreeSeams(t, []string{"alpha.go"}, nil,
		diff.Diff{Files: []diff.FileDiff{{Path: "alpha.go", Added: 1, Deleted: 0}}})

	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "annscope", "i")
	fundSend(t, log, "d1", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "alpha.go", Line: 7})
	require.NoError(t, log.AppendAnnotation(ledger.AnnotationRecord{
		ID: "x", File: "unrelated.go", StartLine: 1, Author: "lead", Body: "note on another file"}))
	registerSession("annscope", LiveConfig{RepoDir: ".", BaseRev: "own-b", FixRev: "own-f", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	viaApp, defLog, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	})
	require.NoError(t, err)
	server := httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	body := bodyOf(vt.NewClient(t, server, "/review?key=annscope&wo=1").HTML())
	assert.NotContains(t, body, "note on another file", "an annotation on an unchanged file is not scoped to this packet")
}
