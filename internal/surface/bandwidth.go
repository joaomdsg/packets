package surface

import (
	"strconv"

	"github.com/go-via/via/h"
)

// RenderBandwidth renders the earned attention bandwidth — the second meter beside
// the catch balance — as its own row. Like the balance it is a held quantity the
// Lead grows by unblocking work and spends on dispatching it, NOT a percentage
// gauge: data-bandwidth carries the count as a stable marker, and a zero reads
// calm, never as guilt. Distinct from the balance/stock/verdict rows (one row
// never speaks for another).
func RenderBandwidth(bandwidth int) h.H {
	return h.Div(
		h.Class("bandwidth-row"),
		h.Data("state", "bandwidth"),
		h.Data("bandwidth", strconv.Itoa(bandwidth)),
		h.P(h.Class("bandwidth-row__amount"), h.Text("Attention bandwidth: "+strconv.Itoa(bandwidth))),
	)
}
