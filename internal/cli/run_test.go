package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/joaomdsg/packets/internal/build"
	"github.com/joaomdsg/packets/internal/ci"
	"github.com/joaomdsg/packets/internal/cli"
	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/journal"
	"github.com/joaomdsg/packets/internal/packet"
	"github.com/joaomdsg/packets/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runFixture registers a fabric plus one packet, writing fabric.yaml
// (which emitFixture doesn't need but run does, to find repo_path/image).
// CI/merge are configured so a packet that reaches node=ci can run the
// whole loop to completion without a real gh or network.
func runFixture(t *testing.T, slug string, st *state.State, pkt *packet.Packet) (fabricSlug string) {
	t.Helper()
	configDir := t.TempDir()
	dataDir := t.TempDir()
	repoDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("XDG_DATA_HOME", dataDir)

	const fabricSlugName = "test-fabric"
	require.NoError(t, os.MkdirAll(filepath.Join(configDir, "packets"), 0o700))
	require.NoError(t, fabric.SaveIndex(filepath.Join(configDir, "packets", "fabrics.yaml"), &fabric.Index{
		Fabrics: []fabric.IndexEntry{{Slug: fabricSlugName, RepoPath: repoDir}},
	}))

	fabricDir := filepath.Join(dataDir, "packets", fabricSlugName)
	require.NoError(t, os.MkdirAll(fabricDir, 0o700))
	require.NoError(t, fabric.Save(filepath.Join(fabricDir, "fabric.yaml"), &fabric.Config{
		Slug: fabricSlugName, RepoPath: repoDir, DefaultBranch: "main", Image: "packets/x:v0",
		CI:    fabric.CI{PollSeconds: 1, TimeoutMinutes: 5, WorkflowName: "packets"},
		Merge: fabric.Merge{Auto: true, Strategy: "squash"},
	}))

	packetDir := filepath.Join(fabricDir, "packets", slug)
	require.NoError(t, os.MkdirAll(packetDir, 0o700))
	require.NoError(t, packet.Save(filepath.Join(packetDir, "packet.yaml"), pkt))
	require.NoError(t, state.Save(filepath.Join(packetDir, "state.json"), st))

	return fabricSlugName
}

func packetDirFor(t *testing.T, fabricSlug, slug string) string {
	t.Helper()
	dataDir := os.Getenv("XDG_DATA_HOME")
	return filepath.Join(dataDir, "packets", fabricSlug, "packets", slug)
}

// repoDirFor reads repo_path back out of the fixture's registered index,
// for tests that need to chdir into it to exercise the cwd-default lookup.
func repoDirFor(t *testing.T, fabricSlug string) string {
	t.Helper()
	configDir := os.Getenv("XDG_CONFIG_HOME")
	idx, err := fabric.LoadIndex(filepath.Join(configDir, "packets", "fabrics.yaml"))
	require.NoError(t, err)
	entry, ok := idx.BySlug(fabricSlug)
	require.True(t, ok)
	return entry.RepoPath
}

// readLogEntriesCLI parses every line of a packet's log.jsonl, letting
// tests assert on more than just the event name (e.g. actor).
func readLogEntriesCLI(t *testing.T, packetDir string) []journal.Entry {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(packetDir, "log.jsonl"))
	require.NoError(t, err)
	var entries []journal.Entry
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var e journal.Entry
		require.NoError(t, json.Unmarshal([]byte(line), &e))
		entries = append(entries, e)
	}
	return entries
}

func readLogEventsCLI(t *testing.T, packetDir string) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(packetDir, "log.jsonl"))
	require.NoError(t, err)
	var events []string
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var e struct {
			Event string `json:"event"`
		}
		require.NoError(t, json.Unmarshal([]byte(line), &e))
		events = append(events, e.Event)
	}
	return events
}

func stubRunDeps() build.Deps {
	git := &stubCLIGit{committed: true}
	return build.Deps{
		Git:    git,
		GH:     &stubCLIGH{prNumber: 7},
		Docker: &stubCLIDocker{},
		Clock:  &stubCLIClock{},
		Getenv: func(key string) string {
			if key == "ANTHROPIC_API_KEY" {
				return "fake-key"
			}
			return ""
		},
		Exists: func(string) bool { return false },
	}
}

