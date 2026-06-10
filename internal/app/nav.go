package app

import "github.com/go-via/via/h"

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
	}
	if key != "" {
		crumb = append(crumb,
			h.Span(h.Class("board-nav__sep"), h.Text(" › ")),
			h.Span(h.Class("board-nav__key"), h.Text(key)),
		)
	}
	return h.Nav(
		h.Class("board-nav"),
		h.A(h.Href("/board"), h.Class("board-nav__home"), h.Text("packets")),
		h.Div(crumb...),
	)
}
