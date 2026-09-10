package cli_test

import (
	"bytes"
	"testing"

	"github.com/joaomdsg/packets/internal/cli"
	"github.com/joaomdsg/packets/internal/packet"
	"github.com/joaomdsg/packets/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatus_printsStateFieldsAndLogsWithHumanActor(t *testing.T) {
	// t.Setenv (inside runFixture) forbids t.Parallel.
	pkt := &packet.Packet{Goal: "fix x", Terminal: []string{"true"}, Budget: packet.Budget{Retries: 3, Minutes: 30}}
	st := &state.State{Slug: "fix-x", Version: 2, State: state.Building, Branch: "packets/fix-x-v2"}
	fabricSlug := runFixture(t, "fix-x", st, pkt)

	out := &bytes.Buffer{}
	root := cli.NewRoot()
	root.SetOut(out)
	root.SetArgs([]string{"status", "--fabric", fabricSlug, "fix-x"})

	err := root.Execute()

	require.NoError(t, err)
	assert.Contains(t, out.String(), "state: building")
	assert.Contains(t, out.String(), "branch: packets/fix-x-v2")

	entries := readLogEntriesCLI(t, packetDirFor(t, fabricSlug, "fix-x"))
	require.Len(t, entries, 1)
	assert.Equal(t, "status", entries[0].Event)
	assert.Equal(t, "human", entries[0].Actor)
}

func TestStatus_reportsErrorWhenDirSlugDoesNotMatchStateSlug(t *testing.T) {
	// t.Setenv (inside runFixture) forbids t.Parallel. The packet dir is
	// "fake" but its state.json claims slug "other" — a desync that must
	// be reported rather than trusted.
	pkt := &packet.Packet{Goal: "fix x", Terminal: []string{"true"}, Budget: packet.Budget{Retries: 3, Minutes: 30}}
	st := &state.State{Slug: "other", Version: 1, State: state.Building}
	fabricSlug := runFixture(t, "fake", st, pkt)

	root := cli.NewRoot()
	root.SetArgs([]string{"status", "--fabric", fabricSlug, "fake"})

	err := root.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fake")
	assert.Contains(t, err.Error(), "other")
}
