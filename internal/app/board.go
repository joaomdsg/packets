package app

import (
	"sort"
	"strconv"

	"github.com/go-via/via"
	"github.com/go-via/via/h"

	"github.com/joaomdsg/packets/internal/ledger"
)

// CardRow is one session's line on the fleet board — a calm cross-card tally
// projected purely from that session's own log. It is ACTIVITY, never priority:
// the board orders rows by Queued (work awaiting drain) so the Lead sees where
// motion is, NOT a leverage rank (blocked-downstream is uncomputable today).
type CardRow struct {
	Key              string
	Confirmed        int
	Reinvested       int
	Balance          int
	Queued           int
	Running          int
	Done             int
	Misses           int // done orders that minted NOTHING (Done − Reinvested) — honest losses made visible, not silently discarded
	BacklogRemaining int
	seq              int // registration ordinal — the deterministic tie-break, not rendered
}

// BoardRows projects one row per registered session by ranging liveReg, reading
// ONLY each session's own log projections (a ledger read failure degrades that
// field to zero, never breaks the board). Rows are ordered by Queued descending
// — the queued-awaiting-drain ACTIVITY signal — tie-broken by registration order
// (seq), so the order is deterministic across renders despite sync.Map's
// nondeterministic Range and the absence of any timestamp to sort by.
func BoardRows() []CardRow {
	var rows []CardRow
	liveReg.Range(func(k, v any) bool {
		e := v.(*liveEntry)
		row := CardRow{Key: k.(string), seq: e.seq}
		if e.log != nil {
			if recs, err := e.log.Records(); err == nil {
				st := ledger.ConfirmedCatches(recs)
				row.Confirmed, row.Reinvested = st.Count, st.Reinvested
			}
			if b, err := e.log.Balance(); err == nil {
				row.Balance = b
			}
			if c, err := e.log.DispatchStatusCounts(); err == nil {
				row.Queued, row.Running, row.Done = c.Queued, c.Running, c.Done
			}
			// Misses = done orders that minted no catch (Done minus the reinvested
			// catches, which each came from a done order). Clamp at 0 against the brief
			// window where a "wo:" catch is appended just before its done-status line.
			if m := row.Done - row.Reinvested; m > 0 {
				row.Misses = m
			}
			row.BacklogRemaining = len(fundableBacklog(e.cfg, e.log))
		}
		rows = append(rows, row)
		return true
	})
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Queued != rows[j].Queued {
			return rows[i].Queued > rows[j].Queued // most work awaiting drain first
		}
		return rows[i].seq < rows[j].seq // deterministic tie-break: earlier-registered first
	})
	return rows
}

// BoardCard is the cross-card FLEET surface: a calm row-per-session tally of the
// whole registry, ordered by queued ACTIVITY (see BoardRows). It is read-only —
// it holds no per-tab state, it re-projects liveReg on render — and it never
// labels a card by priority or leverage (the Lead sees where work is MOVING, not
// a fabricated importance rank).
type BoardCard struct{}

// hitRateLabel is the card's standing — the ONE honest progression number: Hits
// (catches a bet minted, = Reinvested) over Bets (resolved dispatched orders,
// = Done). A pure COUNT ratio of logged events, never an inferred probability or
// forecast, so it redeems against the mint/miss the Lead actually earned. Done==0
// reads a calm "hit-rate 0/0" — a string ratio, never a divide-by-zero.
//
// The numerator is clamped to Done: a "wo:" catch is Appended just before its
// order's done-status line (runOneOrder), so a board read can briefly observe
// Reinvested > Done. Hits can never exceed Bets, so the display clamps rather than
// leak a nonsense "hit-rate 1/0" — mirroring the Misses = max(0, Done−Reinvested)
// guard in BoardRows against the same transient window.
func hitRateLabel(r CardRow) string {
	hits := r.Reinvested
	if hits > r.Done {
		hits = r.Done
	}
	return "hit-rate " + strconv.Itoa(hits) + "/" + strconv.Itoa(r.Done)
}

// View renders one row per registered session: its confirmed/reinvested stock,
// spendable balance, queued/running/done activity, the distinct work still
// awaiting a spend, and the hit-rate standing. Calm spans in the stock idiom —
// no gauges, no priority, no forecast.
func (c *BoardCard) View(_ *via.CtxR) h.H {
	parts := []h.H{h.Class("board"), h.Data("state", "board")}
	for _, r := range BoardRows() {
		parts = append(parts, h.Div(
			h.Class("board-row"),
			h.Data("key", r.Key),
			h.Span(h.Class("board-row__key"), h.Text(r.Key)),
			h.Span(h.Class("board-row__stock"), h.Text(strconv.Itoa(r.Confirmed)+" confirmed, "+strconv.Itoa(r.Reinvested)+" reinvested")),
			h.Span(h.Class("board-row__balance"), h.Text("balance "+strconv.Itoa(r.Balance))),
			h.Span(h.Class("board-row__activity"), h.Text("queued "+strconv.Itoa(r.Queued)+", running "+strconv.Itoa(r.Running)+", done "+strconv.Itoa(r.Done))),
			h.Span(h.Class("board-row__misses"), h.Text(strconv.Itoa(r.Misses)+" misses")),
			h.Span(h.Class("board-row__hitrate"), h.Text(hitRateLabel(r))),
			h.Span(h.Class("board-row__backlog"), h.Text(strconv.Itoa(r.BacklogRemaining)+" awaiting")),
		))
	}
	return h.Div(parts...)
}
