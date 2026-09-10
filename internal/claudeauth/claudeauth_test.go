package claudeauth_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joaomdsg/packets/internal/claudeauth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func noEnv(string) string { return "" }
func noFile(string) bool  { return false }
func envMap(m map[string]string) func(string) string {
	return func(key string) string { return m[key] }
}

func TestResolve_prefersExplicitPathWhenConfigured(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "creds.json")
	require.NoError(t, os.WriteFile(path, []byte("placeholder"), 0o600))

	args, err := claudeauth.Resolve(path, envMap(map[string]string{"ANTHROPIC_API_KEY": "fake"}),
		func(p string) bool { return p == path })

	require.NoError(t, err)
	assert.Contains(t, args, "--tmpfs")
	assert.Contains(t, args, claudeauth.TmpfsMount)
	assert.Contains(t, args, path+":"+claudeauth.ConfigDir+"/.credentials.json:ro")
}

func TestResolve_errorsWhenExplicitPathConfiguredButMissing(t *testing.T) {
	t.Parallel()

	_, err := claudeauth.Resolve("/does/not/exist.json", noEnv, noFile)

	assert.ErrorContains(t, err, "/does/not/exist.json")
}

func TestResolve_mountsCredentialsFileFromClaudeConfigDirEnv(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	credPath := filepath.Join(dir, ".credentials.json")
	require.NoError(t, os.WriteFile(credPath, []byte("placeholder"), 0o600))

	args, err := claudeauth.Resolve("", envMap(map[string]string{"CLAUDE_CONFIG_DIR": dir}),
		func(p string) bool { return p == credPath })

	require.NoError(t, err)
	assert.Contains(t, args, credPath+":"+claudeauth.ConfigDir+"/.credentials.json:ro")
}

func TestResolve_mountsCredentialsFileFromHomeClaudeDirWhenConfigDirUnset(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	credPath := filepath.Join(home, ".claude", ".credentials.json")

	args, err := claudeauth.Resolve("", envMap(map[string]string{"HOME": home}),
		func(p string) bool { return p == credPath })

	require.NoError(t, err)
	assert.Contains(t, args, credPath+":"+claudeauth.ConfigDir+"/.credentials.json:ro")
}

func TestResolve_fallsBackToAPIKeyWhenNoCredentialsFileExists(t *testing.T) {
	t.Parallel()

	args, err := claudeauth.Resolve("", envMap(map[string]string{"ANTHROPIC_API_KEY": "fake"}), noFile)

	require.NoError(t, err)
	assert.Equal(t, []string{"--tmpfs", claudeauth.TmpfsMount, "-e", "ANTHROPIC_API_KEY"}, args)
}

func TestResolve_errorsNamingBothOptionsWhenNeitherCredentialExists(t *testing.T) {
	t.Parallel()

	_, err := claudeauth.Resolve("", noEnv, noFile)

	assert.ErrorContains(t, err, "credentials_path")
	assert.ErrorContains(t, err, "ANTHROPIC_API_KEY")
}
