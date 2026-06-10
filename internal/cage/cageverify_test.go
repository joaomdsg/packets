package cage_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/cage"
	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/pipe"
	"github.com/joaomdsg/packets/internal/sandbox"
)

// fakeRunner is a test double for the sandbox.Runner I/O boundary: it records the
// Spec it was handed (so the launch the verifier built can be asserted) and
// returns a canned Result/err (so transcript parsing and verdict derivation can
// be exercised offline, without Docker).
type fakeRunner struct {
	got              sandbox.Spec
	result           sandbox.Result
	err              error
	repoExistedAtRun bool // whether the mounted workdir held repo/ when Run was called
	block            bool // when true, block until ctx is cancelled (a hung verify)
}

func (f *fakeRunner) Run(ctx context.Context, s sandbox.Spec) (sandbox.Result, error) {
	f.got = s
	if len(s.Mounts) > 0 {
		if fi, err := os.Stat(filepath.Join(s.Mounts[0].Source, "repo")); err == nil && fi.IsDir() {
			f.repoExistedAtRun = true
		}
	}
	if f.block {
		<-ctx.Done() // a verify that never finishes — the deadline must cancel it
		return sandbox.Result{}, ctx.Err()
	}
	return f.result, f.err
}

// The cage must not hang on a verify that never finishes: the per-claim deadline
// cancels it (the runner gets a cancelled ctx, kills the container) and the
// verifier returns an error, minting nothing — rather than blocking forever.
func TestCageVerifier_cancelsAVerifyThatExceedsTheDeadline(t *testing.T) {
	t.Parallel()
	host, base, fix, tip := hostRepoWithThreeRevs(t)
	fake := &fakeRunner{block: true}

	// 2s (not a few ms) so the real git Materialize completes and it's the blocking
	// RUNNER the deadline cancels — the realistic hung-verify point — not the clone.
	start := time.Now()
	rec, err := cage.CageVerifier(fake, host, "img", 2*time.Second)(claimOver(base, fix, tip))
	elapsed := time.Since(start)

	require.Error(t, err, "a verify that exceeds the deadline must be cancelled and surfaced as an error")
	assert.Nil(t, rec, "a cancelled verify mints nothing")
	assert.Less(t, elapsed, 5*time.Second, "the deadline must cancel the verify, not let it hang")
}

func claimOver(base, fix, tip string) ledger.ClaimRecord {
	return ledger.ClaimRecord{Target: targetOf(base, fix, tip)} // targetOf: Path "adult.go", Line 2
}

// A claim the host can never resolve (a producer's commit that never reached the
// host, or a bogus SHA) is a PERMANENT failure. CageVerifier must surface it as
// ledger.ErrClaimUnverifiable — the seam the in-package-ledger consumer keys on to
// durably REJECT the claim — rather than as a plain (transient) error that would
// leave the bet lingering in flight forever. A transient runner failure over a
// RESOLVABLE claim must NOT carry that sentinel.
func TestCageVerifier_anUnresolvableClaimSurfacesThePermanentRejectSentinel(t *testing.T) {
	t.Parallel()
	host, _, fix, tip := hostRepoWithThreeRevs(t)
	bogus := "0123456789012345678901234567890123456789"

	t.Run("unresolvable revision is permanent (the reject sentinel)", func(t *testing.T) {
		t.Parallel()
		fake := &fakeRunner{result: sandbox.Result{Output: catchTranscriptJSON(t, "adult.go", 2)}}
		rec, err := cage.CageVerifier(fake, host, "img", 30*time.Second)(
			ledger.ClaimRecord{Target: ledger.Target{BaseRev: bogus, FixRev: fix, TipRev: tip, Path: "adult.go", Line: 2}})
		require.Error(t, err)
		assert.Nil(t, rec, "an unverifiable claim mints nothing")
		assert.True(t, errors.Is(err, ledger.ErrClaimUnverifiable),
			"the consumer must be able to durably reject this — it can never verify")
	})

	t.Run("a runner failure over a resolvable claim is transient (no reject sentinel)", func(t *testing.T) {
		t.Parallel()
		fake := &fakeRunner{err: errors.New("docker daemon hiccup")}
		rec, err := cage.CageVerifier(fake, host, "img", 30*time.Second)(claimOver(fixBase(t, host), fix, tip))
		require.Error(t, err)
		assert.Nil(t, rec)
		assert.False(t, errors.Is(err, ledger.ErrClaimUnverifiable),
			"a transient cage/runner failure must stay resubmittable, never branded permanently unverifiable")
	})
}

// fixBase returns a resolvable base SHA for the host repo (its first commit), so
// the resolvable-claim subtest exercises a real Materialize that succeeds and a
// runner that then fails — isolating the transient (runner) failure path.
func fixBase(t *testing.T, host string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", host, "rev-list", "--max-parents=0", "HEAD").Output()
	require.NoError(t, err)
	return strings.TrimSpace(string(out))
}

