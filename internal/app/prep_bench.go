package app

import (
	"strconv"

	"github.com/go-via/via/h"

	"github.com/joaomdsg/packets/internal/ledger"
)

// benchCap bounds how many fundable targets the bench shows — the Lead curates the
// next few, not an unbounded list (mirrors RecentDispatches' recency cap).
const benchCap = 5

// renderBench shows the session's fundable work — "the bench" — as a calm list of
// the next targets a Spend can fund (path:line), the FIFO-next marked, so the Lead
// sees what's on deck while compute runs (the dead-air-killer) instead of dispatch
// being a blind auto-pick. Read-only in this slice; a later slice makes each item a
// fund-this choice. Returns nil when there is no fundable work, so the caller omits
// it. Capped at benchCap.
func renderBench(targets []ledger.Target) h.H {
	if len(targets) == 0 {
		return nil
	}
	if len(targets) > benchCap {
		targets = targets[:benchCap]
	}
	items := []h.H{h.Class("bench"), h.Span(h.Class("bench__label"), h.Text("on the bench:"))}
	for i, t := range targets {
		at := t.Path + ":" + strconv.Itoa(t.Line)
		label := at
		if i == 0 {
			label += " (next)" // the FIFO head a plain Spend would fund
		}
		items = append(items, h.Span(
			h.Class("bench__item"),
			h.Data("target", at),
			h.Text(label),
		))
	}
	return h.Div(items...)
}
