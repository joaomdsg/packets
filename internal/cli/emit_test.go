package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joaomdsg/packets/internal/cli"
	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// queueLLM is a hand-rolled stand-in for the "claude -p" process boundary:
// it returns canned responses in call order and is never a real process.
type queueLLM struct {
	responses []string
	calls     int
}

func (q *queueLLM) Review(prompt string) (string, error) {
	i := q.calls
	q.calls++
	if i >= len(q.responses) {
		return `{"warnings":[],"suggested_terminal":[]}`, nil
	}
	return q.responses[i], nil
}

// emitFixture registers a fabric under isolated XDG dirs and returns the
// packets dir emit will read and write, plus the --fabric slug to pass.
// t.Setenv forbids t.Parallel on these tests, since it mutates process env.
func emitFixture(t *testing.T) (fabricSlug, packetsDir string) {
	t.Helper()
	configDir := t.TempDir()
	dataDir := t.TempDir()
	repoDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(repoDir, "go.mod"), []byte("module x\n"), 0o600))
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("XDG_DATA_HOME", dataDir)

	const slugName = "test-fabric"
	require.NoError(t, os.MkdirAll(filepath.Join(configDir, "packets"), 0o700))
	require.NoError(t, fabric.SaveIndex(filepath.Join(configDir, "packets", "fabrics.yaml"), &fabric.Index{
		Fabrics: []fabric.IndexEntry{{Slug: slugName, RepoPath: repoDir}},
	}))
	return slugName, filepath.Join(dataDir, "packets", slugName, "packets")
}

func writePacketFile(t *testing.T, dir, contents string) string {
	t.Helper()
	path := filepath.Join(dir, "packet.yaml")
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
	return path
}

func TestEmit_printsEveryFailingRuleOnInvalidPacket(t *testing.T) {
	fabricSlug, _ := emitFixture(t)
	path := writePacketFile(t, t.TempDir(), `
goal: ""
terminal: []
budget:
  retries: 0
  minutes: 0
`)
	out := &bytes.Buffer{}
	root := cli.NewRoot(cli.WithGateLLM(&queueLLM{}))
	root.SetOut(out)
	root.SetArgs([]string{"emit", "--fabric", fabricSlug, path})

	err := root.Execute()

	assert.Error(t, err)
	assert.Contains(t, out.String(), "goal: must not be empty")
	assert.Contains(t, out.String(), "terminal: must contain at least one entry")
	assert.Contains(t, out.String(), "budget.retries: must be >= 1")
	assert.Contains(t, out.String(), "budget.minutes: must be >= 1")

	fabricLogData, err := os.ReadFile(filepath.Join(filepath.Dir(mustPacketsDir(t, fabricSlug)), "log.jsonl"))
	require.NoError(t, err)
	assert.Contains(t, string(fabricLogData), `"event":"emit_reject"`)
	assert.Contains(t, string(fabricLogData), `"actor":"human"`)
}

