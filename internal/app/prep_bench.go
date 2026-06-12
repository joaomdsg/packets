package app

import (
	"context"
	"strconv"
	"strings"

	"github.com/go-via/via"
	"github.com/go-via/via/h"
	"github.com/go-via/via/on"

	"github.com/joaomdsg/packets/internal/ledger"
)

// benchCap bounds how many fundable targets the bench shows — the Lead curates the
// next few, not an unbounded list (mirrors RecentDispatches' recency cap).
const benchCap = 5

// renderBench shows the session's fundable work — "the bench" — as a calm list of
// the next targets a Spend can fund (path:line), the FIFO-next marked, so the Lead
// sees what's on deck while compute runs (the dead-air-killer) instead of dispatch
// being a blind auto-pick. Each item is a FUND-THIS button: clicking it sets the
// card's FundTarget to that item's key (on.SetSignal) and fires FundChosen, so the
// Lead dispatches the CHOSEN target — a real curation decision. Returns nil when
// there is no fundable work, so the caller omits it. Capped at benchCap.
func renderBench(c *LiveCard, targets []ledger.Target, annos map[ledger.Target]benchAnno) h.H {
	if len(targets) == 0 {
		return nil
	}
	if len(targets) > benchCap {
		targets = targets[:benchCap]
	}
	items := []h.H{h.Class("bench"), h.Span(h.Class("pk-section-label bench__label"), h.Text("on the bench:"))}
	for i, t := range targets {
		at := t.Path + ":" + strconv.Itoa(t.Line)
		fundLabel := "fund " + at
		marker := ""
		if i == 0 {
			marker = " (next)" // the FIFO head a plain Spend would also fund
		}
		card := []h.H{
			h.Class("pk-card bench__item"),
			h.Data("target", at),
			// The header carries the target identity + the fund affordance — the chip's
			// original FundChosen wiring is unchanged (flow d), now the card's head.
			h.Div(h.Class("bench__head"),
				h.Span(h.Class("bench__target"), h.Text(at+marker)),
				h.Button(
					on.Click(c.FundChosen, on.SetSignal(&c.FundTarget.Signal, at)),
					h.Class("pk-btn--quiet bench__fund"),
					h.Text(fundLabel),
				),
			),
		}
		if anno, ok := annos[t]; ok {
			card = append(card, renderBenchAnno(anno))
		}
		card = append(card, renderSharpen(c, at))
		items = append(items, h.Div(card...))
	}
	return h.Div(items...)
}

// benchAnno is the criteria/convention a target has been sharpened with — folded
// from the worefine facts for the card body (splits are NOT annotations; a split
// target is already replaced in the fundable set by its sub-targets).
type benchAnno struct {
	Criteria []string
	Note     string
}

// benchAnnotations folds the criteria/convention refinements per target — a pure
// projection of the log, so it survives a reopen. Splits are skipped (they change
// the fundable set, not the card body).
func benchAnnotations(log *ledger.Log) map[ledger.Target]benchAnno {
	refs, err := log.Refinements()
	if err != nil {
		return nil
	}
	out := map[ledger.Target]benchAnno{}
	for _, r := range refs {
		switch r.Refine {
		case "criteria":
			a := out[r.Target]
			a.Criteria = append(a.Criteria, r.Criteria...)
			out[r.Target] = a
		case "convention":
			a := out[r.Target]
			a.Note = r.Note // last-writer-wins, mirroring the split fold
			out[r.Target] = a
		}
	}
	return out
}

// renderBenchAnno shows a target's attached criteria/convention as calm dim lines on
// the card, so a sharpening the Lead made is visible (and, being logged, persists).
func renderBenchAnno(a benchAnno) h.H {
	lines := []h.H{h.Class("bench__anno")}
	for _, cr := range a.Criteria {
		lines = append(lines, h.Div(h.Class("bench__anno-item"), h.Text("✓ "+cr)))
	}
	if a.Note != "" {
		lines = append(lines, h.Div(h.Class("bench__anno-item"), h.Text("convention: "+a.Note)))
	}
	return h.Div(lines...)
}

