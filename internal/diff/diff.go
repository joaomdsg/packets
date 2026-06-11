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
// at repoDir. Rename detection is intentionally OFF (--no-renames overrides any
// diff.renames config): a rename shows as a delete + an add (see RISKS.md
// "reanchor-rename-similarity-cliff", a later brick). Line CONTENT is not
// captured (ranges/counts only).
//
// Paths and add/delete counts come from `--numstat -z`, NOT from the patch
// headers: git C-quotes any path containing a tab/newline/double-quote/control
// char (and, without quotepath=false, every non-ASCII path) in the
// `diff --git`/`+++ b/` headers (`+++ "b/tab\tname.txt"`), which a header parse
// would mangle so a consumer matching on the anchor's path never finds the file.
// `--numstat -z` emits RAW paths NUL-terminated, immune to all quoting. Hunks
// still come from the unified patch, associated to files by git's stable
// emission order (identical for both invocations given identical args).
func Compute(ctx context.Context, repoDir, fromRev, toRev string) (Diff, error) {
	stat, err := git(ctx, repoDir, "diff", "--numstat", "-z", "--no-renames", fromRev, toRev)
	if err != nil {
		return Diff{}, err
	}
	patch, err := git(ctx, repoDir, "diff",
		"--no-color", "--no-ext-diff", "--no-renames",
		"--src-prefix=a/", "--dst-prefix=b/",
		fromRev, toRev)
	if err != nil {
		return Diff{}, err
	}
	files := parseNumstatZ(stat)
	groups := parseHunkGroups(patch)
	for i := range files {
		if i < len(groups) {
			files[i].Hunks = groups[i]
		}
	}
	return Diff{Files: files}, nil
}

func git(ctx context.Context, repoDir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoDir
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(errBuf.String()))
	}
	return out.String(), nil
}

// parseNumstatZ turns `git diff --numstat -z` output into per-file Path + counts.
// Each record is NUL-terminated and tab-separated as "added\tdeleted\tpath"; the
// path is taken as everything after the second tab so a TAB in the filename is
// preserved. A "-" count (a binary file) becomes 0.
func parseNumstatZ(text string) []FileDiff {
	var files []FileDiff
	for _, rec := range strings.Split(text, "\x00") {
		if rec == "" {
			continue
		}
		fields := strings.SplitN(rec, "\t", 3)
		if len(fields) != 3 {
			continue
		}
		files = append(files, FileDiff{
			Path:    fields[2],
			Added:   countOrZero(fields[0]),
			Deleted: countOrZero(fields[1]),
		})
	}
	return files
}

func countOrZero(field string) int {
	if field == "-" {
		return 0 // binary file: numstat reports "-" for both sides
	}
	n, err := strconv.Atoi(field)
	if err != nil {
		return 0
	}
	return n
}

// parseHunkGroups returns each file's hunks from the unified patch, in git's
// emission order — one group per `diff --git` header, so the i-th group belongs
// to the i-th file parseNumstatZ reports. Paths in the headers are ignored (they
// may be quoted); only the ordered `@@` ranges are read.
func parseHunkGroups(text string) [][]Hunk {
	var groups [][]Hunk
	started := false
	for _, line := range strings.Split(text, "\n") {
		switch {
		case strings.HasPrefix(line, "diff --git "):
			groups = append(groups, nil)
			started = true
		case started && strings.HasPrefix(line, "@@"):
			if h, ok := parseHunkHeader(line); ok {
				groups[len(groups)-1] = append(groups[len(groups)-1], h)
			}
		}
	}
	return groups
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
