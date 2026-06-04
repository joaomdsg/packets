package refactor_test

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

	"github.com/joaomdsg/agntpr/internal/diff"
	"github.com/joaomdsg/agntpr/internal/mutation"
	"github.com/joaomdsg/agntpr/internal/reanchor"
)

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git %v: %s", args, out)
	return strings.TrimSpace(string(out))
}

func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "-q")
	runGit(t, dir, "config", "user.email", "t@t")
	runGit(t, dir, "config", "user.name", "t")
	return dir
}

func commitAll(t *testing.T, dir, msg string) string {
	t.Helper()
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-qm", msg)
	return runGit(t, dir, "rev-parse", "HEAD")
}

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
}

func TestRefactorTrace_largeRenameOrphansEveryAnchoredThread(t *testing.T) {
	t.Parallel()
	const n = 40
	dir := initRepo(t)
	for i := 0; i < n; i++ {
		write(t, dir, fmt.Sprintf("f%02d.go", i), fmt.Sprintf("package p\n\nvar V%02d = %d\n", i, i))
	}
	base := commitAll(t, dir, "base: 40 files")
	for i := 0; i < n; i++ {
		runGit(t, dir, "mv", fmt.Sprintf("f%02d.go", i), fmt.Sprintf("r%02d.go", i))
	}
	head := commitAll(t, dir, "rename all 40 files")

	orphaned := 0
	for i := 0; i < n; i++ {
		a := reanchor.Anchor{Path: fmt.Sprintf("f%02d.go", i), Start: 3, End: 3, LineHash: reanchor.HashLines(fmt.Sprintf("var V%02d = %d", i, i))}
		got, err := reanchor.Reanchor(context.Background(), dir, a, base, head)
		require.NoError(t, err)
		if got.State == reanchor.LostViaRename || got.State == reanchor.Outdated {
			orphaned++
		}
	}
	assert.Equal(t, n, orphaned, "every anchored thread is orphaned by the rename cliff (carnage baseline)")
}

func TestRefactorTrace_neutralRenameYieldsLostViaRenameNotACatch(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	src := "package p\n\nfunc f(a, b int) bool {\n\treturn a >= b\n}\n"
	write(t, dir, "orig.go", src)
	base := commitAll(t, dir, "base")
	runGit(t, dir, "mv", "orig.go", "moved.go") // content byte-identical
	head := commitAll(t, dir, "neutral rename")

	a := reanchor.Anchor{Path: "orig.go", Start: 4, End: 4, LineHash: reanchor.HashLines("\treturn a >= b")}
	got, err := reanchor.Reanchor(context.Background(), dir, a, base, head)
	require.NoError(t, err)
	// The anchor does not survive as Same/Moved on its original path — so the
	// catch layer gets no anchored line to compare and cannot fire.
	assert.Equal(t, reanchor.LostViaRename, got.State)
	assert.Equal(t, "moved.go", got.Path)
	assert.NotEqual(t, reanchor.Moved, got.State)
}

func TestRefactorTrace_extractModuleIsInvisibleToDiffAndReMutated(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	const helper = "func atLeast(a, b int) bool {\n\treturn a >= b\n}\n"
	write(t, dir, "a.go", "package p\n\nfunc Use(x int) bool {\n\treturn atLeast(x, 18)\n}\n\n"+helper)
	write(t, dir, "b.go", "package p\n")
	base := commitAll(t, dir, "base")
	// Extract atLeast from a.go into b.go — pure relocation, behavior identical.
	write(t, dir, "a.go", "package p\n\nfunc Use(x int) bool {\n\treturn atLeast(x, 18)\n}\n")
	write(t, dir, "b.go", "package p\n\n"+helper)
	head := commitAll(t, dir, "extract atLeast into b.go")

	d, err := diff.Compute(context.Background(), dir, base, head)
	require.NoError(t, err)
	a, aok := fileByPath(d, "a.go")
	b, bok := fileByPath(d, "b.go")
	require.True(t, aok && bok, "the move surfaces as two unlinked file changes")
	// Invisible-as-a-move: A only loses lines, B only gains them; nothing links them.
	assert.Positive(t, a.Deleted, "a.go loses the extracted function")
	assert.Positive(t, b.Added, "b.go gains the extracted function")

	// The relocated `>=` in b.go is re-mutated as a net-new site, not recognized
	// as preserved behavior.
	newB := "package p\n\n" + helper
	mutants, err := mutation.GenerateMutants([]byte(newB), nil)
	require.NoError(t, err)
	relocated := 0
	for _, m := range mutants {
		if m.Original == ">=" {
			relocated++
		}
	}
	assert.Positive(t, relocated, "the moved `>=` operator is re-mutated as if it were new code")
}

func fileByPath(d diff.Diff, path string) (diff.FileDiff, bool) {
	for _, f := range d.Files {
		if f.Path == path {
			return f, true
		}
	}
	return diff.FileDiff{}, false
}
