package app

import (
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/go-via/via/h"
	"github.com/go-via/via/on"

	"github.com/joaomdsg/packets/internal/packet"
)

// consoleNeedsYouCap bounds the needs-you rail to a handful of full cards
// (the console's paged queue) — the rest collapse
// into a single "and N more" line so a drowning session stays scannable.
const consoleNeedsYouCap = 4

// consoleSettledCap bounds the settled rail to the "last ~5" the slice spec
// calls for.
const consoleSettledCap = 5

// consoleIntentWords bounds the in-flight strip's intent preview to a
// scannable clause rather than a full paragraph.
const consoleIntentWords = 8

// renderConsole assembles the "/" Console shell around the PRESERVED center
// content: a needs-you rail of this session's HELD packets, the center column
// (a hero stat + in-flight strip above the untouched act-now/state-history
// sections), and a settled+watches rail. Every region folds from the SAME
// packets slice (the packet aggregate) — one source of truth,
// never a fabricated placeholder.
func renderConsole(c *LiveCard, navKey string, packets []packet.Packet, addr packet.Addr, center h.H) h.H {
	verified := 0
	for _, p := range packets {
		if p.State == packet.Verified {
			verified++
		}
	}
	mainParts := []h.H{h.Class("console__main"), renderHeroStat(navKey, verified, addr)}
	if strip := renderInFlightStrip(packets); strip != nil {
		mainParts = append(mainParts, strip)
	}
	mainParts = append(mainParts, renderLaneHealthGrid(navKey, packets))
	mainParts = append(mainParts, center)

	return h.Div(
		h.Class("console"),
		renderNeedsYouRail(navKey, packets),
		h.Div(mainParts...),
		renderSettledRail(c, navKey, packets),
	)
}

// heldPackets filters to packets a human must look at (Hold != HoldNone),
// blocking packets first — the most attention-worthy lead the queue. A stable
// sort keeps each hold kind's own relative (send-recency) order intact.
func heldPackets(packets []packet.Packet) []packet.Packet {
	var held []packet.Packet
	for _, p := range packets {
		if p.Hold != packet.HoldNone {
			held = append(held, p)
		}
	}
	sort.SliceStable(held, func(i, j int) bool {
		return held[i].Hold == packet.HoldBlocking && held[j].Hold != packet.HoldBlocking
	})
	return held
}

// consoleDryAside is the house voice's one permitted editorial sentence per
// screen (design/guidelines/voice.md rule 10), verbatim from concepts.md's
// attention-economics framing — rendered lowercase, matching this repo's
// operational-copy casing, exactly ONCE, only in the truly-empty needs-you
// branch (never appended alongside real held cards).
const consoleDryAside = "✱ an empty queue is success, not idleness."

// renderNeedsYouRail is the left rail: one card per held packet (capped at
// consoleNeedsYouCap, "+N more" beyond that), each linking straight into the
// packet's own review — or the victory empty state (plus the dry aside) when
// nothing is held. A calibration draw sits below it: a
// real skim-worthy card when the auto-forwarded set has something to draw
// from, otherwise the honest dashed "no calibration draws yet" placeholder.
func renderNeedsYouRail(navKey string, packets []packet.Packet) h.H {
	held := heldPackets(packets)
	header := h.Div(h.Class("console__panel-header"), h.Text("needs you · "+strconv.Itoa(len(held))))

	body := []h.H{h.Class("console__rail-body")}
	if len(held) == 0 {
		body = append(body,
			h.Div(h.Class("console__card", "console__card--dashed"), h.Text("nothing needs you")),
			h.Div(h.Class("console__dry-aside"), h.Text(consoleDryAside)),
		)
	} else {
		shown := held
		more := 0
		if len(shown) > consoleNeedsYouCap {
			more = len(shown) - consoleNeedsYouCap
			shown = shown[:consoleNeedsYouCap]
		}
		for _, p := range shown {
			body = append(body, renderNeedsYouCard(navKey, p))
		}
		if more > 0 {
			body = append(body, h.Div(h.Class("console__more"), h.Text("and "+strconv.Itoa(more)+" more")))
		}
	}
	body = append(body, renderCalibrationCard(navKey, packets))

	return h.Aside(h.Class("console__rail", "console__rail--needs-you"),
		header,
		h.Div(body...),
	)
}

