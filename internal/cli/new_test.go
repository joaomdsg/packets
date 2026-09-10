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

func TestNew_refusesToOverwriteAnExistingPacketYAML(t *testing.T) {
	dir := t.TempDir()
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { require.NoError(t, os.Chdir(cwd)) })

	const existing = "goal: \"hand-authored draft\"\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "packet.yaml"), []byte(existing), 0o600))

	root := cli.NewRoot()
	root.SetArgs([]string{"new"})

	err = root.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
	got, readErr := os.ReadFile(filepath.Join(dir, "packet.yaml"))
	require.NoError(t, readErr)
	assert.Equal(t, existing, string(got))
}

func TestNew_overwritesAnExistingPacketYAMLWhenForced(t *testing.T) {
	dir := t.TempDir()
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { require.NoError(t, os.Chdir(cwd)) })

	require.NoError(t, os.WriteFile(filepath.Join(dir, "packet.yaml"), []byte("goal: \"old\"\n"), 0o600))

	out := &bytes.Buffer{}
	root := cli.NewRoot()
	root.SetOut(out)
	root.SetArgs([]string{"new", "--force"})

	err = root.Execute()

	require.NoError(t, err)
	assert.Contains(t, out.String(), "packet.yaml")
	got, readErr := os.ReadFile(filepath.Join(dir, "packet.yaml"))
	require.NoError(t, readErr)
	assert.NotContains(t, string(got), "old")
}