// catchTranscriptJSON is the verdict bytes a cage emits for a genuine catch on
// the anchored line: a stable inventory whose survivor-set went non-empty→empty.
func catchTranscriptJSON(t *testing.T, path string, line int) string {
	t.Helper()
	b, err := json.Marshal(pipe.Transcript{
		Outcome: catch.Catch, Reason: pipe.ReasonNone, Path: path, Line: line, Land: pipe.LandClean,
		Before: catch.LineState{Inventory: []string{">="}, Survivors: []string{">="}},
		After:  catch.LineState{Inventory: []string{">="}, Survivors: nil},
	})
	require.NoError(t, err)
	return string(b)
}

func envValue(s sandbox.Spec, name string) (string, bool) {
	for _, e := range s.Env {
		if e.Name == name {
			return e.Value, true
		}
	}
	return "", false
}

// The verifier must hand the runner a launch that mounts the materialized workdir
// WRITABLE at /work and routes the go toolchain at the in-container paths — the
// exact shape the cage-exec spike proved verify-catch needs.
func TestCageVerifier_launchesAWritableWorkMountWithToolchainEnv(t *testing.T) {
	t.Parallel()
	host, base, fix, tip := hostRepoWithThreeRevs(t)
	fake := &fakeRunner{result: sandbox.Result{Output: catchTranscriptJSON(t, "adult.go", 2)}}

	_, err := cage.CageVerifier(fake, host, "packets-cage:dev", 30*time.Second)(claimOver(base, fix, tip))
	require.NoError(t, err)

	require.Len(t, fake.got.Mounts, 1, "exactly one mount: the writable workdir")
	m := fake.got.Mounts[0]
	assert.Equal(t, "/work", m.Target)
	assert.False(t, m.Readonly, "the workdir must be writable (the oracle writes worktrees)")
	assert.True(t, fake.repoExistedAtRun, "at run time the mount source must be the materialized Root holding repo/")
	assert.Equal(t, "packets-cage:dev", fake.got.Image)

	for name, want := range map[string]string{
		"HOME": "/work", "GOCACHE": "/work/gocache", "GOTMPDIR": "/work/gotmp",
		"GOPATH": "/work/gopath", "TMPDIR": "/work/tmp", "GOTOOLCHAIN": "local", "GOFLAGS": "-mod=mod",
	} {
		got, ok := envValue(fake.got, name)
		assert.Truef(t, ok, "%s must be set in the cage env", name)
		assert.Equalf(t, want, got, "%s env value", name)
	}

	cmd := strings.Join(fake.got.Cmd, " ")
	assert.Contains(t, cmd, "verify-catch")
	assert.Contains(t, cmd, "-repo /work/repo")
	assert.Contains(t, cmd, "-base "+base)
	assert.Contains(t, cmd, "-fix "+fix)
	assert.Contains(t, cmd, "-tip "+tip)
	assert.Contains(t, cmd, "-file adult.go")
	assert.Contains(t, cmd, "-line 2")
}

// The verdict comes from re-deriving the cage's transcript: a catch transcript
// (whose evidence supports it and whose anchor matches the claim) mints a record
// carrying the TRUSTED claim's revs.
func TestCageVerifier_mintsACatchFromAnEvidenceBackedTranscript(t *testing.T) {
	t.Parallel()
	host, base, fix, tip := hostRepoWithThreeRevs(t)
	fake := &fakeRunner{result: sandbox.Result{Output: catchTranscriptJSON(t, "adult.go", 2)}}

	rec, err := cage.CageVerifier(fake, host, "img", 30*time.Second)(claimOver(base, fix, tip))
	require.NoError(t, err)
	require.NotNil(t, rec)
	assert.Equal(t, catch.Catch, rec.Outcome)
	assert.Equal(t, base, rec.BeforeRev)
	assert.Equal(t, fix, rec.AfterRev)
}

// The cage's combined output may carry surrounding log noise around the verdict
// JSON; the verifier must still recover the transcript.
func TestCageVerifier_recoversTheTranscriptFromNoisyOutput(t *testing.T) {
	t.Parallel()
	host, base, fix, tip := hostRepoWithThreeRevs(t)
	noisy := "go: downloading ...\n" + catchTranscriptJSON(t, "adult.go", 2) + "\nexit status 0\n"
	fake := &fakeRunner{result: sandbox.Result{Output: noisy}}

	rec, err := cage.CageVerifier(fake, host, "img", 30*time.Second)(claimOver(base, fix, tip))
	require.NoError(t, err)
	require.NotNil(t, rec, "the verdict must be recovered from output surrounded by log noise")
	assert.Equal(t, catch.Catch, rec.Outcome)
}

