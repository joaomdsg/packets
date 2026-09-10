package app

import (
	"strconv"

	"github.com/go-via/via"
)

// ConfirmIntentFidelity is G1's real human action (one of the gauntlet's six
// gates — the human residual, an Inspector affordance, never a computed
// gate): it marks
// the packet named by the ConfirmWO signal as intent-fidelity-confirmed,
// naming the session key as the confirming identity (there is no other
// authenticated "you" yet — no fabricated identity). A blank, zero, or
// non-numeric ConfirmWO, or an unknown session, is a calm no-op, mirroring
// ResolveAdjustment/AddAdjustment's handling of blank input.
func (c *ReviewCard) ConfirmIntentFidelity(ctx *via.Ctx) {
	key := c.Key
	if key == "" {
		key = defaultSessionKey
	}
	e := lookupLiveEntry(key)
	if e == nil {
		return
	}
	orderID, err := strconv.Atoi(c.ConfirmWO.Read(ctx))
	if err != nil || orderID <= 0 {
		return
	}
	e.confirmIntentFidelity(orderID, key)
}
