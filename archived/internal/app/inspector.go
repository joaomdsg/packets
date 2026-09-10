package app

import (
	"strconv"

	"github.com/go-via/via/h"

	"github.com/joaomdsg/packets/internal/packet"
	"github.com/joaomdsg/packets/internal/review"
)

// gauntletGateRow names one of the six gates alongside its Gate value, in
// G1..G6 order (the gauntlet's six gates) — the fixed row order renderInspectorTimeline
// renders the gauntlet in.
type gauntletGateRow struct {
	name string
	gate packet.Gate
}

// gauntletGateRows returns g's six gates as named rows, in G1..G6 order.
func gauntletGateRows(g packet.Gauntlet) []gauntletGateRow {
	return []gauntletGateRow{
		{"intent fidelity", g.IntentFidelity},
		{"handshake conformance", g.HandshakeConformance},
		{"handshake tightness", g.HandshakeTightness},
		{"build · vet · lint", g.BuildVetLint},
		{"test sensitivity", g.TestSensitivity},
		{"independent check", g.IndependentCheck},
	}
}

// shortRev truncates rev to the short-SHA convention (7 characters); a rev at
// or under that length renders as-is — never padded or fabricated.
func shortRev(rev string) string {
	if len(rev) > 7 {
		return rev[:7]
	}
	return rev
}

// renderInspectorTitlebar renders the Inspector's identity strip: the scope
// name (pkt#<id> or the raw session key — honest, never a display alias), the
// packet's own Name beside it when the scope folds to one (packetName ""
// omits it), a base→fix rev chip in short SHAs (omitted entirely when either
// rev is unknown — the Inspector never fabricates a revision), a neutral lane
// chip (lane is QoS, never a state color; LaneUnmeasured
// when the scope folds to no single packet or nothing has been measured yet,
// rendered honestly rather than omitted), and the repo addr (packet.ParseAddr
// — owner/name, or the honest local/<dir> fallback). navHeader already
// carries the brand mark + lockup, so this strip adds no second mark (the
// design system's Titlebar pattern).
func renderInspectorTitlebar(scope, baseRev, fixRev string, addr packet.Addr, packetName string, lane packet.Lane) h.H {
	parts := []h.H{
		h.Class("inspector__titlebar"),
		h.Span(h.Class("inspector__name"), h.Text(scope)),
	}
	if packetName != "" {
		parts = append(parts, h.Span(h.Class("inspector__packet-name"), h.Text(packetName)))
	}
	if baseRev != "" && fixRev != "" {
		parts = append(parts, h.Span(h.Class("inspector__rev"),
			h.Text(shortRev(baseRev)+"→"+shortRev(fixRev))))
	}
	parts = append(parts, h.Span(h.Class("inspector__lane"), h.Text("lane "+lane.String())))
	if addr.Name != "" {
		parts = append(parts, h.Span(h.Class("inspector__addr"), h.Text(addr.String())))
	}
	return h.Div(parts...)
}

// renderInspectorGrid assembles the 3-column Inspector body (252px|1fr|312px):
// the changed-files tree, the Monaco island + answer form, and the
// annotation rail — each hairline-bounded, mirroring the Console shell's
// region approach.
func renderInspectorGrid(left, main h.H, rail []h.H) h.H {
	return h.Div(
		h.Class("inspector"),
		h.Div(h.Class("inspector__tree"), left),
		h.Div(h.Class("inspector__main"), main),
		h.Aside(append([]h.H{h.Class("inspector__rail")}, rail...)...),
	)
}

// renderInspectorEmptyTree is the honest left-rail empty state for a review
// that isn't scoped to one packet: there is no single base→fix diff to
// scope a tree to, so the Inspector says so rather than showing an arbitrary
// tree — the open threads still list on the annotation rail.
func renderInspectorEmptyTree() h.H {
	return h.Div(h.Class("inspector__tree-empty"),
		h.Text("pick a packet to scope the file tree"))
}

