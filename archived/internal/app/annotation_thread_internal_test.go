package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/ledger"
)

// Top-level annotations become threads in the order they were written, so the
// rail reads chronologically — the first thing said sits first.
func TestFoldAnnotationThreads_topLevelAnnotationsBecomeThreadsInAppendPacket(t *testing.T) {
	got := foldAnnotationThreads([]ledger.AnnotationRecord{
		{ID: "a", File: "x.go", StartLine: 1, Author: "lead", Body: "first"},
		{ID: "b", File: "y.go", StartLine: 2, Author: "lead", Body: "second"},
	})
	require.Len(t, got, 2)
	assert.Equal(t, "a", got[0].Root.ID)
	assert.Equal(t, "b", got[1].Root.ID)
	assert.Empty(t, got[0].Replies, "a fresh annotation has no replies")
}

// A reply nests under the annotation it answers rather than floating as its own
// thread — that IS the conversation the feature exists to show.
func TestFoldAnnotationThreads_replyNestsUnderTheAnnotationItAnswers(t *testing.T) {
	got := foldAnnotationThreads([]ledger.AnnotationRecord{
		{ID: "root", File: "x.go", StartLine: 3, Author: "lead", Body: "why not <=?"},
		{ID: "r1", ParentID: "root", File: "x.go", StartLine: 3, Author: "agent", Body: "fixed"},
	})
	require.Len(t, got, 1, "the reply does not spawn a second top-level thread")
	assert.Equal(t, "root", got[0].Root.ID)
	require.Len(t, got[0].Replies, 1)
	assert.Equal(t, "r1", got[0].Replies[0].ID)
	assert.Equal(t, "agent", got[0].Replies[0].Author)
}

// Several replies on one annotation keep their chronological order, so a
// back-and-forth reads top to bottom the way it happened.
func TestFoldAnnotationThreads_multipleRepliesKeepChronologicalPacket(t *testing.T) {
	got := foldAnnotationThreads([]ledger.AnnotationRecord{
		{ID: "root", Author: "lead", Body: "q"},
		{ID: "r1", ParentID: "root", Author: "agent", Body: "a1"},
		{ID: "r2", ParentID: "root", Author: "lead", Body: "a2"},
	})
	require.Len(t, got, 1)
	require.Len(t, got[0].Replies, 2)
	assert.Equal(t, "r1", got[0].Replies[0].ID, "the earlier reply comes first")
	assert.Equal(t, "r2", got[0].Replies[1].ID)
}

// A reply whose parent isn't present stands as its own thread rather than
// vanishing — a durable record is never silently dropped, even if its parent
// somehow isn't in this view.
func TestFoldAnnotationThreads_orphanReplyIsNotLost(t *testing.T) {
	got := foldAnnotationThreads([]ledger.AnnotationRecord{
		{ID: "r1", ParentID: "missing", File: "x.go", Author: "lead", Body: "reply to nothing"},
	})
	require.Len(t, got, 1, "an orphan reply still surfaces as its own thread")
	assert.Equal(t, "r1", got[0].Root.ID)
	assert.Empty(t, got[0].Replies)
}

// A reply to a reply flattens under the ROOT of the conversation — the whole
// back-and-forth reads as one thread, not a fragmenting tree, so the human
// follows a single chronological exchange.
func TestFoldAnnotationThreads_replyToAReplyFlattensUnderTheRoot(t *testing.T) {
	got := foldAnnotationThreads([]ledger.AnnotationRecord{
		{ID: "root", Author: "lead", Body: "q"},
		{ID: "r1", ParentID: "root", Author: "agent", Body: "a1"},
		{ID: "r2", ParentID: "r1", Author: "lead", Body: "follow-up on a1"},
	})
	require.Len(t, got, 1, "the deeper reply doesn't spawn a new thread")
	require.Len(t, got[0].Replies, 2, "both descendants sit under the one root")
	assert.Equal(t, "r1", got[0].Replies[0].ID)
	assert.Equal(t, "r2", got[0].Replies[1].ID, "the follow-up flattens under the root, in order")
}

// A root recognized regardless of input position: a reply appearing BEFORE its
// root in the slice still nests correctly — the fold is order-independent about
// which record it sees first.
func TestFoldAnnotationThreads_nestsCorrectlyEvenWhenAReplyPrecedesItsRoot(t *testing.T) {
	got := foldAnnotationThreads([]ledger.AnnotationRecord{
		{ID: "r1", ParentID: "root", Author: "agent", Body: "reply"},
		{ID: "root", Author: "lead", Body: "question"},
	})
	require.Len(t, got, 1, "the root is still recognized though its reply came first")
	assert.Equal(t, "root", got[0].Root.ID)
	require.Len(t, got[0].Replies, 1)
	assert.Equal(t, "r1", got[0].Replies[0].ID)
}

// Two independent annotations keep their own replies — a reply never leaks onto
// the wrong thread.
func TestFoldAnnotationThreads_repliesRouteToTheirOwnRootNotAnother(t *testing.T) {
	got := foldAnnotationThreads([]ledger.AnnotationRecord{
		{ID: "A", Author: "lead", Body: "qA"},
		{ID: "B", Author: "lead", Body: "qB"},
		{ID: "ra", ParentID: "A", Author: "agent", Body: "answer to A"},
		{ID: "rb", ParentID: "B", Author: "agent", Body: "answer to B"},
	})
	require.Len(t, got, 2)
	assert.Equal(t, "A", got[0].Root.ID)
	require.Len(t, got[0].Replies, 1)
	assert.Equal(t, "ra", got[0].Replies[0].ID, "A's reply stays on A")
	assert.Equal(t, "B", got[1].Root.ID)
	require.Len(t, got[1].Replies, 1)
	assert.Equal(t, "rb", got[1].Replies[0].ID, "B's reply stays on B")
}

// No annotations yields no threads — a clean packet shows an empty rail, never a
// fabricated card.
func TestFoldAnnotationThreads_isEmptyWhenNothingIsAnnotated(t *testing.T) {
	assert.Empty(t, foldAnnotationThreads(nil))
}
