package packet

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ProbeReport is one adversarial probe's outcome: whether the real gate
// caught the seeded known-bad revision it was aimed at.
type ProbeReport struct {
	Name   string
	Gate   Gate
	Caught bool
}

// String reports the probe's outcome in the operator's terms: a caught
// probe confirms the gate is doing its job; an escaped one is a live
// warning that the gate just failed to catch something it should have.
func (r ProbeReport) String() string {
	if r.Caught {
		return r.Name + ": caught (" + r.Gate.Detail + ")"
	}
	return r.Name + ": ESCAPED (" + r.Gate.Detail + ") — the gate did not catch a known-bad revision"
}

// probeGit runs one git command in dir, failing loudly on error — a probe
// with a broken git/exec environment should fail visibly, the same way
// RunBuildVetGate's own exec calls do; there is nothing honest to report if
// the probe itself can't run.
func probeGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("probe: git %v: %w: %s", args, err, out)
	}
	return strings.TrimSpace(string(out)), nil
}

// materializeKnownBadRevision builds a throwaway git repo containing a Go
// module with a deliberate syntax error — a revision a healthy G4 gate must
// always fail. The caller owns cleanup (os.RemoveAll on the returned dir).
func materializeKnownBadRevision() (dir, rev string, err error) {
	dir, err = os.MkdirTemp("", "packets-probe-*")
	if err != nil {
		return "", "", fmt.Errorf("probe: %w", err)
	}
	if _, err = probeGit(dir, "init", "-q"); err != nil {
		os.RemoveAll(dir)
		return "", "", err
	}
	if _, err = probeGit(dir, "config", "user.email", "probe@packets"); err != nil {
		os.RemoveAll(dir)
		return "", "", err
	}
	if _, err = probeGit(dir, "config", "user.name", "packets-probe"); err != nil {
		os.RemoveAll(dir)
		return "", "", err
	}
	if err = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module packetsprobe\n\ngo 1.21\n"), 0o644); err != nil {
		os.RemoveAll(dir)
		return "", "", fmt.Errorf("probe: %w", err)
	}
	// The deliberate defect: invalid Go syntax, guaranteed to fail `go build`.
	if err = os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() { this is not valid go\n"), 0o644); err != nil {
		os.RemoveAll(dir)
		return "", "", fmt.Errorf("probe: %w", err)
	}
	if _, err = probeGit(dir, "add", "-A"); err != nil {
		os.RemoveAll(dir)
		return "", "", err
	}
	if _, err = probeGit(dir, "commit", "-qm", "known-bad probe fixture"); err != nil {
		os.RemoveAll(dir)
		return "", "", err
	}
	rev, err = probeGit(dir, "rev-parse", "HEAD")
	if err != nil {
		os.RemoveAll(dir)
		return "", "", err
	}
	return dir, rev, nil
}

// RunAdversarialProbe is the ADVERSARIAL inspection mode (design/guidelines/
// concepts.md: "actively tries to break packets; seeds probes to keep
// everyone honest"): it materializes a self-contained, deliberately-broken
// revision and runs it through the SAME real gate a genuine packet's G4
// uses. Nothing here touches any session's ledger or repo — the probe is
// fully self-contained, so it can never pollute a real economy. A healthy
// gate reports GateFailed (Caught); anything else means the gate just let a
// known-bad revision through.
func RunAdversarialProbe(ctx context.Context) (ProbeReport, error) {
	dir, rev, err := materializeKnownBadRevision()
	if err != nil {
		return ProbeReport{}, err
	}
	defer os.RemoveAll(dir)

	gate := RunBuildVetGate(ctx, dir, rev)
	return ProbeReport{
		Name:   "known-bad-build",
		Gate:   gate,
		Caught: gate.Status == GateFailed,
	}, nil
}
