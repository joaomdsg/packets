package cli_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joaomdsg/packets/internal/cli"
	"github.com/joaomdsg/packets/internal/packet"
	"github.com/joaomdsg/packets/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rewriteEditor is a hand-rolled Editor stub: instead of launching a real
// editor, it overwrites the file with fixed contents, standing in for a
// human's save.
type rewriteEditor struct{ contents string }

func (e *rewriteEditor) Open(path string) error {
	return os.WriteFile(path, []byte(e.contents), 0o600)
}

func TestAmend_bumpsVersionSnapshotsResetsRetriesAndLogsSummary(t *testing.T) {
	// t.Setenv (inside runFixture) forbids t.Parallel.
	reason := "budget"
	pkt := &packet.Packet{
		Goal: "fix x", Terminal: []string{"true"},
		Budget: packet.Budget{Retries: 3, Minutes: 30},
	}
	st := &state.State{Slug: "fix-x", Version: 1, State: state.Halted, HaltReason: &reason, RetriesUsed: 3, Branch: "packets/fix-x-v1"}
	fabricSlug := runFixture(t, "fix-x", st, pkt)

	editor := &rewriteEditor{contents: "goal: fix x better\nterminal:\n  - \"true\"\nbudget:\n  retries: 3\n  minutes: 30\n"}
	root := cli.NewRoot(cli.WithGateLLM(&queueLLM{}), cli.WithEditor(editor))
	root.SetArgs([]string{"amend", "--fabric", fabricSlug, "fix-x"})

	err := root.Execute()

	require.NoError(t, err)
	packetDir := packetDirFor(t, fabricSlug, "fix-x")

	gotState, err := state.Load(filepath.Join(packetDir, "state.json"))
	require.NoError(t, err)
	assert.Equal(t, 2, gotState.Version)
	assert.Equal(t, state.Emitted, gotState.State)
	assert.Zero(t, gotState.RetriesUsed)
	assert.Nil(t, gotState.HaltReason)
	assert.Equal(t, "packets/fix-x-v1", gotState.Branch, "amend must not clear the branch: build restarts on the previous branch")

	_, err = os.Stat(filepath.Join(packetDir, "versions", "v2.yaml"))
	assert.NoError(t, err, "amend must snapshot the new version")

	gotPkt, err := packet.Load(filepath.Join(packetDir, "packet.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "fix x better", gotPkt.Goal)

	entries := readLogEntriesCLI(t, packetDir)
	last := entries[len(entries)-1]
	assert.Equal(t, "amend", last.Event)
	assert.Equal(t, "human", last.Actor)
	assert.Equal(t, "-goal: fix x", last.Detail["summary"])
}

func TestAmend_refusesAPacketThatIsNotHaltedOrAwaitingApproval(t *testing.T) {
	// t.Setenv (inside runFixture) forbids t.Parallel.
	pkt := &packet.Packet{Goal: "fix x", Terminal: []string{"true"}, Budget: packet.Budget{Retries: 3, Minutes: 30}}
	st := &state.State{Slug: "fix-x", Version: 1, State: state.Building}
	fabricSlug := runFixture(t, "fix-x", st, pkt)

	editor := &rewriteEditor{contents: "goal: fix x\nterminal:\n  - \"true\"\nbudget:\n  retries: 3\n  minutes: 30\n"}
	root := cli.NewRoot(cli.WithGateLLM(&queueLLM{}), cli.WithEditor(editor))
	root.SetArgs([]string{"amend", "--fabric", fabricSlug, "fix-x"})

	err := root.Execute()

	assert.Error(t, err)
}

func TestAmend_restartsAtBuildWithThePreviousBranchCheckedOut(t *testing.T) {
	// t.Setenv (inside runFixture) forbids t.Parallel.
	pkt := &packet.Packet{
		Goal: "fix x", Terminal: []string{"true"},
		Budget: packet.Budget{Retries: 3, Minutes: 30},
	}
	st := &state.State{Slug: "fix-x", Version: 1, State: state.Halted, Branch: "packets/fix-x-v1", PRNumber: 7}
	fabricSlug := runFixture(t, "fix-x", st, pkt)

	editor := &rewriteEditor{contents: "goal: fix x better\nterminal:\n  - \"true\"\nbudget:\n  retries: 3\n  minutes: 30\n"}
	amendRoot := cli.NewRoot(cli.WithGateLLM(&queueLLM{}), cli.WithEditor(editor))
	amendRoot.SetArgs([]string{"amend", "--fabric", fabricSlug, "fix-x"})
	require.NoError(t, amendRoot.Execute())

	buildDeps := stubRunDeps()
	git := buildDeps.Git.(*stubCLIGit)
	runRoot := cli.NewRoot(cli.WithBuildDeps(buildDeps), cli.WithCIDeps(stubRunCIDeps()))
	runRoot.SetArgs([]string{"run", "--fabric", fabricSlug, "fix-x"})
	require.NoError(t, runRoot.Execute())

	assert.True(t, git.checkedOutExisting, "run must check out the amended packet's previous branch, not create a new one")
	assert.False(t, git.createdNewBranch)
}
