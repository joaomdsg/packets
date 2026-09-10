package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/ledger"
)

func gitSplit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %s: %s", strings.Join(args, " "), out)
	return strings.TrimSpace(string(out))
}

func writeSplitFile(t *testing.T, dir, name, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
}

func commitSplit(t *testing.T, dir, msg string) string {
	t.Helper()
	gitSplit(t, dir, "add", "-A")
	gitSplit(t, dir, "commit", "-qm", msg)
	return gitSplit(t, dir, "rev-parse", "HEAD")
}

// splitRepo builds a repo whose base→fix changes a file in TWO well-separated
// regions, so the diff yields two hunks — the raw material a split proposal is
// harvested from. Returns the dir + the base/fix shas.
func splitRepo(t *testing.T) (dir, base, fix string) {
	t.Helper()
	dir = t.TempDir()
	gitSplit(t, dir, "init", "-q")
	gitSplit(t, dir, "config", "user.email", "t@t")
	gitSplit(t, dir, "config", "user.name", "t")
	var b strings.Builder
	for i := 1; i <= 25; i++ {
		fmt.Fprintf(&b, "line %d\n", i)
	}
	writeSplitFile(t, dir, "pay.go", b.String())
	base = commitSplit(t, dir, "base")
	// Change two regions far apart (>6 lines) so git emits two distinct hunks.
	changed := strings.ReplaceAll(b.String(), "line 2\n", "line 2 CHANGED\n")
	changed = strings.ReplaceAll(changed, "line 20\n", "line 20 CHANGED\n")
	writeSplitFile(t, dir, "pay.go", changed)
	fix = commitSplit(t, dir, "change two regions")
	return dir, base, fix
}

func TestSplitCandidates_proposesTheChangedRegionsAsSubTargets(t *testing.T) {
	t.Parallel()
	dir, base, fix := splitRepo(t)
	parent := ledger.Target{BaseRev: base, FixRev: fix, TipRev: fix, Path: "pay.go", Line: 10}

	cands := splitCandidates(context.Background(), dir, parent)
	require.GreaterOrEqual(t, len(cands), 2, "two changed regions propose at least two sub-targets to split into")
	for _, c := range cands {
		assert.Equal(t, "pay.go", c.Path, "a proposed sub-target stays in the parent's file")
		assert.NotEqual(t, parent.Line, c.Line, "a proposal is a DISTINCT line, never the parent itself")
		assert.Equal(t, base, c.BaseRev, "the sub-target carries the parent's revs so it runs the same cycle")
		assert.Equal(t, fix, c.FixRev)
	}
	// The proposed lines are distinct (no duplicate sub-target for one region).
	seen := map[int]bool{}
	for _, c := range cands {
		require.False(t, seen[c.Line], "each proposed sub-target is a distinct line")
		seen[c.Line] = true
	}
}

func TestSplitCandidates_proposesNothingWhenTheTargetsFileIsUnchanged(t *testing.T) {
	t.Parallel()
	dir, base, fix := splitRepo(t)
	// A target on a file the diff never touched has nothing to split into.
	parent := ledger.Target{BaseRev: base, FixRev: fix, TipRev: fix, Path: "untouched.go", Line: 3}

	assert.Empty(t, splitCandidates(context.Background(), dir, parent),
		"no diff in the target's file → no split proposal (not a fabricated one)")
}

func TestSplitCandidates_proposesNothingWithoutAResolvableDiff(t *testing.T) {
	t.Parallel()
	// No repo dir / no revs → no proposal, never a panic or a git error surfaced.
	assert.Empty(t, splitCandidates(context.Background(), "", ledger.Target{Path: "pay.go", Line: 1}))
	dir, base, fix := splitRepo(t)
	assert.Empty(t, splitCandidates(context.Background(), dir, ledger.Target{Path: "pay.go", Line: 1, BaseRev: "", FixRev: fix}),
		"a target missing a base rev cannot be diffed → no proposal")
	_ = base
}
