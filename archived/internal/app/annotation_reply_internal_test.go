package app

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-via/via/vt"

	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/ledger"
)

// findAnnotation locates the annotation a reply targets — the parent must
// really exist before a reply can anchor to it, so a phantom id resolves to
// not-found rather than a fabricated anchor.
func TestFindAnnotation_locatesTheTargetOrReportsNotFound(t *testing.T) {
	anns := []ledger.AnnotationRecord{
		{ID: "ann-1", File: "a.go", StartLine: 3},
		{ID: "ann-2", File: "b.go", StartLine: 9},
	}
	got, ok := findAnnotation(anns, "ann-2")
	require.True(t, ok)
	assert.Equal(t, "b.go", got.File, "the found record carries the parent's anchor")

	_, ok = findAnnotation(anns, "ghost")
	assert.False(t, ok, "a phantom id is not found — never a fabricated parent")
}

// Replying to an annotation persists a durable reply under it — ParentID set,
// inheriting the parent's file/line anchor — and re-triggers the harness with
// the reply as the turn. This is the write half of "annotations with replies".
func TestReviewCard_replyToAnnotationPersistsAReplyUnderItsParent(t *testing.T) {
	resetConsumersForTest()
	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "reply", "i")
	// seed a parent annotation to reply to
	require.NoError(t, log.AppendAnnotation(ledger.AnnotationRecord{ID: "ann-1", File: "main.go", StartLine: 7, Author: "lead", Body: "guard the negative case"}))
	registerSession("reply", LiveConfig{RepoDir: ".", BaseRev: "own-b", FixRev: "own-f", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	viaApp, defLog, err := NewServer(LiveConfig{RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(), TestCmd: []string{"true"}, LedgerPath: defLogPath})
	require.NoError(t, err)
	server := httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	require.Equal(t, 200, vt.NewClient(t, server, "/review?key=reply").
		Action((&ReviewCard{Key: "reply"}).ReplyToAnnotation).
		WithSignal("replyparent", "ann-1").WithSignal("replytext", "done, added the guard").Fire())

	anns, err := log.Annotations()
	require.NoError(t, err)
	require.Len(t, anns, 2, "the reply persisted alongside its parent")
	reply := anns[1]
	assert.Equal(t, "ann-1", reply.ParentID, "the reply points at the annotation it answers")
	assert.Equal(t, "main.go", reply.File, "the reply inherits the parent's file anchor")
	assert.Equal(t, 7, reply.StartLine, "and the parent's line")
	assert.Equal(t, "done, added the guard", reply.Body)
	assert.Equal(t, "lead", reply.Author)
}

// A reply to an annotation that doesn't exist is a silent no-op — never a reply
// dangling off a phantom parent.
func TestReviewCard_replyToAMissingAnnotationIsANoOp(t *testing.T) {
	resetConsumersForTest()
	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "replyghost", "i")
	registerSession("replyghost", LiveConfig{RepoDir: ".", BaseRev: "own-b", FixRev: "own-f", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	viaApp, defLog, err := NewServer(LiveConfig{RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(), TestCmd: []string{"true"}, LedgerPath: defLogPath})
	require.NoError(t, err)
	server := httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	require.Equal(t, 200, vt.NewClient(t, server, "/review?key=replyghost").
		Action((&ReviewCard{Key: "replyghost"}).ReplyToAnnotation).
		WithSignal("replyparent", "ghost").WithSignal("replytext", "reply to nothing").Fire())

	anns, err := log.Annotations()
	require.NoError(t, err)
	assert.Empty(t, anns, "a reply to a phantom parent records nothing")
}

// An empty reply is nothing to record — a silent no-op even against a real
// parent.
func TestReviewCard_emptyReplyIsANoOp(t *testing.T) {
	resetConsumersForTest()
	ctx := context.Background()
	f, err := fabric.Start(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "replyempty", "i")
	require.NoError(t, log.AppendAnnotation(ledger.AnnotationRecord{ID: "ann-1", File: "main.go", StartLine: 7, Author: "lead", Body: "q"}))
	registerSession("replyempty", LiveConfig{RepoDir: ".", BaseRev: "own-b", FixRev: "own-f", Anchor: anchorForCap(), TestCmd: []string{"true"}}, log)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	viaApp, defLog, err := NewServer(LiveConfig{RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(), TestCmd: []string{"true"}, LedgerPath: defLogPath})
	require.NoError(t, err)
	server := httptest.NewServer(viaApp)
	t.Cleanup(func() { _ = defLog.Close() })

	require.Equal(t, 200, vt.NewClient(t, server, "/review?key=replyempty").
		Action((&ReviewCard{Key: "replyempty"}).ReplyToAnnotation).
		WithSignal("replyparent", "ann-1").WithSignal("replytext", "   ").Fire())

	anns, err := log.Annotations()
	require.NoError(t, err)
	require.Len(t, anns, 1, "only the parent — the empty reply recorded nothing")
}