// stubRunCIDeps drives a packet at node=ci straight to a green,
// auto-merged termination.
func stubRunCIDeps() ci.Deps {
	return ci.Deps{
		GH:    &stubCIGH{checksSeq: [][]ci.Check{{{Name: "packets", State: "SUCCESS"}}}},
		Git:   &stubCIGit{currentCommit: "abc123"},
		Clock: &stubCIClock{},
	}
}

func TestRun_happyPathDrivesBuildThroughCIToTerminated(t *testing.T) {
	// t.Setenv (inside runFixture) forbids t.Parallel.
	pkt := &packet.Packet{Goal: "fix x", Terminal: []string{"true"}, Budget: packet.Budget{Retries: 3, Minutes: 30}}
	st := &state.State{Slug: "fix-x", Version: 1, State: state.Emitted}
	fabricSlug := runFixture(t, "fix-x", st, pkt)

	out := &bytes.Buffer{}
	root := cli.NewRoot(cli.WithBuildDeps(stubRunDeps()), cli.WithCIDeps(stubRunCIDeps()))
	root.SetOut(out)
	root.SetArgs([]string{"run", "--fabric", fabricSlug, "fix-x"})

	err := root.Execute()
	require.NoError(t, err)

	got, err := state.Load(filepath.Join(packetDirFor(t, fabricSlug, "fix-x"), "state.json"))
	require.NoError(t, err)
	assert.Equal(t, state.Terminated, got.State)
	assert.Nil(t, got.Node)
}

func TestRun_printsStateAndExitsForTerminalStates(t *testing.T) {
	tests := []string{state.Terminated, state.Killed, state.Halted, state.AwaitingApproval}
	for _, s := range tests {
		t.Run(s, func(t *testing.T) {
			// t.Setenv (inside runFixture) forbids t.Parallel.
			pkt := &packet.Packet{Goal: "fix x", Terminal: []string{"true"}, Budget: packet.Budget{Retries: 3, Minutes: 30}}
			st := &state.State{Slug: "fix-x", Version: 1, State: s}
			fabricSlug := runFixture(t, "fix-x", st, pkt)

			out := &bytes.Buffer{}
			root := cli.NewRoot(cli.WithBuildDeps(stubRunDeps()), cli.WithCIDeps(stubRunCIDeps()))
			root.SetOut(out)
			root.SetArgs([]string{"run", "--fabric", fabricSlug, "fix-x"})

			err := root.Execute()

			require.NoError(t, err)
			assert.Contains(t, out.String(), s)
		})
	}
}

func TestRun_resumesFromNodeCIWithoutReenteringBuild(t *testing.T) {
	// t.Setenv (inside runFixture) forbids t.Parallel.
	node := state.NodeCI
	pkt := &packet.Packet{Goal: "fix x", Terminal: []string{"true"}, Budget: packet.Budget{Retries: 3, Minutes: 30}}
	st := &state.State{Slug: "fix-x", Version: 1, State: state.CI, Node: &node, PRNumber: 7}
	fabricSlug := runFixture(t, "fix-x", st, pkt)

	deps := stubRunDeps()
	docker := deps.Docker.(*stubCLIDocker)
	root := cli.NewRoot(cli.WithBuildDeps(deps), cli.WithCIDeps(stubRunCIDeps()))
	root.SetArgs([]string{"run", "--fabric", fabricSlug, "fix-x"})

	err := root.Execute()

	require.NoError(t, err)
	assert.Zero(t, docker.calls, "must not spawn a container when already at node=ci")
}

