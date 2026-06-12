package ledger

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/joaomdsg/packets/internal/fabric"
)

// Projection is the economy state folded from an ordered ledger event sequence,
// independent of where the bytes live. It exposes the SAME read projections as
// Log — the substrate-independent economy logic the migration moves unchanged —
// so a caller can fold from the stream exactly what the JSONL scan folds from the
// file. Its methods mirror Log's read methods minus the error: the events are
// already in memory, so there is no I/O to fail.
type Projection struct {
	catches  []CatchRecord
	balance  int
	orders   []WorkOrderRecord
	status   map[int]string
	verdicts map[int]string // per-order oracle verdict (last-writer-wins), diagnostic only
	// blocks/unblocks hold the FIRST stamp seen per id (earliest block, earliest
	// clearing) — a block is raised and cleared once, so a duplicate never re-pays
	// nor moves the latency interval. The attention-bandwidth earn folds from these.
	blocks      map[string]int64
	unblocks    map[string]int64
	bwSpent     int                  // total bandwidth debited (the meter's sink)
	refinements []RefinedOrderRecord // dead-air sharpenings the backlog folds on read
}

// Bandwidth is the earned attention bandwidth: the sum of awards across every
// cleared block (a block id with a matching unblock), each award folding the
// throughput base + the latency bonus. An open block earns nothing.
func (p Projection) Bandwidth() int {
	total := 0
	for id, blockMs := range p.blocks {
		unblockMs, ok := p.unblocks[id]
		if !ok {
			continue // an open block — not yet cleared, earns nothing
		}
		latency := time.Duration(unblockMs-blockMs) * time.Millisecond
		total += bandwidthAward(latency)
	}
	return total - p.bwSpent
}

// Balance is credits (confirmed catches) minus debits (positive spends), folded
// identically to Log.Balance — a non-positive spend amount never debits, and a
// non-catch outcome never credits.
func (p Projection) Balance() int { return p.balance }

// Records is the catch-kind record stream, UNFILTERED by ShouldRecord (mirroring
// Log.Records): a forged non-catch line survives the projection, while never
// contributing to Balance or the confirmed stock.
func (p Projection) Records() []CatchRecord { return p.catches }

// WorkOrders is the funded work-order ledger in funding (id) order.
func (p Projection) WorkOrders() []WorkOrderRecord { return p.orders }

// Refinements is the refined-work-order ledger in append order — the sharpening
// facts (split/criteria/convention) the backlog projection folds on read.
func (p Projection) Refinements() []RefinedOrderRecord { return p.refinements }

// DispatchStatusCounts tallies the orders by CURRENT status (last status line
// per id wins; an unknown status counts as queued), mirroring Log.
func (p Projection) DispatchStatusCounts() DispatchCounts {
	var c DispatchCounts
	for _, o := range p.orders {
		switch p.status[o.ID] {
		case "running":
			c.Running++
		case "done":
			c.Done++
		default:
			c.Queued++
		}
	}
	return c
}

// caughtWorkOrders maps "wo:<id>" → true for every dispatched-order catch — the
// shared catch-provenance lookup behind RecentDispatches and ScoutingReport: a
// catch tagged Producer "wo:<id>" marks that order CAUGHT, while a "connect" catch
// never does (the two-scores provenance gate).
func (p Projection) caughtWorkOrders() map[string]bool {
	caught := make(map[string]bool)
	for _, c := range p.catches {
		if strings.HasPrefix(c.Producer, "wo:") {
			caught[c.Producer] = true
		}
	}
	return caught
}

// RecentDispatches projects the funded orders into DispatchViews, NEWEST FIRST,
// capped at n (n<=0 = all). Per order: its current status (last status line,
// default queued) and whether its run minted a catch (a catch tagged
// Producer "wo:<id>"). Pure projection; mirrors Log.RecentDispatches.
func (p Projection) RecentDispatches(n int) []DispatchView {
	caughtIDs := p.caughtWorkOrders()
	views := make([]DispatchView, 0, len(p.orders))
	for i := len(p.orders) - 1; i >= 0; i-- { // newest (highest id) first
		o := p.orders[i]
		status := p.status[o.ID]
		if status == "" {
			status = "queued"
		}
		views = append(views, DispatchView{
			ID:      o.ID,
			Target:  o.Target,
			Status:  status,
			Caught:  caughtIDs["wo:"+strconv.Itoa(o.ID)],
			Verdict: p.verdicts[o.ID], // "" when none persisted yet
		})
		if n > 0 && len(views) == n {
			break
		}
	}
	return views
}

