package app

import (
	"strconv"

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
func renderBench(c *LiveCard, targets []ledger.Target) h.H {
	if len(targets) == 0 {
		return nil
	}
	if len(targets) > benchCap {
		targets = targets[:benchCap]
	}
	items := []h.H{h.Class("bench"), h.Span(h.Class("bench__label"), h.Text("on the bench:"))}
	for i, t := range targets {
		at := t.Path + ":" + strconv.Itoa(t.Line)
		label := "fund " + at
		if i == 0 {
			label += " (next)" // the FIFO head a plain Spend would also fund
		}
		items = append(items, h.Button(
			on.Click(c.FundChosen, on.SetSignal(&c.FundTarget.Signal, at)),
			h.Class("bench__item"),
			h.Data("target", at),
			h.Text(label),
		))
	}
	return h.Div(items...)
}
