package surface

import (
	"github.com/go-via/via/h"

	"github.com/joaomdsg/packets/internal/pipe"
)

// RenderLand renders the integration verdict (the result of rebasing the fix
// onto trunk tip and checking the integrated tree) as its OWN card row, separate
// from the oracle verdict — it never overloads RenderVerdict's row. LandClean is
// a calm, no-action row (the integration surface stays out of the way: there is
// nothing to act on); LandConflict and LandChecksRed are the only actionable
// rows, each naming what the reviewer must do. The catch verdict and this land
// verdict are orthogonal — one row never speaks for the other.
func RenderLand(land pipe.LandState) h.H {
	state, headline, detail := presentLand(land)
	return h.Div(
		h.Class("pk-card land-row"),
		h.Data("state", state),
		h.P(h.Class("land-row__headline"), h.Text(headline)),
		h.P(h.Class("land-row__detail"), h.Text(detail)),
	)
}

// presentLand maps a land verdict to its rendered state token, headline, and
// detail. Clean and pending carry no actionable copy on purpose — info is gated
// to the moment it is actionable, which is only conflict/checks-red.
func presentLand(land pipe.LandState) (state, headline, detail string) {
	switch land {
	case pipe.LandClean:
		return "land-clean", "", "" // integrates cleanly — nothing to act on
	case pipe.LandConflict:
		return "land-conflict", "Trunk moved — rebase needed",
			"Trunk advanced under this change; it must be rebased onto the new tip before it can merge."
	case pipe.LandChecksRed:
		return "land-checks-red", "Red on trunk tip",
			"Green before integration, but the checks fail once rebased onto the current trunk tip."
	default: // empty/unknown → the cycle has not produced an integration verdict yet
		return "land-pending", "", ""
	}
}
