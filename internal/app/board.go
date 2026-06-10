package app

import (
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/go-via/via"
	"github.com/go-via/via/h"
	"github.com/go-via/via/on"

	"github.com/joaomdsg/packets/internal/fabric"
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
	InFlight         int // claims submitted but not yet minted — producers' pending BETS, never confirmed catches (two-scores)
	Rejected         int // verified-lost: bets the host verified and found no catch — a RESOLVED loss, distinct from a pending in-flight bet and from a confirmed catch (two-scores)
	Dispatches       []ledger.DispatchView // this session's recent funded work-orders + their caught/missed outcome — honest per-order round-trip legibility, never a fabricated rank
	Balance          int
	Queued           int
	Running          int
	Done             int
	Misses           int // done orders that minted NOTHING (Done − Reinvested) — honest losses made visible, not silently discarded
	BacklogRemaining int
	OpenQuestions    int // the session's latest-cycle open review questions (surviving mutants) — test debt the green verdict hides, made visible across the fleet; a diagnostic, never scored (off the economy)
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
			// Claims in flight are producers' pending bets, projected from the claim
			// subtree alone — kept off Confirmed/Balance (two-scores). Degrade to 0 on
			// a read error, like every other field.
			if n, err := e.log.ClaimsInFlight(); err == nil {
				row.InFlight = n
			}
			// Verified-losses (bets the host rejected) are the resolved counterpart
			// to in-flight bets, kept off Confirmed/Balance (two-scores). Degrade to
			// 0 on a read error, like every other field.
			if n, err := e.log.ClaimsRejected(); err == nil {
				row.Rejected = n
			}
			// Recent funded work-orders + their caught/missed outcome — the
			// round-trip made legible. Degrade to nil on a read error like the rest.
			if ds, err := e.log.RecentDispatches(5); err == nil {
				row.Dispatches = ds
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
		// Open review questions are the session's latest-cycle surviving mutants, read
		// from the in-memory findings cache (not the log) — test debt the green verdict
		// hides, surfaced across the fleet. A diagnostic count, never scored.
		row.OpenQuestions = len(e.openFindings())
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
// whole registry, ordered by queued ACTIVITY (see BoardRows). It re-projects
// liveReg on render and never labels a card by priority or leverage (the Lead sees
// where work is MOVING, not a fabricated importance rank). It also carries the one
// command the fleet view owns: creating a new session.
type BoardCard struct {
	// NewKey holds the key typed into the create-session input (two-way bound), read
	// by CreateSession on submit. Per-tab signal, not authoritative session state.
	NewKey via.SignalStr `via:"newkey"`
	// RetireKey carries the key of the row whose retire button was clicked — set by
	// that button (on.SetSignal) just before the post, then read by RetireSession.
	RetireKey via.SignalStr `via:"retirekey"`
}

// RetireSession removes a session from the fleet view — the honest completion of
// CreateSession, so experiment sessions don't accumulate on the board. It unmounts
// the key from the registry (the ledger's events persist on the fabric; this only
// drops the live entry). The seeded default is NEVER retired — it is the "/" route's
// single-card fallback — and an empty/unknown key is a no-op.
//
// A retired key's durable claim consumer goroutine (per-session, spawned by the
// consumerSpawner and NOT tied to liveReg membership) keeps running until process
// shutdown — Delete drops only the registry entry, not the goroutine. This leak is
// BENIGN and intentionally NOT torn down here: POST /claim gates on liveReg.Load, so
// a retired key 404s and receives no new claims, leaving the consumer parked on an
// empty fetch. Adding teardown machinery would be out of scope for a fleet-view
// retire and would risk racing an in-flight verify; the goroutine costs nothing idle.
func (c *BoardCard) RetireSession(ctx *via.Ctx) {
	key := strings.TrimSpace(c.RetireKey.Read(ctx))
	if key == "" || key == defaultSessionKey {
		return // never strand the default fallback
	}
	liveReg.Delete(key)
}

// CreateSession starts a new session economy from the fleet view: it registers the
// typed key (inheriting the default session's config) so the Lead can work it
// immediately via the in-process card flow — no boot edit, no claim consumer needed
// (consumers serve only the untrusted-producer POST /claim path). An invalid
// subject token or a key that already exists is an honest no-op: a create never
// forges a bad token nor clobbers a live economy's log. (Producer claims for a
// runtime-created session are unsupported in V1 — the card flow works fully.)
func (c *BoardCard) CreateSession(ctx *via.Ctx) {
	key := strings.TrimSpace(c.NewKey.Read(ctx))
	if key == "" || !fabric.ValidToken(key) {
		return // never forge an invalid subject token
	}
	if _, exists := liveReg.Load(key); exists {
		return // never clobber a live economy
	}
	cfg, _ := readLiveState(defaultSessionKey) // inherit the default config (same repo/revs)
	_, _ = AddSession(key, cfg)                // validated above; a bind error leaves the registry unchanged
}

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
// the producers' bet lifecycle (in-flight bets and verified-losses, each its own
// span, never folded into the confirmed stock — two-scores), spendable balance,
// queued/running/done activity, the distinct work still awaiting a spend, and the
// hit-rate standing. Calm spans in the stock idiom — no gauges, no priority, no
// forecast.
func (c *BoardCard) View(_ *via.CtxR) h.H {
	parts := []h.H{h.Class("board"), h.Data("state", "board"), navHeader(""),
		// The fleet view's one command: start a new session economy. A calm input +
		// button, in the surface idiom — no modal, no menu.
		h.Div(h.Class("board-create"),
			h.Input(h.Type("text"), c.NewKey.Bind(), h.Class("board-create__key"), h.Placeholder("new session key")),
			h.Button(on.Click(c.CreateSession), h.Class("board-create__btn"), h.Text("Create session")),
		),
	}
	for _, r := range BoardRows() {
		row := []h.H{
			h.Class("board-row"),
			h.Data("key", r.Key),
			// The row key DRILLS into that session's card — the fleet board is not a
			// dead end. The default row links to /?key=default (explicit + honest). The
			// key is URL-escaped: fabric.ValidToken admits query metacharacters ('&',
			// '=', '#', '+'), which interpolated raw would split or truncate the query
			// and target the WRONG session — QueryEscape makes the link round-trip.
			h.A(h.Href("/?key="+url.QueryEscape(r.Key)), h.Class("board-row__key"), h.Text(r.Key)),
			h.Span(h.Class("board-row__stock"), h.Text(strconv.Itoa(r.Confirmed)+" confirmed, "+strconv.Itoa(r.Reinvested)+" reinvested")),
			// The producers' BET lifecycle, sealed into one explicitly-labelled
			// cluster so a pending/lost bet can't blend into the confirmed stock at a
			// glance — the two-scores separation carried by STRUCTURE, not by hoping a
			// reader parses each label. The inner spans keep their class hooks so a
			// future stylesheet can color bets muted-vs-solid with no server change.
			h.Div(h.Class("board-row__bets"),
				h.Span(h.Class("board-row__bets-label"), h.Text("bets:")),
				h.Span(h.Class("board-row__inflight"), h.Text(strconv.Itoa(r.InFlight)+" in flight")),
				h.Span(h.Class("board-row__rejected"), h.Text(strconv.Itoa(r.Rejected)+" verified-lost")),
			),
			h.Span(h.Class("board-row__balance"), h.Text("balance "+strconv.Itoa(r.Balance))),
			h.Span(h.Class("board-row__activity"), h.Text("queued "+strconv.Itoa(r.Queued)+", running "+strconv.Itoa(r.Running)+", done "+strconv.Itoa(r.Done))),
			h.Span(h.Class("board-row__misses"), h.Text(strconv.Itoa(r.Misses)+" misses")),
			h.Span(h.Class("board-row__hitrate"), h.Text(hitRateLabel(r))),
			h.Span(h.Class("board-row__backlog"), h.Text(strconv.Itoa(r.BacklogRemaining)+" awaiting")),
		}
		// Open review questions (surviving mutants) — surfaced only when there ARE any,
		// so a session carrying test debt the green verdict hides stands out at a glance
		// without nagging the clean ones. Links into that session's /review surface.
		if r.OpenQuestions > 0 {
			row = append(row, h.A(
				h.Href("/review?key="+url.QueryEscape(r.Key)),
				h.Class("board-row__questions"),
				h.Text(strconv.Itoa(r.OpenQuestions)+" open questions"),
			))
		}
		// The funded work-order round-trip made legible: recent dispatches with their
		// caught/missed outcome, in their own cluster (omitted when there are none).
		// Honest per-order outcomes, never a fabricated rank.
		if d := renderDispatches(r.Dispatches); d != nil {
			row = append(row, d)
		}
		// A retire control on every NON-default row — the default is the "/" route's
		// fallback and is never retirable. The button sets retirekey to THIS row's key
		// (on.SetSignal) just before the post, so RetireSession removes the right one.
		if r.Key != defaultSessionKey {
			row = append(row, h.Button(
				on.Click(c.RetireSession, on.SetSignal(&c.RetireKey.Signal, r.Key)),
				h.Class("board-row__retire"), h.Text("retire"),
			))
		}
		parts = append(parts, h.Div(row...))
	}
	return h.Div(parts...)
}

// renderDispatches renders a session's recent work-orders as a calm cluster —
// one span per order: "WO#<id> <path>:<line> <status>[ caught|missed]". The
// caught/missed outcome is shown only for a done order (a queued/running order
// has no outcome yet). Returns nil when there are none, so the cluster is omitted.
func renderDispatches(views []ledger.DispatchView) h.H {
	if len(views) == 0 {
		return nil
	}
	spans := []h.H{h.Class("board-row__dispatches"), h.Span(h.Class("board-row__dispatches-label"), h.Text("dispatches:"))}
	for _, v := range views {
		text := "WO#" + strconv.Itoa(v.ID) + " " + v.Target.Path + ":" + strconv.Itoa(v.Target.Line) + " " + v.Status
		span := []h.H{h.Class("board-row__dispatch")}
		// A resolved order carries its outcome as a hook so the calm palette can
		// color caught vs missed at a glance (a queued/running order has no outcome
		// yet, so no hook — it stays neutral).
		if v.Status == "done" {
			if v.Caught {
				text += " caught"
				span = append(span, h.Data("outcome", "caught"))
			} else {
				text += " missed"
				span = append(span, h.Data("outcome", "missed"))
			}
		}
		span = append(span, h.Text(text))
		// The oracle's verdict for a resolved order — the WHY behind a catch/miss
		// (no-catch vs lost-via-rename vs no-oracle-signal …) — as a calm secondary
		// detail. Omitted when none is persisted (never an empty "why" tag).
		if v.Status == "done" && v.Verdict != "" {
			span = append(span, h.Span(h.Class("board-row__dispatch-why"), h.Text(" "+v.Verdict)))
		}
		spans = append(spans, h.Span(span...))
	}
	return h.Div(spans...)
}
