package packet

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// handshakeTestTimeout bounds `go test ./handshake/...` — the handshake is
// meant to run at line rate, so a wedged run must never
// hang the caller.
const handshakeTestTimeout = 30 * time.Second

// notFoundDetail is G2's curated detail for a handshake package that is
// missing or fails to compile at the fix revision — a REAL finding (the
// human authored one; its absence is not "unmeasured"), never GateNotRun.
const notFoundDetail = "handshake test package not found at this revision"

// RunHandshakeGate is G2 (handshake conformance): once a
// handshake exists, it materializes fixRev into a THROWAWAY git worktree
// (mirroring RunBuildVetGate's worktree pattern exactly) and runs
// `go test ./handshake/...` inside it. handshakePath=="" means no handshake
// has been authored yet — the honest GateNotRun default, matching the gauntlet's
// placeholder. A missing or uncompileable handshake package at fixRev is
// distinguished from a genuine test failure by whether `go test` ever got as
// far as running an individual test (a "--- FAIL:" line in its output): no
// such line means the package itself never ran, which is the missing/
// uncompileable case (notFoundDetail); a real failure gets the same
// truncated-tail treatment as RunBuildVetGate.
func RunHandshakeGate(ctx context.Context, repoDir, fixRev, handshakePath string) Gate {
	if handshakePath == "" {
		return Gate{Status: GateNotRun, Detail: "no handshake authored"}
	}
	if fixRev == "" || !isGitRepo(ctx, repoDir) {
		return Gate{Status: GateNotRun, Detail: notRunNoRevision}
	}

	ctx, cancel := context.WithTimeout(ctx, handshakeTestTimeout)
	defer cancel()

	tmpDir, err := os.MkdirTemp("", "packets-gauntlet-handshake-*")
	if err != nil {
		return Gate{Status: GateNotRun, Detail: "could not prepare a scratch worktree"}
	}
	defer os.RemoveAll(tmpDir)

	worktree := filepath.Join(tmpDir, "wt")
	if _, err := runIn(ctx, repoDir, "git", "worktree", "add", "--detach", "--quiet", worktree, fixRev); err != nil {
		if ctx.Err() != nil {
			return Gate{Status: GateNotRun, Detail: "timed out checking out fix revision"}
		}
		return Gate{Status: GateNotRun, Detail: "could not check out fix revision"}
	}
	defer func() {
		_, _ = runIn(context.Background(), repoDir, "git", "worktree", "remove", "--force", worktree)
	}()

	out, err := runIn(ctx, worktree, "go", "test", "./handshake/...")
	if err == nil {
		return Gate{Status: GatePassed, Detail: "handshake tests pass"}
	}
	if ctx.Err() != nil {
		// A kill mid-run is indistinguishable from "package missing" by output
		// alone (neither ever prints "--- FAIL:") — check the timeout FIRST so a
		// merely-slow handshake is never mistaken for one that doesn't exist.
		return Gate{Status: GateNotRun, Detail: "go test: timed out before finishing"}
	}
	if !strings.Contains(out, "--- FAIL:") {
		return Gate{Status: GateFailed, Detail: notFoundDetail}
	}
	return Gate{Status: GateFailed, Detail: "go test: " + truncateTail(out, detailTailLimit)}
}
