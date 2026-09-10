package bootstrap_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joaomdsg/packets/internal/bootstrap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func touch(t *testing.T, dir, name string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("{}"), 0o600))
}

func TestDetectToolchains_matchesInPriorityOrder(t *testing.T) {
	tests := []struct {
		name    string
		markers []string
		want    []string
	}{
		{"go only", []string{"go.mod"}, []string{"go"}},
		{"node only", []string{"package.json"}, []string{"node"}},
		{"pyproject only", []string{"pyproject.toml"}, []string{"python"}},
		{"requirements only", []string{"requirements.txt"}, []string{"python"}},
		{"pyproject and requirements is one ecosystem", []string{"pyproject.toml", "requirements.txt"}, []string{"python"}},
		{"go and node both present", []string{"go.mod", "package.json"}, []string{"go", "node"}},
		{"none present", nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for _, m := range tt.markers {
				touch(t, dir, m)
			}

			found, err := bootstrap.DetectToolchains(dir)

			require.NoError(t, err)
			var names []string
			for _, tc := range found {
				names = append(names, tc.Name)
			}
			assert.Equal(t, tt.want, names)
		})
	}
}

func TestPromptToolchain_returnsChosenCandidate(t *testing.T) {
	t.Parallel()
	candidates := []bootstrap.Toolchain{
		{Name: "go", Base: "golang:1.22"},
		{Name: "node", Base: "node:20"},
	}
	r := strings.NewReader("2\n")
	w := &strings.Builder{}

	got, err := bootstrap.PromptToolchain(r, w, candidates)

	require.NoError(t, err)
	assert.Equal(t, "node", got.Name)
	assert.Contains(t, w.String(), "go")
	assert.Contains(t, w.String(), "node")
}

func TestPromptToolchain_errorsOnInvalidSelection(t *testing.T) {
	t.Parallel()
	candidates := []bootstrap.Toolchain{{Name: "go", Base: "golang:1.22"}}

	_, err := bootstrap.PromptToolchain(strings.NewReader("9\n"), &strings.Builder{}, candidates)

	assert.ErrorContains(t, err, "invalid toolchain selection")
}

func TestResolveToolchain_fallsBackWhenNoneDetected(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	tc, err := bootstrap.ResolveToolchain(dir, strings.NewReader(""), &strings.Builder{})

	require.NoError(t, err)
	assert.Equal(t, bootstrap.Fallback.Base, tc.Base)
}

func TestResolveToolchain_asksHumanWhenMultipleDetected(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	touch(t, dir, "go.mod")
	touch(t, dir, "package.json")

	tc, err := bootstrap.ResolveToolchain(dir, strings.NewReader("1\n"), &strings.Builder{})

	require.NoError(t, err)
	assert.Equal(t, "go", tc.Name)
}
