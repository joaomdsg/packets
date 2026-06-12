package app

import (
	"github.com/go-via/via/h"
	"github.com/go-via/via/on"

	"github.com/joaomdsg/packets/internal/ledger"
)

// renderFundWork co-locates the two ways the Lead FUNDS work (Flow B): Spend turns a
// confirmed catch (balance hue) into a funded work-order, and PlaceOrder authors a
// live order against earned attention bandwidth (accent hue). They read as unrelated
// today; here they sit under ONE "fund work" group with a dim two-currency explainer
// so the Lead sees both funding moves and the currency each draws. Each affordance is
// gated on its own currency (spend on balance, authoring/place on bandwidth) and the
// whole group is omitted when neither is available — never an empty heading, never a
// meter/gauge/bar (constraint 3): a labelled affordance pair only.
func renderFundWork(c *LiveCard, cfg LiveConfig, log *ledger.Log, balance, bandwidth int) h.H {
	var controls []h.H
	// The Spend action — turning a confirmed catch into a funded work-order — rendered
	// ONLY when there is balance to spend (offering it with nothing to spend is a
	// dishonest no-op click).
	if balance > 0 {
		controls = append(controls, h.Button(
			on.Click(c.Spend),
			h.Class("pk-btn spend-action"),
			h.Text(spendButtonLabel(cfg, log)),
		))
	}
	// AUTHOR + place a live order, funded by earned attention bandwidth (the
	// responsiveness the Lead earned funds the autonomous work they dispatch). Rendered
	// only with bandwidth to spend.
	if bandwidth > 0 {
		controls = append(controls, renderAuthoring(c))
	}
	if len(controls) == 0 {
		return nil
	}
	group := []h.H{
		h.Class("fund-work"),
		h.Span(h.Class("pk-section-label fund-work__label"), h.Text("fund work")),
		h.P(h.Class("fund-work__explainer"),
			h.Text("balance spends a catch; bandwidth places a live order.")),
	}
	return h.Div(append(group, controls...)...)
}
