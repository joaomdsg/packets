package cli_test

import (
	"testing"

	"github.com/joaomdsg/packets/internal/cli"
	"github.com/stretchr/testify/assert"
)

func TestRoot_refusesToRunWithARequiredBinaryMissingFromPATH(t *testing.T) {
	// t.Setenv mutates the process environment, forbidding t.Parallel.
	t.Setenv("PATH", t.TempDir())

	root := cli.NewRoot()
	root.SetArgs([]string{"new"})

	err := root.Execute()

	assert.ErrorContains(t, err, "required tool(s) not found on PATH")
	assert.ErrorContains(t, err, "git")
	assert.ErrorContains(t, err, "gh")
	assert.ErrorContains(t, err, "docker")
}
