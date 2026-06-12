package app

import (
	"net/url"

	"github.com/go-via/via/h"
)

// cardReturnCrumb is the back-affordance a drill-in surface (/review, /settings)
// renders so the Lead is never stranded: an anchor BACK to the originating session
// card (/?key=<key>), reusing the breadcrumb crumb idiom. The key is URL-escaped so
// a query-metacharacter key (valid as a NATS token, e.g. "a&b") round-trips to the
// exact session rather than splitting the query (mirrors the board drill href).
func cardReturnCrumb(key string) h.H {
	return h.A(
		h.Href("/?key="+url.QueryEscape(key)),
		h.Class("board-nav__crumb drill-return"),
		h.Text("← back to card"),
	)
}

// reviewSessionCrumb is the per-order review's UP-link to the SESSION review (drop
// the wo scope), closing the per-order→session leg of the symmetric review nav so a
// funded order's test-debt isn't a dead end.
func reviewSessionCrumb(key string) h.H {
	return h.A(
		h.Href("/review?key="+url.QueryEscape(key)),
		h.Class("board-nav__crumb drill-return"),
		h.Text("↑ session review"),
	)
}

// navHeader is the shared, stateless nav bar prepended to every page (the fleet
// board and each session card), turning the disconnected URLs into a navigable
// app. It carries a "packets" home link and a breadcrumb: a "fleet" link back to
// /board, plus — on a session card — the RAW session key (honest, never a
// fabricated label, so the Lead always knows which session they are on). It is
// pure markup + href-based browser navigation: no JS, no client state, no menus
// (keyboard nav is a later slice).
func navHeader(key string) h.H {
	crumb := []h.H{
		h.Class("board-nav__breadcrumb"),
		h.A(h.Href("/board"), h.Class("board-nav__crumb"), h.Text("fleet")),
		h.Span(h.Class("board-nav__sep"), h.Text(" · ")),
		h.A(h.Href("/settings"), h.Class("board-nav__crumb"), h.Text("settings")),
	}
	if key != "" {
		crumb = append(crumb,
			h.Span(h.Class("board-nav__sep"), h.Text(" › ")),
			h.Span(h.Class("board-nav__key"), h.Text(key)),
		)
	}
	return h.Nav(
		h.Class("board-nav"),
		// A named navigation landmark, distinct from the main content region, so an
		// assistive-tech user can jump between chrome and content rather than tabbing
		// through everything.
		h.Attr("aria-label", "primary"),
		h.A(h.Href("/board"), h.Class("board-nav__home"), h.Text("packets")),
		h.Div(crumb...),
	)
}