func TestRun_rebaseConflictHaltsBeforeEverReachingCI(t *testing.T) {
	// t.Setenv (inside runFixture) forbids t.Parallel.
	pkt := &packet.Packet{Goal: "fix x", Terminal: []string{"true"}, Budget: packet.Budget{Retries: 3, Minutes: 30}}
	st := &state.State{Slug: "fix-x", Version: 1, State: state.Emitted}
	fabricSlug := runFixture(t, "fix-x", st, pkt)

	buildDeps := stubRunDeps()
	buildDeps.Git.(*stubCLIGit).rebaseConflict = true
	// A ci.GH that errors proves ci.Run is never reached.
	ciDeps := ci.Deps{GH: &stubCIGH{checksErr: assert.AnError}, Git: &stubCIGit{}, Clock: &stubCIClock{}}

	root := cli.NewRoot(cli.WithBuildDeps(buildDeps), cli.WithCIDeps(ciDeps))
	root.SetArgs([]string{"run", "--fabric", fabricSlug, "fix-x"})

	err := root.Execute()

	require.NoError(t, err)
	got, err := state.Load(filepath.Join(packetDirFor(t, fabricSlug, "fix-x"), "state.json"))
	require.NoError(t, err)
	assert.Equal(t, state.Halted, got.State)
	assert.Equal(t, "conflict", *got.HaltReason)
}

// stubCLIGit/stubCLIGH/stubCLIDocker/stubCLIClock mirror the build
// package's own doubles: run.go's job is wiring, so these just need to
// let a full container-to-PR pass complete without a real process.
type stubCLIGit struct {
	committed      bool
	rebaseConflict bool

	createdNewBranch   bool
	checkedOutExisting bool
}

func (g *stubCLIGit) Fetch(string) error { return nil }
func (g *stubCLIGit) CheckoutNewBranch(string, string, string) error {
	g.createdNewBranch = true
	return nil
}
func (g *stubCLIGit) Checkout(string, string) error {
	g.checkedOutExisting = true
	return nil
}
func (g *stubCLIGit) WorkingDiffNameStatus(string, string) (string, error) { return "", nil }
func (g *stubCLIGit) CommitAll(string, string) (bool, error)               { return g.committed, nil }
func (g *stubCLIGit) RebaseOnto(string, string) (bool, error)              { return g.rebaseConflict, nil }
func (g *stubCLIGit) RebaseAbort(string) error                             { return nil }
func (g *stubCLIGit) PushForceWithLease(string, string) error              { return nil }

type stubCLIGH struct{ prNumber int }

func (g *stubCLIGH) CreatePR(string, string, string, string, string) (int, error) {
	return g.prNumber, nil
}

type stubCLIDocker struct{ calls int }

func (d *stubCLIDocker) Run(args []string) (string, int, error) {
	d.calls++
	return `{"usage":{"input_tokens":1,"output_tokens":1}}`, 0, nil
}

type stubCLIClock struct{}

func (stubCLIClock) Now() time.Time { return time.Time{} }

// stubCIGH/stubCIGit/stubCIClock mirror the ci package's own doubles,
// used here only to drive a full run through the ci node.
type stubCIGH struct {
	checksSeq [][]ci.Check
	checksErr error
	pollCalls int
	prURL     string
	closed    []int
}

func (g *stubCIGH) PRChecks(dir string, pr int) ([]ci.Check, error) {
	if g.checksErr != nil {
		return nil, g.checksErr
	}
	if len(g.checksSeq) == 0 {
		return nil, nil
	}
	idx := g.pollCalls
	if idx >= len(g.checksSeq) {
		idx = len(g.checksSeq) - 1
	}
	g.pollCalls++
	return g.checksSeq[idx], nil
}

func (g *stubCIGH) RunViewLogFailed(dir, runID string) (string, error) { return "", nil }
func (g *stubCIGH) MergePR(dir string, pr int, strategy string) error  { return nil }
func (g *stubCIGH) PRURL(dir string, pr int) (string, error)           { return g.prURL, nil }
func (g *stubCIGH) ClosePR(dir string, pr int) error {
	g.closed = append(g.closed, pr)
	return nil
}

type stubCIGit struct{ currentCommit string }

func (g *stubCIGit) DiffStat(dir, from string) (string, error) { return "", nil }
func (g *stubCIGit) CurrentCommit(dir string) (string, error)  { return g.currentCommit, nil }
func (g *stubCIGit) DiffAddedFiles(dir, base, head string) ([]string, error) {
	return nil, nil
}

type stubCIClock struct{ now time.Time }

func (c *stubCIClock) Now() time.Time        { return c.now }
func (c *stubCIClock) Sleep(d time.Duration) { c.now = c.now.Add(d) }