// renderInspectorTimeline is the Inspector's full-width footer: the six
// gauntlet gates, one row each, in G1..G6 order. Every
// row renders even when its status is GateNotRun — a real, honest absence,
// never hidden (the replayable packet-life timeline itself is a later
// slice; this footer is the gauntlet record the design reserved it for).
// orderID>0 additionally renders the ConfirmIntentFidelity affordance
// inline on the IntentFidelity row while that gate is still NotRun (the G1
// human residual — a real action, never a computed gate); orderID<=0 (the
// session-scoped review, which has no single packet to gauntlet) omits it,
// since there is no packet to confirm against.
func renderInspectorTimeline(navKey string, orderID int, g packet.Gauntlet) h.H {
	var rows []h.H
	for _, row := range gauntletGateRows(g) {
		canConfirm := orderID > 0 && row.name == "intent fidelity" && row.gate.Status == packet.GateNotRun
		rows = append(rows, renderGauntletGateRow(orderID, row, canConfirm))
	}
	return h.Div(h.Class("inspector__timeline gauntlet"),
		h.Div(h.Class("inspector__timeline-kicker"), h.Text("gauntlet")),
		h.Div(append([]h.H{h.Class("gauntlet__list")}, rows...)...),
	)
}

// renderGauntletGateRow renders one gate as `<name> · <status> · <detail>`
// in mono voice: a name label, a neutral status pill carrying
// data-status=<lowercase status> (the CSS hook for the status→color idiom —
// verified/held/risk/text-faint, never invented colors), and the Detail note
// when non-empty. canConfirm adds the inline confirm affordance.
func renderGauntletGateRow(orderID int, row gauntletGateRow, canConfirm bool) h.H {
	parts := []h.H{
		h.Class("gauntlet-gate"),
		h.Span(h.Class("gauntlet-gate__name"), h.Text(row.name)),
		h.Span(h.Class("gauntlet-gate__pill"), h.Data("status", row.gate.Status.String()), h.Text(row.gate.Status.String())),
	}
	if row.gate.Detail != "" {
		parts = append(parts, h.Span(h.Class("gauntlet-gate__detail"), h.Text(row.gate.Detail)))
	}
	if canConfirm {
		expr := "$confirmwo=" + strconv.Itoa(orderID) + ";@post('/_action/ConfirmIntentFidelity')"
		parts = append(parts, h.Button(h.Type("button"), h.Class("pk-btn gauntlet-gate__confirm"),
			h.Data("on:click", expr), h.Text("confirm")))
	}
	return h.Div(parts...)
}

// annotationRailHeader renders the rail's kicker in the house voice
// ("label · N" counts, the vocabulary map).
func annotationRailHeader(n int) h.H {
	return h.Div(h.Class("inspector__rail-header"), h.Text("annotations · "+strconv.Itoa(n)))
}

// renderAnnotationCard renders one open thread as an Inspector annotation
// card (the design system's AnnotationCard spec): an "agent" author chip (every open
// thread here is an oracle finding, never user-authored), a severity chip
// carrying the thread's Conventional-Comment tag, the file:line anchor in
// mono, and the body as prose. It KEEPS the "review-thread" class and the
// data-file/data-line attributes the answer-form anchor flow and the editor
// island payload depend on — the annotation-card classes are additive, never
// a replacement.
func renderAnnotationCard(t review.Thread) h.H {
	return h.Div(
		h.Class("review-thread annotation-card"),
		h.Data("file", t.File),
		h.Data("line", strconv.Itoa(t.StartLine)),
		h.Div(h.Class("annotation-card__head"),
			h.Span(h.Class("annotation-card__chip annotation-card__chip--author"), h.Text("agent")),
			h.Span(h.Class("annotation-card__chip annotation-card__chip--sev"), h.Text(t.Tag)),
			h.Span(h.Class("review-thread__anchor annotation-card__where"),
				h.Text(t.File+":"+strconv.Itoa(t.StartLine))),
		),
		h.Span(h.Class("review-thread__body annotation-card__body"), h.Text(t.Render())),
	)
}

// renderAnnotationCards renders every thread as an annotation card, in order.
func renderAnnotationCards(threads []review.Thread) []h.H {
	out := make([]h.H, 0, len(threads))
	for _, t := range threads {
		out = append(out, renderAnnotationCard(t))
	}
	return out
}
