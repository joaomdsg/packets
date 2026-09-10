package app

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/joaomdsg/packets/internal/diff"
	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/review"
)

// The tree badge must count REAL open threads per file, so the Lead can see at a
// glance which changed files still carry unanswered questions — several threads
// on one file tally together, a file with none has no count.
func TestAnnotationCountsByFile_talliesOpenThreadsPerFile(t *testing.T) {
	threads := []review.Thread{
		{File: "internal/app/live.go", StartLine: 3},
		{File: "internal/app/live.go", StartLine: 9},
		{File: "README.md", StartLine: 1},
	}
	got := annotationCountsByFile(threads)
	assert.Equal(t, 2, got["internal/app/live.go"], "two threads on one file tally together")
	assert.Equal(t, 1, got["README.md"])
	assert.Equal(t, 0, got["untouched.go"], "a file with no threads carries no count")
}

// No threads yields an empty tally — never a fabricated badge on a clean file.
func TestAnnotationCountsByFile_isEmptyWhenNothingIsOpen(t *testing.T) {
	assert.Len(t, annotationCountsByFile(nil), 0)
}

// A changed file that carries open annotations shows a badge with the real
// count; a file with none stays unbadged — the badge points attention only where
// a question is actually waiting. The badged file is NOT the selected one, so a
// shortcut that badges the open file (rather than the map) is caught; the count
// is anchored to the badge span, so a hardcoded number can't pass either.
func TestRenderFileTree_flagsOnlyFilesCarryingOpenAnnotations(t *testing.T) {
	swapTreeSeams(t,
		[]string{"a.go", "b.go"},
		nil,
		diff.Diff{Files: []diff.FileDiff{
			{Path: "a.go", Added: 1, Deleted: 0},
			{Path: "b.go", Added: 1, Deleted: 0},
		}},
	)
	tgt := ledger.Target{BaseRev: "b", FixRev: "f", Path: "a.go"} // a.go is the OPEN file
	counts := map[string]int{"b.go": 4}                           // but b.go is the one with annotations
	body := renderHTML(t, renderFileTree(LiveConfig{RepoDir: "."}, tgt, 7, "a.go", counts))

	assert.Equal(t, 1, strings.Count(body, "file-tree__badge"),
		"only the file in the count map is badged — not the selected file, not the clean sibling")
	assert.Contains(t, body, `file-tree__badge">4`,
		"the badge value is the map's real count, tied to the badge element")
}

// A file present in the map with a ZERO count is NOT badged — an explicit zero is
// still nothing to flag, the same as being absent. Guards the count>0 rule.
func TestRenderFileTree_doesNotBadgeAnExplicitZeroCount(t *testing.T) {
	swapTreeSeams(t,
		[]string{"a.go"},
		nil,
		diff.Diff{Files: []diff.FileDiff{{Path: "a.go", Added: 1, Deleted: 0}}},
	)
	tgt := ledger.Target{BaseRev: "b", FixRev: "f", Path: "a.go"}
	body := renderHTML(t, renderFileTree(LiveConfig{RepoDir: "."}, tgt, 7, "", map[string]int{"a.go": 0}))
	assert.NotContains(t, body, "file-tree__badge", "an explicit zero count is nothing to flag")
}

// With no open annotations, the tree carries no badges at all — a clean packet
// looks clean, never decorated with a zero.
func TestRenderFileTree_carriesNoBadgesWhenNothingIsOpen(t *testing.T) {
	swapTreeSeams(t,
		[]string{"a.go"},
		nil,
		diff.Diff{Files: []diff.FileDiff{{Path: "a.go", Added: 1, Deleted: 0}}},
	)
	tgt := ledger.Target{BaseRev: "b", FixRev: "f", Path: "a.go"}
	body := renderHTML(t, renderFileTree(LiveConfig{RepoDir: "."}, tgt, 7, "", nil))
	assert.NotContains(t, body, "file-tree__badge", "no open annotations → no badges")
}
