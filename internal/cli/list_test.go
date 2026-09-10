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
