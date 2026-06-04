package app_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/agntpr/internal/app"
	"github.com/joaomdsg/agntpr/internal/catch"
	"github.com/joaomdsg/agntpr/internal/ledger"
	"github.com/joaomdsg/agntpr/internal/reanchor"
	"github.com/joaomdsg/agntpr/internal/surface"
)

var goTestCmd = []string{"env", "-u", "GOROOT", "go", "test", "./..."}

const adultGo = "package adult\n\nfunc IsAdult(age int) bool {\n\treturn age >= 18\n}\n"

const weakTest = "package adult\n\nimport \"testing\"\n\nfunc TestIsAdult(t *testing.T) {\n\tif !IsAdult(25) {\n\t\tt.Fatal(\"25\")\n\t}\n}\n"

const strongTest = "package adult\n\nimport \"testing\"\n\nfunc TestIsAdult(t *testing.T) {\n\tif IsAdult(17) {\n\t\tt.Fatal(\"17 is not an adult\")\n\t}\n\tif !IsAdult(18) {\n\t\tt.Fatal(\"18 is an adult\")\n\t}\n}\n"

// adultPadded keeps the `>=` anchor on line 4 but adds room far below it so a
// bottom-of-file edit re-anchors as Moved (not context-overlap Outdated).
const adultPadded = "package adult\n\nfunc IsAdult(age int) bool {\n\treturn age >= 18\n}\n\n// pad\n// pad\n// pad\n// pad\n// pad\n// pad\nvar Marker = 1\n"

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
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
}

func anchor() reanchor.Anchor {
	return reanchor.Anchor{Path: "adult.go", Start: 4, End: 4, LineHash: reanchor.HashLines("\treturn age >= 18")}
}

func TestResolve_mintsAndLogsACatchWhenTheTestIsStrengthened(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "go.mod", "module adultapp\n\ngo 1.23\n")
	write(t, dir, "adult.go", adultGo)
	write(t, dir, "adult_test.go", weakTest)
	base := commitAll(t, dir, "base")
	write(t, dir, "adult_test.go", strongTest)
	fix := commitAll(t, dir, "strengthen the test")

	res, err := app.Resolve(context.Background(), dir, base, fix, anchor(), goTestCmd, true, true)
	require.NoError(t, err)
	assert.Equal(t, string(catch.Catch), res.Verdict)
	require.NotNil(t, res.Record, "a real mint must produce a record")
	assert.Equal(t, catch.Catch, res.Record.Outcome)
	assert.Equal(t, "adult.go", res.Record.Path)
	assert.Equal(t, 1, res.Record.MutantsConsidered)
	assert.True(t, res.Record.SelfFlagged)
	assert.True(t, res.Record.WouldHaveShipped)
	assert.NotEmpty(t, res.Record.BeforeInventory)
	assert.NotEmpty(t, res.Record.AfterInventory)

	logPath := filepath.Join(t.TempDir(), "catches.jsonl")
	l, err := ledger.Open(logPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = l.Close() })
	require.NoError(t, l.Append(*res.Record))
	got, err := l.Records()
	require.NoError(t, err)
	assert.Len(t, got, 1, "the catch is durably logged")
}

func TestResolve_mintsNothingWhenTheFixEditsTheAnchoredLine(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "go.mod", "module adultapp\n\ngo 1.23\n")
	write(t, dir, "adult.go", adultGo)
	write(t, dir, "adult_test.go", weakTest)
	base := commitAll(t, dir, "base")
	write(t, dir, "adult.go", "package adult\n\nfunc IsAdult(age int) bool {\n\treturn age > 18\n}\n")
	fix := commitAll(t, dir, "edit the anchored line")

	res, err := app.Resolve(context.Background(), dir, base, fix, anchor(), goTestCmd, false, false)
	require.NoError(t, err)
	assert.Nil(t, res.Record, "an edited anchored line mints nothing to persist")
	assert.NotEqual(t, string(catch.Catch), res.Verdict, "and never claims a catch over the wire")

	logPath := filepath.Join(t.TempDir(), "catches.jsonl")
	l, err := ledger.Open(logPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = l.Close() })
	got, err := l.Records()
	require.NoError(t, err)
	assert.Empty(t, got, "edit/no-op work leaves no trace in the ledger")
}

func TestResolve_propagatesACycleError(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "go.mod", "module adultapp\n\ngo 1.23\n")
	write(t, dir, "adult.go", adultGo)
	head := commitAll(t, dir, "base")

	_, err := app.Resolve(context.Background(), dir, "deadbeefdeadbeef", head, anchor(), goTestCmd, false, false)
	require.Error(t, err, "a failed cycle (bad revision) must propagate, not silently resolve")
}

func TestResolve_rendersTestedNotBlindForAnAlreadyStrongLine(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "go.mod", "module adultapp\n\ngo 1.23\n")
	write(t, dir, "adult.go", adultPadded)
	write(t, dir, "adult_test.go", strongTest)
	base := commitAll(t, dir, "base: already strong")
	write(t, dir, "adult.go", strings.Replace(adultPadded, "var Marker = 1", "var Marker = 2", 1))
	fix := commitAll(t, dir, "behavior-neutral churn far below the anchor")

	res, err := app.Resolve(context.Background(), dir, base, fix, anchor(), goTestCmd, false, false)
	require.NoError(t, err)
	assert.Equal(t, surface.Tested, res.Verdict, "a verified-strong line reads as Tested, not blind no-signal")
	assert.Nil(t, res.Record, "no catch to mint on an already-strong line")
}
