package app

import (
	"context"
	"time"

	"github.com/joaomdsg/packets/internal/ingest"
)

// PruneIdleProducers makes one housekeeping pass over the registry, reclaiming
// each idle producer's ingested git objects without ever orphaning a pending
// claim. Per session it asks ingest.PruneProducerObjects to delete the producer's
// refs/producers/<key>/* namespace ONLY when that session has no claims in flight
// (the economy-safe retention rule, council R39): a producer's ingested objects
// back its claims' revisions, so they must survive while any verify is pending.
//
// TOCTOU (council R39, accepted): the ClaimsInFlight read and the ref-delete are
// not atomic against concurrent ingest/claim/verify on the same session. The
// economy stays safe regardless — pruning can only ever make a revision
// unresolvable, never mint a wrong catch:
//   - A CLAIMED target is durably in-flight (its claim is appended before any
//     verify reads its objects), so ClaimsInFlight()>0 keeps the whole namespace;
//     a verify can never race a delete of the objects it is mid-reading.
//   - The only live window is upload-without-yet-claim: a producer that POSTs
//     /bundle but has not yet POSTed /claim has ClaimsInFlight()==0, so a prune
//     tick landing in that sub-second gap reclaims the just-uploaded objects as
//     dead weight (by design — see ingest.PruneProducerObjects). The subsequent
//     claim's revs then fail to resolve and the claim is durably rejected; the
//     producer simply re-uploads + re-claims. At the 10m housekeeping cadence this
//     collision is astronomically rare and self-healing, so we take NO lock.
//
// It is best-effort and fail-safe-toward-keeping: a ClaimsInFlight read error
// SKIPS that session (a read failure must never trigger a delete — degrade like
// the board's projections), an empty RepoDir is skipped (no store; never fall
// back to the process cwd), and a prune failure on one session never stops the
// others. It takes no lock — liveReg is a sync.Map and each prune touches only
// its own session's repo.
func PruneIdleProducers(ctx context.Context) {
	liveReg.Range(func(k, v any) bool {
		pruneProducerIfIdle(ctx, k.(string), v.(*liveEntry))
		return true
	})
}

// pruneProducerIfIdle reclaims one session's ingested objects when it has no
// claims in flight, applying the same economy-safe retention + fail-toward-keep
// rules as the sweep (see PruneIdleProducers). It is the unit both the periodic
// sweep and the post-verdict hook (ConsumeClaims' OnResolved) share, so a claim
// resolving reclaims its producer's objects immediately rather than only at the
// next tick.
func pruneProducerIfIdle(ctx context.Context, key string, e *liveEntry) {
	if e == nil || e.cfg.RepoDir == "" || e.log == nil {
		return // no store to prune
	}
	inFlight, err := e.log.ClaimsInFlight()
	if err != nil {
		return // never prune on a read error
	}
	_, _ = ingest.PruneProducerObjects(ctx, e.cfg.RepoDir, key, inFlight > 0)
}

// pruneProducerSession looks the session up in the registry and prunes it if
// idle — the post-verdict hook's entry point, which has only the session key.
func pruneProducerSession(ctx context.Context, key string) {
	if v, ok := liveReg.Load(key); ok {
		pruneProducerIfIdle(ctx, key, v.(*liveEntry))
	}
}

// StartProducerGC runs PruneIdleProducers every interval until ctx is cancelled —
// the background housekeeping that bounds the ingested-object store. The caller
// owns ctx (cancelling it stops the sweep); call once after all sessions are
// registered.
func StartProducerGC(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				PruneIdleProducers(ctx)
			}
		}
	}()
}