// A no-catch transcript mints nothing — and that is not an error.
func TestCageVerifier_mintsNothingForANoCatchTranscript(t *testing.T) {
	t.Parallel()
	host, base, fix, tip := hostRepoWithThreeRevs(t)
	b, _ := json.Marshal(pipe.Transcript{
		Outcome: catch.NoCatch, Path: "adult.go", Line: 2,
		Before: catch.LineState{Inventory: []string{">="}, Survivors: []string{">="}},
		After:  catch.LineState{Inventory: []string{">="}, Survivors: []string{">="}},
	})
	fake := &fakeRunner{result: sandbox.Result{Output: string(b)}}

	rec, err := cage.CageVerifier(fake, host, "img", 30*time.Second)(claimOver(base, fix, tip))
	require.NoError(t, err)
	assert.Nil(t, rec)
}

// Output with no verdict JSON at all (the cage crashed before emitting one) is an
// error, not a silent non-catch — the host must not mint or pass on a missing verdict.
func TestCageVerifier_failsOnOutputWithNoVerdict(t *testing.T) {
	t.Parallel()
	host, base, fix, tip := hostRepoWithThreeRevs(t)
	fake := &fakeRunner{result: sandbox.Result{Output: "panic: something\nexit status 2\n"}}

	rec, err := cage.CageVerifier(fake, host, "img", 30*time.Second)(claimOver(base, fix, tip))
	require.Error(t, err, "output carrying no verdict JSON must be an error")
	assert.Nil(t, rec)
}

// Output that has a JSON object but it does not decode as a transcript (a
// corrupt/garbled verdict) is an error, not a silent non-catch.
func TestCageVerifier_failsOnAnUndecodableVerdict(t *testing.T) {
	t.Parallel()
	host, base, fix, tip := hostRepoWithThreeRevs(t)
	fake := &fakeRunner{result: sandbox.Result{Output: "noise {not valid json, line: \"oops\"} trailing"}}

	rec, err := cage.CageVerifier(fake, host, "img", 30*time.Second)(claimOver(base, fix, tip))
	require.Error(t, err, "a present-but-undecodable verdict object must be an error")
	assert.Nil(t, rec)
}

// If the workdir cannot even be materialized (an unresolvable/forged claim
// revision), the verifier fails fast with that error and mints nothing — and
// must not panic on the nil cleanup.
func TestCageVerifier_propagatesAMaterializeFailure(t *testing.T) {
	t.Parallel()
	host, base, fix, _ := hostRepoWithThreeRevs(t)
	bogus := ledger.ClaimRecord{Target: ledger.Target{
		BaseRev: base, FixRev: fix, TipRev: "0000000000000000000000000000000000000000", Path: "adult.go", Line: 2,
	}}
	fake := &fakeRunner{result: sandbox.Result{Output: catchTranscriptJSON(t, "adult.go", 2)}}

	rec, err := cage.CageVerifier(fake, host, "img", 30*time.Second)(bogus)
	require.Error(t, err, "an unresolvable claim revision must fail before any cage run")
	assert.Nil(t, rec)
	assert.Equal(t, sandbox.Spec{}, fake.got, "the runner must never be invoked when materialization fails")
}

// A runner failure (the launch could not run) propagates as an error — the host
// gets no verdict, mints nothing.
func TestCageVerifier_propagatesARunnerFailure(t *testing.T) {
	t.Parallel()
	host, base, fix, tip := hostRepoWithThreeRevs(t)
	fake := &fakeRunner{err: assertErr{}}

	rec, err := cage.CageVerifier(fake, host, "img", 30*time.Second)(claimOver(base, fix, tip))
	require.Error(t, err)
	assert.Nil(t, rec)
}

type assertErr struct{}

func (assertErr) Error() string { return "runner blew up" }

// The disposable workdir must be reaped after every verification — an unbounded
// claim farm cannot leak a clone-plus-caches per claim. The fake captures the
// mount source (the materialized Root); after the verifier returns it must be gone.
func TestCageVerifier_reapsTheWorkdirAfterVerifying(t *testing.T) {
	t.Parallel()
	host, base, fix, tip := hostRepoWithThreeRevs(t)
	fake := &fakeRunner{result: sandbox.Result{Output: catchTranscriptJSON(t, "adult.go", 2)}}

	_, err := cage.CageVerifier(fake, host, "img", 30*time.Second)(claimOver(base, fix, tip))
	require.NoError(t, err)
	require.Len(t, fake.got.Mounts, 1)

	_, statErr := os.Stat(fake.got.Mounts[0].Source)
	assert.True(t, os.IsNotExist(statErr), "the materialized workdir must be reaped after the run")
}

