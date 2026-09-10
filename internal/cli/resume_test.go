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

func TestResume_movesHaltedToBuildingAndClearsHaltReason(t *testing.T) {
	// t.Setenv (inside runFixture) forbids t.Parallel.
	reason := "conflict"
	pkt := &packet.Packet{Goal: "fix x", Terminal: []string{"true"}, Budget: packet.Budget{Retries: 3, Minutes: 30}}
	st := &state.State{Slug: "fix-x", Version: 1, State: state.Halted, HaltReason: &reason}
	fabricSlug := runFixture(t, "fix-x", st, pkt)

	root := cli.NewRoot()
	root.SetArgs([]string{"resume", "--fabric", fabricSlug, "fix-x"})

	err := root.Execute()

	require.NoError(t, err)
	got, err := state.Load(filepath.Join(packetDirFor(t, fabricSlug, "fix-x"), "state.json"))
	require.NoError(t, err)
	assert.Equal(t, state.Building, got.State)
	assert.Nil(t, got.HaltReason)

	entries := readLogEntriesCLI(t, packetDirFor(t, fabricSlug, "fix-x"))
	assert.Equal(t, "resume", entries[len(entries)-1].Event)
	assert.Equal(t, "human", entries[len(entries)-1].Actor)
}

func TestResume_refusesAPacketNotHalted(t *testing.T) {
	// t.Setenv (inside runFixture) forbids t.Parallel.
	pkt := &packet.Packet{Goal: "fix x", Terminal: []string{"true"}, Budget: packet.Budget{Retries: 3, Minutes: 30}}
	st := &state.State{Slug: "fix-x", Version: 1, State: state.Building}
	fabricSlug := runFixture(t, "fix-x", st, pkt)

	root := cli.NewRoot()
	root.SetArgs([]string{"resume", "--fabric", fabricSlug, "fix-x"})

	err := root.Execute()

	assert.Error(t, err)
}
