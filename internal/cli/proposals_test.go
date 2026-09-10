package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/joaomdsg/packets/internal/cli"
	"github.com/joaomdsg/packets/internal/packet"
	"github.com/joaomdsg/packets/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProposals_listsProposalFilesAndLogs(t *testing.T) {
	// t.Setenv (inside runFixture) forbids t.Parallel.
	pkt := &packet.Packet{Goal: "fix x", Terminal: []string{"true"}, Budget: packet.Budget{Retries: 3, Minutes: 30}}
	st := &state.State{Slug: "fix-x", Version: 1, State: state.Building}
	fabricSlug := runFixture(t, "fix-x", st, pkt)

	packetDir := packetDirFor(t, fabricSlug, "fix-x")
	require.NoError(t, os.MkdirAll(filepath.Join(packetDir, "proposals"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(packetDir, "proposals", "1.md"), []byte("Add a lint rule\nrationale here\n"), 0o600))

	out := &bytes.Buffer{}
	root := cli.NewRoot()
	root.SetOut(out)
	root.SetArgs([]string{"proposals", "--fabric", fabricSlug, "fix-x"})

	err := root.Execute()

	require.NoError(t, err)
	assert.Contains(t, out.String(), "1.md")

	entries := readLogEntriesCLI(t, packetDir)
	assert.Equal(t, "proposals", entries[len(entries)-1].Event)
	assert.Equal(t, "human", entries[len(entries)-1].Actor)
}
