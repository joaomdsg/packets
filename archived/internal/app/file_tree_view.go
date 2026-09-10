package app

import (
	"context"
	"net/url"
	"strconv"

	"github.com/go-via/via/h"

	"github.com/joaomdsg/packets/internal/diff"
	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/review"
)

// diffCompute is the base→fix differ the tree overlay reads; a package var so
// tests inject a canned diff (the real one shells out to git).
var diffCompute = diff.Compute

// renderFileTree renders the packet's full fix-tree as a nested, collapsible file
// tree with the base→fix changes highlighted in place — the review surface's
// left rail. Each leaf is a plain href click-through (/review?wo=<id>&file=<path>),
// NOT a datastar @post: selecting a file is pure navigation, and the diff island
// it drives (Slice 3) is data-ignore-morph + mount-guarded, so an SSE morph could
// never swap its content — only a fresh navigation re-mounts it (council R-converged).
func renderFileTree(cfg LiveConfig, tgt ledger.Target, woID int, selected string, annCounts map[string]int) h.H {
	ctx := context.Background()
	changed, _ := diffCompute(ctx, cfg.RepoDir, tgt.BaseRev, tgt.FixRev)
	allPaths, err := fileListAt(ctx, cfg.RepoDir, tgt.FixRev)
	if err != nil {
		// Without the fix tree, show the changed files AS changed — never mislabel a
		// real edit as a deletion just because the working-tree listing failed.
		allPaths = changedPaths(changed)
	}
	root := buildFileTree(allPaths, changed)
	files, added, deleted := changedSummary(changed)
	kids := []h.H{h.Class("file-tree"), h.Attr("aria-label", "changed files"),
		h.Div(h.Class("file-tree__summary"), h.Text(formatChangedSummary(files, added, deleted)))}
	kids = append(kids, renderTreeChildren(root.children, woID, selected, annCounts)...)
	return h.Div(kids...)
}

// annotationCountsByFile tallies open threads by their file — the count the tree
// badges each changed file with, so attention points only where a question is
// actually waiting. It counts exactly the threads it is given (the caller passes
// the open set); a file with none simply never appears in the map.
func annotationCountsByFile(threads []review.Thread) map[string]int {
	counts := make(map[string]int, len(threads))
	for _, t := range threads {
		counts[t.File]++
	}
	return counts
}

// changedSummary totals the base→fix diff: the number of changed files and the
// summed added / deleted line deltas across them. Pure over the diff the tree
// already computed, so the header can never disagree with the leaves below it.
func changedSummary(d diff.Diff) (files, added, deleted int) {
	for _, f := range d.Files {
		added += f.Added
		deleted += f.Deleted
	}
	return len(d.Files), added, deleted
}

// formatChangedSummary renders the diff stat as "N files · +A −B" (U+2212 minus,
// matching countLabel so the header and the per-leaf counts share one glyph).
// The "no changes" case keys on the FILE count, not the line deltas — a
// pure-rename touches real files with zero +/− lines and must still be reported,
// never hidden behind "no changes".
func formatChangedSummary(files, added, deleted int) string {
	if files == 0 {
		return "no changes"
	}
	noun := "files"
	if files == 1 {
		noun = "file"
	}
	return strconv.Itoa(files) + " " + noun + " · +" + strconv.Itoa(added) + " −" + strconv.Itoa(deleted)
}

func changedPaths(d diff.Diff) []string {
	out := make([]string, 0, len(d.Files))
	for _, f := range d.Files {
		out = append(out, f.Path)
	}
	return out
}

// renderTreeChildren renders a node's children: directories as expanded-by-default
// <details> groups, files as href leaves.
func renderTreeChildren(nodes []*treeNode, woID int, selected string, annCounts map[string]int) []h.H {
	out := make([]h.H, 0, len(nodes))
	for _, n := range nodes {
		if n.isDir {
			out = append(out, h.Details(
				h.Attr("open"),
				h.Summary(h.Class("file-tree__dir"), h.Text(n.name)),
				h.Div(append([]h.H{h.Class("file-tree__children")},
					renderTreeChildren(n.children, woID, selected, annCounts)...)...),
			))
			continue
		}
		out = append(out, renderTreeLeaf(n, woID, selected, annCounts[n.path]))
	}
	return out
}

func renderTreeLeaf(n *treeNode, woID int, selected string, annCount int) h.H {
	class := "file-tree__file"
	switch n.status {
	case statusChanged:
		class += " file-tree__file--changed"
	case statusDeleted:
		class += " file-tree__file--deleted"
	}
	attrs := []h.H{
		h.Class(class),
		h.Href("/review?wo=" + strconv.Itoa(woID) + "&file=" + url.QueryEscape(n.path)),
		h.Span(h.Class("file-tree__name"), h.Text(n.name)),
	}
	if n.path == selected {
		attrs[0] = h.Class(class + " file-tree__file--selected")
		attrs = append(attrs, h.Attr("aria-current", "true"))
	}
	if n.status != statusUnchanged {
		attrs = append(attrs, h.Span(h.Class("file-tree__counts"), h.Text(countLabel(n))))
	}
	// A badge only where a real question is waiting — count>0, never a zero, so a
	// clean file looks clean rather than decorated.
	if annCount > 0 {
		attrs = append(attrs, h.Span(h.Class("file-tree__badge"), h.Text(strconv.Itoa(annCount))))
	}
	return h.A(attrs...)
}

// countLabel formats a changed leaf's line delta as "+A −B" (U+2212 minus, the
// design language's glyph) — a quiet diff stat, never a gauge.
func countLabel(n *treeNode) string {
	return "+" + strconv.Itoa(n.added) + " −" + strconv.Itoa(n.deleted)
}
