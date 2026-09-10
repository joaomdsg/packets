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

func TestList_printsOneLinePerPacketAndLogsToFabricLog(t *testing.T) {
	// t.Setenv (inside runFixture) forbids t.Parallel.
	pkt := &packet.Packet{Goal: "fix x", Terminal: []string{"true"}, Budget: packet.Budget{Retries: 3, Minutes: 30}}
	st := &state.State{Slug: "fix-x", Version: 1, State: state.Building}
	fabricSlug := runFixture(t, "fix-x", st, pkt)

	out := &bytes.Buffer{}
	root := cli.NewRoot()
	root.SetOut(out)
	root.SetArgs([]string{"list", "--fabric", fabricSlug})

	err := root.Execute()

	require.NoError(t, err)
	assert.Contains(t, out.String(), "fix-x")
	assert.Contains(t, out.String(), "building")

	fabricDir := filepath.Dir(packetDirFor(t, fabricSlug, "fix-x"))
	fabricDir = filepath.Dir(fabricDir) // packets/<slug> -> fabric data dir
	data, err := os.ReadFile(filepath.Join(fabricDir, "log.jsonl"))
	require.NoError(t, err)
	assert.Contains(t, string(data), `"event":"list"`)
	assert.Contains(t, string(data), `"actor":"human"`)
}

func TestList_reportsErrorWithItsOwnNameWhenNoFabricRegistered(t *testing.T) {
	// t.Setenv forbids t.Parallel.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	root := cli.NewRoot()
	root.SetArgs([]string{"list"})

	err := root.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "list:")
	assert.NotContains(t, err.Error(), "emit:")
}

func TestList_errorsOnCorruptFabricConfig(t *testing.T) {
	// t.Setenv (inside runFixture) forbids t.Parallel.
	pkt := &packet.Packet{Goal: "fix x", Terminal: []string{"true"}, Budget: packet.Budget{Retries: 3, Minutes: 30}}
	st := &state.State{Slug: "fix-x", Version: 1, State: state.Building}
	fabricSlug := runFixture(t, "fix-x", st, pkt)

	fabricDir := filepath.Dir(filepath.Dir(packetDirFor(t, fabricSlug, "fix-x")))
	require.NoError(t, os.WriteFile(filepath.Join(fabricDir, "fabric.yaml"), []byte("not: [valid: yaml"), 0o600))

	root := cli.NewRoot()
	root.SetArgs([]string{"list", "--fabric", fabricSlug})

	err := root.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "list:")
}

func TestList_rejectsEmptyFabricFlag(t *testing.T) {
	// t.Setenv (inside runFixture) forbids t.Parallel. cwd is inside the
	// fixture's registered repo_path, so an empty --fabric must not be
	// treated as "not passed" and silently fall through to that default.
	pkt := &packet.Packet{Goal: "fix x", Terminal: []string{"true"}, Budget: packet.Budget{Retries: 3, Minutes: 30}}
	st := &state.State{Slug: "fix-x", Version: 1, State: state.Building}
	fabricSlug := runFixture(t, "fix-x", st, pkt)
	repoDir := repoDirFor(t, fabricSlug)

	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(repoDir))
	t.Cleanup(func() { require.NoError(t, os.Chdir(cwd)) })

	root := cli.NewRoot()
	root.SetArgs([]string{"list", "--fabric", ""})

	err = root.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "list:")
}

func TestList_reportsCorruptPacketButStillListsHealthyOnes(t *testing.T) {
	// t.Setenv (inside runFixture) forbids t.Parallel.
	pkt := &packet.Packet{Goal: "fix x", Terminal: []string{"true"}, Budget: packet.Budget{Retries: 3, Minutes: 30}}
	st := &state.State{Slug: "fix-x", Version: 1, State: state.Building}
	fabricSlug := runFixture(t, "fix-x", st, pkt)

	fabricDir := filepath.Dir(filepath.Dir(packetDirFor(t, fabricSlug, "fix-x")))
	corruptDir := filepath.Join(fabricDir, "packets", "broken")
	require.NoError(t, os.MkdirAll(corruptDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(corruptDir, "state.json"), []byte("{not json"), 0o600))

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	root := cli.NewRoot()
	root.SetOut(out)
	root.SetErr(errOut)
	root.SetArgs([]string{"list", "--fabric", fabricSlug})

	err := root.Execute()

	assert.Error(t, err)
	assert.Contains(t, out.String(), "fix-x")
	assert.Contains(t, errOut.String(), "broken")
}
