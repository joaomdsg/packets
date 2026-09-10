package surface

import (
	"strings"

	"github.com/go-via/via/h"
)

// RenderBeats renders the cycle's streamed beats as their OWN row, distinct from
// the verdict and Land rows (one row never speaks for another). It takes the
// comma-joined Kind list the live card accumulates as beats arrive over SSE, and
// renders one marked span per beat so the human feels the loop's tempo (gate ran
// base → fix done → catch → forward) accruing live rather than a spinner. An
// empty list (no beat has streamed yet) renders an empty row — no tempo to show.
func RenderBeats(beats string) h.H {
	parts := []h.H{h.Class("pk-card beat-row"), h.Data("state", "beats")}
	if beats != "" {
		for _, kind := range strings.Split(beats, ",") {
			parts = append(parts, h.Span(h.Class("beat"), h.Data("beat", kind), h.Text(beatLabel(kind))))
		}
	}
	return h.Div(parts...)
}

// beatLabel maps a beat's raw Kind (the wire/attribute identifier, unchanged) to
// vocabulary-clean VISIBLE text — "oracle" and "land" are retired from every
// surface, but the identifier itself stays as-is since it is never rendered as
// prose (only as the data-beat attribute other code and tests key off).
func beatLabel(kind string) string {
	switch kind {
	case "settle-base":
		return "settled base"
	case "oracle-base":
		return "gate ran base"
	case "settle-fix":
		return "settled fix"
	case "oracle-fix":
		return "gate ran fix"
	case "land":
		return "forward"
	default:
		return kind
	}
}
