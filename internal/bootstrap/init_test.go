package bootstrap_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joaomdsg/packets/internal/bootstrap"
	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubGit struct {
	remote      string
	remoteErr   error
	toplevelErr error
	branches    []string
	commits     []string
	pushed      []string
}

func (s *stubGit) RemoteURL(dir string) (string, error) { return s.remote, s.remoteErr }
func (s *stubGit) Toplevel(dir string) (string, error)  { return dir, s.toplevelErr }
func (s *stubGit) CreateBranch(dir, branch string) error {
	s.branches = append(s.branches, branch)
	return nil
}
func (s *stubGit) CommitAll(dir, message string) error {
	s.commits = append(s.commits, message)
	return nil
}
func (s *stubGit) Push(dir, branch string) error {
	s.pushed = append(s.pushed, branch)
	return nil
}

type stubGH struct {
	defaultBranch string
	prURL         string
	prTitles      []string
}

func (s *stubGH) DefaultBranch(dir string) (string, error) { return s.defaultBranch, nil }
func (s *stubGH) CreatePR(dir, title, body, head, base string) (string, error) {
	s.prTitles = append(s.prTitles, title)
	return s.prURL, nil
}

type stubDocker struct {
	buildDir, buildTag string
	buildErr           error
	runArgs            []string
	runOut             string
	runErr             error
}

func (s *stubDocker) Build(dir, tag string) error {
	s.buildDir, s.buildTag = dir, tag
	return s.buildErr
}
func (s *stubDocker) Run(args []string) (string, error) {
	s.runArgs = args
	return s.runOut, s.runErr
}

// stubDeps wires a fully-working set of stubs: a fabric slug "example",
// default branch "main", a successful docker build/smoke test, and a fake
// API key — Init's own tests must not depend on ANTHROPIC_API_KEY being
// set in the real environment.
func stubDeps(t *testing.T) bootstrap.Deps {
	t.Helper()
	return bootstrap.Deps{
		Git:    &stubGit{remote: "git@github.com:acme/example.git"},
		GH:     &stubGH{defaultBranch: "main", prURL: "https://github.com/acme/example/pull/1"},
		Docker: &stubDocker{runOut: `{"result":"OK"}`},
		Getenv: func(key string) string {
			if key == "ANTHROPIC_API_KEY" {
				return "fake-key"
			}
			return ""
		},
		Exists: func(string) bool { return false },
		Stdin:  strings.NewReader(""),
		Stdout: &strings.Builder{},
	}
}

