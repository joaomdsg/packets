package packet

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"path"
	"strings"

	"github.com/joaomdsg/packets/internal/diff"
)

// goListPackage is the subset of `go list -json` output this file reads: a
// package's own import path and every path it imports (module-local or
// not — LoadImportGraph filters to module-local after the fact).
type goListPackage struct {
	ImportPath string
	Imports    []string
}

// goListJSON runs `go list -json <args...>` in repoDir and decodes the
// resulting stream of concatenated JSON objects (one per package — NOT a
// JSON array, so a plain json.Decoder loop is required).
func goListJSON(ctx context.Context, repoDir string, args ...string) ([]goListPackage, error) {
	cmd := exec.CommandContext(ctx, "go", append([]string{"list", "-json"}, args...)...)
	cmd.Dir = repoDir
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("packet: go list -json %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(errBuf.String()))
	}

	dec := json.NewDecoder(&out)
	var pkgs []goListPackage
	for {
		var p goListPackage
		if err := dec.Decode(&p); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("packet: parse go list -json output: %w", err)
		}
		pkgs = append(pkgs, p)
	}
	return pkgs, nil
}

// LoadImportGraph builds the module-local import graph via `go list -json
// ./...` in repoDir: every package's ImportPath becomes a graph key, and an
// import edge survives only when the imported path is ALSO one of those
// keys. An externally-rooted import (standard library or a third-party
// module) is filtered out by checking membership in the module's own
// package set — never by parsing go.mod — so the graph needs no notion of
// the module's own path.
func LoadImportGraph(ctx context.Context, repoDir string) (ImportGraph, error) {
	pkgs, err := goListJSON(ctx, repoDir, "./...")
	if err != nil {
		return nil, err
	}

	local := make(map[string]bool, len(pkgs))
	for _, p := range pkgs {
		local[p.ImportPath] = true
	}

	graph := make(ImportGraph, len(pkgs))
	for _, p := range pkgs {
		var imports []string
		for _, imp := range p.Imports {
			if local[imp] {
				imports = append(imports, imp)
			}
		}
		graph[p.ImportPath] = imports
	}
	return graph, nil
}

// dirArg turns a changed file's directory into a `go list` file-system path
// argument: "." (the repo root) stays "."; anything else is prefixed "./" so
// go list treats it as a path rather than guessing at an import path. Git
// always reports paths with "/" separators regardless of OS, so this is
// built with the "path" package, not "path/filepath".
func dirArg(dir string) string {
	if dir == "." {
		return "."
	}
	return "./" + dir
}

// ChangedPackages maps changed file paths (relative to repoDir) to their
// containing package's import path, via one batched `go list -json` call
// over each unique directory — cheaper than one invocation per file. A file
// whose extension isn't ".go" names no package and contributes nothing. The
// returned set is deduped and may be empty (including when files is empty,
// which short-circuits without invoking go list at all).
func ChangedPackages(repoDir string, files []string) ([]string, error) {
	dirSet := make(map[string]bool, len(files))
	for _, f := range files {
		if path.Ext(f) != ".go" {
			continue
		}
		dirSet[dirArg(path.Dir(f))] = true
	}
	if len(dirSet) == 0 {
		return nil, nil
	}

	dirs := make([]string, 0, len(dirSet))
	for d := range dirSet {
		dirs = append(dirs, d)
	}

	pkgs, err := goListJSON(context.Background(), repoDir, dirs...)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool, len(pkgs))
	var out []string
	for _, p := range pkgs {
		if !seen[p.ImportPath] {
			seen[p.ImportPath] = true
			out = append(out, p.ImportPath)
		}
	}
	return out, nil
}

// Measure derives a Lane for the change between baseRev and fixRev in the
// git repo at repoDir: the diffed file paths become changed packages (via
// ChangedPackages), scored against the repo's own import graph (via
// LoadImportGraph) through BlastRadius/LaneFor. Any failure along the way —
// an unresolved revision, a `go list` error — returns (LaneUnmeasured, err):
// a lane is never guessed when it can't be measured.
//
// ChangedPackages and LoadImportGraph both run `go list` against repoDir's
// current on-disk working tree, not a checkout of fixRev — callers must
// ensure the working tree already reflects fixRev (the live server's
// sessions always run their catch cycle at the fix revision, so this holds
// in practice; a caller comparing two arbitrary historical revisions with a
// different tree checked out would measure the wrong file state).
func Measure(ctx context.Context, repoDir, baseRev, fixRev string) (Lane, error) {
	d, err := diff.Compute(ctx, repoDir, baseRev, fixRev)
	if err != nil {
		return LaneUnmeasured, fmt.Errorf("measure: diff: %s", err)
	}

	files := make([]string, len(d.Files))
	for i, f := range d.Files {
		files[i] = f.Path
	}

	changed, err := ChangedPackages(repoDir, files)
	if err != nil {
		return LaneUnmeasured, fmt.Errorf("measure: changed packages: %s", err)
	}

	graph, err := LoadImportGraph(ctx, repoDir)
	if err != nil {
		return LaneUnmeasured, fmt.Errorf("measure: import graph: %s", err)
	}

	affected, total := BlastRadius(graph, changed)
	return LaneFor(affected, total), nil
}
