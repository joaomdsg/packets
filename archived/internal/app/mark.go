package app

import (
	"strconv"

	"github.com/go-via/via/h"
)

// smallMarkThreshold is the LOCKED small-size rule (the brand pack): below
// this cell size the ghost-outline TR cell's 18%-of-cell stroke goes
// sub-pixel and reads as noise rather than a composing state, so it falls
// back to a solid --delivered-mid fill instead.
const smallMarkThreshold = 14

// packetMark renders the locked 2x2 packets brand mark at the given cell size
// in px — TL/BL fill --signal, TR is ghost-composing (or, below
// smallMarkThreshold, solid --delivered-mid), BR fills --delivered. It is
// built from h.* markup + CSS classes in style.go, never a raster asset:
// packetsStyle derives every dimension (gap, radius, stroke) from the one
// --mark-cell custom property this sets inline.
func packetMark(cell int) h.H {
	return packetMarkCells(cell, false)
}

// packetMarkHeld is the mark's live "something is held" variant: BL burns
// --risk and pulses instead of the calm --signal fill. The other three cells
// and the small-size rule are unaffected.
func packetMarkHeld(cell int) h.H {
	return packetMarkCells(cell, true)
}

func packetMarkCells(cell int, held bool) h.H {
	tr := h.Span(h.Class("pk-mark__cell", "pk-mark__cell--delivered-mid"))
	if cell >= smallMarkThreshold {
		tr = h.Span(h.Class("pk-mark__cell", "pk-mark__cell--ghost"))
	}
	blClass := "pk-mark__cell--signal"
	if held {
		blClass = "pk-mark__cell--held"
	}
	return h.Span(
		h.Class("pk-mark"),
		h.Style("--mark-cell: "+strconv.Itoa(cell)+"px"),
		h.Span(h.Class("pk-mark__cell", "pk-mark__cell--signal")),
		tr,
		h.Span(h.Class("pk-mark__cell", blClass)),
		h.Span(h.Class("pk-mark__cell", "pk-mark__cell--delivered")),
	)
}

// packetLockup is the compact in-chrome brand lockup: the mark plus a
// stacked "packets" wordmark over a per-surface sub label (e.g. "console").
// This is the ONLY wordmark form allowed beside a breadcrumb — the locked
// full inline wordmark form must never share a row with one (the brand
// pack). sub is passed through as lowercase content; packetsStyle uppercases
// it visually via text-transform, so the source copy stays lowercase per the
// house voice rule.
func packetLockup(cell int, sub string) h.H {
	return h.Span(
		h.Class("pk-lockup"),
		// Re-set on the outer wrapper (packetMark already sets it on the grid
		// itself) so the lockup's own CSS — the mark↔label gap and word size —
		// can calc() off the same one parameterized cell size.
		h.Style("--mark-cell: "+strconv.Itoa(cell)+"px"),
		packetMark(cell),
		h.Span(
			h.Class("pk-lockup__labels"),
			h.Span(h.Class("pk-lockup__word"), h.Text("packets")),
			h.Span(h.Class("pk-lockup__sub"), h.Text(sub)),
		),
	)
}
