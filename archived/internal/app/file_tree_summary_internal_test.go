package app

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/joaomdsg/packets/internal/diff"
	"github.com/joaomdsg/packets/internal/ledger"
)

// The header must total the REAL diff — the file count plus the summed added and
// deleted line deltas across every changed file — so the Lead reads the actual
// edit size, never a fabricated one.
func TestChangedSummary_totalsFilesAndLineDeltasAcrossTheDiff(t *testing.T) {
	d := diff.Diff{Files: []diff.FileDiff{
		{Path: "a.go", Added: 5, Deleted: 2},
		{Path: "b.go", Added: 3, Deleted: 0},
		{Path: "c.go", Added: 0, Deleted: 4},
	}}
	files, added, deleted := changedSummary(d)
	assert.Equal(t, 3, files)
	assert.Equal(t, 8, added, "added lines sum across all changed files")
	assert.Equal(t, 6, deleted, "deleted lines sum across all changed files")
}

// An empty diff totals to honest zeros — a summary of nothing must reflect
// nothing, never invent a count.
func TestChangedSummary_isAllZeroForAnEmptyDiff(t *testing.T) {
	files, added, deleted := changedSummary(diff.Diff{})
	assert.Equal(t, 0, files)
	assert.Equal(t, 0, added)
	assert.Equal(t, 0, deleted)
}

// The line reads as the design's diff-stat: "N files · +A −B" with the U+2212
// minus glyph, matching the per-leaf count convention so the two never diverge.
func TestFormatChangedSummary_readsAsTheDiffStatLine(t *testing.T) {
	assert.Equal(t, "3 files · +8 −6", formatChangedSummary(3, 8, 6))
}

// A single changed file is "1 file" (singular), not "1 files" — the copy stays
// grammatical so it reads like a person wrote it.
func TestFormatChangedSummary_singularForASingleFile(t *testing.T) {
	assert.Equal(t, "1 file · +5 −0", formatChangedSummary(1, 5, 0))
}

// No changes is stated plainly, not as a bare "0 files · +0 −0" that reads like a
// broken widget.
func TestFormatChangedSummary_saysNoChangesWhenEmpty(t *testing.T) {
	assert.Equal(t, "no changes", formatChangedSummary(0, 0, 0))
}

// "No changes" keys on the FILE count, not the line deltas: a pure-rename or
// mode-change touches real files with zero +/− lines and must still report those
// files, never collapse to "no changes" and hide a real edit.
func TestFormatChangedSummary_filesWithZeroLineDeltasStillReportTheFiles(t *testing.T) {
	assert.Equal(t, "2 files · +0 −0", formatChangedSummary(2, 0, 0))
}

// The rendered tree carries a summary header totalling the real diff, so the edit
// size sits above the file list at a glance.
func TestRenderFileTree_headerSummarizesTheChangedFilesAndLineCounts(t *testing.T) {
	swapTreeSeams(t,
		[]string{"a.go", "b.go"},
		nil,
		diff.Diff{Files: []diff.FileDiff{
			{Path: "a.go", Added: 5, Deleted: 2},
			{Path: "b.go", Added: 3, Deleted: 1},
		}},
	)
	tgt := ledger.Target{BaseRev: "b", FixRev: "f", Path: "a.go"}
	body := renderHTML(t, renderFileTree(LiveConfig{RepoDir: "."}, tgt, 7, "", nil))
	assert.Contains(t, body, "file-tree__summary", "the summary header renders")
	assert.Contains(t, body, "2 files · +8 −3", "the header totals the real diff, reusing the tree's own diff")
}

// A DIFFERENT diff must yield a DIFFERENT header — proving the summary is derived
// from the real diff each render, never a hardcoded string that happens to match
// one fixture.
func TestRenderFileTree_headerReflectsWhicheverDiffIsRendered(t *testing.T) {
	swapTreeSeams(t,
		[]string{"only.go"},
		nil,
		diff.Diff{Files: []diff.FileDiff{{Path: "only.go", Added: 9, Deleted: 0}}},
	)
	tgt := ledger.Target{BaseRev: "b", FixRev: "f", Path: "only.go"}
	body := renderHTML(t, renderFileTree(LiveConfig{RepoDir: "."}, tgt, 3, "", nil))
	assert.Contains(t, body, "1 file · +9 −0", "the header reflects this diff's real totals")
	assert.NotContains(t, body, "2 files", "no other fixture's total leaks in")
}

// An order that changed nothing renders the honest "no changes" header end to
// end — the summary path degrades truthfully through the real render, never to a
// fabricated count.
func TestRenderFileTree_headerSaysNoChangesForAnEmptyDiff(t *testing.T) {
	swapTreeSeams(t, nil, nil, diff.Diff{})
	tgt := ledger.Target{BaseRev: "b", FixRev: "f"}
	body := renderHTML(t, renderFileTree(LiveConfig{RepoDir: "."}, tgt, 1, "", nil))
	assert.Contains(t, body, "file-tree__summary", "the summary header renders even with nothing changed")
	assert.Contains(t, body, "no changes")
}