// renderCalibrationCard draws the Console's calibration sample
// (the calibration draw): a real skim link into an auto-forwarded
// (Verified) packet when one exists — cached per-session (liveEntry.calibMu)
// so the draw stays STABLE across the 100ms poll's re-renders rather than
// re-rolling on every tick — or the honest dashed empty state when nothing
// has forwarded on its own yet.
func renderCalibrationCard(navKey string, packets []packet.Packet) h.H {
	e := lookupLiveEntry(navKey)
	previous := 0
	if e != nil {
		previous = e.cachedCalibDraw()
	}
	id, ok := drawCalibration(packets, previous)
	if !ok {
		return h.Div(
			h.Class("console__card", "console__card--dashed"),
			h.Div(h.Class("console__empty-kicker"), h.Text("calibration")),
			h.Div(h.Text("no calibration draws yet")),
		)
	}
	if e != nil {
		e.setCalibDraw(id)
	}
	name := ""
	for _, p := range packets {
		if p.ID == id {
			name = p.Name
			break
		}
	}
	href := "/review?key=" + url.QueryEscape(navKey) + "&wo=" + strconv.Itoa(id)
	return h.A(
		h.Href(href),
		h.Class("console__card"),
		h.Div(h.Class("console__empty-kicker"), h.Text("calibration")),
		h.Div(h.Class("console__thread-title"), h.Text(name)),
		h.Div(h.Class("console__thread-arrow"), h.Text("skim →")),
	)
}

// renderNeedsYouCard renders one held packet as a card: an 8px state cell
// (blocking/advisory hued), the packet's Name, its one-clause HoldReason in
// mono microtype, and a trailing "inspect →" naming the destination. The
// whole card is the link into the packet's own review.
func renderNeedsYouCard(navKey string, p packet.Packet) h.H {
	href := "/review?key=" + url.QueryEscape(navKey) + "&wo=" + strconv.Itoa(p.ID)
	state := "held"
	if p.Hold == packet.HoldBlocking {
		state = "held-blocking"
	}
	return h.A(
		h.Href(href),
		h.Class("console__card"),
		h.Span(h.Class("console__cell"), h.Data("state", state)),
		h.Div(h.Class("console__thread-title"), h.Text(p.Name)),
		h.Div(h.Class("console__thread-loc"), h.Text(p.HoldReason)),
		h.Div(h.Class("console__thread-arrow"), h.Text("inspect →")),
	)
}

// renderHeroStat is the center column's header strip: the count of packets
// whose State is Verified (done, the packet's own catch minted, no open
// questions) — distinct from Delivered, which the settled rail names
// separately once a real ACK exists. Beside it, the interrupt KPI names the
// session's REAL weekly interrupt count against the locked cap
// (the console's "N/10 interrupts" KPI), never a
// fabricated number. The addr line is the honest repo identity
// (packet.ParseAddr), never a fabricated owner.
func renderHeroStat(navKey string, verifiedCount int, addr packet.Addr) h.H {
	used, cap := weeklyInterrupts(navKey)
	return h.Div(h.Class("console__hero"),
		h.Span(h.Class("console__hero-stat"), h.Text(strconv.Itoa(verifiedCount))),
		h.Span(h.Class("console__hero-label"), h.Text("packets verified")),
		h.Span(h.Class("console__interrupt-kpi"), h.Text(strconv.Itoa(used)+"/"+strconv.Itoa(cap)+" interrupts")),
		h.Span(h.Class("console__hero-addr"), h.Text("addr "+addr.String())),
	)
}

// renderInFlightStrip is the center column's "in flight · N" strip: one
// pulsing --signal cell per InFlight packet (with a first-words preview of its
// intent) and one ghost-outline cell per Composing packet — the mark's
// composing idiom reused for a queued packet. Returns nil (omitted entirely)
// when nothing is in flight or composing, rather than an empty header.
func renderInFlightStrip(packets []packet.Packet) h.H {
	var inFlight, composing []packet.Packet
	for _, p := range packets {
		switch p.State {
		case packet.InFlight:
			inFlight = append(inFlight, p)
		case packet.Composing:
			composing = append(composing, p)
		}
	}
	n := len(inFlight) + len(composing)
	if n == 0 {
		return nil
	}

	rows := []h.H{h.Div(h.Class("console__panel-header"), h.Text("in flight · "+strconv.Itoa(n)))}
	for _, p := range inFlight {
		rows = append(rows, h.Div(h.Class("console__inflight-row"),
			h.Span(h.Class("console__cell"), h.Data("state", "in-flight")),
			h.Span(h.Class("console__inflight-name"), h.Text(p.Name)),
			h.Span(h.Class("console__inflight-intent"), h.Text(firstWords(p.Intent, consoleIntentWords))),
		))
	}
	for _, p := range composing {
		rows = append(rows, h.Div(h.Class("console__inflight-row"),
			h.Span(h.Class("console__cell"), h.Data("state", "composing")),
			h.Span(h.Class("console__inflight-name"), h.Text(p.Name)),
		))
	}
	return h.Div(append([]h.H{h.Class("console__inflight")}, rows...)...)
}

