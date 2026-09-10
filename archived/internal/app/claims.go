package app

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/joaomdsg/packets/internal/cage"
	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/sandbox"
)

// The claim-verification governor, as plain values so the coupling is auditable:
//   - cageVerifyTimeout bounds a single cage verify (context deadline).
//   - claimAckWait is the durable consumer's redelivery window. It MUST outlast
//     cageVerifyTimeout: the consumer acks AFTER the verify returns, so a slow but
//     legal verify (up to the deadline) must finish and ack before redelivery, or
//     the same claim runs twice concurrently in two cages.
//   - claimBurst / claimRatePerSec throttle a single peer's claim flood.
const (
	cageVerifyTimeout = 120 * time.Second
	claimAckWait      = 240 * time.Second
	claimBurst        = 12.0
	claimRatePerSec   = 0.1
)

// claimConcurrency is the process-wide cap on simultaneous cage verifies. Each
// cage run reserves roughly one CPU and 256m, so it is bounded to leave the host
// responsive — half the cores, never fewer than 1 (a zero-capacity semaphore
// would deadlock every verify) nor more than 4 (so the fleet never oversubscribes).
func claimConcurrency() int { return max(1, min(4, runtime.NumCPU()/2)) }

// StartCageClaimConsumers spawns one durable claim consumer per registered
// session, each verifying claims in the hardened Docker cage via the injected
// runner (production passes sandbox.DockerRunner{}; tests fake it at this seam).
// It applies the shared governor: a per-peer token bucket plus a process-wide
// concurrency semaphore. Call this once after the boot sessions are registered; it
// also arms registerSession so any session created later (a runtime-created
// session) gets its own cage consumer immediately.
func StartCageClaimConsumers(ctx context.Context, image string, runner sandbox.Runner) {
	adm := &ledger.Admission{
		Burst:       claimBurst,
		RatePerSec:  claimRatePerSec,
		Concurrency: make(chan struct{}, claimConcurrency()),
		// Post-verdict GC: the instant a claim resolves, reclaim its peer's
		// objects if it has no other claim in flight — prompt reclamation on top of
		// the periodic StartPeerGC sweep. Uses the consumer ctx so
		// a shutdown cancels an in-progress prune.
		OnResolved: func(session string) { consumerSpawner.noteActivity(session); prunePeerSession(ctx, session) },
	}
	verifierFor := func(cfg LiveConfig) ledger.Verifier {
		return cage.CageVerifier(runner, cfg.RepoDir, image, cageVerifyTimeout)
	}
	cageGauntlet.mu.Lock()
	cageGauntlet.verifierFor = verifierFor
	cageGauntlet.mu.Unlock()
	StartClaimConsumers(ctx, verifierFor, claimAckWait, adm)
}

// cageGauntlet is the process-wide cage wiring gauntletFor needs to re-derive
// G6 (IndependentCheck) for a filled packet — set once by
// StartCageClaimConsumers, left nil until then. It reuses the SAME
// verifierFor factory the claim consumers run, so a G6 re-derivation exercises
// the identical CageVerifier (repo dir, image, timeout) production wires for
// claims, rather than a second, possibly-drifted construction. Separate from
// claimConsumerSpawner because G6 must answer even for a session whose claim
// consumer hasn't been (or will never be) spawned — the two are related but
// distinct concerns (background verification vs render-time re-derivation).
var cageGauntlet struct {
	mu          sync.Mutex
	verifierFor func(LiveConfig) ledger.Verifier
}

// cageVerifierFor returns cfg's cage-backed ledger.Verifier and true when
// StartCageClaimConsumers has configured cage wiring for this process; false
// when cage has never been wired, so the caller can fall back to G6's honest
// not-run default rather than fabricate a measurement.
func cageVerifierFor(cfg LiveConfig) (ledger.Verifier, bool) {
	cageGauntlet.mu.Lock()
	vf := cageGauntlet.verifierFor
	cageGauntlet.mu.Unlock()
	if vf == nil {
		return nil, false
	}
	return vf(cfg), true
}

// resetCageGauntletForTest clears the process-global cage wiring so a prior
// test's StartCageClaimConsumers call can't leak G6 measurements into a later
// test that never configured cage itself. Mirrors resetConsumersForTest.
func resetCageGauntletForTest() {
	cageGauntlet.mu.Lock()
	cageGauntlet.verifierFor = nil
	cageGauntlet.mu.Unlock()
}
