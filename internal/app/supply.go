package app

import (
	"strconv"

	"github.com/joaomdsg/packets/internal/ledger"
)

// nextUnconsumedTarget returns the first backlog target a Spend can still fund —
// head-first (FIFO), skipping targets already CONSUMED (carried by a funded
// work-order, projected purely from the log so it survives a reopen) and the
// card's OWN caught cycle (which AppendDispatch would refuse, so leaving it in
// would stall the head forever, starving the targets behind it). ok=false when
// the backlog is empty or fully drawn down — the honest scarcity signal.
func nextUnconsumedTarget(cfg LiveConfig, log *ledger.Log) (ledger.Target, bool) {
	if f := fundableBacklog(cfg, log); len(f) > 0 {
		return f[0], true
	}
	return ledger.Target{}, false
}

// fundableBacklog is the targets a Spend can still fund, in head-first order: the
// hand-seeded config list THEN the card's own catches' neighborhoods (from-catch
// supply), each filtered to those NOT yet CONSUMED (carried by a funded work-order,
// projected purely from the log) and NOT the card's OWN caught cycle (which
// AppendDispatch refuses). Its length is the board's BacklogRemaining — the honest
// "distinct work left." A drained config still yields derived candidates, so supply
// is a going concern; it returns empty only when neither source has fundable work.
func fundableBacklog(cfg LiveConfig, log *ledger.Log) []ledger.Target {
	consumed := map[ledger.Target]bool{}
	if orders, err := log.WorkOrders(); err == nil {
		for _, o := range orders {
			consumed[o.Target] = true
		}
	}
	own := ownTargetOf(cfg)
	seen := map[ledger.Target]bool{}
	var out []ledger.Target
	// The config list FIRST (the Lead's hand-seeded targets), then the card's own
	// catches' neighborhoods — so a drained config keeps refilling from the card's
	// own output. The same filter applies to both sources: never re-fund a consumed
	// target, never the card's own caught cycle, never a duplicate within one call.
	for _, t := range append(append([]ledger.Target{}, cfg.DispatchBacklog...), candidatesFromCatches(log)...) {
		if t != own && !consumed[t] && !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	return out
}

// candidatesFromCatches derives CANDIDATE work from the card's own confirmed
// catches — the from-catch supply that turns the loop into a going concern. Each
// catch (at base→fix, anchored at Path:Line) proposes a candidate one line
// FORWARD (same revs, Line+1): a real oracle QUESTION in the catch's neighborhood,
// never a guaranteed mint. The oracle judges it — many neighbors yield no catch (an
// honest loss). A candidate that mints seeds the next candidate, so supply refills
// from its own output; a candidate that misses (or reproduces a seen identity, which
// the dedup gate refuses) is a dead end. It is a PURE projection of the log — the
// candidates are derived on read, never stored. Distinct identity from the catch
// (Line+1), so it is never a self-recall of already-caught ground.
func candidatesFromCatches(log *ledger.Log) []ledger.Target {
	recs, err := log.Records()
	if err != nil {
		return nil
	}
	var out []ledger.Target
	for _, r := range recs {
		out = append(out, ledger.Target{
			BaseRev: r.BeforeRev, FixRev: r.AfterRev, TipRev: r.AfterRev,
			Path: r.Path, Line: r.Line + 1,
		})
	}
	return out
}

// spendButtonLabel names what the NEXT Spend will fund — the actual target
// nextUnconsumedTarget would pick — so the Lead knows what they are buying before
// clicking, not a blind verb. It falls back to a generic label when there is no
// fundable target; that case is near-unreachable in practice (supply refills from
// the session's own catches, so a session with balance almost always has work),
// but the fallback keeps the control honest if it ever is reached.
func spendButtonLabel(cfg LiveConfig, log *ledger.Log) string {
	if log != nil {
		if t, ok := nextUnconsumedTarget(cfg, log); ok {
			return "Spend a catch → fund " + t.Path + ":" + strconv.Itoa(t.Line)
		}
	}
	return "Spend a catch → fund a work-order"
}

// ownTargetOf is the card's OWN caught cycle as a Target — what a dispatch must
// NOT re-run (it is already caught; re-running it mints nothing).
func ownTargetOf(cfg LiveConfig) ledger.Target {
	return ledger.Target{
		BaseRev: cfg.BaseRev, FixRev: cfg.FixRev, TipRev: cfg.TipRev,
		Path: cfg.Anchor.Path, Line: cfg.Anchor.Start, LineHash: cfg.Anchor.LineHash,
	}
}