// laneHealthBuckets is the fixed, honest display order of the lane-health
// grid's cards — every bucket always renders, even at zero, so an empty
// bucket is never mistaken for a missing mechanic (the honest-empty-state
// invariant).
var laneHealthBuckets = []packet.Lane{
	packet.LaneBestEffort, packet.LaneStandard, packet.LaneStrict, packet.LaneUnmeasured,
}

// renderLaneHealthGrid is the center column's "lane health" section:
// one kicker+count card per lane bucket, tallied ONLY from lanes
// ALREADY in the session's lane cache (liveEntry.cachedLane — a pure map
// read). A packet never opened in the Inspector has no cached lane yet and
// counts as unmeasured; this NEVER computes a lane itself, since it renders
// on every "/" poll tick and the exec-based Measure must stay off that path.
func renderLaneHealthGrid(navKey string, packets []packet.Packet) h.H {
	e := lookupLiveEntry(navKey)
	counts := make(map[packet.Lane]int, len(laneHealthBuckets))
	for _, p := range packets {
		lane := packet.LaneUnmeasured
		if e != nil {
			lane = e.cachedLane(p.ID)
		}
		counts[lane]++
	}

	cards := make([]h.H, 0, len(laneHealthBuckets))
	for _, lane := range laneHealthBuckets {
		cards = append(cards, renderLaneHealthCard(lane, counts[lane]))
	}

	return h.Div(h.Class("console__lane-health"),
		h.Div(h.Class("console__panel-header"), h.Text("lane health")),
		h.Div(append([]h.H{h.Class("console__lane-grid")}, cards...)...),
	)
}

// renderLaneHealthCard renders one bucket: a lowercase mono kicker naming the
// lane and a tabular-numeral count. data-lane carries the lane name so a
// specific bucket's count is addressable without relying on card order.
func renderLaneHealthCard(lane packet.Lane, n int) h.H {
	return h.Div(h.Class("console__lane-card"),
		h.Span(h.Class("console__lane-kicker"), h.Text(lane.String())),
		h.Span(h.Class("console__lane-count"), h.Data("lane", lane.String()), h.Text(strconv.Itoa(n))),
	)
}

// firstWords returns the first n whitespace-separated words of s — a
// scannable clause rather than a full paragraph. No ellipsis marker; a
// truncated clause reads fine without one in this dense mono row.
func firstWords(s string, n int) string {
	words := strings.Fields(s)
	if len(words) > n {
		words = words[:n]
	}
	return strings.Join(words, " ")
}

// renderSettledRail is the right rail: the honest replacement for a fake
// "recently delivered" (delivered isn't real until ACK). It lists
// verified, held, and delivered packets — a run failure IS settled-red, only
// composing/in-flight packets are excluded — each a lifecycle-colored state
// square + Name + one-word state, or the dashed empty state when nothing has
// settled. Below it, "your watches" (standing inspection): the
// three canonical standing triggers, each with a real precision score folded
// from human fired-vs-useful marks — "no history yet" until one exists,
// never a fabricated score.
func renderSettledRail(c *LiveCard, navKey string, packets []packet.Packet) h.H {
	var settled []packet.Packet
	for _, p := range packets {
		if p.State == packet.Verified || p.State == packet.Held || p.State == packet.Delivered {
			settled = append(settled, p)
		}
	}
	// The header counts every settled packet — captured BEFORE the display cap
	// below, so a caller that ever folds more than consoleSettledCap packets
	// still gets an honest count, not a silently truncated one.
	count := len(settled)
	if len(settled) > consoleSettledCap {
		settled = settled[:consoleSettledCap]
	}

	header := h.Div(h.Class("console__panel-header"), h.Text("settled · "+strconv.Itoa(count)))
	body := []h.H{h.Class("console__rail-body")}
	if len(settled) == 0 {
		body = append(body, h.Div(h.Class("console__card", "console__card--dashed"), h.Text("nothing settled yet")))
	} else {
		for _, p := range settled {
			body = append(body, renderSettledRow(p))
		}
	}

	return h.Aside(h.Class("console__rail", "console__rail--settled"),
		header, h.Div(body...),
		renderLearningCard(packets),
		renderWatchesRail(c, navKey, packets),
	)
}

// renderLearningCard shows a repo's real learning progress: the
// honest running count of settled packets against
// packet.LearningThreshold until it's reached, then "converged" — never a
// fabricated verdict, and never a stale progress fraction once real history
// clears the bar.
func renderLearningCard(packets []packet.Packet) h.H {
	settled := packet.SettledCount(packets)
	state := "learning"
	line := strconv.Itoa(settled) + "/" + strconv.Itoa(packet.LearningThreshold) + " settled"
	if packet.Converged(packets) {
		state = "converged"
		line = "converged"
	}
	header := h.Div(h.Class("console__panel-header"), h.Text("learning"))
	card := h.Div(h.Class("console__card"), h.Data("state", state), h.Text(line))
	return h.Div(header, card)
}

