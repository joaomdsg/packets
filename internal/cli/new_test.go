package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/joaomdsg/packets/internal/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_writesTemplateAndPrintsPath(t *testing.T) {
	dir := t.TempDir()
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { require.NoError(t, os.Chdir(cwd)) })

	out := &bytes.Buffer{}
	root := cli.NewRoot()
	root.SetOut(out)
	root.SetArgs([]string{"new"})

	require.NoError(t, root.Execute())

	assert.Contains(t, out.String(), "packet.yaml")
	_, err = os.Stat(filepath.Join(dir, "packet.yaml"))
	assert.NoError(t, err)
}
