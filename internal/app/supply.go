package app

import (
	"context"
	"strconv"

	"github.com/joaomdsg/packets/internal/diff"
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
	splits := splitRefinements(log)
	own := ownTargetOf(cfg)
	seen := map[ledger.Target]bool{}
	var out []ledger.Target
	// The same filter applies to every fundable target regardless of source or
	// whether it came from a split: never re-fund a consumed target, never the
	// card's own caught cycle, never a duplicate within one call.
	add := func(t ledger.Target) {
		if t != own && !consumed[t] && !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	// The config list FIRST (the Lead's hand-seeded targets), then the card's own
	// catches' neighborhoods — so a drained config keeps refilling from the card's
	// own output. A target the Lead SPLIT is replaced in place by its sub-targets
	// (the dead-air sharpening, folded on read); criteria/convention refinements
	// annotate the card body and are not folded here.
	for _, t := range append(append([]ledger.Target{}, cfg.DispatchBacklog...), candidatesFromCatches(log)...) {
		if subs, ok := splits[t]; ok {
			for _, s := range subs {
				add(s)
			}
			continue
		}
		add(t)
	}
	return out
}

// splitCandidates proposes the sub-targets a broad target could be SPLIT into,
// harvested from the change itself: it diffs the target's BaseRev→FixRev and
// proposes each changed hunk's new-side start line (same revs, same file) as a
// distinct sub-target, excluding the parent's own line. This is the
// "proposed-then-accept" source — the system proposes from the real diff, never a
// fabricated split. Returns nil when the target can't be diffed (no repo / no revs)
// or its file is unchanged. Capped at benchCap. NOT called per render (a git diff
// is too costly for the SSE hot path) — only when the Lead asks to split (the
// SplitChosen action).
func splitCandidates(ctx context.Context, repoDir string, t ledger.Target) []ledger.Target {
	if repoDir == "" || t.BaseRev == "" || t.FixRev == "" {
		return nil
	}
	d, err := diff.Compute(ctx, repoDir, t.BaseRev, t.FixRev)
	if err != nil {
		return nil
	}
	seen := map[int]bool{t.Line: true} // never propose the parent's own line
	var out []ledger.Target
	for _, f := range d.Files {
		if f.Path != t.Path {
			continue
		}
		for _, h := range f.Hunks {
			line := h.NewStart
			if line <= 0 || seen[line] {
				continue
			}
			seen[line] = true
			out = append(out, ledger.Target{
				BaseRev: t.BaseRev, FixRev: t.FixRev, TipRev: t.TipRev,
				Path: t.Path, Line: line,
			})
			if len(out) >= benchCap {
				return out
			}
		}
	}
	return out
}

// splitRefinements maps each split-refined target to the sub-targets it was
// sharpened into (the last split per parent wins, mirroring the append-only
// last-writer status semantics). Only "split" refinements change the fundable set;
// criteria/convention annotate the card body, so they are not folded here. A pure
// projection of the log — derived on read, never stored.
func splitRefinements(log *ledger.Log) map[ledger.Target][]ledger.Target {
	refs, err := log.Refinements()
	if err != nil {
		return nil
	}
	out := map[ledger.Target][]ledger.Target{}
	for _, r := range refs {
		// An empty split is degenerate (the proposed-then-accept UI never emits it);
		// folding it would silently ERASE the parent from the fundable set, so ignore
		// it and leave the parent fundable — losing real work is worse.
		if r.Refine == "split" && len(r.Splits) > 0 {
			out[r.Target] = r.Splits
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
