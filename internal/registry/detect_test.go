package registry_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joaomdsg/packets/internal/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectGateSuggested_findsTokensThatExistAsRepoPaths(t *testing.T) {
	t.Parallel()
	repoPath := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(repoPath, "test"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(repoPath, "test", "login.timeout.spec.js"), []byte("x"), 0o600))

	found := registry.DetectGateSuggested(repoPath, []string{"npm run check test/login.timeout.spec.js"})

	assert.Equal(t, []string{"test/login.timeout.spec.js"}, found)
}

func TestDetectGateSuggested_ignoresTokensThatDoNotExist(t *testing.T) {
	t.Parallel()
	repoPath := t.TempDir()

	found := registry.DetectGateSuggested(repoPath, []string{"npm run something-else"})

	assert.Empty(t, found)
}

type stubGit struct {
	added []string
	err   error
}

func (s *stubGit) DiffAddedFiles(dir, base, head string) ([]string, error) {
	return s.added, s.err
}

func TestDetectCIFailure_findsNewTestDirFilesNotAlreadyGateSuggested(t *testing.T) {
	t.Parallel()
	g := &stubGit{added: []string{"test/new_test.go", "src/main.go", "spec/other_spec.rb"}}

	found, err := registry.DetectCIFailure(g, "/repo", "origin/main", "packets/x-v1", []string{"spec/other_spec.rb"})

	require.NoError(t, err)
	assert.Equal(t, []string{"test/new_test.go"}, found)
}

func TestDetectCIFailure_propagatesGitError(t *testing.T) {
	t.Parallel()
	g := &stubGit{err: assert.AnError}

	_, err := registry.DetectCIFailure(g, "/repo", "origin/main", "packets/x-v1", nil)

	assert.Error(t, err)
}
