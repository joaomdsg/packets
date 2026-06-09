package app

import (
	"context"
	"runtime"
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
//   - claimBurst / claimRatePerSec throttle a single producer's claim flood.
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
// It applies the shared governor: a per-producer token bucket plus a process-wide
// concurrency semaphore. Per the StartClaimConsumers contract, call this EXACTLY
// ONCE, AFTER every session is registered.
func StartCageClaimConsumers(ctx context.Context, image string, runner sandbox.Runner) {
	adm := &ledger.Admission{
		Burst:       claimBurst,
		RatePerSec:  claimRatePerSec,
		Concurrency: make(chan struct{}, claimConcurrency()),
	}
	verifierFor := func(cfg LiveConfig) ledger.Verifier {
		return cage.CageVerifier(runner, cfg.RepoDir, image, cageVerifyTimeout)
	}
	StartClaimConsumers(ctx, verifierFor, claimAckWait, adm)
}
