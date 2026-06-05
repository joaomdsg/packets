package surface

import (
	"strconv"

	"github.com/go-via/via/h"

	"github.com/joaomdsg/packets/internal/ledger"
)

// RenderDispatch renders the dispatched-work tally — split by status so it MOVES
// queued→running→done as a funded order runs, not merely rises. It is the visible
// other half of the economy loop: a spend drains the balance row AND funds a
// queued order here; the order runs (running) and pays back (done), so the Lead
// watches the catch become work and the work pay off. The three counts carry as
// stable markers; all-zero reads calm (a held tally before the first spend). The
// row is distinct from the stock/balance/verdict/land/beat rows — one row never
// speaks for another.
func RenderDispatch(c ledger.DispatchCounts) h.H {
	return h.Div(
		h.Class("dispatch-row"),
		h.Data("state", "dispatch"),
		h.Data("dispatch-queued", strconv.Itoa(c.Queued)),
		h.Data("dispatch-running", strconv.Itoa(c.Running)),
		h.Data("dispatch-done", strconv.Itoa(c.Done)),
		h.P(h.Class("dispatch-row__counts"), h.Text(
			"Dispatched: "+strconv.Itoa(c.Queued)+" queued, "+
				strconv.Itoa(c.Running)+" running, "+strconv.Itoa(c.Done)+" done")),
	)
}
