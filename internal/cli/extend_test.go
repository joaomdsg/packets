package cli_test

import (
	"path/filepath"
	"testing"

	"github.com/joaomdsg/packets/internal/cli"
	"github.com/joaomdsg/packets/internal/packet"
	"github.com/joaomdsg/packets/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtend_addsRetriesAndResumesWhenHaltedOnBudget(t *testing.T) {
	// t.Setenv (inside runFixture) forbids t.Parallel.
	reason := "budget"
	pkt := &packet.Packet{Goal: "fix x", Terminal: []string{"true"}, Budget: packet.Budget{Retries: 3, Minutes: 30}}
	st := &state.State{Slug: "fix-x", Version: 1, State: state.Halted, HaltReason: &reason, RetriesUsed: 3}
	fabricSlug := runFixture(t, "fix-x", st, pkt)

	root := cli.NewRoot()
	root.SetArgs([]string{"extend", "--fabric", fabricSlug, "fix-x", "--retries", "2"})

	err := root.Execute()

	require.NoError(t, err)
	packetDir := packetDirFor(t, fabricSlug, "fix-x")
	gotPkt, err := packet.Load(filepath.Join(packetDir, "packet.yaml"))
	require.NoError(t, err)
	assert.Equal(t, 5, gotPkt.Budget.Retries)

	gotState, err := state.Load(filepath.Join(packetDir, "state.json"))
	require.NoError(t, err)
	assert.Equal(t, state.Building, gotState.State)
	assert.Nil(t, gotState.HaltReason)

	entries := readLogEntriesCLI(t, packetDir)
	assert.Equal(t, "extend", entries[len(entries)-1].Event)
	assert.Equal(t, "human", entries[len(entries)-1].Actor)
}

func TestExtend_refusesWhenHaltReasonIsNotBudget(t *testing.T) {
	// t.Setenv (inside runFixture) forbids t.Parallel.
	reason := "conflict"
	pkt := &packet.Packet{Goal: "fix x", Terminal: []string{"true"}, Budget: packet.Budget{Retries: 3, Minutes: 30}}
	st := &state.State{Slug: "fix-x", Version: 1, State: state.Halted, HaltReason: &reason}
	fabricSlug := runFixture(t, "fix-x", st, pkt)

	root := cli.NewRoot()
	root.SetArgs([]string{"extend", "--fabric", fabricSlug, "fix-x", "--retries", "2"})

	err := root.Execute()

	assert.Error(t, err)
}

func TestExtend_refusesWhenNotHalted(t *testing.T) {
	// t.Setenv (inside runFixture) forbids t.Parallel.
	pkt := &packet.Packet{Goal: "fix x", Terminal: []string{"true"}, Budget: packet.Budget{Retries: 3, Minutes: 30}}
	st := &state.State{Slug: "fix-x", Version: 1, State: state.Building}
	fabricSlug := runFixture(t, "fix-x", st, pkt)

	root := cli.NewRoot()
	root.SetArgs([]string{"extend", "--fabric", fabricSlug, "fix-x", "--retries", "2"})

	err := root.Execute()

	assert.Error(t, err)
}
