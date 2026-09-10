package bootstrap

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Toolchain names one ecosystem init can detect: the Dockerfile base image
// to use and the CI workflow's toolchain-setup shell snippet.
type Toolchain struct {
	Name  string
	Base  string
	Setup string
}

// Fallback is used when no ecosystem marker is found in the repo.
var Fallback = Toolchain{Name: "none", Base: "ubuntu:24.04", Setup: "          echo no toolchain-specific setup required"}

// toolchains lists every detectable ecosystem in §11 step 4's priority
// order. A single ecosystem may match on more than one marker file
// (pyproject.toml or requirements.txt both mean python) without counting
// as "more than one" for the ask-the-human rule.
var toolchains = []struct {
	Toolchain
	markers []string
}{
	{Toolchain{"go", "golang:1.22", "          go mod download"}, []string{"go.mod"}},
	{Toolchain{"node", "node:20", "          npm ci"}, []string{"package.json"}},
	{Toolchain{"python", "python:3.12", "          pip install -r requirements.txt || pip install ."}, []string{"pyproject.toml", "requirements.txt"}},
}

// DetectToolchains reports every ecosystem whose marker file exists at the
// repo root, in priority order. Zero or one result means init can proceed
// automatically; more than one means the human must be asked (§11 step 4,
// §17).
func DetectToolchains(repoPath string) ([]Toolchain, error) {
	var found []Toolchain
	for _, tc := range toolchains {
		present := false
		for _, marker := range tc.markers {
			_, err := os.Stat(filepath.Join(repoPath, marker))
			if err == nil {
				present = true
				break
			}
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("bootstrap: stat %s: %s", marker, err)
			}
		}
		if present {
			found = append(found, tc.Toolchain)
		}
	}
	return found, nil
}

// PromptToolchain asks the human to pick among candidates when more than
// one ecosystem marker is present, reading a 1-based index from r.
func PromptToolchain(r io.Reader, w io.Writer, candidates []Toolchain) (Toolchain, error) {
	fmt.Fprintln(w, "packets: multiple toolchains detected, pick one:")
	for i, c := range candidates {
		fmt.Fprintf(w, "  %d) %s (%s)\n", i+1, c.Name, c.Base)
	}
	fmt.Fprint(w, "> ")

	scanner := bufio.NewScanner(r)
	if !scanner.Scan() {
		return Toolchain{}, fmt.Errorf("bootstrap: no toolchain selection given")
	}
	choice, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
	if err != nil || choice < 1 || choice > len(candidates) {
		return Toolchain{}, fmt.Errorf("bootstrap: invalid toolchain selection %q", scanner.Text())
	}
	return candidates[choice-1], nil
}

// ResolveToolchain implements §11 step 4 in full: detect, then either
// return the single match, fall back to Fallback, or ask the human.
func ResolveToolchain(repoPath string, r io.Reader, w io.Writer) (Toolchain, error) {
	found, err := DetectToolchains(repoPath)
	if err != nil {
		return Toolchain{}, err
	}
	switch len(found) {
	case 0:
		return Fallback, nil
	case 1:
		return found[0], nil
	default:
		return PromptToolchain(r, w, found)
	}
}
