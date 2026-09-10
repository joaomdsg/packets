package ledger_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/ledger"
)

// A human-authored annotation is DURABLE — it replays back off the log carrying
// every field the Inspector needs to anchor it and thread replies under it, so an
// annotation survives a restart rather than evaporating like the old in-memory
// adjustment anchors did.
func TestAnnotation_replaysBackWithItsAnchorAndAuthorSoItSurvivesARestart(t *testing.T) {
	t.Parallel()
	l, _ := openLog(t)

	require.NoError(t, l.AppendAnnotation(ledger.AnnotationRecord{
		ID: "a1", File: "internal/pay/charge.go", StartLine: 42, EndLine: 45,
		Author: "lead", Body: "this branch never rejects a negative amount", AtUnixMs: 1_700_000_000_000,
	}))

	anns, err := l.Annotations()
	require.NoError(t, err)
	require.Len(t, anns, 1)
	got := anns[0]
	assert.Equal(t, "a1", got.ID, "the id lets replies target this exact annotation")
	assert.Equal(t, "internal/pay/charge.go", got.File)
	assert.Equal(t, 42, got.StartLine)
	assert.Equal(t, 45, got.EndLine, "a line RANGE round-trips, not just a single line")
	assert.Equal(t, "lead", got.Author)
	assert.Equal(t, "this branch never rejects a negative amount", got.Body)
	assert.Empty(t, got.ParentID, "a top-level annotation has no parent")
	assert.Equal(t, int64(1_700_000_000_000), got.AtUnixMs,
		"the timestamp round-trips so the read model can order a thread chronologically")
}

// A reply carries its parent's id so the read model can nest it under the
// annotation it answers — the durable substrate for a threaded conversation.
func TestAnnotation_aReplyCarriesItsParentSoTheThreadCanNest(t *testing.T) {
	t.Parallel()
	l, _ := openLog(t)

	require.NoError(t, l.AppendAnnotation(ledger.AnnotationRecord{ID: "root", File: "a.go", StartLine: 3, Author: "lead", Body: "why not <=?"}))
	require.NoError(t, l.AppendAnnotation(ledger.AnnotationRecord{ID: "r1", ParentID: "root", File: "a.go", StartLine: 3, Author: "agent", Body: "fixed to <="}))

	anns, err := l.Annotations()
	require.NoError(t, err)
	require.Len(t, anns, 2, "both the annotation and its reply are durable")
	assert.Empty(t, anns[0].ParentID, "the root is top-level")
	assert.Equal(t, "root", anns[1].ParentID, "the reply points at the annotation it answers")
	assert.Equal(t, "agent", anns[1].Author, "a reply records who authored it")
}

// A whole-file annotation carries no line anchor (StartLine 0) and still replays
// — the tree/rail can attach it to the file rather than a line.
func TestAnnotation_aFileLevelAnnotationHasNoLineAnchor(t *testing.T) {
	t.Parallel()
	l, _ := openLog(t)

	require.NoError(t, l.AppendAnnotation(ledger.AnnotationRecord{ID: "f1", File: "go.mod", Author: "lead", Body: "don't touch this"}))

	anns, err := l.Annotations()
	require.NoError(t, err)
	require.Len(t, anns, 1)
	assert.Equal(t, "go.mod", anns[0].File)
	assert.Equal(t, 0, anns[0].StartLine, "a file-level annotation has no line anchor")
}

// An annotation shares the append-only log with the economic events. The fold
// must LEARN the new kind and keep folding a mixed stream — its closed-kind-set
// default errors on anything unknown, so a real annotation beside a catch proves
// the new kind slots in cleanly rather than crashing every replayer.
func TestAnnotation_coexistsWithOtherEventKindsWithoutBreakingTheFold(t *testing.T) {
	t.Parallel()
	l, _ := openLog(t)

	require.NoError(t, l.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 1, ReasonTag: "catch"}))
	require.NoError(t, l.AppendAnnotation(ledger.AnnotationRecord{ID: "a1", File: "c.go", StartLine: 1, Author: "lead", Body: "note"}))

	anns, err := l.Annotations()
	require.NoError(t, err, "the fold must not error on a stream mixing a catch and an annotation")
	require.Len(t, anns, 1, "only the annotation is returned here; the catch folds on its own projection")
}