func TestEmit_reportsValidationFailuresWithNoFabricRegistered(t *testing.T) {
	// t.Setenv forbids t.Parallel. No fabric is registered at all, unlike
	// emitFixture's isolated-but-registered setup.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	path := writePacketFile(t, t.TempDir(), `
goal: ""
terminal: []
budget:
  retries: 0
  minutes: 0
`)
	out := &bytes.Buffer{}
	root := cli.NewRoot(cli.WithGateLLM(&queueLLM{}))
	root.SetOut(out)
	root.SetArgs([]string{"emit", path})

	err := root.Execute()

	assert.Error(t, err)
	assert.Contains(t, out.String(), "goal: must not be empty")
	assert.Contains(t, out.String(), "terminal: must contain at least one entry")
}

// mustPacketsDir re-derives the packets dir for a fabric slug registered
// under the current test's XDG env, mirroring what resolveFabric computes.
func mustPacketsDir(t *testing.T, fabricSlug string) string {
	t.Helper()
	dataDir := os.Getenv("XDG_DATA_HOME")
	return filepath.Join(dataDir, "packets", fabricSlug, "packets")
}

func TestEmit_createsPacketAndPrintsSlugWhenGateHasNoWarnings(t *testing.T) {
	fabricSlug, packetsDir := emitFixture(t)
	path := writePacketFile(t, t.TempDir(), `
goal: "Fix the login timeout"
terminal:
  - "true"
budget:
  retries: 5
  minutes: 60
`)
	out := &bytes.Buffer{}
	root := cli.NewRoot(cli.WithGateLLM(&queueLLM{responses: []string{
		`{"warnings":[],"suggested_terminal":[]}`,
	}}))
	root.SetOut(out)
	root.SetArgs([]string{"emit", "--fabric", fabricSlug, path})

	err := root.Execute()

	require.NoError(t, err)
	assert.Contains(t, out.String(), "fix-the-login-timeout")

	dir := filepath.Join(packetsDir, "fix-the-login-timeout")
	got, err := state.Load(filepath.Join(dir, "state.json"))
	require.NoError(t, err)
	assert.Equal(t, state.Emitted, got.State)

	logData, err := os.ReadFile(filepath.Join(dir, "log.jsonl"))
	require.NoError(t, err)
	assert.Contains(t, string(logData), `"event":"emit"`)
}

func TestEmit_ambiguousGoalWarningSurfacesAndOverrideEmits(t *testing.T) {
	fabricSlug, packetsDir := emitFixture(t)
	path := writePacketFile(t, t.TempDir(), `
goal: "improve things"
terminal:
  - "true"
budget:
  retries: 5
  minutes: 60
`)
	out := &bytes.Buffer{}
	root := cli.NewRoot(cli.WithGateLLM(&queueLLM{responses: []string{
		`{"warnings":[{"code":"AMBIGUOUS_GOAL","detail":"no measure given"}],"suggested_terminal":[]}`,
	}}))
	root.SetOut(out)
	root.SetIn(strings.NewReader("y\n"))
	root.SetArgs([]string{"emit", "--fabric", fabricSlug, path})

	err := root.Execute()

	require.NoError(t, err)
	assert.Contains(t, out.String(), "AMBIGUOUS_GOAL")

	dir := filepath.Join(packetsDir, "improve-things")
	logData, err := os.ReadFile(filepath.Join(dir, "log.jsonl"))
	require.NoError(t, err)
	assert.Contains(t, string(logData), `"event":"emit_warn"`)
	assert.Contains(t, string(logData), `"event":"emit_override"`)
	assert.Contains(t, string(logData), `"AMBIGUOUS_GOAL"`)
}

func TestEmit_decliningOverrideExitsWithoutCreatingPacket(t *testing.T) {
	fabricSlug, packetsDir := emitFixture(t)
	path := writePacketFile(t, t.TempDir(), `
goal: "improve things"
terminal:
  - "true"
budget:
  retries: 5
  minutes: 60
`)
	out := &bytes.Buffer{}
	root := cli.NewRoot(cli.WithGateLLM(&queueLLM{responses: []string{
		`{"warnings":[{"code":"AMBIGUOUS_GOAL","detail":"no measure given"}],"suggested_terminal":[]}`,
	}}))
	root.SetOut(out)
	root.SetIn(strings.NewReader("N\n"))
	root.SetArgs([]string{"emit", "--fabric", fabricSlug, path})

	err := root.Execute()

	assert.Error(t, err)
	_, statErr := os.Stat(filepath.Join(packetsDir, "improve-things"))
	assert.True(t, os.IsNotExist(statErr))
}

func TestEmit_acceptingSuggestedTerminalAppendsAndRecordsIt(t *testing.T) {
	fabricSlug, packetsDir := emitFixture(t)
	path := writePacketFile(t, t.TempDir(), `
goal: "Fix the login timeout"
terminal:
  - "true"
budget:
  retries: 5
  minutes: 60
`)
	out := &bytes.Buffer{}
	root := cli.NewRoot(cli.WithGateLLM(&queueLLM{responses: []string{
		`{"warnings":[],"suggested_terminal":["npm test"]}`,
	}}))
	root.SetOut(out)
	root.SetIn(strings.NewReader("y\n"))
	root.SetArgs([]string{"emit", "--fabric", fabricSlug, path})

	err := root.Execute()

	require.NoError(t, err)

	dir := filepath.Join(packetsDir, "fix-the-login-timeout")
	packetData, err := os.ReadFile(filepath.Join(dir, "packet.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(packetData), "npm test")

	got, err := state.Load(filepath.Join(dir, "state.json"))
	require.NoError(t, err)
	assert.Equal(t, []string{"npm test"}, got.GateSuggestedTerminals)
}

func TestEmit_llmFailureProceedsWithZeroWarningsAndLogsError(t *testing.T) {
	fabricSlug, packetsDir := emitFixture(t)
	path := writePacketFile(t, t.TempDir(), `
goal: "Fix the login timeout"
terminal:
  - "true"
budget:
  retries: 5
  minutes: 60
`)
	out := &bytes.Buffer{}
	root := cli.NewRoot(cli.WithGateLLM(&queueLLM{responses: []string{
		"not json at all",
		"still not json",
	}}))
	root.SetOut(out)
	root.SetArgs([]string{"emit", "--fabric", fabricSlug, path})

	err := root.Execute()

	require.NoError(t, err)
	assert.Contains(t, out.String(), "fix-the-login-timeout")

	dir := filepath.Join(packetsDir, "fix-the-login-timeout")
	logData, err := os.ReadFile(filepath.Join(dir, "log.jsonl"))
	require.NoError(t, err)
	assert.Contains(t, string(logData), `"event":"gate_llm_error"`)
	assert.Contains(t, string(logData), `"event":"emit"`)
}
