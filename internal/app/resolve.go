// Package app is the host-side wire of the review loop (DESIGN §6, §17): it
// composes the settle/oracle/catch pipe, the surface presenter, and the catch
// ledger into the single seam the live server drives. Resolve is that seam — it
// turns two revisions into a card verdict plus the record to persist.
package app

import (
	"context"

	"github.com/joaomdsg/packets/internal/catch"
	"github.com/joaomdsg/packets/internal/ledger"
	"github.com/joaomdsg/packets/internal/pipe"
	"github.com/joaomdsg/packets/internal/reanchor"
	"github.com/joaomdsg/packets/internal/surface"
)

// Resolution is the outcome of resolving one catch cycle for the surface: the
// verdict token a ReviewCard renders, and the ledger record to append — non-nil
// only when a real catch was minted (nil means there is nothing to persist).
type Resolution struct {
	Verdict string
	// Land is the integration verdict (clean/conflict/checks-red) rendered as its
	// own card row, orthogonal to Verdict — the catch is minted on the base, this
	// answers whether it integrates onto trunk tip.
	Land   pipe.LandState
	// Trace is the ordered, typed, timestamped beats of the cycle, surfaced so the
	// live card can stream each as its own SSE patch (the felt loop). Purely
	// additive — it never alters the verdict, Land, or the ledger record.
	Trace  []pipe.TraceEvent
	Record *ledger.CatchRecord
}

// Resolve runs the catch cycle over the two revisions and maps it for the
// surface and the ledger. The verdict is derived through surface.PresentVerdict
// (so a verified-strong line reads as Tested, not blind no-signal); a record is
// produced only for a confirmed catch (ledger.ShouldRecord), capturing the
// mint-time facts — including the self-flag and would-have-shipped bits the
// caller supplies — that cannot be reconstructed later. The caller appends the
// record to the ledger; Resolve performs no log I/O of its own.
func Resolve(ctx context.Context, repoDir, baseRev, fixRev, tipRev string, anchor reanchor.Anchor, testCmd []string, selfFlagged, wouldHaveShipped bool) (Resolution, error) {
	return ResolveStreaming(ctx, repoDir, baseRev, fixRev, tipRev, anchor, testCmd, selfFlagged, wouldHaveShipped, nil)
}

// ResolveStreaming is Resolve with a live beats channel threaded to the cycle, so
// the live card can stream each beat as its own SSE patch (the felt loop). The
// returned Resolution is identical to Resolve's; `beats` (nil for the
// non-streaming path) receives each TraceEvent at its real transition and must be
// buffered or drained concurrently by the caller.
func ResolveStreaming(ctx context.Context, repoDir, baseRev, fixRev, tipRev string, anchor reanchor.Anchor, testCmd []string, selfFlagged, wouldHaveShipped bool, beats chan<- pipe.TraceEvent) (Resolution, error) {
	res, err := pipe.RunCatchCycleStreaming(ctx, repoDir, baseRev, fixRev, tipRev, anchor, testCmd, beats)
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
	return Resolution{Verdict: verdict, Land: res.Land, Trace: res.Trace, Record: record}, nil
}
