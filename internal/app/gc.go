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
		e := v.(*liveEntry)
		if e.cfg.RepoDir == "" || e.log == nil {
			return true // no store to prune
		}
		inFlight, err := e.log.ClaimsInFlight()
		if err != nil {
			return true // never prune on a read error
		}
		_, _ = ingest.PruneProducerObjects(ctx, e.cfg.RepoDir, k.(string), inFlight > 0)
		return true
	})
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
