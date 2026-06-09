package cage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/pipe"
	"github.com/joaomdsg/packets/internal/sandbox"
)

// cageMount is where the workdir Root is bind-mounted inside the cage. The
// in-container tool paths derive from it and the materialize subdir consts, so
// the launch and the layout share one source of truth.
const cageMount = "/work"

// CageVerifier is the sandboxed ledger.Verifier: it verifies a claim by running
// the SAME `packets verify-catch` oracle inside the hardened cage over a
// disposable copy of the claim's revisions, then re-derives the verdict from the
// emitted transcript (trusting the survivor-set evidence, never the cage's
// self-report). runner is the isolation backend (DockerRunner in production, a
// double in tests); hostRepo is the trusted repo the host holds; image is the
// pinned cage image. It mirrors InProcVerifier's seam but executes untrusted-safe
// in the cage rather than in-process.
func CageVerifier(runner sandbox.Runner, hostRepo, image string) ledger.Verifier {
	return func(c ledger.ClaimRecord) (*ledger.CatchRecord, error) {
		ctx := context.Background()
		wd, cleanup, err := Materialize(ctx, hostRepo, c.Target)
		if err != nil {
			return nil, err
		}
		defer cleanup()

		// The cage runs as uid 65534, but the clone and scratch dirs are created
		// owned by the host uid (git's repo/ is 0755, the cache dirs 0755). Open
		// them so the cage's go toolchain and git can write the workdir.
		if err := chmodTree(wd.Root, 0o777); err != nil {
			return nil, err
		}

		res, err := runner.Run(ctx, buildSpec(wd, c.Target, image))
		if err != nil {
			return nil, err
		}

		transcript, err := parseTranscript(res.Output)
		if err != nil {
			return nil, err
		}
		return DeriveCatch(transcript, c.Target)
	}
}

// buildSpec renders the cage launch for a Workdir + Target: the writable workdir
// mounted at /work, the go toolchain routed at the in-container paths the
// cage-exec layout provides, and the host-fixed verify-catch command over the
// materialized repo. The test command is never agent-supplied.
func buildSpec(wd *Workdir, t ledger.Target, image string) sandbox.Spec {
	in := func(sub string) string { return cageMount + "/" + sub }
	return sandbox.Spec{
		Image:  image,
		Mounts: []sandbox.Mount{{Source: wd.Root, Target: cageMount}}, // writable
		Env: []sandbox.EnvVar{
			{Name: "HOME", Value: cageMount},
			{Name: "GOCACHE", Value: in(subdirGoCache)},
			{Name: "GOTMPDIR", Value: in(subdirGoTmp)},
			{Name: "GOPATH", Value: in(subdirGoPath)},
			{Name: "TMPDIR", Value: in(subdirTmp)},
			{Name: "GOTOOLCHAIN", Value: "local"},
			{Name: "GOFLAGS", Value: "-mod=mod"},
		},
		Cmd: []string{
			"packets", "verify-catch",
			"-repo", in(subdirRepo),
			"-base", t.BaseRev,
			"-fix", t.FixRev,
			"-tip", t.TipRev,
			"-file", t.Path,
			"-line", strconv.Itoa(t.Line),
		},
	}
}

// parseTranscript recovers the verdict Transcript from the cage's combined
// output, which may carry surrounding log noise: it takes the outermost JSON
// object (first '{' to last '}') and decodes it. Output with no object, or one
// that does not decode, is an error — a missing verdict must never read as a
// silent non-catch.
func parseTranscript(output string) (pipe.Transcript, error) {
	start := strings.IndexByte(output, '{')
	end := strings.LastIndexByte(output, '}')
	if start < 0 || end < start {
		return pipe.Transcript{}, fmt.Errorf("cage: no verdict transcript in output: %q", strings.TrimSpace(output))
	}
	var tr pipe.Transcript
	if err := json.Unmarshal([]byte(output[start:end+1]), &tr); err != nil {
		return pipe.Transcript{}, fmt.Errorf("cage: decode transcript: %w", err)
	}
	return tr, nil
}

// chmodTree opens every file and dir under root to mode, so the cage's non-root
// user can write the materialized workdir. root is a fresh disposable dir the
// host just created, so widening its modes exposes nothing of the host.
func chmodTree(root string, mode os.FileMode) error {
	return filepath.Walk(root, func(p string, _ os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		return os.Chmod(p, mode)
	})
}
