package cli_test

import (
	"path/filepath"
	"testing"

	"github.com/joaomdsg/packets/internal/ci"
	"github.com/joaomdsg/packets/internal/cli"
	"github.com/joaomdsg/packets/internal/packet"
	"github.com/joaomdsg/packets/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApprove_mergesAndTerminatesAPacketAwaitingApproval(t *testing.T) {
	// t.Setenv (inside runFixture) forbids t.Parallel.
	pkt := &packet.Packet{Goal: "fix x", Terminal: []string{"human_approval"}, Budget: packet.Budget{Retries: 3, Minutes: 30}}
	st := &state.State{Slug: "fix-x", Version: 1, State: state.AwaitingApproval, PRNumber: 7}
	fabricSlug := runFixture(t, "fix-x", st, pkt)

	gh := &stubCIGH{}
	root := cli.NewRoot(cli.WithCIDeps(ci.Deps{GH: gh, Git: &stubCIGit{currentCommit: "abc123"}, Clock: &stubCIClock{}}))
	root.SetArgs([]string{"approve", "--fabric", fabricSlug, "fix-x"})

	err := root.Execute()

	require.NoError(t, err)
	got, err := state.Load(filepath.Join(packetDirFor(t, fabricSlug, "fix-x"), "state.json"))
	require.NoError(t, err)
	assert.Equal(t, state.Terminated, got.State)
	assert.True(t, got.Approved)

	events := readLogEventsCLI(t, packetDirFor(t, fabricSlug, "fix-x"))
	assert.Contains(t, events, "approve")
	assert.Contains(t, events, "merge")
	assert.Contains(t, events, "terminate")
}

func TestApprove_refusesAPacketNotAwaitingApproval(t *testing.T) {
	// t.Setenv (inside runFixture) forbids t.Parallel.
	pkt := &packet.Packet{Goal: "fix x", Terminal: []string{"true"}, Budget: packet.Budget{Retries: 3, Minutes: 30}}
	st := &state.State{Slug: "fix-x", Version: 1, State: state.Building}
	fabricSlug := runFixture(t, "fix-x", st, pkt)

	root := cli.NewRoot(cli.WithCIDeps(ci.Deps{GH: &stubCIGH{}, Git: &stubCIGit{}, Clock: &stubCIClock{}}))
	root.SetArgs([]string{"approve", "--fabric", fabricSlug, "fix-x"})

	err := root.Execute()

	assert.Error(t, err)
}
