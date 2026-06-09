package app_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/app"
	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/reanchor"
)

func knownCatchRepo(t *testing.T) (dir, base, fix string) {
	t.Helper()
	dir = initRepo(t)
	write(t, dir, "go.mod", "module adultapp\n\ngo 1.23\n")
	write(t, dir, "adult.go", adultGo)
	write(t, dir, "adult_test.go", weakTest)
	base = commitAll(t, dir, "base")
	write(t, dir, "adult_test.go", strongTest)
	fix = commitAll(t, dir, "strengthen the test")
	return dir, base, fix
}

func claimFor(base, fix string) ledger.ClaimRecord {
	return ledger.ClaimRecord{Target: ledger.Target{
		BaseRev:  base,
		FixRev:   fix,
		TipRev:   fix,
		Path:     "adult.go",
		Line:     4,
		LineHash: reanchor.HashLines("\treturn age >= 18"),
	}}
}

func TestInProcVerifier_returnsACatchRecordForAConfirmedClaim(t *testing.T) {
	t.Parallel()
	dir, base, fix := knownCatchRepo(t)

	rec, err := app.InProcVerifier(dir, goTestCmd)(claimFor(base, fix))
	require.NoError(t, err)
	require.NotNil(t, rec, "a strengthened test is a confirmed catch — the verifier must mint a record")
	assert.Equal(t, catch.Catch, rec.Outcome)
	assert.Equal(t, base, rec.BeforeRev)
	assert.Equal(t, fix, rec.AfterRev)
}

func TestInProcVerifier_returnsNilForANoCatchClaim(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	write(t, dir, "go.mod", "module adultapp\n\ngo 1.23\n")
	write(t, dir, "adult.go", adultGo)
	write(t, dir, "adult_test.go", weakTest)
	base := commitAll(t, dir, "base")
	write(t, dir, "adult.go", adultPadded) // churn below the anchor; the test is NOT strengthened
	fix := commitAll(t, dir, "no strengthening")

	rec, err := app.InProcVerifier(dir, goTestCmd)(claimFor(base, fix))
	require.NoError(t, err)
	assert.Nil(t, rec, "no strengthening → the mutant survives both revs → no catch → nothing to mint")
}

func TestInProcVerifier_mintsOnceThroughTheClaimConsumerAndDedupesAReplay(t *testing.T) {
	t.Parallel()
	dir, base, fix := knownCatchRepo(t)

	f, err := fabric.Start(context.Background(), t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	log := ledger.Bind(f, "s", "i")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = log.ConsumeClaims(ctx, app.InProcVerifier(dir, goTestCmd)) }()

	balance := func() int { b, _ := log.Balance(); return b }

	require.NoError(t, mustPublish(ctx, f, claimFor(base, fix)))
	require.Eventually(t, func() bool { return balance() == 1 },
		20*time.Second, 50*time.Millisecond, "a confirmed claim must mint exactly one catch through the consumer")

	require.NoError(t, mustPublish(ctx, f, claimFor(base, fix))) // replay the same claim
	require.Never(t, func() bool { return balance() != 1 },
		1*time.Second, 50*time.Millisecond, "a replayed claim reproduces the same identity → mints nothing more")
}

func mustPublish(ctx context.Context, f *fabric.Fabric, c ledger.ClaimRecord) error {
	_, err := ledger.PublishClaim(ctx, f, "s", "i", c)
	return err
}
