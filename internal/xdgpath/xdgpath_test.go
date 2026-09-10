package xdgpath_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/joaomdsg/packets/internal/xdgpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func statDir(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%04o", info.Mode().Perm()), nil
}

func TestConfigDir_honorsXDGConfigHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/custom/config")

	got, err := xdgpath.ConfigDir()

	require.NoError(t, err)
	assert.Equal(t, "/custom/config/packets", got)
}

func TestConfigDir_fallsBackToHomeConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "/home/tester")

	got, err := xdgpath.ConfigDir()

	require.NoError(t, err)
	assert.Equal(t, "/home/tester/.config/packets", got)
}

func TestDataDir_honorsXDGDataHome(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/custom/data")

	got, err := xdgpath.DataDir()

	require.NoError(t, err)
	assert.Equal(t, "/custom/data/packets", got)
}

func TestDataDir_fallsBackToHomeLocalShare(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("HOME", "/home/tester")

	got, err := xdgpath.DataDir()

	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/home/tester", ".local", "share", "packets"), got)
}

func TestEnsureDir_createsDirWithRestrictedMode(t *testing.T) {
	tmp := t.TempDir()
	target := filepath.Join(tmp, "nested", "dir")

	err := xdgpath.EnsureDir(target)

	require.NoError(t, err)
	info, statErr := statDir(target)
	require.NoError(t, statErr)
	assert.Equal(t, "0700", info)
}
