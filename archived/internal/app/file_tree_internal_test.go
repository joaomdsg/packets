package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/diff"
)

// childNamed returns the direct child of n with the given segment name, or nil.
func childNamed(n *treeNode, name string) *treeNode {
	for _, c := range n.children {
		if c.name == name {
			return c
		}
	}
	return nil
}

// childNames lists a node's children's segment names in order — the assertion
// surface for deterministic ordering.
func childNames(n *treeNode) []string {
	out := make([]string, 0, len(n.children))
	for _, c := range n.children {
		out = append(out, c.name)
	}
	return out
}

func TestBuildFileTree_emptyInputsYieldAnEmptyRoot(t *testing.T) {
	t.Parallel()
	root := buildFileTree(nil, diff.Diff{})
	require.NotNil(t, root)
	assert.True(t, root.isDir, "root is a directory")
	assert.Empty(t, root.children, "no files means no children")
}

func TestBuildFileTree_nestsPathsIntoFolders(t *testing.T) {
	t.Parallel()
	root := buildFileTree([]string{"internal/app/live.go", "internal/diff/diff.go"}, diff.Diff{})

	internal := childNamed(root, "internal")
	require.NotNil(t, internal, "internal folder present")
	assert.True(t, internal.isDir)
	assert.Equal(t, "internal", internal.path)

	app := childNamed(internal, "app")
	require.NotNil(t, app, "internal/app folder present")
	assert.True(t, app.isDir)

	live := childNamed(app, "live.go")
	require.NotNil(t, live, "the file leaf is reachable at its nested path")
	assert.False(t, live.isDir)
	assert.Equal(t, "internal/app/live.go", live.path)

	diffDir := childNamed(internal, "diff")
	require.NotNil(t, diffDir)
	require.NotNil(t, childNamed(diffDir, "diff.go"))
}

func TestBuildFileTree_rootLevelFileIsADirectChild(t *testing.T) {
	t.Parallel()
	root := buildFileTree([]string{"main.go"}, diff.Diff{})
	leaf := childNamed(root, "main.go")
	require.NotNil(t, leaf)
	assert.False(t, leaf.isDir)
	assert.Equal(t, "main.go", leaf.path)
}

func TestBuildFileTree_unchangedFileCarriesNoChange(t *testing.T) {
	t.Parallel()
	root := buildFileTree([]string{"a.go"}, diff.Diff{})
	leaf := childNamed(root, "a.go")
	require.NotNil(t, leaf)
	assert.Equal(t, statusUnchanged, leaf.status)
	assert.Zero(t, leaf.added)
	assert.Zero(t, leaf.deleted)
}

func TestBuildFileTree_changedFileIsHighlightedWithCounts(t *testing.T) {
	t.Parallel()
	changed := diff.Diff{Files: []diff.FileDiff{
		{Path: "internal/app/live.go", Added: 5, Deleted: 2},
	}}
	root := buildFileTree([]string{"internal/app/live.go", "internal/app/board.go"}, changed)

	app := childNamed(childNamed(root, "internal"), "app")
	require.NotNil(t, app)

	live := childNamed(app, "live.go")
	require.NotNil(t, live)
	assert.Equal(t, statusChanged, live.status, "a file in the diff is highlighted as changed")
	assert.Equal(t, 5, live.added)
	assert.Equal(t, 2, live.deleted)

	board := childNamed(app, "board.go")
	require.NotNil(t, board)
	assert.Equal(t, statusUnchanged, board.status, "a sibling not in the diff stays unchanged")
}

func TestBuildFileTree_deletedFileAbsentFromTreeIsStillReachable(t *testing.T) {
	t.Parallel()
	// gone.go is in the diff (the agent deleted it) but NOT in the fix tree, so it
	// would vanish from the surface entirely unless re-injected as a deleted leaf.
	changed := diff.Diff{Files: []diff.FileDiff{
		{Path: "internal/old/gone.go", Added: 0, Deleted: 9},
	}}
	root := buildFileTree([]string{"internal/app/live.go"}, changed)

	old := childNamed(childNamed(root, "internal"), "old")
	require.NotNil(t, old, "the deleted file's directory is materialized")
	gone := childNamed(old, "gone.go")
	require.NotNil(t, gone, "the deleted file is reachable")
	assert.Equal(t, statusDeleted, gone.status)
	assert.Equal(t, 9, gone.deleted)
}

func TestBuildFileTree_deletedLeafMergesIntoAnExistingDirectory(t *testing.T) {
	t.Parallel()
	// A surviving sibling and a deleted file share a directory: the directory must
	// be materialized ONCE holding both, never duplicated into two "old" nodes.
	changed := diff.Diff{Files: []diff.FileDiff{
		{Path: "internal/old/gone.go", Added: 0, Deleted: 9},
	}}
	root := buildFileTree([]string{"internal/old/kept.go"}, changed)

	internal := childNamed(root, "internal")
	require.NotNil(t, internal)
	assert.Equal(t, []string{"old"}, childNames(internal), "exactly one 'old' directory")

	old := childNamed(internal, "old")
	require.NotNil(t, old)
	assert.Equal(t, []string{"gone.go", "kept.go"}, childNames(old), "both files under the merged dir")
	assert.Equal(t, statusDeleted, childNamed(old, "gone.go").status)
	assert.Equal(t, statusUnchanged, childNamed(old, "kept.go").status)
}

func TestBuildFileTree_fileInTreeIsChangedNotDeletedEvenWhenOnlyDeletions(t *testing.T) {
	t.Parallel()
	// A modified file whose hunk only removed lines (Added 0, Deleted 7) is STILL
	// present in the fix tree — it must read as changed, never mistaken for a
	// deletion (deletion = absent from the tree, not "zero additions").
	changed := diff.Diff{Files: []diff.FileDiff{
		{Path: "a.go", Added: 0, Deleted: 7},
	}}
	root := buildFileTree([]string{"a.go"}, changed)
	leaf := childNamed(root, "a.go")
	require.NotNil(t, leaf)
	assert.Equal(t, statusChanged, leaf.status)
	assert.Equal(t, 7, leaf.deleted)
}

func TestBuildFileTree_repeatedPathYieldsASingleLeaf(t *testing.T) {
	t.Parallel()
	// The builder must be idempotent on a repeated path (a defensive contract:
	// the surface should never show a file twice), collapsing it to one leaf.
	root := buildFileTree([]string{"a/b.go", "a/b.go"}, diff.Diff{})
	a := childNamed(root, "a")
	require.NotNil(t, a)
	assert.Equal(t, []string{"b.go"}, childNames(a), "the duplicate collapses to one leaf")
}

func TestBuildFileTree_PacketsDirectoriesBeforeFilesEachAlphabetical(t *testing.T) {
	t.Parallel()
	root := buildFileTree([]string{"z.go", "m.go", "a/b.go", "a/a.go"}, diff.Diff{})
	// Directories sort ahead of files; within each group, alphabetical — a stable
	// render order the tree-view and its tests can both rely on.
	assert.Equal(t, []string{"a", "m.go", "z.go"}, childNames(root))

	a := childNamed(root, "a")
	require.NotNil(t, a)
	assert.Equal(t, []string{"a.go", "b.go"}, childNames(a))
}
