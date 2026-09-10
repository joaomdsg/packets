package packet_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joaomdsg/packets/internal/packet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git %v:\n%s", args, out)
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

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	full := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
}

func commitAll(t *testing.T, dir, msg string) string {
	t.Helper()
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-qm", msg)
	return runGit(t, dir, "rev-parse", "HEAD")
}

// chainModule builds a reverse-dependency chain a->b->c (a imports b, b
// imports c) plus a standalone package d, four packages total. c also
// imports a standard-library package, so graph tests can assert that a
// non-local import never becomes an edge. Returns the repo dir and the base
// commit.
func chainModule(t *testing.T) (dir, base string) {
	t.Helper()
	dir = initRepo(t)
	write(t, dir, "go.mod", "module fixture.test/chain\n\ngo 1.21\n")
	write(t, dir, "a/a.go", "package a\n\nimport \"fixture.test/chain/b\"\n\nfunc A() { b.B() }\n")
	write(t, dir, "b/b.go", "package b\n\nimport \"fixture.test/chain/c\"\n\nfunc B() { c.C() }\n")
	write(t, dir, "c/c.go", "package c\n\nimport \"fmt\"\n\nfunc C() { fmt.Println(\"c\") }\n")
	write(t, dir, "d/d.go", "package d\n\nfunc D() {}\n")
	base = commitAll(t, dir, "base")
	return dir, base
}

// wideModule builds a diamond (a imports b and c; b and c both import d)
// among 16 total packages: 12 unrelated standalone "pad" packages dilute the
// module so a touched package's ratio lands cleanly inside a specific lane
// band, proving Measure scores against the WHOLE module's graph, not just
// the packages nearest the change.
func wideModule(t *testing.T) (dir, base string) {
	t.Helper()
	dir = initRepo(t)
	write(t, dir, "go.mod", "module fixture.test/wide\n\ngo 1.21\n")
	write(t, dir, "a/a.go", "package a\n\nimport (\n\t\"fixture.test/wide/b\"\n\t\"fixture.test/wide/c\"\n)\n\nfunc A() { b.B(); c.C() }\n")
	write(t, dir, "b/b.go", "package b\n\nimport \"fixture.test/wide/d\"\n\nfunc B() { d.D() }\n")
	write(t, dir, "c/c.go", "package c\n\nimport \"fixture.test/wide/d\"\n\nfunc C() { d.D() }\n")
	write(t, dir, "d/d.go", "package d\n\nfunc D() {}\n")
	for i := 0; i < 12; i++ {
		name := fmt.Sprintf("pad%d", i)
		write(t, dir, name+"/"+name+".go", fmt.Sprintf("package %s\n\nfunc F() {}\n", name))
	}
	base = commitAll(t, dir, "base")
	return dir, base
}

func TestLoadImportGraph_keysEveryModuleLocalPackageByItsImportPath(t *testing.T) {
	dir, _ := chainModule(t)

	graph, err := packet.LoadImportGraph(context.Background(), dir)

	require.NoError(t, err)
	assert.Contains(t, graph, "fixture.test/chain/a")
	assert.Contains(t, graph, "fixture.test/chain/b")
	assert.Contains(t, graph, "fixture.test/chain/c")
	assert.Contains(t, graph, "fixture.test/chain/d")
}

func TestLoadImportGraph_edgesFollowRealImportsAndDropStandardLibraryOnes(t *testing.T) {
	dir, _ := chainModule(t)

	graph, err := packet.LoadImportGraph(context.Background(), dir)

	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"fixture.test/chain/b"}, graph["fixture.test/chain/a"], "a imports b")
	assert.ElementsMatch(t, []string{"fixture.test/chain/c"}, graph["fixture.test/chain/b"], "b imports c")
	assert.Empty(t, graph["fixture.test/chain/c"], "c's only import is fmt — filtered as non-local")
	assert.Empty(t, graph["fixture.test/chain/d"], "d is standalone")
}

func TestChangedPackages_mapsEachChangedGoFileToItsContainingImportPath(t *testing.T) {
	dir, _ := chainModule(t)

	pkgs, err := packet.ChangedPackages(dir, []string{"c/c.go", "d/d.go"})

	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"fixture.test/chain/c", "fixture.test/chain/d"}, pkgs)
}

func TestChangedPackages_dedupesMultipleChangedFilesInTheSamePackage(t *testing.T) {
	dir := initRepo(t)
	write(t, dir, "go.mod", "module fixture.test/dedup\n\ngo 1.21\n")
	write(t, dir, "p/one.go", "package p\n\nfunc One() {}\n")
	write(t, dir, "p/two.go", "package p\n\nfunc Two() {}\n")
	commitAll(t, dir, "base")

	pkgs, err := packet.ChangedPackages(dir, []string{"p/one.go", "p/two.go"})

	require.NoError(t, err)
	assert.Equal(t, []string{"fixture.test/dedup/p"}, pkgs)
}

func TestChangedPackages_ignoresNonGoFilesLeavingAnEmptySetWhenNoneQualify(t *testing.T) {
	dir, _ := chainModule(t)

	pkgs, err := packet.ChangedPackages(dir, []string{"README.md", "docs/notes.txt"})

	require.NoError(t, err)
	assert.Empty(t, pkgs)
}

