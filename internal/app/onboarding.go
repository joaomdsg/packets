package app

import (
	"github.com/go-via/via/h"

	"github.com/joaomdsg/packets/internal/ledger"
)

// onboardingHint is the calm first-run affordance on a brand-new session card.
// Without it a fresh session renders nothing but bare zeros — 0 confirmed,
// balance 0, 0 dispatched — stranding the Lead at the entry to the core loop with
// no sense of WHAT to do or WHY nothing is moving. The hint names the real flow
// (the oracle mints a catch → balance → spend funds a work-order → a caught order
// reinvests), never a fabricated metric.
//
// It renders ONLY for a truly-fresh session, gated on stock.Count == 0. That single
// check is the COMPLETE emptiness test, not a shortcut: the stock count is
// monotonic (a confirmed catch is never un-minted), and it is the prerequisite for
// every other sign of activity — a spendable balance comes only from a minted catch,
// and a dispatched work-order comes only from spending that balance. So stock.Count
// == 0 holds exactly when the session has no catches, no balance, and no dispatches:
// the blank entry screen. Returns nil otherwise, so the caller omits it.
// hasAnchor reports whether the session runs the connect catch cycle on load (a
// base revision + anchored file). Only an anchored session honestly promises an
// automatic catch on load; a repo-only session runs no cycle, so it must not — that
// would tell the Lead to wait for a mint that never comes. The first step varies on
// it accordingly.
func onboardingHint(stock ledger.Stock, hasAnchor bool) h.H {
	if stock.Count != 0 {
		return nil
	}
	mintStep := "Funded work-orders run the catch cycle — when the oracle confirms a catch it mints to your balance."
	if hasAnchor {
		mintStep = "This card runs the catch cycle on load — when the oracle confirms a catch it mints to your balance."
	}
	return h.Section(
		h.Class("pk-card onboarding"),
		h.Data("state", "empty"),
		h.P(h.Class("onboarding__lead"), h.Text("No confirmed catches yet.")),
		h.P(h.Class("onboarding__step"), h.Text(mintStep)),
		h.P(h.Class("onboarding__step"), h.Text("Spend balance to fund a work-order; a funded order that catches reinvests, compounding the stock.")),
	)
}
