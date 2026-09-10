package fabric_test

import (
	"path/filepath"
	"testing"

	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveLoad_roundTripsAllFields(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "fabric.yaml")
	want := &fabric.Config{
		Slug:            "auth-service",
		Remote:          "git@github.com:acme/auth-service.git",
		RepoPath:        "/home/user/repos/auth-service",
		DefaultBranch:   "main",
		Image:           "packets/auth-service:v0",
		CredentialsPath: "/home/user/.claude/.credentials.json",
		BudgetDefaults: fabric.BudgetDefaults{
			Retries: 5,
			Minutes: 60,
		},
		SmellTriggers: fabric.SmellTriggers{
			MaxFilesTouched:      10,
			SameErrorRepeats:     3,
			DenyTestModification: true,
			DenyDependencyAdd:    true,
			DenyTerminalEdit:     true,
		},
		CI: fabric.CI{
			PollSeconds:    30,
			TimeoutMinutes: 30,
			WorkflowName:   "packets",
		},
		Merge: fabric.Merge{
			Auto:     true,
			Strategy: "squash",
		},
	}

	require.NoError(t, fabric.Save(path, want))
	got, err := fabric.Load(path)

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestLoad_errorsOnMissingFile(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "does-not-exist.yaml")

	_, err := fabric.Load(path)

	assert.Error(t, err)
}