// The verifier must propagate DeriveCatch's refusal: a transcript whose anchor
// does not match the claim (the cage verified the wrong line) is an error, not a
// silently-minted catch — covering the CageVerifier→DeriveCatch seam.
func TestCageVerifier_propagatesAnAnchorMismatchRefusal(t *testing.T) {
	t.Parallel()
	host, base, fix, tip := hostRepoWithThreeRevs(t)
	// claim anchors adult.go:2, but the cage's transcript reports a different line
	fake := &fakeRunner{result: sandbox.Result{Output: catchTranscriptJSON(t, "adult.go", 99)}}

	rec, err := cage.CageVerifier(fake, host, "img", 30*time.Second)(claimOver(base, fix, tip))
	require.Error(t, err, "a transcript for a different anchor than the claim must be refused")
	assert.Nil(t, rec)
}

// --- integration: the real cage, end to end ---

func requireCageImage(t *testing.T, image string) {
	t.Helper()
	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Skip("docker not available; skipping cage integration test")
	}
	if err := exec.Command("docker", "image", "inspect", image).Run(); err != nil {
		t.Skipf("cage image %q not present (build: docker build -f internal/cage/Dockerfile -t %s .); skipping", image, image)
	}
}

// catchRepo builds a stdlib-only repo with a genuine catch on the `>=` line: the
// base test pins 20/10 but not the boundary 18 (so the >= -> '>' mutant
// survives), the fix test adds the 18 assertion (killing it).
func catchRepo(t *testing.T) (dir, base, fix string) {
	t.Helper()
	dir = t.TempDir()
	runGit(t, dir, "init", "-q")
	runGit(t, dir, "config", "user.email", "t@t")
	runGit(t, dir, "config", "user.name", "t")
	write(t, dir, "go.mod", "module capm\n\ngo 1.23\n")
	write(t, dir, "adult.go", "package capm\n\nfunc Adult(age int) bool { return age >= 18 }\n")
	write(t, dir, "adult_test.go", "package capm\n\nimport \"testing\"\n\nfunc TestAdult(t *testing.T){\n\tif !Adult(20){t.Fatal(\"20\")}\n\tif Adult(10){t.Fatal(\"10\")}\n}\n")
	base = commitAll(t, dir, "weak")
	write(t, dir, "adult_test.go", "package capm\n\nimport \"testing\"\n\nfunc TestAdult(t *testing.T){\n\tif !Adult(20){t.Fatal(\"20\")}\n\tif Adult(10){t.Fatal(\"10\")}\n\tif !Adult(18){t.Fatal(\"18\")}\n}\n")
	fix = commitAll(t, dir, "strong")
	return dir, base, fix
}

func catchClaim(base, fix string) ledger.ClaimRecord {
	return ledger.ClaimRecord{Target: ledger.Target{
		BaseRev: base, FixRev: fix, TipRev: fix, Path: "adult.go", Line: 3,
	}}
}

func TestCageVerifier_verifiesAGenuineCatchInsideTheRealCage(t *testing.T) {
	requireCageImage(t, "packets-cage:dev")
	host, base, fix := catchRepo(t)

	rec, err := cage.CageVerifier(sandbox.DockerRunner{}, host, "packets-cage:dev", 30*time.Second)(catchClaim(base, fix))
	require.NoError(t, err)
	require.NotNil(t, rec, "a strengthened boundary test is a confirmed catch the cage must derive")
	assert.Equal(t, catch.Catch, rec.Outcome)
	assert.Equal(t, base, rec.BeforeRev)
	assert.Equal(t, fix, rec.AfterRev)
}

func TestCageVerifier_mintsNothingForANoCatchInsideTheRealCage(t *testing.T) {
	requireCageImage(t, "packets-cage:dev")
	dir := t.TempDir()
	runGit(t, dir, "init", "-q")
	runGit(t, dir, "config", "user.email", "t@t")
	runGit(t, dir, "config", "user.name", "t")
	write(t, dir, "go.mod", "module capm\n\ngo 1.23\n")
	write(t, dir, "adult.go", "package capm\n\nfunc Adult(age int) bool { return age >= 18 }\n")
	write(t, dir, "adult_test.go", "package capm\n\nimport \"testing\"\n\nfunc TestAdult(t *testing.T){\n\tif !Adult(20){t.Fatal(\"20\")}\n\tif Adult(10){t.Fatal(\"10\")}\n\tif !Adult(18){t.Fatal(\"18\")}\n}\n")
	base := commitAll(t, dir, "already-strong")
	write(t, dir, "extra.go", "package capm\n") // churn below the anchor; test not strengthened
	fix := commitAll(t, dir, "churn")

	rec, err := cage.CageVerifier(sandbox.DockerRunner{}, dir, "packets-cage:dev", 30*time.Second)(catchClaim(base, fix))
	require.NoError(t, err)
	assert.Nil(t, rec, "no strengthening across the revs → no catch → nothing minted")
}
