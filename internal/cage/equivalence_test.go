package cage_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/app"
	"github.com/joaomdsg/packets/internal/cage"
	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/reanchor"
	"github.com/joaomdsg/packets/internal/sandbox"
)

// equivGoTestCmd mirrors the host-side suite command app's own tests use; `env -u
// GOROOT` keeps the test harness's GOROOT from leaking into the child go test.
var equivGoTestCmd = []string{"env", "-u", "GOROOT", "go", "test", "./..."}

// equivClaim carries the base-line content hash a real producer would, so the
// in-process re-anchor (which trusts Target.LineHash) and the cage (which
// recomputes the hash itself) anchor the SAME line — the comparison is then of
// the verdict, not of an anchoring mismatch.
func equivClaim(base, fix string) ledger.ClaimRecord {
	return ledger.ClaimRecord{Target: ledger.Target{
		BaseRev: base, FixRev: fix, TipRev: fix, Path: "adult.go", Line: 3,
		LineHash: reanchor.HashLines("func Adult(age int) bool { return age >= 18 }"),
	}}
}

// THE LOAD-BEARING GATE (#6c step 5): the sandboxed verdict must be IDENTICAL to
// the in-process one. DeriveCatch alone only checks a transcript is
// self-consistent; this differential lock is what proves the cage's survivor-set
// evidence is REAL — the same oracle over the same revisions yields the same
// CatchRecord whether it ran in-process or in the cage. It must be green before
// the live default may flip to sandboxed verification.
func TestEquivalence_cagedCatchProjectionMatchesInProcess(t *testing.T) {
	requireCageImage(t, "packets-cage:dev")
	host, base, fix := catchRepo(t)
	claim := equivClaim(base, fix)

	inproc, err := app.InProcVerifier(host, equivGoTestCmd)(claim)
	require.NoError(t, err)
	require.NotNil(t, inproc, "the in-process oracle must see this catch — else the corpus is wrong, not the cage")

	caged, err := cage.CageVerifier(sandbox.DockerRunner{}, host, "packets-cage:dev", 30*time.Second)(claim)
	require.NoError(t, err)

	assert.Equal(t, inproc, caged, "the sandboxed catch record must be byte-identical to the in-process one")
}

// The lock must hold on the negative too: a no-catch claim mints nothing in BOTH
// paths — neither over- nor under-counts relative to the other.
func TestEquivalence_cagedNoCatchMatchesInProcess(t *testing.T) {
	requireCageImage(t, "packets-cage:dev")
	host, base, fix := noCatchRepo(t)
	claim := equivClaim(base, fix)

	inproc, err := app.InProcVerifier(host, equivGoTestCmd)(claim)
	require.NoError(t, err)
	require.Nil(t, inproc, "the in-process oracle must see no catch here — else the corpus is wrong")

	caged, err := cage.CageVerifier(sandbox.DockerRunner{}, host, "packets-cage:dev", 30*time.Second)(claim)
	require.NoError(t, err)

	assert.Equal(t, inproc, caged, "both paths must agree there is nothing to mint")
}

// noCatchRepo is the negative corpus: the test already pins the boundary at base,
// so the fix (mere churn below the anchor) strengthens nothing — no catch.
func noCatchRepo(t *testing.T) (dir, base, fix string) {
	t.Helper()
	dir = t.TempDir()
	runGit(t, dir, "init", "-q")
	runGit(t, dir, "config", "user.email", "t@t")
	runGit(t, dir, "config", "user.name", "t")
	write(t, dir, "go.mod", "module capm\n\ngo 1.23\n")
	write(t, dir, "adult.go", "package capm\n\nfunc Adult(age int) bool { return age >= 18 }\n")
	strong := "package capm\n\nimport \"testing\"\n\nfunc TestAdult(t *testing.T){\n\tif !Adult(20){t.Fatal(\"20\")}\n\tif Adult(10){t.Fatal(\"10\")}\n\tif !Adult(18){t.Fatal(\"18\")}\n}\n"
	write(t, dir, "adult_test.go", strong)
	base = commitAll(t, dir, "already-strong")
	write(t, dir, "extra.go", "package capm\n") // churn below the anchor; the test is unchanged
	fix = commitAll(t, dir, "churn")
	return dir, base, fix
}
