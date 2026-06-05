package surface

import (
	"strings"

	"github.com/go-via/via/h"
)

// RenderBeats renders the cycle's streamed beats as their OWN row, distinct from
// the verdict and Land rows (one row never speaks for another). It takes the
// comma-joined Kind list the live card accumulates as beats arrive over SSE, and
// renders one marked span per beat so the human feels the loop's tempo (oracle
// base done → fix done → catch → land) accruing live rather than a spinner. An
// empty list (no beat has streamed yet) renders an empty row — no tempo to show.
func RenderBeats(beats string) h.H {
	parts := []h.H{h.Class("beat-row"), h.Data("state", "beats")}
	if beats != "" {
		for _, kind := range strings.Split(beats, ",") {
			parts = append(parts, h.Span(h.Class("beat"), h.Data("beat", kind), h.Text(kind)))
		}
	}
	return h.Div(parts...)
}
