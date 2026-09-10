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

func TestApproveProposal_writesDraftPacketAndRegistryEntry(t *testing.T) {
	// t.Setenv (inside runFixture) forbids t.Parallel.
	pkt := &packet.Packet{Goal: "fix x", Terminal: []string{"true"}, Budget: packet.Budget{Retries: 3, Minutes: 30}}
	st := &state.State{Slug: "fix-x", Version: 1, State: state.Building}
	fabricSlug := runFixture(t, "fix-x", st, pkt)

	packetDir := packetDirFor(t, fabricSlug, "fix-x")
	require.NoError(t, os.MkdirAll(filepath.Join(packetDir, "proposals"), 0o700))
	require.NoError(t, os.WriteFile(
		filepath.Join(packetDir, "proposals", "1.md"),
		[]byte("Add a lint rule\nThe repo has no linter configured.\n"), 0o600))

	dir := t.TempDir()
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { require.NoError(t, os.Chdir(cwd)) })

	out := &bytes.Buffer{}
	root := cli.NewRoot()
	root.SetOut(out)
	root.SetArgs([]string{"approve-proposal", "--fabric", fabricSlug, "fix-x", "1"})

	err = root.Execute()

	require.NoError(t, err)
	assert.Contains(t, out.String(), "packet.yaml")

	got, err := packet.Load(filepath.Join(dir, "packet.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "Add a lint rule", got.Goal)
	assert.Equal(t, "The repo has no linter configured.", got.Context)
	require.NotNil(t, got.CausedBy)
	assert.Equal(t, "fix-x", *got.CausedBy)

	fabricDir := filepath.Dir(filepath.Dir(packetDir)) // packets/<slug> -> fabric data dir
	registryData, err := os.ReadFile(filepath.Join(fabricDir, "registry.jsonl"))
	require.NoError(t, err)
	assert.Contains(t, string(registryData), `"cause":"proposal"`)
	assert.Contains(t, string(registryData), `"packet":"fix-x"`)

	entries := readLogEntriesCLI(t, packetDir)
	assert.Equal(t, "proposal", entries[len(entries)-1].Event)
	assert.Equal(t, "human", entries[len(entries)-1].Actor)
}

func TestApproveProposal_rejectsAnEmptyProposalFile(t *testing.T) {
	// t.Setenv (inside runFixture) forbids t.Parallel.
	pkt := &packet.Packet{Goal: "fix x", Terminal: []string{"true"}, Budget: packet.Budget{Retries: 3, Minutes: 30}}
	st := &state.State{Slug: "fix-x", Version: 1, State: state.Building}
	fabricSlug := runFixture(t, "fix-x", st, pkt)

	packetDir := packetDirFor(t, fabricSlug, "fix-x")
	require.NoError(t, os.MkdirAll(filepath.Join(packetDir, "proposals"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(packetDir, "proposals", "1.md"), nil, 0o600))

	root := cli.NewRoot()
	root.SetArgs([]string{"approve-proposal", "--fabric", fabricSlug, "fix-x", "1"})

	err := root.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "approve-proposal:")
	assert.Contains(t, err.Error(), "1.md")
}

func TestApproveProposal_rejectsNegativeIndexWithAClearError(t *testing.T) {
	// t.Setenv (inside runFixture) forbids t.Parallel. Cobra treats a
	// leading "-" as a shorthand flag unless the command stops scanning
	// for flags after the first positional; this proves "-1" reaches
	// validation instead of dying in flag parsing.
	pkt := &packet.Packet{Goal: "fix x", Terminal: []string{"true"}, Budget: packet.Budget{Retries: 3, Minutes: 30}}
	st := &state.State{Slug: "fix-x", Version: 1, State: state.Building}
	fabricSlug := runFixture(t, "fix-x", st, pkt)

	root := cli.NewRoot()
	root.SetArgs([]string{"approve-proposal", "--fabric", fabricSlug, "fix-x", "-1"})

	err := root.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "approve-proposal:")
	assert.Contains(t, err.Error(), "-1")
	assert.NotContains(t, err.Error(), "unknown shorthand flag")
}