func TestInit_writesFabricYAMLWithSectionFourDefaults(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	repo := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(repo, "go.mod"), []byte("module x\n"), 0o600))

	opts := bootstrap.InitOptions{RepoDir: repo, ConfigDir: filepath.Join(dir, "config"), DataDir: filepath.Join(dir, "data")}
	result, err := bootstrap.Init(stubDeps(t), opts)
	require.NoError(t, err)
	assert.Equal(t, "example", result.Slug)

	cfg, err := fabric.Load(filepath.Join(dir, "data", "example", "fabric.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "example", cfg.Slug)
	assert.Equal(t, "git@github.com:acme/example.git", cfg.Remote)
	assert.Equal(t, repo, cfg.RepoPath)
	assert.Equal(t, "main", cfg.DefaultBranch)
	assert.Equal(t, "packets/example:v0", cfg.Image)
	assert.Equal(t, 5, cfg.BudgetDefaults.Retries)
	assert.Equal(t, 60, cfg.BudgetDefaults.Minutes)
	assert.Equal(t, 10, cfg.SmellTriggers.MaxFilesTouched)
	assert.True(t, cfg.SmellTriggers.DenyTestModification)
	assert.True(t, cfg.Merge.Auto)
	assert.Equal(t, "squash", cfg.Merge.Strategy)
}

func TestInit_appendsFabricsYAMLIndex(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	repo := t.TempDir()

	opts := bootstrap.InitOptions{RepoDir: repo, ConfigDir: filepath.Join(dir, "config"), DataDir: filepath.Join(dir, "data")}
	_, err := bootstrap.Init(stubDeps(t), opts)
	require.NoError(t, err)

	idx, err := fabric.LoadIndex(filepath.Join(dir, "config", "fabrics.yaml"))
	require.NoError(t, err)
	entry, ok := idx.BySlug("example")
	require.True(t, ok)
	assert.Equal(t, "git@github.com:acme/example.git", entry.Remote)
	assert.Equal(t, repo, entry.RepoPath)
}

func TestInit_shortCircuitsWhenAlreadyInitialized(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	repo := t.TempDir()

	opts := bootstrap.InitOptions{RepoDir: repo, ConfigDir: filepath.Join(dir, "config"), DataDir: filepath.Join(dir, "data")}
	deps := stubDeps(t)
	_, err := bootstrap.Init(deps, opts)
	require.NoError(t, err)

	docker := deps.Docker.(*stubDocker)
	docker.buildTag = ""

	out := &strings.Builder{}
	deps.Stdout = out
	result, err := bootstrap.Init(deps, opts)

	require.NoError(t, err)
	assert.True(t, result.AlreadyInit)
	assert.Contains(t, out.String(), "already initialized")
	assert.Empty(t, docker.buildTag, "second init must not rebuild the image")
}

func TestInit_runsDockerBuildWithExpectedArgv(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	repo := t.TempDir()

	deps := stubDeps(t)
	opts := bootstrap.InitOptions{RepoDir: repo, ConfigDir: filepath.Join(dir, "config"), DataDir: filepath.Join(dir, "data")}
	_, err := bootstrap.Init(deps, opts)
	require.NoError(t, err)

	docker := deps.Docker.(*stubDocker)
	assert.Equal(t, filepath.Join(dir, "data", "example"), docker.buildDir)
	assert.Equal(t, "packets/example:v0", docker.buildTag)
}

func TestInit_smokeTestFailsClearlyWithoutAPIKey(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	repo := t.TempDir()

	deps := stubDeps(t)
	deps.Getenv = func(string) string { return "" }
	opts := bootstrap.InitOptions{RepoDir: repo, ConfigDir: filepath.Join(dir, "config"), DataDir: filepath.Join(dir, "data")}

	_, err := bootstrap.Init(deps, opts)

	assert.ErrorContains(t, err, "ANTHROPIC_API_KEY")
	docker := deps.Docker.(*stubDocker)
	assert.Nil(t, docker.runArgs, "must not invoke docker run when the API key is missing")
}

func TestInit_smokeTestMountsCredentialsFileInsteadOfAPIKeyWhenPresent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	repo := t.TempDir()
	credDir := t.TempDir()
	credPath := filepath.Join(credDir, ".credentials.json")
	require.NoError(t, os.WriteFile(credPath, []byte("placeholder"), 0o600))

	deps := stubDeps(t)
	deps.Getenv = func(key string) string {
		if key == "CLAUDE_CONFIG_DIR" {
			return credDir
		}
		return ""
	}
	deps.Exists = func(p string) bool { return p == credPath }
	opts := bootstrap.InitOptions{RepoDir: repo, ConfigDir: filepath.Join(dir, "config"), DataDir: filepath.Join(dir, "data")}

	_, err := bootstrap.Init(deps, opts)
	require.NoError(t, err)

	docker := deps.Docker.(*stubDocker)
	assert.Contains(t, docker.runArgs, credPath+":/home/agent/.claude-config/.credentials.json:ro")
	assert.NotContains(t, docker.runArgs, "ANTHROPIC_API_KEY")
}

func TestInit_opensWorkflowPRWithoutTouchingOtherRepoFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	repo := t.TempDir()

	deps := stubDeps(t)
	opts := bootstrap.InitOptions{RepoDir: repo, ConfigDir: filepath.Join(dir, "config"), DataDir: filepath.Join(dir, "data")}
	result, err := bootstrap.Init(deps, opts)
	require.NoError(t, err)

	assert.Equal(t, "https://github.com/acme/example/pull/1", result.PRURL)

	workflow, err := os.ReadFile(filepath.Join(repo, ".github", "workflows", "packets.yml"))
	require.NoError(t, err)
	assert.Contains(t, string(workflow), "name: packets")
	assert.NotContains(t, string(workflow), "{{TOOLCHAIN_SETUP}}")

	git := deps.Git.(*stubGit)
	assert.Equal(t, []string{"packets/init"}, git.branches)
	assert.Len(t, git.commits, 1)
	assert.Equal(t, []string{"packets/init"}, git.pushed)

	gh := deps.GH.(*stubGH)
	assert.Equal(t, []string{"packets: add CI workflow"}, gh.prTitles)
}
