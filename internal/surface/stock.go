package surface

import (
	"sort"
	"strconv"

	"github.com/go-via/via/h"

	"github.com/joaomdsg/packets/internal/ledger"
)

// RenderStock renders the confirmed-catch STOCK as its own retrospective row — a
// calm tally of catches that have ALREADY been minted, never a live gauge. It is
// the read side of the ledger (the write→read loop the felt loop left open): a
// count plus per-reason and mint-bit tallies, derived purely from the logged
// records. No meter, no percentage, no number-going-up affordance — a tally of
// facts that already happened cannot induce guilt. data-state="stock" is disjoint
// from the verdict, Land, and beat rows (one row never speaks for another).
func RenderStock(s ledger.Stock) h.H {
	parts := []h.H{
		h.Class("pk-card stock-row"),
		h.Data("state", "stock"),
		h.Span(h.Class("stock__count"), h.Text(strconv.Itoa(s.Count)+" confirmed")),
		// The reinvested share (dispatch-minted) is its own span beside the count, so
		// the Lead SEES compounding — a spend's catch distinct from a fresh mint — not
		// two equal bumps. At 0 it reads calm: a held fact, never a nudge.
		h.Span(h.Class("stock__reinvested"), h.Text("reinvested "+strconv.Itoa(s.Reinvested))),
	}
	reasons := make([]string, 0, len(s.ByReason))
	for r := range s.ByReason {
		reasons = append(reasons, r)
	}
	sort.Strings(reasons) // stable order — the row reads the same every render
	for _, r := range reasons {
		parts = append(parts, h.Span(h.Class("stock__reason"), h.Text(r+": "+strconv.Itoa(s.ByReason[r]))))
	}
	parts = append(parts,
		h.Span(h.Class("stock__self-flagged"), h.Text("self-flagged "+strconv.Itoa(s.SelfFlagged))),
		h.Span(h.Class("stock__would-ship"), h.Text("would-have-shipped "+strconv.Itoa(s.WouldHaveShipped))),
	)
	return h.Div(parts...)
}