// ScoutReport is a per-session FIRST-PASS catch-rate — the outward Trust Ledger
// signal ("this lane ships clean — N/M first-pass"). Completed counts orders that
// ran to done; Caught counts those whose run minted a confirmed catch. Counts-only
// and retrospective: it redeems against logged facts (order status + the wo:<id>
// catch provenance), never a model judgment.
type ScoutReport struct {
	Caught    int
	Completed int
}

// FirstPassRate is Caught/Completed. It returns 0 when nothing has completed — the
// caller MUST gate on Completed>0 to tell "no signal yet" from a true 0% lane.
func (s ScoutReport) FirstPassRate() float64 {
	if s.Completed == 0 {
		return 0
	}
	return float64(s.Caught) / float64(s.Completed)
}

// ScoutingReport folds the per-session first-pass catch-rate: of the orders that
// completed (status done — a failed/queued/running order is not a completed pass),
// how many minted a confirmed catch (a CatchRecord tagged Producer "wo:<id>"). Pure
// projection; Caught is gated on completion, so it never exceeds Completed.
func (p Projection) ScoutingReport() ScoutReport {
	caughtIDs := p.caughtWorkOrders()
	var r ScoutReport
	for _, o := range p.orders {
		if p.status[o.ID] != "done" {
			continue
		}
		r.Completed++
		if caughtIDs["wo:"+strconv.Itoa(o.ID)] {
			r.Caught++
		}
	}
	return r
}

// QueuedWorkOrders returns the orders whose current status is exactly queued, in
// funding order — the runner's input, mirroring Log.QueuedWorkOrders.
func (p Projection) QueuedWorkOrders() []WorkOrderRecord {
	var queued []WorkOrderRecord
	for _, o := range p.orders {
		if p.status[o.ID] == "queued" {
			queued = append(queued, o)
		}
	}
	return queued
}

// ReplayProjection folds the economy state for session+instance from the fabric:
// it replays every MINTED ledger event (the authoritative source-of-truth
// subtree, demuxed from any scratch fan-out), decodes each by its subject kind,
// and folds the SAME projections the JSONL scan produces. The events keep their
// authoritative global sequence order, so the order-dependent status fold
// (last-writer-wins per id) matches the append-order file scan.
func ReplayProjection(ctx context.Context, f *fabric.Fabric, session, instance string) (Projection, error) {
	filter := fabric.EventSubject(session, instance, fabric.StatusMinted, "*")
	events, err := f.ReplaySubject(ctx, filter)
	if err != nil {
		return Projection{}, err
	}
	return foldEvents(events)
}

// FleetProjection folds every session's economy from the fabric in one replay:
// it replays the cross-session minted subtree, groups events by their session
// token, and folds each group with the same canonical fold ReplayProjection
// uses — returning one Projection per session. This is the cross-process
// aggregator: the fleet board derives from the authoritative stream, not from
// any in-process registry, so it reflects sessions written by any producer.
func FleetProjection(ctx context.Context, f *fabric.Fabric) (map[string]Projection, error) {
	events, err := f.ReplaySubject(ctx, fabric.FleetMintedSubject())
	if err != nil {
		return nil, err
	}
	// The fleet filter only matches canonical subjects, so SessionOf always
	// yields a real session token here.
	bySession := map[string][]fabric.Event{}
	for _, e := range events {
		s := fabric.SessionOf(e.Subject)
		bySession[s] = append(bySession[s], e)
	}
	fleet := make(map[string]Projection, len(bySession))
	for s, evs := range bySession {
		p, err := foldEvents(evs)
		if err != nil {
			return nil, err
		}
		fleet[s] = p
	}
	return fleet, nil
}

// FleetView is one session's full board row: its confirmed economy (the embedded
// Projection, whose Balance/Records/DispatchStatusCounts promote) PLUS the
// producers' claim lifecycle — InFlight pending bets and Rejected verified-losses.
// The bet counts are independent axes from the confirmed economy (two-scores): a
// pending or lost bet never folds into Balance or the confirmed stock.
type FleetView struct {
	Projection
	InFlight int
	Rejected int
}

