// Package reanchor maps a thread's line anchor from the revision it was filed
// against onto a later revision (DESIGN §28). It distinguishes "the code moved"
// (re-anchor) from "the code changed" (outdate) via a stored content hash, and
// surfaces a rename as a DISTINCT LostViaRename state rather than a silent drop
// — at the confirmed-catch layer that becomes a no-oracle-signal, never a
// phantom catch.
package reanchor

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os/exec"
	"strings"

	"github.com/joaomdsg/agntpr/internal/diff"
)

// State is how an anchor fared across the two revisions.
type State string

const (
	// Same: the anchored file was untouched between the revisions.
	Same State = "same"
	// Moved: the file changed elsewhere; the anchored lines shifted but their
	// content still matches, so the anchor re-points to the new line range.
	Moved State = "moved"
	// Outdated: the anchored lines were edited (or drifted) — the comment no
	// longer reliably points at the same code.
	Outdated State = "outdated"
	// LostViaRename: the anchored file was renamed; the line set does not
	// cleanly carry across, so the anchor is reported lost with the new path.
	LostViaRename State = "lost_via_rename"
)

// Anchor is a thread's stored anchor against fromRev: a 1-based inclusive line
// range in Path, plus the content hash of those lines (HashLines) used to tell
// a move from an edit.
type Anchor struct {
	Path     string
	Start    int
	End      int
	LineHash string
}

// Result is where an Anchor lands at toRev. For LostViaRename, Path is the new
// path and Start/End are zero; for Outdated the original path is retained.
type Result struct {
	State State
	Path  string
	Start int
	End   int
}

// HashLines returns a stable content hash of the given text, used to compute an
// Anchor.LineHash and to verify a moved anchor still covers the same content.
func HashLines(lines string) string {
	sum := sha256.Sum256([]byte(lines))
	return hex.EncodeToString(sum[:])
}

// Reanchor applies the DESIGN §28 algorithm: it classifies how a.Path changed
// between fromRev and toRev and maps the anchor accordingly.
func Reanchor(ctx context.Context, repoDir string, a Anchor, fromRev, toRev string) (Result, error) {
	status, err := fileStatus(ctx, repoDir, a.Path, fromRev, toRev)
	if err != nil {
		return Result{}, err
	}
	switch {
	case status.kind == statusUnchanged:
		return Result{State: Same, Path: a.Path, Start: a.Start, End: a.End}, nil
	case status.kind == statusRenamed:
		return Result{State: LostViaRename, Path: status.newPath}, nil
	case status.kind == statusDeleted:
		return Result{State: Outdated, Path: a.Path}, nil
	}

	d, err := diff.Compute(ctx, repoDir, fromRev, toRev)
	if err != nil {
		return Result{}, err
	}
	var hunks []diff.Hunk
	for _, f := range d.Files {
		if f.Path == a.Path {
			hunks = f.Hunks
			break
		}
	}

	delta := 0
	for _, h := range hunks {
		oldEnd := h.OldStart + h.OldLines - 1
		if h.OldLines > 0 && h.OldStart <= a.End && a.Start <= oldEnd {
			return Result{State: Outdated, Path: a.Path}, nil // edited lines overlap the anchor
		}
		if oldEnd < a.Start {
			delta += h.NewLines - h.OldLines
		}
	}

	s1, e1 := a.Start+delta, a.End+delta
	content, err := fileAt(ctx, repoDir, toRev, a.Path)
	if err != nil {
		return Result{}, err
	}
	if !rangeHashMatches(content, s1, e1, a.LineHash) {
		return Result{State: Outdated, Path: a.Path}, nil // drifted: don't mis-anchor
	}
	return Result{State: Moved, Path: a.Path, Start: s1, End: e1}, nil
}

func rangeHashMatches(content string, start, end int, want string) bool {
	lines := strings.Split(content, "\n")
	if start < 1 || end > len(lines) || start > end {
		return false
	}
	return HashLines(strings.Join(lines[start-1:end], "\n")) == want
}

type statusKind int

const (
	statusUnchanged statusKind = iota
	statusModified
	statusDeleted
	statusRenamed
)

type fileChange struct {
	kind    statusKind
	newPath string
}

// fileStatus classifies how path changed between the revisions using git's
// name-status with rename detection. A path absent from the output is
// unchanged; an "R" record (path on the old side) is a rename; "D" a deletion;
// anything else a modification.
func fileStatus(ctx context.Context, repoDir, path, fromRev, toRev string) (fileChange, error) {
	out, err := git(ctx, repoDir, "diff", "--find-renames", "--name-status", fromRev, toRev)
	if err != nil {
		return fileChange{}, err
	}
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		code := fields[0]
		switch {
		case strings.HasPrefix(code, "R") && len(fields) == 3:
			if fields[1] == path {
				return fileChange{kind: statusRenamed, newPath: fields[2]}, nil
			}
		case code == "D" && len(fields) == 2:
			if fields[1] == path {
				return fileChange{kind: statusDeleted}, nil
			}
		case len(fields) == 2:
			if fields[1] == path {
				return fileChange{kind: statusModified}, nil
			}
		}
	}
	return fileChange{kind: statusUnchanged}, nil
}

// fileAt returns the contents of path at rev.
func fileAt(ctx context.Context, repoDir, rev, path string) (string, error) {
	return git(ctx, repoDir, "show", rev+":"+path)
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
