package surface

import (
	"strconv"

	"github.com/go-via/via/h"
)

// RenderDispatch renders the dispatched-work tally — the count of work-orders a
// Spend has funded — as its own row, the thing a spend BUYS. It is the visible
// other half of the economy loop: a spend drains the balance row AND ticks this
// row up, so the Lead sees the catch go somewhere, not vanish. data-dispatch
// carries the count as a stable marker; RenderDispatch(0) reads calm (a held
// tally before the first spend, not an error or empty chrome). The row is
// distinct from the stock/balance/verdict/land/beat rows — one row never speaks
// for another.
func RenderDispatch(n int) h.H {
	return h.Div(
		h.Class("dispatch-row"),
		h.Data("state", "dispatch"),
		h.Data("dispatch", strconv.Itoa(n)),
		h.P(h.Class("dispatch-row__count"), h.Text("Dispatched: "+strconv.Itoa(n))),
	)
}
