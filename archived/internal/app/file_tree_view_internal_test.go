package app

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/joaomdsg/packets/internal/diff"
	"github.com/joaomdsg/packets/internal/ledger"
)

// swapTreeSeams points the lister + differ at canned data for one test and
// restores them after. Tests that swap package vars must not run in parallel.
func swapTreeSeams(t *testing.T, paths []string, listErr error, changed diff.Diff) {
	t.Helper()
	origList, origDiff := fileListAt, diffCompute
	fileListAt = func(context.Context, string, string) ([]string, error) { return paths, listErr }
	diffCompute = func(context.Context, string, string, string) (diff.Diff, error) { return changed, nil }
	t.Cleanup(func() { fileListAt, diffCompute = origList, origDiff })
}

// The tree must show the WHOLE fix tree (so the Lead has full context) while
// highlighting only the files the order actually changed — and each leaf must be
// a click-through keyed on its path + the work-order id (the Slice-3 selector).
func TestRenderFileTree_highlightsOnlyChangedFilesAndLinksEachLeaf(t *testing.T) {
	swapTreeSeams(t,
		[]string{"internal/app/live.go", "internal/app/board.go", "README.md"},
		nil,
		diff.Diff{Files: []diff.FileDiff{{Path: "internal/app/live.go", Added: 5, Deleted: 2}}},
	)
	tgt := ledger.Target{BaseRev: "b", FixRev: "f", Path: "internal/app/live.go"}
	body := renderHTML(t, renderFileTree(LiveConfig{RepoDir: "."}, tgt, 7, "", nil))

	assert.Contains(t, body, "file-tree", "the tree container renders")
	assert.Contains(t, body, "live.go")
	assert.Contains(t, body, "board.go")
	assert.Contains(t, body, "README.md", "the full fix tree shows, not only changed files")

	// Nesting: directories are native collapsible <details>/<summary> groups,
	// expanded by default so the changed files are visible without a click.
	assert.Contains(t, body, "<details open")
	assert.Contains(t, body, "<summary")
	assert.Contains(t, body, "internal")
	assert.Contains(t, body, "app")
	// internal/app/* nests two deep — pin the real recursion, not a flat render.
	assert.Equal(t, 2, strings.Count(body, "<details"), "internal then app, nested")

	// Exactly one leaf is highlighted as changed — the sibling stays plain.
	assert.Equal(t, 1, strings.Count(body, "file-tree__file--changed"),
		"only the changed file carries the changed-highlight class")
	assert.Contains(t, body, "+5 −2", "the changed leaf shows both counts together")

	// Each leaf links to /review?wo=<id>&file=<path> (path query-escaped).
	assert.Contains(t, body, "/review?wo=7&amp;file=internal%2Fapp%2Fboard.go",
		"an unchanged leaf is a click-through keyed on its path and the order id")
}

// A file that the order changed AND is the one currently open must carry BOTH
// markers and still link correctly — the common case of inspecting a real edit.
func TestRenderFileTree_changedAndSelectedLeafCarriesBothAndLinks(t *testing.T) {
	swapTreeSeams(t,
		[]string{"pkg/x.go"},
		nil,
		diff.Diff{Files: []diff.FileDiff{{Path: "pkg/x.go", Added: 4, Deleted: 1}}},
	)
	tgt := ledger.Target{BaseRev: "b", FixRev: "f", Path: "pkg/x.go"}
	body := renderHTML(t, renderFileTree(LiveConfig{RepoDir: "."}, tgt, 9, "pkg/x.go", nil))

	assert.Equal(t, 1, strings.Count(body, "file-tree__file--changed"))
	assert.Equal(t, 1, strings.Count(body, "file-tree__file--selected"))
	assert.Contains(t, body, "/review?wo=9&amp;file=pkg%2Fx.go",
		"the open, changed leaf still links with the right path and order id")
}

// A file the order DELETED is gone from the fix tree; it must still surface,
// flagged deleted, so the review of the change is complete.
func TestRenderFileTree_surfacesADeletedFileFlaggedDeleted(t *testing.T) {
	swapTreeSeams(t,
		[]string{"a.go"},
		nil,
		diff.Diff{Files: []diff.FileDiff{{Path: "internal/old/gone.go", Deleted: 9}}},
	)
	tgt := ledger.Target{BaseRev: "b", FixRev: "f", Path: "a.go"}
	body := renderHTML(t, renderFileTree(LiveConfig{RepoDir: "."}, tgt, 1, "", nil))

	assert.Contains(t, body, "gone.go")
	assert.Equal(t, 1, strings.Count(body, "file-tree__file--deleted"))
	assert.Contains(t, body, "+0 −9", "a deletion shows its removed-line count")
}

// The currently-open file is marked so the Lead sees where they are; the marker
// is an accessibility-grade aria-current, not just a color.
func TestRenderFileTree_marksTheSelectedFile(t *testing.T) {
	swapTreeSeams(t,
		[]string{"internal/app/live.go", "internal/app/board.go"},
		nil,
		diff.Diff{},
	)
	tgt := ledger.Target{BaseRev: "b", FixRev: "f", Path: "internal/app/live.go"}
	body := renderHTML(t, renderFileTree(LiveConfig{RepoDir: "."}, tgt, 1, "internal/app/board.go", nil))

	assert.Equal(t, 1, strings.Count(body, "file-tree__file--selected"),
		"exactly the open file is marked selected")
	assert.Contains(t, body, `aria-current`)
}

// When the fix-tree lister fails, the tree degrades to the changed files shown
// as CHANGED — it must never mislabel a real change as a deletion just because
// the working-tree listing was unavailable.
func TestRenderFileTree_degradesToChangedNotDeletedWhenListerFails(t *testing.T) {
	swapTreeSeams(t,
		nil,
		assertAnError(),
		diff.Diff{Files: []diff.FileDiff{{Path: "a.go", Added: 3}}},
	)
	tgt := ledger.Target{BaseRev: "b", FixRev: "f", Path: "a.go"}
	body := renderHTML(t, renderFileTree(LiveConfig{RepoDir: "."}, tgt, 1, "", nil))

	assert.Contains(t, body, "a.go")
	assert.Equal(t, 1, strings.Count(body, "file-tree__file--changed"))
	assert.NotContains(t, body, "file-tree__file--deleted",
		"a lister failure must not turn a change into a phantom deletion")
}

// The tree is a calm control-room surface: no alarm hues, no gauges/meters.
func TestRenderFileTree_carriesNoAlarmColorsOrGauges(t *testing.T) {
	swapTreeSeams(t,
		[]string{"a.go"},
		nil,
		diff.Diff{Files: []diff.FileDiff{{Path: "a.go", Added: 1, Deleted: 1}}},
	)
	tgt := ledger.Target{BaseRev: "b", FixRev: "f", Path: "a.go"}
	body := renderHTML(t, renderFileTree(LiveConfig{RepoDir: "."}, tgt, 1, "", nil))

	for _, banned := range []string{"#ff0000", "#00ff00", "progress-bar", "<progress", "<meter", "role=\"progressbar\""} {
		assert.NotContains(t, body, banned)
	}
}

func assertAnError() error { return errorString("ls-tree failed") }
