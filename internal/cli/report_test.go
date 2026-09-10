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

func TestReport_printsAPlainTextTableForTheResolvedFabric(t *testing.T) {
	// t.Setenv (inside runFixture) forbids t.Parallel.
	pkt := &packet.Packet{Goal: "fix x", Terminal: []string{"true"}, Budget: packet.Budget{Retries: 3, Minutes: 30}}
	st := &state.State{Slug: "fix-x", Version: 1, State: state.Building}
	fabricSlug := runFixture(t, "fix-x", st, pkt)

	out := &bytes.Buffer{}
	root := cli.NewRoot()
	root.SetOut(out)
	root.SetArgs([]string{"report", "--fabric", fabricSlug})

	err := root.Execute()

	require.NoError(t, err)
	assert.Contains(t, out.String(), "GATE")
	assert.Contains(t, out.String(), "TERMINATION")
	assert.Contains(t, out.String(), "ACCRETION")
	assert.Contains(t, out.String(), "AMEND")
	assert.Contains(t, out.String(), "HALTS")
}

func TestReport_succeedsAndWarnsOnATornLogLine(t *testing.T) {
	// t.Setenv (inside runFixture) forbids t.Parallel. A single malformed
	// line is exactly what a crash mid-append leaves behind; report must
	// still compute every metric from the lines that did parse.
	pkt := &packet.Packet{Goal: "fix x", Terminal: []string{"true"}, Budget: packet.Budget{Retries: 3, Minutes: 30}}
	st := &state.State{Slug: "fix-x", Version: 1, State: state.Building}
	fabricSlug := runFixture(t, "fix-x", st, pkt)

	packetDir := packetDirFor(t, fabricSlug, "fix-x")
	logPath := filepath.Join(packetDir, "log.jsonl")
	f, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	require.NoError(t, err)
	_, err = f.WriteString("not json\n")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	root := cli.NewRoot()
	root.SetOut(out)
	root.SetErr(errOut)
	root.SetArgs([]string{"report", "--fabric", fabricSlug})

	err = root.Execute()

	require.NoError(t, err)
	assert.Contains(t, out.String(), "GATE")
	assert.Contains(t, errOut.String(), "skipped")
	assert.Contains(t, errOut.String(), logPath)
}
