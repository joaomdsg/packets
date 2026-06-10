// Package diff computes a structured difference between two git revisions: the
// changed files, their added/deleted line counts, and each hunk's old/new line
// ranges. It is the substrate the review diff surface and the re-anchor
// algorithm read.
package diff

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Hunk is one unified-diff hunk's line ranges. A count of 0 (old side of an
// added file, or new side of a deletion) means the side is empty.
type Hunk struct {
	OldStart, OldLines int
	NewStart, NewLines int
}

// FileDiff is the change to one file: its path (the new path; for a deletion,
// the old path), the added/deleted content-line counts, and the hunks.
type FileDiff struct {
	Path    string
	Added   int
	Deleted int
	Hunks   []Hunk
}

// Diff is the structured difference between two revisions, files in the order
// git emits them.
type Diff struct {
	Files []FileDiff
}

// Compute returns the structured diff between fromRev and toRev in the git repo
// at repoDir. Diff output is pinned canonical (--no-color, --no-ext-diff,
// --no-renames, and explicit a/ b/ prefixes) so the repo/user git config cannot
// change the format the parser reads. Rename detection is intentionally OFF for
// now (--no-renames overrides any diff.renames config): a rename shows as a
// delete + an add (see RISKS.md "reanchor-rename-similarity-cliff", a later
// brick). Line CONTENT is not captured (ranges/counts only).
func Compute(ctx context.Context, repoDir, fromRev, toRev string) (Diff, error) {
	// -c core.quotepath=false: git's default octal-quotes non-ASCII paths in the
	// `diff --git`/`+++ b/` headers (café.txt → "caf\303\251.txt"), so the parsed
	// FileDiff.Path would be the mangled quoted form and never match a consumer's
	// real anchor path.
	cmd := exec.CommandContext(ctx, "git", "-c", "core.quotepath=false", "diff",
		"--no-color", "--no-ext-diff", "--no-renames",
		"--src-prefix=a/", "--dst-prefix=b/",
		fromRev, toRev)
	cmd.Dir = repoDir
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return Diff{}, fmt.Errorf("git diff %s %s: %w: %s", fromRev, toRev, err, strings.TrimSpace(errBuf.String()))
	}
	return parseUnifiedDiff(out.String()), nil
}

// parseUnifiedDiff turns canonical `git diff` text into a Diff. It is a pure
// function over the diff text (the I/O lives in Compute), exercised through
// Compute's real-git tests.
func parseUnifiedDiff(text string) Diff {
	var files []FileDiff
	var cur FileDiff
	var inFile bool
	flush := func() {
		if inFile {
			files = append(files, cur)
		}
	}
	for _, line := range strings.Split(text, "\n") {
		switch {
		case strings.HasPrefix(line, "diff --git "):
			flush()
			cur = FileDiff{Path: pathFromDiffGit(line)}
			inFile = true
		case !inFile:
			// preamble before the first file header (none for plain git diff)
		case strings.HasPrefix(line, "+++ b/"):
			cur.Path = strings.TrimPrefix(line, "+++ b/")
		case strings.HasPrefix(line, "@@"):
			if h, ok := parseHunkHeader(line); ok {
				cur.Hunks = append(cur.Hunks, h)
			}
		case strings.HasPrefix(line, "+++ "), strings.HasPrefix(line, "--- "):
			// file headers (incl. /dev/null sides) — not content. Detected with
			// the trailing space so an added line like "++++x" is NOT skipped.
		case strings.HasPrefix(line, "+"):
			cur.Added++
		case strings.HasPrefix(line, "-"):
			cur.Deleted++
		}
	}
	flush()
	return Diff{Files: files}
}

// pathFromDiffGit extracts the new path from a "diff --git a/<p> b/<p>" header
// (the text after " b/"). Renames/quoted paths-with-spaces are out of scope.
func pathFromDiffGit(line string) string {
	if i := strings.Index(line, " b/"); i >= 0 {
		return line[i+len(" b/"):]
	}
	return ""
}

// parseHunkHeader parses "@@ -OldStart[,OldLines] +NewStart[,NewLines] @@ ...".
// An omitted count means 1. Returns ok=false if the ranges can't be parsed.
func parseHunkHeader(line string) (Hunk, bool) {
	var oldSeg, newSeg string
	for _, f := range strings.Fields(line) {
		if oldSeg == "" && strings.HasPrefix(f, "-") {
			oldSeg = f[1:]
		}
		if newSeg == "" && strings.HasPrefix(f, "+") {
			newSeg = f[1:]
		}
	}
	os, ol, ok1 := parseRange(oldSeg)
	ns, nl, ok2 := parseRange(newSeg)
	if !ok1 || !ok2 {
		return Hunk{}, false
	}
	return Hunk{OldStart: os, OldLines: ol, NewStart: ns, NewLines: nl}, true
}

// parseRange parses a hunk range segment "start,lines" or "start" (lines
// defaults to 1, per unified-diff convention for a count of one).
func parseRange(seg string) (start, lines int, ok bool) {
	if seg == "" {
		return 0, 0, false
	}
	if i := strings.IndexByte(seg, ','); i >= 0 {
		st, e1 := strconv.Atoi(seg[:i])
		ln, e2 := strconv.Atoi(seg[i+1:])
		if e1 != nil || e2 != nil {
			return 0, 0, false
		}
		return st, ln, true
	}
	st, err := strconv.Atoi(seg)
	if err != nil {
		return 0, 0, false
	}
	return st, 1, true
}
