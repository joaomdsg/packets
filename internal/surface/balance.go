package surface

import (
	"strconv"

	"github.com/go-via/via/h"
)

// RenderBalance renders the spendable balance — confirmed catches minus spends —
// as its own row, the read side of the economy's sink. It is a held quantity the
// Lead can act on (spend), not a percentage gauge: data-balance carries the count
// as a stable marker, and a spent-down zero reads calm, never as guilt. The row
// is distinct from the stock/verdict/land/beat rows (one row never speaks for
// another).
func RenderBalance(balance int) h.H {
	return h.Div(
		h.Class("pk-card balance-row"),
		h.Data("state", "balance"),
		h.Data("balance", strconv.Itoa(balance)),
		h.P(h.Class("balance-row__amount"), h.Text("Balance: "+strconv.Itoa(balance))),
	)
}