// renderSharpen is the collapsible sharpen body: a native <details> disclosure
// (keyboard-accessible, stripped-CSS legible, no client JS) holding an acceptance-
// criteria input bound to RefineText and the two refine submits (criteria /
// convention). Each submit sets RefineTarget+RefineKind then fires RefineChosen.
func renderSharpen(c *LiveCard, at string) h.H {
	return h.Details(h.Class("bench__sharpen-wrap"),
		h.Summary(h.Class("bench__sharpen"), h.Text("sharpen this work")),
		h.Div(h.Class("bench__body"),
			h.Textarea(
				c.RefineText.Bind(),
				h.Class("pk-input bench__criteria"),
				h.Attr("aria-label", "acceptance criteria, one per line"),
				h.Placeholder("acceptance criteria, one per line"),
			),
			h.Button(
				on.Click(c.RefineChosen, on.SetSignal(&c.RefineTarget.Signal, at), on.SetSignal(&c.RefineKind.Signal, "criteria")),
				h.Class("pk-btn bench__refine"),
				h.Text("attach criteria"),
			),
			h.Button(
				on.Click(c.RefineChosen, on.SetSignal(&c.RefineTarget.Signal, at), on.SetSignal(&c.RefineKind.Signal, "convention")),
				h.Class("pk-btn bench__refine"),
				h.Text("note as convention"),
			),
			// Split is system-PROPOSED (harvested from the diff on click), not free
			// text — so it sets only the target and lets SplitChosen harvest the
			// changed regions to split into.
			h.Button(
				on.Click(c.SplitChosen, on.SetSignal(&c.RefineTarget.Signal, at)),
				h.Class("pk-btn bench__split"),
				h.Text("split into changed regions"),
			),
		),
	)
}

// SplitChosen splits a chosen broad bench target into the changed regions of its own
// diff: it harvests the sub-targets (splitCandidates) and records a split refinement
// into them, which the fundableBacklog fold (brick 2) replaces the parent with. The
// candidates are PROPOSED by the system from the real diff (V§13.5: the system, not
// the Lead's free text, defines the units), so this is proposed-then-accept in one
// click. Off-bench, or a target with no changed regions to split into, is a no-op.
func (c *LiveCard) SplitChosen(ctx *via.Ctx) {
	cfg, log := readLiveState(c.Key)
	if log == nil {
		return
	}
	tgt, ok := chosenFundable(cfg, log, strings.TrimSpace(c.RefineTarget.Read(ctx)))
	if !ok {
		return // not on the bench: a no-op
	}
	subs := splitCandidates(context.Background(), cfg.RepoDir, tgt)
	if len(subs) == 0 {
		return // nothing to split into: a no-op
	}
	if err := log.AppendRefine(ledger.RefinedOrderRecord{Target: tgt, Refine: "split", Splits: subs}); err != nil {
		return
	}
	if refs, err := log.Refinements(); err == nil {
		c.Bench.Write(ctx, strconv.Itoa(len(refs))) // rises on every append → always re-renders
	}
}

// buildRefinement turns the Lead's sharpen inputs into the worefine fact to append,
// or reports false when there is nothing to record. "criteria" becomes one fact per
// non-blank line; "convention" carries the trimmed note. A split is built elsewhere
// (it needs harvested sub-targets, not free text), and an unknown kind or empty
// content is refused so the bench is never polluted with a contentless refinement.
func buildRefinement(tgt ledger.Target, kind, text string) (ledger.RefinedOrderRecord, bool) {
	switch kind {
	case "criteria":
		lines := nonEmptyLines(text)
		if len(lines) == 0 {
			return ledger.RefinedOrderRecord{}, false
		}
		return ledger.RefinedOrderRecord{Target: tgt, Refine: "criteria", Criteria: lines}, true
	case "convention":
		note := strings.TrimSpace(text)
		if note == "" {
			return ledger.RefinedOrderRecord{}, false
		}
		return ledger.RefinedOrderRecord{Target: tgt, Refine: "convention", Note: note}, true
	default:
		return ledger.RefinedOrderRecord{}, false
	}
}

// nonEmptyLines splits text on newlines and returns the trimmed, non-blank lines —
// each acceptance criterion the Lead typed, one per line, blanks dropped.
func nonEmptyLines(text string) []string {
	var out []string
	for _, ln := range strings.Split(text, "\n") {
		if t := strings.TrimSpace(ln); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// RefineChosen sharpens a chosen bench target during dead-air: it appends a worefine
// fact (criteria / convention) for the target the Lead picked, validated against the
// fundable set exactly like FundChosen — so a contentless input, an off-bench target,
// or an unknown kind appends nothing. On success it nudges the Bench cell so the card
// re-renders the sharpening over SSE.
func (c *LiveCard) RefineChosen(ctx *via.Ctx) {
	cfg, log := readLiveState(c.Key)
	if log == nil {
		return
	}
	tgt, ok := chosenFundable(cfg, log, strings.TrimSpace(c.RefineTarget.Read(ctx)))
	if !ok {
		return // not on the bench: a no-op
	}
	rec, ok := buildRefinement(tgt, strings.TrimSpace(c.RefineKind.Read(ctx)), c.RefineText.Read(ctx))
	if !ok {
		return // contentless / unknown kind: a no-op
	}
	if err := log.AppendRefine(rec); err != nil {
		return
	}
	if refs, err := log.Refinements(); err == nil {
		c.Bench.Write(ctx, strconv.Itoa(len(refs))) // rises on every append → always re-renders
	}
}
