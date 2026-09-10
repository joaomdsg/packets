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

func TestKill_closesTheOpenPRAndKillsThePacket(t *testing.T) {
	// t.Setenv (inside runFixture) forbids t.Parallel.
	pkt := &packet.Packet{Goal: "fix x", Terminal: []string{"true"}, Budget: packet.Budget{Retries: 3, Minutes: 30}}
	st := &state.State{Slug: "fix-x", Version: 1, State: state.Building, PRNumber: 42}
	fabricSlug := runFixture(t, "fix-x", st, pkt)

	gh := &stubCIGH{}
	root := cli.NewRoot(cli.WithCIDeps(ci.Deps{GH: gh, Git: &stubCIGit{}, Clock: &stubCIClock{}}))
	root.SetArgs([]string{"kill", "--fabric", fabricSlug, "fix-x"})

	err := root.Execute()

	require.NoError(t, err)
	assert.Equal(t, []int{42}, gh.closed)

	got, err := state.Load(filepath.Join(packetDirFor(t, fabricSlug, "fix-x"), "state.json"))
	require.NoError(t, err)
	assert.Equal(t, state.Killed, got.State)

	entries := readLogEntriesCLI(t, packetDirFor(t, fabricSlug, "fix-x"))
	assert.Equal(t, "kill", entries[len(entries)-1].Event)
	assert.Equal(t, "human", entries[len(entries)-1].Actor)
}

func TestKill_doesNotCloseAPRWhenNoneIsOpen(t *testing.T) {
	// t.Setenv (inside runFixture) forbids t.Parallel.
	pkt := &packet.Packet{Goal: "fix x", Terminal: []string{"true"}, Budget: packet.Budget{Retries: 3, Minutes: 30}}
	st := &state.State{Slug: "fix-x", Version: 1, State: state.Emitted}
	fabricSlug := runFixture(t, "fix-x", st, pkt)

	gh := &stubCIGH{}
	root := cli.NewRoot(cli.WithCIDeps(ci.Deps{GH: gh, Git: &stubCIGit{}, Clock: &stubCIClock{}}))
	root.SetArgs([]string{"kill", "--fabric", fabricSlug, "fix-x"})

	err := root.Execute()

	require.NoError(t, err)
	assert.Empty(t, gh.closed)
}

func TestKill_refusesAPacketAlreadyInAFinalState(t *testing.T) {
	// t.Setenv (inside runFixture) forbids t.Parallel.
	pkt := &packet.Packet{Goal: "fix x", Terminal: []string{"true"}, Budget: packet.Budget{Retries: 3, Minutes: 30}}
	st := &state.State{Slug: "fix-x", Version: 1, State: state.Killed}
	fabricSlug := runFixture(t, "fix-x", st, pkt)

	root := cli.NewRoot(cli.WithCIDeps(ci.Deps{GH: &stubCIGH{}, Git: &stubCIGit{}, Clock: &stubCIClock{}}))
	root.SetArgs([]string{"kill", "--fabric", fabricSlug, "fix-x"})

	err := root.Execute()

	assert.Error(t, err)
}

func TestKill_furtherMutatingCommandsRefuseAfterwards(t *testing.T) {
	// t.Setenv (inside runFixture) forbids t.Parallel.
	pkt := &packet.Packet{Goal: "fix x", Terminal: []string{"true"}, Budget: packet.Budget{Retries: 3, Minutes: 30}}
	st := &state.State{Slug: "fix-x", Version: 1, State: state.Building}
	fabricSlug := runFixture(t, "fix-x", st, pkt)

	killRoot := cli.NewRoot(cli.WithCIDeps(ci.Deps{GH: &stubCIGH{}, Git: &stubCIGit{}, Clock: &stubCIClock{}}))
	killRoot.SetArgs([]string{"kill", "--fabric", fabricSlug, "fix-x"})
	require.NoError(t, killRoot.Execute())

	resumeRoot := cli.NewRoot()
	resumeRoot.SetArgs([]string{"resume", "--fabric", fabricSlug, "fix-x"})
	assert.Error(t, resumeRoot.Execute())
}
