package fabric_test

import (
	"path/filepath"
	"testing"

	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadIndex_returnsEmptyIndexWhenFileMissing(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "fabrics.yaml")

	idx, err := fabric.LoadIndex(path)

	require.NoError(t, err)
	assert.Empty(t, idx.Fabrics)
}

func TestSaveLoadIndex_roundTrips(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "fabrics.yaml")
	want := &fabric.Index{Fabrics: []fabric.IndexEntry{
		{Slug: "auth-service", Remote: "git@github.com:acme/auth-service.git", RepoPath: "/repos/auth-service"},
	}}

	require.NoError(t, fabric.SaveIndex(path, want))
	got, err := fabric.LoadIndex(path)

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestIndexBySlug_findsRegisteredFabric(t *testing.T) {
	t.Parallel()
	idx := &fabric.Index{Fabrics: []fabric.IndexEntry{
		{Slug: "auth-service", RepoPath: "/repos/auth-service"},
	}}

	got, ok := idx.BySlug("auth-service")

	assert.True(t, ok)
	assert.Equal(t, "/repos/auth-service", got.RepoPath)
}

func TestIndexBySlug_reportsNotFound(t *testing.T) {
	t.Parallel()
	idx := &fabric.Index{}

	_, ok := idx.BySlug("missing")

	assert.False(t, ok)
}

func TestIndexByCWD_findsFabricContainingCWD(t *testing.T) {
	t.Parallel()
	idx := &fabric.Index{Fabrics: []fabric.IndexEntry{
		{Slug: "auth-service", RepoPath: "/repos/auth-service"},
	}}

	got, err := idx.ByCWD("/repos/auth-service/internal/pkg")

	require.NoError(t, err)
	assert.Equal(t, "auth-service", got.Slug)
}

func TestIndexByCWD_errorsWhenNoFabricContainsCWD(t *testing.T) {
	t.Parallel()
	idx := &fabric.Index{Fabrics: []fabric.IndexEntry{
		{Slug: "auth-service", RepoPath: "/repos/auth-service"},
	}}

	_, err := idx.ByCWD("/repos/other-service")

	assert.Error(t, err)
}