func TestChangedPackages_dropsNonGoFilesFromAMixedBatchButStillResolvesTheGoOnes(t *testing.T) {
	dir, _ := chainModule(t)

	pkgs, err := packet.ChangedPackages(dir, []string{"c/c.go", "README.md"})

	require.NoError(t, err)
	assert.Equal(t, []string{"fixture.test/chain/c"}, pkgs)
}

func TestChangedPackages_returnsNothingForAnEmptyFileSetWithoutRunningGoList(t *testing.T) {
	dir, _ := chainModule(t)

	pkgs, err := packet.ChangedPackages(dir, nil)

	require.NoError(t, err)
	assert.Empty(t, pkgs)
}

func TestChangedPackages_resolvesARootLevelChangedFileToTheRootPackage(t *testing.T) {
	dir := initRepo(t)
	write(t, dir, "go.mod", "module fixture.test/root\n\ngo 1.21\n")
	write(t, dir, "main.go", "package main\n\nfunc main() {}\n")
	commitAll(t, dir, "base")

	pkgs, err := packet.ChangedPackages(dir, []string{"main.go"})

	require.NoError(t, err)
	assert.Equal(t, []string{"fixture.test/root"}, pkgs)
}

func TestMeasure_strictLaneWhenAChainLeafRipplesThroughMostOfTheModule(t *testing.T) {
	dir, base := chainModule(t)
	write(t, dir, "c/c.go", "package c\n\nimport \"fmt\"\n\nfunc C() { fmt.Println(\"c changed\") }\n")
	fix := commitAll(t, dir, "touch c")

	lane, err := packet.Measure(context.Background(), dir, base, fix)

	require.NoError(t, err)
	assert.Equal(t, packet.LaneStrict, lane, "c's reverse closure {a,b,c} out of {a,b,c,d} is 75%, past the 40% strict threshold")
}

func TestMeasure_standardLaneWhenADiamondLeafRipplesThroughAQuarterOfTheModule(t *testing.T) {
	dir, base := wideModule(t)
	write(t, dir, "d/d.go", "package d\n\nfunc D() { _ = 1 }\n")
	fix := commitAll(t, dir, "touch d")

	lane, err := packet.Measure(context.Background(), dir, base, fix)

	require.NoError(t, err)
	assert.Equal(t, packet.LaneStandard, lane, "d's reverse closure {a,b,c,d} out of 16 total packages is 25%")
}

func TestMeasure_bestEffortLaneWhenTheChangeTouchesOnlyAnIsolatedPackage(t *testing.T) {
	dir, base := wideModule(t)
	write(t, dir, "pad0/pad0.go", "package pad0\n\nfunc F() { _ = 1 }\n")
	fix := commitAll(t, dir, "touch pad0")

	lane, err := packet.Measure(context.Background(), dir, base, fix)

	require.NoError(t, err)
	assert.Equal(t, packet.LaneBestEffort, lane, "pad0 has no importers: 1 of 16 packages is 6.25%")
}

func TestMeasure_unmeasuredWhenTheOnlyChangeIsNotGoCode(t *testing.T) {
	dir, base := chainModule(t)
	write(t, dir, "README.md", "hello\n")
	fix := commitAll(t, dir, "touch readme")

	lane, err := packet.Measure(context.Background(), dir, base, fix)

	require.NoError(t, err)
	assert.Equal(t, packet.LaneUnmeasured, lane, "a non-.go change contributes no changed package to measure")
}

func TestMeasure_neverGuessesALaneWhenARevisionFailsToResolve(t *testing.T) {
	dir, base := chainModule(t)

	lane, err := packet.Measure(context.Background(), dir, base, "not-a-real-rev")

	assert.Error(t, err)
	assert.Equal(t, packet.LaneUnmeasured, lane, "an unresolved revision must never fall back to a guessed lane")
}

func TestMeasure_neverGuessesALaneWhenTheChangedPackageCannotBeResolved(t *testing.T) {
	dir, base := chainModule(t)
	// A broken go.mod makes `go list` fail for the changed .go file's own
	// directory — the ChangedPackages stage, reached BEFORE LoadImportGraph.
	write(t, dir, "go.mod", "this is not a valid go.mod\n")
	write(t, dir, "c/c.go", "package c\n\nfunc C() { _ = 1 }\n")
	fix := commitAll(t, dir, "break go.mod and touch c")

	lane, err := packet.Measure(context.Background(), dir, base, fix)

	require.Error(t, err)
	assert.ErrorContains(t, err, "measure: changed packages", "the error names which stage failed")
	assert.Equal(t, packet.LaneUnmeasured, lane, "an unresolvable changed package must never fall back to a guessed lane")
}

func TestMeasure_neverGuessesALaneWhenTheImportGraphCannotBeLoaded(t *testing.T) {
	dir, base := chainModule(t)
	// Only a non-.go file changes (ChangedPackages short-circuits without
	// running go list at all), but go.mod is broken too, so the LATER
	// LoadImportGraph stage is the one that fails.
	write(t, dir, "go.mod", "this is not a valid go.mod\n")
	write(t, dir, "README.md", "hello\n")
	fix := commitAll(t, dir, "break go.mod and touch readme")

	lane, err := packet.Measure(context.Background(), dir, base, fix)

	require.Error(t, err)
	assert.ErrorContains(t, err, "measure: import graph", "the error names which stage failed")
	assert.Equal(t, packet.LaneUnmeasured, lane, "an unresolvable import graph must never fall back to a guessed lane")
}