// FleetBoard folds the whole fabric into one per-session board row: the minted
// economy (FleetProjection) overlaid with each session's claim lifecycle. It
// replays the cross-session claim subtree separately from the minted subtree —
// claim/verdict events are NOT economy events and must not go through the minted
// fold — and computes InFlight/Rejected through the SAME pure projections the
// per-Log ClaimsInFlight/ClaimsRejected use, so the stream and the in-process
// board can never disagree on how a bet is classified. A session that has only
// submitted claims (no mint yet) still appears, so a producer's new bets are
// never invisible. Either replay error degrades the caller to (nil, err).
func FleetBoard(ctx context.Context, f *fabric.Fabric) (map[string]FleetView, error) {
	minted, err := FleetProjection(ctx, f)
	if err != nil {
		return nil, err
	}
	claimEvents, err := f.ReplaySubject(ctx, fabric.FleetClaimSubject())
	if err != nil {
		return nil, err
	}

	out := make(map[string]FleetView, len(minted))
	for s, p := range minted {
		out[s] = FleetView{Projection: p}
	}

	bySession := map[string][]fabric.Event{}
	for _, e := range claimEvents {
		s := fabric.SessionOf(e.Subject)
		bySession[s] = append(bySession[s], e)
	}
	for s, evs := range bySession {
		v := out[s] // zero FleetView (nil Projection.Records()) for a claim-only session
		v.InFlight = claimsInFlightFrom(evs, v.Records())
		v.Rejected = claimsRejectedFrom(evs, v.Records())
		out[s] = v
	}
	return out, nil
}

func foldEvents(events []fabric.Event) (Projection, error) {
	p := Projection{status: map[int]string{}, verdicts: map[int]string{},
		blocks: map[string]int64{}, unblocks: map[string]int64{}}
	for _, e := range events {
		switch kind := e.Subject[strings.LastIndex(e.Subject, ".")+1:]; kind {
		case subjectKindCatch:
			r, err := DecodeCatch(e.Data)
			if err != nil {
				return Projection{}, err
			}
			p.catches = append(p.catches, r)
			if ShouldRecord(r.Outcome) {
				p.balance++
			}
		case kindSpend:
			s, err := DecodeSpend(e.Data)
			if err != nil {
				return Projection{}, err
			}
			if s.Amount > 0 {
				p.balance -= s.Amount
			}
		case kindWorkOrder:
			w, err := DecodeWorkOrder(e.Data)
			if err != nil {
				return Projection{}, err
			}
			p.orders = append(p.orders, w)
			p.status[w.ID] = w.Status
		case kindWOStatus:
			st, err := DecodeStatus(e.Data)
			if err != nil {
				return Projection{}, err
			}
			p.status[st.ID] = st.Status
		case kindWOVerdict:
			v, err := DecodeWorkOrderVerdict(e.Data)
			if err != nil {
				return Projection{}, err
			}
			p.verdicts[v.ID] = v.Verdict // last-writer-wins, like status
		case kindBlock:
			b, err := DecodeBlock(e.Data)
			if err != nil {
				return Projection{}, err
			}
			if _, seen := p.blocks[b.ID]; !seen {
				p.blocks[b.ID] = b.AtUnixMs // first block stamp wins (the interval start)
			}
		case kindUnblock:
			u, err := DecodeUnblock(e.Data)
			if err != nil {
				return Projection{}, err
			}
			if _, seen := p.unblocks[u.ID]; !seen {
				p.unblocks[u.ID] = u.AtUnixMs // a block clears once; duplicates never re-pay
			}
		case kindBWSpend:
			s, err := DecodeBandwidthSpend(e.Data)
			if err != nil {
				return Projection{}, err
			}
			if s.Amount > 0 {
				p.bwSpent += s.Amount
			}
		case kindWORefine:
			r, err := DecodeRefine(e.Data)
			if err != nil {
				return Projection{}, err
			}
			p.refinements = append(p.refinements, r)
		default:
			return Projection{}, fmt.Errorf("ledger: replay encountered unknown event kind %q", kind)
		}
	}
	return p, nil
}
