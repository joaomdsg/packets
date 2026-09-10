package cli_test

import (
	"bytes"
	"testing"

	"github.com/joaomdsg/packets/internal/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoot_printsBuildVersionWhenVersionFlagSet(t *testing.T) {
	t.Parallel()

	out := &bytes.Buffer{}
	root := cli.NewRoot()
	root.SetOut(out)
	root.SetArgs([]string{"--version"})

	err := root.Execute()

	require.NoError(t, err)
	assert.Contains(t, out.String(), "packets")
	assert.Regexp(t, `\d+\.\d+`, out.String())
}

func TestRoot_printsVersionWhenRequiredToolsAreMissing(t *testing.T) {
	// t.Setenv forbids t.Parallel. An empty PATH makes every deps.Check
	// lookup fail, so this fails unless --version short-circuits the
	// persistent pre-run.
	t.Setenv("PATH", "")

	out := &bytes.Buffer{}
	root := cli.NewRoot()
	root.SetOut(out)
	root.SetArgs([]string{"--version"})

	err := root.Execute()

	require.NoError(t, err)
	assert.Regexp(t, `\d+\.\d+`, out.String())
}
