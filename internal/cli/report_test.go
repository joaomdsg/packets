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