// renderWatchesRail lists the three canonical standing watches:
// each card names the watch, its real precision (or the honest
// "no history yet" before any fire has been marked), and — for a kind's
// unmarked fire, only while that kind hasn't lost the right to
// interrupt — a mark prompt naming the packet that tripped it.
func renderWatchesRail(c *LiveCard, navKey string, packets []packet.Packet) h.H {
	var fires []packet.WatchFire
	if e := lookupLiveEntry(navKey); e != nil {
		fires = e.watchFireSnapshot()
	}
	header := h.Div(h.Class("console__panel-header"), h.Text("your watches"))
	body := []h.H{h.Class("console__rail-body")}
	for _, kind := range standingWatchKinds {
		body = append(body, renderWatchCard(c, packets, kind, fires))
	}
	return h.Div(header, h.Div(body...))
}

// renderWatchCard renders one standing watch's status card. useful/sampled
// are counted directly from fires (never re-derived from Precision's
// fraction, which would risk a lossy float round-trip back to exact
// integers) while the NOISY judgment itself calls packet.IsNoisy — the one
// piece of real domain logic here, never duplicated.
func renderWatchCard(c *LiveCard, packets []packet.Packet, kind packet.WatchKind, fires []packet.WatchFire) h.H {
	var sampled, useful int
	var unmarked *packet.WatchFire
	for i := range fires {
		if fires[i].Kind != kind {
			continue
		}
		if fires[i].Useful == nil {
			f := fires[i]
			unmarked = &f // the LAST match wins — fires are appended chronologically
			continue
		}
		sampled++
		if *fires[i].Useful {
			useful++
		}
	}
	score := 0.0
	if sampled > 0 {
		score = float64(useful) / float64(sampled)
	}
	noisy := packet.IsNoisy(score, sampled)

	precisionLine := "no history yet"
	if sampled > 0 {
		precisionLine = strconv.Itoa(useful) + "/" + strconv.Itoa(sampled) + " useful"
		if noisy {
			precisionLine += " · noisy — lost interrupt rights"
		}
	}

	card := []h.H{
		h.Class("console__card"),
		h.Div(h.Class("console__watch-name"), h.Text(kind.String())),
		h.Div(h.Class("console__watch-precision"), h.Text(precisionLine)),
	}
	if unmarked != nil && !noisy {
		name := ""
		for _, p := range packets {
			if p.ID == unmarked.PacketID {
				name = p.Name
				break
			}
		}
		kindStr := strconv.Itoa(int(kind))
		woStr := strconv.Itoa(unmarked.PacketID)
		card = append(card, h.Div(h.Class("console__watch-prompt"),
			h.Span(h.Class("console__watch-prompt-name"), h.Text(name)),
			h.Button(
				on.Click(c.MarkWatchFire,
					on.SetSignal(&c.MarkWatchKind.Signal, kindStr),
					on.SetSignal(&c.MarkWatchWO.Signal, woStr),
					on.SetSignal(&c.MarkUseful.Signal, "true"),
				),
				h.Class("pk-btn", "console__watch-mark"),
				h.Text("useful"),
			),
			h.Button(
				on.Click(c.MarkWatchFire,
					on.SetSignal(&c.MarkWatchKind.Signal, kindStr),
					on.SetSignal(&c.MarkWatchWO.Signal, woStr),
					on.SetSignal(&c.MarkUseful.Signal, "false"),
				),
				h.Class("pk-btn--quiet", "console__watch-mark"),
				h.Text("noise"),
			),
		))
	}
	return h.Div(card...)
}

// renderSettledRow renders one settled packet: a state square colored by
// State/Hold (verified solid, held advisory amber, held blocking red,
// delivered dark-cyan — fills ONLY on a real ACK), its Name, and its
// lifecycle State.String() ("verified"/"held"/"delivered") — the same
// one-word vocabulary the needs-you rail and the packet aggregate already
// use, never a second name for the same fact.
func renderSettledRow(p packet.Packet) h.H {
	state := "verified"
	switch {
	case p.State == packet.Delivered:
		state = "delivered"
	case p.State == packet.Held:
		state = "held"
		if p.Hold == packet.HoldBlocking {
			state = "held-blocking"
		}
	}
	return h.Div(h.Class("console__card", "console__settled-row"),
		h.Span(h.Class("console__cell"), h.Data("state", state)),
		h.Span(h.Class("console__settled-id"), h.Text(p.Name)),
		h.Span(h.Class("console__settled-outcome"), h.Text(p.State.String())),
	)
}
