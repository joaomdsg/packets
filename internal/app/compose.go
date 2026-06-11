package app

import (
	"github.com/go-via/via/h"
	"github.com/go-via/via/on"
)

// renderCompose builds the order-authoring control: a prompt textarea bound to the
// card's OrderPrompt signal and a place-order button. When no Anthropic key is
// configured a calm note links to the setup surface — the authored order would fail
// to run without one, so the Lead is pointed at the fix rather than left to place a
// dead order.
func renderCompose(c *LiveCard) h.H {
	parts := []h.H{
		h.Class("compose"),
		h.Attr("aria-label", "author a live order"),
		h.Textarea(c.OrderPrompt.Bind(), h.Class("compose__prompt"),
			h.Placeholder("Describe the task for a live order…")),
		h.Button(on.Click(c.PlaceOrder), h.Class("compose__place"), h.Text("Place order")),
	}
	if tokenStore == nil || !tokenStore.Configured() {
		parts = append(parts, h.Div(
			h.Class("compose__needs-key"),
			h.Text("No Anthropic API key configured — "),
			h.A(h.Href("/settings"), h.Class("compose__needs-key-link"), h.Text("set one in settings")),
			h.Text(" to run live orders."),
		))
	}
	return h.Div(parts...)
}
