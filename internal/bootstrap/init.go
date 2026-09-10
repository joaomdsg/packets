// Package bootstrap implements `packets init` (§11): registering a fabric,
// building its image, and opening the CI workflow PR.
package bootstrap

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/joaomdsg/packets/internal/claudeauth"
	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/slug"
)

// Deps are init's external-process and I/O boundaries. Tests inject
// hand-rolled stubs for Git, GH, and Docker; production wires ExecGit,
// ExecGH, ExecDocker, os.Getenv, an os.Stat-backed Exists, os.Stdin,
// os.Stdout.
type Deps struct {
	Git    Git
	GH     GH
	Docker Docker
	Getenv func(string) string
	Exists func(string) bool
	Stdin  io.Reader
	Stdout io.Writer
}

// InitOptions locates the repo being initialized and the XDG dirs to
// write to; tests override ConfigDir/DataDir to avoid touching the real
// home directory.
type InitOptions struct {
	RepoDir   string
	ConfigDir string
	DataDir   string
}

// Result reports what Init did.
type Result struct {
	Slug        string
	AlreadyInit bool
	ImageTag    string
	PRURL       string
}

const smokeTestPrompt = "reply with the single word OK"

// Init runs §11's 8 bootstrap steps in order, stopping at the first
// failure.
func Init(deps Deps, opts InitOptions) (Result, error) {
	// Step 1: remote → fabric slug.
	remote, err := deps.Git.RemoteURL(opts.RepoDir)
	if err != nil {
		return Result{}, fmt.Errorf("bootstrap: get remote: %s", err)
	}
	fabricSlug := slug.Fabric(remote)

	indexPath := filepath.Join(opts.ConfigDir, "fabrics.yaml")
	idx, err := fabric.LoadIndex(indexPath)
	if err != nil {
		return Result{}, fmt.Errorf("bootstrap: %s", err)
	}

	// Step 2: already-initialized short-circuit.
	for _, e := range idx.Fabrics {
		if e.Remote == remote {
			fmt.Fprintln(deps.Stdout, "already initialized")
			return Result{Slug: e.Slug, AlreadyInit: true}, nil
		}
	}

	// Step 3: fabric.yaml with §4 defaults.
	repoPath, err := deps.Git.Toplevel(opts.RepoDir)
	if err != nil {
		return Result{}, fmt.Errorf("bootstrap: get repo toplevel: %s", err)
	}
	defaultBranch, err := deps.GH.DefaultBranch(opts.RepoDir)
	if err != nil {
		return Result{}, fmt.Errorf("bootstrap: get default branch: %s", err)
	}

	fabricDir := filepath.Join(opts.DataDir, fabricSlug)
	if err := os.MkdirAll(fabricDir, 0o700); err != nil {
		return Result{}, fmt.Errorf("bootstrap: create %s: %s", fabricDir, err)
	}

	imageTag := fmt.Sprintf("packets/%s:v0", fabricSlug)
	cfg := &fabric.Config{
		Slug:          fabricSlug,
		Remote:        remote,
		RepoPath:      repoPath,
		DefaultBranch: defaultBranch,
		Image:         imageTag,
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
	if err := fabric.Save(filepath.Join(fabricDir, "fabric.yaml"), cfg); err != nil {
		return Result{}, fmt.Errorf("bootstrap: %s", err)
	}

	// Step 4: toolchain detection, or ask the human if ambiguous.
	toolchain, err := ResolveToolchain(repoPath, deps.Stdin, deps.Stdout)
	if err != nil {
		return Result{}, fmt.Errorf("bootstrap: %s", err)
	}

	// Step 5: image assets + docker build.
	if err := writeImageAssets(fabricDir, toolchain.Base); err != nil {
		return Result{}, fmt.Errorf("bootstrap: %s", err)
	}
	if err := deps.Docker.Build(fabricDir, imageTag); err != nil {
		return Result{}, fmt.Errorf("bootstrap: docker build: %s", err)
	}

	// Step 6: smoke test.
	if err := smokeTest(deps, imageTag, cfg.CredentialsPath); err != nil {
		return Result{}, err
	}

	// Step 7: CI workflow PR.
	prURL, err := openWorkflowPR(deps, repoPath, defaultBranch, toolchain)
	if err != nil {
		return Result{}, fmt.Errorf("bootstrap: %s", err)
	}

	// Step 8: register the fabric.
	idx.Fabrics = append(idx.Fabrics, fabric.IndexEntry{
		Slug:     fabricSlug,
		Remote:   remote,
		RepoPath: repoPath,
	})
	if err := os.MkdirAll(opts.ConfigDir, 0o700); err != nil {
		return Result{}, fmt.Errorf("bootstrap: create %s: %s", opts.ConfigDir, err)
	}
	if err := fabric.SaveIndex(indexPath, idx); err != nil {
		return Result{}, fmt.Errorf("bootstrap: %s", err)
	}

	return Result{Slug: fabricSlug, ImageTag: imageTag, PRURL: prURL}, nil
}

// smokeTest requires either mountable OAuth credentials or
// ANTHROPIC_API_KEY (internal/claudeauth) — silently skipping it would let
// a broken image pass init, so having neither is a hard failure with an
// actionable message, not a soft skip.
func smokeTest(deps Deps, imageTag, credentialsPath string) error {
	authArgs, err := claudeauth.Resolve(credentialsPath, deps.Getenv, deps.Exists)
	if err != nil {
		return fmt.Errorf("bootstrap: smoke test: %s", err)
	}
	args := append([]string{"run", "--rm"}, authArgs...)
	args = append(args, imageTag, "claude", "-p", smokeTestPrompt, "--output-format", "json")
	out, err := deps.Docker.Run(args)
	if err != nil {
		return fmt.Errorf("bootstrap: smoke test: %s", err)
	}
	if !strings.Contains(out, "OK") {
		return fmt.Errorf("bootstrap: smoke test: response did not contain OK: %s", out)
	}
	return nil
}

// openWorkflowPR writes the CI workflow into the repo (the only file the
// harness ever adds to it, §0.4) and opens a PR for the human to merge.
func openWorkflowPR(deps Deps, repoPath, defaultBranch string, toolchain Toolchain) (string, error) {
	content, err := workflowContent(toolchain.Setup)
	if err != nil {
		return "", err
	}
	workflowDir := filepath.Join(repoPath, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		return "", fmt.Errorf("create %s: %s", workflowDir, err)
	}
	workflowPath := filepath.Join(workflowDir, "packets.yml")
	if err := os.WriteFile(workflowPath, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write %s: %s", workflowPath, err)
	}

	branch := "packets/init"
	if err := deps.Git.CreateBranch(repoPath, branch); err != nil {
		return "", fmt.Errorf("create branch %s: %s", branch, err)
	}
	if err := deps.Git.CommitAll(repoPath, "packets: add CI workflow"); err != nil {
		return "", fmt.Errorf("commit workflow: %s", err)
	}
	if err := deps.Git.Push(repoPath, branch); err != nil {
		return "", fmt.Errorf("push %s: %s", branch, err)
	}
	url, err := deps.GH.CreatePR(repoPath, "packets: add CI workflow",
		"Adds the packets CI workflow. Merge this to enable packets run for this repo.",
		branch, defaultBranch)
	if err != nil {
		return "", fmt.Errorf("open PR: %s", err)
	}
	return url, nil
}
