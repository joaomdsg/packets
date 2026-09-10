package app

import (
	"context"
	"os/exec"
	"sort"
	"strings"

	"github.com/joaomdsg/packets/internal/diff"
)

// treeStatus is a file leaf's change state relative to the reviewed base→fix
// diff: unchanged (in the fix tree, untouched), changed (in the diff), or
// deleted (in the diff but gone from the fix tree).
type treeStatus int

const (
	statusUnchanged treeStatus = iota
	statusChanged
	statusDeleted
)

// treeNode is one node of the changed-file tree the review surface renders: a
// directory grouping children, or a file leaf carrying its change status and
// the diff's added/deleted line counts. children are sorted directories-first
// then files, each alphabetical, so the render and its tests are deterministic.
type treeNode struct {
	name     string
	path     string
	isDir    bool
	children []*treeNode
	status   treeStatus
	added    int
	deleted  int
}

// fileListAt lists every file path in the repo at a revision, the I/O seam the
// tree builder reads the full fix tree from. A package var so the render layer
// can swap a canned lister in tests (mirrors reviewFileReader). quotepath=false
// + -z give raw NUL-terminated paths, immune to git's path quoting (see
// internal/diff on why header paths can't be trusted).
var fileListAt = func(ctx context.Context, repoDir, rev string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "-c", "core.quotepath=false",
		"ls-tree", "-r", "--name-only", "-z", rev)
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, p := range strings.Split(string(out), "\x00") {
		if p != "" {
			paths = append(paths, p)
		}
	}
	return paths, nil
}

// buildFileTree assembles the nested tree from the full fix-tree file list with
// the base→fix diff overlaid. A changed file present in allPaths is highlighted
// in place; a changed path absent from allPaths is a deletion, re-injected as a
// leaf so it stays reachable rather than vanishing from the surface.
//
// Invariant (guaranteed by the fileListAt/diff.Compute seams): every path is a
// valid, non-empty git path, with no trailing slash, and no path is both a file
// and a directory prefix — so a segment name is unambiguously dir-or-file.
func buildFileTree(allPaths []string, changed diff.Diff) *treeNode {
	byPath := make(map[string]diff.FileDiff, len(changed.Files))
	for _, f := range changed.Files {
		byPath[f.Path] = f
	}
	inTree := make(map[string]bool, len(allPaths))
	for _, p := range allPaths {
		inTree[p] = true
	}
	root := &treeNode{isDir: true}
	for _, p := range allPaths {
		leaf := insert(root, p)
		if f, ok := byPath[p]; ok {
			leaf.status = statusChanged
			leaf.added, leaf.deleted = f.Added, f.Deleted
		}
	}
	for _, f := range changed.Files {
		if inTree[f.Path] {
			continue // a changed file in the tree is already highlighted in place
		}
		leaf := insert(root, f.Path)
		leaf.status = statusDeleted
		leaf.added, leaf.deleted = f.Added, f.Deleted
	}
	sortTree(root)
	return root
}

// insert walks path's segments from root, creating directory nodes as needed,
// and returns the (existing or new) file leaf at its tail. Idempotent: a path
// inserted twice yields the same leaf.
func insert(root *treeNode, path string) *treeNode {
	cur := root
	segs := strings.Split(path, "/")
	for i, seg := range segs {
		isLeaf := i == len(segs)-1
		child := childByName(cur, seg)
		if child == nil {
			child = &treeNode{name: seg, path: strings.Join(segs[:i+1], "/"), isDir: !isLeaf}
			cur.children = append(cur.children, child)
		}
		cur = child
	}
	return cur
}

func childByName(n *treeNode, name string) *treeNode {
	for _, c := range n.children {
		if c.name == name {
			return c
		}
	}
	return nil
}

// sortTree orders every node's children directories-first then files, each
// group alphabetical, recursively.
func sortTree(n *treeNode) {
	sort.SliceStable(n.children, func(i, j int) bool {
		a, b := n.children[i], n.children[j]
		if a.isDir != b.isDir {
			return a.isDir // directories before files
		}
		return a.name < b.name
	})
	for _, c := range n.children {
		sortTree(c)
	}
}
