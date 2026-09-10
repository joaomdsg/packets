package cli

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/app"
	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/ledger"
)

func TestDeployVerdict_anExplicitHostCommandWithNoCheckIsItsOwnEvidence(t *testing.T) {
	t.Parallel()

	status, refusal := deployVerdict("deployed", false, nil)
	assert.Equal(t, "deployed", status)
	assert.Empty(t, refusal)

	status, refusal = deployVerdict("regressed", false, nil)
	assert.Equal(t, "regressed", status)
	assert.Empty(t, refusal)
}

func TestDeployVerdict_deployedRefusesWhenTheCheckCommandFails(t *testing.T) {
	t.Parallel()

	status, refusal := deployVerdict("deployed", true, errors.New("exit status 1"))
	assert.Empty(t, status, "a failing check must never append a status")
	assert.NotEmpty(t, refusal, "the refusal explains why nothing was appended")
}

func TestDeployVerdict_deployedProceedsWhenTheCheckCommandPasses(t *testing.T) {
	t.Parallel()

	status, refusal := deployVerdict("deployed", true, nil)
	assert.Equal(t, "deployed", status)
	assert.Empty(t, refusal)
}

func TestDeployVerdict_regressedRefusesWhenTheCheckCommandStillPasses(t *testing.T) {
	t.Parallel()

	status, refusal := deployVerdict("regressed", true, nil)
	assert.Empty(t, status, "a still-passing check contradicts an asserted regression — refuse, don't fabricate")
	assert.NotEmpty(t, refusal)
}

func TestDeployVerdict_regressedProceedsWhenTheCheckCommandFails(t *testing.T) {
	t.Parallel()

	status, refusal := deployVerdict("regressed", true, errors.New("exit status 1"))
	assert.Equal(t, "regressed", status)
	assert.Empty(t, refusal)
}

// The whole point of `packets deployed`: a running (or later-restarted) server
// replaying the SAME session must see the ACK. That only holds if the CLI binds
// the ledger under the identical subject instance every server session uses
// (app.LedgerInstance) — a different instance string writes to a wire subject no
// server projection ever folds, so the ACK is durably invisible. NOT parallel
// (shared fabric directory across the seed/deploy/verify phases).
func TestRunDeploy_theAckLandsWhereARealServerSessionWouldFoldIt(t *testing.T) {
	// Pin the VALUE, not just the export — a fix that exports LedgerInstance under
	// a different string than every server session already binds under would still
	// break this test's premise silently otherwise.
	require.Equal(t, "ledger", app.LedgerInstance,
		"app.LedgerInstance must be the exact token every server session already binds under (internal/app/live.go), not a fresh/independent value")

	ctx := context.Background()
	ledgerBase := filepath.Join(t.TempDir(), "catches")

	// Seed phase: fund and dispatch one work order under the SAME instance a real
	// server session binds under — exactly how -backlog/-live seed a fresh boot.
	f, err := fabric.Start(ctx, ledgerBase+"-fabric")
	require.NoError(t, err)
	seedLog := ledger.Bind(f, "default", app.LedgerInstance)
	require.NoError(t, seedLog.Append(ledger.CatchRecord{Outcome: catch.Catch, Path: "c.go", Line: 1, ReasonTag: "catch"}))
	own := ledger.Target{BaseRev: "ob", FixRev: "of", TipRev: "of", Path: "own.go", Line: 1}
	require.NoError(t, seedLog.AppendSend("d1", ledger.Target{BaseRev: "b", FixRev: "f", TipRev: "f", Path: "alpha.go", Line: 7}, own))
	require.NoError(t, f.Close())

	// The real CLI subcommand — exactly what an operator's deploy hook runs.
	require.NoError(t, runDeploy("deployed", []string{"-ledger", ledgerBase, "-session", "default", "-wo", "1"}, io.Discard))

	// Negative control: the RETIRED "cli" instance must NOT see the ACK — proving
	// the two instance tokens really are distinct, non-overlapping subjects, not
	// just "this test didn't happen to look in the right place."
	fOld, err := fabric.Start(ctx, ledgerBase+"-fabric")
	require.NoError(t, err)
	oldLog := ledger.Bind(fOld, "default", "cli")
	oldViews, err := oldLog.RecentSends(0)
	require.NoError(t, err)
	assert.Empty(t, oldViews, "the retired \"cli\" instance must carry no dispatch at all — confirming it is a genuinely separate subject, not an aliased or shared one")
	require.NoError(t, fOld.Close())

	// Verify phase: reopen fresh under app.LedgerInstance, precisely how a server
	// (re)boot replays the durable log — never the CLI's own binding.
	f2, err := fabric.Start(ctx, ledgerBase+"-fabric")
	require.NoError(t, err)
	defer f2.Close()
	verifyLog := ledger.Bind(f2, "default", app.LedgerInstance)
	views, err := verifyLog.RecentSends(0)
	require.NoError(t, err)
	require.Len(t, views, 1)
	assert.Equal(t, "deployed", views[0].Status,
		"the CLI's ACK must be visible under the exact instance token a real server session folds — a mismatched instance silently strands the ACK on a subject nothing ever reads")
}
