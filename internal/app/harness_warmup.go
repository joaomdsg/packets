package app

import (
	"context"
	"crypto/rand"
	"fmt"
	"os/exec"
	"time"
)

// warmHarnessTimeout bounds the warm-up explore so a stuck run can't leak a process
// or hold the session cold forever.
const warmHarnessTimeout = 2 * time.Minute

// warmHarnessRun is the seam the warm-up explore runs through (default shells claude;
// tests swap it for a recorder, no real model call).
var warmHarnessRun = runWarmHarness

// runWarmHarness explores repoDir under the pinned sessionID, establishing the
// resumable session the later analyze/order requests fork from.
func runWarmHarness(ctx context.Context, repoDir, sessionID string) error {
	cmd := exec.CommandContext(ctx, "claude", warmArgs(sessionID)...)
	cmd.Dir = repoDir
	return cmd.Run()
}

// startWarmHarness generates this session's resumable id, remembers it on the entry,
// and explores the repo in the background — so the session is warm by the time the
// Lead authors or places work. The id is set synchronously (remembered immediately);
// the entry only becomes WARM (resumable) when the explore completes, so a request
// before then runs cold rather than resuming a half-established session.
func startWarmHarness(e *liveEntry, repoDir string) {
	id := newSessionID()
	if id == "" {
		return // no randomness → no warm session; requests run cold
	}
	e.findingsMu.Lock()
	e.harnessSessionID = id
	e.findingsMu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), warmHarnessTimeout)
		defer cancel()
		if err := warmHarnessRun(ctx, repoDir, id); err == nil {
			e.markWarm()
		}
	}()
}

// newSessionID mints a random v4 UUID for --session-id. Returns "" on an
// unrecoverable randomness failure, so the caller leaves the session cold.
func newSessionID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return ""
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// warmExplorePrompt is the warm-up task: it asks the harness to explore the repo so
// the session's resumable id carries a real mental model of the codebase, which every
// later analyze + order resumes. A one-line reply keeps the warm-up cheap.
const warmExplorePrompt = "Explore this repository — its layout, primary language, build, tests, and conventions — so you can act on focused tasks in it later. Reply with a one-line summary of what it is."

// warmArgs is the claude argv for the warm-up explore. It PINS the session id
// (--session-id) so later requests can --resume it, reads the repo with tool access
// (bypassPermissions), and is a one-shot text run (the reply is discarded; the point
// is to populate the session with repo context). No --model override: the explore
// runs the default model, the richer base the order fills resume.
func warmArgs(sessionID string) []string {
	return []string{
		"--session-id", sessionID,
		"-p", warmExplorePrompt,
		"--output-format", "text",
		"--permission-mode", "bypassPermissions",
	}
}
