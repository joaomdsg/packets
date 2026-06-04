// Package app is the host-side wire of the review loop (DESIGN §6, §17): it
// composes the settle/oracle/catch pipe, the surface presenter, and the catch
// ledger into the single seam the live server drives. Resolve is that seam — it
// turns two revisions into a card verdict plus the record to persist.
package app

import (
	"context"

	"github.com/joaomdsg/agntpr/internal/catch"
	"github.com/joaomdsg/agntpr/internal/ledger"
	"github.com/joaomdsg/agntpr/internal/pipe"
	"github.com/joaomdsg/agntpr/internal/reanchor"
	"github.com/joaomdsg/agntpr/internal/surface"
)

// Resolution is the outcome of resolving one catch cycle for the surface: the
// verdict token a ReviewCard renders, and the ledger record to append — non-nil
// only when a real catch was minted (nil means there is nothing to persist).
type Resolution struct {
	Verdict string
	Record  *ledger.CatchRecord
}

// Resolve runs the catch cycle over the two revisions and maps it for the
// surface and the ledger. The verdict is derived through surface.PresentVerdict
// (so a verified-strong line reads as Tested, not blind no-signal); a record is
// produced only for a confirmed catch (ledger.ShouldRecord), capturing the
// mint-time facts — including the self-flag and would-have-shipped bits the
// caller supplies — that cannot be reconstructed later. The caller appends the
// record to the ledger; Resolve performs no log I/O of its own.
func Resolve(ctx context.Context, repoDir, baseRev, fixRev string, anchor reanchor.Anchor, testCmd []string, selfFlagged, wouldHaveShipped bool) (Resolution, error) {
	res, err := pipe.RunCatchCycle(ctx, repoDir, baseRev, fixRev, fixRev, anchor, testCmd)
	if err != nil {
		return Resolution{}, err
	}

	verdict := surface.PresentVerdict(false, res.Outcome, res.Reason, len(res.After.Inventory), len(res.After.Survivors))

	var record *ledger.CatchRecord
	if ledger.ShouldRecord(res.Outcome) {
		record = &ledger.CatchRecord{
			Outcome:           res.Outcome,
			Path:              res.Path,
			Line:              res.Line,
			BeforeRev:         baseRev,
			AfterRev:          fixRev,
			BeforeInventory:   res.Before.Inventory,
			AfterInventory:    res.After.Inventory,
			MutantsConsidered: len(res.After.Inventory),
			ReasonTag:         string(catch.Catch),
			SelfFlagged:       selfFlagged,
			WouldHaveShipped:  wouldHaveShipped,
		}
	}
	return Resolution{Verdict: verdict, Record: record}, nil
}
