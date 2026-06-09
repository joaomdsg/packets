package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/pipe"
	"github.com/joaomdsg/packets/internal/sandbox"
)

// The governor values must couple correctly REGARDLESS of platform: a durable
// consumer acks AFTER the verify completes, so AckWait MUST exceed the per-claim
// verify deadline — otherwise a slow-but-legal verify is redelivered while the
// first is still running, producing a concurrent DOUBLE cage run on one claim.
// And the concurrency cap must never be a zero-capacity (deadlocking) semaphore.
func TestClaimGovernorValues_ackWaitOutlastsTheVerifyDeadline(t *testing.T) {
	require.Greater(t, claimAckWait, cageVerifyTimeout,
		"AckWait must outlast the verify deadline or a slow verify is redelivered into a concurrent double cage run")
	require.GreaterOrEqual(t, claimConcurrency(), 1,
		"the concurrent-verify cap must be at least 1 — never a zero-capacity, deadlocking semaphore")
	require.LessOrEqual(t, claimConcurrency(), 4,
		"the concurrent-verify cap is bounded — each cage run reserves ~1cpu/256m, so the fleet never oversubscribes")
}

// blessingRunner is a fake sandbox.Runner standing in for Docker at the I/O
// seam: it returns a canned catch transcript so the real Materialize → buildSpec
// → (faked) Run → parseTranscript → DeriveCatch → mint path can be exercised
// offline. The real DockerRunner path is locked separately by the equivalence lock.
type blessingRunner struct {
	output  string
	invoked *atomic.Int32
}

func (r blessingRunner) Run(_ context.Context, _ sandbox.Spec) (sandbox.Result, error) {
	r.invoked.Add(1)
	return sandbox.Result{Output: r.output}, nil
}

// StartCageClaimConsumers wires a WORKING CageVerifier through the live
// route→publish→consume→verify→mint path: a producer POSTs a claim over a real
// repo's revisions, the server-spawned consumer materializes the repo, runs the
// (faked) cage, re-derives the verdict from the transcript, and mints — proving
// the production wiring builds a verifier that actually confirms, with Docker
// faked only at the runner seam.
func TestStartCageClaimConsumers_wiresAWorkingCageVerifierThatMints(t *testing.T) {
	repo, base, fix, tip := inlineRepoWithTwoRevs(t)

	output, err := json.Marshal(pipe.Transcript{
		Outcome: catch.Catch, Reason: pipe.ReasonNone, Path: "adult.go", Line: 2, Land: pipe.LandClean,
		Before: catch.LineState{Inventory: []string{">="}, Survivors: []string{">="}},
		After:  catch.LineState{Inventory: []string{">="}, Survivors: nil},
	})
	require.NoError(t, err)

	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: repo, BaseRev: base, FixRev: fix, TipRev: tip, Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	var invoked atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	StartCageClaimConsumers(ctx, "img", blessingRunner{output: string(output), invoked: &invoked})

	body := `{"base_rev":"` + base + `","fix_rev":"` + fix + `","tip_rev":"` + tip + `","path":"adult.go","line":2}`
	resp, err := http.Post(server.URL+"/claim", "application/json", strings.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusAccepted, resp.StatusCode)

	require.Eventually(t, func() bool {
		b, err := log.Balance()
		return err == nil && b == 1
	}, 5*time.Second, 25*time.Millisecond,
		"the wired CageVerifier must materialize the repo, run the faked cage, derive the catch, and mint")
	require.GreaterOrEqual(t, invoked.Load(), int32(1),
		"the wiring must actually invoke the runner — proving a real CageVerifier was built, not a stub that mints blind")
}

// inlineRepoWithTwoRevs builds a real git repo with two commits so a claim
// Target's base/fix/tip resolve in Materialize. Offline, no network.
func inlineRepoWithTwoRevs(t *testing.T) (dir, base, fix, tip string) {
	t.Helper()
	dir = t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
		out, err := cmd.CombinedOutput()
		require.NoErrorf(t, err, "git %v: %s", args, out)
	}
	rev := func() string {
		t.Helper()
		out, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
		require.NoError(t, err)
		return strings.TrimSpace(string(out))
	}
	write := func(name, content string) {
		t.Helper()
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
	}
	git("init", "-q")
	git("config", "user.email", "t@t")
	git("config", "user.name", "t")
	write("adult.go", "package adult\nfunc Adult(age int) bool { return age >= 18 }\n")
	git("add", "-A")
	git("commit", "-q", "-m", "base")
	base = rev()
	write("adult.go", "package adult\nfunc Adult(age int) bool { return age >= 21 }\n")
	git("add", "-A")
	git("commit", "-q", "-m", "fix")
	fix = rev()
	write("extra.go", "package adult\n")
	git("add", "-A")
	git("commit", "-q", "-m", "tip")
	tip = rev()
	return dir, base, fix, tip
}
