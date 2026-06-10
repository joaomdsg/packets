package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/go-via/via"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/ledger"
)

// confirmingVerifier is a stub host verifier standing in for the sandboxed
// CageVerifier (whose real path is locked by the equivalence lock): it confirms
// every claim into a catch keyed to that claim's target, executing no code.
func confirmingVerifier(c ledger.ClaimRecord) (*ledger.CatchRecord, error) {
	t := c.Target
	return ledger.NewCatchRecord(catch.Catch, t.Path, t.Line, t.BaseRev, t.FixRev, nil, nil, false, false), nil
}

func claimConsumerServer(t *testing.T) (*httptest.Server, *ledger.Log) {
	t.Helper()
	defLogPath := filepath.Join(t.TempDir(), "default.jsonl")
	var server *httptest.Server
	_, log, err := NewServer(LiveConfig{
		RepoDir: ".", BaseRev: "b", FixRev: "f", TipRev: "f", Anchor: anchorForCap(),
		TestCmd: []string{"true"}, LedgerPath: defLogPath,
	}, via.WithTestServer(&server))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })
	return server, log
}

// The live server actually DRAINS posted claims through its configured verifier
// and mints: a producer POSTs a claim, the server-spawned consumer runs the
// verifier, and a confirmed catch appears in the economy while the target leaves
// the in-flight set. This proves the whole route→publish→consume→verify→mint path
// with a stub verifier — so the only Docker-needing part is the real CageVerifier
// (locked separately by the equivalence lock), and the server wiring is covered here.
func TestStartClaimConsumers_drainsAPostedClaimThroughTheServerVerifierToAMint(t *testing.T) {
	server, log := claimConsumerServer(t)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	StartClaimConsumers(ctx, func(LiveConfig) ledger.Verifier { return confirmingVerifier }, 30*time.Second, nil)

	resp, err := http.Post(server.URL+"/claim", "application/json", strings.NewReader(validClaimBody))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusAccepted, resp.StatusCode)

	require.Eventually(t, func() bool {
		b, err := log.Balance()
		return err == nil && b == 1
	}, 3*time.Second, 20*time.Millisecond, "the server's consumer must verify the posted claim and mint a catch")
	inflight, err := log.ClaimsInFlight()
	require.NoError(t, err)
	require.Equal(t, 0, inflight, "a minted claim has left the in-flight set")
}

// A claim the verifier does NOT confirm never becomes a confirmed score: it
// mints nothing (the two-scores invariant at runtime). A rejecting verdict must
// not move the confirmed economy. Post-C3a it also writes a durable rejection
// marker, so the resolved target leaves the in-flight set rather than lingering
// forever — rejected is "resolved, not confirmed", not "still pending".
func TestStartClaimConsumers_aRejectedClaimNeverBecomesAConfirmedScore(t *testing.T) {
	server, log := claimConsumerServer(t)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	var invoked atomic.Int32
	reject := func(ledger.ClaimRecord) (*ledger.CatchRecord, error) {
		invoked.Add(1)
		return nil, nil
	}
	StartClaimConsumers(ctx, func(LiveConfig) ledger.Verifier { return reject }, 30*time.Second, nil)

	resp, err := http.Post(server.URL+"/claim", "application/json", strings.NewReader(validClaimBody))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusAccepted, resp.StatusCode)

	// The consumer must actually RUN the verifier (not a no-op) — and the verdict
	// being "no catch" must mint nothing. Asserting the verifier was invoked makes
	// the no-mint result meaningful, not vacuous.
	require.Eventually(t, func() bool { return invoked.Load() >= 1 },
		3*time.Second, 20*time.Millisecond, "the server consumer must invoke the verifier on the posted claim")
	require.Never(t, func() bool {
		b, err := log.Balance()
		return err == nil && b > 0
	}, 1*time.Second, 50*time.Millisecond, "a verifier that does not confirm must mint nothing — a pending bet is never a confirmed score")
	// Post-C3a: the no-catch verdict writes a durable rejection marker, so the
	// resolved target leaves the in-flight set (it is not pending — it lost).
	require.Eventually(t, func() bool {
		inflight, err := log.ClaimsInFlight()
		return err == nil && inflight == 0
	}, 3*time.Second, 20*time.Millisecond, "a rejected claim resolves OUT of flight via its durable rejection marker")
}
